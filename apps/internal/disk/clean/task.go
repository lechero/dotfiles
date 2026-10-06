package clean

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

// Tier says what a cleanup costs you.
type Tier int

const (
	Tier1 Tier = 1 // pure caches: they regenerate on their own
	Tier2 Tier = 2 // regenerable, but you pay with a re-download or a rebuild
)

// Item is one thing a task can clean: usually a folder, sometimes a command.
type Item struct {
	Paths    []string // deleted together; the first one is shown
	Label    string
	Size     int64 // bytes it should free (an estimate for commands)
	Selected bool
	Blocked  string // why it cannot be cleaned right now; empty when it can
	Note     string // context for the decision: "newest v22, kept"
	Data     any    // whatever a task's Run needs beyond the paths
}

// Path is the item's main path.
func (it Item) Path() string {
	if len(it.Paths) == 0 {
		return ""
	}
	return it.Paths[0]
}

// Ready reports whether a run would clean the item.
func (it Item) Ready() bool { return it.Selected && it.Blocked == "" }

// Task is one kind of cleanup.
type Task struct {
	ID    string
	Title string
	Tier  Tier
	About string // what it removes and why that is safe
	Cost  string // what you give up
	// Heavy tasks walk millions of files to size their items; they wait for
	// the overview scan and read sizes off its tree instead.
	Heavy    bool
	Discover func(ctx context.Context, env *Env) ([]Item, error)
	// Run cleans the chosen items. Nil means "delete each item's paths".
	Run func(ctx context.Context, env *Env, it Item, log func(string)) error
}

// Env is everything a task may touch, so tests can point it at a fake home.
type Env struct {
	Home     string
	Guard    Guard
	LogDir   string // where run logs go
	StateDir string // where run history goes
	AppDirs  []string

	procsMu    sync.Mutex
	procsTaken bool
	procs      Procs
	// SnapshotFn takes the process snapshot; tests replace it.
	SnapshotFn func(ctx context.Context) (Procs, error)

	treeMu sync.RWMutex
	tree   *scan.Node

	// LookPath and Command are swapped out in tests.
	LookPath func(string) (string, error)
	Command  func(ctx context.Context, name string, args ...string) *exec.Cmd
	Now      func() time.Time
}

// NewEnv builds the real environment for home.
func NewEnv(home string) *Env {
	return &Env{
		Home:       home,
		Guard:      Guard{Home: home},
		LogDir:     filepath.Join(home, "Library/Logs/manage-disk"),
		StateDir:   filepath.Join(home, "Library/Application Support/manage-disk"),
		AppDirs:    []string{"/Applications", filepath.Join(home, "Applications")},
		SnapshotFn: SnapshotProcs,
		LookPath:   exec.LookPath,
		Command:    exec.CommandContext,
		Now:        time.Now,
	}
}

// Procs returns the process snapshot, taking it on first use. Call
// ResetProcs before a new discovery round so it reflects what runs now.
func (e *Env) Procs(ctx context.Context) Procs {
	e.procsMu.Lock()
	defer e.procsMu.Unlock()
	if !e.procsTaken && e.SnapshotFn != nil {
		e.procs, _ = e.SnapshotFn(ctx)
		e.procsTaken = true
	}
	return e.procs
}

// ResetProcs forgets the snapshot.
func (e *Env) ResetProcs() {
	e.procsMu.Lock()
	e.procsTaken = false
	e.procs = nil
	e.procsMu.Unlock()
}

// SetTree hands a finished scan to the sizer; nil forgets it.
func (e *Env) SetTree(root *scan.Node) {
	e.treeMu.Lock()
	e.tree = root
	e.treeMu.Unlock()
}

// EditTree runs fn with the scan tree locked for writing (after a cleanup,
// to drop what was deleted). fn is not called when there is no tree.
func (e *Env) EditTree(fn func(root *scan.Node)) {
	e.treeMu.Lock()
	defer e.treeMu.Unlock()
	if e.tree != nil {
		fn(e.tree)
	}
}

// Size measures path, reading it off the scan tree when the tree has it.
func (e *Env) Size(ctx context.Context, path string) int64 {
	e.treeMu.RLock()
	var n *scan.Node
	if e.tree != nil {
		n = e.tree.Find(path)
	}
	e.treeMu.RUnlock()
	if n != nil && n.Skipped == "" {
		return n.Size
	}
	return scan.SizeOf(ctx, path)
}

// Stream runs a command and hands every line it prints to log.
func (e *Env) Stream(ctx context.Context, log func(string), extraEnv []string, name string, args ...string) error {
	cmd := e.Command(ctx, name, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		return err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" {
				log(line)
			}
		}
		io.Copy(io.Discard, pr) // keep draining if a line was too long
	}()
	err := cmd.Wait()
	pw.Close()
	<-done
	return err
}

// Output runs a command and returns what it printed on stdout.
func (e *Env) Output(ctx context.Context, name string, args ...string) (string, error) {
	out, err := e.Command(ctx, name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// EventKind is what happened to an item during a run.
type EventKind int

const (
	EventStart EventKind = iota
	EventDone
	EventSkip
	EventFail
	EventLog
)

// Event reports progress from a run.
type Event struct {
	Kind  EventKind
	Task  string // task title
	Item  string // item label
	Path  string
	Bytes int64 // for EventDone: the item's estimated size
	Msg   string
}

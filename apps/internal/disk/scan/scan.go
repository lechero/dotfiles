package scan

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Options tune a scan.
type Options struct {
	Home           string // privacy rules apply to paths under Home
	MinNode        int64  // smaller entries are folded into their parent's Small total
	MaxChildren    int    // keep at most this many entries per directory, largest first
	IncludePrivate bool   // also read the consent-guarded places (may pop macOS dialogs)
	Workers        int    // directories read in parallel
	// DirTimeout gives up on a folder that does not answer — what a pending
	// macOS consent dialog looks like from here. Zero waits forever.
	DirTimeout time.Duration

	readNames func(path string) ([]string, error) // tests replace how folders are read
}

// DefaultOptions keeps every entry of 1 MiB or more, which holds a home full of
// node_modules in tens of megabytes of memory instead of gigabytes.
func DefaultOptions(home string) Options {
	return Options{Home: home, MinNode: 1 << 20, MaxChildren: 400, Workers: 8 * runtime.NumCPU(), DirTimeout: 15 * time.Second}
}

// Progress is updated while a scan runs and is safe to read from any goroutine.
type Progress struct {
	files, dirs, bytes atomic.Int64
	current            atomic.Pointer[string]
	waiting            atomic.Pointer[string]
}

// Waiting is a folder that has kept the scan waiting for a while, or "".
func (p *Progress) Waiting() string {
	if s := p.waiting.Load(); s != nil {
		return *s
	}
	return ""
}

func (p *Progress) Files() int64 { return p.files.Load() }
func (p *Progress) Dirs() int64  { return p.dirs.Load() }
func (p *Progress) Bytes() int64 { return p.bytes.Load() }

// Current is the directory most recently opened.
func (p *Progress) Current() string {
	if s := p.current.Load(); s != nil {
		return *s
	}
	return ""
}

// Result is a finished (or cancelled) scan.
type Result struct {
	Root    *Node
	Took    time.Duration
	Errors  int64
	Private []string // places left out for privacy, relative to Options.Home
	// NoResponse are folders that never answered — usually a macOS consent
	// dialog waiting on screen. Absolute paths.
	NoResponse []string
}

// Scan measures everything under root. When ctx is cancelled it stops early
// and returns the partial tree together with ctx.Err().
func Scan(ctx context.Context, root string, opts Options, prog *Progress) (*Result, error) {
	if prog == nil {
		prog = &Progress{}
	}
	start := time.Now()
	var st syscall.Stat_t
	if err := syscall.Lstat(root, &st); err != nil {
		return nil, err
	}
	n := &Node{Name: root, Size: st.Blocks * 512}
	if st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		n.Files = 1
		return &Result{Root: n, Took: time.Since(start)}, nil
	}
	n.IsDir = true
	w := &walker{
		ctx:     ctx,
		opts:    opts,
		rootDev: st.Dev,
		prog:    prog,
		sem:     make(chan struct{}, max(1, opts.Workers)),
	}
	w.dir(n, root)
	sort.Strings(w.private)
	sort.Strings(w.noResponse)
	return &Result{Root: n, Took: time.Since(start), Errors: w.errs.Load(), Private: w.private, NoResponse: w.noResponse}, ctx.Err()
}

type walker struct {
	ctx     context.Context
	opts    Options
	rootDev int32
	prog    *Progress
	sem     chan struct{} // bounds goroutines; a full channel means "recurse inline"
	links   sync.Map      // (dev, inode) of multiply-linked files already counted
	errs    atomic.Int64

	mu         sync.Mutex
	private    []string
	noResponse []string
}

func (w *walker) dir(n *Node, path string) {
	if w.ctx.Err() != nil {
		return
	}
	names, err := w.list(path)
	if err != nil && len(names) == 0 {
		n.Skipped = SkipNoAccess
		if errors.Is(err, errNoResponse) {
			n.Skipped = SkipNoResponse
			w.mu.Lock()
			w.noResponse = append(w.noResponse, path)
			w.mu.Unlock()
		}
		w.errs.Add(1)
		return
	}
	w.prog.current.Store(&path)

	var (
		wg           sync.WaitGroup
		dirs, kept   []*Node
		files, bytes int64
	)
	for _, name := range names {
		child := path + "/" + name
		var st syscall.Stat_t
		if err := syscall.Lstat(child, &st); err != nil {
			w.errs.Add(1)
			continue
		}
		size := st.Blocks * 512
		if st.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			d := &Node{Name: name, IsDir: true, Size: size, Parent: n}
			dirs = append(dirs, d)
			switch {
			case st.Dev != w.rootDev:
				d.Skipped = SkipOtherVolume
			case w.privacy(child) != "":
				d.Skipped = SkipPrivacy
				w.notePrivate(child)
			default:
				select {
				case w.sem <- struct{}{}:
					wg.Add(1)
					go func() {
						defer wg.Done()
						w.dir(d, child)
						<-w.sem
					}()
				default:
					w.dir(d, child)
				}
			}
			continue
		}
		if st.Nlink > 1 && st.Mode&syscall.S_IFMT == syscall.S_IFREG {
			if _, seen := w.links.LoadOrStore([2]uint64{uint64(st.Dev), st.Ino}, struct{}{}); seen {
				continue
			}
		}
		files++
		bytes += size
		if size >= w.opts.MinNode {
			kept = append(kept, &Node{Name: name, Size: size, Files: 1, Parent: n})
		} else {
			n.Small += size
			n.SmallN++
		}
	}
	w.prog.files.Add(files)
	w.prog.dirs.Add(int64(len(dirs)))
	w.prog.bytes.Add(bytes)
	wg.Wait()

	n.Size += bytes
	n.Files += files
	for _, d := range dirs {
		n.Size += d.Size
		n.Files += d.Files
		if d.Skipped != "" || d.skipsInside {
			n.skipsInside = true
			kept = append(kept, d)
		} else if d.Size >= w.opts.MinNode {
			kept = append(kept, d)
		} else {
			n.Small += d.Size
			n.SmallN++
		}
	}
	sortNodes(kept)
	if limit := w.opts.MaxChildren; limit > 0 && len(kept) > limit {
		for _, c := range kept[limit:] {
			n.Small += c.Size
			n.SmallN++
		}
		kept = kept[:limit:limit]
	}
	n.Children = kept
}

var errNoResponse = errors.New("folder did not answer")

// slowFolder is how long a folder may take before Progress reports waiting on it.
const slowFolder = 2 * time.Second

// list reads a folder's names without letting one that never answers stall the
// scan. The read that hangs is left behind; it ends when macOS gets an answer.
func (w *walker) list(path string) ([]string, error) {
	read := w.opts.readNames
	if read == nil {
		read = readNames
	}
	if w.opts.DirTimeout <= 0 {
		return read(path)
	}
	type result struct {
		names []string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		names, err := read(path)
		ch <- result{names, err}
	}()
	first := min(slowFolder, w.opts.DirTimeout)
	slow := time.NewTimer(first)
	defer slow.Stop()
	select {
	case r := <-ch:
		return r.names, r.err
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-slow.C:
	}
	w.prog.waiting.Store(&path)
	defer w.prog.waiting.CompareAndSwap(&path, nil)
	giveUp := time.NewTimer(w.opts.DirTimeout - first)
	defer giveUp.Stop()
	select {
	case r := <-ch:
		return r.names, r.err
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-giveUp.C:
		return nil, errNoResponse
	}
}

func readNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // read-only
	return f.Readdirnames(-1)
}

func (w *walker) privacy(path string) string {
	home := w.opts.Home
	if home == "" {
		return ""
	}
	rel, ok := strings.CutPrefix(path, home+"/")
	if !ok {
		return ""
	}
	return PrivacyReason(rel, w.opts.IncludePrivate)
}

func (w *walker) notePrivate(path string) {
	rel := strings.TrimPrefix(path, w.opts.Home+"/")
	w.mu.Lock()
	w.private = append(w.private, rel)
	w.mu.Unlock()
}

// SizeOf measures one path the way a scan would (allocated bytes, same
// volume, hardlinks once, symlinks not followed) without keeping a tree.
func SizeOf(ctx context.Context, path string) int64 {
	opts := Options{MinNode: 1 << 62, Workers: 2 * runtime.NumCPU(), DirTimeout: 15 * time.Second}
	res, _ := Scan(ctx, path, opts, nil)
	if res == nil {
		return 0
	}
	return res.Root.Size
}

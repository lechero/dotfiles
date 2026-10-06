package clean

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

// Selection is what a run cleans: the chosen items of one task.
type Selection struct {
	Task  *Task
	Items []Item
}

// Cleaned records an item a run dealt with, so the caller can update its scan tree.
type Cleaned struct {
	Task    *Task
	Item    Item
	Command bool // cleaned by a command (re-measure), not by deleting its paths (drop)
}

// Summary describes a finished run.
type Summary struct {
	Started, Ended        time.Time
	DryRun, Cancelled     bool
	FreeBefore, FreeAfter int64
	Estimated             int64 // what the cleaned items were measured at
	Cleaned               []Cleaned
	Skipped, Failed       int
	LogPath               string
}

// Freed is how much free space grew, as the volume reports it.
func (s Summary) Freed() int64 { return s.FreeAfter - s.FreeBefore }

// Run cleans the selections in order, reporting progress. It never deletes a
// path the Guard refuses, and in a dry run it deletes and runs nothing.
func Run(ctx context.Context, env *Env, sels []Selection, dryRun bool, report func(Event)) Summary {
	s := Summary{Started: env.Now(), DryRun: dryRun}
	if v, err := scan.VolumeOf(env.Home); err == nil {
		s.FreeBefore = v.Free
	}
	logf, closeLog := openRunLog(env, &s)
	defer closeLog()
	mode := "run"
	if dryRun {
		mode = "dry run"
	}
	logf("=== %s started %s", mode, s.Started.Format(time.RFC3339))

	for _, sel := range sels {
		for _, it := range sel.Items {
			ev := Event{Task: sel.Task.Title, Item: it.Label, Path: it.Path()}
			switch {
			case ctx.Err() != nil:
				s.Cancelled = true
				s.Skipped++
				ev.Kind, ev.Msg = EventSkip, "cancelled"
				report(ev)
				continue
			case !it.Ready():
				s.Skipped++
				ev.Kind, ev.Msg = EventSkip, it.Blocked
				report(ev)
				logf("skip  %s: %s", ev.Path, it.Blocked)
				continue
			}
			ev.Kind = EventStart
			report(ev)

			err := cleanItem(ctx, env, sel.Task, it, dryRun, func(line string) {
				report(Event{Kind: EventLog, Task: sel.Task.Title, Item: it.Label, Msg: line})
				logf("      %s", line)
			})
			if err != nil {
				s.Failed++
				ev.Kind, ev.Msg = EventFail, err.Error()
				report(ev)
				logf("FAIL  %s: %v", ev.Path, err)
				continue
			}
			s.Estimated += it.Size
			s.Cleaned = append(s.Cleaned, Cleaned{Task: sel.Task, Item: it, Command: sel.Task.Run != nil})
			ev.Kind, ev.Bytes = EventDone, it.Size
			if dryRun {
				ev.Msg = "would clean"
			}
			report(ev)
			logf("%s %s (%s) [%s]", doneWord(dryRun), ev.Path, human.Bytes(it.Size), sel.Task.ID)
		}
	}

	s.Ended = env.Now()
	if v, err := scan.VolumeOf(env.Home); err == nil {
		s.FreeAfter = v.Free
	}
	logf("=== done in %s: %d cleaned, %d skipped, %d failed, free space %s → %s",
		human.Duration(s.Ended.Sub(s.Started)), len(s.Cleaned), s.Skipped, s.Failed,
		human.Bytes(s.FreeBefore), human.Bytes(s.FreeAfter))
	if !dryRun && len(s.Cleaned) > 0 {
		recordHistory(env, s)
	}
	return s
}

func doneWord(dryRun bool) string {
	if dryRun {
		return "would"
	}
	return "done "
}

func cleanItem(ctx context.Context, env *Env, t *Task, it Item, dryRun bool, log func(string)) error {
	if t.Run != nil { // a command decides what it deletes; its paths are only measured
		if dryRun {
			log("would run: " + it.Label)
			return nil
		}
		return t.Run(ctx, env, it, log)
	}
	for _, p := range it.Paths {
		if err := env.Guard.Check(p); err != nil {
			return err
		}
	}
	if dryRun {
		return nil
	}
	for _, p := range it.Paths {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func openRunLog(env *Env, s *Summary) (logf func(string, ...any), closeLog func()) {
	noop := func(string, ...any) {}
	if env.LogDir == "" || os.MkdirAll(env.LogDir, 0o755) != nil {
		return noop, func() {}
	}
	s.LogPath = filepath.Join(env.LogDir, s.Started.Format("2006-01-02")+".log")
	f, err := os.OpenFile(s.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		s.LogPath = ""
		return noop, func() {}
	}
	w := bufio.NewWriter(f)
	return func(format string, args ...any) {
			fmt.Fprintf(w, "%s %s\n", env.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
		}, func() {
			// The log is a convenience: failing to write it mustn't fail the run.
			_ = w.Flush()
			_ = f.Close()
		}
}

// HistoryEntry is one real (not dry) run, as kept in history.jsonl.
type HistoryEntry struct {
	Time      time.Time `json:"time"`
	Freed     int64     `json:"freed"`
	Estimated int64     `json:"estimated"`
	Cleaned   int       `json:"cleaned"`
	Failed    int       `json:"failed"`
	Tasks     []string  `json:"tasks"`
}

func recordHistory(env *Env, s Summary) {
	if env.StateDir == "" || os.MkdirAll(env.StateDir, 0o755) != nil {
		return
	}
	seen := map[string]bool{}
	var tasks []string
	for _, c := range s.Cleaned {
		if !seen[c.Task.ID] {
			seen[c.Task.ID] = true
			tasks = append(tasks, c.Task.ID)
		}
	}
	e := HistoryEntry{Time: s.Ended, Freed: s.Freed(), Estimated: s.Estimated, Cleaned: len(s.Cleaned), Failed: s.Failed, Tasks: tasks}
	b, _ := json.Marshal(e)
	f, err := os.OpenFile(filepath.Join(env.StateDir, "history.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	// History only feeds the totals, so a lost entry costs one run's numbers.
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(b, '\n'))
}

// LastRun returns the most recent real run, or nil when there is none.
func LastRun(env *Env) *HistoryEntry {
	b, err := os.ReadFile(filepath.Join(env.StateDir, "history.jsonl"))
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		var e HistoryEntry
		if json.Unmarshal([]byte(lines[i]), &e) == nil && !e.Time.IsZero() {
			return &e
		}
	}
	return nil
}

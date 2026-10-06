package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"

	"manage-disk/internal/clean"
	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

// report prints the overview and what each cleanup would take, deleting nothing.
func report(w io.Writer, env *clean.Env, includePrivate, verbose bool) error {
	ctx := context.Background()
	if v, err := scan.VolumeOf(env.Home); err == nil {
		fmt.Fprintf(w, "Disk: %s free of %s (%.0f%% full)\n\n",
			human.Bytes(v.Free), human.Bytes(v.Total), 100*float64(v.Used())/float64(max(1, v.Total)))
	}

	opts := scan.DefaultOptions(env.Home)
	opts.IncludePrivate = includePrivate
	prog := &scan.Progress{}
	stop := showProgress(prog)
	res, err := scan.Scan(ctx, env.Home, opts, prog)
	stop()
	if err != nil {
		return err
	}
	env.SetTree(res.Root)
	root := res.Root
	fmt.Fprintf(w, "Scanned %s files, %s, in %s\n\n", human.Count(root.Files), human.Bytes(root.Size), human.Duration(res.Took))

	fmt.Fprintln(w, "Where your space goes")
	for _, c := range scan.Categories(root) {
		if c.Partial && c.Size < 1<<20 {
			fmt.Fprintf(w, "  %-26s %10s  not scanned *\n", c.Name, "—")
			continue
		}
		mark := ""
		if c.Partial {
			mark = " *"
		}
		fmt.Fprintf(w, "  %-26s %10s  %3.0f%%%s\n", c.Name, human.Bytes(c.Size), 100*float64(c.Size)/float64(max(1, root.Size)), mark)
	}
	fmt.Fprintln(w, "\nBiggest spots")
	for _, n := range scan.Hotspots(root, 15, 256<<20) {
		fmt.Fprintf(w, "  %10s  %s\n", human.Bytes(n.Size), "~/"+n.Rel())
	}
	if len(res.NoResponse) > 0 {
		fmt.Fprintf(w, "\n! Never answered, so not measured (macOS may have asked for permission): %s\n", strings.Join(res.NoResponse, ", "))
	}
	if asks := scan.SummarizePrivate(res.Private); len(asks) > 0 {
		fmt.Fprintf(w, "\n* Not scanned, macOS asks first: %s — use --include-private\n", strings.Join(asks, ", "))
	}

	tasks := clean.Catalog()
	results := make([][]clean.Item, len(tasks))
	errs := make([]error, len(tasks))
	var wg sync.WaitGroup
	for i, t := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = t.Discover(ctx, env)
		}()
	}
	wg.Wait()

	fmt.Fprintln(w, "\nCleanup plan (nothing was deleted)")
	for tier := clean.Tier1; tier <= clean.Tier2; tier++ {
		fmt.Fprintf(w, "  Tier %d\n", tier)
		for i, t := range tasks {
			if t.Tier != tier {
				continue
			}
			var size int64
			var ready, blocked int
			for _, it := range results[i] {
				if it.Ready() {
					size += it.Size
					ready++
				} else if it.Selected && it.Blocked != "" {
					blocked++
				}
			}
			line := fmt.Sprintf("    %-32s %10s  %d of %d items", t.Title, human.Bytes(size), ready, len(results[i]))
			if blocked > 0 {
				line += fmt.Sprintf(", %d blocked", blocked)
			}
			if errs[i] != nil {
				line += "  error: " + errs[i].Error()
			}
			fmt.Fprintln(w, line)
			for _, it := range results[i] {
				if !verbose && it.Blocked == "" {
					continue
				}
				state := "keep"
				switch {
				case it.Blocked != "":
					state = "BLOCKED " + it.Blocked
				case it.Selected:
					state = "clean"
				}
				note := ""
				if it.Note != "" {
					note = "  (" + it.Note + ")"
				}
				fmt.Fprintf(w, "        %10s  %-8s %s%s\n", human.Bytes(it.Size), state, it.Label, note)
			}
		}
	}
	return nil
}

// showProgress draws a one-line counter on stderr while a scan runs, if stderr is a terminal.
func showProgress(p *scan.Progress) (stop func()) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return func() {}
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			case <-t.C:
				fmt.Fprintf(os.Stderr, "\r\033[Kscanning… %s files, %s", human.Count(p.Files()), human.Bytes(p.Bytes()))
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}

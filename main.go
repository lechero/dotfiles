// Command manage-disk shows where disk space goes under your home folder and
// re-runs the cache cleanups that are safe to repeat.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"manage-disk/internal/clean"
	"manage-disk/internal/ui"
)

const usageText = `manage-disk — where your disk space goes, and the cleanups that are safe to repeat

Usage:
  manage-disk [flags]            open the interface
  manage-disk report [flags]     print the overview and the cleanup plan; changes nothing
  manage-disk worktrees [--fetch] list worktrees with a verdict on each; changes nothing

Flags:
  --dry-run           in the interface: show what a clean would do, delete nothing
  --include-private   also scan Desktop, Documents, Downloads… (macOS may ask for permission)
  -v                  report: list every item, not just the totals
  --fetch             worktrees: git fetch each repo first, to judge against today's main
`

func main() {
	fs := flag.NewFlagSet("manage-disk", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "")
	private := fs.Bool("include-private", false, "")
	verbose := fs.Bool("v", false, "")
	fetch := fs.Bool("fetch", false, "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }

	// Let flags sit before or after the subcommand.
	var cmd string
	var flagArgs []string
	for _, arg := range os.Args[1:] {
		if cmd == "" && !strings.HasPrefix(arg, "-") {
			cmd = arg
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}
	fs.Parse(flagArgs)

	home, err := os.UserHomeDir()
	if err != nil {
		fail(err)
	}
	env := clean.NewEnv(home)

	switch cmd {
	case "":
		p := tea.NewProgram(ui.New(env, ui.Options{DryRun: *dryRun, IncludePrivate: *private}), tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fail(err)
		}
	case "report":
		if err := report(os.Stdout, env, *private, *verbose); err != nil {
			fail(err)
		}
	case "worktrees":
		if err := listWorktrees(os.Stdout, env, *fetch); err != nil {
			fail(err)
		}
	case "help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usageText)
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "manage-disk:", err)
	os.Exit(1)
}

// Command manage-disk shows where disk space goes under your home folder and
// re-runs the cache cleanups that are safe to repeat.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/docker"
	"github.com/lechero/dotfiles/apps/internal/disk/ui"
)

const usageText = `manage-disk — where your disk space goes, and the cleanups that are safe to repeat

Usage:
  manage-disk [flags]            open the interface
  manage-disk report [flags]     print the overview and the cleanup plan; changes nothing
  manage-disk worktrees [--fetch] list worktrees with a verdict on each; changes nothing
  manage-disk docker [--target N] Docker's disk, a verdict on each image, container and volume,
                                 and the cheapest way to N GiB free; changes nothing

Flags:
  --dry-run           in the interface: show what a clean would do, delete nothing
  --include-private   also scan Desktop, Documents, Downloads… (macOS may ask for permission)
  -v                  report: list every item, not just the totals
  --fetch             worktrees: git fetch each repo first, to judge against today's main
  --target N          docker: GiB of free space Docker's disk should have (default 22)
`

func main() {
	fs := flag.NewFlagSet("manage-disk", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "")
	private := fs.Bool("include-private", false, "")
	verbose := fs.Bool("v", false, "")
	fetch := fs.Bool("fetch", false, "")
	target := fs.Float64("target", float64(docker.DefaultTarget)/(1<<30), "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }

	// Let flags sit before or after the subcommand; a flag's value is not one.
	var cmd string
	var flagArgs []string
	takesValue := false
	for _, arg := range os.Args[1:] {
		switch {
		case takesValue:
			flagArgs = append(flagArgs, arg)
			takesValue = false
		case cmd == "" && !strings.HasPrefix(arg, "-"):
			cmd = arg
		default:
			flagArgs = append(flagArgs, arg)
			name := strings.TrimLeft(arg, "-")
			takesValue = !strings.Contains(name, "=") && name == "target"
		}
	}
	_ = fs.Parse(flagArgs) // ExitOnError: a bad flag exits inside Parse
	if *target <= 0 {
		fmt.Fprintln(os.Stderr, "manage-disk: --target is GiB of free space, more than 0")
		os.Exit(2)
	}
	targetBytes := int64(*target * (1 << 30))

	home, err := os.UserHomeDir()
	if err != nil {
		fail(err)
	}
	env := clean.NewEnv(home)

	switch cmd {
	case "":
		opts := ui.Options{DryRun: *dryRun, IncludePrivate: *private, DockerTarget: targetBytes}
		p := tea.NewProgram(ui.New(env, opts))
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
	case "docker":
		if err := listDocker(os.Stdout, env, targetBytes); err != nil {
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

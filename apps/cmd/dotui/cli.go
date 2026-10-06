package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	tea "charm.land/bubbletea/v2"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	diskui "github.com/lechero/dotfiles/apps/internal/disk/ui"
	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/tui"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

// defaultPrio is how far check and install go when --prio isn't given.
const defaultPrio = 2

const usage = `dotui shows the Homebrew packages this setup needs, the state of the
dotfiles, and where the disk space goes. It installs or applies what's
missing, and cleans what's safe to clean.

Usage:
  dotui [--file F] [--source DIR] [--disk-dry-run]
                                           open the interactive view
  dotui list [--prio N] [--missing]        list packages and whether they're installed
  dotui check [--prio N]                   fail if a package up to priority N is missing
  dotui install [--prio N] [name ...]      install what's missing up to priority N,
                                           or the named packages

Priorities: 1 core, 2 daily, 3 sometimes, 4 rarely. check and install
default to --prio 2; list shows everything.

Flags:
  --file F      package list (default: $DOTUI_PACKAGES, or packages.yaml in
                chezmoi's source directory)
  --source DIR  chezmoi source directory for the dotfiles view
                (default: chezmoi's own)
  --disk-dry-run
                in the disk view: show what a clean would do, delete nothing
`

// errUsage marks errors in how dotui was called.
var errUsage = errors.New("usage")

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	err := dispatch(ctx, args, stdin, stdout, stderr)
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(stdout, usage)
		return exitOK
	case errors.Is(err, errUsage):
		fmt.Fprintf(stderr, "dotui: %v\n\n%s", err, usage)
		return exitUsage
	case errors.Is(err, errSilent):
		return exitFailure
	default:
		fmt.Fprintf(stderr, "dotui: %v\n", err)
		return exitFailure
	}
}

// errSilent fails a command that has already said why.
var errSilent = errors.New("failed")

func dispatch(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "":
		return runTUI(ctx, args)
	case "list":
		return runList(ctx, args, stdout)
	case "check":
		return runCheck(ctx, args, stdout)
	case "install":
		return runInstall(ctx, args, stdin, stdout, stderr)
	case "help":
		return flag.ErrHelp
	case "_exec":
		return runExecThenWait(ctx, args, stdin, stdout, stderr)
	default:
		return fmt.Errorf("%w: unknown command %q", errUsage, cmd)
	}
}

func newFlags(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "package list")
	return fs, file
}

func parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return fmt.Errorf("%w: %v", errUsage, err)
	}
	return nil
}

func prioFlag(fs *flag.FlagSet, def int) *int {
	return fs.Int("prio", def, "highest priority to include, 1 to 4")
}

func checkPrio(prio int) error {
	if prio < catalog.MinPrio || prio > catalog.MaxPrio {
		return fmt.Errorf("%w: --prio must be %d to %d", errUsage, catalog.MinPrio, catalog.MaxPrio)
	}
	return nil
}

// packageFile finds the package list: --file, then $DOTUI_PACKAGES, then
// packages.yaml in chezmoi's source directory.
func packageFile(ctx context.Context, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("DOTUI_PACKAGES"); env != "" {
		return env, nil
	}
	out, err := exec.CommandContext(ctx, "chezmoi", "source-path").Output()
	if err != nil {
		return "", fmt.Errorf("no --file given, and chezmoi source-path failed: %w", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "packages.yaml"), nil
}

func loadCatalog(ctx context.Context, flagValue string) (catalog.Catalog, string, error) {
	path, err := packageFile(ctx, flagValue)
	if err != nil {
		return catalog.Catalog{}, "", err
	}
	c, err := catalog.Load(path)
	return c, path, err
}

func runList(ctx context.Context, args []string, stdout io.Writer) error {
	fs, file := newFlags("list")
	prio := prioFlag(fs, catalog.MaxPrio)
	missingOnly := fs.Bool("missing", false, "only missing packages")
	if err := parse(fs, args); err != nil {
		return err
	}
	if err := checkPrio(*prio); err != nil {
		return err
	}
	c, _, err := loadCatalog(ctx, *file)
	if err != nil {
		return err
	}
	inv, err := brew.List(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATE\tPRIO\tNAME\tKIND\tNOTE")
	for _, p := range c.UpTo(*prio) {
		state := inv.State(p)
		if *missingOnly && state != brew.Missing {
			continue
		}
		fmt.Fprintf(w, "%s\t%d %s\t%s\t%s\t%s\n", stateWord(state), p.Prio, catalog.PrioNames[p.Prio], p.Name, kind(p), p.Note)
	}
	return w.Flush()
}

func runCheck(ctx context.Context, args []string, stdout io.Writer) error {
	fs, file := newFlags("check")
	prio := prioFlag(fs, defaultPrio)
	if err := parse(fs, args); err != nil {
		return err
	}
	if err := checkPrio(*prio); err != nil {
		return err
	}
	c, _, err := loadCatalog(ctx, *file)
	if err != nil {
		return err
	}
	inv, err := brew.List(ctx)
	if err != nil {
		return err
	}

	pkgs := c.UpTo(*prio)
	missing := inv.Missing(pkgs)
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "All %s up to priority %d %s installed.\n", count(len(pkgs), "package", "packages"), *prio, plural(len(pkgs), "is", "are"))
		return nil
	}
	fmt.Fprintf(stdout, "%d of %s up to priority %d %s missing:\n", len(missing), count(len(pkgs), "package", "packages"), *prio, plural(len(missing), "is", "are"))
	for _, p := range missing {
		fmt.Fprintf(stdout, "  %d  %-34s %s\n", p.Prio, p.Name, p.Note)
	}
	fmt.Fprintf(stdout, "Install them with: dotui install --prio %d\n", *prio)
	return errSilent
}

func runInstall(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs, file := newFlags("install")
	prio := prioFlag(fs, defaultPrio)
	if err := parse(fs, args); err != nil {
		return err
	}
	if err := checkPrio(*prio); err != nil {
		return err
	}
	c, _, err := loadCatalog(ctx, *file)
	if err != nil {
		return err
	}

	var wanted []catalog.Package
	if names := fs.Args(); len(names) > 0 {
		for _, name := range names {
			p, ok := c.Find(name)
			if !ok {
				return fmt.Errorf("%w: %s isn't in the package list", errUsage, name)
			}
			wanted = append(wanted, p)
		}
	} else {
		wanted = c.UpTo(*prio)
	}

	inv, err := brew.List(ctx)
	if err != nil {
		return err
	}
	missing := inv.Missing(wanted)
	if len(missing) == 0 {
		fmt.Fprintln(stdout, "Nothing to install.")
		return nil
	}

	plan := brew.PlanInstall(missing)
	for _, brewArgs := range plan.Commands() {
		fmt.Fprintf(stderr, "==> brew %s\n", strings.Join(brewArgs, " "))
		cmd := exec.CommandContext(ctx, "brew", brewArgs...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		if err := cmd.Run(); err != nil {
			if len(plan.Taps) > 0 {
				fmt.Fprintf(stderr, "If brew refused a tap as untrusted, trust it with `brew trust <tap>` and try again.\n")
			}
			return fmt.Errorf("brew %s: %w", brewArgs[0], err)
		}
	}
	fmt.Fprintf(stdout, "Installed %s.\n", count(len(missing), "package", "packages"))
	return nil
}

// runExecThenWait runs a command, then waits for Enter, so its output stays
// readable before the interactive view takes the screen back.
func runExecThenWait(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: _exec needs a command", errUsage)
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(stderr, "\n%s failed: %v\n", filepath.Base(args[0]), err)
	}
	fmt.Fprint(stdout, "\nPress Enter to go back to dotui.")
	_, _ = bufio.NewReader(stdin).ReadString('\n')
	if err != nil {
		return errSilent
	}
	return nil
}

func runTUI(ctx context.Context, args []string) error {
	fs, file := newFlags("dotui")
	source := fs.String("source", "", "chezmoi source directory")
	diskDryRun := fs.Bool("disk-dry-run", false, "disk view: delete nothing")
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("%w: unexpected argument %q", errUsage, fs.Arg(0))
	}
	c, path, err := loadCatalog(ctx, *file)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	newDisk := func() tui.Disk {
		return diskui.New(clean.NewEnv(home), diskui.Options{DryRun: *diskDryRun, Embedded: true})
	}
	m := tui.New(tui.Config{Catalog: c, File: path, Source: *source, Self: self, NewDisk: newDisk})
	_, err = tea.NewProgram(m, tea.WithContext(ctx)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}

func stateWord(s brew.State) string {
	switch s {
	case brew.Installed:
		return "installed"
	case brew.Outside:
		return "installed, not by brew"
	default:
		return "missing"
	}
}

func kind(p catalog.Package) string {
	if p.Cask {
		return "cask"
	}
	return "formula"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func count(n int, one, many string) string {
	return fmt.Sprintf("%d %s", n, plural(n, one, many))
}

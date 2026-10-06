// Package brew asks Homebrew what is installed and plans installs.
package brew

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lechero/dotfiles/dotui/internal/catalog"
)

// State is whether a package is on this machine.
type State int

const (
	Missing State = iota
	Installed
	// Outside means a cask's app is present but Homebrew didn't install it.
	Outside
)

// Inventory is what Homebrew has installed, by short name.
type Inventory struct {
	Formulae map[string]bool
	Casks    map[string]bool
	// AppDirs are searched for the app of a cask Homebrew didn't install.
	AppDirs []string
}

// List runs `brew list` for formulae and casks.
func List(ctx context.Context) (Inventory, error) {
	formulae, err := list(ctx, "--formula")
	if err != nil {
		return Inventory{}, err
	}
	casks, err := list(ctx, "--cask")
	if err != nil {
		return Inventory{}, err
	}
	home, _ := os.UserHomeDir()
	return Inventory{
		Formulae: formulae,
		Casks:    casks,
		AppDirs:  []string{"/Applications", filepath.Join(home, "Applications")},
	}, nil
}

func list(ctx context.Context, kind string) (map[string]bool, error) {
	out, err := exec.CommandContext(ctx, "brew", "list", kind, "-1").Output()
	if err != nil {
		return nil, fmt.Errorf("brew list %s: %w", kind, err)
	}
	return ParseList(string(out)), nil
}

// ParseList turns `brew list -1` output into a set of names.
func ParseList(out string) map[string]bool {
	names := map[string]bool{}
	for line := range strings.Lines(out) {
		if name := strings.TrimSpace(line); name != "" {
			names[name] = true
		}
	}
	return names
}

// State reports whether p is installed.
func (inv Inventory) State(p catalog.Package) State {
	if p.Cask {
		if inv.Casks[p.Short()] {
			return Installed
		}
		if p.App != "" {
			for _, dir := range inv.AppDirs {
				if _, err := os.Stat(filepath.Join(dir, p.App)); err == nil {
					return Outside
				}
			}
		}
		return Missing
	}
	if inv.Formulae[p.Short()] {
		return Installed
	}
	return Missing
}

// Missing returns the packages in pkgs that aren't on this machine.
func (inv Inventory) Missing(pkgs []catalog.Package) []catalog.Package {
	return slices.DeleteFunc(slices.Clone(pkgs), func(p catalog.Package) bool {
		return inv.State(p) != Missing
	})
}

// Plan is the brew commands that install a set of packages: taps first,
// then formulae, then casks.
type Plan struct {
	Taps     []string
	Formulae []string
	Casks    []string
}

// PlanInstall groups packages into the brew commands that install them.
func PlanInstall(pkgs []catalog.Package) Plan {
	var plan Plan
	for _, p := range pkgs {
		if tap := p.Tap(); tap != "" && !slices.Contains(plan.Taps, tap) {
			plan.Taps = append(plan.Taps, tap)
		}
		if p.Cask {
			plan.Casks = append(plan.Casks, p.Name)
		} else {
			plan.Formulae = append(plan.Formulae, p.Name)
		}
	}
	slices.Sort(plan.Taps)
	return plan
}

// Commands returns the brew argument lists for the plan, in order.
func (plan Plan) Commands() [][]string {
	var cmds [][]string
	for _, tap := range plan.Taps {
		cmds = append(cmds, []string{"tap", tap})
	}
	if len(plan.Formulae) > 0 {
		cmds = append(cmds, append([]string{"install"}, plan.Formulae...))
	}
	if len(plan.Casks) > 0 {
		cmds = append(cmds, append([]string{"install", "--cask"}, plan.Casks...))
	}
	return cmds
}

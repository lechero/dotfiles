// Package catalog reads packages.yaml: the Homebrew packages this setup
// installs, each with a priority from 1 (core) to 4 (rarely used).
package catalog

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Priorities run from MinPrio, the packages the dotfiles need, to MaxPrio,
// the ones that are candidates to remove.
const (
	MinPrio = 1
	MaxPrio = 4
)

// PrioNames are short labels for each priority, indexed by priority.
var PrioNames = [...]string{1: "core", 2: "daily", 3: "sometimes", 4: "rarely"}

// Package is one Homebrew formula or cask.
type Package struct {
	// Name is the formula or cask name, tap-qualified for third-party taps,
	// such as charmbracelet/tap/crush.
	Name string `yaml:"name"`
	Prio int    `yaml:"prio"`
	Note string `yaml:"note"`
	Cask bool   `yaml:"cask"`
	// App is the .app bundle a cask installs, used to recognize a cask
	// that was installed without Homebrew.
	App string `yaml:"app"`
}

// Short returns the name without its tap, as `brew list` prints it.
func (p Package) Short() string {
	return p.Name[strings.LastIndex(p.Name, "/")+1:]
}

// Tap returns the package's tap, such as charmbracelet/tap, or "" for
// Homebrew's own.
func (p Package) Tap() string {
	if i := strings.LastIndex(p.Name, "/"); i >= 0 {
		return p.Name[:i]
	}
	return ""
}

// Catalog is the parsed package list, sorted by priority and then name.
type Catalog struct {
	Packages []Package
}

// Load reads and checks a package list.
func Load(path string) (Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	c, err := Parse(data)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Parse decodes and checks a package list.
func Parse(data []byte) (Catalog, error) {
	var file struct {
		Packages []Package `yaml:"packages"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil {
		return Catalog{}, err
	}

	var errs []error
	seen := make(map[string]bool, len(file.Packages))
	for i, p := range file.Packages {
		switch {
		case p.Name == "":
			errs = append(errs, fmt.Errorf("package %d has no name", i+1))
		case seen[p.Name]:
			errs = append(errs, fmt.Errorf("%s is listed twice", p.Name))
		}
		seen[p.Name] = true
		if p.Prio < MinPrio || p.Prio > MaxPrio {
			errs = append(errs, fmt.Errorf("%s: prio is %d, expected %d to %d", p.Name, p.Prio, MinPrio, MaxPrio))
		}
		if p.App != "" && !p.Cask {
			errs = append(errs, fmt.Errorf("%s: app is only for casks", p.Name))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return Catalog{}, err
	}

	pkgs := slices.Clone(file.Packages)
	slices.SortStableFunc(pkgs, func(a, b Package) int {
		return cmp.Or(cmp.Compare(a.Prio, b.Prio), cmp.Compare(a.Name, b.Name))
	})
	return Catalog{Packages: pkgs}, nil
}

// UpTo returns the packages with a priority of at most prio.
func (c Catalog) UpTo(prio int) []Package {
	return slices.DeleteFunc(slices.Clone(c.Packages), func(p Package) bool { return p.Prio > prio })
}

// Find returns the package with the given name, with or without its tap.
func (c Catalog) Find(name string) (Package, bool) {
	for _, p := range c.Packages {
		if p.Name == name || p.Short() == name {
			return p, true
		}
	}
	return Package{}, false
}

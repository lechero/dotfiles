package brew

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Update is an installed package that has a newer version.
type Update struct {
	Name      string
	Installed string
	Latest    string
	Cask      bool
	Pinned    bool
}

// Outdated lists what `brew upgrade` would upgrade, from what Homebrew
// already knows: it doesn't update Homebrew first, so it's quick and
// offline.
func Outdated(ctx context.Context) ([]Update, error) {
	cmd := exec.CommandContext(ctx, "brew", "outdated", "--json=v2")
	cmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("brew outdated: %w", err)
	}
	return ParseOutdated(out)
}

// ParseOutdated reads `brew outdated --json=v2`.
func ParseOutdated(data []byte) ([]Update, error) {
	type entry struct {
		Name              string
		InstalledVersions json.RawMessage `json:"installed_versions"`
		CurrentVersion    string          `json:"current_version"`
		Pinned            bool
	}
	var out struct {
		Formulae []entry
		Casks    []entry
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("brew outdated printed something other than JSON: %w", err)
	}
	var updates []Update
	add := func(e entry, cask bool) {
		updates = append(updates, Update{Name: e.Name, Installed: versions(e.InstalledVersions), Latest: e.CurrentVersion, Cask: cask, Pinned: e.Pinned})
	}
	for _, e := range out.Formulae {
		add(e, false)
	}
	for _, e := range out.Casks {
		add(e, true)
	}
	return updates, nil
}

// versions reads installed_versions, a list for formulae and, in some
// Homebrew versions, a single string for casks.
func versions(raw json.RawMessage) string {
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.Join(list, ", ")
	}
	var one string
	_ = json.Unmarshal(raw, &one) // anything else reads as no version
	return one
}

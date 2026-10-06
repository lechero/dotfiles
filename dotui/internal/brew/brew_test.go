package brew

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lechero/dotfiles/dotui/internal/catalog"
)

func TestParseList(t *testing.T) {
	got := ParseList("fish\ntmux\n\n  crush  \n")
	for _, name := range []string{"fish", "tmux", "crush"} {
		if !got[name] {
			t.Errorf("%s missing from %v", name, got)
		}
	}
	if len(got) != 3 {
		t.Errorf("got %d names, want 3", len(got))
	}
}

func TestState(t *testing.T) {
	apps := t.TempDir()
	if err := os.Mkdir(filepath.Join(apps, "kitty.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	inv := Inventory{
		Formulae: map[string]bool{"fish": true, "crush": true},
		Casks:    map[string]bool{"rancher": true},
		AppDirs:  []string{apps},
	}
	tests := []struct {
		pkg  catalog.Package
		want State
	}{
		{catalog.Package{Name: "fish"}, Installed},
		{catalog.Package{Name: "charmbracelet/tap/crush"}, Installed},
		{catalog.Package{Name: "tmux"}, Missing},
		{catalog.Package{Name: "rancher", Cask: true}, Installed},
		{catalog.Package{Name: "kitty", Cask: true, App: "kitty.app"}, Outside},
		{catalog.Package{Name: "bruno", Cask: true, App: "Bruno.app"}, Missing},
		// A formula and a cask can share a name; only the right kind counts.
		{catalog.Package{Name: "fish", Cask: true}, Missing},
	}
	for _, tt := range tests {
		if got := inv.State(tt.pkg); got != tt.want {
			t.Errorf("State(%s, cask=%v) = %v, want %v", tt.pkg.Name, tt.pkg.Cask, got, tt.want)
		}
	}
}

func TestPlanCommands(t *testing.T) {
	plan := PlanInstall([]catalog.Package{
		{Name: "fish"},
		{Name: "charmbracelet/tap/mods"},
		{Name: "kitty", Cask: true},
		{Name: "charmbracelet/tap/crush"},
		{Name: "dhth/tap/act3"},
	})
	var got []string
	for _, args := range plan.Commands() {
		got = append(got, strings.Join(args, " "))
	}
	want := []string{
		"tap charmbracelet/tap",
		"tap dhth/tap",
		"install fish charmbracelet/tap/mods charmbracelet/tap/crush dhth/tap/act3",
		"install --cask kitty",
	}
	if !slices.Equal(got, want) {
		t.Errorf("commands =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if len(PlanInstall(nil).Commands()) != 0 {
		t.Error("an empty plan has commands")
	}
}

package catalog

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSortsByPrioThenName(t *testing.T) {
	c, err := Parse([]byte(`
packages:
  - {name: zoxide, prio: 1}
  - {name: yq, prio: 3}
  - {name: bat, prio: 1}
  - {name: kitty, cask: true, app: kitty.app, prio: 2}
`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range c.Packages {
		got = append(got, p.Name)
	}
	if want := "bat zoxide kitty yq"; strings.Join(got, " ") != want {
		t.Errorf("order = %v, want %s", got, want)
	}
}

func TestParseRejectsBadEntries(t *testing.T) {
	tests := map[string]struct {
		yaml string
		want string
	}{
		"duplicate":      {"packages:\n  - {name: fd, prio: 1}\n  - {name: fd, prio: 2}\n", "fd is listed twice"},
		"prio too high":  {"packages:\n  - {name: fd, prio: 5}\n", "prio is 5"},
		"prio missing":   {"packages:\n  - {name: fd}\n", "prio is 0"},
		"no name":        {"packages:\n  - {prio: 1}\n", "has no name"},
		"app on formula": {"packages:\n  - {name: fd, prio: 1, app: fd.app}\n", "app is only for casks"},
		"unknown field":  {"packages:\n  - {name: fd, prio: 1, priority: 2}\n", "priority"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestUpTo(t *testing.T) {
	c, err := Parse([]byte("packages:\n  - {name: a, prio: 1}\n  - {name: b, prio: 2}\n  - {name: c, prio: 4}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(c.UpTo(2)); got != 2 {
		t.Errorf("UpTo(2) has %d packages, want 2", got)
	}
	if got := len(c.UpTo(MaxPrio)); got != 3 {
		t.Errorf("UpTo(MaxPrio) has %d packages, want 3", got)
	}
}

func TestTapAndShort(t *testing.T) {
	tests := []struct{ name, tap, short string }{
		{"fish", "", "fish"},
		{"charmbracelet/tap/crush", "charmbracelet/tap", "crush"},
		{"python@3.12", "", "python@3.12"},
	}
	for _, tt := range tests {
		p := Package{Name: tt.name}
		if p.Tap() != tt.tap || p.Short() != tt.short {
			t.Errorf("%s: Tap() = %q, Short() = %q, want %q and %q", tt.name, p.Tap(), p.Short(), tt.tap, tt.short)
		}
	}
}

func TestFind(t *testing.T) {
	c, err := Parse([]byte("packages:\n  - {name: charmbracelet/tap/crush, prio: 3}\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"charmbracelet/tap/crush", "crush"} {
		if _, ok := c.Find(name); !ok {
			t.Errorf("Find(%q) found nothing", name)
		}
	}
	if _, ok := c.Find("tap/crush"); ok {
		t.Error("Find matched a partial tap path")
	}
}

// The repo's own list has to load, since the Taskfile and dotui read it.
func TestRepoPackageList(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "..", "packages.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.UpTo(MinPrio)) == 0 {
		t.Error("the list has no core packages")
	}
}

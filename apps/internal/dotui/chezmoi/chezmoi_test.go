package chezmoi

import (
	"context"
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	out := " M .config/fish/config.fish\nMM .zshrc\n A .config/tmux/My Notes.md\n R install-fish-plugins.sh\n\n"
	got := Parse(out)
	want := []Change{
		{Path: ".config/fish/config.fish", Local: ' ', Apply: 'M'},
		{Path: ".zshrc", Local: 'M', Apply: 'M'},
		{Path: ".config/tmux/My Notes.md", Local: ' ', Apply: 'A'},
		{Path: "install-fish-plugins.sh", Local: ' ', Apply: 'R'},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Parse =\n%v\nwant\n%v", got, want)
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		change   Change
		want     string
		conflict bool
	}{
		{Change{Local: ' ', Apply: 'M'}, "will be updated", false},
		{Change{Local: ' ', Apply: 'A'}, "will be created", false},
		{Change{Local: ' ', Apply: 'R'}, "script will run", false},
		{Change{Local: 'M', Apply: 'M'}, "will be updated, but was changed outside chezmoi", true},
		{Change{Local: 'M', Apply: ' '}, "in sync, but was changed outside chezmoi", false},
	}
	for _, tt := range tests {
		if got := tt.change.Describe(); got != tt.want {
			t.Errorf("Describe(%q%q) = %q, want %q", tt.change.Local, tt.change.Apply, got, tt.want)
		}
		if got := tt.change.Conflict(); got != tt.conflict {
			t.Errorf("Conflict(%q%q) = %v, want %v", tt.change.Local, tt.change.Apply, got, tt.conflict)
		}
	}
}

func TestCommandSource(t *testing.T) {
	ctx := context.Background()
	if got := Command(ctx, "", "diff").Args; !slices.Equal(got, []string{"chezmoi", "diff"}) {
		t.Errorf("Args = %v", got)
	}
	if got := Command(ctx, "/src", "apply").Args; !slices.Equal(got, []string{"chezmoi", "--source", "/src", "apply"}) {
		t.Errorf("Args = %v", got)
	}
}

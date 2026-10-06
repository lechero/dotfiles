// Package chezmoi reads `chezmoi status` and builds chezmoi commands.
package chezmoi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Change is one line of `chezmoi status`.
type Change struct {
	Path string
	// Local is the first column: how the file changed since chezmoi last
	// wrote it. A space means it didn't.
	Local byte
	// Apply is the second column: what `chezmoi apply` would do.
	Apply byte
}

// Describe says in words what applying would do.
func (c Change) Describe() string {
	var what string
	switch c.Apply {
	case 'A':
		what = "will be created"
	case 'D':
		what = "will be removed"
	case 'M':
		what = "will be updated"
	case 'R':
		what = "script will run"
	default:
		what = "in sync"
	}
	if c.Local != ' ' {
		what += ", but was changed outside chezmoi"
	}
	return what
}

// Conflict reports whether applying would overwrite a change made outside
// chezmoi.
func (c Change) Conflict() bool {
	return c.Local != ' ' && c.Apply != ' '
}

// Parse reads `chezmoi status` output.
func Parse(out string) []Change {
	var changes []Change
	for line := range strings.Lines(out) {
		line = strings.TrimRight(line, "\r\n")
		if len(line) < 4 {
			continue
		}
		changes = append(changes, Change{Local: line[0], Apply: line[1], Path: line[3:]})
	}
	return changes
}

// Status runs `chezmoi status`, against source if it isn't empty.
func Status(ctx context.Context, source string) ([]Change, error) {
	cmd := Command(ctx, source, "status")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("chezmoi status: %w", err)
	}
	return Parse(string(out)), nil
}

// Command builds a chezmoi command, against source if it isn't empty.
func Command(ctx context.Context, source string, args ...string) *exec.Cmd {
	if source != "" {
		args = append([]string{"--source", source}, args...)
	}
	return exec.CommandContext(ctx, "chezmoi", args...)
}

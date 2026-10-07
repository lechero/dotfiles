package work

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner runs a CLI and returns what it printed on stdout. Services take one
// so tests can answer for the CLI.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// Exec is the real Runner. It never prompts, since nobody is there to
// answer, and a failure carries the first line the command wrote to stderr.
func Exec(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "NO_PROMPT=1", "NO_COLOR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := firstLine(stderr.String()); msg != "" {
			msg = strings.TrimPrefix(msg, name+": ") // glab names itself already
			return out, fmt.Errorf("%s: %s", name, msg)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// RunJSON runs a CLI and decodes its JSON output into v.
func RunJSON(ctx context.Context, run Runner, v any, name string, args ...string) error {
	out, err := run(ctx, name, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("%s printed something other than the JSON expected: %w", name, err)
	}
	return nil
}

func firstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			const maxLen = 200
			if len(line) > maxLen {
				line = line[:maxLen] + "…"
			}
			return line
		}
	}
	return ""
}

// Package dockertest fakes the docker command for tests: a script that
// answers from canned output and writes down every call.
package dockertest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Fake is a docker command that answers by the leading words of its
// arguments ("system df", "container inspect", "rm"…).
type Fake struct {
	Path    string // the script
	dir     string
	keys    []string
	log     string
	replies map[string]string
}

// New writes a fake docker. The longest leading-words key that matches a call
// picks its reply; a reply starting with "!" fails, printing the rest to
// stderr. Calls nothing matches print nothing and succeed.
func New(t testing.TB, replies map[string]string) *Fake {
	t.Helper()
	dir := t.TempDir()
	f := &Fake{Path: filepath.Join(dir, "docker"), dir: dir, log: filepath.Join(dir, "calls"), replies: map[string]string{}}
	for k, v := range replies {
		f.replies[k] = v
		f.keys = append(f.keys, k)
	}
	// Longest first, so "builder prune" isn't answered as "builder".
	sort.Slice(f.keys, func(i, j int) bool { return len(f.keys[i]) > len(f.keys[j]) })
	var b strings.Builder
	fmt.Fprintf(&b, "#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\ncase \"$*\" in\n", f.log)
	for i, k := range f.keys {
		file := filepath.Join(dir, fmt.Sprintf("reply-%d", i))
		fmt.Fprintf(&b, "  %q*)\n    if [ \"$(head -c1 %q)\" = '!' ]; then tail -c +2 %q >&2; exit 1; fi\n    cat %q; exit 0;;\n",
			k, file, file, file)
	}
	b.WriteString("esac\nexit 0\n")
	if err := os.WriteFile(f.Path, []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	for k, v := range replies {
		f.Set(t, k, v)
	}
	return f
}

// Set changes the reply to calls starting with key, which New must know.
func (f *Fake) Set(t testing.TB, key, reply string) {
	t.Helper()
	for i, k := range f.keys {
		if k == key {
			if err := os.WriteFile(filepath.Join(f.dir, fmt.Sprintf("reply-%d", i)), []byte(reply), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("dockertest: no reply for %q was set up", key)
}

// Calls lists every call so far, each as its arguments joined by spaces.
func (f *Fake) Calls() []string {
	b, _ := os.ReadFile(f.log)
	s := strings.TrimSpace(string(b))
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// Changes lists the calls that would change something: everything but reads
// and the throwaway container that measures the disk.
func (f *Fake) Changes() []string {
	var out []string
	for _, c := range f.Calls() {
		switch {
		case strings.HasPrefix(c, "system df"), strings.HasPrefix(c, "info"), strings.HasPrefix(c, "version"),
			strings.Contains(c, " inspect"), strings.HasPrefix(c, "run --rm --pull never"):
		default:
			out = append(out, c)
		}
	}
	return out
}

// Command stands in for exec.CommandContext, sending docker to the fake.
func (f *Fake) Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	if name == "docker" {
		name = f.Path
	}
	return exec.CommandContext(ctx, name, args...)
}

// LookPath finds the fake for docker, and nothing else.
func (f *Fake) LookPath(name string) (string, error) {
	if name == "docker" {
		return f.Path, nil
	}
	return "", exec.ErrNotFound
}

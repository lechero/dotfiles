package worktrees

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// gitEnv keeps tests away from your own git config (signing, hooks, aliases).
var gitEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0",
	"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
	"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
}

func gitIn(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), gitEnv...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), gitEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, p, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commit writes a file and commits it.
func commit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	write(t, filepath.Join(dir, file), content)
	mustGit(t, dir, "add", file)
	mustGit(t, dir, "commit", "-q", "-m", msg)
}

type fixture struct {
	home, repo string
	wt         map[string]string // name → path
}

// newFixture builds a repo with an origin and one worktree per case.
func newFixture(t *testing.T) fixture {
	t.Helper()
	home := t.TempDir()
	if r, err := filepath.EvalSymlinks(home); err == nil {
		home = r // /var → /private/var, as git reports it
	}
	origin := filepath.Join(home, "origin.git")
	repo := filepath.Join(home, "projects", "app")
	os.MkdirAll(repo, 0o755)
	mustGit(t, home, "init", "-q", "--bare", "-b", "main", origin)
	mustGit(t, repo, "init", "-q", "-b", "main")
	commit(t, repo, "README.md", "hello\n", "initial")
	write(t, filepath.Join(repo, ".gitignore"), "node_modules/\n.env.local\n")
	mustGit(t, repo, "add", ".gitignore")
	mustGit(t, repo, "commit", "-q", "-m", "ignore")
	mustGit(t, repo, "remote", "add", "origin", origin)
	mustGit(t, repo, "push", "-q", "-u", "origin", "main")
	mustGit(t, repo, "remote", "set-head", "origin", "main")

	f := fixture{home: home, repo: repo, wt: map[string]string{}}
	add := func(name string) string {
		p := filepath.Join(repo, ".claude", "worktrees", name)
		mustGit(t, repo, "worktree", "add", "-q", "-b", "claude/"+name, p)
		f.wt[name] = p
		return p
	}

	// Fast-forward merged into main and pushed.
	p := add("merged")
	commit(t, p, "a.txt", "a\n", "feature a")
	mustGit(t, repo, "merge", "-q", "--ff-only", "claude/merged")
	mustGit(t, repo, "push", "-q", "origin", "main")

	// Squash-merged: main has the change, but not these commits.
	p = add("squashed")
	commit(t, p, "b.txt", "b1\n", "b part 1")
	commit(t, p, "b.txt", "b1\nb2\n", "b part 2")
	mustGit(t, p, "push", "-q", "origin", "claude/squashed")
	mustGit(t, repo, "merge", "-q", "--squash", "claude/squashed")
	mustGit(t, repo, "commit", "-q", "-m", "b (squashed)")
	mustGit(t, repo, "push", "-q", "origin", "main")

	// Squash-merged, and a commit landed after the PR.
	p = add("after-pr")
	commit(t, p, "c.txt", "c\n", "c")
	mustGit(t, p, "push", "-q", "origin", "claude/after-pr")
	mustGit(t, repo, "merge", "-q", "--squash", "claude/after-pr")
	mustGit(t, repo, "commit", "-q", "-m", "c (squashed)")
	mustGit(t, repo, "push", "-q", "origin", "main")
	commit(t, p, "c2.txt", "more\n", "after the PR")

	p = add("dirty")
	write(t, filepath.Join(p, "README.md"), "changed\n")

	p = add("untracked")
	write(t, filepath.Join(p, "notes.txt"), "draft\n")

	p = add("precious")
	write(t, filepath.Join(p, ".env.local"), "SECRET=1\n")
	write(t, filepath.Join(p, "node_modules", "x", "index.js"), "x\n")

	// An agent copied the main checkout's .env.local in: nothing of its own.
	write(t, filepath.Join(repo, ".env.local"), "SHARED=1\n")
	p = add("copied-env")
	write(t, filepath.Join(p, ".env.local"), "SHARED=1\n")

	p = add("old")
	commit(t, p, "d.txt", "d\n", "old work")
	mustGit(t, p, "push", "-q", "origin", "claude/old")

	p = add("unpushed")
	commit(t, p, "e.txt", "e\n", "local only")

	p = add("gone")
	os.RemoveAll(p)

	p = add("locked")
	mustGit(t, repo, "worktree", "lock", p)

	add("busy")

	p = filepath.Join(repo, ".claude", "worktrees", "detached")
	mustGit(t, repo, "worktree", "add", "-q", "--detach", p, "main")
	f.wt["detached"] = p
	return f
}

func audit(t *testing.T, f fixture, prs map[string]*PR, now time.Time) map[string]Worktree {
	t.Helper()
	d := Deps{
		Home: f.home,
		Now:  now,
		Git:  gitIn,
		PR: func(_ context.Context, _, branch string) (*PR, error) {
			return prs[branch], nil
		},
		InUse: func(p string) string {
			if p == f.wt["busy"] {
				return "claude (pid 42)"
			}
			return ""
		},
	}
	out := map[string]Worktree{}
	for _, w := range Audit(context.Background(), []string{f.repo}, d) {
		out[w.Name()] = w
	}
	return out
}

func TestVerdicts(t *testing.T) {
	f := newFixture(t)
	squashHead := mustGit(t, f.wt["squashed"], "rev-parse", "HEAD")
	afterPRHead := mustGit(t, f.wt["after-pr"], "rev-parse", "HEAD~1") // the PR's head, before the late commit
	prs := map[string]*PR{
		"claude/squashed": {Number: 7, State: "MERGED", HeadOID: squashHead},
		"claude/after-pr": {Number: 8, State: "MERGED", HeadOID: afterPRHead},
	}
	got := audit(t, f, prs, time.Now().Add(60*24*time.Hour)) // two months on: "old" applies

	want := map[string]Verdict{
		"merged":     Merged, // fast-forwarded into main
		"squashed":   Merged, // the PR says so, though git cherry cannot tell
		"after-pr":   Review, // merged PR, then another commit
		"dirty":      Keep,
		"untracked":  Keep,
		"precious":   Review, // .env.local is ignored but not build output
		"copied-env": Merged, // its .env.local is the main checkout's, byte for byte
		"old":        Old,    // pushed, untouched, no PR
		"unpushed":   Review, // its commit exists only here
		"gone":       Stale,
		"locked":     Keep,
		"busy":       Keep,
		"detached":   Merged,
	}
	for name, v := range want {
		w, ok := got[name]
		if !ok {
			t.Errorf("%s: not listed", name)
			continue
		}
		if w.Verdict != v {
			t.Errorf("%s: verdict %s, want %s (reasons %v)", name, w.Verdict, v, w.Reasons)
		}
	}
	if w := got["precious"]; !slices.Equal(w.Ignored, []string{".env.local"}) {
		t.Errorf("precious ignored files = %v; node_modules is build output", w.Ignored)
	}
	if w := got["squashed"]; w.Unique != 2 || w.Branch != "claude/squashed" {
		t.Errorf("squashed: unique %d branch %q; cherry should still see 2 commits", w.Unique, w.Branch)
	}
	if w := got["dirty"]; w.Changed != 1 || !strings.Contains(strings.Join(w.Reasons, ";"), "1 changed file") {
		t.Errorf("dirty: %+v", w.Reasons)
	}
	if w := got["busy"]; !strings.Contains(strings.Join(w.Reasons, ";"), "claude (pid 42)") {
		t.Errorf("busy should name what uses it: %v", w.Reasons)
	}
	if w := got["detached"]; w.Branch != "" || w.DeletableBranch() {
		t.Error("a detached worktree has no branch to delete")
	}
	if !got["squashed"].DeletableBranch() || got["old"].DeletableBranch() {
		t.Error("only merged work may take its branch along")
	}
}

func TestWithoutThePRASquashMergeNeedsReview(t *testing.T) {
	f := newFixture(t)
	got := audit(t, f, nil, time.Now())
	if w := got["squashed"]; w.Verdict != Review {
		t.Errorf("without the PR, a squash merge is not provably merged: %s %v", w.Verdict, w.Reasons)
	}
	if w := got["old"]; w.Verdict != Review {
		t.Errorf("recent work is not old: %s", w.Verdict)
	}
}

func TestParseList(t *testing.T) {
	out := "worktree /r\nHEAD aaa\nbranch refs/heads/main\n\n" +
		"worktree /r/.claude/worktrees/x\nHEAD bbb\ndetached\n\n" +
		"worktree /tmp/y\nHEAD ccc\nbranch refs/heads/claude/y\nlocked busy\nprunable gitdir file points to non-existent location\n"
	list := parseList(out)
	if len(list) != 3 || list[1].Branch != "" || list[2].Branch != "claude/y" || !list[2].Locked || !list[2].Prunable {
		t.Errorf("parseList = %+v", list)
	}
}

func TestPrecious(t *testing.T) {
	cases := map[string]bool{
		".env.local": true, "apps/web/var/uploads": true,
		"node_modules": false, "apps/web/.next": false, "debug.log": false,
		"packages/ui/tsconfig.tsbuildinfo": false, ".claude/settings.local.json": false,
		".husky/_": false, "apps/web/next-env.d.ts": false, "test-results": false,
		"godot/georginauts/.godot": false, ".pnpm-store": false,
	}
	for p, want := range cases {
		if got := precious(p); got != want {
			t.Errorf("precious(%q) = %v, want %v", p, got, want)
		}
	}
}

package clean

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"manage-disk/internal/worktrees"
)

var testGitEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
	"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
	"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
}

func tgit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), testGitEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repoWithWorktrees makes ~/projects/app with a merged, a dirty-later and a
// vanished worktree, and returns the audit keyed by worktree name.
func repoWithWorktrees(t *testing.T) (*Env, string, map[string]worktrees.Worktree) {
	t.Helper()
	env := testEnv(t, nil)
	home, err := filepath.EvalSymlinks(env.Home)
	if err != nil {
		t.Fatal(err)
	}
	env.Home, env.Guard = home, Guard{Home: home}
	env.StateDir, env.LogDir = home+"/state", home+"/logs"
	repo := home + "/projects/app"
	os.MkdirAll(repo, 0o755)
	tgit(t, repo, "init", "-q", "-b", "main")
	os.WriteFile(repo+"/README.md", []byte("hi\n"), 0o644)
	tgit(t, repo, "add", ".")
	tgit(t, repo, "commit", "-q", "-m", "initial")
	for _, name := range []string{"merged", "later-dirty", "gone"} {
		tgit(t, repo, "worktree", "add", "-q", "-b", "claude/"+name, repo+"/.claude/worktrees/"+name)
	}
	os.RemoveAll(repo + "/.claude/worktrees/gone")

	got := map[string]worktrees.Worktree{}
	for _, w := range AuditWorktrees(context.Background(), env, false) {
		got[w.Name()] = w
	}
	return env, repo, got
}

func TestWorktreeRemovalKeepsBranchesAndNeverForces(t *testing.T) {
	env, repo, got := repoWithWorktrees(t)
	merged, dirty, gone := got["merged"], got["later-dirty"], got["gone"]
	if merged.Verdict != worktrees.Merged || dirty.Verdict != worktrees.Merged || gone.Verdict != worktrees.Stale {
		t.Fatalf("verdicts: merged=%s later-dirty=%s gone=%s", merged.Verdict, dirty.Verdict, gone.Verdict)
	}
	if merged.PRError == "" {
		t.Error("without gh the audit should say PRs could not be checked")
	}

	// A change lands after the audit; git must refuse, and nothing may force it.
	os.WriteFile(dirty.Path+"/README.md", []byte("edited\n"), 0o644)

	task := WorktreeTask(false)
	sels := []Selection{{Task: task, Items: []Item{
		WorktreeItem(merged, 1), WorktreeItem(dirty, 1), WorktreeItem(gone, 0),
	}}}
	sum := Run(context.Background(), env, sels, false, func(Event) {})

	if _, err := os.Stat(merged.Path); !os.IsNotExist(err) {
		t.Error("the merged worktree should be gone")
	}
	if tgit(t, repo, "branch", "--list", "claude/merged") == "" {
		t.Error("its branch should stay unless asked")
	}
	if _, err := os.Stat(dirty.Path + "/README.md"); err != nil {
		t.Error("a worktree with changes must survive")
	}
	if sum.Failed != 1 || len(sum.Cleaned) != 2 {
		t.Errorf("summary: %d cleaned, %d failed; want 2 and 1", len(sum.Cleaned), sum.Failed)
	}
	if list := tgit(t, repo, "worktree", "list"); strings.Contains(list, "/gone") {
		t.Errorf("the vanished worktree should be pruned:\n%s", list)
	}
}

func TestWorktreeRemovalCanTakeMergedBranches(t *testing.T) {
	env, repo, got := repoWithWorktrees(t)
	sels := []Selection{{Task: WorktreeTask(true), Items: []Item{WorktreeItem(got["merged"], 1)}}}
	Run(context.Background(), env, sels, false, func(Event) {})
	if tgit(t, repo, "branch", "--list", "claude/merged") != "" {
		t.Error("with branches on, a merged worktree's branch goes too")
	}
	if tgit(t, repo, "branch", "--list", "main") == "" {
		t.Error("main must never be touched")
	}
}

func TestKeptWorktreesAreNeverRemoved(t *testing.T) {
	env, _, got := repoWithWorktrees(t)
	w := got["merged"]
	w.Verdict, w.Reasons = worktrees.Keep, []string{"in use by claude (pid 1)"}
	sum := Run(context.Background(), env, []Selection{{Task: WorktreeTask(true), Items: []Item{WorktreeItem(w, 1)}}}, false, func(Event) {})
	if sum.Failed != 1 {
		t.Errorf("a kept worktree must fail to remove, summary %+v", sum)
	}
	if _, err := os.Stat(w.Path); err != nil {
		t.Error("a kept worktree must survive")
	}
}

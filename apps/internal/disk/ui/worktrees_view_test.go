package ui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
)

var uiGitEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
	"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
	"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
}

func uiGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), uiGitEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// worktreeApp is an app whose home holds ~/projects/app with a worktree whose
// work is merged and one with a changed file.
func worktreeApp(t *testing.T, opts Options) (*app, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := clean.NewEnv(home)
	env.SnapshotFn = func(context.Context) (clean.Procs, error) { return nil, nil }
	env.LookPath = func(string) (string, error) { return "", errors.New("not in tests") }
	a := New(env, opts).a
	a.update(tea.WindowSizeMsg{Width: 140, Height: 40})

	repo := home + "/projects/app"
	os.MkdirAll(repo, 0o755)
	uiGit(t, repo, "init", "-q", "-b", "main")
	os.WriteFile(repo+"/README.md", []byte("hi\n"), 0o644)
	uiGit(t, repo, "add", ".")
	uiGit(t, repo, "commit", "-q", "-m", "initial")
	for _, name := range []string{"merged", "dirty"} {
		uiGit(t, repo, "worktree", "add", "-q", "-b", "claude/"+name, repo+"/.claude/worktrees/"+name)
	}
	os.WriteFile(repo+"/.claude/worktrees/dirty/README.md", []byte("work in progress\n"), 0o644)
	drive(t, a, a.auditWorktrees(false))
	return a, repo
}

func TestWorktreesTabListsJudgesAndRemoves(t *testing.T) {
	a, repo := worktreeApp(t, Options{})
	merged, dirty := repo+"/.claude/worktrees/merged", repo+"/.claude/worktrees/dirty"
	press(t, a, "5")
	out := a.view()
	for _, want := range []string{"2 worktrees in 1 repos", "✓ merged", "✗ keep", "app › merged", "1 changed file"} {
		if !strings.Contains(out, want) {
			t.Errorf("worktrees tab is missing %q:\n%s", want, out)
		}
	}
	if !a.wt.chosen[merged] || a.wt.chosen[dirty] {
		t.Fatalf("verified work should be picked, kept work not: %v", a.wt.chosen)
	}

	// Picking a kept worktree explains instead of picking it.
	for i, it := range a.wt.list.Items() {
		if it.(wtEntry).w.Path == dirty {
			a.wt.list.Select(i)
		}
	}
	press(t, a, " ")
	if a.wt.chosen[dirty] || !strings.Contains(a.flash, "1 changed file") {
		t.Errorf("a kept worktree cannot be picked: chosen %v, flash %q", a.wt.chosen[dirty], a.flash)
	}

	press(t, a, "c") // checks again, then asks
	if a.wt.mode != wtConfirm || !strings.Contains(a.view(), "Remove 1 worktree") {
		t.Fatalf("c should re-check and ask: mode %d\n%s", a.wt.mode, a.view())
	}
	press(t, a, "y")
	if a.wt.mode != wtDone {
		t.Fatalf("removal did not finish: mode %d", a.wt.mode)
	}
	if _, err := os.Stat(merged); !os.IsNotExist(err) {
		t.Error("the merged worktree should be gone")
	}
	if _, err := os.Stat(dirty + "/README.md"); err != nil {
		t.Error("the worktree with changes must stay")
	}
	if uiGit(t, repo, "branch", "--list", "claude/merged") == "" {
		t.Error("branches stay unless b was pressed")
	}
	press(t, a, "enter")
	if a.wt.mode != wtList || len(a.wt.all) != 1 {
		t.Errorf("after the summary the list is checked again: mode %d, %d left", a.wt.mode, len(a.wt.all))
	}
}

func TestWorktreesDryRunRemovesNothing(t *testing.T) {
	a, repo := worktreeApp(t, Options{DryRun: true})
	press(t, a, "5", "b", "c", "y")
	if _, err := os.Stat(repo + "/.claude/worktrees/merged"); err != nil {
		t.Error("a dry run removed a worktree")
	}
	if uiGit(t, repo, "branch", "--list", "claude/merged") == "" {
		t.Error("a dry run deleted a branch")
	}
	if !strings.Contains(a.view(), "Dry run finished") {
		t.Errorf("done view:\n%s", a.view())
	}
}

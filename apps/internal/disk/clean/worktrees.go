package clean

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/disk/worktrees"
)

// git runs a git command in dir without taking optional locks, so a status
// never fights a running agent's own git work. Output includes stderr, for
// error messages.
func (e *Env) git(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := e.Command(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), gitQuiet...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// gitOut is git for reading: stdout only, so warnings can't corrupt a parse.
func (e *Env) gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := e.Command(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), gitQuiet...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

var errNoGH = errors.New("gh isn't installed, so PRs can't be checked")

// pullRequest asks GitHub, through gh, for the PR whose head is branch:
// a merged one if any, else an open one, else the newest.
func (e *Env) pullRequest(ctx context.Context, repo, branch string) (*worktrees.PR, error) {
	if _, err := e.LookPath("gh"); err != nil {
		return nil, errNoGH
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := e.Command(ctx, "gh", "pr", "list", "--head", branch, "--state", "all",
		"--json", "number,state,title,url,headRefOid", "--limit", "10")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "NO_COLOR=1", "GH_NO_UPDATE_NOTIFIER=1")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("gh: %s", strings.TrimSpace(headLine(string(ee.Stderr))))
		}
		return nil, fmt.Errorf("gh: %v", err)
	}
	var prs []worktrees.PR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("gh: %v", err)
	}
	for _, want := range []string{"MERGED", "OPEN"} {
		for i := range prs {
			if prs[i].State == want {
				return &prs[i], nil
			}
		}
	}
	if len(prs) > 0 {
		return &prs[0], nil
	}
	return nil, nil
}

// AuditWorktrees lists the linked worktrees of every repo under ~/projects
// with the evidence about each. With fetch, it first updates each repo's view
// of its origin, so "merged" is judged against today's main.
func AuditWorktrees(ctx context.Context, env *Env, fetch bool) []worktrees.Worktree {
	repos := gitRepos(ctx, env.Home+"/projects", 3)
	if fetch {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for _, repo := range repos {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if _, err := env.gitOut(ctx, repo, "remote", "get-url", "origin"); err == nil {
					env.git(ctx, repo, "fetch", "--quiet", "--prune", "origin")
				}
			}()
		}
		wg.Wait()
	}
	procs := env.Procs(ctx)
	return worktrees.Audit(ctx, repos, worktrees.Deps{
		Home: env.Home,
		Now:  env.Now(),
		Git:  env.gitOut,
		PR:   env.pullRequest,
		InUse: func(path string) string {
			if p := procs.Using(path); p != nil {
				return fmt.Sprintf("%s (pid %d)", p.Name(), p.PID)
			}
			return ""
		},
	})
}

// WorktreeItem wraps a worktree for a removal run.
func WorktreeItem(w worktrees.Worktree, size int64) Item {
	return Item{
		Paths:    []string{w.Path},
		Label:    w.RepoName() + " › " + w.Name(),
		Size:     size,
		Selected: true,
		Note:     w.Verdict.String() + ": " + strings.Join(w.Reasons, "; "),
		Data:     w,
	}
}

// WorktreeTask removes worktrees with `git worktree remove`, which refuses
// any worktree with changes — there is no --force here. Branches stay unless
// deleteBranches is set, and then only for work already merged.
func WorktreeTask(deleteBranches bool) *Task {
	return &Task{
		ID:    "worktrees",
		Title: "Worktrees",
		Tier:  Tier2,
		About: "Removes the chosen worktrees with `git worktree remove` (never --force: git refuses " +
			"one with changes) and prunes git's entries for folders already gone.",
		Cost: "A Claude or Codex session that lived in a removed worktree cannot be resumed there.",
		Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
			w, ok := it.Data.(worktrees.Worktree)
			if !ok {
				return errors.New("not a worktree")
			}
			run := func(args ...string) error {
				log("$ git " + strings.Join(args, " "))
				out, err := env.git(ctx, w.Repo, args...)
				for _, line := range strings.Split(out, "\n") {
					if line = strings.TrimSpace(line); line != "" {
						log(line)
					}
				}
				if err != nil && out != "" {
					return fmt.Errorf("git %s: %s", args[0]+" "+args[1], headLine(out))
				}
				return err
			}
			if w.Verdict == worktrees.Stale {
				return run("worktree", "prune", "--verbose")
			}
			if !w.Verdict.Removable() {
				return errors.New("kept: " + strings.Join(w.Reasons, "; "))
			}
			if err := env.Guard.Check(w.Path); err != nil {
				return err
			}
			if err := run("worktree", "remove", w.Path); err != nil {
				return err
			}
			if deleteBranches && w.DeletableBranch() {
				return run("branch", "-D", w.Branch)
			}
			return nil
		},
	}
}

// headLine is the first line of a command's output.
func headLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// gitQuiet keeps git from taking optional locks or asking for anything: a
// fetch whose SSH key wants a passphrase fails instead of prompting mid-TUI.
var gitQuiet = []string{
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=10",
}

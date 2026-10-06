package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"manage-disk/internal/clean"
	"manage-disk/internal/human"
	"manage-disk/internal/worktrees"
)

// listWorktrees prints every linked worktree under ~/projects with its verdict
// and the evidence for it. It changes nothing (with fetch, it runs git fetch).
func listWorktrees(w io.Writer, env *clean.Env, fetch bool) error {
	if fetch {
		fmt.Fprintln(os.Stderr, "fetching origin for each repo…")
	}
	all := clean.AuditWorktrees(context.Background(), env, fetch)
	if len(all) == 0 {
		fmt.Fprintln(w, "No linked worktrees under ~/projects.")
		return nil
	}
	repos := map[string]bool{}
	for _, wt := range all {
		repos[wt.Repo] = true
	}
	fmt.Fprintf(w, "%d worktrees in %d repos\n", len(all), len(repos))

	now := time.Now()
	for v := worktrees.Stale; v >= worktrees.Keep; v-- {
		var group []worktrees.Worktree
		for _, wt := range all {
			if wt.Verdict == v {
				group = append(group, wt)
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s (%d) — %s\n", strings.ToUpper(v.String()), len(group), verdictMeaning[v])
		for _, wt := range group {
			fmt.Fprintf(w, "  %-58s %s\n", truncate(wt.RepoName()+" › "+wt.Name(), 58), describe(wt, now))
			fmt.Fprintf(w, "  %-58s %s\n", "", strings.Join(wt.Reasons, "; "))
		}
	}
	if fetched := oldestFetch(all); !fetched.IsZero() && now.Sub(fetched) > 24*time.Hour {
		fmt.Fprintf(w, "\nSome repos last fetched %s; `manage-disk worktrees --fetch` checks against today's main.\n", human.Ago(now, fetched))
	}
	return nil
}

var verdictMeaning = map[worktrees.Verdict]string{
	worktrees.Stale:  "the folder is gone; git still lists it",
	worktrees.Merged: "its work is already in the default branch",
	worktrees.Old:    "clean, pushed, no open PR, untouched 30+ days",
	worktrees.Review: "removable without losing commits, but look first",
	worktrees.Keep:   "in use, locked, or has changes",
}

func describe(wt worktrees.Worktree, now time.Time) string {
	var parts []string
	if wt.Branch != "" {
		parts = append(parts, wt.Branch)
	} else if len(wt.Head) >= 7 {
		parts = append(parts, "detached "+wt.Head[:7])
	}
	if wt.PR != nil {
		parts = append(parts, fmt.Sprintf("PR #%d %s", wt.PR.Number, strings.ToLower(wt.PR.State)))
	}
	if !wt.LastCommit.IsZero() {
		parts = append(parts, human.Ago(now, wt.LastCommit))
	}
	return strings.Join(parts, " · ")
}

func oldestFetch(all []worktrees.Worktree) time.Time {
	var oldest time.Time
	for _, wt := range all {
		if !wt.Fetched.IsZero() && (oldest.IsZero() || wt.Fetched.Before(oldest)) {
			oldest = wt.Fetched
		}
	}
	return oldest
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

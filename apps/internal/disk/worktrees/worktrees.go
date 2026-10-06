// Package worktrees lists the linked git worktrees of your repos and judges,
// from evidence, whether removing one could lose anything.
//
// Removing a worktree with `git worktree remove` keeps its branch and every
// commit; what it can lose is work that never became a commit (changed,
// untracked, or ignored-but-precious files) and the checkout a running
// program sits in. Deleting the branch as well is the step that can lose
// commits, so that is only offered for work already in the default branch.
package worktrees

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Verdict is what the evidence says about removing a worktree, safest last.
type Verdict int

const (
	Keep   Verdict = iota // in use, locked, has changes, or outside your home
	Review                // removable without losing commits, but look first
	Old                   // clean, pushed, no open PR, untouched for a while
	Merged                // its work is already in the default branch
	Stale                 // the folder is gone; only git's bookkeeping is left
)

func (v Verdict) String() string {
	return [...]string{"keep", "review", "old", "merged", "stale"}[v]
}

// Removable reports whether the worktree may be removed at all.
func (v Verdict) Removable() bool { return v != Keep }

// Preselect reports whether the evidence is strong enough to pick it for you.
func (v Verdict) Preselect() bool { return v >= Old }

// PR is a pull request whose head is the worktree's branch.
type PR struct {
	Number  int    `json:"number"`
	State   string `json:"state"` // OPEN, MERGED or CLOSED
	Title   string `json:"title"`
	URL     string `json:"url"`
	HeadOID string `json:"headRefOid"`
}

// Worktree is one linked worktree and the evidence about it.
type Worktree struct {
	Repo     string // the main checkout
	Path     string
	Branch   string // empty when HEAD is detached
	Head     string // commit id
	Locked   bool
	Prunable bool // its folder is gone

	Base       string // what "merged" is measured against, e.g. origin/main
	Fetched    time.Time
	LastCommit time.Time
	Subject    string

	Changed   int      // tracked files with changes
	Untracked int      // new files git does not know
	Ignored   []string // ignored files that are not build output (.env.local, var/…)

	Ahead, Behind int // commits relative to Base
	Unique        int // commits whose change is not in Base (git cherry "+")
	Unpushed      int // commits only this branch holds: on no remote, not in Base
	PR            *PR
	PRError       string
	HeadInPR      bool   // HEAD is the PR's head or behind it
	InUse         string // the program working in it, if any

	Verdict Verdict
	Reasons []string
}

// Name is the worktree's folder name, with its parent when the folder is just
// the repo's name again (Codex keeps …/worktrees/<id>/<repo>).
func (w Worktree) Name() string {
	name := filepath.Base(w.Path)
	if name == filepath.Base(w.Repo) {
		return filepath.Base(filepath.Dir(w.Path)) + "/" + name
	}
	return name
}

// RepoName is the main checkout's folder name.
func (w Worktree) RepoName() string { return filepath.Base(w.Repo) }

// BaseBranch is Base without its remote: origin/main → main.
func (w Worktree) BaseBranch() string {
	if i := strings.IndexByte(w.Base, '/'); i >= 0 {
		return w.Base[i+1:]
	}
	return w.Base
}

// Deps is how the audit reaches git, GitHub and the process table.
type Deps struct {
	Home string
	Now  time.Time
	// StaleAfter is how long untouched counts as old (default 30 days).
	StaleAfter time.Duration
	Git        func(ctx context.Context, dir string, args ...string) (string, error)
	// PR finds the pull request for a branch; nil skips the lookup.
	PR func(ctx context.Context, repo, branch string) (*PR, error)
	// InUse names a program working inside path, or returns "".
	InUse func(path string) string
}

// Audit lists every linked worktree of repos with its evidence and verdict,
// repo by repo, most removable first.
func Audit(ctx context.Context, repos []string, d Deps) []Worktree {
	if d.StaleAfter == 0 {
		d.StaleAfter = 30 * 24 * time.Hour
	}
	var all []Worktree
	for _, repo := range repos {
		all = append(all, auditRepo(ctx, repo, d)...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Verdict != all[j].Verdict {
			return all[i].Verdict > all[j].Verdict
		}
		if all[i].Repo != all[j].Repo {
			return all[i].Repo < all[j].Repo
		}
		return all[i].Path < all[j].Path
	})
	return all
}

func auditRepo(ctx context.Context, repo string, d Deps) []Worktree {
	out, err := d.Git(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}
	list := parseList(out)
	if len(list) <= 1 {
		return nil // only the main checkout
	}
	base := defaultBase(ctx, repo, d)
	var fetched time.Time
	if gitDir, err := d.Git(ctx, repo, "rev-parse", "--git-common-dir"); err == nil {
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(repo, gitDir)
		}
		if fi, err := os.Stat(filepath.Join(gitDir, "FETCH_HEAD")); err == nil {
			fetched = fi.ModTime()
		}
	}

	wts := list[1:] // the first entry is the main checkout
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range wts {
		w := &wts[i]
		w.Repo, w.Base, w.Fetched = repo, base, fetched
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			gather(ctx, w, d)
			judge(w, d)
		}()
	}
	wg.Wait()
	return wts
}

// parseList reads `git worktree list --porcelain`.
func parseList(out string) []Worktree {
	var list []Worktree
	var cur *Worktree
	for _, line := range strings.Split(out, "\n") {
		field, value, _ := strings.Cut(line, " ")
		switch field {
		case "worktree":
			list = append(list, Worktree{Path: value})
			cur = &list[len(list)-1]
		case "HEAD":
			if cur != nil {
				cur.Head = value
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(value, "refs/heads/")
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
			}
		}
	}
	return list
}

// defaultBase is the ref merged work ends up in: origin's HEAD, else
// origin/main or origin/master, else a local main or master.
func defaultBase(ctx context.Context, repo string, d Deps) string {
	if ref, err := d.Git(ctx, repo, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil && ref != "" {
		return strings.TrimPrefix(ref, "refs/remotes/")
	}
	for _, ref := range []string{"origin/main", "origin/master", "main", "master"} {
		if _, err := d.Git(ctx, repo, "rev-parse", "--verify", "--quiet", ref); err == nil {
			return ref
		}
	}
	return ""
}

func gather(ctx context.Context, w *Worktree, d Deps) {
	if fi, err := os.Stat(w.Path); err != nil || !fi.IsDir() {
		w.Prunable = true
	}
	if w.Prunable {
		return
	}
	if d.InUse != nil {
		w.InUse = d.InUse(w.Path)
	}
	if out, err := d.Git(ctx, w.Path, "status", "--porcelain", "--ignored", "--untracked-files=normal"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if len(line) < 4 {
				continue
			}
			code, p := line[:2], strings.TrimSuffix(line[3:], "/")
			switch code {
			case "??":
				w.Untracked++
			case "!!":
				if precious(p) && !copyOfMain(w, p) {
					w.Ignored = append(w.Ignored, p)
				}
			default:
				w.Changed++
			}
		}
	}
	if out, err := d.Git(ctx, w.Path, "log", "-1", "--format=%ct%x09%s"); err == nil {
		ts, subject, _ := strings.Cut(out, "\t")
		if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
			w.LastCommit = time.Unix(sec, 0)
		}
		w.Subject = subject
	}
	// Commits only this branch holds: on no remote and not in the base branch
	// (which keeps them even if they were never pushed).
	unpushed := []string{"rev-list", "--count", "HEAD", "--not", "--remotes"}
	if w.Base != "" {
		unpushed = append(unpushed, w.Base)
	}
	if out, err := d.Git(ctx, w.Path, unpushed...); err == nil {
		w.Unpushed, _ = strconv.Atoi(out)
	}
	if w.Base != "" {
		if out, err := d.Git(ctx, w.Path, "rev-list", "--left-right", "--count", w.Base+"...HEAD"); err == nil {
			if f := strings.Fields(out); len(f) == 2 {
				w.Behind, _ = strconv.Atoi(f[0])
				w.Ahead, _ = strconv.Atoi(f[1])
			}
		}
		w.Unique = w.Ahead
		// git cherry compares patches, so a rebased or cherry-picked commit
		// counts as merged; a squash merge still reads as unique — that is what
		// the PR lookup is for. Skip it on huge histories.
		if w.Ahead > 0 && w.Ahead <= 500 {
			if out, err := d.Git(ctx, w.Path, "cherry", w.Base, "HEAD"); err == nil {
				w.Unique = strings.Count("\n"+out, "\n+")
			}
		}
	}
	if w.Branch != "" && d.PR != nil {
		pr, err := d.PR(ctx, w.Repo, w.Branch)
		switch {
		case err != nil:
			w.PRError = err.Error()
		case pr != nil:
			w.PR = pr
			w.HeadInPR = pr.HeadOID != "" && (pr.HeadOID == w.Head ||
				gitOK(ctx, d, w.Path, "merge-base", "--is-ancestor", "HEAD", pr.HeadOID))
		}
	}
}

func gitOK(ctx context.Context, d Deps, dir string, args ...string) bool {
	_, err := d.Git(ctx, dir, args...)
	return err == nil
}

// buildOutput are ignored names that regenerate; anything else ignored is
// kept for you to look at (.env files, local data, uploads…).
var buildOutput = map[string]bool{
	"node_modules": true, ".next": true, ".turbo": true, "dist": true, "build": true, "out": true,
	".cache": true, "coverage": true, "target": true, ".venv": true, "venv": true, "__pycache__": true,
	".pytest_cache": true, ".mypy_cache": true, ".ruff_cache": true, ".expo": true, ".gradle": true,
	"Pods": true, "DerivedData": true, ".DS_Store": true, ".eslintcache": true, ".vercel": true,
	".svelte-kit": true, ".nuxt": true, ".output": true, ".parcel-cache": true, "storybook-static": true,
	".task": true, ".idea": true, ".vscode": true, ".claude": true, "tmp": true, ".tmp": true, "logs": true,
	".husky": true, "next-env.d.ts": true, "test-results": true, "playwright-report": true, "blob-report": true,
	".pnpm-store": true, ".godot": true, ".wrangler": true, ".terraform": true,
}

func precious(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if buildOutput[part] {
			return false
		}
	}
	switch path.Ext(p) {
	case ".log", ".tsbuildinfo", ".pyc":
		return false
	}
	return true
}

// copyOfMain reports whether an ignored file holds nothing of its own: a
// symlink, or a byte-for-byte copy of the main checkout's file at the same
// place (the .env files agents copy into their worktrees).
func copyOfMain(w *Worktree, rel string) bool {
	full := filepath.Join(w.Path, rel)
	fi, err := os.Lstat(full)
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return true
	}
	if !fi.Mode().IsRegular() || fi.Size() > 1<<20 {
		return false
	}
	other, err := os.Lstat(filepath.Join(w.Repo, rel))
	if err != nil || !other.Mode().IsRegular() || other.Size() != fi.Size() {
		return false
	}
	a, errA := os.ReadFile(full)
	b, errB := os.ReadFile(filepath.Join(w.Repo, rel))
	return errA == nil && errB == nil && string(a) == string(b)
}

// judge turns the evidence into a verdict and the reasons for it.
func judge(w *Worktree, d Deps) {
	if w.Prunable {
		w.Verdict, w.Reasons = Stale, []string{"its folder is gone; git still lists it"}
		return
	}
	var hard, soft []string
	if w.InUse != "" {
		hard = append(hard, "in use by "+w.InUse)
	}
	if w.Locked {
		hard = append(hard, "locked in git")
	}
	if w.Changed > 0 {
		hard = append(hard, plural(w.Changed, "changed file"))
	}
	if w.Untracked > 0 {
		hard = append(hard, plural(w.Untracked, "untracked file"))
	}
	if d.Home != "" && !strings.HasPrefix(w.Path, d.Home+"/") {
		hard = append(hard, "outside your home folder")
	}

	merged, why := w.merged()
	if len(w.Ignored) > 0 {
		soft = append(soft, "ignored files worth a look: "+strings.Join(first(w.Ignored, 3), ", "))
	}
	mergedWithHead := w.PR != nil && w.PR.State == "MERGED" && w.HeadInPR
	if w.Unpushed > 0 && !mergedWithHead {
		soft = append(soft, plural(w.Unpushed, "commit")+" on no remote")
	}
	if w.PR != nil && w.PR.State == "OPEN" {
		soft = append(soft, fmt.Sprintf("PR #%d is open", w.PR.Number))
	}
	if w.PR != nil && w.PR.State == "CLOSED" && !merged {
		soft = append(soft, fmt.Sprintf("PR #%d was closed without merging", w.PR.Number))
	}
	if w.PR != nil && w.PR.State == "MERGED" && !w.HeadInPR && w.Unique > 0 {
		soft = append(soft, fmt.Sprintf("commits after merged PR #%d", w.PR.Number))
	}
	idle := d.Now.Sub(w.LastCommit)

	switch {
	case len(hard) > 0:
		w.Verdict, w.Reasons = Keep, append(hard, soft...)
	case merged && len(soft) == 0:
		w.Verdict, w.Reasons = Merged, []string{why}
	case !merged && len(soft) == 0 && w.Unpushed == 0 && idle >= d.StaleAfter:
		w.Verdict, w.Reasons = Old, []string{fmt.Sprintf("untouched for %d days, every commit pushed", int(idle.Hours()/24))}
	default:
		w.Verdict, w.Reasons = Review, soft
		if !merged && w.Base != "" {
			w.Reasons = append(w.Reasons, fmt.Sprintf("%s not in %s", plural(w.Unique, "commit"), w.Base))
		}
		if len(w.Reasons) == 0 {
			w.Reasons = []string{"nothing proves its work is merged yet"}
		}
	}
}

// merged says whether the worktree's work is already in the default branch.
func (w Worktree) merged() (bool, string) {
	if w.PR != nil && w.PR.State == "MERGED" && (w.HeadInPR || w.Unique == 0) {
		return true, fmt.Sprintf("PR #%d merged", w.PR.Number)
	}
	if w.Base == "" {
		return false, ""
	}
	if w.Ahead == 0 {
		return true, "nothing beyond " + w.Base
	}
	if w.Unique == 0 {
		return true, "every commit's change is already in " + w.Base
	}
	return false, ""
}

// DeletableBranch reports whether its branch may go too: only for merged work,
// never the default branch.
func (w Worktree) DeletableBranch() bool {
	return w.Verdict == Merged && w.Branch != "" && w.Branch != w.BaseBranch()
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return append(append([]string(nil), xs[:n]...), fmt.Sprintf("+%d more", len(xs)-n))
	}
	return xs
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	if strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") || strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") {
		return fmt.Sprintf("%d %ses", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

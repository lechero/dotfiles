package clean

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"manage-disk/internal/scan"
)

// glob expands pattern relative to home, treating home itself literally.
// Only for folders we own: filepath.Glob opens every folder a "*" matches, and
// "Application Support/*/x" opened AddressBook and raised a Contacts dialog.
// Across Library use entries / entriesHolding instead.
func glob(home, pattern string) []string {
	matches, _ := filepath.Glob(escapeGlob(home) + "/" + pattern)
	return matches
}

// entries lists the names in dir that match, skipping anything macOS guards.
// It reads dir itself and nothing inside its entries.
func entries(home, dir string, match func(name string) bool) []string {
	list, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range list {
		p := dir + "/" + e.Name()
		if match(e.Name()) && !guarded(home, p) && scan.PrivacyReason(strings.TrimPrefix(p, home+"/"), false) == "" {
			out = append(out, p)
		}
	}
	return out
}

// entriesHolding returns dir/<entry>/child for every entry of dir that has one,
// checking with lstat rather than opening each entry.
func entriesHolding(home, dir, child string) []string {
	var out []string
	for _, p := range entries(home, dir, func(string) bool { return true }) {
		if isDir(p + "/" + child) {
			out = append(out, p+"/"+child)
		}
	}
	return out
}

func escapeGlob(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`*?[\`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// guarded reports whether path sits where macOS asks before letting a program
// in (Documents, Downloads, other apps' containers…). Discovery never goes there.
func guarded(home, path string) bool {
	rel, ok := strings.CutPrefix(path, home+"/")
	return ok && scan.InsidePrompting(rel)
}

// bundleID reads an app's CFBundleIdentifier, giving up after a few seconds.
func bundleID(ctx context.Context, env *Env, app string) string {
	if guarded(env.Home, app) {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	id, err := env.Output(ctx, "plutil", "-extract", "CFBundleIdentifier", "raw", "-o", "-", app+"/Contents/Info.plist")
	if err != nil {
		return ""
	}
	return strings.ToLower(id)
}

// isDir reports whether path is a real directory (not a symlink to one).
func isDir(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.IsDir()
}

// findDirs walks root and returns every directory for which match is true,
// without descending into matches or into directories named in prune.
func findDirs(ctx context.Context, root string, maxDepth int, prune map[string]bool, match func(path, name string) bool) []string {
	var out []string
	base := strings.Count(root, "/")
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil || !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		name := d.Name()
		if match(path, name) {
			out = append(out, path)
			return filepath.SkipDir
		}
		if prune[name] || strings.Count(path, "/")-base >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	return out
}

// tilde shortens a path under home to "~/…" for display.
func tilde(home, path string) string {
	if rel, ok := strings.CutPrefix(path, home+"/"); ok {
		return "~/" + rel
	}
	return path
}

// sizeItems fills in item sizes, a few at a time.
func sizeItems(ctx context.Context, env *Env, items []Item) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			var total int64
			for _, p := range items[i].Paths {
				total += env.Size(ctx, p)
			}
			items[i].Size = total
		}()
	}
	wg.Wait()
}

// dropEmpty removes items too small to be worth a line (under 1 MiB).
func dropEmpty(items []Item) []Item {
	out := items[:0]
	for _, it := range items {
		if it.Size >= 1<<20 {
			out = append(out, it)
		}
	}
	return out
}

// parseVersion reads "v22.16.0" or "mac_arm-150.0.7871.24" into numbers.
func parseVersion(s string) ([]int, bool) {
	if i := strings.LastIndexByte(s, '-'); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return nil, false
	}
	var out []int
	for _, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func compareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// installedBundleIDs reads CFBundleIdentifier from every app under the app
// folders (ChatGPT.app is com.openai.codex, so names alone would mislead).
func installedBundleIDs(ctx context.Context, env *Env) map[string]bool {
	var apps []string
	for _, dir := range env.AppDirs {
		base := strings.Count(dir, "/")
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".app") {
				apps = append(apps, path)
				return filepath.SkipDir
			}
			if strings.Count(path, "/")-base >= 4 {
				return filepath.SkipDir
			}
			return nil
		})
	}
	ids := map[string]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, app := range apps {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if id := bundleID(ctx, env, app); id != "" {
				mu.Lock()
				ids[id] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return ids
}

// gitRepos finds git checkouts (folders with a .git directory) under root.
func gitRepos(ctx context.Context, root string, maxDepth int) []string {
	prune := map[string]bool{"node_modules": true, ".git": true, ".next": true, "target": true}
	return findDirs(ctx, root, maxDepth, prune, func(path, _ string) bool {
		return isDir(path + "/.git")
	})
}

// linkedWorktrees lists a repo's extra worktrees that still exist on disk.
func linkedWorktrees(ctx context.Context, env *Env, repo string) []string {
	out, err := env.Output(ctx, "git", "-C", repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}
	var wts []string
	for _, line := range strings.Split(out, "\n") {
		wt, ok := strings.CutPrefix(line, "worktree ")
		if !ok || wt == repo || !isDir(wt) {
			continue
		}
		wts = append(wts, wt)
	}
	return wts
}

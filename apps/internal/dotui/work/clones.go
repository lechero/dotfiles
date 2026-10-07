package work

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Clones maps remote repositories, as "host/owner/repo", to a local clone.
type Clones map[string]string

// skipDirs are never searched for clones.
var skipDirs = map[string]bool{"node_modules": true, "vendor": true, "target": true, "dist": true, "build": true}

// FindClones looks for git checkouts up to depth folders below root, and
// reads their remotes from .git/config. Linked worktrees are left out: a
// checkout goes in the main clone. When two clones share a remote, the
// first in path order wins.
func FindClones(root string, depth int) Clones {
	c := Clones{}
	var walk func(dir string, left int)
	walk = func(dir string, left int) {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			for _, url := range remoteURLs(filepath.Join(dir, ".git", "config")) {
				if key, ok := RepoKey(url); ok {
					if _, seen := c[key]; !seen {
						c[key] = dir
					}
				}
			}
			return
		}
		if left == 0 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !skipDirs[e.Name()] {
				walk(filepath.Join(dir, e.Name()), left-1)
			}
		}
	}
	walk(root, depth)
	return c
}

// Find returns the clone of path, like "lechero/dotfiles", on host.
func (c Clones) Find(host, path string) (string, bool) {
	dir, ok := c[strings.ToLower(host+"/"+path)]
	return dir, ok
}

// remoteURLs reads every remote's url from a git config file.
func remoteURLs(config string) []string {
	f, err := os.Open(config)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }() // read-only
	var urls []string
	inRemote := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inRemote = strings.HasPrefix(line, "[remote ")
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok && inRemote && strings.TrimSpace(key) == "url" {
			urls = append(urls, strings.TrimSpace(value))
		}
	}
	return urls
}

// RepoKey turns a git remote URL into "host/owner/repo", lower case.
// It reads https://host/owner/repo(.git), ssh://git@host[:port]/owner/repo
// and git@host:owner/repo forms.
func RepoKey(url string) (string, bool) {
	var host, path string
	switch {
	case strings.Contains(url, "://"):
		_, rest, _ := strings.Cut(url, "://")
		hostPart, p, ok := strings.Cut(rest, "/")
		if !ok {
			return "", false
		}
		if _, h, ok := strings.Cut(hostPart, "@"); ok {
			hostPart = h
		}
		host, _, _ = strings.Cut(hostPart, ":")
		path = p
	case strings.Contains(url, ":"):
		hostPart, p, _ := strings.Cut(url, ":")
		if _, h, ok := strings.Cut(hostPart, "@"); ok {
			hostPart = h
		}
		host, path = hostPart, p
	default:
		return "", false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if host == "" || !strings.Contains(path, "/") {
		return "", false
	}
	return strings.ToLower(host + "/" + path), true
}

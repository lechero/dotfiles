// Package clean knows which caches are safe to delete, checks that nothing
// running still needs them, and deletes them.
package clean

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Guard refuses any deletion that is not a specific folder deep inside $HOME.
// Tasks only delete paths they discovered themselves, so this is the second
// line of defence: a bug that turned "~/.cache/act" into "~/.cache" stops here.
type Guard struct{ Home string }

// protectedRel are folders that hold caches but are never caches themselves.
var protectedRel = toSet([]string{
	"Library", "Library/Caches", "Library/Application Support", "Library/Containers",
	"Library/Group Containers", "Library/Developer", "Library/Developer/Xcode",
	"Library/pnpm", "Library/pnpm/store", "Library/Preferences", "Library/Android",
	"projects", ".cache", ".local", ".local/share", ".local/share/nvm", ".local/state",
	".config", ".npm", ".npm/_npx", ".nvm", ".nvm/versions", ".nvm/versions/node",
	".claude", ".codex", ".codex/worktrees", ".android", ".android/avd", ".gradle",
	".cargo", ".rustup", "go", "go/pkg", "Applications", ".docker", ".bun",
})

// neverInside are places no cleanup has any business in.
var neverInside = []string{
	".ssh", ".gnupg", "Library/Keychains", "Library/Mobile Documents", "Library/CloudStorage",
	"Documents", "Desktop", "Downloads", "Pictures", "Movies", "Music", ".Trash",
}

var errNoHome = errors.New("guard has no home folder")

// Check returns nil only when path may be deleted.
func (g Guard) Check(path string) error {
	if g.Home == "" || !filepath.IsAbs(g.Home) || filepath.Clean(g.Home) != g.Home {
		return errNoHome
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("refusing %q: not a clean absolute path", path)
	}
	rel, ok := strings.CutPrefix(path, g.Home+"/")
	if !ok {
		return fmt.Errorf("refusing %q: outside %s", path, g.Home)
	}
	if !strings.Contains(rel, "/") {
		return fmt.Errorf("refusing %q: a top-level folder of your home", path)
	}
	if protectedRel[rel] {
		return fmt.Errorf("refusing %q: it holds caches but is not one", path)
	}
	for _, p := range neverInside {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return fmt.Errorf("refusing %q: inside ~/%s", path, p)
		}
	}
	return nil
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

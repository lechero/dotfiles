package work

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestFollowUpsAreMostUrgentFirstAndListedOnce(t *testing.T) {
	now := time.Now()
	d := Dashboard{Sections: []Section{
		{Items: []Item{
			{ID: "a#1", Next: "checks failing", Urgency: 80, Updated: now.Add(-time.Hour)},
			{ID: "a#2"}, // nothing to do
			{ID: "a#3", Next: "ready to merge", Urgency: 70, Updated: now},
		}},
		{Items: []Item{
			{ID: "b#4", Next: "review requested", Urgency: 90, Updated: now.Add(-48 * time.Hour)},
			{ID: "a#3", Next: "mentioned", Urgency: 60, Updated: now}, // less urgent than in its other section
		}},
	}}
	var got []string
	for _, it := range d.FollowUps() {
		got = append(got, it.ID+" "+it.Next)
	}
	want := []string{"b#4 review requested", "a#1 checks failing", "a#3 ready to merge"}
	if !slices.Equal(got, want) {
		t.Errorf("FollowUps = %q, want %q", got, want)
	}
}

func TestFollowUpsWithoutTimesKeepTheServicesOrder(t *testing.T) {
	d := Dashboard{Sections: []Section{{Items: []Item{
		{ID: "WEB-9", Next: "to do", Urgency: 35},
		{ID: "WEB-10", Next: "to do", Urgency: 35},
		{ID: "WEB-2", Next: "to do", Urgency: 35},
	}}}}
	var got []string
	for _, it := range d.FollowUps() {
		got = append(got, it.ID)
	}
	if want := []string{"WEB-9", "WEB-10", "WEB-2"}; !slices.Equal(got, want) {
		t.Errorf("FollowUps = %q, want the order they came in: %q", got, want)
	}
}

func TestRepoKey(t *testing.T) {
	for url, want := range map[string]string{
		"git@github.com:lechero/dotfiles.git":                "github.com/lechero/dotfiles",
		"https://github.com/Lechero/Dotfiles":                "github.com/lechero/dotfiles",
		"ssh://git@gitlab.com:2222/group/sub/project.git":    "gitlab.com/group/sub/project",
		"https://oauth2:secret@gitlab.example.com/a/b.git":   "gitlab.example.com/a/b",
		"ssh://git@ssh.github.com:443/lechero/dotfiles.git/": "ssh.github.com/lechero/dotfiles",
		"git@gitlab.com:vodafoneziggodi/public/vozitui.git":  "gitlab.com/vodafoneziggodi/public/vozitui",
	} {
		got, ok := RepoKey(url)
		if !ok || got != want {
			t.Errorf("RepoKey(%q) = %q, %v; want %q", url, got, ok, want)
		}
	}
	for _, url := range []string{"", "/some/local/path", "https://github.com/"} {
		if got, ok := RepoKey(url); ok {
			t.Errorf("RepoKey(%q) = %q, want no key", url, got)
		}
	}
}

func TestFindClonesReadsRemotesAndSkipsWorktrees(t *testing.T) {
	root := t.TempDir()
	clone := func(rel, config string) {
		t.Helper()
		dir := filepath.Join(root, rel, ".git")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	clone("lechero/dotfiles", "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = git@github.com:lechero/dotfiles.git\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n")
	clone("work/vozitui", "[remote \"upstream\"]\n\turl = https://gitlab.com/vodafoneziggodi/public/vozitui.git\n[branch \"main\"]\n\tremote = upstream\n")
	clone("deep/a/b/c/too-deep", "[remote \"origin\"]\n\turl = git@github.com:x/too-deep.git\n")
	clone("lechero/node_modules/pkg", "[remote \"origin\"]\n\turl = git@github.com:x/pkg.git\n")
	// A linked worktree has a .git file, not a folder.
	wt := filepath.Join(root, "lechero", "dotfiles-wt")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := FindClones(root, 3)
	if dir, ok := c.Find("github.com", "Lechero/Dotfiles"); !ok || dir != filepath.Join(root, "lechero/dotfiles") {
		t.Errorf("dotfiles clone = %q, %v", dir, ok)
	}
	if dir, ok := c.Find("gitlab.com", "vodafoneziggodi/public/vozitui"); !ok || dir != filepath.Join(root, "work/vozitui") {
		t.Errorf("vozitui clone = %q, %v", dir, ok)
	}
	if len(c) != 2 {
		t.Errorf("found %v; past the depth limit and in node_modules shouldn't count", c)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	c := Cache{Dir: filepath.Join(t.TempDir(), "dotui")}
	if _, ok := c.Load("GitHub"); ok {
		t.Fatal("an empty cache loaded something")
	}
	when := time.Date(2026, 10, 7, 9, 30, 0, 0, time.UTC)
	c.Save("GitHub", Dashboard{Account: "me", Fetched: when, Sections: []Section{{Key: "PRs", Items: []Item{{ID: "a#1", Badges: []Badge{{"draft", Dim}}}}}}})
	d, ok := c.Load("GitHub")
	if !ok || d.Account != "me" || !d.Fetched.Equal(when) || d.Sections[0].Items[0].Badges[0].Text != "draft" {
		t.Errorf("loaded %+v, %v", d, ok)
	}
	info, err := os.Stat(c.path("GitHub"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("cache file mode %v, %v; only its owner should read it", info.Mode().Perm(), err)
	}
	(Cache{}).Save("GitHub", d) // no folder: keeps nothing, and doesn't fail
	if _, ok := (Cache{}).Load("GitHub"); ok {
		t.Error("a cache without a folder loaded something")
	}
}

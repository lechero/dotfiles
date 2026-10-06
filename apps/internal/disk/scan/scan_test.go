package scan

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

const mib = 1 << 20

// writeFile writes n non-zero bytes so APFS allocates real blocks.
func writeFile(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0xA5}, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	writeFile(t, home+"/projects/app/node_modules/big.bin", 2*mib)
	writeFile(t, home+"/projects/app/small.txt", 1024)
	writeFile(t, home+"/Desktop/secret.bin", 2*mib)
	writeFile(t, home+"/Library/Containers/com.other.app/Data/x.bin", 2*mib)
	writeFile(t, home+"/Library/Containers/com.docker.docker/Data/Docker.raw", 3*mib)
	// Same folder, so the folder's total is right whichever link is read first.
	if err := os.Link(home+"/projects/app/node_modules/big.bin", home+"/projects/app/node_modules/link.bin"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(home+"/projects/app/node_modules/big.bin", home+"/projects/app/sym"); err != nil {
		t.Fatal(err)
	}
	return home
}

func scanHome(t *testing.T, home string, includePrivate bool) *Result {
	t.Helper()
	opts := DefaultOptions(home)
	opts.MinNode = 512 << 10
	opts.IncludePrivate = includePrivate
	res, err := Scan(context.Background(), home, opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestScanSkipsPrivatePlacesAndCountsHardlinksOnce(t *testing.T) {
	home := fakeHome(t)
	res := scanHome(t, home, false)
	root := res.Root

	if d := root.Find(home + "/Desktop"); d == nil || d.Skipped != SkipPrivacy {
		t.Fatalf("Desktop should be kept as a skipped node, got %+v", d)
	}
	// All of Library/Containers is skipped, Docker's too: it prompted from a terminal.
	if c := root.Find(home + "/Library/Containers"); c == nil || c.Skipped != SkipPrivacy || len(c.Children) != 0 {
		t.Fatalf("Library/Containers should be skipped unread, got %+v", c)
	}
	if !slices.Equal(res.Private, []string{"Desktop", "Library/Containers"}) {
		t.Errorf("Private = %v", res.Private)
	}
	if _, private := lookup(root, "Library/Containers/com.docker.docker"); !private {
		t.Error("a place under a skipped folder should read as private")
	}

	app := root.Find(home + "/projects/app")
	if app == nil {
		t.Fatal("projects/app missing")
	}
	// big.bin and its hardlink share blocks; the symlink is not followed.
	if app.Size < 2*mib || app.Size > 2*mib+256<<10 {
		t.Errorf("app.Size = %d, want ~2 MiB (hardlink counted once)", app.Size)
	}
	if app.SmallN < 2 {
		t.Errorf("small.txt and sym should be folded into Small, SmallN = %d", app.SmallN)
	}
	if root.Size >= 8*mib {
		t.Errorf("private places leaked into the total: %d", root.Size)
	}

	all := scanHome(t, home, true).Root
	if all.Size < root.Size+7*mib {
		t.Errorf("IncludePrivate should add Desktop and both containers: %d vs %d", all.Size, root.Size)
	}
	if raw := all.Find(home + "/Library/Containers/com.docker.docker/Data/Docker.raw"); raw == nil || raw.Size < 3*mib {
		t.Errorf("with IncludePrivate Docker.raw should be measured, got %+v", raw)
	}
}

func TestAFolderThatNeverAnswersIsSkipped(t *testing.T) {
	home := fakeHome(t)
	stuck := home + "/projects"
	release := make(chan struct{})
	defer close(release)
	opts := DefaultOptions(home)
	opts.DirTimeout = 200 * time.Millisecond
	opts.readNames = func(path string) ([]string, error) {
		if path == stuck {
			<-release // a consent dialog nobody answers
		}
		return readNames(path)
	}
	done := make(chan *Result, 1)
	go func() {
		res, _ := Scan(context.Background(), home, opts, nil)
		done <- res
	}()
	select {
	case res := <-done:
		if p := res.Root.Find(stuck); p == nil || p.Skipped != SkipNoResponse {
			t.Errorf("the stuck folder should be marked, got %+v", p)
		}
		if !slices.Equal(res.NoResponse, []string{stuck}) {
			t.Errorf("NoResponse = %v", res.NoResponse)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the scan hung on a folder that never answered")
	}
}

func TestRemoveAndReplaceKeepTotalsRight(t *testing.T) {
	home := fakeHome(t)
	root := scanHome(t, home, true).Root // true: Docker.raw sits in Library/Containers
	before := root.Size
	nm := home + "/projects/app/node_modules"

	freed, ok := root.Remove(nm)
	if !ok || freed < 2*mib {
		t.Fatalf("Remove = %d, %v", freed, ok)
	}
	if root.Find(nm) != nil {
		t.Error("node still in tree")
	}
	if root.Size != before-freed {
		t.Errorf("root.Size = %d, want %d", root.Size, before-freed)
	}

	raw := home + "/Library/Containers/com.docker.docker/Data/Docker.raw"
	old := root.Find(raw).Size
	if !root.Replace(raw, &Node{Size: old + 5*mib, Files: 1}) {
		t.Fatal("Replace failed")
	}
	if root.Size != before-freed+5*mib {
		t.Errorf("after Replace root.Size = %d, want %d", root.Size, before-freed+5*mib)
	}
	if got := root.Find(raw); got == nil || got.Name != "Docker.raw" {
		t.Errorf("replacement should keep the name, got %+v", got)
	}
}

func TestSizeOfMatchesScan(t *testing.T) {
	home := fakeHome(t)
	root := scanHome(t, home, false).Root
	nm := home + "/projects/app/node_modules"
	if got, want := SizeOf(context.Background(), nm), root.Find(nm).Size; got != want {
		t.Errorf("SizeOf = %d, scan says %d", got, want)
	}
}

// tree builds a synthetic tree; sizes are in "units" for readable tests.
func tree(name string, size int64, kids ...*Node) *Node {
	n := &Node{Name: name, IsDir: len(kids) > 0, Size: size, Children: kids}
	for _, k := range kids {
		k.Parent = n
	}
	sortNodes(n.Children)
	return n
}

func TestCategoriesAreDisjointAndAddUp(t *testing.T) {
	root := tree("/home", 46,
		tree("projects", 10),
		tree(".cache", 5, tree("lm-studio", 2)),
		tree(".codex", 4, tree("worktrees", 1)),
		tree("Library", 20,
			tree("Caches", 3),
			tree("Application Support", 8, tree("Claude", 6)),
			tree("Containers", 9, tree("com.docker.docker", 9)),
		),
		tree("other", 7),
	)
	got := map[string]int64{}
	var sum int64
	for _, c := range Categories(root) {
		got[c.Name] = c.Size
		sum += c.Size
	}
	want := map[string]int64{
		"Projects & worktrees": 11, // projects + .codex/worktrees
		"Docker VM":            9,
		"App & tool caches":    6,  // Library/Caches + .cache without lm-studio
		"AI models & agents":   11, // lm-studio + .codex without worktrees + Claude
		"Other app data":       2,  // Application Support without Claude
		EverythingElse:         7,
	}
	for name, size := range want {
		if got[name] != size {
			t.Errorf("%s = %d, want %d", name, got[name], size)
		}
	}
	if sum != root.Size {
		t.Errorf("categories add up to %d, want %d", sum, root.Size)
	}
}

func TestHotspotsFollowDominantChildren(t *testing.T) {
	root := tree("/home", 100,
		tree("Library", 30,
			tree("Containers", 22,
				tree("com.docker.docker", 22, tree("Data", 22, &Node{Name: "Docker.raw", Size: 22, Files: 1}))),
		),
		tree("projects", 40,
			tree("hs", 33,
				tree(".claude", 28, tree("worktrees", 28, tree("a", 8), tree("b", 4), tree("c", 3))),
				tree("node_modules", 3)),
		),
		tree(".codex", 10, tree("sessions", 4), tree("worktrees", 3)),
	)
	var names []string
	for _, h := range Hotspots(root, 4, 3) {
		names = append(names, h.Rel())
	}
	want := []string{
		"projects/hs/.claude/worktrees",
		"Library/Containers/com.docker.docker/Data/Docker.raw",
		".codex",
		"projects/hs/node_modules",
	}
	if !slices.Equal(names, want) {
		t.Errorf("Hotspots = %v\nwant       %v", names, want)
	}
}

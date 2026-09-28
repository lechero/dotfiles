package clean

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func testEnv(t *testing.T, procs Procs) *Env {
	t.Helper()
	home := t.TempDir()
	env := NewEnv(home)
	env.AppDirs = []string{home + "/Applications"}
	env.SnapshotFn = func(context.Context) (Procs, error) { return procs, nil }
	env.LookPath = func(string) (string, error) { return "", errors.New("not in tests") }
	env.Now = func() time.Time { return fixedNow }
	return env
}

func put(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0x5A}, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

func discover(t *testing.T, task *Task, env *Env) []Item {
	t.Helper()
	items, err := task.Discover(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func labels(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Label)
	}
	slices.Sort(out)
	return out
}

func byLabel(items []Item, label string) *Item {
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	return nil
}

func TestGuard(t *testing.T) {
	g := Guard{Home: "/Users/me"}
	allowed := []string{
		"/Users/me/.cache/act",
		"/Users/me/projects/app/apps/web/.next",
		"/Users/me/Library/Caches/Google/Chrome",
		"/Users/me/.android/avd/Pixel.avd",
		"/Users/me/.local/share/nvm/v18.17.1",
	}
	refused := []string{
		"/Users/me", "/Users/me/.cache", "/Users/me/projects", "/Users/me/Library/Caches",
		"/Users/me/.local/share/nvm", "/Users/me/Documents/x/.next", "/Users/me/.ssh/keys",
		"/Users/other/.cache/act", "relative/path", "/Users/me/.cache/../.ssh",
		"/Users/me/Library", "/Users/me/.npm/_npx",
	}
	for _, p := range allowed {
		if err := g.Check(p); err != nil {
			t.Errorf("Check(%q) refused: %v", p, err)
		}
	}
	for _, p := range refused {
		if g.Check(p) == nil {
			t.Errorf("Check(%q) allowed, want refused", p)
		}
	}
	if (Guard{}).Check("/Users/me/.cache/act") == nil {
		t.Error("a guard without home must refuse everything")
	}
}

func TestParseProcs(t *testing.T) {
	exes := []byte("  101 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome\n  202 node\n")
	cmds := []byte("  101 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome --type=x\n  202 node /Users/me/.npm/_npx/abc123/node_modules/.bin/mcp-remote https://x\n")
	cwds := []byte("p101\nfcwd\nn/\np202\nfcwd\nn/Users/me/projects/app\n")
	ps := parseProcs(exes, cmds, cwds)
	if len(ps) != 2 || ps[1].Cwd != "/Users/me/projects/app" || !strings.HasPrefix(ps[1].Command, "node ") {
		t.Fatalf("parseProcs = %+v", ps)
	}
	if ps.App("MacOS/Google Chrome") == nil || ps.App("MacOS/Arc") != nil {
		t.Error("App matching is off")
	}
	if ps.Using("/Users/me/.npm/_npx/abc123") == nil || ps.Using("/Users/me/.npm/_npx/zzz") != nil {
		t.Error("Using should match command-line paths exactly")
	}
	if ps.Using("/Users/me/projects") == nil {
		t.Error("Using should match a working directory inside")
	}

	launcher := Proc{PID: 3, Exe: "/Applications/Claude.app/Contents/Helpers/disclaimer",
		Command: "/Applications/Claude.app/Contents/Helpers/disclaimer --pgroup -- /Users/me/Library/Application Support/Claude/claude-code/2.1/claude.app/Contents/MacOS/claude --output-format stream-json"}
	if got := launcher.Name(); got != "claude" {
		t.Errorf("Name() of a disclaimer launch = %q, want claude", got)
	}
}

func TestBuildCachesSkipNodeModulesAndRunningServers(t *testing.T) {
	var env *Env
	env = testEnv(t, nil)
	h := env.Home
	put(t, h+"/projects/a/apps/web/.next/cache.bin", 1<<20)
	put(t, h+"/projects/a/node_modules/pkg/.next/x", 1<<20) // inside node_modules: never
	put(t, h+"/projects/b/.turbo/cache/x", 1<<20)
	env.SnapshotFn = func(context.Context) (Procs, error) {
		return Procs{{PID: 7, Exe: "node", Command: "node next dev --port 3200", Cwd: h + "/projects/a/apps/web"}}, nil
	}

	items := discover(t, buildCachesTask(), env)
	if got := labels(items); !slices.Equal(got, []string{"a/apps/web/.next", "b/.turbo"}) {
		t.Fatalf("labels = %v", got)
	}
	if it := byLabel(items, "a/apps/web/.next"); it.Blocked == "" {
		t.Error("the .next a dev server uses must be blocked")
	}
	if it := byLabel(items, "b/.turbo"); it.Blocked != "" || !it.Selected || it.Size < 1<<20 {
		t.Errorf("b/.turbo should be ready and sized: %+v", it)
	}
}

func TestPuppeteerKeepsNewest(t *testing.T) {
	env := testEnv(t, nil)
	put(t, env.Home+"/.cache/puppeteer/chrome/mac_arm-117.0.5938.92/x", 1<<20)
	put(t, env.Home+"/.cache/puppeteer/chrome/mac_arm-150.0.7871.24/x", 1<<20)
	put(t, env.Home+"/.cache/puppeteer/chrome/mac_arm-145.0.7632.77/x", 1<<20)
	items := discover(t, puppeteerTask(), env)
	if got := labels(items); !slices.Equal(got, []string{"chrome mac_arm-117.0.5938.92", "chrome mac_arm-145.0.7632.77"}) {
		t.Errorf("labels = %v", got)
	}
}

func TestBrowserAndNpxInUseAreBlocked(t *testing.T) {
	var env *Env
	env = testEnv(t, nil)
	h := env.Home
	put(t, h+"/Library/Caches/Google/Chrome/x", 1<<20)
	put(t, h+"/Library/Caches/Firefox/x", 1<<20)
	put(t, h+"/.npm/_npx/aaa/package.json", 0)
	os.WriteFile(h+"/.npm/_npx/aaa/package.json", []byte(`{"dependencies":{"mcp-remote":"^1"}}`), 0o644)
	put(t, h+"/.npm/_npx/aaa/node_modules/mcp-remote/index.js", 1<<20)
	put(t, h+"/.npm/_npx/bbb/x", 1<<20)
	env.SnapshotFn = func(context.Context) (Procs, error) {
		return Procs{
			{PID: 1, Exe: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
			{PID: 2, Exe: "node", Command: "node " + h + "/.npm/_npx/aaa/node_modules/.bin/mcp-remote"},
		}, nil
	}

	browsers := discover(t, browserCachesTask(), env)
	if byLabel(browsers, "Chrome").Blocked == "" || byLabel(browsers, "Firefox").Blocked != "" {
		t.Errorf("browsers = %+v", browsers)
	}
	dev := discover(t, devCachesTask(), env)
	if it := byLabel(dev, "npx mcp-remote"); it == nil || it.Blocked == "" {
		t.Errorf("the npx folder mcp-remote runs from must be blocked: %+v", dev)
	}
	if it := byLabel(dev, "npx bbb"); it == nil || it.Blocked != "" {
		t.Errorf("an unused npx folder should be ready: %+v", dev)
	}
}

func TestNodeVersionPolicy(t *testing.T) {
	env := testEnv(t, nil)
	h := env.Home
	for _, v := range []string{"v18.17.1", "v20.10.0", "v20.18.0", "v22.9.0", "v22.16.0"} {
		put(t, h+"/.local/share/nvm/"+v+"/bin/node", 1<<20)
	}
	put(t, h+"/.local/share/nvm/v22.9.0/lib/node_modules/typescript/package.json", 10)
	put(t, h+"/.local/share/nvm/v22.9.0/lib/node_modules/npm/package.json", 10)
	put(t, h+"/.local/share/nvm/v20.10.0/lib/node_modules/pnpm/package.json", 10) // package managers don't count
	put(t, h+"/.config/fish/fish_variables", 0)
	os.WriteFile(h+"/.config/fish/fish_variables", []byte("# fish\nSETUVAR nvm_default_version:20\n"), 0o644)
	put(t, h+"/projects/old/.nvmrc", 0)
	os.WriteFile(h+"/projects/old/.nvmrc", []byte("v18.17.1\n"), 0o644)

	items := discover(t, nodeVersionsTask(), env)
	want := map[string]bool{
		"fish v18.17.1": true,  // old major
		"fish v20.10.0": true,  // superseded patch
		"fish v20.18.0": false, // newest v20 and fish's default
		"fish v22.9.0":  false, // has a global package
		"fish v22.16.0": false, // newest v22
	}
	for label, sel := range want {
		it := byLabel(items, label)
		if it == nil {
			t.Fatalf("%s missing from %v", label, labels(items))
		}
		if it.Selected != sel {
			t.Errorf("%s Selected = %v, want %v (%s)", label, it.Selected, sel, it.Note)
		}
	}
	if n := byLabel(items, "fish v18.17.1").Note; !strings.Contains(n, "pinned by 1 .nvmrc") {
		t.Errorf("v18 note = %q", n)
	}
	if n := byLabel(items, "fish v22.9.0").Note; !strings.Contains(n, "globals: typescript") {
		t.Errorf("v22.9 note = %q", n)
	}
}

func TestOrphanedCachesUseBundleIDs(t *testing.T) {
	env := testEnv(t, nil)
	h := env.Home
	put(t, h+"/Applications/Foo.app/Contents/Info.plist", 0)
	os.WriteFile(h+"/Applications/Foo.app/Contents/Info.plist", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.example.Foo</string></dict></plist>`), 0o644)
	big := 21 << 20
	put(t, h+"/Library/Caches/com.example.Foo/x", big)           // installed
	put(t, h+"/Library/Caches/com.example.foo.helper/x", big)    // installed app's helper
	put(t, h+"/Library/Caches/com.gone.app/x", big)              // orphan
	put(t, h+"/Library/Caches/com.gone.tiny/x", 1<<20)           // orphan, too small to bother
	put(t, h+"/Library/Caches/com.apple.Safari/x", big)          // Apple's own
	put(t, h+"/Library/Caches/Homebrew/x", big)                  // not a bundle id
	put(t, h+"/Library/Caches/copilot-desktop-gh-2.98.0/x", big) // a versioned download, not an id

	items := discover(t, orphanedCachesTask(), env)
	if got := labels(items); !slices.Equal(got, []string{"com.gone.app"}) {
		t.Errorf("orphans = %v", got)
	}
}

func TestUpdaterLeftoversNeverLookInsideGuardedFolders(t *testing.T) {
	env := testEnv(t, nil)
	h := env.Home
	support := h + "/Library/Application Support"
	put(t, support+"/Code/CachedExtensionVSIXs/ext.vsix", 1<<20)
	put(t, support+"/AddressBook/CachedExtensionVSIXs/x", 1<<20) // Contacts: macOS would ask
	put(t, support+"/MobileSync/CachedExtensionVSIXs/x", 1<<20)  // Full Disk Access only
	put(t, h+"/Library/Caches/com.example.app/org.sparkle-project.Sparkle/Installation/x", 1<<20)
	put(t, h+"/Library/Caches/com.example.app.ShipIt/update.zip", 1<<20)
	items := discover(t, updaterLeftoversTask(), env)
	want := []string{
		"~/Library/Application Support/Code/CachedExtensionVSIXs",
		"~/Library/Caches/com.example.app.ShipIt",
		"~/Library/Caches/com.example.app/org.sparkle-project.Sparkle",
	}
	if got := labels(items); !slices.Equal(got, want) {
		t.Errorf("leftovers = %v\nwant       %v", got, want)
	}
}

func TestRustTargetsNeedCargoTag(t *testing.T) {
	env := testEnv(t, nil)
	h := env.Home
	put(t, h+"/projects/tui/target/CACHEDIR.TAG", 100)
	put(t, h+"/projects/tui/target/debug/bin", 1<<20)
	put(t, h+"/projects/site/src/target/page.html", 1<<20) // a source folder named target
	items := discover(t, rustTargetsTask(), env)
	if got := labels(items); !slices.Equal(got, []string{"tui/target"}) {
		t.Errorf("targets = %v", got)
	}
}

func TestParseDockerSize(t *testing.T) {
	cases := map[string]int64{"16.4GB (51%)": 16_400_000_000, "512kB": 512_000, "0B": 0, "1.5MB": 1_500_000, "junk": 0}
	for in, want := range cases {
		if got := parseDockerSize(in); got != want {
			t.Errorf("parseDockerSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestResolveNodeSpec(t *testing.T) {
	vs := listNodeVersions("") // empty
	if resolveNodeSpec("20", vs) != "" {
		t.Error("nothing installed resolves to nothing")
	}
	vs = []nodeVersion{
		{dir: "a", name: "v20.10.0", num: []int{20, 10, 0}},
		{dir: "b", name: "v20.18.0", num: []int{20, 18, 0}},
		{dir: "c", name: "v22.1.0", num: []int{22, 1, 0}},
	}
	cases := map[string]string{"20": "b", "v20.10.0": "a", "20.10": "a", "node": "c", "lts/*": "", "": ""}
	for spec, want := range cases {
		if got := resolveNodeSpec(spec, vs); got != want {
			t.Errorf("resolveNodeSpec(%q) = %q, want %q", spec, got, want)
		}
	}
	eols := map[int]string{18: "2025-04-30", 20: "2026-04-30", 22: "2027-04-30", 23: "2025-06-01", 25: "2026-06-01"}
	for major, want := range eols {
		if got := nodeEOL(major).Format("2006-01-02"); got != want {
			t.Errorf("nodeEOL(%d) = %s, want %s", major, got, want)
		}
	}
	if got := unescapeFish(`v20\x2e18\x2e0`); got != "v20.18.0" {
		t.Errorf("unescapeFish = %q", got)
	}
}

func TestRunDryRunDeletesNothingAndRealRunDeletes(t *testing.T) {
	env := testEnv(t, nil)
	h := env.Home
	put(t, h+"/.cache/act/x", 1<<20)
	put(t, h+"/.cache/phpactor/x", 1<<20)
	put(t, h+"/projects/keep/file", 100)
	task := &Task{ID: "t", Title: "Test"}
	sels := []Selection{{Task: task, Items: []Item{
		{Paths: []string{h + "/.cache/act"}, Label: "act", Size: 1 << 20, Selected: true},
		{Paths: []string{h + "/.cache/phpactor"}, Label: "phpactor", Selected: true, Blocked: "busy"},
		{Paths: []string{h + "/projects"}, Label: "danger", Selected: true}, // the guard refuses this
	}}}

	var kinds []EventKind
	report := func(e Event) { kinds = append(kinds, e.Kind) }

	dry := Run(context.Background(), env, sels, true, report)
	if !isDir(h+"/.cache/act") || !isDir(h+"/projects") {
		t.Fatal("a dry run deleted something")
	}
	if len(dry.Cleaned) != 1 || dry.Skipped != 1 || dry.Failed != 1 {
		t.Errorf("dry summary = %+v", dry)
	}
	if LastRun(env) != nil {
		t.Error("a dry run must not be recorded as history")
	}

	kinds = nil
	real := Run(context.Background(), env, sels, false, report)
	if isDir(h + "/.cache/act") {
		t.Error("act cache should be gone")
	}
	if !isDir(h+"/.cache/phpactor") || !isDir(h+"/projects/keep") {
		t.Error("blocked and refused paths must survive")
	}
	if len(real.Cleaned) != 1 || real.Failed != 1 || real.Skipped != 1 {
		t.Errorf("summary = %+v", real)
	}
	if !slices.Contains(kinds, EventDone) || !slices.Contains(kinds, EventFail) || !slices.Contains(kinds, EventSkip) {
		t.Errorf("events = %v", kinds)
	}
	last := LastRun(env)
	if last == nil || last.Cleaned != 1 || !slices.Equal(last.Tasks, []string{"t"}) {
		t.Errorf("history = %+v", last)
	}
	if b, err := os.ReadFile(real.LogPath); err != nil || !strings.Contains(string(b), "FAIL") {
		t.Errorf("log %s should record the refusal: %v", real.LogPath, err)
	}
}

func TestCommandTasksRunOnlyForReal(t *testing.T) {
	env := testEnv(t, nil)
	ran := 0
	task := &Task{ID: "cmd", Title: "Cmd", Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
		ran++
		log("did it")
		return nil
	}}
	sels := []Selection{{Task: task, Items: []Item{{Paths: []string{env.Home + "/Library/Caches/Homebrew"}, Label: "brew cleanup", Selected: true}}}}
	Run(context.Background(), env, sels, true, func(Event) {})
	if ran != 0 {
		t.Fatal("dry run executed a command")
	}
	s := Run(context.Background(), env, sels, false, func(Event) {})
	if ran != 1 || len(s.Cleaned) != 1 || !s.Cleaned[0].Command {
		t.Errorf("ran=%d summary=%+v", ran, s)
	}
}

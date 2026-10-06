package clean

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func buildCachesTask() *Task {
	return &Task{
		ID:    "build-caches",
		Tier:  Tier1,
		Title: "Build caches (.next, .turbo)",
		About: "Next.js and Turborepo output across ~/projects, worktrees included. " +
			"The next dev server or build recreates it; that first start is slower. " +
			"Folders a running dev server uses are skipped.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			root := env.Home + "/projects"
			prune := map[string]bool{"node_modules": true, ".git": true}
			dirs := findDirs(ctx, root, 12, prune, func(_, name string) bool {
				return name == ".next" || name == ".turbo"
			})
			servers := env.Procs(ctx).Matching("next dev", "next-server", "next start", "next build",
				"turbopack", "turbo run", "turbo daemon", "turbo watch")
			items := make([]Item, 0, len(dirs))
			for _, d := range dirs {
				it := Item{Paths: []string{d}, Label: strings.TrimPrefix(d, root+"/"), Selected: true}
				if p := blockingServer(servers, d); p != nil {
					it.Blocked = fmt.Sprintf("%s is running there (pid %d)", p.Name(), p.PID)
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

// blockingServer finds a dev server working on the project that owns cacheDir.
// A server whose working directory is unknown blocks everything.
func blockingServer(servers Procs, cacheDir string) *Proc {
	project := filepath.Dir(cacheDir)
	for i := range servers {
		p := &servers[i]
		if p.Cwd == "" || within(cacheDir, p.Cwd) || within(p.Cwd, project) {
			return p
		}
	}
	return nil
}

func updaterLeftoversTask() *Task {
	return &Task{
		ID:    "updater-leftovers",
		Tier:  Tier1,
		Title: "App update leftovers",
		About: "Installers that app updaters (Sparkle, Squirrel, electron-updater, Google, VS Code) " +
			"downloaded and already applied. An app that needs one again downloads it again.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			caches := env.Home + "/Library/Caches"
			support := env.Home + "/Library/Application Support"
			var found []string
			found = append(found, entriesHolding(env.Home, caches, "org.sparkle-project.Sparkle")...)
			found = append(found, entries(env.Home, caches, func(n string) bool {
				return strings.HasSuffix(n, ".ShipIt") || strings.HasSuffix(n, "-updater")
			})...)
			found = append(found, entriesHolding(env.Home, support, "CachedExtensionVSIXs")...)
			if p := support + "/Google/GoogleUpdater/crx_cache"; isDir(p) {
				found = append(found, p)
			}
			var items []Item
			for _, m := range found {
				if isDir(m) {
					items = append(items, Item{Paths: []string{m}, Label: tilde(env.Home, m), Selected: true})
				}
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

// Every part starts with a letter, so "gh-2.98.0" (a versioned download) is not an id.
var bundleIDLike = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*(\.[A-Za-z][A-Za-z0-9-]*){2,}$`)

func orphanedCachesTask() *Task {
	return &Task{
		ID:    "orphaned-caches",
		Tier:  Tier1,
		Title: "Caches of apps that are gone",
		About: "Folders in ~/Library/Caches named after a bundle id that no installed or running app has. " +
			"Everything in Caches is disposable by design, so a wrong guess only costs a re-download.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			ids := installedBundleIDs(ctx, env)
			for id := range runningBundleIDs(ctx, env) {
				ids[id] = true
			}
			dir := env.Home + "/Library/Caches"
			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil, nil
			}
			var items []Item
			for _, e := range entries {
				name := e.Name()
				low := strings.ToLower(name)
				if !e.IsDir() || !bundleIDLike.MatchString(name) ||
					strings.HasPrefix(low, "com.apple.") || strings.HasSuffix(low, ".shipit") ||
					belongsToInstalled(ids, low) {
					continue
				}
				items = append(items, Item{
					Paths: []string{dir + "/" + name}, Label: name, Selected: true,
					Note: "no installed app has this bundle id",
				})
			}
			sizeItems(ctx, env, items)
			out := items[:0]
			for _, it := range items {
				if it.Size >= 20<<20 {
					out = append(out, it)
				}
			}
			return out, nil
		},
	}
}

// belongsToInstalled matches an app's own id and its helpers' ("com.x.app.helper").
func belongsToInstalled(ids map[string]bool, low string) bool {
	if ids[low] {
		return true
	}
	for id := range ids {
		if strings.HasPrefix(low, id+".") {
			return true
		}
	}
	return false
}

// runningBundleIDs covers apps running from outside the app folders.
func runningBundleIDs(ctx context.Context, env *Env) map[string]bool {
	ids := map[string]bool{}
	seen := map[string]bool{}
	for _, p := range env.Procs(ctx) {
		i := strings.Index(p.Exe, ".app/Contents/MacOS/")
		if i < 0 {
			continue
		}
		app := p.Exe[:i+4]
		if seen[app] {
			continue
		}
		seen[app] = true
		if id := bundleID(ctx, env, app); id != "" {
			ids[id] = true
		}
	}
	return ids
}

func devCachesTask() *Task {
	return &Task{
		ID:    "dev-caches",
		Tier:  Tier1,
		Title: "Dev tool caches",
		About: "Download and build caches of pnpm, npm/npx, node-gyp, TypeScript, Go, pip, uv, act, " +
			"Xcode DerivedData and friends. Each tool refills its cache on next use. " +
			"npx folders a running program uses (like an MCP server) are kept.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			known := []struct{ rel, label string }{
				{"Library/Caches/pnpm", "pnpm metadata cache"},
				{"Library/Caches/node-gyp", "node-gyp headers"},
				{"Library/Caches/typescript", "TypeScript type downloads"},
				{"Library/Caches/go-build", "Go build cache"},
				{"Library/Caches/pip", "pip cache"},
				{"Library/Caches/Yarn", "Yarn cache"},
				{"Library/Caches/deno", "Deno cache"},
				{"Library/Caches/VisualStudio", "Visual Studio for Mac cache"},
				{".cache/act", "act (GitHub Actions runner) cache"},
				{".cache/phpactor", "phpactor index"},
				{".cache/uv", "uv cache"},
				{".bun/install/cache", "Bun install cache"},
				{".npm/_cacache", "npm cache"},
				{"Library/Developer/Xcode/DerivedData", "Xcode DerivedData"},
			}
			procs := env.Procs(ctx)
			var items []Item
			for _, k := range known {
				p := env.Home + "/" + k.rel
				if !isDir(p) {
					continue
				}
				it := Item{Paths: []string{p}, Label: k.label, Selected: true, Note: tilde(env.Home, p)}
				if k.rel == "Library/Developer/Xcode/DerivedData" {
					if x := procs.App("MacOS/Xcode"); x != nil {
						it.Blocked = "Xcode is running"
					} else if b := procs.Matching("xcodebuild"); len(b) > 0 {
						it.Blocked = "xcodebuild is running"
					}
				}
				items = append(items, it)
			}
			for _, m := range glob(env.Home, ".cache/codex-runtimes/codex-runtime-install-*") {
				if isDir(m) {
					items = append(items, Item{Paths: []string{m}, Label: "Codex runtime installer leftover", Selected: true, Note: tilde(env.Home, m)})
				}
			}
			for _, m := range glob(env.Home, ".npm/_npx/*") {
				if !isDir(m) {
					continue
				}
				it := Item{Paths: []string{m}, Label: "npx " + npxPackages(m), Selected: true, Note: tilde(env.Home, m)}
				if p := procs.Using(m); p != nil {
					it.Blocked = fmt.Sprintf("in use by %s (pid %d)", p.Name(), p.PID)
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

// npxPackages names what an npx cache folder holds, from its package.json.
func npxPackages(dir string) string {
	b, err := os.ReadFile(dir + "/package.json")
	if err != nil {
		return filepath.Base(dir)
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil || len(pkg.Dependencies) == 0 {
		return filepath.Base(dir)
	}
	names := make([]string, 0, len(pkg.Dependencies))
	for n := range pkg.Dependencies {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func browserCachesTask() *Task {
	return &Task{
		ID:    "browser-caches",
		Tier:  Tier1,
		Title: "Browser caches",
		About: "Page caches of Chrome, Arc, Firefox, Zen, Brave, Edge, Vivaldi and Opera. " +
			"Sign-ins, history and tabs live elsewhere and stay. " +
			"A browser that is running is skipped — quit it and refresh.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			browsers := []struct{ label, rel, exe string }{
				{"Chrome", "Library/Caches/Google/Chrome", "MacOS/Google Chrome"},
				{"Arc", "Library/Caches/Arc", "MacOS/Arc"},
				{"Firefox", "Library/Caches/Firefox", "MacOS/firefox"},
				{"Zen", "Library/Caches/zen", "MacOS/zen"},
				{"Brave", "Library/Caches/BraveSoftware", "MacOS/Brave Browser"},
				{"Edge", "Library/Caches/Microsoft Edge", "MacOS/Microsoft Edge"},
				{"Vivaldi", "Library/Caches/Vivaldi", "MacOS/Vivaldi"},
				{"Opera", "Library/Caches/com.operasoftware.Opera", "MacOS/Opera"},
			}
			procs := env.Procs(ctx)
			var items []Item
			for _, b := range browsers {
				p := env.Home + "/" + b.rel
				if !isDir(p) {
					continue
				}
				it := Item{Paths: []string{p}, Label: b.label, Selected: true, Note: tilde(env.Home, p)}
				if procs.App(b.exe) != nil {
					it.Blocked = b.label + " is running — quit it first"
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

func homebrewTask() *Task {
	return &Task{
		ID:    "homebrew",
		Tier:  Tier1,
		Title: "Homebrew cleanup",
		About: "Runs `brew cleanup -s --prune=all`: removes every cached download and the old " +
			"versions of formulae you have upgraded. Installed packages are untouched.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			if _, err := env.LookPath("brew"); err != nil {
				return nil, nil
			}
			cache := env.Home + "/Library/Caches/Homebrew"
			it := Item{
				Paths: []string{cache}, Label: "brew cleanup -s --prune=all", Selected: true,
				Note: "size shown is the download cache; old versions come on top",
			}
			if len(env.Procs(ctx).Matching("Homebrew/brew.sh", "Homebrew/brew.rb")) > 0 {
				it.Blocked = "another brew command is running"
			}
			it.Size = env.Size(ctx, cache)
			return []Item{it}, nil
		},
		Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
			return env.Stream(ctx, log, []string{"HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ENV_HINTS=1"},
				"brew", "cleanup", "-s", "--prune=all")
		},
	}
}

func puppeteerTask() *Task {
	return &Task{
		ID:    "puppeteer",
		Tier:  Tier1,
		Title: "Old puppeteer browsers",
		About: "Puppeteer downloads a Chrome per version and never deletes one. This keeps the newest " +
			"of each kind. A project pinned to an older one fetches it again with " +
			"`npx puppeteer browsers install chrome`.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			root := env.Home + "/.cache/puppeteer"
			kinds, err := os.ReadDir(root)
			if err != nil {
				return nil, nil
			}
			procs := env.Procs(ctx)
			var items []Item
			for _, k := range kinds {
				if !k.IsDir() {
					continue
				}
				builds, newest := versionedDirs(root + "/" + k.Name())
				for _, b := range builds {
					if b == newest {
						continue
					}
					it := Item{
						Paths: []string{b}, Label: k.Name() + " " + filepath.Base(b), Selected: true,
						Note: "keeping " + filepath.Base(newest),
					}
					if p := procs.Using(b); p != nil {
						it.Blocked = fmt.Sprintf("in use by %s (pid %d)", p.Name(), p.PID)
					}
					items = append(items, it)
				}
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

// versionedDirs lists dir's subfolders that end in a version, and the newest one.
func versionedDirs(dir string) (all []string, newest string) {
	entries, _ := os.ReadDir(dir)
	var best []int
	for _, e := range entries {
		v, ok := parseVersion(e.Name())
		if !e.IsDir() || !ok {
			continue
		}
		p := dir + "/" + e.Name()
		all = append(all, p)
		if best == nil || compareVersions(v, best) > 0 {
			best, newest = v, p
		}
	}
	return all, newest
}

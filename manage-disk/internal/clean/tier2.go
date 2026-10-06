package clean

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"manage-disk/internal/docker"
)

func dockerTask() *Task {
	return &Task{
		ID:    "docker",
		Tier:  Tier2,
		Title: "Docker build cache",
		About: "Runs `docker builder prune -f`, `docker image prune -f`, then `docker builder prune -f` " +
			"again — the first pass leaves cache pinned by dangling images. Volumes are never " +
			"touched: your dev databases live there. The Mac gets the space back a few minutes " +
			"later, as Docker trims its disk image.",
		Cost: "The next image build starts cold.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			if _, err := env.LookPath("docker"); err != nil {
				return nil, nil
			}
			raw := env.Home + "/Library/Containers/com.docker.docker/Data/vms/0/data/Docker.raw"
			it := Item{
				Paths: []string{raw}, Label: "builder prune → image prune → builder prune", Selected: true,
				Note: "never touches volumes",
			}
			if len(env.Procs(ctx).Matching("docker build", "buildx", "docker compose build", "bake")) > 0 {
				it.Blocked = "a docker build is running"
			}
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			// raw is never touched here: it sits in a folder macOS guards with a dialog.
			if _, err := env.Output(cctx, "docker", "info", "--format", "{{.ServerVersion}}"); err != nil {
				it.Blocked = "Docker isn't running — start Docker Desktop, then refresh"
				it.Note = "starting Docker shows how much build cache it holds"
				return []Item{it}, nil
			}
			out, err := env.Output(cctx, "docker", "system", "df", "--format", "{{json .}}")
			if err == nil {
				for _, line := range strings.Split(out, "\n") {
					var row struct{ Type, Size, Reclaimable string }
					if json.Unmarshal([]byte(line), &row) == nil && row.Type == "Build Cache" {
						it.Size = docker.ParseSize(row.Reclaimable)
						it.Note = "build cache " + row.Size + ", reclaimable " + row.Reclaimable + "; never touches volumes"
					}
				}
			}
			return []Item{it}, nil
		},
		Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
			for _, args := range [][]string{{"builder", "prune", "-f"}, {"image", "prune", "-f"}, {"builder", "prune", "-f"}} {
				log("$ docker " + strings.Join(args, " "))
				if err := env.Stream(ctx, log, nil, "docker", args...); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func pnpmStoresTask() *Task {
	return &Task{
		ID:    "pnpm-stores",
		Tier:  Tier2,
		Title: "pnpm stores",
		About: "pnpm's download stores (v3 is pnpm 7–9's format). Installed node_modules keep " +
			"working: pnpm cloned the files, it did not link them. The real gain is smaller than " +
			"shown, because clones share disk blocks with node_modules.",
		Cost:  "The next installs download packages again.",
		Heavy: true,
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			installing := installingPnpm(env.Procs(ctx))
			var items []Item
			for _, m := range glob(env.Home, "Library/pnpm/store/v*") {
				if !isDir(m) {
					continue
				}
				it := Item{Paths: []string{m}, Label: "store " + filepath.Base(m), Selected: true, Note: tilde(env.Home, m)}
				if filepath.Base(m) == "v3" {
					it.Note = "pnpm 7–9's store format"
				}
				if installing != nil {
					it.Blocked = fmt.Sprintf("pnpm is installing (pid %d)", installing.PID)
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

func installingPnpm(procs Procs) *Proc {
	for i := range procs {
		c := procs[i].Command
		if !strings.Contains(c, "pnpm") {
			continue
		}
		for _, verb := range []string{" install", " i ", " add ", " update", " fetch", " store "} {
			if strings.Contains(c+" ", verb) {
				return &procs[i]
			}
		}
	}
	return nil
}

func worktreeModulesTask() *Task {
	return &Task{
		ID:    "worktree-node-modules",
		Tier:  Tier2,
		Title: "node_modules in worktrees",
		About: "The node_modules of every linked git worktree of your repos under ~/projects " +
			"(.claude/worktrees, .worktrees, ~/.codex/worktrees…). Code, branches and uncommitted " +
			"work are untouched. Worktrees something is working in are skipped.",
		Cost:  "Run pnpm install before working in one of them again.",
		Heavy: true,
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			procs := env.Procs(ctx)
			seen := map[string]bool{}
			var items []Item
			for _, repo := range gitRepos(ctx, env.Home+"/projects", 3) {
				for _, wt := range linkedWorktrees(ctx, env, repo) {
					if seen[wt] || !within(wt, env.Home) || guarded(env.Home, wt) {
						continue
					}
					seen[wt] = true
					mods := findDirs(ctx, wt, 4, map[string]bool{".git": true}, func(_, name string) bool {
						return name == "node_modules"
					})
					if len(mods) == 0 {
						continue
					}
					it := Item{Paths: mods, Label: tilde(env.Home, wt), Selected: true, Note: fmt.Sprintf("%d node_modules folders", len(mods))}
					if p := procs.Using(wt); p != nil {
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

func buildToolCachesTask() *Task {
	return &Task{
		ID:    "build-tool-caches",
		Tier:  Tier2,
		Title: "Gradle & CocoaPods caches",
		About: "Gradle's dependency and transform caches, and the CocoaPods download cache. " +
			"Installed Pods in your projects stay.",
		Cost: "The next Android build and pod install download everything again.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			procs := env.Procs(ctx)
			var items []Item
			if p := env.Home + "/.gradle/caches"; isDir(p) {
				it := Item{Paths: []string{p}, Label: "Gradle caches", Selected: true, Note: "~/.gradle/caches"}
				if len(procs.Matching("GradleDaemon")) > 0 {
					it.Blocked = "a Gradle daemon is running — `gradle --stop` first"
				}
				items = append(items, it)
			}
			if p := env.Home + "/Library/Caches/CocoaPods"; isDir(p) {
				it := Item{Paths: []string{p}, Label: "CocoaPods cache", Selected: true, Note: "~/Library/Caches/CocoaPods"}
				if len(procs.Matching("bin/pod ")) > 0 {
					it.Blocked = "pod is running"
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

func rustTargetsTask() *Task {
	return &Task{
		ID:    "rust-targets",
		Tier:  Tier2,
		Title: "Rust build output (target/)",
		About: "Cargo's target folders under ~/projects — only folders Cargo marked with " +
			"CACHEDIR.TAG, so a source folder that happens to be called target is never touched.",
		Cost:  "The next cargo build compiles from scratch.",
		Heavy: true,
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			root := env.Home + "/projects"
			prune := map[string]bool{"node_modules": true, ".git": true}
			dirs := findDirs(ctx, root, 6, prune, func(path, name string) bool {
				_, err := os.Stat(path + "/CACHEDIR.TAG")
				return name == "target" && err == nil
			})
			busy := env.Procs(ctx).Matching("cargo", "rustc", "rust-analyzer")
			items := make([]Item, 0, len(dirs))
			for _, d := range dirs {
				it := Item{Paths: []string{d}, Label: strings.TrimPrefix(d, root+"/"), Selected: true}
				if p := busy.Using(filepath.Dir(d)); p != nil {
					it.Blocked = fmt.Sprintf("%s is working there (pid %d)", p.Name(), p.PID)
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

func goModcacheTask() *Task {
	return &Task{
		ID:    "go-modcache",
		Tier:  Tier2,
		Title: "Go module cache",
		About: "Runs `go clean -modcache` (the files are read-only, so a plain rm cannot).",
		Cost:  "The next go build downloads its modules again.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			if _, err := env.LookPath("go"); err != nil {
				return nil, nil
			}
			dir, err := env.Output(ctx, "go", "env", "GOMODCACHE")
			if err != nil || dir == "" || !isDir(dir) {
				return nil, nil
			}
			it := Item{Paths: []string{dir}, Label: "go clean -modcache", Selected: true, Note: tilde(env.Home, dir)}
			if busy := env.Procs(ctx).Program("go").Matching(" build", " test", " run", " mod ", " install", " get "); len(busy) > 0 {
				it.Blocked = fmt.Sprintf("go is running (pid %d)", busy[0].PID)
			}
			it.Size = env.Size(ctx, dir)
			return dropEmpty([]Item{it}), nil
		},
		Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
			return env.Stream(ctx, log, nil, "go", "clean", "-modcache")
		},
	}
}

func androidAVDTask() *Task {
	return &Task{
		ID:    "android-avd",
		Tier:  Tier2,
		Title: "Android emulators",
		About: "Android Virtual Devices in ~/.android/avd. Emulators unused for 30 days are preselected.",
		Cost:  "Recreate a device in Android Studio's Device Manager when you need it.",
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			emulators := env.Procs(ctx).Matching("qemu-system", "emulator -avd", "emulator @")
			var items []Item
			for _, avd := range glob(env.Home, ".android/avd/*.avd") {
				if !isDir(avd) {
					continue
				}
				name := strings.TrimSuffix(filepath.Base(avd), ".avd")
				paths := []string{avd}
				if ini := filepath.Dir(avd) + "/" + name + ".ini"; fileExists(ini) {
					paths = append(paths, ini)
				}
				days := int(env.Now().Sub(newestMtime(avd)).Hours() / 24)
				it := Item{Paths: paths, Label: name, Selected: days > 30, Note: fmt.Sprintf("last used %d days ago", days)}
				for _, p := range emulators {
					if strings.Contains(p.Command, name) {
						it.Blocked = "this emulator is running"
					}
				}
				items = append(items, it)
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// newestMtime is the latest modification time of dir or anything directly in it.
func newestMtime(dir string) time.Time {
	var newest time.Time
	if fi, err := os.Stat(dir); err == nil {
		newest = fi.ModTime()
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if fi, err := e.Info(); err == nil && fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
	}
	return newest
}

package clean

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Catalog is every cleanup this tool knows, Tier 1 first.
func Catalog() []*Task {
	return []*Task{
		buildCachesTask(),
		updaterLeftoversTask(),
		orphanedCachesTask(),
		devCachesTask(),
		browserCachesTask(),
		homebrewTask(),
		puppeteerTask(),
		dockerTask(),
		pnpmStoresTask(),
		worktreeModulesTask(),
		buildToolCachesTask(),
		rustTargetsTask(),
		goModcacheTask(),
		nodeVersionsTask(),
		androidAVDTask(),
	}
}

func nodeVersionsTask() *Task {
	return &Task{
		ID:    "node-versions",
		Tier:  Tier2,
		Title: "Old Node.js versions",
		About: "Versions installed by nvm (zsh) and nvm.fish. Preselected: versions past Node's " +
			"end of life and patches a newer one supersedes — but never your default, nor a " +
			"version holding global packages you installed.",
		Cost:  "A project pinned to a removed version needs `nvm install` again.",
		Heavy: true,
		Discover: func(ctx context.Context, env *Env) ([]Item, error) {
			pins := nvmrcPins(ctx, env.Home+"/projects")
			procs := env.Procs(ctx)
			var items []Item
			for _, m := range nodeManagers(env.Home) {
				versions := listNodeVersions(m.dir)
				newest := map[int]nodeVersion{}
				for _, v := range versions {
					if cur, ok := newest[v.major()]; !ok || compareVersions(v.num, cur.num) > 0 {
						newest[v.major()] = v
					}
				}
				def := resolveNodeSpec(m.defaultSpec, versions)
				for _, v := range versions {
					it := Item{Paths: []string{v.dir}, Label: m.label + " " + v.name, Selected: true}
					var notes []string
					switch eol := nodeEOL(v.major()); {
					case !env.Now().Before(eol):
						notes = append(notes, "end of life since "+eol.Format("Jan 2006"))
					case newest[v.major()].dir == v.dir:
						it.Selected = false
						notes = append(notes, fmt.Sprintf("newest v%d", v.major()))
					default:
						notes = append(notes, "newer v"+strconv.Itoa(v.major())+" installed")
					}
					if v.dir == def {
						it.Selected = false
						notes = append(notes, "your default")
					}
					if g := globalPackages(v.dir); len(g) > 0 {
						it.Selected = false
						notes = append(notes, "globals: "+strings.Join(g, ", "))
					}
					if n := pins[strings.TrimPrefix(v.name, "v")]; n > 0 {
						notes = append(notes, fmt.Sprintf("pinned by %d .nvmrc", n))
					}
					if p := procs.Using(v.dir); p != nil {
						it.Blocked = fmt.Sprintf("in use by %s (pid %d)", p.Name(), p.PID)
					}
					it.Note = strings.Join(notes, " · ")
					items = append(items, it)
				}
			}
			sizeItems(ctx, env, items)
			return dropEmpty(items), nil
		},
	}
}

// nodeEOL follows Node's fixed schedule: even majors ship in April of year
// 2013+major/2 and end three years later on April 30; odd majors ship that
// October and end the next June 1.
func nodeEOL(major int) time.Time {
	if major%2 == 0 {
		return time.Date(2013+major/2+3, time.April, 30, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(2013+(major-1)/2+1, time.June, 1, 0, 0, 0, 0, time.UTC)
}

type nodeManager struct {
	label, dir, defaultSpec string
}

func nodeManagers(home string) []nodeManager {
	return []nodeManager{
		{"fish", home + "/.local/share/nvm", fishDefaultNode(home)},
		{"nvm", home + "/.nvm/versions/node", firstLine(home + "/.nvm/alias/default")},
	}
}

type nodeVersion struct {
	dir, name string
	num       []int
}

func (v nodeVersion) major() int { return v.num[0] }

func listNodeVersions(dir string) []nodeVersion {
	entries, _ := os.ReadDir(dir)
	var out []nodeVersion
	for _, e := range entries {
		num, ok := parseVersion(e.Name())
		if !e.IsDir() || !ok || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		out = append(out, nodeVersion{dir: dir + "/" + e.Name(), name: e.Name(), num: num})
	}
	sort.Slice(out, func(i, j int) bool { return compareVersions(out[i].num, out[j].num) < 0 })
	return out
}

// resolveNodeSpec turns "20", "v20.18.0", "20.18" or "node" into the installed
// version it selects; "" when it selects none (e.g. "lts/*").
func resolveNodeSpec(spec string, versions []nodeVersion) string {
	spec = strings.TrimPrefix(strings.TrimSpace(spec), "v")
	if spec == "node" || spec == "latest" || spec == "current" {
		if len(versions) > 0 {
			return versions[len(versions)-1].dir
		}
		return ""
	}
	want, ok := parseVersion(spec)
	if !ok {
		return ""
	}
	best := ""
	for _, v := range versions { // ascending, so the last match is the newest
		if len(v.num) >= len(want) && compareVersions(v.num[:len(want)], want) == 0 {
			best = v.dir
		}
	}
	return best
}

// fishDefaultNode reads nvm.fish's universal nvm_default_version.
func fishDefaultNode(home string) string {
	b, err := os.ReadFile(home + "/.config/fish/fish_variables")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "SETUVAR nvm_default_version:"); ok {
			return unescapeFish(v)
		}
	}
	return ""
}

// unescapeFish decodes fish's \xHH escapes in universal variables.
func unescapeFish(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && s[i+1] == 'x' {
			if n, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func firstLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimSpace(line)
}

// globalPackages lists the tools installed with `npm -g`. Package managers
// don't count: nearly every version has pnpm or yarn, and they reinstall in seconds.
func globalPackages(versionDir string) []string {
	root := versionDir + "/lib/node_modules"
	entries, _ := os.ReadDir(root)
	var out []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case name == "npm" || name == "corepack" || name == "pnpm" || name == "yarn" || strings.HasPrefix(name, "."):
		case strings.HasPrefix(name, "@"):
			scoped, _ := os.ReadDir(root + "/" + name)
			for _, s := range scoped {
				out = append(out, name+"/"+s.Name())
			}
		default:
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// nvmrcPins counts .nvmrc files under root per exact version they pin.
func nvmrcPins(ctx context.Context, root string) map[string]int {
	pins := map[string]int{}
	base := strings.Count(root, "/")
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if n := d.Name(); n == "node_modules" || n == ".git" || strings.Count(path, "/")-base >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == ".nvmrc" {
			v := strings.TrimPrefix(firstLine(path), "v")
			if strings.Count(v, ".") == 2 {
				pins[v]++
			}
		}
		return nil
	})
	return pins
}

package scan

import (
	"sort"
	"strings"
)

// Category is a named share of the home folder. Categories never overlap, and
// together with "Everything else" they add up to the whole scan.
type Category struct {
	Name    string
	Size    int64
	Partial bool // part of it was left out of the scan for privacy
}

// A place may sit inside another category's place (.cache/lm-studio inside
// .cache); it then counts for its own category and is taken off the outer one.
var categoryDefs = []struct {
	name string
	rel  []string
}{
	{"Projects & worktrees", []string{"projects", ".codex/worktrees"}},
	{"Docker VM", []string{"Library/Containers/com.docker.docker"}},
	{"App & tool caches", []string{"Library/Caches", ".cache", ".npm"}},
	{"Node & JS toolchains", []string{"Library/pnpm", ".nvm", ".local/share/nvm", ".bun", ".yarn", ".pnpm-state"}},
	{"Xcode & simulators", []string{"Library/Developer"}},
	{"Android, JVM, Go, Rust", []string{".android", "Library/Android", ".gradle", ".m2", "go", ".cargo", ".rustup", ".sdkman", ".gem", ".cocoapods", ".swiftpm"}},
	{"AI models & agents", []string{".ollama", ".cache/lm-studio", ".lmstudio", ".local/share/goose", ".claude", ".codex", "Library/Application Support/Claude"}},
	{"Other app data", []string{"Library/Application Support"}},
	{"Media & personal", []string{"audiobooks", "Pictures", "Movies", "Music", "Downloads", "Desktop", "Documents"}},
}

// EverythingElse names the remainder category.
const EverythingElse = "Everything else"

// Categories splits a home-folder scan into disjoint categories, largest first,
// with "Everything else" last.
func Categories(root *Node) []Category {
	type place struct {
		rel     string
		cat     int
		size    int64
		private bool
	}
	var places []place
	for i, d := range categoryDefs {
		for _, rel := range d.rel {
			p := place{rel: rel, cat: i}
			p.size, p.private = lookup(root, rel)
			places = append(places, p)
		}
	}
	under := func(a, b string) bool { return strings.HasPrefix(a, b+"/") }

	sizes := make([]int64, len(categoryDefs))
	partial := make([]bool, len(categoryDefs))
	var topmost int64
	for i, p := range places {
		own := p.size
		for j, q := range places {
			if i == j || !under(q.rel, p.rel) {
				continue
			}
			nearest := true // q is not inside another place that is itself inside p
			for k, r := range places {
				if k != i && k != j && under(q.rel, r.rel) && under(r.rel, p.rel) {
					nearest = false
					break
				}
			}
			if nearest {
				own -= q.size
			}
		}
		sizes[p.cat] += own
		partial[p.cat] = partial[p.cat] || p.private
		inside := false
		for j, q := range places {
			if i != j && under(p.rel, q.rel) {
				inside = true
				break
			}
		}
		if !inside {
			topmost += p.size
		}
	}

	out := make([]Category, 0, len(categoryDefs)+1)
	for i, d := range categoryDefs {
		out = append(out, Category{Name: d.name, Size: max(0, sizes[i]), Partial: partial[i]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return append(out, Category{Name: EverythingElse, Size: max(0, root.Size-topmost)})
}

// lookup sizes the place at rel, and says whether it (or a folder above it)
// went unread because macOS guards it or never answered.
func lookup(root *Node, rel string) (size int64, private bool) {
	cur := root
	for _, part := range strings.Split(rel, "/") {
		if cur.Skipped == SkipPrivacy || cur.Skipped == SkipNoResponse {
			return 0, true
		}
		var next *Node
		for _, c := range cur.Children {
			if c.Name == part {
				next = c
				break
			}
		}
		if next == nil {
			return 0, false
		}
		cur = next
	}
	return cur.Size, cur.Skipped == SkipPrivacy || cur.Skipped == SkipNoResponse
}

// containerRel are folders whose size says little on its own — Library,
// projects, .cache — so hotspots always look inside them.
var containerRel = map[string]bool{
	"": true, "Library": true, "Library/Application Support": true, "Library/Caches": true,
	"Library/Containers": true, "Library/Developer": true, "projects": true,
	".local": true, ".local/share": true, ".cache": true, ".config": true,
}

// Hotspots finds the most telling big entries: it walks down while one child
// holds 80% or more of a folder (so a Docker VM shows as Docker.raw, not as
// .../Data/vms/0), and stops at the first folder whose size is spread out.
func Hotspots(root *Node, limit int, min int64) []*Node {
	var out []*Node
	var visit func(x *Node, rel string)
	visit = func(x *Node, rel string) {
		if x.Size < min || x.Skipped != "" {
			return
		}
		dominated := len(x.Children) > 0 && float64(x.Children[0].Size) >= 0.8*float64(x.Size)
		if x.IsDir && len(x.Children) > 0 && (containerRel[rel] || dominated) {
			for _, c := range x.Children {
				visit(c, join(rel, c.Name))
			}
			return
		}
		if rel != "" {
			out = append(out, x)
		}
	}
	visit(root, "")
	sortNodes(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func join(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

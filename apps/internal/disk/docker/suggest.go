package docker

import (
	"fmt"
	"slices"
	"sort"
	"time"
)

// DefaultTarget is the free space plans aim for unless told otherwise: room
// for a full image bake, with some to spare.
const DefaultTarget int64 = 22 << 30

// Plan is the cheapest set of removals that leaves Docker's disk with at
// least Target free, going by Docker's own estimates. Volumes are never in
// it: when nothing else is enough, it says how much is missing and which
// volumes would cover the rest, for you to decide.
type Plan struct {
	Target, Free int64 // the free space wanted, and what there is now
	Picks        []Pick
	Frees        int64    // about what the picks free together
	Short        int64    // still missing with everything but volumes; 0 when the target is met
	Volumes      []string // when short: keys of the largest volumes that would cover it
	Err          string   // why there is no plan
}

// Pick is one resource in a plan and what removing it costs.
type Pick struct {
	Key   string
	Cost  string
	Frees int64 // what it frees in this plan: all of it, or the part of the build cache pruned
	// Before is set when only part of the build cache goes: what was last
	// used before it, oldest first.
	Before  time.Time
	Records int // for a partial prune: how many records go
}

// Need is what must be freed to reach the target.
func (p Plan) Need() int64 { return max(0, p.Target-p.Free) }

// Reached reports whether the disk already has the target free.
func (p Plan) Reached() bool { return p.Err == "" && p.Need() == 0 }

// Keys are the picks' resource keys.
func (p Plan) Keys() []string {
	keys := make([]string, len(p.Picks))
	for i, pk := range p.Picks {
		keys[i] = pk.Key
	}
	return keys
}

// Ranks of what a removal loses, cheapest first, with what each costs per
// byte: a gigabyte of recent build cache is rebuilt by builds that run anyway,
// a gigabyte of image is downloaded again, and one only a rebuild brings back
// may not come back the same.
var (
	costs = []string{
		"nothing is lost",
		"nothing used it for a month, and a pull brings it back",
		"the next build of what made it starts colder",
		"docker compose up or run makes it again",
		"a pull brings it back",
		"only a rebuild brings it back",
	}
	weights = []int64{0, 1, 2, 3, 3, 8}
)

// rank orders what removing r loses; -1 means a plan never takes it: it is
// in use, or a volume whose data exists nowhere else.
func rank(r Resource) int {
	switch r.Verdict {
	case InUse, Data:
		return -1
	case Unused, Orphan:
		return 0
	case Old:
		return 1
	}
	switch { // Review
	case r.Kind == BuildCache:
		return 2
	case r.Kind == Container:
		return 3
	case r.Kind == Image && r.InRegistry:
		return 4
	}
	return 5
}

// A cut prunes the recent build cache in part, oldest first: the records
// last used before a moment. The last cut is all of it.
type cut struct {
	before  time.Time // zero for all of it
	size    int64
	records int
}

// cuts lists where the recent cache can be cut, smallest first. Records
// used within a minute of each other go together, so a clock a little off
// between the Mac and Docker's VM can't split a build's cache.
//
// A cut runs as builder prune --filter until=…, which prunes exactly what is
// older. Not --min-free-space: Docker 29.5 pruned a record with that target
// already met (tried on a throwaway record, 2026-09-29), so it would have
// taken the whole cache.
func (r Resource) cuts() []cut {
	uses := slices.Clone(r.uses)
	sort.SliceStable(uses, func(i, j int) bool { return uses[i].used.Before(uses[j].used) })
	var out []cut
	var size int64
	for i, u := range uses {
		size += u.size
		switch {
		case i+1 == len(uses):
			out = append(out, cut{size: size, records: i + 1})
		case uses[i+1].used.Sub(u.used) >= time.Minute:
			mid := u.used.Add(uses[i+1].used.Sub(u.used) / 2)
			out = append(out, cut{before: mid, size: size, records: i + 1})
		}
	}
	return out
}

// A step is one resource a plan takes and what it brings along.
type step struct {
	main       string
	keys       []string // newly taken, dependencies first
	needs      []string // everything it relies on, taken earlier or now
	cost, gain int64
	cut        int // the recent cache cut this step brings it to, or -1
	prevCut    int // what it was before
}

// Suggest plans the cheapest way to target free on Docker's disk, where cheap
// means losing as little as possible: every byte costs its rank's weight.
// What costs nothing all goes. Then it keeps taking whatever costs least per
// byte still missing, so overshooting counts against a pick, and finally
// drops, dearest first, any step the others made unnecessary. Recent build
// cache can go in part, oldest first, so the next build keeps what it used
// last. Estimates are Docker's rounded numbers, so it aims 5% past what is
// missing.
func Suggest(rep Report, target int64) Plan {
	p := Plan{Target: target, Free: rep.Disk.Free}
	switch {
	case rep.Err != "":
		p.Err = rep.Err
		return p
	case !rep.Disk.OK():
		p.Err = "Docker's disk isn't measured: " + rep.Disk.Err
		return p
	}
	need := p.Need()
	if need == 0 {
		return p
	}
	aim := need + need/20

	byKey := make(map[string]Resource, len(rep.Resources))
	var cache []string // build cache slices, which images built here need pruned too
	var recent string  // the slice that can be pruned in part
	var cuts []cut
	for _, r := range rep.Resources {
		byKey[r.Key()] = r
		if r.Kind == BuildCache {
			cache = append(cache, r.Key())
			if r.Until == 0 && rank(r) >= 0 {
				recent, cuts = r.Key(), r.cuts()
			}
		}
	}

	// closure is k and everything that must go with it, dependencies first;
	// nil when some of it can't go.
	var closure func(k string, seen map[string]bool) []string
	closure = func(k string, seen map[string]bool) []string {
		if seen[k] {
			return []string{}
		}
		seen[k] = true
		r, ok := byKey[k]
		if !ok || rank(r) < 0 {
			return nil
		}
		var out []string
		for _, d := range r.Implies {
			more := closure(d, seen)
			if more == nil {
				return nil
			}
			out = append(out, more...)
		}
		if r.Kind == Image && r.BuiltHere && len(r.Tags) > 0 {
			// Its layers may sit in the build cache: only a prune after it frees
			// them. Unless a build holds the cache; then the image goes alone.
			for _, c := range cache {
				if more := closure(c, seen); more != nil {
					out = append(out, more...)
				}
			}
		}
		return append(out, k)
	}

	taken := map[string]bool{}
	cutAt := -1 // how far the recent cache is taken: an index into cuts
	var steps []step
	plan := func(k string, left int64) (step, bool) {
		all := closure(k, map[string]bool{})
		if all == nil {
			return step{}, false
		}
		s := step{main: k, needs: all, cut: -1}
		for _, x := range all {
			r := byKey[x]
			size := r.Size
			if x == recent && len(cuts) > 0 {
				want := len(cuts) - 1 // what depends on the cache needs all of it
				if x == k {           // on its own, the smallest cut that covers what is left
					want = sort.Search(len(cuts)-1, func(i int) bool { return cuts[i].size >= left })
				}
				if want <= cutAt {
					continue
				}
				size = cuts[want].size
				if cutAt >= 0 {
					size -= cuts[cutAt].size
				}
				s.cut = want
			} else if taken[x] {
				continue
			}
			if !taken[x] {
				s.keys = append(s.keys, x)
			}
			s.gain += size
			s.cost += weights[rank(r)] * size
		}
		return s, s.gain > 0
	}
	take := func(s step) step {
		for _, k := range s.keys {
			taken[k] = true
		}
		if s.cut >= 0 {
			s.prevCut, cutAt = cutAt, s.cut
		}
		p.Frees += s.gain
		steps = append(steps, s)
		return s
	}

	var free, costly []Resource
	for _, r := range rep.Resources {
		switch rk := rank(r); {
		case rk == 0:
			free = append(free, r)
		case rk > 0:
			costly = append(costly, r)
		}
	}
	for _, r := range free {
		if s, ok := plan(r.Key(), 0); ok && !taken[r.Key()] {
			take(s)
		}
	}
	for p.Frees < aim {
		left := aim - p.Frees
		var best step
		var bestRatio float64
		found := false
		for _, r := range costly {
			if taken[r.Key()] {
				continue
			}
			s, ok := plan(r.Key(), left)
			if !ok {
				continue
			}
			// Cost per byte still missing (in floats: weight × bytes × bytes
			// overflows); on a tie, the smaller removal.
			ratio := float64(s.cost) / float64(min(s.gain, left))
			if !found || ratio < bestRatio || ratio == bestRatio && s.gain < best.gain {
				best, bestRatio, found = s, ratio, true
			}
		}
		if !found {
			break
		}
		take(best)
	}

	// Drop, dearest first, what the rest made unnecessary.
	if p.Frees >= aim {
		order := make([]int, len(steps))
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(i, j int) bool { return steps[order[i]].cost > steps[order[j]].cost })
		dropped := map[int]bool{}
		for _, i := range order {
			s := steps[i]
			if s.cost == 0 || p.Frees-s.gain < aim || s.cut >= 0 && s.cut != cutAt {
				continue // free, needed, or its cut was taken further since
			}
			relied := false
			for j, o := range steps {
				if j != i && !dropped[j] && slices.ContainsFunc(s.keys, func(k string) bool { return slices.Contains(o.needs, k) }) {
					relied = true
					break
				}
			}
			if !relied {
				dropped[i] = true
				p.Frees -= s.gain
				if s.cut >= 0 {
					cutAt = s.prevCut
				}
			}
		}
		kept := steps[:0]
		for i, s := range steps {
			if !dropped[i] {
				kept = append(kept, s)
			}
		}
		steps = kept
	}

	for _, s := range steps {
		main := byKey[s.main]
		for _, k := range s.keys {
			r := byKey[k]
			pk := Pick{Key: k, Cost: costs[rank(r)], Frees: r.Size}
			if k == recent && cutAt >= 0 && cutAt < len(cuts)-1 {
				c := cuts[cutAt]
				pk.Frees, pk.Before, pk.Records = c.size, c.before, c.records
				pk.Cost = fmt.Sprintf("the oldest %d of its %d records go, so the next build keeps what it used last", c.records, r.Records)
			}
			switch {
			case k == s.main:
			case r.Kind == BuildCache && main.Kind == Image: // pruned after it, which frees its layers
				pk.Cost += ", and pruning it frees the layers of " + main.Name
			default:
				pk.Cost += ", and it must go before " + main.Name
			}
			p.Picks = append(p.Picks, pk)
		}
	}

	if p.Frees < need {
		p.Short = need - p.Frees
		var vols []Resource
		for _, r := range rep.Resources {
			if r.Kind == Volume && r.Verdict == Data {
				vols = append(vols, r)
			}
		}
		sort.SliceStable(vols, func(i, j int) bool { return vols[i].Size > vols[j].Size })
		var covered int64
		for _, v := range vols {
			if covered >= p.Short {
				break
			}
			p.Volumes = append(p.Volumes, v.Key())
			covered += v.Size
		}
	}
	return p
}

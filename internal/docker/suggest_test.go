package docker

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

const gib = 1 << 30

func fixtureReport(t *testing.T) Report {
	t.Helper()
	var calls []string
	rep := Audit(context.Background(), testDeps(&calls))
	if rep.Err != "" {
		t.Fatal(rep.Err)
	}
	return rep
}

func sorted(keys []string) []string {
	keys = slices.Clone(keys)
	slices.Sort(keys)
	return keys
}

func TestSuggestTakesWhatCostsNothingFirst(t *testing.T) {
	rep := fixtureReport(t) // 17.7 GiB free; 1.5 GB costs nothing
	p := Suggest(rep, rep.Disk.Free+gib/2)
	want := []string{"build cache:stale", "image:sha256:dangling"} // not the empty volume: it frees nothing
	if got := sorted(p.Keys()); !slices.Equal(got, want) {
		t.Errorf("picks = %v, want %v", got, want)
	}
	if p.Frees != 1.5e9 || p.Short != 0 {
		t.Errorf("frees %d short %d", p.Frees, p.Short)
	}
	for _, pk := range p.Picks {
		if pk.Cost != "nothing is lost" {
			t.Errorf("%s: %q", pk.Key, pk.Cost)
		}
	}
}

func TestSuggestClimbsTheCostsOnlyAsFarAsItMust(t *testing.T) {
	rep := fixtureReport(t)
	// 3 GiB more: what costs nothing, then the recent build cache, which is
	// enough on its own, so neither the month-old redis nor the image only a
	// rebuild brings back.
	p := Suggest(rep, rep.Disk.Free+3*gib)
	want := []string{"build cache:recent", "build cache:stale", "image:sha256:dangling"}
	if got := sorted(p.Keys()); !slices.Equal(got, want) {
		t.Errorf("picks = %v, want %v", got, want)
	}
	if p.Frees < 3*gib || p.Short != 0 {
		t.Errorf("frees %d short %d", p.Frees, p.Short)
	}
}

func TestSuggestNeverTakesVolumesAndSaysWhatIsMissing(t *testing.T) {
	rep := fixtureReport(t)
	p := Suggest(rep, rep.Disk.Free+5*gib)
	for _, k := range p.Keys() {
		if r := resource(rep, k); r.Kind == Volume && r.Size > 0 || r.Verdict == InUse {
			t.Errorf("a plan took %s (%s)", k, r.Verdict)
		}
	}
	// Everything else goes, the app image with the stopped container pinning it.
	for _, k := range []string{"image:sha256:app", "container:c-worker"} {
		if !slices.Contains(p.Keys(), k) {
			t.Errorf("%s should be picked when short: %v", k, p.Keys())
		}
	}
	if want := int64(5*gib) - p.Frees; p.Short != want {
		t.Errorf("short %d, want %d", p.Short, want)
	}
	if !slices.Equal(p.Volumes, []string{"volume:old_data"}) {
		t.Errorf("the volume that would cover the rest: %v", p.Volumes)
	}
	var worker Pick
	for _, pk := range p.Picks {
		if pk.Key == "container:c-worker" {
			worker = pk
		}
	}
	if !strings.Contains(worker.Cost, "must go before app:local") {
		t.Errorf("a dependency says why it goes: %q", worker.Cost)
	}
}

func resource(rep Report, key string) Resource {
	for _, r := range rep.Resources {
		if r.Key() == key {
			return r
		}
	}
	return Resource{}
}

// Every byte removed costs something, so the plan loses as little as it can.
func TestSuggestLosesAsLittleAsItCan(t *testing.T) {
	img := func(id string, size int64) Resource {
		return Resource{Kind: Image, ID: id, Name: id, Tags: []string{id + ":1"}, Size: size, InRegistry: true, Verdict: Review}
	}
	rep := Report{Disk: Disk{Total: 60 * gib, Free: 10 * gib}, Resources: []Resource{
		img("huge", 8*gib), img("big", 3*gib), img("fits", 1200<<20), img("small", 300<<20),
	}}
	// 1 GiB to go (1.05 aimed): fits alone covers it, with the least to spare.
	if got := Suggest(rep, 11*gib).Keys(); !slices.Equal(got, []string{"image:fits"}) {
		t.Errorf("1 GiB: %v", got)
	}
	// 4 GiB: three smaller images lose 4.5 GiB, where huge alone loses 8.
	if got := sorted(Suggest(rep, 14*gib).Keys()); !slices.Equal(got, []string{"image:big", "image:fits", "image:small"}) {
		t.Errorf("4 GiB: %v", got)
	}
	// 8.5 GiB: huge, and fits to cover the rest; big and small turn out unneeded.
	if got := sorted(Suggest(rep, 18*gib+gib/2).Keys()); !slices.Equal(got, []string{"image:fits", "image:huge"}) {
		t.Errorf("8.5 GiB: %v", got)
	}
	// 13 GiB is more than all of them: everything goes, and the plan says it's short.
	if p := Suggest(rep, 23*gib); len(p.Picks) != 4 || p.Short != 13*gib-p.Frees {
		t.Errorf("13 GiB: %v short %d", p.Keys(), p.Short)
	}
}

func TestSuggestWhenThereIsNothingToDo(t *testing.T) {
	rep := fixtureReport(t)
	if p := Suggest(rep, rep.Disk.Free-gib); !p.Reached() || len(p.Picks) > 0 {
		t.Errorf("already there: %+v", p)
	}
	rep.Disk = Disk{Err: "no alpine"}
	if p := Suggest(rep, 22*gib); p.Err == "" || p.Reached() {
		t.Errorf("unmeasured disk: %+v", p)
	}
}

// An image built here only frees its layers once the build cache is pruned
// after it, so the plan prunes it too, unless a build is using the cache.
func TestSuggestPrunesTheCacheAfterImagesBuiltHere(t *testing.T) {
	built := Resource{Kind: Image, ID: "web", Name: "web:1", Tags: []string{"web:1"}, Size: 2 * gib, InRegistry: true, BuiltHere: true, Verdict: Review}
	cache := Resource{Kind: BuildCache, ID: "recent", Name: "recent cache", Size: 100 << 20, Verdict: Review}
	rep := Report{Disk: Disk{Total: 60 * gib, Free: 10 * gib}, Resources: []Resource{built, cache}}
	// The cache alone is too small, so the image is needed, and the cache with it.
	p := Suggest(rep, 11*gib)
	if got := sorted(p.Keys()); !slices.Equal(got, []string{"build cache:recent", "image:web"}) {
		t.Errorf("picks = %v", got)
	}
	cache.Verdict = InUse
	rep.Resources = []Resource{built, cache}
	if got := Suggest(rep, 11*gib).Keys(); !slices.Equal(got, []string{"image:web"}) {
		t.Errorf("with a build running, the image still goes alone: %v", got)
	}
}

// Recent build cache goes oldest first, and only as much as the target needs,
// so the next build keeps the cache it used last.
func TestSuggestPrunesRecentCacheOldestFirst(t *testing.T) {
	now := time.Now()
	recent := Resource{Kind: BuildCache, ID: "recent", Name: "recent cache", Verdict: Review, Size: 7e9, Records: 4,
		uses: []cacheUse{
			{now.Add(-10 * time.Minute), 4e9},
			{now.Add(-3 * time.Hour), 1e9},
			{now.Add(-2 * time.Hour), 1.5e9},
			{now.Add(-2*time.Hour + 30*time.Second), 0.5e9}, // the same build: goes with the one above
		}}
	rep := Report{Disk: Disk{Total: 60 * gib, Free: 10 * gib}, Resources: []Resource{recent}}

	p := Suggest(rep, 10*gib+2_500_000_000) // 2.5 GB to go: the two oldest builds (3 GB) cover it
	if len(p.Picks) != 1 {
		t.Fatalf("picks = %+v", p.Picks)
	}
	pk := p.Picks[0]
	if pk.Frees != 3e9 || pk.Records != 3 || p.Frees != 3e9 {
		t.Errorf("the oldest two builds: frees %d of %d records (plan %d)", pk.Frees, pk.Records, p.Frees)
	}
	if !pk.Before.After(now.Add(-2*time.Hour)) || !pk.Before.Before(now.Add(-10*time.Minute)) {
		t.Errorf("the cut falls between the builds: %s", now.Sub(pk.Before))
	}
	if size, n := recent.Cut(pk.Before); size != 3e9 || n != 3 {
		t.Errorf("cut measures %d in %d records", size, n)
	}
	r := recent
	r.Before = pk.Before
	cmd := strings.Join(r.Command(), " ")
	var secs int
	if _, err := fmt.Sscanf(cmd, "builder prune -f --filter until=%ds", &secs); err != nil || secs <= 600 || secs >= 7200 {
		t.Errorf("the command prunes only what is older than the cut: %q", cmd)
	}

	// More than all of it: it all goes, and the plan is short.
	p = Suggest(rep, 10*gib+8e9)
	if len(p.Picks) != 1 || !p.Picks[0].Before.IsZero() || p.Picks[0].Frees != 7e9 || p.Short == 0 {
		t.Errorf("past all of it: %+v short %d", p.Picks, p.Short)
	}
}

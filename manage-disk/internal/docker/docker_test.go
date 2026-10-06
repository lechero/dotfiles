package docker

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func ago(d time.Duration) string { return now.Add(-d).Format(time.RFC3339Nano) }

const day = 24 * time.Hour

var anonVolume = strings.Repeat("ab", 32)

// fakeDocker answers like a Docker engine holding a small dev setup:
//
//   - shop: a running compose project (db container, postgres image, volume)
//   - gone: a stopped compose container whose worktree folder was removed,
//     pinning a redis image and a volume
//   - worker: a container stopped two hours ago, running an image never pushed
//   - an untagged image, an empty volume, an anonymous one, and a volume
//     of a project remembered to have run from a folder now gone
//   - build cache: 1 GB stale, 2 GB recent, 3 GB shared with images
func fakeDocker(calls *[]string) func(context.Context, ...string) (string, error) {
	df := fmt.Sprintf(`{
	"Images": [
		{"ID": "sha256:postgres", "Repository": "postgres", "Tag": "17", "Size": "291MB", "SharedSize": "8.6MB", "UniqueSize": "282MB"},
		{"ID": "sha256:redis", "Repository": "redis", "Tag": "7", "Size": "40MB", "SharedSize": "0B", "UniqueSize": "40MB"},
		{"ID": "sha256:dangling", "Repository": "<none>", "Tag": "<none>", "Size": "500MB", "SharedSize": "0B", "UniqueSize": "500MB"},
		{"ID": "sha256:app", "Repository": "app", "Tag": "local", "Size": "100MB", "SharedSize": "0B", "UniqueSize": "100MB"},
		{"ID": "sha256:alpine", "Repository": "alpine", "Tag": "latest", "Size": "8.6MB", "SharedSize": "8.6MB", "UniqueSize": "0B"}
	],
	"Containers": [
		{"ID": "c-db", "Names": "shop-db-1", "Image": "postgres:17", "State": "running", "Size": "63B"},
		{"ID": "c-gone", "Names": "gone-cache-1", "Image": "redis:7", "State": "exited", "Size": "0B"},
		{"ID": "c-worker", "Names": "worker", "Image": "app:local", "State": "exited", "Size": "0B"}
	],
	"Volumes": [
		{"Name": "shop_db", "Driver": "local", "Links": "1", "Size": "300MB"},
		{"Name": "gone_cache", "Driver": "local", "Links": "1", "Size": "5MB"},
		{"Name": "empty_one", "Driver": "local", "Links": "0", "Size": "0B"},
		{"Name": %q, "Driver": "local", "Links": "0", "Size": "1MB"},
		{"Name": "old_data", "Driver": "local", "Links": "0", "Size": "4GB"}
	],
	"BuildCache": [
		{"ID": "r1", "Size": "1GB", "InUse": "false", "Shared": "false", "LastUsedAt": %q},
		{"ID": "r2", "Size": "2GB", "InUse": "false", "Shared": "false", "LastUsedAt": %q},
		{"ID": "r3", "Size": "3GB", "InUse": "false", "Shared": "true", "LastUsedAt": %q}
	]}`, anonVolume,
		now.Add(-10*day).Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		now.Add(-day).Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		now.Add(-day).Format("2006-01-02 15:04:05.999999999 -0700 MST"))

	containers := fmt.Sprintf(`[
		{"Id": "c-db", "Name": "/shop-db-1", "Image": "sha256:postgres", "Created": %q, "SizeRw": 63,
		 "State": {"Status": "running", "Running": true, "FinishedAt": "0001-01-01T00:00:00Z"},
		 "Config": {"Image": "postgres:17", "Labels": {"com.docker.compose.project": "shop", "com.docker.compose.service": "db",
		   "com.docker.compose.project.working_dir": "/home/projects/shop"}},
		 "Mounts": [{"Type": "volume", "Name": "shop_db"}, {"Type": "bind", "Source": "/home/projects/shop/init.sql"}]},
		{"Id": "c-gone", "Name": "/gone-cache-1", "Image": "sha256:redis", "Created": %q, "SizeRw": 0,
		 "State": {"Status": "exited", "FinishedAt": %q},
		 "Config": {"Image": "redis:7", "Labels": {"com.docker.compose.project": "gone",
		   "com.docker.compose.project.working_dir": "/home/projects/app/.claude/worktrees/gone"}},
		 "Mounts": [{"Type": "volume", "Name": "gone_cache"}]},
		{"Id": "c-worker", "Name": "/worker", "Image": "sha256:app", "Created": %q, "SizeRw": 0,
		 "State": {"Status": "exited", "FinishedAt": %q}, "Config": {"Image": "app:local"}}
	]`, ago(3*day), ago(40*day), ago(35*day), ago(5*day), ago(2*time.Hour))

	images := fmt.Sprintf(`[
		{"Id": "sha256:postgres", "RepoTags": ["postgres:17"], "RepoDigests": ["postgres@sha256:1"], "Created": %q, "Metadata": {"LastTagTime": "0001-01-01T00:00:00Z"}},
		{"Id": "sha256:redis", "RepoTags": ["redis:7"], "RepoDigests": ["redis@sha256:2"], "Created": %q, "Metadata": {"LastTagTime": "0001-01-01T00:00:00Z"}},
		{"Id": "sha256:dangling", "RepoTags": [], "RepoDigests": [], "Created": %q, "Metadata": {"LastTagTime": %q}},
		{"Id": "sha256:app", "RepoTags": ["app:local", "app:dev"], "RepoDigests": [], "Created": %q, "Metadata": {"LastTagTime": %q}},
		{"Id": "sha256:alpine", "RepoTags": ["alpine:latest"], "RepoDigests": ["alpine@sha256:3"], "Created": %q, "Metadata": {"LastTagTime": "0001-01-01T00:00:00Z"}}
	]`, ago(10*day), ago(60*day), ago(3*day), ago(3*day), ago(5*day), ago(5*day), ago(11*day))

	volumes := fmt.Sprintf(`[
		{"Name": "shop_db", "CreatedAt": %q, "Labels": {"com.docker.compose.project": "shop"}},
		{"Name": "gone_cache", "CreatedAt": %q, "Labels": {"com.docker.compose.project": "gone"}},
		{"Name": "empty_one", "CreatedAt": %q, "Labels": null},
		{"Name": %q, "CreatedAt": %q, "Labels": {"com.docker.volume.anonymous": ""}},
		{"Name": "old_data", "CreatedAt": %q, "Labels": {"com.docker.compose.project": "old"}}
	]`, ago(90*day), ago(40*day), ago(2*day), anonVolume, ago(20*day), ago(200*day))

	var mu sync.Mutex // the audit asks several things at once
	return func(_ context.Context, args ...string) (string, error) {
		mu.Lock()
		*calls = append(*calls, strings.Join(args, " "))
		mu.Unlock()
		switch strings.Join(args[:min(2, len(args))], " ") {
		case "system df":
			return df, nil
		case "container inspect":
			return containers, nil
		case "image inspect":
			return images, nil
		case "volume inspect":
			return volumes, nil
		}
		if args[0] == "run" {
			return "Filesystem 1-blocks Used Available Capacity Mounted on\n" +
				"overlay 62671097856 43708727296 18962370560 70% /\n", nil
		}
		return "", fmt.Errorf("unexpected docker %v", args)
	}
}

func testDeps(calls *[]string) Deps {
	return Deps{
		Home:      "/home",
		Now:       now,
		Docker:    fakeDocker(calls),
		DirExists: func(p string) bool { return p == "/home/projects/shop" },
		Projects: func(name string) []string {
			if name == "old" {
				return []string{"/home/projects/old"} // remembered, now gone
			}
			return nil
		},
	}
}

func find(t *testing.T, rep Report, kind Kind, id string) Resource {
	t.Helper()
	for _, r := range rep.Resources {
		if r.Kind == kind && r.ID == id {
			return r
		}
	}
	t.Fatalf("no %s %s in the report", kind, id)
	return Resource{}
}

func reasons(r Resource) string { return strings.Join(r.Reasons, "; ") }

func TestAuditJudgesEveryResource(t *testing.T) {
	var calls []string
	rep := Audit(context.Background(), testDeps(&calls))
	if rep.Err != "" {
		t.Fatal(rep.Err)
	}
	cases := []struct {
		kind    Kind
		id      string
		verdict Verdict
		reason  string
	}{
		{Container, "c-db", InUse, "running"},
		{Image, "sha256:postgres", InUse, "running container shop-db-1 uses it"},
		{Volume, "shop_db", InUse, "running container shop-db-1 mounts it"},
		{Container, "c-gone", Orphan, "ran from ~/projects/app/.claude/worktrees/gone, which is gone"},
		{Image, "sha256:redis", Old, "must go first"},
		{Volume, "gone_cache", Data, "compose project gone ran from ~/projects/app/.claude/worktrees/gone, which is gone"},
		{Container, "c-worker", Review, "stopped 2 hours ago"},
		{Image, "sha256:app", Review, "never pushed"},
		{Image, "sha256:dangling", Unused, "untagged"},
		{Image, "sha256:alpine", Review, "measures Docker's disk"},
		{Volume, "empty_one", Unused, "empty"},
		{Volume, anonVolume, Data, "anonymous"},
		{Volume, "old_data", Data, "ran from ~/projects/old, which is gone"},
		{BuildCache, "stale", Unused, "no build has used it for a week"},
		{BuildCache, "recent", Review, "takes the older cache too"},
	}
	for _, c := range cases {
		r := find(t, rep, c.kind, c.id)
		if r.Verdict != c.verdict || !strings.Contains(reasons(r), c.reason) {
			t.Errorf("%s %s: %s (%s), want %s with %q", c.kind, c.id, r.Verdict, reasons(r), c.verdict, c.reason)
		}
	}

	// What must go first travels with the resource.
	if got := find(t, rep, Image, "sha256:redis").Implies; !slices.Equal(got, []string{"container:c-gone"}) {
		t.Errorf("redis image implies %v, want its stopped container", got)
	}
	if got := find(t, rep, Volume, "gone_cache").Implies; !slices.Equal(got, []string{"container:c-gone"}) {
		t.Errorf("gone_cache implies %v", got)
	}
	if got := find(t, rep, BuildCache, "recent").Implies; !slices.Equal(got, []string{"build cache:stale"}) {
		t.Errorf("recent cache implies %v", got)
	}

	// Sizes are what removal frees; shared cache is only reported.
	if s := find(t, rep, BuildCache, "stale").Size; s != 1e9 {
		t.Errorf("stale cache %d", s)
	}
	if s := find(t, rep, BuildCache, "recent").Size; s != 2e9 {
		t.Errorf("recent cache %d", s)
	}
	if rep.SharedCache != 3e9 {
		t.Errorf("shared cache %d", rep.SharedCache)
	}
	if img := find(t, rep, Image, "sha256:app"); img.Name != "app:local" || !slices.Equal(img.Tags, []string{"app:local", "app:dev"}) || !img.BuiltHere {
		t.Errorf("app image: %+v", img)
	}

	if rep.Disk.Total != 62671097856 || rep.Disk.Free != 18962370560 {
		t.Errorf("disk %+v", rep.Disk)
	}
	if !slices.ContainsFunc(calls, func(c string) bool {
		return strings.HasPrefix(c, "run --rm --pull never") && strings.Contains(c, "sha256:alpine df") || strings.Contains(c, "--entrypoint df sha256:alpine")
	}) {
		t.Errorf("disk should be measured with the local alpine, never pulling: %v", calls)
	}

	// Most removable first, largest first within a verdict.
	if first := rep.Resources[0]; first.ID != "stale" && first.ID != "sha256:dangling" {
		t.Errorf("first resource %s %s", first.Kind, first.ID)
	}
	for i := 1; i < len(rep.Resources); i++ {
		a, b := rep.Resources[i-1], rep.Resources[i]
		if a.Verdict < b.Verdict || a.Verdict == b.Verdict && a.Size < b.Size {
			t.Errorf("out of order: %s %s before %s %s", a.Verdict, a.Name, b.Verdict, b.Name)
		}
	}
	if dirs := rep.ProjectDirs["gone"]; !slices.Equal(dirs, []string{"/home/projects/app/.claude/worktrees/gone"}) {
		t.Errorf("project dirs to remember: %v", rep.ProjectDirs)
	}
}

func TestVolumesAreNeverPreselected(t *testing.T) {
	var calls []string
	rep := Audit(context.Background(), testDeps(&calls))
	for _, r := range rep.Resources {
		if r.Kind == Volume && r.Size > 0 && r.Verdict.Preselect() {
			t.Errorf("volume %s holding %d bytes is preselected (%s)", r.Name, r.Size, r.Verdict)
		}
		// Whatever is picked for you, what it needs gone first is too.
		if r.Verdict.Preselect() {
			for _, k := range r.Implies {
				for _, x := range rep.Resources {
					if x.Key() == k && !x.Verdict.Preselect() {
						t.Errorf("%s is preselected but needs %s (%s) first", r.Name, x.Name, x.Verdict)
					}
				}
			}
		}
	}
}

func TestCommandsNeverForce(t *testing.T) {
	cases := []struct {
		r    Resource
		want string
	}{
		{Resource{Kind: Container, ID: "c1"}, "rm c1"},
		{Resource{Kind: Image, ID: "sha256:x", Tags: []string{"a:1", "a:2"}}, "rmi a:1 a:2"},
		{Resource{Kind: Image, ID: "sha256:x"}, "rmi sha256:x"},
		{Resource{Kind: Volume, ID: "data"}, "volume rm data"},
		{Resource{Kind: BuildCache, ID: "stale", Until: 7 * day}, "builder prune -f --filter until=168h"},
		{Resource{Kind: BuildCache, ID: "recent"}, "builder prune -f"},
	}
	for _, c := range cases {
		got := strings.Join(c.r.Command(), " ")
		if got != c.want {
			t.Errorf("%s: %q, want %q", c.r.Kind, got, c.want)
		}
		if c.r.Kind != BuildCache && (strings.Contains(got, "-f") || strings.Contains(got, "--force")) {
			t.Errorf("%s removal forces: %q", c.r.Kind, got)
		}
	}
}

func TestBuildingKeepsTheCache(t *testing.T) {
	var calls []string
	d := testDeps(&calls)
	d.Building = func() string { return "docker-buildx (pid 42)" }
	rep := Audit(context.Background(), d)
	for _, id := range []string{"stale", "recent"} {
		if r := find(t, rep, BuildCache, id); r.Verdict != InUse {
			t.Errorf("%s cache during a build: %s", id, r.Verdict)
		}
	}
}

func TestEngineDownAndNoMeasuringImage(t *testing.T) {
	rep := Audit(context.Background(), Deps{Now: now, Docker: func(context.Context, ...string) (string, error) {
		return "", errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?")
	}})
	if !rep.Down || rep.Err != "Docker isn't running" {
		t.Errorf("down: %+v", rep)
	}

	disk := measureDisk(context.Background(), Deps{Docker: func(context.Context, ...string) (string, error) {
		t.Error("nothing should run without an image to measure with")
		return "", nil
	}}, []dfImage{{ID: "sha256:x", Repository: "postgres", Size: "1GB"}})
	if disk.OK() || !strings.Contains(disk.Err, "docker pull alpine") {
		t.Errorf("disk without alpine: %+v", disk)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{"16.4GB (51%)": 16_400_000_000, "512kB": 512_000, "0B": 0, "1.5MB": 1_500_000, "junk": 0, "N/A": 0}
	for in, want := range cases {
		if got := ParseSize(in); got != want {
			t.Errorf("ParseSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseDF(t *testing.T) {
	if _, ok := ParseDF("df: /: No such file"); ok {
		t.Error("an error message isn't a disk")
	}
	d, ok := ParseDF("Filesystem 1-blocks Used Available Capacity Mounted on\noverlay 100 60 40 60% /")
	if !ok || d.Total != 100 || d.Free != 40 || d.Used() != 60 {
		t.Errorf("%+v %v", d, ok)
	}
}

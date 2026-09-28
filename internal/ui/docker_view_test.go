package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"manage-disk/internal/docker/dockertest"
)

const dfBefore = "Filesystem 1-blocks Used Available Capacity Mounted on\noverlay 62671097856 43708727296 18962370560 70% /\n"

// dockerApp is an app whose Docker holds a stopped container from a removed
// worktree (with its nginx image and a volume), an untagged image, stale
// build cache, and the alpine image the disk is measured with.
func dockerApp(t *testing.T, opts Options) (*app, *dockertest.Fake) {
	t.Helper()
	a := testApp(t, opts)
	now := time.Now()
	ago := func(d time.Duration) string { return now.Add(-d).Format(time.RFC3339Nano) }
	fake := dockertest.New(t, map[string]string{
		"system df": fmt.Sprintf(`{
			"Images": [
				{"ID": "sha256:nginx", "Repository": "nginx", "Tag": "1", "Size": "190MB", "SharedSize": "0B", "UniqueSize": "190MB"},
				{"ID": "sha256:dangling", "Repository": "<none>", "Tag": "<none>", "Size": "500MB", "SharedSize": "0B", "UniqueSize": "500MB"},
				{"ID": "sha256:alpine", "Repository": "alpine", "Tag": "latest", "Size": "8.6MB", "SharedSize": "0B", "UniqueSize": "8.6MB"}
			],
			"Containers": [{"ID": "c-web", "Names": "feature-web-1", "State": "exited", "Size": "0B"}],
			"Volumes": [{"Name": "feature_data", "Driver": "local", "Links": "1", "Size": "80MB"}],
			"BuildCache": [{"ID": "r1", "Size": "1GB", "InUse": "false", "Shared": "false", "LastUsedAt": %q}]
		}`, now.Add(-10*24*time.Hour).UTC().Format("2006-01-02 15:04:05.999999999 -0700 MST")),
		"container inspect": fmt.Sprintf(`[{"Id": "c-web", "Name": "/feature-web-1", "Image": "sha256:nginx", "Created": %q,
			"State": {"Status": "exited", "FinishedAt": %q},
			"Config": {"Image": "nginx:1", "Labels": {"com.docker.compose.project": "feature",
				"com.docker.compose.project.working_dir": %q}},
			"Mounts": [{"Type": "volume", "Name": "feature_data"}]}]`,
			ago(72*time.Hour), ago(48*time.Hour), a.home+"/projects/app/.claude/worktrees/feature"),
		"image inspect": fmt.Sprintf(`[
			{"Id": "sha256:nginx", "RepoTags": ["nginx:1"], "RepoDigests": ["nginx@sha256:1"], "Created": %q},
			{"Id": "sha256:dangling", "RepoTags": [], "RepoDigests": [], "Created": %q},
			{"Id": "sha256:alpine", "RepoTags": ["alpine:latest"], "RepoDigests": ["alpine@sha256:2"], "Created": %q}]`,
			ago(60*24*time.Hour), ago(24*time.Hour), ago(24*time.Hour)),
		"volume inspect": `[{"Name": "feature_data", "CreatedAt": "2026-09-01T00:00:00Z", "Labels": {"com.docker.compose.project": "feature"}}]`,
		"run":            dfBefore,
		"rm":             "c-web",
		"rmi":            "Deleted: sha256:x",
		"volume rm":      "feature_data",
		"builder prune":  "Total:\t1GB",
	})
	a.env.Command, a.env.LookPath = fake.Command, fake.LookPath
	drive(t, a, a.auditDocker())
	return a, fake
}

func (a *app) dkSelect(t *testing.T, key string) {
	t.Helper()
	for i, it := range a.dk.list.Items() {
		if it.(dkEntry).r.Key() == key {
			a.dk.list.Select(i)
			return
		}
	}
	t.Fatalf("no %s in the list", key)
}

func TestDockerTabJudgesPicksAndRemoves(t *testing.T) {
	a, fake := dockerApp(t, Options{})
	press(t, a, "6")
	out := a.view()
	for _, want := range []string{"Docker's disk", "17.7 GiB free of 58.4 GiB", "✓ unused", "∅ orphan", "◷ old", "◆ data",
		"feature-web-1", "<untagged>", "Build cache unused for a week", "its compose project ran from ~/projects/app/.claude/worktrees/"} {
		if !strings.Contains(out, want) {
			t.Errorf("docker tab is missing %q:\n%s", want, out)
		}
	}
	const (
		web, nginx, vol = "container:c-web", "image:sha256:nginx", "volume:feature_data"
		dangling, cache = "image:sha256:dangling", "build cache:stale"
	)
	for k, want := range map[string]bool{web: true, nginx: true, dangling: true, cache: true, vol: false} {
		if a.dk.chosen[k] != want {
			t.Errorf("%s picked = %v, want %v", k, a.dk.chosen[k], want)
		}
	}

	// Unpicking the container unpicks the image it pins; picking the image
	// picks the container back.
	a.dkSelect(t, web)
	press(t, a, " ")
	if a.dk.chosen[web] || a.dk.chosen[nginx] {
		t.Errorf("unpicking the container should unpick its image: %v", a.dk.chosen)
	}
	a.dkSelect(t, nginx)
	press(t, a, " ")
	if !a.dk.chosen[web] || !a.dk.chosen[nginx] {
		t.Errorf("picking the image should pick its stopped container: %v", a.dk.chosen)
	}
	if !strings.Contains(a.view(), "about 19.2 GiB free after") { // 17.7 GiB + 190 MB + 500 MB + 1 GB
		t.Errorf("header should project the free space:\n%s", a.dkHeader())
	}

	press(t, a, "c") // looks again, then asks
	if a.dk.mode != dkConfirm || !strings.Contains(a.view(), "Remove 4 resources") {
		t.Fatalf("c should re-check and ask: mode %d\n%s", a.dk.mode, a.view())
	}
	if strings.Contains(a.view(), "gone for good") {
		t.Error("no volume is picked, so no data warning")
	}
	fake.Set(t, "run", "Filesystem 1-blocks Used Available Capacity Mounted on\noverlay 62671097856 41708727296 20962370560 67% /\n")
	press(t, a, "y")
	if a.dk.mode != dkDone {
		t.Fatalf("removal did not finish: mode %d", a.dk.mode)
	}
	// Containers first, the build cache last, and nothing forced.
	want := []string{"rm c-web", "rmi sha256:dangling", "rmi nginx:1", "builder prune -f --filter until=168h"}
	if got := fake.Changes(); !slices.Equal(got, want) {
		t.Errorf("changes = %v\nwant      %v", got, want)
	}
	if out := a.view(); !strings.Contains(out, "Docker's disk 17.7 GiB → 19.5 GiB free") {
		t.Errorf("done view should show Docker's disk before and after:\n%s", out)
	}
	press(t, a, "enter")
	if a.dk.mode != dkList {
		t.Errorf("after the summary the list is back: mode %d", a.dk.mode)
	}
}

func TestDockerVolumesAreOnlyEverPickedByHandAndWarned(t *testing.T) {
	a, fake := dockerApp(t, Options{})
	press(t, a, "6", "n") // nothing picked
	a.dkSelect(t, "volume:feature_data")
	press(t, a, " ")
	if !a.dk.chosen["volume:feature_data"] || !a.dk.chosen["container:c-web"] {
		t.Fatalf("picking a volume picks the stopped container mounting it: %v", a.dk.chosen)
	}
	press(t, a, "c")
	if out := a.view(); !strings.Contains(out, "1 volume: the data in them is gone for good") {
		t.Errorf("confirm should warn about the volume's data:\n%s", out)
	}
	press(t, a, "n")
	if a.dk.mode != dkList || len(fake.Changes()) > 0 {
		t.Errorf("cancel changes nothing: mode %d, changes %v", a.dk.mode, fake.Changes())
	}
	press(t, a, "a") // everything verified: never a volume
	if !a.dk.chosen["volume:feature_data"] {
		t.Error("a adds to picks, it doesn't drop the volume you picked")
	}
	press(t, a, "n", "a")
	if a.dk.chosen["volume:feature_data"] {
		t.Error("a must never pick a volume")
	}
}

func TestDockerDryRunRemovesNothing(t *testing.T) {
	a, fake := dockerApp(t, Options{DryRun: true})
	press(t, a, "6", "c", "y")
	if changes := fake.Changes(); len(changes) > 0 {
		t.Errorf("a dry run ran %v", changes)
	}
	if out := a.view(); !strings.Contains(out, "Dry run finished") {
		t.Errorf("done view:\n%s", out)
	}
}

func TestDockerNotRunning(t *testing.T) {
	a := testApp(t, Options{})
	fake := dockertest.New(t, map[string]string{"system df": "!Cannot connect to the Docker daemon. Is the docker daemon running?"})
	a.env.Command, a.env.LookPath = fake.Command, fake.LookPath
	drive(t, a, a.auditDocker())
	press(t, a, "6")
	if out := a.view(); !strings.Contains(out, "Docker isn't running. Start Docker Desktop, then press r.") {
		t.Errorf("view:\n%s", out)
	}
	press(t, a, "c")
	if len(fake.Changes()) > 0 {
		t.Error("nothing to remove when Docker is down")
	}
}

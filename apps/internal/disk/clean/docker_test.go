package clean

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lechero/dotfiles/apps/internal/disk/docker"
	"github.com/lechero/dotfiles/apps/internal/disk/docker/dockertest"
)

func TestComposeProjectsAreNamedLikeComposeNamesThem(t *testing.T) {
	env := testEnv(t, nil)
	p := env.Home + "/projects"
	put(t, p+"/Shop.App/compose.yaml", 10) // the folder's name, as compose normalizes it
	writeFile(t, p+"/named/docker-compose.yml", "name: \"billing\"\nservices:\n  db:\n    image: x\n")
	put(t, p+"/envnamed/compose.yml", 10)
	writeFile(t, p+"/envnamed/.env", "FOO=1\nCOMPOSE_PROJECT_NAME=crm\n")
	put(t, p+"/repo/.claude/worktrees/feature-x/compose.yaml", 10) // a worktree, four levels down
	put(t, p+"/repo/node_modules/pkg/compose.yaml", 10)            // never looked at

	got := composeProjects(context.Background(), p)
	want := map[string][]string{
		"shopapp":   {p + "/Shop.App"},
		"billing":   {p + "/named"},
		"crm":       {p + "/envnamed"},
		"feature-x": {p + "/repo/.claude/worktrees/feature-x"},
	}
	if len(got) != len(want) {
		t.Errorf("projects = %v", got)
	}
	for name, dirs := range want {
		if !slices.Equal(got[name], dirs) {
			t.Errorf("%s: %v, want %v", name, got[name], dirs)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	put(t, path, 0)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A volume outlives its containers; where they ran from is remembered, so the
// volume can still say its folder is gone once they are.
func TestAuditDockerRemembersWhereProjectsRan(t *testing.T) {
	env := testEnv(t, nil)
	gone := env.Home + "/projects/app/.claude/worktrees/feature" // never created: removed long ago
	withContainer := `{"Images": [], "BuildCache": [],
		"Containers": [{"ID": "c1", "Names": "feature-db-1", "State": "exited"}],
		"Volumes": [{"Name": "feature_db", "Driver": "local", "Links": "1", "Size": "80MB"}]}`
	fake := dockertest.New(t, map[string]string{
		"system df": withContainer,
		"container inspect": fmt.Sprintf(`[{"Id": "c1", "Name": "/feature-db-1", "Image": "sha256:pg", "Created": %q,
			"State": {"Status": "exited", "FinishedAt": %q},
			"Config": {"Labels": {"com.docker.compose.project": "feature", "com.docker.compose.project.working_dir": %q}},
			"Mounts": [{"Type": "volume", "Name": "feature_db"}]}]`,
			fixedNow.Add(-48*time.Hour).Format(time.RFC3339), fixedNow.Add(-24*time.Hour).Format(time.RFC3339), gone),
		"volume inspect": `[{"Name": "feature_db", "CreatedAt": "2026-09-01T00:00:00Z", "Labels": {"com.docker.compose.project": "feature"}}]`,
	})
	env.Command, env.LookPath = fake.Command, fake.LookPath

	rep := AuditDocker(context.Background(), env)
	if c := findResource(rep, docker.Container, "c1"); c.Verdict != docker.Orphan {
		t.Fatalf("container from a removed worktree: %s (%v)", c.Verdict, c.Reasons)
	}

	// The container is removed; only the volume is left.
	fake.Set(t, "system df", `{"Images": [], "Containers": [], "BuildCache": [],
		"Volumes": [{"Name": "feature_db", "Driver": "local", "Links": "0", "Size": "80MB"}]}`)
	rep = AuditDocker(context.Background(), env)
	v := findResource(rep, docker.Volume, "feature_db")
	if v.Verdict != docker.Data || !v.ProjectGone || !strings.Contains(strings.Join(v.Reasons, "; "), "which is gone") {
		t.Errorf("the volume should still know its folder is gone: %s %v", v.Verdict, v.Reasons)
	}
	if rep.Disk.OK() || !strings.Contains(rep.Disk.Err, "alpine") {
		t.Errorf("without alpine or busybox the disk isn't measured, and says why: %+v", rep.Disk)
	}
	if changes := fake.Changes(); len(changes) > 0 {
		t.Errorf("an audit changed things: %v", changes)
	}
}

func findResource(rep docker.Report, kind docker.Kind, id string) docker.Resource {
	for _, r := range rep.Resources {
		if r.Kind == kind && r.ID == id {
			return r
		}
	}
	return docker.Resource{}
}

func TestAuditDockerWithoutDocker(t *testing.T) {
	env := testEnv(t, nil) // LookPath finds nothing
	if rep := AuditDocker(context.Background(), env); !rep.Missing {
		t.Errorf("no docker command: %+v", rep)
	}
	fake := dockertest.New(t, map[string]string{
		"system df": "!Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?",
	})
	env.Command, env.LookPath = fake.Command, fake.LookPath
	if rep := AuditDocker(context.Background(), env); !rep.Down {
		t.Errorf("engine down: %+v", rep)
	}
}

func TestDockerTaskRunsDockersOwnCommandAndReportsItsError(t *testing.T) {
	env := testEnv(t, nil)
	fake := dockertest.New(t, map[string]string{
		"rmi": "!Error response from daemon: conflict: unable to remove repository reference \"nginx:1\" (must force) - container c9 is using its referenced image",
	})
	env.Command, env.LookPath = fake.Command, fake.LookPath
	task := DockerTask()
	var log []string
	logf := func(s string) { log = append(log, s) }

	vol := docker.Resource{Kind: docker.Volume, ID: "old_data", Name: "old_data", Verdict: docker.Data}
	if err := task.Run(context.Background(), env, DockerItem(vol), logf); err != nil {
		t.Fatal(err)
	}
	img := docker.Resource{Kind: docker.Image, ID: "sha256:n", Tags: []string{"nginx:1"}, Verdict: docker.Old}
	err := task.Run(context.Background(), env, DockerItem(img), logf)
	if err == nil || !strings.Contains(err.Error(), "container c9 is using its referenced image") {
		t.Errorf("docker's refusal should be the error: %v", err)
	}
	inUse := docker.Resource{Kind: docker.Container, ID: "c1", Verdict: docker.InUse, Reasons: []string{"running"}}
	if err := task.Run(context.Background(), env, DockerItem(inUse), logf); err == nil {
		t.Error("something in use must never be removed")
	}
	if got := fake.Calls(); !slices.Equal(got, []string{"volume rm old_data", "rmi nginx:1"}) {
		t.Errorf("calls = %v", got)
	}
	if !slices.Contains(log, "$ docker volume rm old_data") {
		t.Errorf("the log should show each command: %v", log)
	}
}

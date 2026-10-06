package clean

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/disk/docker"
)

// dockerQuiet keeps the docker command from adding "What's next" hints.
var dockerQuiet = []string{"DOCKER_CLI_HINTS=false"}

// docker runs the docker command for reading: stdout only, so a warning can't
// corrupt a parse, with the first line of stderr as the error.
func (e *Env) docker(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := e.Command(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), dockerQuiet...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			err = errors.New(headLine(string(ee.Stderr)))
		}
	}
	return strings.TrimSpace(string(out)), err
}

// AuditDocker judges everything Docker keeps, and measures the disk it keeps
// it on. It removes nothing.
func AuditDocker(ctx context.Context, env *Env) docker.Report {
	if _, err := env.LookPath("docker"); err != nil {
		return docker.Report{Missing: true, Err: "Docker isn't installed", At: env.Now()}
	}
	procs := env.Procs(ctx)
	seen := loadComposeDirs(env.StateDir)
	var once sync.Once
	var named map[string][]string
	rep := docker.Audit(ctx, docker.Deps{
		Home:   env.Home,
		Now:    env.Now(),
		Docker: env.docker,
		Projects: func(name string) []string {
			once.Do(func() { named = composeProjects(ctx, env.Home+"/projects") })
			return append(append([]string(nil), seen[name]...), named[name]...)
		},
		Building: func() string {
			if b := procs.Matching("docker build", "buildx build", "buildx bake", "docker compose build", "docker bake"); len(b) > 0 {
				return fmt.Sprintf("%s (pid %d)", b[0].Name(), b[0].PID)
			}
			return ""
		},
	})
	if rep.Err == "" {
		saveComposeDirs(env.StateDir, seen, rep.ProjectDirs)
	}
	return rep
}

// The folders compose projects ran from, remembered in compose-projects.json
// next to the run history: a volume outlives its containers, and with them
// the only record of where it came from.
func loadComposeDirs(dir string) map[string][]string {
	m := map[string][]string{}
	if b, err := os.ReadFile(filepath.Join(dir, "compose-projects.json")); err == nil {
		// A damaged file is no record at all, rather than part of one.
		if json.Unmarshal(b, &m) != nil {
			return map[string][]string{}
		}
	}
	return m
}

func saveComposeDirs(dir string, seen, now map[string][]string) {
	changed := false
	for project, dirs := range now {
		for _, d := range dirs {
			if !slices.Contains(seen[project], d) {
				seen[project] = append(seen[project], d)
				changed = true
			}
		}
		if n := len(seen[project]); n > 20 { // the newest folders are the telling ones
			seen[project] = seen[project][n-20:]
		}
	}
	if !changed || dir == "" || os.MkdirAll(dir, 0o755) != nil {
		return
	}
	b, _ := json.Marshal(seen)
	// Best effort: without the file, compose projects just go unnamed.
	_ = os.WriteFile(filepath.Join(dir, "compose-projects.json"), b, 0o644)
}

var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// composeProjects maps compose project names to the folders under root that
// start them, named the way compose names them: the compose file's top-level
// name:, else COMPOSE_PROJECT_NAME from the folder's .env, else the folder's
// own name. Worktrees count: .claude/worktrees/<name> is four levels down.
func composeProjects(ctx context.Context, root string) map[string][]string {
	out := map[string][]string{}
	prune := map[string]bool{"node_modules": true, ".git": true, "vendor": true, ".next": true, "target": true, "dist": true}
	findDirs(ctx, root, 4, prune, func(dir, _ string) bool {
		for _, f := range composeFiles {
			if fileExists(filepath.Join(dir, f)) {
				name := composeName(dir, f)
				out[name] = append(out[name], dir)
				break
			}
		}
		return false // keep walking: a folder may hold more compose projects below it
	})
	return out
}

var topLevelName = regexp.MustCompile(`^name:\s*["']?([^"'#\s]+)`)

func composeName(dir, file string) string {
	if name := firstMatch(filepath.Join(dir, file), topLevelName); name != "" && !strings.Contains(name, "$") {
		return normalizeProject(name)
	}
	if name := firstMatch(filepath.Join(dir, ".env"), envProjectName); name != "" {
		return normalizeProject(name)
	}
	return normalizeProject(filepath.Base(dir))
}

var envProjectName = regexp.MustCompile(`^COMPOSE_PROJECT_NAME\s*=\s*["']?([^"'#\s]+)`)

func firstMatch(path string, re *regexp.Regexp) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }() // read-only
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := re.FindStringSubmatch(sc.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

// normalizeProject names a project the way compose does: lower case, only
// letters, digits, dashes and underscores, starting with a letter or digit.
func normalizeProject(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.TrimLeft(b.String(), "-_")
}

// DockerItem wraps a resource for a removal run.
func DockerItem(r docker.Resource) Item {
	return Item{
		Paths:    []string{r.Key()},
		Label:    r.Kind.String() + " " + r.Name,
		Size:     r.Size,
		Selected: true,
		Note:     r.Verdict.String() + ": " + strings.Join(r.Reasons, "; "),
		Data:     r,
	}
}

// DockerTask removes resources with docker's own commands, none forced:
// docker rm refuses a running container, and rmi and volume rm refuse what a
// container still uses. So something that came back into use since the check
// stays.
func DockerTask() *Task {
	return &Task{
		ID:    "docker-resources",
		Title: "Docker",
		Tier:  Tier2,
		About: "Removes the chosen containers, images, volumes and build cache with docker rm, rmi, " +
			"volume rm and builder prune. Nothing is forced.",
		Cost: "A removed volume's data is gone for good; images come back with a pull or a rebuild.",
		Run: func(ctx context.Context, env *Env, it Item, log func(string)) error {
			r, ok := it.Data.(docker.Resource)
			if !ok {
				return errors.New("not a docker resource")
			}
			if !r.Verdict.Removable() {
				return errors.New("kept: " + strings.Join(r.Reasons, "; "))
			}
			args := r.Command()
			log("$ docker " + strings.Join(args, " "))
			cmd := env.Command(ctx, "docker", args...)
			cmd.Env = append(os.Environ(), dockerQuiet...)
			out, err := cmd.CombinedOutput()
			for _, line := range strings.Split(string(out), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					log(line)
				}
			}
			if err != nil && strings.TrimSpace(string(out)) != "" {
				return fmt.Errorf("docker %s: %s", args[0], lastLine(string(out)))
			}
			return err
		},
	}
}

// lastLine is the last line a command printed: docker's error comes last.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

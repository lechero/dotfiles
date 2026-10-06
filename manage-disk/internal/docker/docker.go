// Package docker judges, from evidence, what Docker keeps on its disk —
// images, containers, volumes and build cache — and what removing each costs.
//
// Docker Desktop keeps all of it on one Linux filesystem inside its VM, with a
// size limit of its own: builds fail when that fills up, however much room the
// Mac has left. So the number that matters is the free space in there.
//
// Removing a container keeps its volumes. Removing an image loses nothing a
// pull or a rebuild can't bring back. A volume's data exists nowhere else, so
// volumes are never picked for you.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"manage-disk/internal/human"
)

// Kind is what sort of thing a resource is, in the order a removal goes:
// containers pin the images they run and the volumes they mount, and images
// built here keep their layers in the build cache until that is pruned too.
type Kind int

const (
	Container Kind = iota
	Image
	Volume
	BuildCache
)

func (k Kind) String() string {
	return [...]string{"container", "image", "volume", "build cache"}[k]
}

// Verdict is what the evidence says about removing a resource, safest last.
type Verdict int

const (
	InUse  Verdict = iota // a running container, or what one runs from or mounts
	Data                  // a volume: what is in it exists nowhere else
	Review                // removable, but it looks wanted, or only a rebuild brings it back
	Old                   // nothing has used it for a while, and it can come back
	Orphan                // its compose project's folder is gone
	Unused                // nothing refers to it: untagged images, stale build cache, empty volumes
)

func (v Verdict) String() string {
	return [...]string{"in use", "data", "review", "old", "orphan", "unused"}[v]
}

// Removable reports whether the resource may be removed at all.
func (v Verdict) Removable() bool { return v != InUse }

// Preselect reports whether the evidence is strong enough to pick it for you.
func (v Verdict) Preselect() bool { return v >= Old }

// Ref names a container that runs an image or mounts a volume.
type Ref struct {
	ID, Name string
	Running  bool
}

// Resource is one container, image, volume or slice of build cache, and the
// evidence about it.
type Resource struct {
	Kind Kind
	ID   string // container or image id, volume name, or a build cache slice
	Name string // container name, first repo:tag, volume name

	Size    int64 // what removing it should free, as Docker reckons it
	Created time.Time

	// Containers.
	State   string    // running, exited, created…
	Stopped time.Time // when it last stopped (or was made, if it never ran)
	Image   string    // the image it runs, as named when it was made
	Written int64     // bytes in its own layer, which go with it
	Mounts  []string  // named volumes it mounts; they stay when it goes

	// Images.
	Tags       []string // every repo:tag; none when dangling
	InRegistry bool     // it has a registry digest, so docker pull brings it back
	BuiltHere  bool     // tagged on this machine, by a build or docker tag
	Shared     int64    // bytes it shares with other images, which stay

	// Volumes.
	Anonymous bool

	// Build cache.
	Records  int
	LastUsed time.Time
	Until    time.Duration // prune only cache unused for this long; 0 prunes it all
	Before   time.Time     // or prune only what was last used before this (a plan's partial prune)

	// Compose.
	Project     string
	Service     string
	ProjectDir  string // a folder that runs the project, else the last one it ran from
	ProjectGone bool   // ProjectDir is gone: nothing known runs the project any more

	Users []Ref // containers that run this image or mount this volume
	// Implies are the resources that must go first (by Key): the stopped
	// containers pinning an image or volume, the older cache under a newer slice.
	Implies []string

	Verdict Verdict
	Reasons []string

	imageID    string     // containers: the id of the image they run
	empty      bool       // volumes: Docker measured nothing in it
	cacheInUse bool       // build cache: a build holds some of it
	uses       []cacheUse // build cache: each record's last use and size
}

type cacheUse struct {
	used time.Time
	size int64
}

// Cut is what pruning the build cache last used before t frees, and how many
// records that is. Records never used count as oldest.
func (r Resource) Cut(t time.Time) (size int64, records int) {
	for _, u := range r.uses {
		if u.used.Before(t) {
			size += u.size
			records++
		}
	}
	return size, records
}

// Key identifies a resource from one audit to the next.
func (r Resource) Key() string { return r.Kind.String() + ":" + r.ID }

// Command is the docker command that removes it. None of them force: docker
// refuses a running container, and an image or volume one still uses.
func (r Resource) Command() []string {
	switch r.Kind {
	case Container:
		return []string{"rm", r.ID}
	case Image:
		if len(r.Tags) > 0 { // by name: an id with several tags would need -f
			return append([]string{"rmi"}, r.Tags...)
		}
		return []string{"rmi", r.ID}
	case Volume:
		return []string{"volume", "rm", r.ID}
	}
	// -f only skips prune's own y/N prompt; cache a build uses is never pruned.
	switch {
	case r.Until > 0:
		return []string{"builder", "prune", "-f", "--filter", fmt.Sprintf("until=%dh", int(r.Until.Hours()))}
	case !r.Before.IsZero():
		// until is a duration back from the daemon's now, so it is worked out
		// when the command runs. Rounding it up puts the cut no later than
		// Before: never more pruned than planned.
		secs := max(1, int64(math.Ceil(time.Since(r.Before).Seconds())))
		return []string{"builder", "prune", "-f", "--filter", fmt.Sprintf("until=%ds", secs)}
	}
	return []string{"builder", "prune", "-f"}
}

// Disk is the filesystem Docker keeps everything on: inside the VM, for
// Docker Desktop. The Mac's own free space doesn't limit a build; this does.
type Disk struct {
	Total, Free int64
	Err         string // why it couldn't be measured
}

// OK reports whether the disk was measured.
func (d Disk) OK() bool { return d.Total > 0 }

// Used is everything but the free space.
func (d Disk) Used() int64 { return max(0, d.Total-d.Free) }

// Report is one look at Docker.
type Report struct {
	Missing bool   // the docker command isn't installed
	Down    bool   // the engine didn't answer
	Err     string // why there is nothing to show

	Disk        Disk
	Resources   []Resource // most removable first, then largest
	SharedCache int64      // build cache images share; it goes when they do
	// ProjectDirs are the folders each compose project's containers ran from,
	// worth remembering: once the containers go, their volumes still say where
	// they came from.
	ProjectDirs map[string][]string
	At          time.Time
}

// Deps is how the audit reaches Docker and the folders that run it.
type Deps struct {
	Home string
	Now  time.Time
	// OldAfter is how long unused counts as old (default 30 days).
	OldAfter time.Duration
	// CacheAfter splits build cache into stale and recent (default a week).
	CacheAfter time.Duration
	// Docker runs the docker command and returns what it printed on stdout.
	Docker func(ctx context.Context, args ...string) (string, error)
	// DirExists reports whether a folder is still there (default os.Stat).
	DirExists func(path string) bool
	// Projects lists folders known to start a compose project (remembered from
	// earlier looks, or found by name), for volumes whose containers are gone;
	// nil skips it.
	Projects func(name string) []string
	// Building names a build in progress, or returns "".
	Building func() string
}

// Audit looks at everything Docker keeps and judges each resource. It changes
// nothing, though it runs a throwaway container to measure Docker's disk.
func Audit(ctx context.Context, d Deps) Report {
	if d.OldAfter == 0 {
		d.OldAfter = 30 * 24 * time.Hour
	}
	if d.CacheAfter == 0 {
		d.CacheAfter = 7 * 24 * time.Hour
	}
	if d.DirExists == nil {
		d.DirExists = func(p string) bool {
			fi, err := os.Stat(p)
			return err == nil && fi.IsDir()
		}
	}
	rep := Report{At: d.Now}
	out, err := d.Docker(ctx, "system", "df", "-v", "--format", "{{json .}}")
	if err != nil {
		rep.Err = err.Error()
		rep.Down = engineDown(rep.Err)
		if rep.Down {
			rep.Err = "Docker isn't running"
		}
		return rep
	}
	var df systemDF
	if err := json.Unmarshal([]byte(out), &df); err != nil {
		rep.Err = "docker system df: " + err.Error()
		return rep
	}

	// Inspect everything, and measure the disk, all at once.
	var (
		wg         sync.WaitGroup
		containers []containerJSON
		images     []imageJSON
		volumes    []volumeJSON
	)
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	if ids := column(df.Containers, func(c dfContainer) string { return c.ID }); len(ids) > 0 {
		run(func() { inspect(ctx, d, &containers, append([]string{"container", "inspect", "--size"}, ids...)) })
	}
	if ids := column(df.Images, func(i dfImage) string { return i.ID }); len(ids) > 0 {
		run(func() { inspect(ctx, d, &images, append([]string{"image", "inspect"}, ids...)) })
	}
	if names := column(df.Volumes, func(v dfVolume) string { return v.Name }); len(names) > 0 {
		run(func() { inspect(ctx, d, &volumes, append([]string{"volume", "inspect"}, names...)) })
	}
	run(func() { rep.Disk = measureDisk(ctx, d, df.Images) })
	wg.Wait()

	a := &audit{d: d, projectDirs: map[string][]string{}}
	a.containers(df.Containers, containers)
	a.images(df.Images, images)
	a.volumes(df.Volumes, volumes)
	rep.SharedCache = a.buildCache(df.BuildCache)
	rep.ProjectDirs = a.projectDirs
	for _, r := range a.all {
		judge(r, a, d)
	}
	for _, r := range a.all {
		rep.Resources = append(rep.Resources, *r)
	}
	sort.SliceStable(rep.Resources, func(i, j int) bool {
		x, y := rep.Resources[i], rep.Resources[j]
		if x.Verdict != y.Verdict {
			return x.Verdict > y.Verdict
		}
		if x.Size != y.Size {
			return x.Size > y.Size
		}
		return x.Name < y.Name
	})
	return rep
}

// engineDown recognises the ways the docker command says nothing answered.
func engineDown(msg string) bool {
	for _, s := range []string{"Cannot connect to the Docker daemon", "Is the docker daemon running", "error during connect", "docker.sock: connect"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// What `docker system df -v --format '{{json .}}'` prints: every size is
// Docker's decimal text ("4.69GB").
type systemDF struct {
	Images     []dfImage
	Containers []dfContainer
	Volumes    []dfVolume
	BuildCache []dfCache
}

type dfImage struct {
	ID, Repository, Tag, Size, SharedSize, UniqueSize string
}

type dfContainer struct {
	ID, Names, Image, State, Size string
}

type dfVolume struct {
	Name, Driver, Links, Size string
}

type dfCache struct {
	ID, CacheType, Size, InUse, Shared, LastUsedAt string
}

type containerJSON struct {
	ID      string `json:"Id"`
	Name    string
	Image   string // the image id
	Created string
	SizeRw  int64
	State   struct {
		Status                      string
		Running, Paused, Restarting bool
		StartedAt, FinishedAt       string
	}
	Config struct {
		Image  string
		Labels map[string]string
	}
	Mounts []struct{ Type, Name, Source string }
}

type imageJSON struct {
	ID          string `json:"Id"`
	RepoTags    []string
	RepoDigests []string
	Created     string
	Metadata    struct{ LastTagTime string }
}

type volumeJSON struct {
	Name      string
	CreatedAt string
	Labels    map[string]string
}

func column[T any](xs []T, f func(T) string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if v := f(x); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// inspect decodes `docker … inspect` into out. A resource that vanished in
// between makes docker complain but still print the others, so a partial
// answer is kept.
func inspect[T any](ctx context.Context, d Deps, out *[]T, args []string) {
	s, _ := d.Docker(ctx, args...)
	json.Unmarshal([]byte(s), out)
}

// measureDisk runs df in a throwaway container: on overlay storage a
// container's / reports the filesystem Docker keeps everything on, so no
// privileges are needed. Only an alpine or busybox image already here is used;
// nothing is pulled.
func measureDisk(ctx context.Context, d Deps, images []dfImage) Disk {
	img, best := "", int64(math.MaxInt64)
	for _, i := range images {
		repo := i.Repository[strings.LastIndexByte(i.Repository, '/')+1:]
		if s := ParseSize(i.Size); (repo == "alpine" || repo == "busybox") && s < best {
			img, best = i.ID, s
		}
	}
	if img == "" {
		return Disk{Err: "measuring it needs a local alpine or busybox image: docker pull alpine"}
	}
	out, err := d.Docker(ctx, "run", "--rm", "--pull", "never", "--network", "none", "--log-driver", "none",
		"--entrypoint", "df", img, "-PB1", "/")
	if err != nil {
		return Disk{Err: "df in a container failed: " + err.Error()}
	}
	disk, ok := ParseDF(out)
	if !ok {
		return Disk{Err: "couldn't read df's answer"}
	}
	return disk
}

// ParseDF reads the one filesystem `df -PB1` reports.
func ParseDF(out string) (Disk, bool) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return Disk{}, false
	}
	f := strings.Fields(lines[len(lines)-1])
	if len(f) < 4 {
		return Disk{}, false
	}
	total, err1 := strconv.ParseInt(f[1], 10, 64)
	free, err2 := strconv.ParseInt(f[3], 10, 64)
	if err1 != nil || err2 != nil || total <= 0 {
		return Disk{}, false
	}
	return Disk{Total: total, Free: free}, true
}

// ParseSize reads Docker's decimal sizes: "16.4GB (51%)", "512kB", "0B".
func ParseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	units := []struct {
		suffix string
		mult   float64
	}{{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"KB", 1e3}, {"B", 1}}
	for _, u := range units {
		if num, ok := strings.CutSuffix(s, u.suffix); ok {
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return int64(math.Round(f * u.mult))
		}
	}
	return 0
}

// parseTime reads the times docker prints: RFC 3339 in inspect, Go's own
// format in system df. Docker's "never" (year 1) reads as zero.
func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999 -0700 MST"} {
		if t, err := time.Parse(layout, s); err == nil && t.Year() > 1 {
			return t
		}
	}
	return time.Time{}
}

type audit struct {
	d           Deps
	all         []*Resource
	byKey       map[string]*Resource
	projectDirs map[string][]string // compose project → folders its containers ran from
}

func (a *audit) add(r *Resource) {
	if a.byKey == nil {
		a.byKey = map[string]*Resource{}
	}
	a.all = append(a.all, r)
	a.byKey[r.Key()] = r
}

func (a *audit) containers(rows []dfContainer, details []containerJSON) {
	byID := map[string]containerJSON{}
	for _, c := range details {
		byID[c.ID] = c
	}
	for _, row := range rows {
		c, ok := byID[row.ID]
		if !ok {
			continue // gone between the two looks
		}
		r := &Resource{
			Kind: Container, ID: c.ID, Name: strings.TrimPrefix(c.Name, "/"),
			State: c.State.Status, Image: c.Config.Image, Written: c.SizeRw, Size: c.SizeRw,
			Created: parseTime(c.Created),
		}
		r.Stopped = parseTime(c.State.FinishedAt)
		if r.Stopped.IsZero() {
			r.Stopped = r.Created
		}
		if c.State.Running || c.State.Paused || c.State.Restarting {
			r.State = map[bool]string{true: "paused", false: "running"}[c.State.Paused]
			if c.State.Restarting {
				r.State = "restarting"
			}
		}
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				r.Mounts = append(r.Mounts, m.Name)
			}
		}
		labels := c.Config.Labels
		r.Project, r.Service = labels["com.docker.compose.project"], labels["com.docker.compose.service"]
		if dir := labels["com.docker.compose.project.working_dir"]; dir != "" {
			r.ProjectDir = dir
			r.ProjectGone = !a.d.DirExists(dir)
			if r.Project != "" {
				a.projectDirs[r.Project] = appendNew(a.projectDirs[r.Project], dir)
			}
		}
		r.imageID = c.Image
		a.add(r)
	}
}

func (a *audit) images(rows []dfImage, details []imageJSON) {
	byID := map[string]imageJSON{}
	for _, i := range details {
		byID[i.ID] = i
	}
	for _, row := range rows {
		i, ok := byID[row.ID]
		if !ok {
			continue
		}
		r := &Resource{
			Kind: Image, ID: i.ID, Tags: i.RepoTags,
			Size: ParseSize(row.UniqueSize), Shared: ParseSize(row.SharedSize),
			Created: parseTime(i.Created), InRegistry: len(i.RepoDigests) > 0,
			BuiltHere: !parseTime(i.Metadata.LastTagTime).IsZero(),
		}
		r.Name = "<untagged> " + shortID(i.ID)
		if len(i.RepoTags) > 0 {
			r.Name = i.RepoTags[0]
		}
		for _, c := range a.all {
			if c.Kind == Container && c.imageID == i.ID {
				r.Users = append(r.Users, Ref{ID: c.ID, Name: c.Name, Running: c.running()})
			}
		}
		a.add(r)
	}
}

func (a *audit) volumes(rows []dfVolume, details []volumeJSON) {
	byName := map[string]volumeJSON{}
	for _, v := range details {
		byName[v.Name] = v
	}
	for _, row := range rows {
		v, ok := byName[row.Name]
		if !ok {
			continue
		}
		r := &Resource{
			Kind: Volume, ID: v.Name, Name: v.Name, Size: ParseSize(row.Size),
			Created: parseTime(v.CreatedAt), Project: v.Labels["com.docker.compose.project"],
		}
		_, anon := v.Labels["com.docker.volume.anonymous"]
		r.Anonymous = anon || hexName.MatchString(v.Name)
		r.empty = row.Driver == "local" && strings.TrimSpace(row.Size) == "0B"
		for _, c := range a.all {
			if c.Kind == Container && slices.Contains(c.Mounts, v.Name) {
				r.Users = append(r.Users, Ref{ID: c.ID, Name: c.Name, Running: c.running()})
			}
		}
		if r.Project != "" {
			dirs := a.projectDirs[r.Project]
			if a.d.Projects != nil {
				for _, dir := range a.d.Projects(r.Project) {
					dirs = appendNew(dirs, dir)
				}
			}
			for _, dir := range dirs {
				if a.d.DirExists(dir) {
					r.ProjectDir = dir
					break
				}
			}
			if r.ProjectDir == "" && len(dirs) > 0 {
				r.ProjectDir, r.ProjectGone = dirs[0], true
			}
		}
		a.add(r)
	}
}

var hexName = regexp.MustCompile(`^[0-9a-f]{64}$`)

// buildCache splits the cache no image shares into what no build touched for
// CacheAfter and the rest, and returns what images share.
func (a *audit) buildCache(rows []dfCache) (shared int64) {
	stale := &Resource{Kind: BuildCache, ID: "stale", Until: a.d.CacheAfter,
		Name: "Build cache unused for " + days(a.d.CacheAfter)}
	recent := &Resource{Kind: BuildCache, ID: "recent", Name: "Build cache used within " + days(a.d.CacheAfter)}
	for _, row := range rows {
		size := ParseSize(row.Size)
		if row.Shared == "true" {
			shared += size
			continue
		}
		r := recent
		used := parseTime(row.LastUsedAt)
		if !used.IsZero() && a.d.Now.Sub(used) >= a.d.CacheAfter {
			r = stale
		}
		if row.InUse == "true" {
			r.cacheInUse = true
		}
		r.Size += size
		r.Records++
		r.uses = append(r.uses, cacheUse{used: used, size: size})
		if used.After(r.LastUsed) {
			r.LastUsed = used
		}
	}
	if stale.Records > 0 {
		a.add(stale)
	}
	if recent.Records > 0 {
		if stale.Records > 0 {
			recent.Implies = []string{stale.Key()} // pruning it all takes the stale part too
		}
		a.add(recent)
	}
	return shared
}

// judge turns the evidence into a verdict and the reasons for it.
func judge(r *Resource, a *audit, d Deps) {
	switch r.Kind {
	case Container:
		judgeContainer(r, d)
	case Image:
		judgeImage(r, a, d)
	case Volume:
		judgeVolume(r, a, d)
	case BuildCache:
		judgeCache(r, d)
	}
}

func judgeContainer(r *Resource, d Deps) {
	if r.running() {
		r.Verdict, r.Reasons = InUse, []string{r.State}
		return
	}
	stopped := "stopped " + human.Ago(d.Now, r.Stopped)
	if r.State == "created" {
		stopped = "made " + human.Ago(d.Now, r.Created) + ", never started"
	}
	switch {
	case r.ProjectGone:
		r.Verdict, r.Reasons = Orphan, []string{"its compose project ran from " + tilde(d.Home, r.ProjectDir) + ", which is gone", stopped}
	case d.Now.Sub(r.Stopped) >= d.OldAfter:
		r.Verdict, r.Reasons = Old, []string{stopped}
	default:
		r.Verdict, r.Reasons = Review, []string{stopped + ": you may start it again"}
		if r.ProjectDir != "" {
			r.Reasons = append(r.Reasons, "docker compose up makes it again")
		}
	}
	if r.Written >= 10<<20 { // more than logs and pid files: something it made lives in there
		r.Reasons = append([]string{human.Bytes(r.Written) + " written inside it go with it"}, r.Reasons...)
		r.Verdict = min(r.Verdict, Review)
	}
	if len(r.Mounts) > 0 {
		r.Reasons = append(r.Reasons, "its volumes stay")
	}
}

func judgeImage(r *Resource, a *audit, d Deps) {
	if u, ok := firstRunning(r.Users); ok {
		r.Verdict, r.Reasons = InUse, []string{"running container " + u.Name + " uses it"}
		return
	}
	age := d.Now.Sub(r.Created)
	switch {
	case len(r.Tags) == 0:
		r.Verdict, r.Reasons = Unused, []string{"untagged: a newer build or pull took its name"}
	case !r.InRegistry:
		r.Verdict, r.Reasons = Review, []string{"built here and never pushed: only a rebuild brings it back"}
	case age >= d.OldAfter:
		r.Verdict, r.Reasons = Old, []string{fmt.Sprintf("created %s, and docker pull brings it back", human.Ago(d.Now, r.Created))}
	default:
		r.Verdict, r.Reasons = Review, []string{fmt.Sprintf("created %s: you may run it again soon", human.Ago(d.Now, r.Created)),
			"docker pull brings it back"}
	}
	a.pinnedBy(r)
	if measuring(r) {
		r.Reasons = append(r.Reasons, "manage-disk measures Docker's disk with it")
	}
	if r.BuiltHere && len(r.Tags) > 0 {
		r.Reasons = append(r.Reasons, "the build cache may hold its layers too: they come back once that is pruned as well")
	}
}

func judgeVolume(r *Resource, a *audit, d Deps) {
	if u, ok := firstRunning(r.Users); ok {
		r.Verdict, r.Reasons = InUse, []string{"running container " + u.Name + " mounts it"}
		return
	}
	if r.empty && len(r.Users) == 0 {
		r.Verdict, r.Reasons = Unused, []string{"empty, and no container mounts it"}
		return
	}
	r.Verdict = Data // never picked for you, whatever else is true
	switch {
	case r.Project != "" && r.ProjectGone:
		r.Reasons = append(r.Reasons, "compose project "+r.Project+" ran from "+tilde(d.Home, r.ProjectDir)+", which is gone")
	case r.Project != "" && r.ProjectDir != "":
		r.Reasons = append(r.Reasons, "compose project "+r.Project+", in "+tilde(d.Home, r.ProjectDir))
	case r.Project != "":
		r.Reasons = append(r.Reasons, "compose project "+r.Project+": no folder under ~/projects goes by that name")
	case r.Anonymous:
		r.Reasons = append(r.Reasons, "anonymous: made for a container, not by name")
	default:
		r.Reasons = append(r.Reasons, "made by name, not by compose")
	}
	if len(r.Users) == 0 {
		r.Reasons = append(r.Reasons, "no container mounts it")
	}
	a.pinnedBy(r)
	if !r.Created.IsZero() {
		r.Reasons = append(r.Reasons, "created "+human.Ago(d.Now, r.Created))
	}
}

func judgeCache(r *Resource, d Deps) {
	if d.Building != nil {
		if b := d.Building(); b != "" {
			r.Verdict, r.Reasons = InUse, []string{"a build is running: " + b}
			return
		}
	}
	if r.cacheInUse {
		r.Verdict, r.Reasons = InUse, []string{"a build is using it"}
		return
	}
	used := plural(r.Records, "record") + ", last used " + human.Ago(d.Now, r.LastUsed)
	if r.LastUsed.IsZero() {
		used = plural(r.Records, "record")
	}
	if r.Until > 0 {
		r.Verdict, r.Reasons = Unused, []string{"no build has used it for " + days(r.Until), used}
		return
	}
	r.Verdict, r.Reasons = Review, []string{"recent: the next build of what made it starts colder", used}
	if len(r.Implies) > 0 {
		r.Reasons = append(r.Reasons, "pruning it takes the older cache too")
	}
}

// pinnedBy notes the stopped containers that must go before r can: they are
// removed first, so r's verdict is no safer than theirs. Containers come first
// in a.all, so they are judged by the time this runs.
func (a *audit) pinnedBy(r *Resource) {
	var names []string
	for _, u := range r.Users {
		c := a.byKey[Resource{Kind: Container, ID: u.ID}.Key()]
		if c == nil {
			continue
		}
		r.Implies = append(r.Implies, c.Key())
		r.Verdict = min(r.Verdict, c.Verdict)
		names = append(names, u.Name)
	}
	if len(names) > 0 {
		r.Reasons = append(r.Reasons, "stopped "+plural(len(names), "container")+" "+strings.Join(first(names, 3), ", ")+" must go first")
	}
}

// measuring reports whether r is an image measureDisk would run.
func measuring(r *Resource) bool {
	for _, t := range r.Tags {
		repo, _, _ := strings.Cut(t[strings.LastIndexByte(t, '/')+1:], ":")
		if repo == "alpine" || repo == "busybox" {
			return true
		}
	}
	return false
}

func tilde(home, path string) string {
	if rel, ok := strings.CutPrefix(path, home+"/"); ok && home != "" {
		return "~/" + rel
	}
	return path
}

func (r *Resource) running() bool {
	return r.State == "running" || r.State == "paused" || r.State == "restarting"
}

func firstRunning(refs []Ref) (Ref, bool) {
	for _, u := range refs {
		if u.Running {
			return u, true
		}
	}
	return Ref{}, false
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ShortID is an id the way docker prints it.
func (r Resource) ShortID() string { return shortID(r.ID) }

func days(d time.Duration) string {
	n := int(d.Hours() / 24)
	switch {
	case n == 7:
		return "a week"
	case n == 1:
		return "a day"
	}
	return fmt.Sprintf("%d days", n)
}

func appendNew(xs []string, x string) []string {
	if slices.Contains(xs, x) {
		return xs
	}
	return append(xs, x)
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return append(append([]string(nil), xs[:n]...), fmt.Sprintf("+%d more", len(xs)-n))
	}
	return xs
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

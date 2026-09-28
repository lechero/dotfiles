# manage-disk

Where your disk space goes, and the cleanups that are safe to repeat. It's a
[Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI for macOS.

```bash
task dry        # build and open it with deletion switched off
task run        # build and open it for real
task report     # print the overview and cleanup plan; changes nothing
task worktrees  # list every worktree with a verdict; changes nothing
task docker     # Docker's own disk, and a verdict on everything in it; changes nothing
task install    # copy the binary to ~/.local/bin
```

`task --list` shows the rest: test, fmt, logs, clean. From another directory,
use `task -d ~/manage-disk dry`. The Makefile has the same targets for
machines without Task.

## Tabs

**1 Overview.** Free space, and a stacked bar of where it goes by category
(projects, Docker, caches, toolchains, AI models, app data…). The biggest
spots are a table: pick one with ↑↓ and press enter to see it on the map.
Once a few days of samples exist, a sparkline of free space over time joins
the status line.

**2 Map.** A treemap of the folder you are in: each block's area is its share
of the disk, and each big block shows what's inside it. Arrow keys move
between blocks, enter zooms in, and ⌫ or esc zooms out. `v` switches the
nested blocks off, `o` reveals the block in Finder, and ♻ marks what a
cleanup would delete. With the mouse, click to select, click again to zoom
in, and right-click to zoom out.

**3 Explorer.** The same folder as a list, largest first, like ncdu. It's
read-only. `/` filters by name and `o` reveals in Finder. The map and the
explorer hand the selection to each other, and the bars use the map's colours.

**4 Clean.** The cleanup catalogue. Space switches a task on or off, enter
opens its items (filter them with `/`), and `c` cleans. A run shows a progress
bar, a stopwatch and a log you can scroll back through. `d` toggles dry run.

**5 Worktrees.** Every linked git worktree of the repos under `~/projects`
(`.claude/worktrees`, `.worktrees`, `~/.codex/worktrees`…), each with a verdict
and the evidence behind it. See [Worktrees](#worktrees) for how that is
decided. `space` picks, `a` picks everything verified, `c` removes (after a
fresh check and a confirm), and `b` also deletes the branches of merged work.
`f` fetches every repo first, so "merged" is judged against today's main.

**6 Docker.** How full Docker's own disk is, inside its VM, which is what a
build runs out of. Below it, every image, container, volume and slice of build
cache has a verdict and the evidence behind it, most removable first and
largest first. See [Docker](#docker) for how verdicts are decided. `space`
picks, `a` picks everything verified, and `c` removes after a fresh look and
a confirm. The header shows what your picks would leave free.

`?` shows every key for the screen you're on, and tabs are clickable. With the
mouse on, most terminals need ⌥ (iTerm: ⌥, Terminal.app: fn) held down to
select text.

## What it cleans

**Tier 1: caches that regenerate on their own.** These are on by default.

| Task | What |
|---|---|
| Build caches | `.next` and `.turbo` across `~/projects`, worktrees included |
| App update leftovers | Sparkle / Squirrel (`*.ShipIt`) / electron-updater downloads, GoogleUpdater, cached VS Code extension installers |
| Caches of apps that are gone | `~/Library/Caches/<bundle id>` with no installed or running app of that id |
| Dev tool caches | pnpm metadata, npm, npx, node-gyp, TypeScript, Go build, pip, uv, Bun, act, phpactor, Xcode DerivedData |
| Browser caches | Chrome, Arc, Firefox, Zen, Brave, Edge, Vivaldi, Opera. Sign-ins and history stay. |
| Homebrew cleanup | `brew cleanup -s --prune=all` |
| Old puppeteer browsers | keeps the newest Chrome of each kind |

**Tier 2: you pay with a re-download or a rebuild.** These are off until you
switch them on.

| Task | What |
|---|---|
| Docker build cache | `builder prune` → `image prune` → `builder prune`. **Never volumes**, because dev databases live there. The [Docker](#docker) tab picks resource by resource |
| pnpm stores | `~/Library/pnpm/store/v*` |
| node_modules in worktrees | every linked git worktree of the repos under `~/projects` (`.claude/worktrees`, `.worktrees`, `~/.codex/worktrees`) |
| Gradle & CocoaPods caches | `~/.gradle/caches`, `~/Library/Caches/CocoaPods` |
| Rust build output | `target/` folders Cargo marked with `CACHEDIR.TAG` |
| Go module cache | `go clean -modcache` |
| Old Node.js versions | nvm and nvm.fish. It preselects versions past Node's end of life and superseded patches, but never your default or a version holding global tools. |
| Android emulators | AVDs, preselected when unused for 30 days |

## Worktrees

Removing a worktree with `git worktree remove` keeps its branch and every
commit. What removal can lose is work that never became a commit, and the
checkout a program is running in. So each worktree gets a verdict:

| Verdict | When | Picked for you |
|---|---|---|
| **merged** | its PR is merged (and HEAD is the PR's head or behind it), or every commit's change is already in `origin/main` | yes |
| **old** | clean, every commit pushed, no open PR, untouched for 30 days | yes |
| **stale** | its folder is gone; git still lists it (removal = `git worktree prune`) | yes |
| **review** | removable without losing commits, but something needs a look: an open or closed-unmerged PR, commits held only here, a commit after the merged PR, or ignored files that aren't build output (`.env` that differs from the main checkout's, local data) | no |
| **keep** | a process works in it, it's locked, it has changed or untracked files, or it's outside your home. It can't be picked | — |

Where the evidence comes from:

- **git:** `git status --ignored`, `rev-list` against the default branch, and
  `git cherry`, which catches rebased and cherry-picked commits.
- **GitHub:** PR state and head commit, via `gh pr list --head <branch>`. A
  squash merge only shows up there; `git cherry` still sees its commits as
  unique.
- **Processes:** every process's working directory, from `lsof`, which is how
  live Claude and Codex sessions are found.

Build output doesn't count as ignored work: `node_modules`, `.next`,
`test-results`, `.husky/_` and similar. Neither does a `.env` byte-identical
to the main checkout's copy.

**Removal is never forced.** The run uses `git worktree remove` without
`--force`, so git itself refuses a worktree that picked up changes after the
check. Everything is checked again right before the confirm dialog. Branches
stay unless you press `b`, and even then only the branches of **merged**
worktrees are deleted. Git never prompts: a fetch whose SSH key wants a
passphrase fails instead of asking.

## Docker

Docker Desktop keeps images, containers, volumes and build cache on one Linux
filesystem inside its VM. That filesystem has its own size limit (Settings →
Resources → Virtual disk limit), so a build can fail with plenty of room left
on the Mac. The tab measures it the way a build would see it: `df` in a
throwaway container, whose `/` reports the filesystem Docker keeps everything
on. That needs no `--privileged`. It uses an `alpine` or `busybox` image
already on the machine and never pulls one. Without either, it says so.

| Verdict | When | Picked for you |
|---|---|---|
| **unused** | nothing refers to it: an untagged image, build cache no build used for a week, an empty volume | yes |
| **orphan** | a stopped container whose compose project ran from a folder that's gone, such as a removed worktree | yes |
| **old** | nothing runs it for 30+ days: a container stopped that long, or a pulled image created that long ago | yes |
| **review** | removable, but it looks wanted: stopped recently, created recently, recent build cache, or built here and never pushed (only a rebuild brings it back) | no |
| **data** | a volume holding anything. Its data exists nowhere else | never |
| **in use** | a running container, or an image one runs or a volume one mounts. It can't be picked | — |

Where the evidence comes from:

- **Docker:** `docker system df -v` for what each resource frees (an image's
  unique size, not the layers it shares), and `inspect` for tags, registry
  digests, states, mounts and compose labels.
- **Compose labels:** every compose container records the folder it was
  started from. A volume outlives its containers, so manage-disk remembers
  those folders in `compose-projects.json`. It also looks under `~/projects`
  for compose files that would start a project of that name.

**What must go first comes along.** Docker won't remove an image a stopped
container was made from, or a volume one mounts. Picking the image or volume
picks those containers too, and unpicking a container unpicks what needed
it. A removal goes containers → images → volumes → build cache. Build cache
goes last because images built here share their layers with it: removing
such an image frees little until the cache is pruned too, and the confirm
dialog says so when you haven't picked it.

**Nothing is forced.** `docker rm`, `rmi` and `volume rm` run without `-f`.
Docker itself refuses a container that started, or an image or volume a
container began using, after the check. Everything is checked again just
before the confirm dialog. The dialog warns in red when a volume's data is
about to go, and the summary shows Docker's disk before and after.

## Safety

- **It re-checks just before cleaning.** When you press `c`, it takes a fresh
  process snapshot and re-measures, then asks you to confirm. It skips
  anything in use:
  - a browser cache while that browser runs
  - `.next` under a running dev server
  - an npx folder an MCP server runs from
  - a worktree a process works in
  - DerivedData while Xcode runs
  - the Gradle cache while a Gradle daemon runs
- **A guard refuses** any path that isn't a specific folder inside your home.
  That includes `~`, `~/Library/Caches` itself, `~/projects`, anything under
  `~/.ssh`, Documents or Desktop. This is on top of the tasks only ever
  deleting paths they discovered themselves.
- **macOS privacy.** Folders macOS guards with a consent dialog aren't read
  unless you press `p` or pass `--include-private`: Desktop, Documents,
  Downloads, Pictures, Movies, Music, other apps' containers (Docker's VM disk
  included), iCloud and CloudStorage. A folder that doesn't answer within 15s
  (a pending dialog) is skipped, so a scan never hangs.
- **Dry run.** `--dry-run`, or `d` in the Clean tab, runs the whole flow and
  deletes nothing. That covers the Worktrees and Docker tabs too.
- **Logs and history.** Every run appends to
  `~/Library/Logs/manage-disk/<date>.log`. Real runs are also recorded in
  `~/Library/Application Support/manage-disk/history.jsonl`, which feeds the
  "last clean" line in the header.

## Measuring

Sizes are allocated bytes (`st_blocks × 512`), the same as `du`: a sparse
60 GB `Docker.raw` counts the 22 GB it actually uses. Hardlinks count once and
the scan never crosses into another volume. The scan runs in parallel and
keeps only entries of 1 MiB or more; smaller ones are folded into
"… N smaller items". A 5-million-file home takes about 30 seconds.

What a clean frees can come out lower than what was measured. pnpm clones its
files (APFS copy-on-write), so a `node_modules` folder shares blocks with the
store and with other checkouts. Docker hands space back to the Mac a few
minutes after a prune. The summary shows the real change in free space.

## Layout

```
main.go, report.go      entry point, the `report` command
*_cmd.go                the `worktrees` and `docker` commands
internal/scan           parallel walker, tree, categories, hotspots, privacy rules
internal/clean          task catalogue, process snapshot, guard, runner, history
internal/worktrees      judges each worktree from git, GitHub and process evidence
internal/docker         judges each image, container, volume and build cache slice
internal/treemap        squarified treemap layout, snapped to terminal cells
internal/ui             Bubble Tea model: overview, map, explorer, clean, worktrees, docker
internal/human          sizes, counts and times for people
```

The interface is built from [bubbles](https://github.com/charmbracelet/bubbles):
`key` and `help` for the key bindings and footer, `table` for the biggest
spots, `list` for a task's items, `textinput` for the explorer filter,
`progress` for the disk gauge, scan and clean bars, `viewport` for the run
log, `stopwatch` for its timer, and `spinner` while anything is measuring.

The interface also keeps small files next to the run history in
`~/Library/Application Support/manage-disk`: the last scan's file count, which
gives the next scan a real progress bar, free-space samples for the trend, and
the folders compose projects ran from.

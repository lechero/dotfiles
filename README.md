# manage-disk

Where your disk space goes, and the cleanups that are safe to repeat. It's a
[Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI for macOS.

```bash
task dry        # build and open it with deletion switched off
task run        # build and open it for real
task report     # print the overview and cleanup plan; changes nothing
task install    # copy the binary to ~/.local/bin
```

`task --list` shows the rest: test, fmt, logs, clean. From another directory,
use `task -d ~/manage-disk dry`. The Makefile has the same targets for
machines without Task.

## Tabs

**1 Overview.** Free space, a breakdown by category (projects, Docker, caches,
toolchains, AI models, app data…), the biggest spots, and what is cleanable
right now.

**2 Explorer.** Drill into any folder, largest first, like ncdu. It's
read-only; `o` reveals the selected item in Finder. Folders a cleanup would
delete are tagged `T1`/`T2`.

**3 Clean.** The cleanup catalogue. Space switches a task on or off, enter
opens its items so you can pick them one by one, and `c` cleans. `d` toggles
dry run.

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
| Docker build cache | `builder prune` → `image prune` → `builder prune`. **Never volumes**, because dev databases live there |
| pnpm stores | `~/Library/pnpm/store/v*` |
| node_modules in worktrees | every linked git worktree of the repos under `~/projects` (`.claude/worktrees`, `.worktrees`, `~/.codex/worktrees`) |
| Gradle & CocoaPods caches | `~/.gradle/caches`, `~/Library/Caches/CocoaPods` |
| Rust build output | `target/` folders Cargo marked with `CACHEDIR.TAG` |
| Go module cache | `go clean -modcache` |
| Old Node.js versions | nvm and nvm.fish. It preselects versions past Node's end of life and superseded patches, but never your default or a version holding global tools. |
| Android emulators | AVDs, preselected when unused for 30 days |

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
  deletes nothing.
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
internal/scan           parallel walker, tree, categories, hotspots, privacy rules
internal/clean          task catalogue, process snapshot, guard, runner, history
internal/ui             Bubble Tea model: overview, explorer, clean tab
internal/human          sizes, counts and times for people
```

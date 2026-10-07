# Fuentastic `dotfiles` FTW

Ya tu sabes!

Dotfiles managed with [chezmoi](https://www.chezmoi.io), plus the Homebrew packages they need, ranked by priority in [`packages.yaml`](packages.yaml).

## Set up a machine

With [Homebrew](https://brew.sh) installed:

```bash
brew install chezmoi go go-task
```

```bash
chezmoi init lechero/dotfiles
```

```bash
task -d ~/.local/share/chezmoi install
```

That installs every priority 1 and 2 package, then applies the dotfiles. `PRIO=1` installs only the core, and `PRIO=4` everything.

`task install` also works from any other clone of this repo, like `~/dotfiles`. When chezmoi isn't set up yet, it runs `chezmoi init` with that clone's origin first, so chezmoi gets its own copy in `~/.local/share/chezmoi`. `task dotfiles:init` does only that step.

## Everyday

| Command | What it does |
| --- | --- |
| `task` | List the tasks |
| `task check` | Show missing packages and what `chezmoi apply` would change |
| `task deps:install PRIO=3` | Install the missing packages up to priority 3 |
| `task dotfiles:preview` | Show what applying this checkout would change, before merging |
| `task dotfiles:update` | Pull the latest dotfiles and apply them |
| `task disk:dry` | See where the disk space goes, with deletion switched off |
| `task lint` | Lint the shell scripts, fish files, Neovim Lua and the Go apps |

## Priorities

| Priority | Meaning |
| --- | --- |
| 1 core | The dotfiles in this repo need it |
| 2 daily | Used regularly, or Neovim, fish or Rancher depend on it |
| 3 sometimes | Used in the last year, by a project, or by another package |
| 4 rarely | Not used in over a year: a candidate to remove |

## dotui

`task dotui` opens an interactive view of the same list, of what `chezmoi apply` would change, of where the disk space goes, and of your work on GitHub, GitLab and Jira. `tab` and `shift+tab` switch between them.

It opens on the **Dashboard**, which sums them all up:

- **This Mac:** its name, macOS version, uptime, load and battery.
- **Cards:** one per tab, numbered with the key that opens it.
  - **Packages:** what's installed per priority, what's missing, and how many Homebrew updates are waiting.
  - **Dotfiles:** changes to apply, the source's last commit, uncommitted files, and whether origin has newer commits.
  - **Disk:** how full it is, the free-space trend over the last week, and manage-disk's last clean.
  - **GitHub, GitLab and Jira:** the account, each list's total, and what needs you.
- **Next up:** the follow-ups from every service in one list, most urgent first. `enter` opens one on its tab.

On the dashboard, `i` installs what's missing up to priority 2, `u` pulls and applies the dotfiles, `U` upgrades the Homebrew packages (after asking, since `brew upgrade` doesn't), and `r` refreshes everything. Until you press one of those, it only reads: the update check doesn't update Homebrew, and the origin check uses `git ls-remote`, which changes nothing locally.

| Key | Packages | Key | Dotfiles |
| --- | --- | --- | --- |
| `1`–`4` | Show priorities up to | `d` | Diff |
| `m` | Missing only | `a` | Apply |
| `i` / `I` | Install the selected / all shown | `u` | Pull and apply |
| `/` | Filter | `r` | Refresh |

The Disk tab is [manage-disk](apps/cmd/manage-disk/README.md), whole: an overview, a treemap, an explorer, cache cleanups, worktrees and Docker. Its tabs are `1`–`6`, `?` lists its keys, and the mouse works there. Its scan of your home folder starts the first time you open the tab. `dotui --disk-dry-run` deletes nothing.

### GitHub, GitLab and Jira

Each of these tabs reads your work through the service's own CLI: `gh`, `glab` and `acli`. When the CLI is missing, `i` installs it, and when it isn't signed in, `c` hands the terminal to its sign-in. A tab's name counts what needs you.

The first view, Follow-ups, totals each list and ranks what needs you, most urgent first:

- reviews waiting for you;
- your pull or merge requests with changes requested, failing checks or conflicts, then the ones ready to merge;
- mentions and to-dos;
- Jira work assigned to you.

The other views are the lists themselves: reviews, your pull requests, assigned issues, and notifications or to-dos. On Jira they're your work items, the ones you watch, the ones you reported, and what you finished lately.

| Key | What it does |
| --- | --- |
| `enter` | Open the item's page: its state, checks or subtasks, description and activity |
| `o` / `y` | Open it in the browser / copy its link |
| `d` `c` `v` `C` `m` | Diff, checks or pipeline, review or approve, comment, merge |
| `b` | Check out its branch, in your clone of the repository under `~/projects` |
| `t` `a` | Jira: move it to a status, assign it to yourself |
| `1`–`5` | Switch views |
| `r` | Refresh. Dashboards also refresh every 5 minutes |

Actions hand the terminal to the CLI, which asks before it changes anything. Where it wouldn't ask, like approving on GitLab or checking out a branch, dotui asks first. A comment is written in `$EDITOR`, and an empty one isn't posted. The last dashboards are kept in `~/Library/Caches/dotui`, so the tabs have something to show while they refresh.

Without `--file`, dotui reads `packages.yaml` from chezmoi's source directory.

## manage-disk

The disk analyzer also runs on its own: `task disk` (or `task disk:dry`), plus `task disk:report`, `disk:worktrees` and `disk:docker`, which print and change nothing. See [its README](apps/cmd/manage-disk/README.md).

## The Go apps

dotui and manage-disk share one Go module in [`apps/`](apps). `task apps:install` puts both in `~/.local/bin`, and `task apps:test` vets, tests and lints them.

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

`task dotui` opens an interactive view of the same list, of what `chezmoi apply` would change, and of where the disk space goes. `tab` and `shift+tab` switch between them.

| Key | Packages | Key | Dotfiles |
| --- | --- | --- | --- |
| `1`–`4` | Show priorities up to | `d` | Diff |
| `m` | Missing only | `a` | Apply |
| `i` / `I` | Install the selected / all shown | `u` | Pull and apply |
| `/` | Filter | `r` | Refresh |

The Disk tab is [manage-disk](apps/cmd/manage-disk/README.md), whole: an overview, a treemap, an explorer, cache cleanups, worktrees and Docker. Its tabs are `1`–`6`, `?` lists its keys, and the mouse works there. Its scan of your home folder starts the first time you open the tab. `dotui --disk-dry-run` deletes nothing.

Without `--file`, dotui reads `packages.yaml` from chezmoi's source directory.

## manage-disk

The disk analyzer also runs on its own: `task disk` (or `task disk:dry`), plus `task disk:report`, `disk:worktrees` and `disk:docker`, which print and change nothing. See [its README](apps/cmd/manage-disk/README.md).

## The Go apps

dotui and manage-disk share one Go module in [`apps/`](apps). `task apps:install` puts both in `~/.local/bin`, and `task apps:test` vets, tests and lints them.

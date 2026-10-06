# Repository Guidelines

## Project Structure & Module Organization
Top-level dotfiles (`dot_bashrc`, `dot_zshrc`, aliases) mirror the home directory layout. Editor configuration lives in `dot_config/nvim`, with Lua modules under `dot_config/nvim/lua` and supporting docs in `dot_config/nvim/doc`. Automation and helper scripts sit in `scripts/` (CLI installers, Git utilities, tmux helpers), while `agents/prd` holds product design notes and `cv/` stores resume assets. `packages.yaml` ranks the Homebrew packages by priority, `Taskfile.yml` installs and checks them and the dotfiles, and `apps/` is one Go module with two apps: `dotui`, a TUI over packages, dotfiles and disk, and `manage-disk`, the disk analyzer dotui embeds as its Disk tab. All of these are repo-only, listed in `.chezmoiignore`. Keep additions grouped with similar artifacts to simplify future `chezmoi` syncs.

## Build, Test, and Development Commands
- `task install` sets up a machine: installs the packages up to `PRIO` (default 2) from `packages.yaml`, then runs `chezmoi apply`. `task check` reports missing packages and pending dotfile changes; `task` lists the rest.
- `task dotui` opens the TUI and `task disk:dry` the disk analyzer on its own; `task apps:test` vets, tests and lints both, and `task apps:install` puts them in `~/.local/bin`.
- `./setup` bootstraps a new machine: installs zoxide, initializes `chezmoi`, and applies the tracked dotfiles.
- `scripts/update-codex.sh [stable|nightly]` refreshes the Codex CLI; defaults to nightly and respects env overrides such as `INSTALL_DIR`.
- `scripts/source-changed.sh <base> <head>` reports whether tracked JS/TS files differ between refs; pair it with CI or pre-commit checks.
- `scripts/source-sha.sh` prints a stable SHA over JS/TS sources for cache keys or deployment guards.

## Coding Style & Naming Conventions
Shell scripts use `#!/usr/bin/env bash`, `set -euo pipefail`, two-space indentation, and descriptive kebab-case filenames (e.g., `update-codex.sh`). Lua code follows the `dot_config/nvim/dot_stylua.toml` profile (2-space indent, Unix line endings, single quotes preferred). stylua doesn't find that file under its chezmoi name, so check formatting with `task lint:lua`, which passes it explicitly. Go code in `apps/` is gofmt-formatted and passes `golangci-lint` (configured in `apps/.golangci.yml`). Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task needs. Markdown docs favor sentence-case headings and fenced command examples. Avoid committing machine-specific secrets; prefer environment variables or local-only `chezmoi` ignores.

## Testing Guidelines
Run `task lint` to check the shell scripts (including the rendered chezmoi run scripts), fish files, Neovim Lua and the Go apps in one go. Lint shell updates with `shellcheck scripts/<file>.sh` and rerun `./setup` in a disposable environment after major install changes. For Neovim changes, execute `nvim --headless -u dot_config/nvim/init.lua +qall` to ensure the config loads cleanly. Document any manual test steps in `agents/prd` when they drive product workflows.

## Commit & Pull Request Guidelines
Follow conventional commits (`feat:`, `fix:`, `chore:`) to match history (`feat: added agent orchestrator 'prd' s`). Use concise present-tense summaries and include scoped backticks for filenames when relevant. Pull requests should describe the motivation, link supporting issues or tickets, and note verification steps (command output, screenshots for terminal themes, etc.). Flag breaking changes or machine-specific caveats early so reviewers can rehearse them locally.

## Security & Configuration Tips
Audit installer scripts for remote downloads; prefer pinned tags or checksums when extending `scripts/update-codex.sh`. Never store API keys or tokens inside tracked dotfiles—use `chezmoi` templates or `git-crypt` if sensitive overrides are unavoidable. Validate external URLs referenced in setup scripts to guard against supply-chain pivots, and document new dependencies in `SIDEKICK.md` for visibility.

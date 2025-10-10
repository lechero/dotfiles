# Repository Guidelines

## Project Structure & Module Organization
Top-level dotfiles (`dot_bashrc`, `dot_zshrc`, aliases) mirror the home directory layout. Editor configuration lives in `dot_config/nvim`, with Lua modules under `dot_config/nvim/lua` and supporting docs in `dot_config/nvim/doc`. Automation and helper scripts sit in `scripts/` (CLI installers, Git utilities, tmux helpers), while `agents/prd` holds product design notes and `cv/` stores resume assets. Keep additions grouped with similar artifacts to simplify future `chezmoi` syncs.

## Build, Test, and Development Commands
- `./setup` bootstraps a new machine: installs zoxide, initializes `chezmoi`, and applies the tracked dotfiles.
- `scripts/update-codex.sh [stable|nightly]` refreshes the Codex CLI; defaults to nightly and respects env overrides such as `INSTALL_DIR`.
- `scripts/source-changed.sh <base> <head>` reports whether tracked JS/TS files differ between refs; pair it with CI or pre-commit checks.
- `scripts/source-sha.sh` prints a stable SHA over JS/TS sources for cache keys or deployment guards.

## Coding Style & Naming Conventions
Shell scripts use `#!/usr/bin/env bash`, `set -euo pipefail`, two-space indentation, and descriptive kebab-case filenames (e.g., `update-codex.sh`). Lua code follows the `dot_config/nvim/dot_stylua.toml` profile (2-space indent, Unix line endings, single quotes preferred); run `stylua dot_config/nvim` before committing. Markdown docs favor sentence-case headings and fenced command examples. Avoid committing machine-specific secrets; prefer environment variables or local-only `chezmoi` ignores.

## Testing Guidelines
Run `stylua dot_config/nvim` to confirm Lua formatting and catch syntax slips. Lint shell updates with `shellcheck scripts/<file>.sh` and rerun `./setup` in a disposable environment after major install changes. For Neovim changes, execute `nvim --headless -u dot_config/nvim/init.lua +qall` to ensure the config loads cleanly. Document any manual test steps in `agents/prd` when they drive product workflows.

## Commit & Pull Request Guidelines
Follow conventional commits (`feat:`, `fix:`, `chore:`) to match history (`feat: added agent orchestrator 'prd' s`). Use concise present-tense summaries and include scoped backticks for filenames when relevant. Pull requests should describe the motivation, link supporting issues or tickets, and note verification steps (command output, screenshots for terminal themes, etc.). Flag breaking changes or machine-specific caveats early so reviewers can rehearse them locally.

## Security & Configuration Tips
Audit installer scripts for remote downloads; prefer pinned tags or checksums when extending `scripts/update-codex.sh`. Never store API keys or tokens inside tracked dotfiles—use `chezmoi` templates or `git-crypt` if sensitive overrides are unavoidable. Validate external URLs referenced in setup scripts to guard against supply-chain pivots, and document new dependencies in `SIDEKICK.md` for visibility.

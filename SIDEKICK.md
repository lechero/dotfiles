# Sidekick.nvim Guide

## What This Plugin Does
- Embeds [folke/sidekick.nvim](https://github.com/folke/sidekick.nvim) into your Neovim setup, giving you Sidekick’s inline edit suggestions and Sidekick CLI integration without leaving the editor.
- Bridges to the standalone Sidekick CLI and agent ecosystem, letting you chat with AI agents, run prompts, and apply generated edits directly to the current buffer.
- Integrates with Sidekick’s “next edit suggestion” flow so that you can jump to or apply edits with a single key press.
- Uses your existing Sidekick configuration (workflows, agents, prompts) and displays responses inside a tmux pane, matching your configured mux backend.

## Requirements & Startup
- Install the Sidekick CLI and configure your accounts/agents there first.
- Because `cli.mux.backend = "tmux"` in your config, Neovim expects a running tmux session; the Sidekick CLI opens in a managed tmux pane.
- Launch Neovim normally. The plugin loads lazily, so the CLI is started only when you trigger one of the mappings below.

## Keybindings
All mappings are defined in `dot_config/nvim/init.lua` around the Sidekick plugin block. They are registered through Lazy and available once the plugin loads.

| Mode(s) | Key | Action |
| --- | --- | --- |
| `n`/`i`/`v`/`t` | `<C-.>` | Focus the Sidekick CLI pane (helpful when you already have a conversation running). |
| `n`/`v` | `<leader>aa` | Toggle the Sidekick CLI window. Opens it if closed, hides it if already visible. |
| `n`/`v` | `<leader>ac` | Open the CLI preloaded with the `claude` agent and focus it immediately. Useful shortcut for your primary AI. |
| `n`/`v` | `<leader>ap` | Open the prompt picker so you can run a saved Sidekick prompt against the current buffer or visual selection. |
| `n` | `<leader>as` | Open the CLI tool picker (lets you choose from installed Sidekick apps/tools). |
| `v` | `<leader>as` | Send the current visual selection to the active CLI agent. Same key as above but only active in visual mode. |
| `n` (expr) | `<Tab>` | Call `sidekick.nes_jump_or_apply()`: jump to the next pending edit suggestion, or apply it if your cursor is already on one. Falls back to a literal tab when no edits exist. |

> **Tip:** The visual-mode `<leader>as` sends the selection immediately, while the normal-mode `<leader>as` opens the picker. They share the same key combo intentionally; rely on mode to choose the behavior.

## Typical Workflows
- **Inline edits:** Request a change from an agent, then press `<Tab>` repeatedly to step through each suggestion. Edit suggestions appear as diff hunks; the mapping either moves you to the next hunk or applies it.
- **Prompting:** Use `<leader>ap` to open your Sidekick prompt library. Prompts can run on the whole buffer (`n` mode) or the highlighted block (`v` mode).
- **Tool hopping:** Press `<leader>as` (normal mode) to switch agents (e.g., `copilot`, `claude`, custom tools). Once selected, the CLI pane opens in tmux.
- **Visual send:** Highlight code, hit `<leader>as` (visual mode) to send the snippet to the current agent without picking a tool again.
- **Quick focus:** When the CLI already exists, `<C-.>` jumps focus there no matter the mode, making it easy to answer follow-up questions or copy output.

## Troubleshooting Notes
- If nothing happens when toggling, make sure you have a tmux session and that the Sidekick CLI binary is on your `$PATH`.
- Inline edit suggestions require the Sidekick agent to provide structured edits. If `<Tab>` always falls back to inserting a tab, confirm that the agent supports edit suggestions and that a request has produced diffs.
- The mapping block is tagged with `-- stylua: ignore`; keep formatting changes minimal if you edit it, or update the directive accordingly.

Happy pairing! Use `:h sidekick.nvim` for upstream docs or open the GitHub repo for advanced configuration examples. The mappings above should cover the fastest way to chat, run prompts, and apply AI edits from within Neovim.***

## Fish shell dependencies

The fish config in `dot_config/fish` adds these, each pinned to a release. `run_onchange_after_install-fish-plugins.sh.tmpl` installs them with fisher whenever `dot_config/fish/fish_plugins` changes, after checking the fisher bootstrap against its sha256.

| Plugin | Version | Purpose |
| --- | --- | --- |
| [jorgebucaran/fisher](https://github.com/jorgebucaran/fisher) | 4.4.8 | Plugin manager |
| [ilancosman/tide](https://github.com/IlanCosman/tide) | v6.2.0 | Prompt |
| [patrickf1/fzf.fish](https://github.com/PatrickF1/fzf.fish) | v11.0 | fzf searches on keys |
| [jorgebucaran/nvm.fish](https://github.com/jorgebucaran/nvm.fish) | 2.2.17 | Node versions |
| [reitzig/sdkman-for-fish](https://github.com/reitzig/sdkman-for-fish) | v2.1.0 | The `sdk` command in fish |

## tmux dependencies

The tmux config in `dot_config/tmux` installs these with [TPM](https://github.com/tmux-plugins/tpm). `run_onchange_after_install-tmux-plugins.sh.tmpl` clones TPM itself at commit `99469c4` and runs its installer whenever the plugin list in `tmux.conf` changes. TPM can't pin plugins to a commit, so the plugins follow their default branches.

| Plugin | Purpose |
| --- | --- |
| [tmux-plugins/tmux-sensible](https://github.com/tmux-plugins/tmux-sensible) | Common defaults |
| [tmux-plugins/tmux-logging](https://github.com/tmux-plugins/tmux-logging) | Save pane output |
| [sainnhe/tmux-fzf](https://github.com/sainnhe/tmux-fzf) | fzf menus for sessions, windows and panes |
| [nhdaly/tmux-better-mouse-mode](https://github.com/nhdaly/tmux-better-mouse-mode) | Mouse scrolling |
| [abhinav/tmux-fastcopy](https://github.com/abhinav/tmux-fastcopy) | Copy text by hint |
| [jaclu/tmux-menus](https://github.com/jaclu/tmux-menus) | Popup menus |
| [tmux-plugins/tmux-yank](https://github.com/tmux-plugins/tmux-yank) | Copy to the system clipboard |
| [jimeh/tmuxifier](https://github.com/jimeh/tmuxifier) | Session layouts |
| [tmux-plugins/tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect) | Save and restore sessions |
| [carlocab/tmux-nvr](https://github.com/carlocab/tmux-nvr) | One Neovim per session |
| [omerxx/tmux-sessionx](https://github.com/omerxx/tmux-sessionx) | Session picker |
| [catppuccin/tmux](https://github.com/catppuccin/tmux) | Theme |

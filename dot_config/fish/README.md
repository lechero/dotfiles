# Fish

Fish 4 config, applied by chezmoi to `~/.config/fish`.

## Layout

| Path | What it holds |
| --- | --- |
| `config.fish` | Interactive setup: vi key bindings, the prompt, zoxide, jump, fzf keys, switching Node on `.nvmrc`. |
| `conf.d/00-env.fish` | `PATH` and environment variables, for every shell including scripts. |
| `conf.d/05-tmux.fish` | Starts tmux in a new terminal window, attaching to the most recent session. |
| `conf.d/10-plugins.fish` | Loads the plugins fisher installed into `plugins/`. |
| `conf.d/abbr.fish` | Abbreviations, such as `gs` for `git status` and `...` for `cd ../../`. |
| `conf.d/colors.fish` | Syntax highlighting colors. |
| `conf.d/tide.fish` | tide prompt settings. |
| `functions/`, `completions/` | Your own functions and completions, one per file, loaded on first use. |
| `fish_plugins` | The fisher plugins, each pinned to a release. |

Not tracked: `plugins/` (installed by fisher), `fish_variables` (fish's per-machine state) and `secrets.fish`.

## Prompt

The prompt is tide unless you pick starship, which shares `~/.config/starship.toml` with zsh:

```fish
prompt-switch starship
```

`prompt-switch` with no argument switches to the other one. The choice is saved per machine and takes effect in new shells. The shell you run it in restarts right away, unless it has jobs running.

## Secrets

API keys go in `~/.config/fish/secrets.fish`, which `conf.d/00-env.fish` loads in every shell. chezmoi ignores it, so it never reaches the repo:

```fish
set -gx OPENAI_API_KEY sk-...
```

## Plugins

Edit `fish_plugins` here and run `chezmoi apply`, which runs `fisher update` whenever the list changes. If you install one with `fisher install` instead, save the updated list back to the repo:

```bash
chezmoi re-add ~/.config/fish/fish_plugins
```

## Keys

| Key | Action |
| --- | --- |
| `ctrl-r` | Search history with fzf |
| `ctrl-f` | Find files with fzf |
| `ctrl-alt-l` / `ctrl-alt-s` | Search the git log / git status |
| `ctrl-alt-p` / `ctrl-alt-v` | Search processes / variables |
| `ctrl-s` | Accept the autosuggestion and run it |
| `ctrl-up` / `ctrl-down` | Search history for the word under the cursor |

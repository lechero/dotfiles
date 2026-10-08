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

## Containers

`dock` picks the container engine for the shell you're in: Rancher Desktop, Docker Desktop or Podman. It sets `DOCKER_HOST`, which `docker`, `docker compose` and lazydocker (`lz`) follow, and starts the engine if it isn't running:

```fish
dock podman      # this shell, and anything you start from it
dock             # pick one with fzf, with which ones are running
dock off         # back to docker's own current context
dock -d rancher  # new shells start on Rancher Desktop too
```

Other shells keep their own engine, and tide shows the current one at the far right, after the clock, in that engine's color. Anything you start from this shell inherits it, so a Neovim started here opens lazydocker (`;d`) on it too. A new tmux pane starts on the default, or on docker's own context if there's none. Switching doesn't touch kubectl's context, which tide only shows for a real cluster: it hides the local `rancher-desktop` and `docker-desktop` ones (`tide_kubectl_default_contexts`).

## Asking for a command

Type what you want at the prompt in plain words and press `alt-a`. Copilot replaces it with a command, which you check, edit if needed and run with `enter`. Nothing runs on its own:

```fish
find files over 500MB below here   # alt-a, and the line becomes:
find . -type f -size +500M
```

`ask` does the same and prints the command, as in `ask list docker volumes nobody uses`. It goes through the Copilot CLI (`copilot`) on your Copilot plan and takes a few seconds, longer for the first question in a while. It uses gpt-4.1 unless `ask_model` names another of the models in `copilot --help` that your plan offers (`set -U ask_model <model>`). Copilot gets no tools for it and runs in an empty folder, so it sees only the question: not your files, and not your MCP servers.

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
| `alt-a` | Swap the question on the command line for Copilot's command (see [Asking for a command](#asking-for-a-command)) |
| `ctrl-up` / `ctrl-down` | Search history for the word under the cursor |

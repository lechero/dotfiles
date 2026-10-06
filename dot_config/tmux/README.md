# tmux

tmux config, applied by chezmoi to `~/.config/tmux`. Opening kitty starts tmux through `~/.config/fish/conf.d/05-tmux.fish`, attaching to the most recent session.

## Plugins

TPM installs the plugins listed in `tmux.conf` into `plugins/`, which isn't tracked. chezmoi runs TPM's installer whenever the list changes. By hand, `prefix + I` installs and `prefix + U` updates.

## Keys

The prefix is `ctrl-space`. Everything below works without it.

| Key | Action |
| --- | --- |
| `alt-r` | Pick a session (sessionx) |
| `alt-e` | Popup with the dotfiles session; `alt-e` again closes it |
| `alt-y` | Popup with the dash session; `alt-y` again closes it |
| `alt-d` | Detach |
| `alt-t` | New window in the current directory |
| `alt-n` / `alt-p`, `alt-.` / `alt-,` | Next / previous window |
| `alt-1` … `alt-9`, `alt-0` | Go to window 1 to 9, or 0 |
| `alt-s` / `alt-v` | Split below / to the right, in the current directory |
| `alt-h` `alt-j` `alt-k` `alt-l` | Move to the pane left, down, up or right |
| `alt-w` | Back to the previous pane |
| `alt-z` | Zoom the pane |
| `alt-arrow` | Resize the pane |
| `alt-f` | Next layout |
| `alt-o` | Rotate panes |
| `alt-?` | List all keys |

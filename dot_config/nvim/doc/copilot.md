# Copilot

Copilot ([copilot.lua](https://github.com/zbirenbaum/copilot.lua)) suggests code inline as you type, and in the completion menu through blink-copilot. It starts the first time you enter insert mode.

## Is it working?

| Command | Shows |
| --- | --- |
| `:CopilotAccount` | Which GitHub account Copilot uses here, and who is signed in to it |
| `:Copilot status` | Whether Copilot is running and attached to this buffer |
| `:checkhealth copilot` | Node, the language server, and the sign-in |

## Another account for some projects

Projects can use a different GitHub account from everything else. List them in `~/.config/copilot-accounts/folders`, one account and folder per line:

```
# account  folder
acme       ~/work/acme
acme-ops   ~/work/acme/ops
```

A folder covers everything inside it, and the deepest listed folder wins. Anything not listed uses your usual sign-in.

Each account signs in once. Open a file in one of its folders and run `:Copilot auth`, then enter the code on GitHub in a browser signed in to that account (a private window works). After that, Neovim in that folder uses the account without asking.

- One Copilot language server serves a whole Neovim, so the file you first type in picks the account. Restart Neovim to switch to another account.
- Each account's sign-in lives in `~/.config/copilot-accounts/<account>`, which the language server gets as its `XDG_CONFIG_HOME`. chezmoi ignores the whole folder, so neither the list nor the sign-ins reach the repo.
- In `:checkhealth copilot`, the "LSP authentication status" line shows the account in use. The "Local credentials" line always looks at the usual sign-in, in `~/.config/github-copilot`.

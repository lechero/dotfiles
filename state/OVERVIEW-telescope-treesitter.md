# Telescope & Treesitter — Technical Overview

## Telescope

**Plugin:** `nvim-telescope/telescope.nvim` (branch `0.1.x`)  
**File:** `dot_config/nvim/lua/fuentastic/plugins/telescope.lua`  
**Load event:** `VimEnter`

### Dependencies

| Plugin | Notes |
|--------|-------|
| `nvim-lua/plenary.nvim` | Required utility library |
| `nvim-telescope/telescope-fzf-native.nvim` | Native FZF sorter, built with `make` (only loaded if `make` is executable) |
| `nvim-telescope/telescope-ui-select.nvim` | Replaces `vim.ui.select` with a Telescope dropdown |
| `nvim-tree/nvim-web-devicons` | Icons (only enabled if `vim.g.have_nerd_font` is set) |

### Configuration

```lua
preview = {
  treesitter = false,   -- treesitter highlighting disabled in preview pane
}
extensions = {
  ['ui-select'] = require('telescope.themes').get_dropdown(),
}
```

Both `fzf` and `ui-select` extensions are loaded via `pcall` (safe load, no error on failure).

### Keymaps

All keymaps are in normal mode under the `<leader>s` namespace, plus a few shortcuts:

| Keymap | Action |
|--------|--------|
| `<leader>sh` | Search help tags |
| `<leader>sk` | Search keymaps |
| `<leader>sf` | Find files |
| `<leader>ss` | Select Telescope builtin picker |
| `<leader>sw` | Grep current word under cursor |
| `<leader>sg` | Live grep across project |
| `<leader>sd` | Search diagnostics |
| `<leader>sr` | Resume last search |
| `<leader>s.` | Search recent (old) files |
| `<leader><leader>` | Find open buffers |
| `<leader>/` | Fuzzy search in current buffer (dropdown, no previewer, `winblend=10`) |
| `<leader>s/` | Live grep across open files only |
| `<leader>sn` | Find files inside Neovim config dir (`stdpath('config')`) |

---

## Treesitter

**Plugin:** `nvim-treesitter/nvim-treesitter` (branch `main`)  
**File:** `dot_config/nvim/lua/fuentastic/plugins/treesitter.lua`  
**Build cmd:** `:TSUpdate bash c diff html lua luadoc markdown markdown_inline query vim vimdoc`

### Installed Parsers

Parsers are explicitly declared — `auto_install` is **off**.

```
bash, c, diff, html, lua, luadoc, markdown, markdown_inline, query, vim, vimdoc
```

### Configuration

| Feature | Setting |
|---------|---------|
| `highlight.enable` | `true` |
| `highlight.additional_vim_regex_highlighting` | `{ 'ruby' }` — Ruby uses regex highlighting as a fallback |
| `indent.enable` | `true` |
| `indent.disable` | `{ 'ruby' }` — Ruby indentation handled elsewhere |
| `auto_install` | `false` — parsers must be manually added |

### Compatibility Shim

A runtime patch is applied to handle API differences across Treesitter versions:

```lua
if parsers.ft_to_lang == nil then
  parsers.ft_to_lang = function(ft)
    if vim.treesitter.language and vim.treesitter.language.get_lang then
      return vim.treesitter.language.get_lang(ft) or ft
    end
    return ft
  end
end
```

This polyfills `parsers.ft_to_lang` when it's absent (removed in newer Treesitter versions), delegating to `vim.treesitter.language.get_lang` instead.

### Telescope Integration Note

Telescope's preview pane has `treesitter = false`, meaning Treesitter syntax highlighting is explicitly **disabled** in Telescope previews. Highlighting in regular buffers is unaffected.

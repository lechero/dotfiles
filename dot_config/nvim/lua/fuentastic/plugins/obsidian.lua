-- The community fork of epwalsh/obsidian.nvim: maintained, and its note and tag completion is an
-- in-process language server, so blink.cmp's lsp source offers it after `[[` and `#`.
return {
  'obsidian-nvim/obsidian.nvim',
  version = '*', -- recommended, use latest release instead of latest commit
  lazy = true,
  ft = 'markdown',
  -- Replace the above line with this if you only want to load obsidian.nvim for markdown files in your vault:
  -- event = {
  --   -- If you want to use the home shortcut '~' here you need to call 'vim.fn.expand'.
  --   -- E.g. "BufReadPre " .. vim.fn.expand "~" .. "/my-vault/*.md"
  --   -- refer to `:h file-pattern` for more examples
  --   "BufReadPre path/to/my-vault/*.md",
  --   "BufNewFile path/to/my-vault/*.md",
  -- },
  opts = {
    legacy_commands = false, -- `:Obsidian new`, not `:ObsidianNew` (removed in 4.0)
    workspaces = {
      {
        name = 'personal',
        path = '~/fuentastic',
      },
    },

    -- see below for full list of options 👇
  },
}

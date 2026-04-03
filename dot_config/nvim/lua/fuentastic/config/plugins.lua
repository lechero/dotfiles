local plugins = {
  'tpope/vim-sleuth',
  {
    'lewis6991/gitsigns.nvim',
    opts = {
      signs = {
        add = { text = '+' },
        change = { text = '~' },
        delete = { text = '_' },
        topdelete = { text = '‾' },
        changedelete = { text = '~' },
      },
    },
  },
  require('fuentastic.plugins.which-key'),
  require('fuentastic.plugins.telescope'),
  require('fuentastic.plugins.conform'),
  require('fuentastic.plugins.todo-comments'),
  require('fuentastic.plugins.sidekick'),
  require('fuentastic.plugins.mini'),
  require('fuentastic.plugins.treesitter'),
  require('fuentastic.plugins.catpuccin'),
  require('fuentastic.plugins.neo-tree'),
  require('fuentastic.plugins.alpha'),
  require('fuentastic.plugins.codex'),
  require('fuentastic.plugins.copilot'),
  require('fuentastic.plugins.possession'),
  require('fuentastic.plugins.obsidian'),
  require('fuentastic.plugins.spectre'),
  require('fuentastic.plugins.diffview'),
  require('fuentastic.plugins.harpoon'),
  require('fuentastic.plugins.oil'),
  require('fuentastic.plugins.yanky'),
  require('fuentastic.plugins.vim-floaterm'),
  require('fuentastic.plugins.trouble'),
  require('fuentastic.plugins.copilot-cmp'),
  require('fuentastic.plugins.nvim-cmp'),
}

vim.list_extend(plugins, require('fuentastic.config.lsp'))

return plugins

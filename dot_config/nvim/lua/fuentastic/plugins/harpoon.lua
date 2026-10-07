return {
  'ThePrimeagen/harpoon',
  branch = 'harpoon2',
  dependencies = { 'nvim-lua/plenary.nvim' },
  config = function()
    local harpoon = require 'harpoon'
    -- Its BufLeave and VimLeavePre hooks: each file's cursor position is saved as you leave it, so
    -- jumping back lands there, and the list is saved as nvim quits.
    harpoon:setup()

    -- Not <leader>a: that's Sidekick's prefix, so Harpoon would wait out which-key's timeout.
    vim.keymap.set('n', '<leader>m', function()
      harpoon:list():add()
    end, { desc = 'Harpoon: [M]ark this file' })
    vim.keymap.set('n', '<C-e>', function()
      harpoon.ui:toggle_quick_menu(harpoon:list())
    end, { desc = 'Harpoon: the marked files' })
    -- Not <leader>1-4, which switch tabs (config/keymaps.lua), nor Alt-1-4, which tmux has.
    for i = 1, 4 do
      vim.keymap.set('n', '<leader>n' .. i, function()
        harpoon:list():select(i)
      end, { desc = 'Harpoon: file ' .. i })
    end
  end,
}

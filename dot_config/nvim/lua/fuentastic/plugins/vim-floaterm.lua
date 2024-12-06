return {
  'voldikss/vim-floaterm',
  config = function()
    vim.keymap.set('n', ';g', '<Cmd>FloatermNew --width=0.8 --height=0.8 --opener=edit --title=Git lazygit<CR>', { remap = false, silent = true })
    vim.keymap.set('n', ';d', '<Cmd>FloatermNew --width=0.8 --height=0.8 --opener=edit --title=Docker lazydocker<CR>', { remap = false, silent = true })
    vim.keymap.set('n', ';s', '<Cmd>FloatermNew --width=0.8 --height=0.8 --opener=edit --title=SQL lazysql<CR>', { remap = false, silent = true })
    vim.keymap.set('n', ';o', '<Cmd>FloatermNew --width=0.8 --height=0.8 --opener=edit --title=Yazi yazi<CR>', { remap = false, silent = true })
  end,
}

return {
  'catppuccin/nvim',
  name = 'catppuccin',
  priority = 1000,
  opt = {
    transparent = true,
    styles = {
      sidebars = 'transparent',
      float = 'transparent',
    },
  },
  init = function()
    vim.cmd.colorscheme 'catppuccin-mocha'
    -- vim.cmd.colorscheme 'catppuccin'
  end,
}

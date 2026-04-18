return {
  'nvim-treesitter/nvim-treesitter',
  branch = 'main',
  build = ':TSUpdate bash c diff html lua luadoc markdown markdown_inline query vim vimdoc',
  opts = {
    ensure_installed = {
      'bash',
      'c',
      'diff',
      'html',
      'lua',
      'luadoc',
      'markdown',
      'markdown_inline',
      'query',
      'vim',
      'vimdoc',
    },
    auto_install = false,
    highlight = {
      enable = true,
      additional_vim_regex_highlighting = { 'ruby' },
    },
    indent = { enable = true, disable = { 'ruby' } },
  },
  config = function(_, opts)
    local ok, parsers = pcall(require, 'nvim-treesitter.parsers')
    if ok and parsers.ft_to_lang == nil then
      parsers.ft_to_lang = function(ft)
        if vim.treesitter.language and vim.treesitter.language.get_lang then
          return vim.treesitter.language.get_lang(ft) or ft
        end
        return ft
      end
    end

    require('nvim-treesitter').setup(opts)
  end,
}

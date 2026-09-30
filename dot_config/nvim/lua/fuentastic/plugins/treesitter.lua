-- The `main` branch ignores the old `ensure_installed` / `highlight` / `indent` options:
-- parsers are installed explicitly and highlighting is started per filetype.
local parsers = {
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
  -- java
  'java',
  'groovy', -- build.gradle
  'kotlin', -- build.gradle.kts
  'properties', -- application.properties
  'xml', -- pom.xml
  'yaml', -- application.yml
}

return {
  'nvim-treesitter/nvim-treesitter',
  branch = 'main',
  lazy = false, -- the main branch can't be lazy-loaded
  build = ':TSUpdate',
  config = function()
    -- Skips what's already installed. Parsers and their queries land in stdpath('data')/site,
    -- which is ahead of the plugin dir on the runtimepath.
    require('nvim-treesitter').install(parsers)

    local filetypes = {}
    for _, lang in ipairs(parsers) do
      vim.list_extend(filetypes, vim.treesitter.language.get_filetypes(lang))
    end

    vim.api.nvim_create_autocmd('FileType', {
      group = vim.api.nvim_create_augroup('fuentastic-treesitter', { clear = true }),
      pattern = filetypes,
      callback = function(args)
        -- Fails quietly while a parser is still installing on first start.
        if pcall(vim.treesitter.start, args.buf) then
          vim.bo[args.buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
        end
      end,
    })
  end,
}

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
  -- vue, plus what its <script> and <style> blocks inject
  'vue',
  'javascript',
  'typescript',
  'tsx',
  'css',
  'scss',
  -- jsp: this config's own parser (tree-sitter-jsp/), which hands the markup to html and the
  -- scriptlets to java
  'jsp',
}

-- Parsers with an indentexpr of their own. nvim-treesitter's goes by the smallest tree around a
-- line, which around scriptlets is the wrong one of JSP's java and html.
local indentexprs = {
  jsp = "v:lua.require'fuentastic.jsp_indent'.indentexpr()",
}

return {
  'nvim-treesitter/nvim-treesitter',
  branch = 'main',
  lazy = false, -- the main branch can't be lazy-loaded
  build = ':TSUpdate',
  config = function()
    -- Nothing upstream parses JSP. nvim-treesitter builds the parser in place; after changing the
    -- grammar, `:TSInstall! jsp`.
    vim.api.nvim_create_autocmd('User', {
      group = vim.api.nvim_create_augroup('fuentastic-treesitter-parsers', { clear = true }),
      pattern = 'TSUpdate',
      callback = function()
        require('nvim-treesitter.parsers').jsp = {
          install_info = { path = vim.fn.stdpath('config') .. '/tree-sitter-jsp' },
        }
      end,
    })

    -- GitLab CI files (ftdetect/gitlab-ci.lua). A FileType autocmd for yaml doesn't fire for them.
    vim.treesitter.language.register('yaml', 'yaml.gitlab')

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
          vim.bo[args.buf].indentexpr = indentexprs[args.match] or "v:lua.require'nvim-treesitter'.indentexpr()"
        end
      end,
    })
  end,
}

-- A Backstage template's skeleton/: fetch:template renders each file in it with Nunjucks, so one with
-- Nunjucks markup is a template of a JSON, YAML... file, not one. Its filetype is jinja (Nunjucks
-- has Jinja's syntax): the markup is highlighted, the text around it in the language of the file it
-- renders, b:template_lang (after/queries/jinja/injections.scm), and the template's linter checks
-- it rather than the language's (doc/linting.md).
vim.filetype.add {
  pattern = {
    ['.*/skeleton/.*'] = function(path, buf)
      if not buf then
        return -- matching a name alone, as below
      end
      local text = table.concat(vim.api.nvim_buf_get_lines(buf, 0, 500, false), '\n')
      if not (text:find('${{', 1, true) or text:find('{%', 1, true)) then
        return
      end
      local ft = vim.filetype.match { filename = path }
      return 'jinja', function(b)
        vim.b[b].template_lang = ft and vim.treesitter.language.get_lang(ft)
      end
    end,
  },
}

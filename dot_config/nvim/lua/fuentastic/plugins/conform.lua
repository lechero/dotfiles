-- The Eclipse profile jdtls formats java with, when the project's Spotless config names one
-- (see plugins/jdtls.lua).
local function spotless_profile(bufnr)
  local client = vim.lsp.get_clients({ bufnr = bufnr, name = 'jdtls' })[1]
  return client and vim.tbl_get(client.settings, 'java', 'format', 'settings', 'url')
end

-- IntelliJ's command-line formatter (Community, Ultimate, or a Toolbox install).
local intellij = vim.fn.glob('/Applications/IntelliJ IDEA*.app/Contents/bin/format.sh', false, true)[1]
  or vim.fn.glob(vim.env.HOME .. '/Applications/IntelliJ IDEA*.app/Contents/bin/format.sh', false, true)[1]

-- A project's IntelliJ code style lives where IntelliJ keeps it. Only used without a Spotless
-- profile: jdtls' Eclipse formatter can't express some IntelliJ rules (spaces around `::`, one).
local function intellij_scheme(bufnr)
  local root = vim.fs.root(bufnr, '.idea')
  local scheme = root and root .. '/.idea/codeStyles/Project.xml'
  return scheme and vim.uv.fs_stat(scheme) and not spotless_profile(bufnr) and scheme or nil
end

-- Its own config/system dirs: IntelliJ allows one instance per dir, so the formatter doesn't
-- collide with an open IDE.
local function idea_properties()
  local dir = vim.fn.stdpath('cache') .. '/intellij-format'
  local file = dir .. '/idea.properties'
  if not vim.uv.fs_stat(file) then
    vim.fn.mkdir(dir, 'p')
    vim.fn.writefile({
      'idea.config.path=' .. dir .. '/config',
      'idea.system.path=' .. dir .. '/system',
      'idea.log.path=' .. dir .. '/log',
      'idea.plugins.path=' .. dir .. '/plugins',
    }, file)
  end
  return file
end

return {
  'stevearc/conform.nvim',
  event = { 'BufWritePre' },
  cmd = { 'ConformInfo' },
  keys = {
    {
      '<leader>f',
      function()
        local conform = require('conform')
        -- IntelliJ needs a few seconds to boot, so say something while it runs.
        local slow = vim.iter((conform.list_formatters_to_run())):any(function(formatter)
          return formatter.name == 'intellij'
        end)
        if slow then
          vim.notify('Formatting with IntelliJ...')
        end
        conform.format({ async = true, lsp_format = 'fallback' }, function(err)
          if slow then
            vim.notify(err or 'Formatted with IntelliJ', err and vim.log.levels.WARN or nil)
          end
        end)
      end,
      mode = '',
      desc = '[F]ormat buffer',
    },
  },
  opts = {
    notify_on_error = false,
    format_on_save = function(bufnr)
      -- java: only with the project's own Spotless profile, which is what CI checks (the first
      -- format after jdtls starts takes ~1s). Without one, jdtls falls back to Eclipse's built-in
      -- profile (it splits `record Point(int x) {}` braces, for one): <leader>f only.
      if vim.bo[bufnr].filetype == 'java' then
        return spotless_profile(bufnr) and { timeout_ms = 3000, lsp_format = 'fallback' } or nil
      end

      local disable_filetypes = { c = true, cpp = true }
      local lsp_format_opt
      if disable_filetypes[vim.bo[bufnr].filetype] then
        lsp_format_opt = 'never'
      else
        lsp_format_opt = 'fallback'
      end
      return {
        timeout_ms = 500,
        lsp_format = lsp_format_opt,
      }
    end,
    formatters_by_ft = {
      lua = { 'stylua' },
      java = { 'intellij' }, -- when it doesn't apply, <leader>f falls back to jdtls
    },
    formatters = {
      intellij = {
        command = intellij,
        args = function(_, ctx)
          return { '-s', intellij_scheme(ctx.buf), '$FILENAME' }
        end,
        stdin = false, -- formats a temp copy of the buffer in place
        env = function()
          return { IDEA_PROPERTIES = idea_properties() }
        end,
        condition = function(_, ctx)
          return intellij ~= nil and intellij_scheme(ctx.buf) ~= nil
        end,
      },
    },
  },
}

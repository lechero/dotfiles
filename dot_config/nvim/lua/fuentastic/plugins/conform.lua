-- The Eclipse profile jdtls formats java with, when the project's Spotless config names one
-- (see plugins/jdtls.lua).
local function spotless_profile(bufnr)
  local client = vim.lsp.get_clients({ bufnr = bufnr, name = 'jdtls' })[1]
  return client and vim.tbl_get(client.settings, 'java', 'format', 'settings', 'url')
end

return {
  'stevearc/conform.nvim',
  event = { 'BufWritePre' },
  cmd = { 'ConformInfo' },
  keys = {
    {
      '<leader>f',
      function()
        require('conform').format({ async = true, lsp_format = 'fallback' })
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
    },
  },
}

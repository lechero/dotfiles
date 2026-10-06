-- Completion: language servers, Copilot, snippets, buffer words, paths, lazydev in Lua, and the
-- command line. blink.cmp replaces nvim-cmp, whose docs window needs vim.lsp.util.stylize_markdown,
-- which Nvim 0.14 removes.

-- Tab selects the next item only after a word, so it still indents at the start of a line.
local function has_words_before()
  local line, col = unpack(vim.api.nvim_win_get_cursor(0))
  return col ~= 0 and vim.api.nvim_buf_get_lines(0, line - 1, line, true)[1]:sub(col, col):match('%s') == nil
end

return {
  'saghen/blink.cmp',
  version = '1.*', -- a release: comes with its prebuilt (checksummed) fuzzy matcher
  event = { 'InsertEnter', 'CmdlineEnter' },
  dependencies = { 'fang2hou/blink-copilot' },
  opts = {
    keymap = {
      preset = 'none',
      ['<C-Space>'] = { 'show', 'show_documentation', 'hide_documentation' },
      ['<C-n>'] = { 'select_next', 'fallback' },
      ['<C-p>'] = { 'select_prev', 'fallback' },
      ['<CR>'] = { 'accept', 'fallback' },
      ['<Tab>'] = {
        function() -- a Copilot inline suggestion first
          local suggestion = require('copilot.suggestion')
          if suggestion.is_visible() then
            suggestion.accept()
            return true
          end
        end,
        function(cmp)
          if has_words_before() then
            return cmp.select_next()
          end
        end,
        'snippet_forward',
        'fallback',
      },
      ['<S-Tab>'] = { 'snippet_backward', 'fallback' },
    },
    completion = {
      -- the first item is selected, and moving to another one inserts it
      list = { selection = { preselect = true, auto_insert = true } },
      documentation = { auto_show = true, auto_show_delay_ms = 200 },
    },
    sources = {
      default = { 'copilot', 'lazydev', 'lsp', 'snippets', 'buffer', 'path' },
      providers = {
        copilot = { name = 'Copilot', module = 'blink-copilot', score_offset = 100, async = true },
        lazydev = { name = 'LazyDev', module = 'lazydev.integrations.blink', score_offset = 100 },
      },
    },
    cmdline = {
      keymap = { preset = 'cmdline' },
      completion = { menu = { auto_show = true } },
    },
  },
}

-- AEM helpers that aren't tied to one filetype (lua/fuentastic/jcr.lua, doc/aem.md).
vim.api.nvim_create_user_command('AemOpen', function()
  require('fuentastic.jcr').open_in_aem()
end, { desc = "Open this file's node on the local AEM: CRXDE, or the page editor for a page" })

vim.keymap.set('n', '<leader>oa', '<Cmd>AemOpen<CR>', { desc = '[O]pen in [A]EM (CRXDE or the page editor)' })

-- HTL scripts (an AEM content package's jcr_root): gd opens what the line references, since no
-- language server understands HTL (lua/fuentastic/htl.lua).
if vim.api.nvim_buf_get_name(0):find('/jcr_root/', 1, true) then
  vim.keymap.set('n', 'gd', require('fuentastic.htl').goto_reference, { buffer = true, desc = 'HTL: [G]oto referenced file' })
end

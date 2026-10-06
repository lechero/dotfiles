-- AEM content XML (a content package's jcr_root): gd opens what the line references, a component,
-- a node or a clientlib (lua/fuentastic/jcr.lua).
if vim.api.nvim_buf_get_name(0):find('/jcr_root/', 1, true) then
  vim.keymap.set('n', 'gd', require('fuentastic.jcr').goto_reference, { buffer = true, desc = 'AEM: [G]oto referenced node' })
end

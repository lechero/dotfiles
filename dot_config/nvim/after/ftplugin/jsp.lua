-- gc: JSP comments, which the server drops (an HTML comment still runs the tags inside it). In a
-- scriptlet's Java lines, treesitter makes it // (queries/jsp/highlights.scm).
vim.bo.commentstring = '<%-- %s --%>'

-- No language server understands JSP: gd opens what the line includes, imports or uses
-- (lua/fuentastic/jsp.lua).
vim.keymap.set('n', 'gd', require('fuentastic.jsp').goto_reference, { buffer = true, desc = 'JSP: [G]oto referenced file' })

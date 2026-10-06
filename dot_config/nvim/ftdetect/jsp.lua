-- JSP fragments (what <%@ include %> pulls in) and tag files. Tag files only where the JSP spec
-- keeps them: Riot.js components are .tag files too.
vim.filetype.add({
  extension = { jspf = 'jsp', jspx = 'xml', tagx = 'xml' },
  pattern = {
    ['.*/WEB%-INF/tags/.*%.tag'] = 'jsp',
    ['.*/META%-INF/tags/.*%.tag'] = 'jsp',
  },
})

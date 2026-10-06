-- AEM's content XML and the repository behind it (doc/aem.md): gd in a .content.xml opens what the
-- line references (a component, a node, a clientlib), and open_in_aem() opens the current file's node
-- on the local AEM. Resolving is shared with HTL and JSP (lua/fuentastic/refs.lua).
local refs = require('fuentastic.refs')

local M = {}

local function references(line, ctx)
  local found = {}
  for attribute, s, value, e in line:gmatch('([%w:]+)%s*=%s*"()([^"]*)()"') do
    local resolve
    if attribute:match('^sling:resource[ST]') then -- sling:resourceType, sling:resourceSuperType
      resolve = function()
        return refs.find_component(value, ctx)
      end
    elseif value:match('^/') then -- any other repository path: cq:template, fileReference...
      resolve = function()
        return refs.find_node(value, ctx)
      end
    end
    if resolve then
      table.insert(found, { s = s, e = e - 1, label = value, resolve = resolve })
    end
  end
  -- clientlib categories: a dialog's extraClientlibs, a clientlib's embed and dependencies
  for _, attribute in ipairs({ '%a*[Cc]lientlibs', 'embed', 'dependencies' }) do
    refs.scan_categories(found, line, '%f[%w]' .. attribute .. '%s*=%s*"%[?()([^"%]]+)', ctx)
  end
  return found
end

function M.goto_reference()
  refs.goto_reference(references)
end

--------------------------------------------------------------------------------------------------
-- Open in AEM

-- The local author: the project's aem.host and aem.port (its root pom's properties), else
-- localhost:4502.
local function author_url(root)
  local pom = table.concat(vim.fn.readfile(root .. '/pom.xml'), '\n')
  local host = pom:match('<aem%.host>%s*([^<%s]+)%s*</aem%.host>') or 'localhost'
  local port = pom:match('<aem%.port>%s*(%d+)%s*</aem%.port>') or '4502'
  return string.format('http://%s:%s', host, port)
end

-- The repository path of a file under jcr_root: FileVault's escaped names back as they are in the
-- repository (_cq_dialog is cq:dialog), and a .content.xml standing for its folder.
local function repository_path(file)
  local path = file:match('/jcr_root(/.*)$')
  return path and (path:gsub('/%.content%.xml$', ''):gsub('/_(%l+)_', '/%1:'))
end

-- A page (a cq:Page under /content) opens in the page editor, anything else in CRXDE.
function M.open_in_aem()
  local file = vim.api.nvim_buf_get_name(0)
  local path = repository_path(file)
  if not path then
    vim.notify('Not a file in a content package (jcr_root/)', vim.log.levels.WARN)
    return
  end
  local root = vim.fs.root(file, '.git') or vim.fs.root(file, 'pom.xml') or vim.uv.cwd()
  local base = author_url(root)
  local page = file:match('^(.*/jcr_root/content/.*)$')
  local folder = page and (vim.fs.basename(file) == '.content.xml' and vim.fs.dirname(file) or file:gsub('%.[^/.]+$', ''))
  local content = folder and refs.is_file(folder .. '/.content.xml') and table.concat(vim.fn.readfile(folder .. '/.content.xml'), '\n')
  if content and content:find('jcr:primaryType="cq:Page"', 1, true) then
    vim.ui.open(base .. '/editor.html' .. folder:match('/jcr_root(/.*)$') .. '.html')
  else
    vim.ui.open(base .. '/crx/de/index.jsp#' .. path)
  end
end

return M

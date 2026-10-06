-- AEM's content XML (doc/aem.md): gd in a .content.xml opens what the line references, a component, a
-- node or a clientlib. Resolving is shared with HTL and JSP (lua/fuentastic/refs.lua).
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

return M

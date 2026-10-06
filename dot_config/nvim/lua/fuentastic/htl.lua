-- gd for HTL, AEM's component scripts. No language server understands HTL, so this opens what the
-- cursor line references: a resourceType's component, a data-sly-use script or Java class, a
-- data-sly-include, or the template a data-sly-call renders. Resolving is shared with JSP's gd
-- (lua/fuentastic/refs.lua).
local refs = require('fuentastic.refs')

local M = {}

-- data-sly-use and data-sly-include take a plain value or an expression: "${'path' @ opt=1}".
local function literal(value)
  return value:match("^%${%s*'([^']+)'") or value:match('^%${%s*"([^"]+)"') or value
end

local function find_use(ref, ctx)
  if ref:match('%.html$') or ref:match('%.js$') then
    return refs.find_script(ref, ctx)
  end
  return refs.find_class(ref, ctx) or refs.find_script(ref, ctx)
end

local function template(name)
  return [[data-sly-template\.]] .. name .. [[\>]]
end

-- data-sly-call="${var.name}" renders template `name` from the file `data-sly-use.var` loads;
-- data-sly-call="${name}" a template in this file.
local function find_call(var, name, ctx)
  if name == '' then
    return ctx.file, template(var)
  end
  for _, line in ipairs(vim.api.nvim_buf_get_lines(0, 0, -1, false)) do
    local value = line:match('data%-sly%-use%.' .. var .. '%s*=%s*"([^"]+)"')
    if value then
      return refs.find_script(literal(value), ctx), template(name)
    end
  end
end

local function references(line, ctx)
  local found = {}
  refs.scan(found, line, "resourceType%s*=%s*'([^']+)'", function(ref)
    return refs.find_component(ref, ctx)
  end)
  refs.scan(found, line, 'data%-sly%-use[%.%w_]*%s*=%s*"([^"]+)"', function(value)
    return find_use(literal(value), ctx)
  end)
  refs.scan(found, line, 'data%-sly%-include%s*=%s*"([^"]+)"', function(value)
    return refs.find_script(literal(value), ctx)
  end)
  refs.scan(found, line, 'data%-sly%-call%s*=%s*"%${%s*([%w_]+)%.?([%w_]*)', function(var, name)
    return find_call(var, name, ctx)
  end)
  return found
end

function M.goto_reference()
  refs.goto_reference(references)
end

return M

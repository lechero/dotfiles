-- gd for JSP. No language server understands JSP, so this opens what the cursor line references:
-- an included or forwarded page, an imported or used Java class, a custom tag's tag file or class,
-- an EL function's class, and in AEM a cq:include'd script or a resourceType's component. Paths
-- resolve like the server resolves them (lua/fuentastic/refs.lua).
local refs = require('fuentastic.refs')

local M = {}

-- An attribute's value in a tag, `<jsp:include page="...">`. References span just the value.
local function attr(tag, name)
  return '<' .. tag .. '%f[%s].-%f[%w]' .. name .. [[%s*=%s*["']()([^"']+)()]]
end

local function directive_attr(directive, name)
  return attr('%%@%s*' .. directive, name)
end

local function read(path)
  local file = io.open(path)
  if not file then
    return nil
  end
  local text = file:read('*a')
  file:close()
  return text
end

local function buffer_text()
  return table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
end

-- The classes this page imports: <%@ page import="java.util.List, com.example.*" %>.
local function imports()
  local list = {}
  for value in buffer_text():gmatch([[%f[%w]import%s*=%s*["']([^"']+)]]) do
    for name in value:gmatch('[%w_.$*]+') do
      table.insert(list, name)
    end
  end
  return list
end

-- The tag libraries this page declares, by prefix: <%@ taglib prefix="t" tagdir="/WEB-INF/tags" %>.
local function taglibs()
  local libs = {}
  for body in buffer_text():gmatch('<%%@%s*taglib(.-)%%>') do
    local function value(name)
      return body:match('%f[%w]' .. name .. [[%s*=%s*["']([^"']+)]])
    end
    if value('prefix') then
      libs[value('prefix')] = { uri = value('uri'), tagdir = value('tagdir') }
    end
  end
  return libs
end

-- The repo's tag library descriptor (.tld) for a taglib uri, and its text.
local function find_tld(uri, ctx)
  if uri:match('%.tld$') then -- the old style: uri="/WEB-INF/tlds/app.tld"
    local path = refs.find_script(uri, ctx)
    return path, path and read(path)
  end
  local tlds = vim.list_extend({}, ctx.tld)
  if ctx.webapp then
    vim.list_extend(tlds, vim.fn.glob(ctx.webapp .. '/WEB-INF/**/*.tld', false, true))
  end
  for _, path in ipairs(tlds) do
    local tld = read(path)
    if tld and tld:match('<uri>%s*(.-)%s*</uri>') == uri then
      return path, tld
    end
  end
end

-- A custom tag (kinds 'tag' and 'tag-file') or EL function ('function') of a tag library: its tag
-- file or Java class, else its entry in the .tld.
local function find_tag(lib, name, kinds, ctx)
  if lib.tagdir then
    local dir = lib.tagdir:gsub('/$', '')
    return refs.find_script(dir .. '/' .. name .. '.tag', ctx) or refs.find_script(dir .. '/' .. name .. '.tagx', ctx)
  end
  local path, tld = find_tld(lib.uri or '', ctx)
  if not path then
    return nil
  end
  for _, kind in ipairs(kinds) do
    for entry in tld:gmatch('<' .. kind .. '>(.-)</' .. kind .. '>') do
      if entry:match('<name>%s*(.-)%s*</name>') == name then
        local class = entry:match('<tag%-class>%s*(.-)%s*</tag%-class>') or entry:match('<function%-class>%s*(.-)%s*</function%-class>')
        -- a tag file's path is relative to its webapp or jar: /WEB-INF/tags/..., /META-INF/tags/...
        local file = entry:match('<path>%s*(.-)%s*</path>')
        local root = path:match('^(.*)/[WM]E[BT]A?%-INF/')
        local found = class and refs.find_class(class, ctx) or file and root and refs.is_file(root .. file) and root .. file
        if found then
          return found
        end
        return path, [[<name>\s*]] .. name .. [[\s*</name>]]
      end
    end
  end
  return path
end

local function find_package(name, ctx)
  for _, root in ipairs(ctx.java) do
    local dir = root .. '/' .. name:gsub('%.', '/')
    if refs.is_dir(dir) then
      return dir
    end
  end
end

-- A class named in Java code (a scriptlet, an expression, an import list over several lines): the
-- dotted name under the cursor, `com.example.Foo` or a `Foo` this page imports.
local function java_class(found, line, ctx)
  local parser = vim.treesitter.get_parser(0, 'jsp', { error = false })
  if parser then
    parser:parse() -- get_node() only sees what's been parsed
    local node = vim.treesitter.get_node({ ignore_injections = true })
    if node and not vim.list_contains({ 'code', 'attribute_value' }, node:type()) then
      return -- a capitalized word in the markup
    end
  end
  local col = vim.fn.col('.')
  for s, dotted, e in line:gmatch('()([%a_$][%w_$.]*)()') do
    if col >= s and col < e then
      local parts = vim.split(dotted, '.', { plain = true })
      for i, part in ipairs(parts) do
        if part:match('^%u') then
          local name = table.concat(parts, '.', 1, i)
          local resolve
          if i > 1 then
            resolve = function()
              return refs.find_class(name, ctx)
            end
          else
            for _, import in ipairs(imports()) do
              if vim.endswith(import, '.' .. name) then
                resolve = function()
                  return refs.find_class(import, ctx)
                end
              elseif not resolve and vim.endswith(import, '.*') then
                local path = refs.find_class(import:sub(1, -2) .. name, ctx)
                resolve = path and function()
                  return path
                end
              end
            end
          end
          if resolve then
            table.insert(found, { s = s, e = e - 1, label = name, resolve = resolve })
          end
          return
        end
      end
      return
    end
  end
end

local function references(line, ctx)
  local found = {}
  local function script(path)
    return refs.find_script(path, ctx)
  end
  local function class(name)
    return refs.find_class(name, ctx)
  end

  refs.scan(found, line, directive_attr('include', 'file'), script)
  refs.scan(found, line, attr('jsp:%a+', 'page'), script) -- jsp:include, jsp:forward
  refs.scan(found, line, attr('c:import', 'url'), script)
  refs.scan(found, line, attr('%a+:%a+', 'script'), script) -- AEM: cq:include, sling:call
  refs.scan(found, line, [[%f[%w]resourceType%s*=%s*["']()([^"']+)()]], function(resource_type)
    return refs.find_component(resource_type, ctx)
  end)

  refs.scan(found, line, attr('jsp:useBean', 'class'), class)
  refs.scan(found, line, attr('jsp:useBean', 'type'), class)
  refs.scan(found, line, attr('%a+:adaptTo', 'adaptTo'), class) -- sling:adaptTo
  refs.scan_categories(found, line, attr('%a+:includeClientLib', 'categories'), ctx) -- cq:, ui:
  -- import="java.util.List, com.example.*": each class (or package) on its own
  for s, value in line:gmatch([[%f[%w]import%s*=%s*["']()([^"']+)]]) do
    for offset, name in value:gmatch('()([%w_.$*]+)') do
      table.insert(found, {
        s = s + offset - 1,
        e = s + offset + #name - 2,
        label = name,
        resolve = function()
          return name:match('%.%*$') and find_package(name:sub(1, -3), ctx) or class(name)
        end,
      })
    end
  end

  refs.scan(found, line, directive_attr('taglib', 'uri'), function(uri)
    return (find_tld(uri, ctx))
  end)
  refs.scan(found, line, directive_attr('taglib', 'tagdir'), function(dir)
    return ctx.webapp and refs.is_dir(ctx.webapp .. dir) and ctx.webapp .. dir or nil
  end)
  -- Custom tags and EL functions of the tag libraries this page declares. Those an included page
  -- declares (global.jsp in AEM) aren't known here.
  for prefix, lib in pairs(taglibs()) do
    local tag = vim.pesc(prefix) .. ':[%w_%-]+'
    refs.scan(found, line, '</?()(' .. tag .. ')()', function(name)
      return find_tag(lib, name:sub(#prefix + 2), { 'tag', 'tag%-file' }, ctx)
    end)
    refs.scan(found, line, '()(' .. tag .. ')()%s*%(', function(name)
      return find_tag(lib, name:sub(#prefix + 2), { 'function' }, ctx)
    end)
  end

  java_class(found, line, ctx)

  -- Values computed when the page runs, page="${view}", can't be followed.
  return vim.tbl_filter(function(ref)
    return not ref.label:find('[$#]{') and not ref.label:find('<%', 1, true)
  end, found)
end

function M.goto_reference()
  refs.goto_reference(references)
end

return M

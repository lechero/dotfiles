-- gd for HTL, AEM's component scripts. No language server understands HTL, so this opens what the
-- cursor line references: a resourceType's component, a data-sly-use script or Java class, a
-- data-sly-include, or the template a data-sly-call renders.
local M = {}

local function is_file(path)
  local stat = vim.uv.fs_stat(path)
  return stat and stat.type == 'file'
end

local function is_dir(path)
  local stat = vim.uv.fs_stat(path)
  return stat and stat.type == 'directory'
end

-- The repo's content packages (ui.apps, ui.content, ...) and Java source folders, per repo.
local sources_by_repo = {}
local function sources(file)
  local repo = vim.fs.root(file, '.git') or vim.fs.dirname(file)
  if not sources_by_repo[repo] then
    local found = { jcr = {}, java = {} }
    for _, depth in ipairs({ '', '/*', '/*/*' }) do
      vim.list_extend(found.jcr, vim.fn.glob(repo .. depth .. '/src/main/content/jcr_root', false, true))
      vim.list_extend(found.java, vim.fn.glob(repo .. depth .. '/src/main/java', false, true))
    end
    sources_by_repo[repo] = found
  end
  return sources_by_repo[repo]
end

-- A script path, resolved like Sling does: relative to the current script, then /apps and /libs.
local function find_script(ref, ctx)
  local candidates = {}
  if vim.startswith(ref, '/') then
    for _, root in ipairs(ctx.jcr) do
      table.insert(candidates, root .. ref)
    end
  else
    table.insert(candidates, ctx.dir .. '/' .. ref)
    for _, root in ipairs(ctx.jcr) do
      table.insert(candidates, root .. '/apps/' .. ref)
      table.insert(candidates, root .. '/libs/' .. ref)
    end
  end
  for _, path in ipairs(candidates) do
    if is_file(path) then
      return vim.fs.normalize(path)
    end
  end
end

-- A resource type's component folder, opened at its main script (<name>.html).
local function find_component(ref, ctx)
  local paths = vim.startswith(ref, '/') and { ref } or { '/apps/' .. ref, '/libs/' .. ref }
  for _, root in ipairs(ctx.jcr) do
    for _, path in ipairs(paths) do
      local dir = root .. path
      if is_dir(dir) then
        for _, name in ipairs({ vim.fs.basename(dir) .. '.html', '.content.xml' }) do
          if is_file(dir .. '/' .. name) then
            return dir .. '/' .. name
          end
        end
        return dir
      end
    end
  end
end

local function find_class(ref, ctx)
  local relative = ref:gsub('%.', '/') .. '.java'
  for _, root in ipairs(ctx.java) do
    if is_file(root .. '/' .. relative) then
      return root .. '/' .. relative
    end
  end
end

-- data-sly-use and data-sly-include take a plain value or an expression: "${'path' @ opt=1}".
local function literal(value)
  return value:match("^%${%s*'([^']+)'") or value:match('^%${%s*"([^"]+)"') or value
end

local function find_use(ref, ctx)
  if ref:match('%.html$') or ref:match('%.js$') then
    return find_script(ref, ctx)
  end
  return find_class(ref, ctx) or find_script(ref, ctx)
end

-- data-sly-call="${var.name}" renders template `name` from the file `data-sly-use.var` loads;
-- data-sly-call="${name}" a template in this file.
local function find_call(var, name, ctx)
  if name == '' then
    return vim.api.nvim_buf_get_name(0), var
  end
  for _, line in ipairs(vim.api.nvim_buf_get_lines(0, 0, -1, false)) do
    local value = line:match('data%-sly%-use%.' .. var .. '%s*=%s*"([^"]+)"')
    if value then
      return find_script(literal(value), ctx), name
    end
  end
end

-- The references on a line, each with its span, the text shown when it can't be found, and a resolver.
local function references(line, ctx)
  local refs = {}
  local function scan(pattern, resolve)
    local init = 1
    while true do
      local s, e, a, b = line:find(pattern, init)
      if not s then
        return
      end
      table.insert(refs, {
        s = s,
        e = e,
        label = b and b ~= '' and a .. '.' .. b or a,
        resolve = function()
          return resolve(a, b)
        end,
      })
      init = e + 1
    end
  end

  scan("resourceType%s*=%s*'([^']+)'", function(ref)
    return find_component(ref, ctx)
  end)
  scan('data%-sly%-use[%.%w_]*%s*=%s*"([^"]+)"', function(value)
    return find_use(literal(value), ctx)
  end)
  scan('data%-sly%-include%s*=%s*"([^"]+)"', function(value)
    return find_script(literal(value), ctx)
  end)
  scan('data%-sly%-call%s*=%s*"%${%s*([%w_]+)%.?([%w_]*)', function(var, name)
    return find_call(var, name, ctx)
  end)
  table.sort(refs, function(x, y)
    return x.s < y.s
  end)
  return refs
end

function M.goto_reference()
  local file = vim.api.nvim_buf_get_name(0)
  local ctx = vim.tbl_extend('force', sources(file), { dir = vim.fs.dirname(file) })
  local refs = references(vim.api.nvim_get_current_line(), ctx)
  if #refs == 0 then
    vim.cmd('normal! gd')
    return
  end

  -- the reference under the cursor, else the line's first
  local col = vim.fn.col('.')
  local ref = refs[1]
  for _, r in ipairs(refs) do
    if col >= r.s and col <= r.e then
      ref = r
    end
  end

  local path, template = ref.resolve()
  if not path then
    vim.notify('Not in this repo: ' .. ref.label, vim.log.levels.WARN)
    return
  end
  vim.cmd("normal! m'")
  if path ~= file then
    vim.cmd.edit(vim.fn.fnameescape(path))
  end
  if template then
    vim.fn.search([[data-sly-template\.]] .. template .. [[\>]], 'cw')
  end
end

return M

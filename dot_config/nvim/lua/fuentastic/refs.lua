-- What HTL's and JSP's gd share (lua/fuentastic/htl.lua, lua/fuentastic/jsp.lua). No language
-- server understands those templates, so gd finds the references on the cursor line itself and
-- resolves them the way the server would: Sling's /apps and /libs in an AEM repo, the webapp's
-- root in a Java webapp.
local M = {}

function M.is_file(path)
  local stat = vim.uv.fs_stat(path)
  return stat and stat.type == 'file'
end

function M.is_dir(path)
  local stat = vim.uv.fs_stat(path)
  return stat and stat.type == 'directory'
end

-- The repo's content packages (ui.apps, ui.content, ...), Java source folders and tag library
-- descriptors, per repo.
local sources_by_repo = {}
local function sources(file)
  local repo = vim.fs.root(file, '.git') or vim.fs.dirname(file)
  if not sources_by_repo[repo] then
    local found = { repo = repo, jcr = {}, java = {}, tld = {} }
    for _, depth in ipairs { '', '/*', '/*/*' } do
      local module = repo .. depth .. '/src/main'
      vim.list_extend(found.jcr, vim.fn.glob(module .. '/content/jcr_root', false, true))
      vim.list_extend(found.java, vim.fn.glob(module .. '/java', false, true))
      vim.list_extend(found.tld, vim.fn.glob(module .. '/webapp/WEB-INF/**/*.tld', false, true))
      vim.list_extend(found.tld, vim.fn.glob(module .. '/resources/META-INF/**/*.tld', false, true))
    end
    sources_by_repo[repo] = found
  end
  return sources_by_repo[repo]
end

-- The root of the webapp a file is in: the folder holding WEB-INF (src/main/webapp in a Maven war).
local function webapp(file)
  local root = vim.fs.root(file, 'WEB-INF')
  if root then
    return root
  end
  for dir in vim.fs.parents(file) do
    if vim.fs.basename(dir) == 'webapp' then
      return dir
    end
  end
end

-- Where the references in `file` resolve.
function M.context(file)
  return vim.tbl_extend('force', sources(file), { file = file, dir = vim.fs.dirname(file), webapp = webapp(file) })
end

-- A script path, resolved like the server does: a relative one next to the current script, then
-- under /apps and /libs; an absolute one from the webapp's root, or a content package's jcr_root.
function M.find_script(ref, ctx)
  local candidates = {}
  if vim.startswith(ref, '/') then
    if ctx.webapp then
      table.insert(candidates, ctx.webapp .. ref)
    end
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
    if M.is_file(path) then
      return vim.fs.normalize(path)
    end
  end
end

-- A resource type's component folder, opened at its main script (<name>.html or <name>.jsp).
function M.find_component(ref, ctx)
  local paths = vim.startswith(ref, '/') and { ref } or { '/apps/' .. ref, '/libs/' .. ref }
  for _, root in ipairs(ctx.jcr) do
    for _, path in ipairs(paths) do
      local dir = root .. path
      if M.is_dir(dir) then
        local name = vim.fs.basename(dir)
        for _, script in ipairs { name .. '.html', name .. '.jsp', '.content.xml' } do
          if M.is_file(dir .. '/' .. script) then
            return dir .. '/' .. script
          end
        end
        return dir
      end
    end
  end
end

-- A repository path's node: its folder's .content.xml (or the folder), or a file. Names with a
-- namespace are escaped on disk the FileVault way: cq:dialog is _cq_dialog.
function M.find_node(path, ctx)
  local escaped = path:gsub('/([%l]+):', '/_%1_'):gsub('/$', '')
  for _, root in ipairs(ctx.jcr) do
    local base = root .. escaped
    for _, candidate in ipairs { base .. '/.content.xml', base .. '.xml', base } do
      if M.is_file(candidate) then
        return candidate
      end
    end
    if M.is_dir(base) then
      return base
    end
  end
end

-- The repo's clientlib folders by category: the .content.xml of each cq:ClientLibraryFolder, found
-- with git grep (fast, and skips build output).
local clientlibs_by_repo = {}
function M.find_clientlib(category, ctx)
  if not clientlibs_by_repo[ctx.repo] then
    local index = {}
    local grep = vim.system({ 'git', 'grep', '-l', 'cq:ClientLibraryFolder', '--', '*.content.xml' }, { cwd = ctx.repo, text = true }):wait()
    for _, relative in ipairs(vim.split(grep.stdout or '', '\n', { trimempty = true })) do
      local file = ctx.repo .. '/' .. relative
      local categories = table.concat(vim.fn.readfile(file), '\n'):match '%scategories="%[?([^"%]]*)'
      for name in (categories or ''):gmatch '[^,%s]+' do
        index[name] = index[name] or file
      end
    end
    clientlibs_by_repo[ctx.repo] = index
  end
  return clientlibs_by_repo[ctx.repo][category]
end

-- Adds a reference for each clientlib category in a list on the line, spanning the category: the
-- values of `categories='a'`, `categories=['a', 'b']`, `extraClientlibs="[a,b]"` and the like.
function M.scan_categories(found, line, pattern, ctx)
  for start, list in line:gmatch(pattern) do
    for offset, name in list:gmatch '()([%w_][%w_.%-]*)' do
      table.insert(found, {
        s = start + offset - 1,
        e = start + offset + #name - 2,
        label = 'clientlib ' .. name,
        resolve = function()
          return M.find_clientlib(name, ctx)
        end,
      })
    end
  end
end

-- A fully qualified class name's .java file, in any module of the repo.
function M.find_class(ref, ctx)
  local relative = ref:gsub('%$.*', ''):gsub('%.', '/') .. '.java' -- Outer$Inner is in Outer.java
  for _, root in ipairs(ctx.java) do
    if M.is_file(root .. '/' .. relative) then
      return root .. '/' .. relative
    end
  end
end

-- Adds a reference for each match of `pattern` on `line`, resolved by calling `resolve` with the
-- pattern's captures. A reference spans its whole match, or only what the pattern brackets with
-- position captures: "()(...)()". It's labelled with its captures, `a.b`, when it can't be found.
function M.scan(found, line, pattern, resolve)
  local init = 1
  while true do
    local match = { line:find(pattern, init) }
    if not match[1] then
      return
    end
    local s, e, captures = match[1], match[2], { unpack(match, 3) }
    if type(captures[1]) == 'number' then
      s, e = captures[1], captures[3] - 1
      captures = { captures[2], unpack(captures, 4) }
    end
    local a, b = captures[1], captures[2]
    table.insert(found, {
      s = s,
      e = e,
      label = b and b ~= '' and a .. '.' .. b or a,
      resolve = function()
        return resolve(unpack(captures))
      end,
    })
    init = match[2] + 1
  end
end

-- gd: opens the reference under the cursor, else the line's first, at the pattern its resolver
-- returns with the path (a vim regex). Vim's own gd on a line without references.
-- `references(line, ctx)` lists the line's references, as `scan` makes them.
function M.goto_reference(references)
  local file = vim.api.nvim_buf_get_name(0)
  local found = references(vim.api.nvim_get_current_line(), M.context(file))
  if #found == 0 then
    vim.cmd 'normal! gd'
    return
  end

  table.sort(found, function(x, y)
    return x.s < y.s
  end)
  local col = vim.fn.col '.'
  local ref = found[1]
  for _, r in ipairs(found) do
    if col >= r.s and col <= r.e then
      ref = r
    end
  end

  local path, pattern = ref.resolve()
  if not path then
    vim.notify('Not in this repo: ' .. ref.label, vim.log.levels.WARN)
    return
  end
  vim.cmd "normal! m'"
  if path ~= file then
    vim.cmd.edit(vim.fn.fnameescape(path))
  end
  if pattern then
    vim.fn.search(pattern, 'cw')
  end
end

return M

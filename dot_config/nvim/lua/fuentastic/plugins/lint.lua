-- The linters that aren't language servers, each run the way the project runs it (doc/linting.md):
-- checkstyle as its Maven build does, Stylelint and html-validate with its own binaries and configs.
-- Nothing runs in a project that doesn't use the linter.

local function read(path)
  local file = io.open(path)
  if not file then
    return nil
  end
  local text = file:read '*a'
  file:close()
  return text
end

local function mtime(path)
  local stat = vim.uv.fs_stat(path)
  return stat and stat.mtime.sec or 0
end

--------------------------------------------------------------------------------------------------
-- checkstyle

-- What the maven-checkstyle-plugin checks for a file: the config and suppressions of the nearest pom
-- up from it that configures them, and the module's src/main/java (Maven skips tests) minus its
-- <excludes>. nil when the build doesn't run checkstyle on the file.
local function checkstyle_setup(file)
  local module = vim.fs.root(file, 'pom.xml')
  local sources = module and module .. '/src/main/java/'
  if not sources or not vim.startswith(file, sources) then
    return nil
  end
  for dir in vim.fs.parents(file) do
    local plugin = (read(dir .. '/pom.xml') or ''):match '<artifactId>maven%-checkstyle%-plugin</artifactId>(.-)</plugin>'
    local config = plugin and plugin:match '<configLocation>%s*(.-)%s*</configLocation>'
    if config then
      local function path(value)
        value = value:gsub('^%${project%.basedir}/?', '')
        return vim.startswith(value, '/') and value or vim.fs.normalize(dir .. '/' .. value)
      end
      for pattern in (plugin:match '<excludes>%s*(.-)%s*</excludes>' or ''):gmatch '[^,%s]+' do
        if vim.glob.to_lpeg(pattern):match(file:sub(#sources + 1)) then
          return nil
        end
      end
      local suppressions = plugin:match '<suppressionsLocation>%s*(.-)%s*</suppressionsLocation>'
      -- a classpath resource (google_checks.xml, a shared config jar) can't be read from here
      return vim.uv.fs_stat(path(config)) and { config = path(config), suppressions = suppressions and path(suppressions) } or nil
    end
    if vim.uv.fs_stat(dir .. '/.git') then
      return nil
    end
  end
end

-- The checkstyle CLI has no suppressions option: the Maven plugin adds a SuppressionFilter to the
-- config. So does this, into a copy in the cache, rewritten when either file changes.
local function checkstyle_config(setup)
  if not setup.suppressions then
    return setup.config
  end
  local copy = vim.fn.stdpath 'cache' .. '/checkstyle/' .. vim.fn.sha256(setup.config .. setup.suppressions):sub(1, 12) .. '.xml'
  if mtime(copy) < math.max(mtime(setup.config), mtime(setup.suppressions)) then
    local filter = string.format('\n  <module name="SuppressionFilter"><property name="file" value="%s"/></module>', setup.suppressions)
    local wrapped = (read(setup.config) or ''):gsub('(<module%s+name%s*=%s*"Checker"%s*>)', '%1' .. filter, 1)
    vim.fn.mkdir(vim.fs.dirname(copy), 'p')
    vim.fn.writefile(vim.split(wrapped, '\n'), copy)
  end
  return copy
end

local checkstyle_severities = {
  ERROR = vim.diagnostic.severity.ERROR,
  WARN = vim.diagnostic.severity.WARN,
  INFO = vim.diagnostic.severity.INFO,
}

local checkstyle = {
  cmd = 'checkstyle', -- Mason's
  stdin = false,
  append_fname = true,
  args = {
    '-c',
    function()
      return checkstyle_config(checkstyle_setup(vim.api.nvim_buf_get_name(0)))
    end,
  },
  ignore_exitcode = true, -- it exits with the number of violations
  -- [WARN] /path/File.java:12:5: Line is longer than 160 characters (found 171). [LineLength]
  parser = function(output)
    local diagnostics = {}
    for line in output:gmatch '[^\n]+' do
      local severity, lnum, rest = line:match '^%[(%u+)%] .-:(%d+):(.*)$'
      if severity then
        local col, message = rest:match '^(%d+): (.*)$'
        message = message or vim.trim(rest)
        table.insert(diagnostics, {
          lnum = tonumber(lnum) - 1,
          col = col and tonumber(col) - 1 or 0,
          severity = checkstyle_severities[severity] or vim.diagnostic.severity.WARN,
          message = message:gsub('%s*%[%w+%]$', ''),
          code = message:match '%[(%w+)%]$',
          source = 'checkstyle',
        })
      end
    end
    return diagnostics
  end,
}

--------------------------------------------------------------------------------------------------
-- Stylelint and html-validate: the project's own, next to its config

local function nearest(buf, names)
  return vim.fs.find(names, { upward = true, path = vim.fs.dirname(vim.api.nvim_buf_get_name(buf)), type = 'file' })[1]
end

local stylelint_configs = {
  '.stylelintrc',
  '.stylelintrc.json',
  '.stylelintrc.yaml',
  '.stylelintrc.yml',
  '.stylelintrc.js',
  '.stylelintrc.cjs',
  '.stylelintrc.mjs',
  'stylelint.config.js',
  'stylelint.config.cjs',
  'stylelint.config.mjs',
}
local html_validate_configs = { '.htmlvalidate.js', '.htmlvalidate.cjs', '.htmlvalidate.mjs', '.htmlvalidate.json' }

-- html-validate's JSON: [{ messages: [{ ruleId, severity (1 warning, 2 error), message, line, column, size }] }]
-- On the saved file, not stdin: only a path gets the project's .htmlvalidateignore applied.
local html_validate = {
  cmd = function()
    return nearest(0, 'node_modules/.bin/html-validate')
  end,
  stdin = false,
  append_fname = true,
  args = { '--formatter', 'json' },
  ignore_exitcode = true,
  parser = function(output)
    local ok, results = pcall(vim.json.decode, output)
    local diagnostics = {}
    for _, result in ipairs(ok and type(results) == 'table' and results or {}) do
      for _, m in ipairs(result.messages or {}) do
        table.insert(diagnostics, {
          lnum = (m.line or 1) - 1,
          col = (m.column or 1) - 1,
          end_col = (m.column or 1) - 1 + (m.size or 1),
          severity = m.severity == 2 and vim.diagnostic.severity.ERROR or vim.diagnostic.severity.WARN,
          message = m.message,
          code = m.ruleId,
          source = 'html-validate',
        })
      end
    end
    return diagnostics
  end,
}

-- The linters that apply to a buffer, and the directory each runs in.
local function linters_for(buf)
  local file = vim.api.nvim_buf_get_name(buf)
  local ft = vim.bo[buf].filetype
  if ft == 'java' then
    return checkstyle_setup(file) and vim.fn.executable 'checkstyle' == 1 and { checkstyle = vim.fs.dirname(file) } or {}
  end
  if ft == 'css' or ft == 'scss' or ft == 'less' then
    local config = nearest(buf, stylelint_configs)
    return config and nearest(buf, 'node_modules/.bin/stylelint') and { stylelint = vim.fs.dirname(config) } or {}
  end
  if ft == 'html' and file:find('/jcr_root/', 1, true) then -- HTL
    local config = nearest(buf, html_validate_configs)
    return config and nearest(buf, 'node_modules/.bin/html-validate') and { html_validate = vim.fs.dirname(config) } or {}
  end
  return {}
end

return {
  'mfussenegger/nvim-lint',
  event = { 'BufReadPost', 'BufNewFile' },
  config = function()
    local lint = require 'lint'
    lint.linters.checkstyle = checkstyle
    lint.linters.html_validate = html_validate
    lint.linters.stylelint.cmd = function()
      return nearest(0, 'node_modules/.bin/stylelint')
    end

    -- On opening (FileType: BufReadPost comes before the filetype is known) and on saving:
    -- checkstyle starts a JVM (~0.5s), so not on every change.
    vim.api.nvim_create_autocmd({ 'FileType', 'BufWritePost' }, {
      group = vim.api.nvim_create_augroup('fuentastic-lint', { clear = true }),
      callback = function(args)
        for name, cwd in pairs(linters_for(args.buf)) do
          lint.try_lint(name, { cwd = cwd })
        end
      end,
    })
  end,
}

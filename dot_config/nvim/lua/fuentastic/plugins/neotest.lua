-- Jest tests from the editor (doc/testing.md), on the keys Java's tests use. They run through the
-- project's own `npm test` script, so its environment (TZ, NODE_ENV) applies as in CI.

local function package_dir(path)
  return vim.fs.root(path, 'package.json')
end

local function read_package(dir)
  local file = dir and io.open(dir .. '/package.json')
  if not file then
    return nil
  end
  local ok, package = pcall(vim.json.decode, file:read('*a'))
  file:close()
  return ok and type(package) == 'table' and package or nil
end

-- A test file of a package that depends on Jest. neotest-jest's own check also reads the
-- package.json in nvim's cwd, and warns "cannot read package.json" for every file it looks at when
-- the cwd is a repo root without one (an AEM project: ui.apps has it).
local jest_packages = {}
local function is_jest_test(file)
  if not require('neotest-jest.util').defaultTestFileMatcher(file) then
    return false
  end
  local dir = package_dir(file)
  if dir and jest_packages[dir] == nil then
    local package = read_package(dir) or {}
    jest_packages[dir] = vim.tbl_get(package, 'devDependencies', 'jest') ~= nil or vim.tbl_get(package, 'dependencies', 'jest') ~= nil
  end
  return dir ~= nil and jest_packages[dir]
end

-- `npm test --` when the project's test script runs Jest, without its coverage (slow, and no use for
-- one test); otherwise the project's own jest binary.
local function jest_command(path)
  local dir = package_dir(path)
  local test = vim.tbl_get(read_package(dir) or {}, 'scripts', 'test') or ''
  if test:match('%f[%w]jest%f[%W]') then
    return 'npm test -- --coverage=false'
  end
  return dir and vim.uv.fs_stat(dir .. '/node_modules/.bin/jest') and dir .. '/node_modules/.bin/jest' or 'npx jest'
end

local function map_keys(buf)
  local neotest = require('neotest')
  local function map(keys, fn, desc)
    vim.keymap.set('n', keys, fn, { buffer = buf, desc = 'Test: ' .. desc })
  end
  map('<leader>tm', neotest.run.run, 'nearest [M]ethod / test')
  map('<leader>tc', function()
    neotest.run.run(vim.api.nvim_buf_get_name(buf))
  end, 'this file ([C]lass)')
  map('<leader>tl', neotest.run.run_last, 'run [L]ast again')
  map('<leader>to', function()
    neotest.output.open({ enter = true, auto_close = true })
  end, '[O]utput of the nearest test')
  map('<leader>ts', neotest.summary.toggle, '[S]ummary of the test files')
  map('<leader>tw', function()
    neotest.watch.toggle(vim.api.nvim_buf_get_name(buf))
  end, '[W]atch this file')
end

return {
  'nvim-neotest/neotest',
  ft = { 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'vue' },
  dependencies = {
    'nvim-neotest/nvim-nio',
    'nvim-lua/plenary.nvim',
    'nvim-neotest/neotest-jest',
  },
  config = function()
    require('neotest').setup({
      adapters = {
        require('neotest-jest')({
          jestCommand = jest_command,
          cwd = package_dir,
          isTestFile = is_jest_test,
          env = { CI = 'true' }, -- no interactive prompts, no watch mode
        }),
      },
    })

    local group = vim.api.nvim_create_augroup('fuentastic-neotest', { clear = true })
    vim.api.nvim_create_autocmd('FileType', {
      group = group,
      pattern = { 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'vue' },
      callback = function(args)
        if is_jest_test(vim.api.nvim_buf_get_name(args.buf)) then
          map_keys(args.buf)
        end
      end,
    })
    -- the buffer that loaded this plugin
    local buf = vim.api.nvim_get_current_buf()
    if is_jest_test(vim.api.nvim_buf_get_name(buf)) then
      map_keys(buf)
    end
  end,
}

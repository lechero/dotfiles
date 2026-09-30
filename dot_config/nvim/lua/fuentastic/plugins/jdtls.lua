local root_markers = {
  { 'mvnw', 'gradlew', 'settings.gradle', 'settings.gradle.kts', '.git' },
  { 'pom.xml', 'build.gradle', 'build.gradle.kts', 'build.xml' },
}

local mason = vim.fn.stdpath('data') .. '/mason'

local function jdk_major(home)
  local file = io.open(home .. '/release')
  if not file then
    return nil
  end
  local version = file:read('*a'):match('JAVA_VERSION="([^"]+)"')
  file:close()
  if not version then
    return nil
  end
  local first, second = version:match('^(%d+)%.?(%d*)')
  if first == '1' then
    return tonumber(second)
  end
  return tonumber(first)
end

-- Every JDK SDKMAN or a macOS installer put on this machine, one per major version.
-- The one JAVA_HOME points at wins its major, so `sdk use` / `sdk default` carry over.
local function installed_jdks()
  local java_home = vim.env.JAVA_HOME and vim.uv.fs_realpath(vim.env.JAVA_HOME)
  local homes = vim.fn.glob(vim.env.HOME .. '/.sdkman/candidates/java/*', false, true)
  vim.list_extend(homes, vim.fn.glob('/Library/Java/JavaVirtualMachines/*/Contents/Home', false, true))

  local by_major = {}
  for _, home in ipairs(homes) do
    local real = vim.uv.fs_realpath(home)
    local major = real and vim.fs.basename(home) ~= 'current' and jdk_major(real)
    if major and (not by_major[major] or real == java_home) then
      by_major[major] = { home = real, major = major, is_java_home = real == java_home }
    end
  end
  return vim.tbl_values(by_major)
end

-- jdtls itself needs Java 21+, whatever the project targets.
local function server_jdk(jdks)
  local best
  for _, jdk in ipairs(jdks) do
    if jdk.major >= 21 then
      if jdk.is_java_home then
        return jdk
      end
      if not best or jdk.major > best.major then
        best = jdk
      end
    end
  end
  return best
end

-- Maps each JDK to the execution environment a pom/gradle file asks for (release 17 -> JavaSE-17).
local function runtimes(jdks)
  local list = {}
  for _, jdk in ipairs(jdks) do
    table.insert(list, {
      name = 'JavaSE-' .. (jdk.major <= 8 and '1.' .. jdk.major or jdk.major),
      path = jdk.home,
      default = jdk.is_java_home or nil,
    })
  end
  return list
end

-- java-test ships these next to its bundles, but launches them itself (the last one is
-- an unversioned duplicate of the plugin jar), so they must not be loaded into jdtls.
local not_bundles = {
  ['com.microsoft.java.test.plugin.jar'] = true,
  ['com.microsoft.java.test.runner-jar-with-dependencies.jar'] = true,
  ['jacocoagent.jar'] = true,
}

-- jdtls extensions installed by mason; they only load when the server starts.
local function bundles()
  local list = vim.fn.glob(mason .. '/share/java-debug-adapter/com.microsoft.java.debug.plugin-*.jar', false, true)
  for _, jar in ipairs(vim.fn.glob(mason .. '/share/java-test/*.jar', false, true)) do
    local name = vim.fs.basename(jar)
    -- A jar jdtls already ships under the same name (asm) is the same bundle; OSGi rejects a second copy.
    if not not_bundles[name] and not vim.uv.fs_stat(mason .. '/share/jdtls/plugins/' .. name) then
      table.insert(list, jar)
    end
  end
  return list
end

local function on_attach(_, bufnr)
  local jdtls = require('jdtls')
  local function map(mode, keys, func, desc)
    vim.keymap.set(mode, keys, func, { buffer = bufnr, desc = 'Java: ' .. desc })
  end

  map('n', '<leader>co', jdtls.organize_imports, '[O]rganize imports')
  map('n', '<leader>cxv', jdtls.extract_variable, 'E[x]tract [V]ariable')
  map('x', '<leader>cxv', '<Esc><Cmd>lua require("jdtls").extract_variable({ visual = true })<CR>', 'E[x]tract [V]ariable')
  map('n', '<leader>cxc', jdtls.extract_constant, 'E[x]tract [C]onstant')
  map('x', '<leader>cxc', '<Esc><Cmd>lua require("jdtls").extract_constant({ visual = true })<CR>', 'E[x]tract [C]onstant')
  map('x', '<leader>cxm', '<Esc><Cmd>lua require("jdtls").extract_method({ visual = true })<CR>', 'E[x]tract [M]ethod')
  map('n', 'gS', jdtls.super_implementation, '[G]oto [S]uper implementation')

  -- Tests run under the debugger (breakpoints work); failures land in the quickfix list.
  map('n', '<leader>tc', jdtls.test_class, '[T]est [C]lass')
  map('n', '<leader>tm', jdtls.test_nearest_method, '[T]est nearest [M]ethod')
  map('n', '<leader>tp', jdtls.pick_test, '[T]est [P]ick')
  map('n', '<leader>tl', require('dap').run_last, 'Re-run [L]ast test or debug session')
end

local function attach()
  -- jdt:// class files opened from a definition jump are attached by nvim-jdtls itself
  if not vim.startswith(vim.uri_from_bufnr(0), 'file://') then
    return
  end

  local jdks = installed_jdks()
  local jdk = server_jdk(jdks)
  if not jdk then
    vim.notify('jdtls needs a JDK 21+; install one with `sdk install java 21-tem`', vim.log.levels.ERROR)
    return
  end

  local root = vim.fs.root(0, root_markers) or vim.fs.dirname(vim.api.nvim_buf_get_name(0))
  local workspace = vim.fn.stdpath('cache') .. '/jdtls/' .. vim.fs.basename(root) .. '-' .. vim.fn.sha256(root):sub(1, 8)

  local capabilities = vim.lsp.protocol.make_client_capabilities()
  capabilities = vim.tbl_deep_extend('force', capabilities, require('cmp_nvim_lsp').default_capabilities())

  require('jdtls').start_or_attach({
    name = 'jdtls',
    cmd = { mason .. '/bin/jdtls', '--jvm-arg=-javaagent:' .. mason .. '/share/jdtls/lombok.jar', '-data', workspace },
    cmd_env = { JAVA_HOME = jdk.home },
    root_dir = root,
    capabilities = capabilities,
    on_attach = on_attach,
    init_options = {
      bundles = bundles(),
      extendedClientCapabilities = require('jdtls').extendedClientCapabilities,
    },
    settings = {
      java = {
        configuration = {
          runtimes = runtimes(jdks),
          updateBuildConfiguration = 'automatic',
        },
        eclipse = { downloadSources = true },
        maven = { downloadSources = true },
        references = { includeDecompiledSources = true },
        signatureHelp = { enabled = true },
        sources = {
          organizeImports = { starThreshold = 9999, staticStarThreshold = 9999 },
        },
      },
    },
  })
end

return {
  'mfussenegger/nvim-jdtls',
  ft = 'java',
  dependencies = { 'mfussenegger/nvim-dap' },
  config = function()
    -- Registers the `java` adapter, so <F5> offers every main class in the project. Only the
    -- first call counts, and nvim-jdtls makes its own (without these options) on LspAttach.
    require('jdtls').setup_dap({ hotcodereplace = 'auto' })

    vim.api.nvim_create_autocmd('FileType', {
      group = vim.api.nvim_create_augroup('fuentastic-jdtls', { clear = true }),
      pattern = 'java',
      callback = attach,
    })
  end,
}

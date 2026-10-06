local root_markers = {
  { 'mvnw', 'gradlew', 'settings.gradle', 'settings.gradle.kts', '.git' },
  { 'pom.xml', 'build.gradle', 'build.gradle.kts', 'build.xml' },
}

local mason = vim.fn.stdpath 'data' .. '/mason'

local function jdk_major(home)
  local file = io.open(home .. '/release')
  if not file then
    return nil
  end
  local version = file:read('*a'):match 'JAVA_VERSION="([^"]+)"'
  file:close()
  if not version then
    return nil
  end
  local first, second = version:match '^(%d+)%.?(%d*)'
  if first == '1' then
    return tonumber(second)
  end
  return tonumber(first)
end

-- Where SDKMAN, macOS installers, IntelliJ downloads, Homebrew and Linux distros put JDKs.
local jdk_globs = {
  vim.env.HOME .. '/.sdkman/candidates/java/*',
  '/Library/Java/JavaVirtualMachines/*/Contents/Home',
  vim.env.HOME .. '/Library/Java/JavaVirtualMachines/*/Contents/Home',
  '/opt/homebrew/opt/openjdk*/libexec/openjdk.jdk/Contents/Home',
  '/usr/lib/jvm/*',
}

-- Every JDK on this machine, one per major version, oldest first.
-- The one JAVA_HOME points at wins its major, so `sdk use` / `sdk default` carry over.
local function installed_jdks()
  local java_home = vim.env.JAVA_HOME and vim.uv.fs_realpath(vim.env.JAVA_HOME)
  local homes = { java_home }
  for _, pattern in ipairs(jdk_globs) do
    vim.list_extend(homes, vim.fn.glob(pattern, false, true))
  end

  local by_major = {}
  for _, home in ipairs(homes) do
    local real = vim.uv.fs_realpath(home)
    local major = real and vim.fs.basename(home) ~= 'current' and jdk_major(real)
    if major and (not by_major[major] or real == java_home) then
      by_major[major] = { home = real, major = major, is_java_home = real == java_home }
    end
  end

  local jdks = vim.tbl_values(by_major)
  table.sort(jdks, function(a, b)
    return a.major < b.major
  end)
  return jdks
end

-- jdtls itself needs Java 21+, whatever the project targets. Without a usable JAVA_HOME
-- (nvim started from a GUI) take the oldest that qualifies: the newest is often a non-LTS.
local function server_jdk(jdks)
  local oldest
  for _, jdk in ipairs(jdks) do
    if jdk.major >= 21 then
      if jdk.is_java_home then
        return jdk
      end
      oldest = oldest or jdk
    end
  end
  return oldest
end

-- Maps each JDK to the execution environment a pom/gradle file asks for (release 17 -> JavaSE-17).
-- The default one (JAVA_HOME, else the server's) builds plain .java files and unpinned projects.
local function runtimes(jdks, server)
  local has_java_home = vim.iter(jdks):any(function(jdk)
    return jdk.is_java_home
  end)
  local list = {}
  for _, jdk in ipairs(jdks) do
    table.insert(list, {
      name = 'JavaSE-' .. (jdk.major <= 8 and '1.' .. jdk.major or jdk.major),
      path = jdk.home,
      default = (jdk.is_java_home or (not has_java_home and jdk == server)) or nil,
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

-- The Eclipse formatter profile the project's Spotless config enforces. jdtls runs the same
-- Eclipse formatter, so <leader>cf (and format-on-save, see conform.lua) match `spotlessCheck`.
--   gradle: spotless { java { eclipse().configFile('config/style.xml') } }
--   maven:  <eclipse><file>${project.basedir}/style.xml</file></eclipse>
local function spotless_profile(root)
  for _, name in ipairs { 'build.gradle', 'build.gradle.kts', 'pom.xml' } do
    local file = io.open(root .. '/' .. name)
    if file then
      local build = file:read '*a'
      file:close()
      local path = build:match 'eclipse%b()%s*%.%s*configFile[^\'"\n]*[\'"]([^\'"\n]+)'
      local maven = build:match '<eclipse>(.-)</eclipse>'
      path = path or (maven and maven:match '<file>%s*(.-)%s*</file>')
      if path then
        -- relative to the project: drop a leading ${project.basedir}/, $rootDir/ and the like
        path = path:gsub('^%$%b{}/', ''):gsub('^%$%w+/', '')
        path = vim.startswith(path, '/') and path or vim.fs.joinpath(root, path)
        return vim.uv.fs_stat(path) and path or nil
      end
    end
  end
end

local function on_attach(_, bufnr)
  local jdtls = require 'jdtls'
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
  map('n', '<leader>cu', jdtls.update_project_config, '[U]pdate project config from pom/gradle')

  -- Tests run under the debugger (breakpoints work); failures land in the quickfix list.
  map('n', '<leader>tc', jdtls.test_class, '[T]est [C]lass')
  map('n', '<leader>tm', jdtls.test_nearest_method, '[T]est nearest [M]ethod')
  map('n', '<leader>tp', jdtls.pick_test, '[T]est [P]ick')
  map('n', '<leader>tl', require('dap').run_last, 'Re-run [L]ast test or debug session')
end

local function attach(args)
  -- Real files only. jdt:// class files opened from a definition jump are attached by
  -- nvim-jdtls itself, and unnamed scratch buffers have no project: vim.filetype.get_option()
  -- (nvim-dap calls it on every debug session) sets filetype=java on a hidden one.
  local name = vim.api.nvim_buf_get_name(args.buf)
  if name == '' or not vim.startswith(vim.uri_from_bufnr(args.buf), 'file://') then
    return
  end

  local jdks = installed_jdks()
  local jdk = server_jdk(jdks)
  if not jdk then
    vim.notify('jdtls needs a JDK 21+; install one with `sdk install java 21-tem`', vim.log.levels.ERROR)
    return
  end

  local root = vim.fs.root(args.buf, root_markers) or vim.fs.dirname(name)
  local workspace = vim.fn.stdpath 'cache' .. '/jdtls/' .. vim.fs.basename(root) .. '-' .. vim.fn.sha256(root):sub(1, 8)

  -- Maven plugin goals not to run on jdtls's builds (frontend-maven-plugin's npm runs). m2e reads this
  -- file from the workspace at startup; java.configuration.maven.lifecycleMappings only takes effect
  -- after the first import has built the project, and that build is where an npm test run hangs.
  local m2e = workspace .. '/.metadata/.plugins/org.eclipse.m2e.core'
  vim.fn.mkdir(m2e, 'p')
  vim.uv.fs_copyfile(vim.fn.stdpath 'config' .. '/jdtls/lifecycle-mapping-metadata.xml', m2e .. '/lifecycle-mapping-metadata.xml')

  local capabilities = vim.lsp.protocol.make_client_capabilities()
  capabilities = vim.tbl_deep_extend('force', capabilities, require('blink.cmp').get_lsp_capabilities())

  require('jdtls').start_or_attach({
    name = 'jdtls',
    cmd = {
      mason .. '/bin/jdtls',
      '--jvm-arg=-javaagent:' .. mason .. '/share/jdtls/lombok.jar',
      -- vscode-java's GC tuning: fast builds, and memory goes back to the OS when idle
      '--jvm-arg=-XX:+UseParallelGC',
      '--jvm-arg=-XX:GCTimeRatio=4',
      '--jvm-arg=-XX:AdaptiveSizePolicyWeight=90',
      '--jvm-arg=-Dsun.zip.disableMemoryMapping=true',
      '-data',
      workspace,
    },
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
          runtimes = runtimes(jdks, jdk),
          updateBuildConfiguration = 'automatic',
        },
        eclipse = { downloadSources = true },
        maven = { downloadSources = true },
        references = { includeDecompiledSources = true },
        signatureHelp = { enabled = true, description = { enabled = true } },
        inlayHints = { parameterNames = { enabled = 'all' } }, -- shown with <leader>th
        -- Turn on null analysis when the project has @Nullable annotations, without asking.
        compile = { nullAnalysis = { mode = 'automatic' } },
        completion = {
          -- Completed (and statically imported) without typing the class name first.
          favoriteStaticMembers = {
            'org.junit.jupiter.api.Assertions.*',
            'org.junit.jupiter.api.Assumptions.*',
            'org.junit.Assert.*',
            'org.junit.Assume.*',
            'org.assertj.core.api.Assertions.*',
            'org.mockito.Mockito.*',
            'org.mockito.ArgumentMatchers.*',
            'org.hamcrest.Matchers.*',
            'java.util.Objects.requireNonNull',
            'java.util.Objects.requireNonNullElse',
          },
        },
        codeGeneration = {
          hashCodeEquals = { useJava7Objects = true },
          useBlocks = true,
        },
        format = { settings = { url = spotless_profile(root) } },
        sources = {
          organizeImports = { starThreshold = 9999, staticStarThreshold = 9999 },
        },
      },
    },
  }, nil, { bufnr = args.buf })
end

return {
  'mfussenegger/nvim-jdtls',
  ft = 'java',
  dependencies = { 'mfussenegger/nvim-dap' },
  config = function()
    -- Registers the `java` adapter, so <F5> offers every main class in the project. Only the
    -- first call counts, and nvim-jdtls makes its own (without these options) on LspAttach.
    require('jdtls').setup_dap { hotcodereplace = 'auto' }

    -- :JdtRestart starts the new client with vim.lsp.start_client(), which Nvim 0.13 removes.
    -- `:lsp restart` does the same: stops jdtls, starts it with its config, reattaches its buffers.
    require('jdtls.setup').restart = function()
      vim.cmd 'lsp restart jdtls'
    end

    vim.api.nvim_create_autocmd('FileType', {
      group = vim.api.nvim_create_augroup('fuentastic-jdtls', { clear = true }),
      pattern = 'java',
      callback = attach,
    })
  end,
}

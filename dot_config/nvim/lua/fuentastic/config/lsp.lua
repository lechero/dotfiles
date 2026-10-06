local function setup_lsp_keymaps(event)
  local function map(keys, func, desc, mode)
    mode = mode or 'n'
    vim.keymap.set(mode, keys, func, { buffer = event.buf, desc = 'LSP: ' .. desc })
  end

  local client = vim.lsp.get_client_by_id(event.data.client_id)

  -- Not over a buffer's own gd: HTL's, JSP's and .content.xml's (after/ftplugin/) know what lemminx
  -- doesn't. Whether a server answers is checked when it's pressed, not on attach: jdtls registers
  -- definitions only after attaching, and Copilot attaches everywhere and answers none (Vim's gd then).
  local own_gd = vim.iter(vim.api.nvim_buf_get_keymap(event.buf, 'n')):any(function(m)
    return m.lhs == 'gd' and not vim.startswith(m.desc or '', 'LSP: ')
  end)
  if not own_gd then
    map('gd', function()
      if #vim.lsp.get_clients({ bufnr = 0, method = vim.lsp.protocol.Methods.textDocument_definition }) > 0 then
        require('telescope.builtin').lsp_definitions()
      else
        vim.cmd('normal! gd')
      end
    end, '[G]oto [D]efinition')
  end
  map('gr', require('telescope.builtin').lsp_references, '[G]oto [R]eferences')
  map('gI', require('telescope.builtin').lsp_implementations, '[G]oto [I]mplementation')
  map('<leader>D', require('telescope.builtin').lsp_type_definitions, 'Type [D]efinition')
  map('<leader>ds', require('telescope.builtin').lsp_document_symbols, '[D]ocument [S]ymbols')
  map('<leader>ws', require('telescope.builtin').lsp_dynamic_workspace_symbols, '[W]orkspace [S]ymbols')
  map('<leader>rn', vim.lsp.buf.rename, '[R]e[n]ame')
  map('<leader>ca', vim.lsp.buf.code_action, '[C]ode [A]ction', { 'n', 'x' })
  map('gD', vim.lsp.buf.declaration, '[G]oto [D]eclaration')

  if client and client:supports_method(vim.lsp.protocol.Methods.textDocument_documentHighlight) then
    local highlight_augroup = vim.api.nvim_create_augroup('kickstart-lsp-highlight', { clear = false })
    vim.api.nvim_create_autocmd({ 'CursorHold', 'CursorHoldI' }, {
      buffer = event.buf,
      group = highlight_augroup,
      callback = vim.lsp.buf.document_highlight,
    })

    vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI' }, {
      buffer = event.buf,
      group = highlight_augroup,
      callback = vim.lsp.buf.clear_references,
    })

    vim.api.nvim_create_autocmd('LspDetach', {
      group = vim.api.nvim_create_augroup('kickstart-lsp-detach', { clear = true }),
      callback = function(event2)
        vim.lsp.buf.clear_references()
        vim.api.nvim_clear_autocmds({ group = 'kickstart-lsp-highlight', buffer = event2.buf })
      end,
    })
  end

  if client and client.name == 'eslint' then
    map('<leader>ce', '<Cmd>LspEslintFixAll<CR>', 'Apply all [E]SLint fixes')
  end

  -- Not gated on the capability: jdtls only registers inlay hints after attaching.
  map('<leader>th', function()
    vim.lsp.inlay_hint.enable(not vim.lsp.inlay_hint.is_enabled({ bufnr = event.buf }))
  end, '[T]oggle Inlay [H]ints')
end

local function setup_lsp()
  vim.api.nvim_create_autocmd('LspAttach', {
    group = vim.api.nvim_create_augroup('kickstart-lsp-attach', { clear = true }),
    callback = setup_lsp_keymaps,
  })

  local capabilities = vim.lsp.protocol.make_client_capabilities()
  capabilities = vim.tbl_deep_extend('force', capabilities, require('blink.cmp').get_lsp_capabilities())

  -- vue_ls only handles a .vue file's template and styles; it forwards the TypeScript to ts_ls
  -- (lspconfig's vue_ls on_init), which needs the Vue plugin that vue-language-server ships.
  local vue_plugin = {
    name = '@vue/typescript-plugin',
    location = vim.fn.stdpath('data') .. '/mason/packages/vue-language-server/node_modules/@vue/language-server',
    languages = { 'vue' },
    configNamespace = 'typescript',
  }

  -- GitLab's CI schema. SchemaStore already maps .gitlab-ci.yml and *.gitlab-ci.yml to it; files
  -- in .gitlab/ci/ that the pipeline includes are named anything.
  local gitlab_ci_schema = 'https://gitlab.com/gitlab-org/gitlab-foss/-/raw/master/app/assets/javascripts/editor/schema/ci.json'

  -- The `--flag`s a project's own lint scripts pass to ESLint (`eslint ./ --flag
  -- v10_config_lookup_from_file`), so the editor picks the same config files as CI.
  local function eslint_flags(root)
    local flags = {}
    for _, file in ipairs(vim.list_extend({ root .. '/package.json' }, vim.fn.glob(root .. '/*/package.json', false, true))) do
      local ok, package = pcall(vim.json.decode, table.concat(vim.fn.readfile(file), '\n'))
      for _, script in pairs(ok and type(package) == 'table' and package.scripts or {}) do
        for flag in script:gmatch('eslint%s.-%-%-flag[%s=]+([%w_]+)') do
          if not vim.list_contains(flags, flag) then
            table.insert(flags, flag)
          end
        end
      end
    end
    return flags
  end
  local eslint_before_init = vim.lsp.config.eslint.before_init

  local servers = {
    rust_analyzer = {},
    ts_ls = {
      init_options = { plugins = { vue_plugin } },
      filetypes = { 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'vue' },
    },
    vue_ls = {},
    yamlls = {
      settings = {
        yaml = {
          schemas = { [gitlab_ci_schema] = { '.gitlab/ci/**/*.yml', '.gitlab/ci/**/*.yaml' } },
          customTags = { '!reference sequence' }, -- GitLab's; otherwise "Unresolved tag" on each use
          format = { enable = false }, -- it would rewrite CI files on save (lspconfig turns it on)
        },
      },
      on_init = function() end, -- lspconfig's claims formatting support regardless
    },
    -- extends, needs, !reference, includes and components (doc/gitlab.md); needs Rust 1.85+ to build
    gitlab_ci_ls = {},
    -- Only in projects with an ESLint config, with the project's own ESLint (doc/linting.md). Not as
    -- a formatter: its fixes rewrite code (let to const...), so they're <leader>ce, not on save.
    eslint = {
      settings = { format = false },
      before_init = function(params, config)
        eslint_before_init(params, config)
        config.settings.options = { flags = eslint_flags(config.root_dir) }
      end,
    },
    -- .content.xml, pom.xml (against the schema it declares). No formatting: AEM's XML stays as
    -- written. Turned off client-side: with xml.format.enabled = false, lemminx unregisters its
    -- formatter with an empty request that nvim rejects, and the formatter stays.
    lemminx = {
      capabilities = {
        textDocument = { formatting = { dynamicRegistration = false }, rangeFormatting = { dynamicRegistration = false } },
      },
      on_init = function(client)
        client.server_capabilities.documentFormattingProvider = false
        client.server_capabilities.documentRangeFormattingProvider = false
      end,
    },
    -- package.json, tsconfig.json, renovate.json, .stylelintrc...: SchemaStore's schemas. OSGi
    -- configs and the like stay as written: no formatting.
    jsonls = {
      init_options = { provideFormatter = false },
      settings = {
        json = {
          schemas = require('schemastore').json.schemas(),
          validate = { enable = true },
        },
      },
    },
    lua_ls = {
      settings = {
        Lua = {
          completion = {
            callSnippet = 'Replace',
          },
        },
      },
    },
  }

  require('mason').setup()

  local ensure_installed = vim.tbl_keys(servers or {})
  vim.list_extend(ensure_installed, {
    'stylua',
    'jdtls', -- started by nvim-jdtls (plugins/jdtls.lua), not the servers loop
    'java-debug-adapter', -- loaded into jdtls as bundles
    'java-test',
    'sonarlint-language-server', -- started by sonarlint.nvim (plugins/sonarlint.lua)
    'checkstyle', -- run by nvim-lint (plugins/lint.lua)
  })

  require('mason-tool-installer').setup({ ensure_installed = ensure_installed })

  require('mason-lspconfig').setup({
    ensure_installed = vim.tbl_keys(servers),
  })

  local has_new_lsp_api = vim.lsp and vim.lsp.config and vim.lsp.enable
  local lspconfig = has_new_lsp_api and nil or require('lspconfig')

  for server_name, server in pairs(servers) do
    server.capabilities = vim.tbl_deep_extend('force', {}, capabilities, server.capabilities or {})
    if has_new_lsp_api then
      vim.lsp.config(server_name, server)
      vim.lsp.enable(server_name)
    else
      lspconfig[server_name].setup(server)
    end
  end
end

return {
  {
    'folke/lazydev.nvim',
    ft = 'lua',
    opts = {
      library = {
        { path = 'luvit-meta/library', words = { 'vim%.uv' } },
      },
    },
  },
  { 'Bilal2453/luvit-meta', lazy = true },
  {
    'neovim/nvim-lspconfig',
    dependencies = {
      { 'williamboman/mason.nvim', config = true },
      {
        'williamboman/mason-lspconfig.nvim',
        opts = {
          automatic_enable = false,
        },
      },
      'WhoIsSethDaniel/mason-tool-installer.nvim',
      'b0o/SchemaStore.nvim', -- jsonls' schemas
      { 'j-hui/fidget.nvim', opts = {} },
      'saghen/blink.cmp', -- its completion capabilities
    },
    config = setup_lsp,
  },
}

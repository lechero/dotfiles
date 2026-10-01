local function setup_lsp_keymaps(event)
  local function map(keys, func, desc, mode)
    mode = mode or 'n'
    vim.keymap.set(mode, keys, func, { buffer = event.buf, desc = 'LSP: ' .. desc })
  end

  local client = vim.lsp.get_client_by_id(event.data.client_id)

  -- Only for servers that answer it: Copilot attaches everywhere, and would swap a buffer's own gd
  -- (HTL's, from after/ftplugin/html.lua) for "server does not support textDocument/definition".
  if client and client:supports_method(vim.lsp.protocol.Methods.textDocument_definition) then
    map('gd', require('telescope.builtin').lsp_definitions, '[G]oto [D]efinition')
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
  capabilities = vim.tbl_deep_extend('force', capabilities, require('cmp_nvim_lsp').default_capabilities())

  -- vue_ls only handles a .vue file's template and styles; it forwards the TypeScript to ts_ls
  -- (lspconfig's vue_ls on_init), which needs the Vue plugin that vue-language-server ships.
  local vue_plugin = {
    name = '@vue/typescript-plugin',
    location = vim.fn.stdpath('data') .. '/mason/packages/vue-language-server/node_modules/@vue/language-server',
    languages = { 'vue' },
    configNamespace = 'typescript',
  }

  local servers = {
    rust_analyzer = {},
    ts_ls = {
      init_options = { plugins = { vue_plugin } },
      filetypes = { 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'vue' },
    },
    vue_ls = {},
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
      { 'j-hui/fidget.nvim', opts = {} },
      'hrsh7th/cmp-nvim-lsp',
      'hrsh7th/cmp-cmdline',
      'hrsh7th/cmp-buffer',
      'hrsh7th/cmp-path',
    },
    config = setup_lsp,
  },
}

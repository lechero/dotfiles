return {
  'zbirenbaum/copilot.lua',
  event = 'InsertEnter', -- ensure it loads when you start typing
  build = ':Copilot auth',
  init = function()
    vim.api.nvim_create_user_command('CopilotAccount', function()
      require('fuentastic.copilot_account').show()
    end, { desc = 'Which GitHub account Copilot uses, and who is signed in to it' })
  end,
  config = function()
    -- Some projects use another GitHub account (lua/fuentastic/copilot_account.lua). One language
    -- server serves the whole Neovim, so the file you first type in picks the account.
    local account = require 'fuentastic.copilot_account'
    vim.g.copilot_account, vim.g.copilot_account_folder = account.current()
    require('copilot').setup {
      suggestion = { enabled = true, auto_trigger = true },
      panel = { enabled = true },
      copilot_node_command = 'node', -- or absolute path if needed
      filetypes = {
        markdown = true,
        gitcommit = true,
        ['*'] = true,
      },
      server_opts_overrides = account.server_opts(vim.g.copilot_account),
    }
  end,
}

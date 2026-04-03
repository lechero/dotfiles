return {
  'folke/sidekick.nvim',
  opts = {
    cli = {
      watch = true,
      mux = {
        backend = 'tmux',
        enabled = true,
      },
      tools = {
        kilocode = {
          cmd = { 'kilocode' },
        },
        deepagents = {
          cmd = { 'deepagents' },
        },
      },
      prompts = {
        refactor = 'Refactor {this} to be simpler and more maintainable. Explain key decisions briefly.',
        fix_tests = 'Write or fix tests for {this}. If needed, explain how to run them.',
        review_pr = 'Review {file} for bugs, edge cases, and style issues. Suggest improvements.',
      },
    },
  },
  keys = {
    {
      '<tab>',
      function()
        if not require('sidekick').nes_jump_or_apply() then
          return '<Tab>'
        end
      end,
      expr = true,
      mode = { 'n' },
      desc = 'Sidekick: goto/apply next edit suggestion',
    },
    {
      '<leader>aa',
      function()
        require('sidekick.cli').toggle()
      end,
      mode = { 'n', 'v' },
      desc = 'Sidekick: toggle CLI',
    },
    {
      '<leader>as',
      function()
        require('sidekick.cli').select()
      end,
      desc = 'Sidekick: select CLI tool',
    },
    {
      '<leader>ad',
      function()
        require('sidekick.cli').close()
      end,
      desc = 'Sidekick: detach/close CLI session',
    },
    {
      '<leader>at',
      function()
        require('sidekick.cli').send({ msg = '{this}' })
      end,
      mode = { 'n', 'v' },
      desc = 'Sidekick: send this (context at cursor / range)',
    },
    {
      '<leader>af',
      function()
        require('sidekick.cli').send({ msg = '{file}' })
      end,
      desc = 'Sidekick: send file',
    },
    {
      '<leader>av',
      function()
        require('sidekick.cli').send({ msg = '{selection}' })
      end,
      mode = { 'v' },
      desc = 'Sidekick: send visual selection',
    },
    {
      '<leader>ap',
      function()
        require('sidekick.cli').prompt()
      end,
      mode = { 'n', 'v' },
      desc = 'Sidekick: insert prompt/context',
    },
    {
      '<c-.>',
      function()
        require('sidekick.cli').focus()
      end,
      mode = { 'n', 'x', 'i', 't' },
      desc = 'Sidekick: switch focus',
    },
    {
      '<leader>ac',
      function()
        require('sidekick.cli').toggle({ name = 'claude', focus = true })
      end,
      mode = { 'n', 'v' },
      desc = 'Sidekick: Claude',
    },
    {
      '<leader>ak',
      function()
        require('sidekick.cli').toggle({ name = 'kilocode', focus = true })
      end,
      mode = { 'n', 'v' },
      desc = 'Sidekick: Kilo Code',
    },
  },
}

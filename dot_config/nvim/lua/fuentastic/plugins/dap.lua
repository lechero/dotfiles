-- Language-agnostic debugger core. Adapters register themselves elsewhere
-- (java: nvim-jdtls wires java-debug in plugins/jdtls.lua).
return {
  'mfussenegger/nvim-dap',
  dependencies = {
    'rcarriga/nvim-dap-ui',
    'nvim-neotest/nvim-nio',
    { 'theHamsta/nvim-dap-virtual-text', opts = {} },
  },
  keys = {
    {
      '<F5>',
      function()
        require('dap').continue()
      end,
      desc = 'Debug: Start/Continue',
    },
    {
      '<S-F5>',
      function()
        require('dap').terminate()
      end,
      desc = 'Debug: Terminate',
    },
    {
      '<F1>',
      function()
        require('dap').step_into()
      end,
      desc = 'Debug: Step Into',
    },
    {
      '<F2>',
      function()
        require('dap').step_over()
      end,
      desc = 'Debug: Step Over',
    },
    {
      '<F3>',
      function()
        require('dap').step_out()
      end,
      desc = 'Debug: Step Out',
    },
    {
      '<F7>',
      function()
        require('dapui').toggle()
      end,
      desc = 'Debug: Toggle UI',
    },
    {
      '<leader>b',
      function()
        require('dap').toggle_breakpoint()
      end,
      desc = 'Debug: Toggle [B]reakpoint',
    },
    {
      '<leader>B',
      function()
        require('dap').set_breakpoint(vim.fn.input('Breakpoint condition: '))
      end,
      desc = 'Debug: Conditional [B]reakpoint',
    },
  },
  config = function()
    local dap = require('dap')
    local dapui = require('dapui')

    dapui.setup()

    -- Open on start but leave it up afterwards, so program output and test
    -- results stay readable once the session ends. <F7> closes it.
    dap.listeners.before.attach['fuentastic-dapui'] = dapui.open
    dap.listeners.before.launch['fuentastic-dapui'] = dapui.open
  end,
}

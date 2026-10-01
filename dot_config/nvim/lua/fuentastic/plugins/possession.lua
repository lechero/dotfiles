return {
  'jedrzejboczar/possession.nvim',
  opts = {},
  config = function()
    require('possession').setup {
      commands = {
        save = 'SSave',
        load = 'SLoad',
        delete = 'SDelete',
        list = 'SList',
      },
    }

    -- <c-t> deletes the highlighted session, <c-r> renames it
    vim.keymap.set('n', '<leader>sS', function()
      require('telescope').extensions.possession.list()
    end, { desc = '[S]earch [S]essions' })

    -- same list as alpha's "Sessions Without Workspace" section
    vim.api.nvim_create_user_command('SClean', function()
      local _, others = require('possession.query').workspaces_with_shortcuts(require 'fuentastic.workspaces', {
        map_session = function(s)
          return s.name
        end,
      })
      local names = vim.tbl_map(function(sws)
        return sws[2]
      end, others)

      if #names == 0 then
        vim.notify 'No sessions without a workspace'
        return
      end

      local msg = ('Delete %d sessions without a workspace?\n\n%s'):format(#names, table.concat(names, '\n'))
      if vim.fn.confirm(msg, '&Yes\n&No', 2) ~= 1 then
        return
      end

      local session = require 'possession.session'
      for _, name in ipairs(names) do
        session.delete(name, { no_confirm = true })
      end
    end, { desc = 'Delete sessions that do not belong to a workspace' })
  end,
}

-- Which GitHub account Copilot signs in with (doc/copilot.md). ~/.config/copilot-accounts/folders
-- names an account for a folder and everything in it, one `<account> <folder>` per line; anything
-- else uses the usual sign-in in ~/.config/github-copilot. The language server keeps its sign-in
-- in $XDG_CONFIG_HOME/github-copilot, so each account gets its own ~/.config/copilot-accounts/
-- <account> as the server's XDG_CONFIG_HOME, and signs in there once with :Copilot auth.
local M = {}

M.dir = (vim.env.XDG_CONFIG_HOME or vim.fn.expand '~/.config') .. '/copilot-accounts'

--- The account for a file or folder, and the folder in the list that chose it. With nested
--- folders in the list, the deepest one wins.
---@param path string
---@param folders_file? string
---@return string? account, string? folder
function M.for_path(path, folders_file)
  folders_file = folders_file or (M.dir .. '/folders')
  if vim.fn.filereadable(folders_file) == 0 then
    return
  end
  path = vim.fs.normalize(path)
  local account, folder
  for line in io.lines(folders_file) do
    local name, dir = line:match '^%s*([^#%s]%S*)%s+(.-)%s*$'
    if name then
      dir = vim.fs.normalize(dir)
      if (path == dir or vim.startswith(path, dir .. '/')) and #dir > #(folder or '') then
        account, folder = name, dir
      end
    end
  end
  return account, folder
end

--- The account for the current buffer's file, or for the working directory when it has none.
---@return string? account, string? folder
function M.current()
  local path = vim.api.nvim_buf_get_name(0)
  if path == '' or vim.bo.buftype ~= '' then
    path = vim.fn.getcwd()
  end
  return M.for_path(path)
end

--- copilot.lua's server_opts_overrides for an account: its own config folder.
---@param account? string
function M.server_opts(account)
  if not account then
    return {}
  end
  local home = M.dir .. '/' .. account
  vim.fn.mkdir(home, 'p')
  return { cmd_env = { XDG_CONFIG_HOME = home } }
end

--- :CopilotAccount, which account this Neovim's Copilot uses and who is signed in to it.
function M.show()
  local account, folder = vim.g.copilot_account, vim.g.copilot_account_folder
  if vim.g.copilot_account == nil and not package.loaded['copilot'] then
    account, folder = M.current()
  end
  local which = account and ('account ' .. account .. ' (' .. vim.fn.fnamemodify(folder, ':~') .. ')') or 'the default account'
  local ok, client = pcall(function()
    return require('copilot.client').get()
  end)
  if not ok or not client then
    vim.notify('Copilot will use ' .. which .. '. It starts when you first type in insert mode.')
    return
  end
  require('copilot.api').check_status(client, {}, function(err, status)
    local who = err and ('status unknown: ' .. tostring(err))
      or (status and status.user) and ('signed in as ' .. status.user)
      or 'not signed in yet: run :Copilot auth'
    vim.schedule(function()
      vim.notify('Copilot uses ' .. which .. ', ' .. who)
    end)
  end)
end

return M

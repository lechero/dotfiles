local map = vim.keymap.set

local function winmove(key)
  local current_window = vim.fn.winnr()
  vim.cmd.wincmd(key)

  if current_window ~= vim.fn.winnr() then
    return
  end

  if key:match('[jk]') then
    vim.cmd.wincmd('v')
  else
    vim.cmd.wincmd('s')
  end

  vim.cmd.wincmd(key)
end

map('n', '<leader>h', function()
  winmove('h')
end, { noremap = true, silent = true })

map('n', '<leader>j', function()
  winmove('j')
end, { noremap = true, silent = true })

map('n', '<leader>k', function()
  winmove('k')
end, { noremap = true, silent = true })

map('n', '<leader>l', function()
  winmove('l')
end, { noremap = true, silent = true })

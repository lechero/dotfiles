local map = vim.keymap.set

local function winmove(key)
  local current_window = vim.fn.winnr()
  vim.cmd.wincmd(key)

  if current_window ~= vim.fn.winnr() then
    return
  end

  if key:match('[jk]') then
    vim.cmd.wincmd('s')
  else
    vim.cmd.wincmd('v')
  end

  vim.cmd.wincmd(key)
end

map('n', '<leader>h', function()
  winmove('h')
end, { noremap = true, silent = true, desc = 'Window left (split if none)' })

map('n', '<leader>j', function()
  winmove('j')
end, { noremap = true, silent = true, desc = 'Window below (split if none)' })

map('n', '<leader>k', function()
  winmove('k')
end, { noremap = true, silent = true, desc = 'Window above (split if none)' })

map('n', '<leader>l', function()
  winmove('l')
end, { noremap = true, silent = true, desc = 'Window right (split if none)' })

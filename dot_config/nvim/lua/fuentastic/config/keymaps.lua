local map = vim.keymap.set

local function get_buffer_path_info(buf)
  local fname = vim.api.nvim_buf_get_name(buf)
  if fname == '' then
    vim.notify('No file name for current buffer', vim.log.levels.WARN)
    return nil
  end

  local abspath = vim.fn.fnamemodify(fname, ':p')
  local filedir = (vim.fs and vim.fs.dirname) and vim.fs.dirname(abspath) or vim.fn.fnamemodify(abspath, ':h')
  local found = vim.fs.find('package.json', { upward = true, path = filedir })[1]
  local root = found and vim.fs.dirname(found) or (vim.uv or vim.loop).cwd()

  local rel
  do
    local ok, result = pcall(function()
      return vim.fs.relative(abspath, root)
    end)
    if ok and result then
      rel = result
    else
      rel = abspath:gsub('^' .. vim.pesc(root .. '/'), '')
    end
  end

  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)

  return {
    abspath = abspath,
    rel = rel,
    body = table.concat(lines, '\n'),
  }
end

local function copy_with_pbcopy(payload, success_message)
  if vim.fn.executable('pbcopy') ~= 1 then
    vim.notify('pbcopy not found on PATH', vim.log.levels.ERROR)
    return
  end

  vim.fn.system('pbcopy', payload)
  vim.notify(success_message)
end

map('n', '<Esc>', '<cmd>nohlsearch<CR>')
map('n', '<leader>q', vim.diagnostic.setloclist, { desc = 'Open diagnostic [Q]uickfix list' })
map('t', '<Esc><Esc>', '<C-\\><C-n>', { desc = 'Exit terminal mode' })

map('v', '<C-r>', '"hy:%s/<C-r>h//gc<left><left><left>')
map('n', '<leader>i', '<cmd>e ~/notes/notes.md<CR>', { silent = true })
map('n', ';tq', ':q<CR>', { silent = true })
map('n', ';to', ':tab split<CR>', { silent = true })
map('n', ';ts', ':tab split<CR>', { silent = true })
map('n', ';tc', ':tabclose<CR>', { silent = true })

map('n', '<leader>fp', function()
  local path = vim.fn.expand('%:p')
  if path == '' then
    vim.notify('No file name for current buffer', vim.log.levels.WARN)
    return
  end

  copy_with_pbcopy(path, 'Copied: ' .. path)
end, { desc = 'Copy absolute file path (pbcopy)' })

map('n', '<leader>fyy', function()
  local info = get_buffer_path_info(0)
  if not info then
    return
  end

  local header = '// ' .. info.rel
  copy_with_pbcopy(header .. '\n' .. info.body, 'Copied with header: ' .. info.rel)
end, { desc = 'Copy file with project-relative header (pbcopy)' })

map('n', '<leader>fyc', function()
  local info = get_buffer_path_info(0)
  if not info then
    return
  end

  local ft = vim.bo[0].filetype or ''
  local lang_map = {
    typescriptreact = 'tsx',
    javascriptreact = 'jsx',
    typescript = 'ts',
    javascript = 'js',
    sh = 'bash',
    yml = 'yaml',
  }
  local lang = lang_map[ft] or ft
  local header = '// ' .. info.rel
  local payload = table.concat({
    '```' .. (lang ~= '' and lang or ''),
    header,
    info.body,
    '```',
    '',
  }, '\n')

  copy_with_pbcopy(payload, 'Copied code block with header: ' .. info.rel)
end, { desc = 'Copy file as fenced code block with header (pbcopy)' })

map('n', '<leader>1', '1gt', { silent = true })
map('n', '<leader>2', '2gt', { silent = true })
map('n', '<leader>3', '3gt', { silent = true })
map('n', '<leader>4', '4gt', { silent = true })
map('n', '<leader>5', '5gt', { silent = true })

map('n', ';l', "\"ayiwoconsole.log('<C-R>a:', <C-R>a);<Esc>", { silent = true })

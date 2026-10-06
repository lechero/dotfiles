-- GitLab from nvim, through glab (lua/fuentastic/gitlab.lua, doc/gitlab.md).
local function gitlab()
  return require 'fuentastic.gitlab'
end

vim.api.nvim_create_user_command('GitlabStatus', function()
  gitlab().status()
end, { desc = 'The branch, its merge request and its pipeline' })
vim.api.nvim_create_user_command('GitlabCiLint', function(args)
  gitlab().lint(args.bang)
end, { bang = true, desc = 'Lint .gitlab-ci.yml on GitLab; ! simulates the pipeline for this branch' })
vim.api.nvim_create_user_command('GitlabPipeline', function()
  gitlab().pipeline_view()
end, { desc = "The branch's pipeline in glab's TUI" })

vim.keymap.set('n', '<leader>gs', '<Cmd>GitlabStatus<CR>', { desc = 'GitLab: branch, MR and pipeline [S]tatus' })
vim.keymap.set('n', '<leader>gl', '<Cmd>GitlabCiLint<CR>', { desc = 'GitLab: [L]int .gitlab-ci.yml' })
vim.keymap.set('n', '<leader>gL', '<Cmd>GitlabCiLint!<CR>', { desc = 'GitLab: [L]int, simulating this branch' })
vim.keymap.set('n', ';p', '<Cmd>GitlabPipeline<CR>', { desc = 'GitLab: pipeline TUI' }) -- next to ;g's lazygit

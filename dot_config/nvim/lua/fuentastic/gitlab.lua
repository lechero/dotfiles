-- GitLab through glab: lint the repo's CI config on GitLab itself, and show where the current
-- branch is at (its merge request, its pipeline and the pipeline's jobs). doc/gitlab.md.
local M = {}

local function notify(msg, level)
  vim.notify(msg, level, { title = 'GitLab' })
end

-- The git root of the current buffer (or of the cwd), and whether glab is there to ask GitLab.
local function repo()
  if vim.fn.executable 'glab' == 0 then
    notify('Needs glab: `brew install glab`, then `glab auth login`', vim.log.levels.ERROR)
    return nil
  end
  local name = vim.api.nvim_buf_get_name(0)
  local root = vim.fs.root(vim.uv.fs_stat(name) and name or vim.uv.cwd(), '.git') -- oil, terminals: the cwd
  if not root then
    notify('Not in a git repository', vim.log.levels.WARN)
  end
  return root
end

-- Runs a command in `dir` and calls back on the main loop with its stdout, or nil and its error.
local function run(cmd, dir, callback)
  vim.system(cmd, { cwd = dir, text = true }, function(result)
    vim.schedule(function()
      if result.code == 0 then
        callback(result.stdout)
      else
        callback(nil, vim.trim(result.stderr ~= '' and result.stderr or result.stdout))
      end
    end)
  end)
end

local function decode(text)
  local ok, value = pcall(vim.json.decode, text or '', { luanil = { object = true, array = true } })
  return ok and value or nil
end

--------------------------------------------------------------------------------------------------
-- Lint

-- The repo's CI files, root first: what `include: local:` can pull in is tracked by git.
local function ci_files(root)
  local files = {}
  for _, name in ipairs { '.gitlab-ci.yml', '.gitlab-ci.yaml' } do
    if vim.uv.fs_stat(root .. '/' .. name) then
      table.insert(files, root .. '/' .. name)
    end
  end
  local tracked = vim.system({ 'git', 'ls-files', '*.gitlab-ci.yml', '*.gitlab-ci.yaml', '.gitlab/ci/*' }, { cwd = root, text = true }):wait()
  for _, path in ipairs(vim.split(tracked.stdout or '', '\n', { trimempty = true })) do
    if not vim.list_contains(files, root .. '/' .. path) then
      table.insert(files, root .. '/' .. path)
    end
  end
  return files
end

-- glab's errors name the job they're about: "jobs:unit test config contains unknown keys: scrip",
-- or "jobs unit test config ...". Point the quickfix entry at that job's definition, else at the top
-- of the root file.
local function locate(message, files)
  local best
  for _, file in ipairs(files) do
    local buf = vim.fn.bufnr(file)
    local lines = buf ~= -1 and vim.api.nvim_buf_is_loaded(buf) and vim.api.nvim_buf_get_lines(buf, 0, -1, false) or vim.fn.readfile(file)
    for lnum, line in ipairs(lines) do
      local job = line:match '^([^%s#][^:]*):%s*$' or line:match '^([^%s#][^:]*):%s+[&!]'
      local names_it = job and (vim.startswith(message, 'jobs:' .. job .. ' ') or vim.startswith(message, 'jobs ' .. job .. ' '))
      if names_it and (not best or #job > #best.job) then
        best = { job = job, filename = file, lnum = lnum }
      end
    end
  end
  return best or { filename = files[1], lnum = 1 }
end

-- glab ci lint on the repo's root CI file, its unsaved changes included. GitLab resolves the
-- includes and components itself; with `dry_run`, it also runs the pipeline's rules for the
-- current branch.
function M.lint(dry_run)
  local root = repo()
  if not root then
    return
  end
  local files = ci_files(root)
  if not files[1] or not files[1]:match '%.gitlab%-ci%.ya?ml$' then
    notify('No .gitlab-ci.yml in ' .. root, vim.log.levels.WARN)
    return
  end

  local file = files[1]
  local buf = vim.fn.bufnr(file)
  if buf ~= -1 and vim.bo[buf].modified then
    file = vim.fn.tempname() .. '.yml'
    vim.fn.writefile(vim.api.nvim_buf_get_lines(buf, 0, -1, false), file)
  end
  local cmd = { 'glab', 'ci', 'lint', file }
  if dry_run then
    local branch = vim.trim(vim.system({ 'git', 'branch', '--show-current' }, { cwd = root, text = true }):wait().stdout or '')
    vim.list_extend(cmd, { '--dry-run', '--ref', branch })
  end

  notify(dry_run and 'Simulating the pipeline…' or 'Linting the CI config…')
  vim.system(cmd, { cwd = root, text = true }, function(result)
    vim.schedule(function()
      local output = (result.stdout or '') .. (result.stderr or '')
      local items = {}
      for message in output:gmatch '\n%d+ ([^\n]+)' do
        local where = locate(message, files)
        table.insert(items, { filename = where.filename, lnum = where.lnum, text = message, type = 'E' })
      end
      vim.fn.setqflist({}, 'r', { title = 'glab ci lint', items = items })
      if #items > 0 then
        vim.cmd.copen()
      elseif result.code == 0 then
        notify('CI/CD config is valid' .. (dry_run and ' for this branch' or ''))
      else
        notify(vim.trim(output), vim.log.levels.ERROR)
      end
    end)
  end)
end

--------------------------------------------------------------------------------------------------
-- Status

local status_icons = {
  success = { '✓', 'DiagnosticOk' },
  failed = { '✗', 'DiagnosticError' },
  running = { '●', 'DiagnosticWarn' },
  pending = { '○', 'DiagnosticInfo' },
  created = { '○', 'Comment' },
  preparing = { '○', 'DiagnosticInfo' },
  waiting_for_resource = { '○', 'DiagnosticInfo' },
  scheduled = { '◷', 'DiagnosticInfo' },
  manual = { '▶', 'DiagnosticHint' },
  canceled = { '⊘', 'Comment' },
  skipped = { '»', 'Comment' },
}
local active = { running = true, pending = true, created = true, preparing = true, waiting_for_resource = true }

local merge_states = {
  mergeable = 'Ready to merge',
  not_approved = 'Waiting for approvals',
  ci_must_pass = 'The pipeline must pass first',
  ci_still_running = 'Waiting for the pipeline',
  discussions_not_resolved = 'Unresolved threads',
  draft_status = 'Still a draft',
  conflict = 'Has merge conflicts',
  need_rebase = 'Needs a rebase',
  not_open = 'Not open',
  blocked_status = 'Blocked by another merge request',
  checking = 'Checking whether it can merge…',
  unchecked = 'Checking whether it can merge…',
}

local function duration(seconds)
  seconds = math.floor(tonumber(seconds) or 0)
  if seconds >= 3600 then
    return string.format('%dh %02dm', seconds / 3600, seconds % 3600 / 60)
  end
  return seconds >= 60 and string.format('%dm %02ds', seconds / 60, seconds % 60) or seconds .. 's'
end

-- Why glab found nothing: the first paragraph of its error (under the boxed ERROR header), unless
-- that just says there is none.
local function reason(err, none)
  local words = {}
  for _, line in ipairs(vim.split(err or '', '\n')) do
    line = vim.trim(line)
    if line ~= '' and line ~= 'ERROR' then
      table.insert(words, line)
    elseif #words > 0 then
      break
    end
  end
  local text = table.concat(words, ' ')
  local lower = text:lower()
  if text == '' or lower:match 'no open merge request' or lower:match 'no pipelines' or lower:match '404' then
    return none
  end
  return text
end

-- Lines and their highlights, plus what each line links to (a url, a job).
local function render(state)
  local lines, marks, links = {}, {}, {}
  local function add(text, link)
    table.insert(lines, text)
    links[#lines] = link
  end
  local function mark(col, len, group)
    table.insert(marks, { #lines - 1, col, col + len, group })
  end
  local function blank()
    if lines[#lines] ~= '' then
      add ''
    end
  end

  add(' ' .. (state.branch or '?'))
  mark(0, #lines[#lines], 'Title')
  if state.upstream then
    local behind, ahead = state.upstream:match '(%d+)%s+(%d+)'
    lines[#lines] = lines[#lines] .. string.format('   %s ahead, %s behind the remote', ahead, behind)
  else
    lines[#lines] = lines[#lines] .. '   not pushed'
  end
  add ''

  local mr = state.mr
  if mr then
    add(string.format(' MR !%d  %s', mr.iid, mr.title), mr.web_url)
    mark(0, #tostring(mr.iid) + 5, 'Title')
    local facts = { mr.draft and 'draft' or mr.state, '→ ' .. mr.target_branch }
    local approvals = state.approvals
    if approvals then
      local required = approvals.approvals_required or 0
      local given = #(approvals.approved_by or {})
      table.insert(facts, required > 0 and string.format('%d of %d approvals', required - (approvals.approvals_left or 0), required) or given .. ' approvals')
    end
    if mr.has_conflicts then
      table.insert(facts, 'conflicts')
    end
    if mr.blocking_discussions_resolved == false then
      table.insert(facts, 'unresolved threads')
    end
    add('   ' .. table.concat(facts, ' · '), mr.web_url)
    local merge = merge_states[mr.detailed_merge_status] or mr.detailed_merge_status
    if merge and mr.detailed_merge_status ~= 'draft_status' then -- the facts say draft
      add('   ' .. merge, mr.web_url)
      mark(3, #merge, mr.detailed_merge_status == 'mergeable' and 'DiagnosticOk' or 'DiagnosticWarn')
    end
  else
    add(' ' .. reason(state.mr_error, 'No merge request for this branch'))
    mark(0, #lines[#lines], 'Comment')
  end
  blank()

  local pipeline = state.pipeline
  if pipeline then
    local icon = status_icons[pipeline.status] or { '?', 'Comment' }
    local title = string.format(' Pipeline #%d  ', pipeline.id)
    add(title .. icon[1] .. ' ' .. pipeline.status .. '  ' .. (pipeline.duration and duration(pipeline.duration) or ''), pipeline.web_url)
    mark(0, #title, 'Title')
    mark(#title, #icon[1] + 1 + #pipeline.status, icon[2])

    -- jobs by stage, stages in the order their first job was created
    local jobs = vim.deepcopy(pipeline.jobs or {})
    table.sort(jobs, function(a, b)
      return a.id < b.id
    end)
    local stages, by_stage = {}, {}
    for _, job in ipairs(jobs) do
      if not by_stage[job.stage] then
        by_stage[job.stage] = {}
        table.insert(stages, job.stage)
      end
      table.insert(by_stage[job.stage], job)
    end
    local stage_width, name_width = 0, 0
    for _, job in ipairs(jobs) do
      stage_width = math.max(stage_width, #job.stage)
      name_width = math.min(math.max(name_width, vim.fn.strdisplaywidth(job.name)), 44)
    end
    for _, stage in ipairs(stages) do
      for i, job in ipairs(by_stage[stage]) do
        local job_icon = status_icons[job.status] or { '?', 'Comment' }
        local label = i == 1 and stage or ''
        local prefix = string.format('   %-' .. stage_width .. 's  ', label)
        local name = job.name .. string.rep(' ', name_width - vim.fn.strdisplaywidth(job.name))
        local text = prefix .. job_icon[1] .. ' ' .. name
        if (tonumber(job.duration) or 0) > 0 then
          text = text .. '  ' .. duration(job.duration)
        end
        if job.allow_failure and job.status == 'failed' then
          text = text .. '  allowed to fail'
        end
        add((text:gsub('%s+$', '')), { url = job.web_url, job = job.id })
        mark(3, #label, 'Comment')
        mark(#prefix, #job_icon[1], job_icon[2])
      end
    end
  elseif reason(state.pipeline_error, '') ~= reason(state.mr_error, '') or not state.pipeline_error then
    add(' ' .. reason(state.pipeline_error, 'No pipeline for this branch'))
    mark(0, #lines[#lines], 'Comment')
  end
  blank()
  add ' o open in browser · l job log · v pipeline view · r refresh · q close'
  mark(0, #lines[#lines], 'Comment')
  -- what `o` opens on a line that links nowhere
  links.default = mr and mr.web_url or pipeline and pipeline.web_url
  return lines, marks, links
end

-- glab ci view (the pipeline TUI: logs, retry, cancel) in a float, like ;g's lazygit.
function M.pipeline_view(root)
  root = root or repo()
  if root then
    vim.cmd('FloatermNew --width=0.8 --height=0.8 --title=Pipeline --cwd=' .. vim.fn.fnameescape(root) .. ' glab ci view')
  end
end

local function job_log(root, job)
  vim.cmd(string.format('FloatermNew --width=0.8 --height=0.8 --title=Job --cwd=%s glab ci trace %d', vim.fn.fnameescape(root), job))
end

-- A window with the branch, its merge request and its pipeline. It refreshes every 10 seconds
-- while the pipeline runs.
function M.status()
  local root = repo()
  if not root then
    return
  end

  local buf = vim.api.nvim_create_buf(false, true)
  vim.bo[buf].bufhidden = 'wipe'
  local width = math.min(90, vim.o.columns - 4)
  local win = vim.api.nvim_open_win(buf, true, {
    relative = 'editor',
    width = width,
    height = 3,
    row = math.floor(vim.o.lines * 0.15),
    col = math.floor((vim.o.columns - width) / 2),
    border = 'rounded',
    title = ' GitLab ',
    title_pos = 'center',
    style = 'minimal',
  })
  vim.wo[win].cursorline = true
  local ns = vim.api.nvim_create_namespace 'fuentastic-gitlab'
  local links, timer = {}, vim.uv.new_timer()

  local function show(lines, marks)
    if not vim.api.nvim_buf_is_valid(buf) then
      return
    end
    vim.bo[buf].modifiable = true
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
    vim.bo[buf].modifiable = false
    vim.api.nvim_buf_clear_namespace(buf, ns, 0, -1)
    for _, m in ipairs(marks or {}) do
      vim.api.nvim_buf_set_extmark(buf, ns, m[1], m[2], { end_col = math.min(m[3], #lines[m[1] + 1]), hl_group = m[4] })
    end
    if vim.api.nvim_win_is_valid(win) then
      vim.api.nvim_win_set_height(win, math.min(#lines, math.floor(vim.o.lines * 0.7)))
    end
  end

  local function refresh()
    local state, pending = {}, 3
    local function done()
      pending = pending - 1
      if pending > 0 then
        return
      end
      local lines, marks
      lines, marks, links = render(state)
      show(lines, marks)
      if state.pipeline and active[state.pipeline.status] and not timer:is_closing() then
        timer:start(10000, 0, vim.schedule_wrap(refresh))
      end
    end
    local function pipeline_and_approvals()
      pending = pending + 1
      local cmd = { 'glab', 'ci', 'get', '--output', 'json' }
      if state.mr then
        vim.list_extend(cmd, { '--merge-request', tostring(state.mr.iid) })
        pending = pending + 1
        run({ 'glab', 'api', string.format('projects/:id/merge_requests/%d/approvals', state.mr.iid) }, root, function(out)
          state.approvals = decode(out)
          done()
        end)
      end
      run(cmd, root, function(out, err)
        state.pipeline, state.pipeline_error = decode(out), err
        done()
      end)
    end

    run({ 'git', 'branch', '--show-current' }, root, function(out)
      state.branch = out and vim.trim(out)
      done()
    end)
    run({ 'git', 'rev-list', '--left-right', '--count', '@{upstream}...HEAD' }, root, function(out)
      state.upstream = out
      done()
    end)
    run({ 'glab', 'mr', 'view', '--output', 'json' }, root, function(out, err)
      state.mr, state.mr_error = decode(out), err
      pipeline_and_approvals()
      done()
    end)
  end

  local function under_cursor()
    return links[vim.api.nvim_win_get_cursor(win)[1]]
  end
  local function map(key, fn)
    vim.keymap.set('n', key, fn, { buffer = buf, nowait = true })
  end
  map('q', '<Cmd>close<CR>')
  map('<Esc>', '<Cmd>close<CR>')
  map('r', refresh)
  map('o', function()
    local link = under_cursor()
    local url = type(link) == 'table' and link.url or type(link) == 'string' and link or links.default
    if url then
      vim.ui.open(url)
    end
  end)
  map('l', function()
    local link = under_cursor()
    if type(link) == 'table' and link.job then
      job_log(root, link.job)
    else
      notify('Put the cursor on a job', vim.log.levels.WARN)
    end
  end)
  map('v', function()
    vim.cmd 'close'
    M.pipeline_view(root)
  end)
  vim.api.nvim_create_autocmd('BufWipeout', {
    buffer = buf,
    callback = function()
      timer:stop()
      timer:close()
    end,
  })

  show { ' Asking GitLab…' }
  refresh()
end

return M

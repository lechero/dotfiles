-- SonarQube issues as diagnostics in Java and TypeScript. A project with a .sonarlint/connectedMode.json
-- in its git root is checked against its own rules on the SonarQube server (connected mode); any other
-- project gets Sonar's default rules.
local mason = vim.fn.stdpath('data') .. '/mason'

-- The file SonarQube for IDE in VS Code and IntelliJ writes when a team shares its binding:
-- { "sonarQubeUri": "https://sonar.example.com", "projectKey": "group:artifact" }
local function read_binding(root)
  local path = root .. '/.sonarlint/connectedMode.json'
  local file = io.open(path)
  if not file then
    return nil
  end
  local ok, binding = pcall(vim.json.decode, file:read('*a'))
  file:close()
  if ok and type(binding) == 'table' and binding.sonarQubeUri and binding.projectKey then
    return binding
  end
  vim.notify(path .. ' needs a sonarQubeUri and a projectKey; using the default rules', vim.log.levels.WARN)
end

-- A token per server from the macOS keychain, else $SONAR_TOKEN. Store one with
--   security add-generic-password -s sonarqube -a sonar.example.com -w
local function get_token(_, url)
  local host = url:match('^%a+://([^/:]+)')
  if host and vim.fn.executable('security') == 1 then
    local result = vim.system({ 'security', 'find-generic-password', '-s', 'sonarqube', '-a', host, '-w' }, { text = true }):wait()
    if result.code == 0 then
      return vim.trim(result.stdout)
    end
  end
  -- the server needs an answer; an empty token makes sonarlint.nvim warn and fall back to local mode
  return vim.env.SONAR_TOKEN or ''
end

return {
  url = 'https://gitlab.com/schrieveslaach/sonarlint.nvim',
  config = function()
    local analyzers = mason .. '/share/sonarlint-analyzers/'

    require('sonarlint').setup({
      server = {
        cmd = {
          mason .. '/bin/sonarlint-language-server',
          '-stdio',
          '-analyzers',
          analyzers .. 'sonarjava.jar',
          analyzers .. 'sonarjavasymbolicexecution.jar',
          analyzers .. 'sonarjs.jar',
        },
        -- one server per git root, so each project binds to its own SonarQube project
        before_init = function(_, config)
          local binding = read_binding(config.root_dir)
          if not binding then
            return
          end
          local url = binding.sonarQubeUri:gsub('/+$', '')
          local connection = url:match('^%a+://([^/:]+)') or url
          config.settings.sonarlint.connectedMode = {
            connections = { sonarqube = { { connectionId = connection, serverUrl = url } } },
            project = { connectionId = connection, projectKey = binding.projectKey },
          }
        end,
      },
      connected = { get_credentials = get_token },
      filetypes = { 'java', 'typescript', 'typescriptreact', 'javascript', 'javascriptreact' },
    })
  end,
}

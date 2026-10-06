-- SonarQube issues as diagnostics in Java and TypeScript. A project with a .sonarlint/connectedMode.json
-- in its git root is checked against its own rules on its SonarQube server or SonarQube Cloud
-- organization (connected mode); any other project gets Sonar's default rules.
local mason = vim.fn.stdpath('data') .. '/mason'

-- The file SonarQube for IDE in VS Code and IntelliJ writes when a team shares its binding:
--   { "sonarQubeUri": "https://sonar.example.com", "projectKey": "group:artifact" }
--   { "sonarCloudOrganization": "my-org", "projectKey": "group:artifact", "region": "EU" }
local function read_binding(root)
  local path = root .. '/.sonarlint/connectedMode.json'
  local file = io.open(path)
  if not file then
    return nil
  end
  local ok, binding = pcall(vim.json.decode, file:read('*a'))
  file:close()
  if ok and type(binding) == 'table' and binding.projectKey and (binding.sonarQubeUri or binding.sonarCloudOrganization) then
    return binding
  end
  vim.notify(path .. ' needs a projectKey, and a sonarQubeUri or sonarCloudOrganization; using the default rules', vim.log.levels.WARN)
end

-- A token per server host from the macOS keychain, else $SONAR_TOKEN. Store one with
--   security add-generic-password -s sonarqube -a sonar.example.com -w
-- For SonarQube Cloud the host is sonarcloud.io (sonarqube.us in the US region): the server asks
-- for its token by region and organization (EU_my-org) rather than by URL.
local cloud_hosts = { EU = 'sonarcloud.io', US = 'sonarqube.us' }
local function get_token(_, server)
  local host = server:match('^%a+://([^/:]+)') or cloud_hosts[server:match('^(%u%u)_')]
  if host and vim.fn.executable('security') == 1 then
    local result = vim.system({ 'security', 'find-generic-password', '-s', 'sonarqube', '-a', host, '-w' }, { text = true }):wait()
    if result.code == 0 then
      return vim.trim(result.stdout)
    end
  end
  -- the server needs an answer; an empty token makes sonarlint.nvim warn and fall back to local mode
  return vim.env.SONAR_TOKEN or ''
end

-- sonarlint.nvim's two connection notices read connections.sonarcloud as one connection, but the
-- server takes a list, like connections.sonarqube; with a SonarQube Cloud binding they'd fail on a
-- nil URL. Replaced before any server starts: each one takes the handlers when it starts.
local function fix_connection_notices()
  local connected_mode = require('sonarlint.connected_mode')
  local function describe(ctx, id)
    local client = vim.lsp.get_client_by_id(ctx.client_id)
    local connections = client and vim.tbl_get(client.config.settings, 'sonarlint', 'connectedMode', 'connections') or {}
    for _, c in ipairs(connections.sonarqube or {}) do
      if c.connectionId == id then
        return c.serverUrl
      end
    end
    for _, c in ipairs(connections.sonarcloud or {}) do
      if c.connectionId == id then
        return 'SonarQube Cloud organization ' .. c.organizationKey
      end
    end
    return id
  end
  connected_mode.notify_connection_result = function(_, params, ctx)
    connected_mode._connected_clients[ctx.client_id] = params.success and 'connected' or 'failed-connection'
    if not params.success then
      vim.notify_once(('Cannot connect to %s: %s'):format(describe(ctx, params.connectionId), params.reason or ''), vim.log.levels.ERROR)
    end
  end
  connected_mode.notify_invalid_token = function(_, params, ctx)
    vim.notify(('Cannot connect to %s: the token is invalid'):format(describe(ctx, params.connectionId)), vim.log.levels.WARN)
  end
end

return {
  url = 'https://gitlab.com/schrieveslaach/sonarlint.nvim',
  config = function()
    local analyzers = mason .. '/share/sonarlint-analyzers/'
    fix_connection_notices()

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
          if binding.sonarCloudOrganization then
            local organization = binding.sonarCloudOrganization
            config.settings.sonarlint.connectedMode = {
              connections = {
                sonarcloud = { { connectionId = organization, organizationKey = organization, region = (binding.region or 'EU'):upper() } },
              },
              project = { connectionId = organization, projectKey = binding.projectKey },
            }
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

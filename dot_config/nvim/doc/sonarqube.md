# SonarQube

Java, TypeScript and JavaScript buffers get SonarQube's issues as diagnostics, through
[sonarlint.nvim](https://gitlab.com/schrieveslaach/sonarlint.nvim) and the language server behind
SonarQube for IDE. A project bound to your SonarQube server or SonarQube Cloud organization is
checked against that project's own rules (its quality profile); any other project gets Sonar's
default rules. The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/plugins/sonarlint.lua` | Starts the server, binds projects to SonarQube (Server or Cloud), looks up tokens |
| `lua/fuentastic/config/lsp.lua` | Mason installs `sonarlint-language-server` |

## Requirements

- Java 17+ on `PATH` to run the server. The JDK 21 that jdtls needs covers it.
- Node 20.12+ on `PATH` for TypeScript and JavaScript.
- For Java: jdtls (see [java.md](java.md)). Sonar gets the project's classpath from it.

Mason installs the server (about 340 MB) on first start. Open a file after it finishes, or
restart nvim.

## Binding a project to your SonarQube server

1. Put a `.sonarlint/connectedMode.json` in the project's git root. It's the file SonarQube for
   IDE writes in VS Code and IntelliJ when a team shares its setup, so some projects already have
   one. For a SonarQube server:

   ```json
   { "sonarQubeUri": "https://sonar.example.com", "projectKey": "group:artifact" }
   ```

   For SonarQube Cloud (sonarcloud.io), the organization instead, and `"US"` as the region for
   sonarqube.us:

   ```json
   { "sonarCloudOrganization": "my-org", "projectKey": "group:artifact", "region": "EU" }
   ```

   The project key is on the project's *Project Information* page in SonarQube, or in the CI
   scan's `-Dsonar.projectKey`. A `sonar-project.properties` with
   `sonar.projectKey=<project.groupId>:<project.artifactId>` means the groupId and artifactId of the
   root `pom.xml`; `sonar.organization` there is the SonarQube Cloud organization. If the team doesn't commit `.sonarlint/`, keep yours out of git
   with `echo .sonarlint >> .git/info/exclude`.

2. Create a user token in SonarQube (*My Account → Security*) and store it in the macOS keychain,
   one per server. With `-w` last, `security` prompts for the token so it stays out of your shell
   history:

   ```sh
   security add-generic-password -s sonarqube -a sonar.example.com -w
   ```

   For SonarQube Cloud, create the token on sonarcloud.io (*My Account → Security*) and store it
   under the host `sonarcloud.io` (`sonarqube.us` in the US region); it works for every
   organization you're a member of:

   ```sh
   security add-generic-password -s sonarqube -a sonarcloud.io -w
   ```

   Without a keychain entry for the server's host, the `SONAR_TOKEN` environment variable is used.

3. Open a Java or TypeScript file in the project. The message "SonarQube language server is ready
   and running in connected mode" confirms the binding; the first sync downloads the project's
   rules.

## Working with issues

Sonar issues are ordinary diagnostics: `]d` / `[d`, the floating diagnostic, and Trouble all show
them, with the rule key (e.g. `java:S1854`) as the code. `<leader>ca` on an issue offers:

| Code action | Does |
| --- | --- |
| SonarQube: Show issue details | The rule's description and examples |
| SonarQube: Deactivate rule | Turns the rule off until nvim restarts; in a bound project the server's rules win |

sonarlint.nvim's own handling of SonarQube Cloud connections reads them as one connection
rather than a list, and errors on its connection notices; `plugins/sonarlint.lua` replaces those
two notices.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| No Sonar issues anywhere | `:checkhealth vim.lsp` should list `sonarlint.nvim`; check `:Mason` finished installing it |
| "get_credentials returned nil: starting in local mode" | No keychain entry for that host and no `SONAR_TOKEN` |
| "Cannot connect to ..." | The `sonarQubeUri`, your VPN, and that the token is still valid |
| A bound project shows no Sonar issues | It's only analysed with the server's rules, so it needs to reach the server |
| No TypeScript issues, Java works | `node` wasn't on `PATH` when nvim started |
| "Cannot find sonarlint client that matches to Java LSP client" | The Java build (`mvnw`, `gradlew`) is in a subfolder: sonarlint.nvim expects jdtls's root to be the git root |

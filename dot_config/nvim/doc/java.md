# Java development

Java buffers get [jdtls](https://github.com/eclipse-jdtls/eclipse.jdt.ls) through
[nvim-jdtls](https://codeberg.org/mfussenegger/nvim-jdtls), plus debugging and JUnit/TestNG
runs through [nvim-dap](https://github.com/mfussenegger/nvim-dap). The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/plugins/jdtls.lua` | JDK discovery, jdtls start/settings, java keymaps |
| `jdtls/lifecycle-mapping-metadata.xml` | Maven plugin goals jdtls's builds skip (see [Notes](#notes)) |
| `lua/fuentastic/plugins/dap.lua` | Debugger core, dap-ui, debug keymaps (all languages) |
| `lua/fuentastic/plugins/treesitter.lua` | `java`, `xml`, `groovy`, `kotlin`, `properties`, `yaml` parsers |
| `after/ftplugin/java.lua` | 4-space indent when vim-sleuth can't detect one |
| `lua/fuentastic/config/lsp.lua` | Mason installs `jdtls`, `java-debug-adapter`, `java-test` |
| `lua/fuentastic/plugins/conform.lua` | Formats with the project's Spotless profile or IntelliJ code style |

## Requirements

- A JDK 21+ to run jdtls, whatever version your projects target. SDKMAN is the expected
  source:

  ```sh
  sdk install java 21-tem
  ```

- `python3` on `PATH` (mason's `jdtls` launcher is a Python script).
- `tree-sitter` CLI and a C compiler, to build the treesitter parsers.

Mason installs jdtls and the debug/test extensions on first start. They are loaded when jdtls
starts, so after that very first install run `:JdtRestart` once.

## Which JDK is used

Every JDK under `~/.sdkman/candidates/java`, `/Library/Java/JavaVirtualMachines`,
`~/Library/Java/JavaVirtualMachines` (IntelliJ downloads), Homebrew's `openjdk*` and
`/usr/lib/jvm` is registered with jdtls as a runtime (`JavaSE-17`, `JavaSE-21`, ...). A project
that sets `<maven.compiler.release>21</maven.compiler.release>` (or a Gradle toolchain) builds
against the matching JDK.

- **Server:** `JAVA_HOME` if it's 21+, otherwise the oldest installed 21+.
- **Default runtime** (plain `.java` files, projects that don't pin a version): `JAVA_HOME`,
  otherwise the server's JDK. `sdk use java <version>` before starting nvim changes it.
- `:JdtSetRuntime` switches a running project to another runtime.

## Keymaps

Java buffers (on top of the usual LSP maps: `gd`, `gr`, `gI`, `<leader>rn`, `<leader>ca`, ...):

| Keys | Action |
| --- | --- |
| `<leader>co` | Organize imports |
| `<leader>cxv` / `<leader>cxc` | Extract variable / constant (normal or visual) |
| `<leader>cxm` | Extract method (visual) |
| `gS` | Go to super implementation |
| `<leader>cu` | Re-import the project's `pom.xml` / `build.gradle` |
| `<leader>tc` | Run the test class |
| `<leader>tm` | Run the test method under the cursor |
| `<leader>tp` | Pick a test to run |
| `<leader>tl` | Re-run the last test or debug session |
| `<leader>th` | Toggle inlay hints (parameter names) |
| `<leader>cf` | Format the buffer or selection (see [Formatting](#formatting)) |

Code generation (constructors, `toString`, `equals`/`hashCode`, delegate methods, overriding or
implementing methods) is under `<leader>ca`.

Debugging, in any language:

| Keys | Action |
| --- | --- |
| `<F5>` | Start (pick a main class) / continue |
| `<S-F5>` | Terminate |
| `<F1>` / `<F2>` / `<F3>` | Step into / over / out |
| `<leader>b` / `<leader>B` | Toggle breakpoint / conditional breakpoint |
| `<F7>` | Toggle the debugger UI |

## Debugging

`<F5>` in a java buffer lists every main class jdtls finds. For program arguments, env vars or
remote attach, add a `.vscode/launch.json` to the project and it shows up in the same list.

Saving a file during a debug session hot-swaps the changed classes into the running program.

The debugger UI opens with a session and stays open after it ends, so program output and test
results stay readable. Close it with `<F7>`.

## Tests

`<leader>tc` / `<leader>tm` run JUnit 4/5/6 or TestNG tests under the debugger, so breakpoints
in tests and code under test are hit. Afterwards each test gets a ✓ / ✗ mark, failures go to the
quickfix list (`:copen`) and the full report is in the dap REPL.

## Formatting

The project decides how java is formatted (`lua/fuentastic/plugins/conform.lua`):

| The project has | `<leader>cf` uses | On save |
| --- | --- | --- |
| Spotless with an Eclipse profile | jdtls with that profile | yes |
| An IntelliJ code style in `.idea/codeStyles/Project.xml` | IntelliJ's formatter (~3s) | no |
| Neither | jdtls with Eclipse's built-in profile | no |

In visual mode `<leader>cf` only formats the selection. That helps in codebases that have drifted
from their own style, where formatting a whole file rewrites lines you never touched.

### Spotless projects

The profile is read from the Spotless config: `eclipse().configFile('config/style.xml')` in
`build.gradle(.kts)`, or `<eclipse><file>...</file></eclipse>` in `pom.xml`. jdtls runs the same
Eclipse formatter as Spotless, so the result matches `spotlessCheck`. The rare exception comes
from Spotless using an older Eclipse release than jdtls; `./gradlew spotlessApply` settles it.

### IntelliJ code style projects

For a scheme you were handed as an exported `.xml`, make it the project's code style, either in
IntelliJ (Settings > Editor > Code Style > Java, scheme "Project", gear icon > Import Scheme) or
by hand: wrap it as `.idea/codeStyles/Project.xml`

```xml
<component name="ProjectCodeStyleConfiguration">
  <code_scheme name="Project" version="173">
    <!-- the <option>, <JavaCodeStyleSettings>, ... elements of the exported scheme -->
  </code_scheme>
</component>
```

and add `.idea/codeStyles/codeStyleConfig.xml` so IntelliJ uses it too:

```xml
<component name="ProjectCodeStyleConfiguration">
  <state>
    <option name="USE_PER_PROJECT_SETTINGS" value="true" />
  </state>
</component>
```

This needs IntelliJ IDEA (Community is enough) in `/Applications` or `~/Applications`. The
formatter keeps its own settings and caches in `~/.cache/nvim/intellij-format`, so it works
while the IDE is open.

## Notes

- A project whose Maven build runs checkstyle gets its violations while editing, with the build's
  own config and suppressions: see [linting.md](linting.md#checkstyle).
- Opening a new `.java` file writes it to disk right away. nvim-jdtls does this because jdtls
  ignores files that don't exist yet.
- Each project gets its own jdtls workspace in `~/.cache/nvim/jdtls/<project>-<hash>`.
- jdtls also writes Eclipse project files (`.project`, `.classpath`, `.factorypath`, `.settings/`)
  into every project it imports. The global git ignore (`dot_config/private_git/ignore`) keeps
  them out of `git status` in repos that don't ignore them.
- jdtls's builds don't run frontend-maven-plugin (`npm install`, the frontend build, JS tests):
  the plugin asks to run on every build, and a test run that never exits stalls the project
  import. Build the frontend from the shell. Other plugin goals to skip go in
  `jdtls/lifecycle-mapping-metadata.xml`; after changing it, `:JdtRestart`.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| Red imports after changing `pom.xml` / `build.gradle` | `<leader>cu` |
| Stale or strange errors across the project | `:JdtWipeDataAndRestart` |
| `<F5>` or `<leader>tc` do nothing | `:JdtRestart` (extensions installed after jdtls started) |
| Import stuck ("Synchronizing projects 0%") | A Maven plugin's process hangs: `pgrep -lf -P "$(pgrep -d, -f jdt.ls.core)"` lists what jdtls started; skip that plugin in `jdtls/lifecycle-mapping-metadata.xml` |
| Server errors | `:JdtShowLogs` |
| IntelliJ format fails with "Only one instance" | A previous `<leader>cf` is still running |
| Formatting doesn't run | `:ConformInfo` |
| No java highlighting | `:checkhealth nvim-treesitter` |

Parsers built by the old `master` branch of nvim-treesitter may still be in
`~/.local/share/nvim/lazy/nvim-treesitter/parser/`. The current ones in
`~/.local/share/nvim/site/parser/` take precedence, so the old directory can be deleted.

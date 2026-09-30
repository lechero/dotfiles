# Java development

Java buffers get [jdtls](https://github.com/eclipse-jdtls/eclipse.jdt.ls) through
[nvim-jdtls](https://codeberg.org/mfussenegger/nvim-jdtls), plus debugging and JUnit/TestNG
runs through [nvim-dap](https://github.com/mfussenegger/nvim-dap). The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/plugins/jdtls.lua` | JDK discovery, jdtls start/settings, java keymaps |
| `lua/fuentastic/plugins/dap.lua` | Debugger core, dap-ui, debug keymaps (all languages) |
| `lua/fuentastic/plugins/treesitter.lua` | `java`, `xml`, `groovy`, `kotlin`, `properties`, `yaml` parsers |
| `after/ftplugin/java.lua` | 4-space indent when vim-sleuth can't detect one |
| `lua/fuentastic/config/lsp.lua` | Mason installs `jdtls`, `java-debug-adapter`, `java-test` |

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
| `<leader>f` | Format with jdtls |

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

Java isn't formatted on save: jdtls formats with Eclipse's built-in profile unless the project
has its own Eclipse formatter settings, and that would reformat files in projects that use
another style. Format by hand with `<leader>f`. To format on save anyway, drop `java` from
`disable_filetypes` in `lua/fuentastic/plugins/conform.lua`.

## Notes

- Opening a new `.java` file writes it to disk right away. nvim-jdtls does this because jdtls
  ignores files that don't exist yet.
- Each project gets its own jdtls workspace in `~/.cache/nvim/jdtls/<project>-<hash>`.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| Red imports after changing `pom.xml` / `build.gradle` | `<leader>cu` |
| Stale or strange errors across the project | `:JdtWipeDataAndRestart` |
| `<F5>` or `<leader>tc` do nothing | `:JdtRestart` (extensions installed after jdtls started) |
| Server errors | `:JdtShowLogs` |
| No java highlighting | `:checkhealth nvim-treesitter` |

Parsers built by the old `master` branch of nvim-treesitter may still be in
`~/.local/share/nvim/lazy/nvim-treesitter/parser/`. The current ones in
`~/.local/share/nvim/site/parser/` take precedence, so the old directory can be deleted.

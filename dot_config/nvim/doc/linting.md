# Linting

The editor runs the linters a project's own build runs, with the project's own configs and, for
the JavaScript tools, its own installed versions: what nvim reports is what CI reports. A linter
only runs in a project that uses it. The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/plugins/lint.lua` | checkstyle, Stylelint and html-validate through nvim-lint |
| `lua/fuentastic/config/lsp.lua` | The ESLint, XML (lemminx) and JSON language servers |

| Linter | Runs on | When the project has | When |
| --- | --- | --- | --- |
| checkstyle | Java in `src/main/java` | maven-checkstyle-plugin with a `configLocation` in a `pom.xml` | Opening and saving |
| ESLint | JS, TS, Vue | An `eslint.config.*` (or an `.eslintrc*`) | As you type |
| Stylelint | CSS, SCSS, Less | A `.stylelintrc*` or `stylelint.config.*`, and stylelint in `node_modules` | Opening and saving |
| html-validate | HTML under `jcr_root/` (HTL) | An `.htmlvalidate.*`, and html-validate in `node_modules` | Opening and saving |

Their findings are diagnostics like any other: `]d` / `[d`, `<C-w>d`, `<leader>sd`, `<leader>xx`.

## checkstyle

As the Maven build runs it:

- the `configLocation` and `suppressionsLocation` of the nearest `pom.xml` that configures the
  plugin (a module's, or the parent's);
- only the module's `src/main/java`: Maven doesn't check tests;
- not the files its `<excludes>` lists.

The checkstyle CLI has no suppressions option, so the config is copied to
`~/.cache/nvim/checkstyle/` with a `SuppressionFilter` added, the way the Maven plugin adds one.
Mason installs checkstyle; it starts a JVM (about half a second), hence opening and saving only.

Mason's checkstyle is its latest release, while a project pins its own in the plugin's
dependencies. Checks rarely change between versions; if the build reports something the editor
doesn't, the version is the first suspect. A `configLocation` that's a classpath resource (a
shared config jar, `google_checks.xml`) can't be read outside Maven, so such projects aren't
linted.

## ESLint

The vscode-eslint language server, with the project's ESLint from `node_modules`: its rules, its
ignores (an ignored file shows nothing, as in CI), and the `--flag`s the project's lint scripts
pass (`eslint ./ --flag v10_config_lookup_from_file` makes each file use its nearest config).

| Keys | Does |
| --- | --- |
| `<leader>ca` | Fix this problem, fix all of its kind, or disable the rule for the line |
| `<leader>ce` | Apply every fix ESLint can make in the file |

ESLint doesn't format on save: its fixes rewrite code (`let` to `const`, for one), so they're
`<leader>ce`. Saving formats as before (prettier where the project has it, else `ts_ls`).

## Stylelint and html-validate

The project's own binaries, run from the folder with their config, so its `extends`, plugins and
ignore files apply. html-validate checks the saved file, which is what makes the project's
`.htmlvalidateignore` apply (stdin skips it); Stylelint checks the buffer.

## XML and JSON

- **lemminx** reports XML that isn't well-formed, closes tags as you type, and validates and
  completes `pom.xml` (and anything else that declares a schema or DTD) against it.
- **jsonls** checks `package.json`, `tsconfig.json`, `renovate.json`, `.stylelintrc`, and the
  other files [SchemaStore](https://www.schemastore.org) has a schema for.

Neither formats: AEM's XML and OSGi configs stay as written.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| No checkstyle diagnostics in a Maven project | Is the file under `src/main/java`, and does `:Mason` show checkstyle? |
| ESLint shows nothing for a file | `npx eslint <file>` in the project: it may be ignored there too |
| "Unable to find ESLint library" | `npm install` in the project |
| No Stylelint or html-validate diagnostics | `npm install` (they're the project's own); then save the file |
| `:checkhealth vim.lsp` lists eslint but nothing shows | `:LspLog`: the project's config may fail to load |

# Linting

The editor runs the linters a project's own build runs, with the project's own configs and, for
the JavaScript tools, its own installed versions: what nvim reports is what CI reports. A linter
only runs in a project that uses it. The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/plugins/lint.lua` | checkstyle, Stylelint, html-validate and Nunjucks through nvim-lint |
| `scripts/nunjucks-lint.js` | Checks a Backstage template the way Backstage renders it |
| `lua/fuentastic/config/lsp.lua` | The ESLint, XML (lemminx), JSON and YAML language servers |

| Linter | Runs on | When the project has | When |
| --- | --- | --- | --- |
| checkstyle | Java in `src/main/java` | maven-checkstyle-plugin with a `configLocation` in a `pom.xml` | Opening and saving |
| ESLint | JS, TS, Vue | An `eslint.config.*` (or an `.eslintrc*`) | As you type |
| Stylelint | CSS, SCSS, Less | A `.stylelintrc*` or `stylelint.config.*`, and stylelint in `node_modules` | Opening and saving |
| html-validate | HTML under `jcr_root/` (HTL) | An `.htmlvalidate.*`, and html-validate in `node_modules` | Opening and saving |
| Nunjucks | Files with Nunjucks markup in a Backstage template's `skeleton/` | Nothing: npm installs Nunjucks the first time | Opening and saving |

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

## Backstage templates

A Backstage software template is a `template.yaml` and a `skeleton/` folder. The template's
`fetch:template` step renders every file in that folder with
[Nunjucks](https://mozilla.github.io/nunjucks/templating.html): `${{ values.name }}` puts a value
in, and `{% for %}` and `{% if %}` repeat or leave out lines. So a `vm.json` in there isn't JSON,
it's a template that renders JSON, and the JSON server would flag every `{% %}` as broken JSON.

A file in a `skeleton/` folder with Nunjucks markup in it (`ftdetect/backstage.lua`) is therefore:

- **filetype `jinja`** (Nunjucks has Jinja's syntax), so jsonls and yamlls stay away from it. Its
  markup is highlighted as Jinja, and the text around the markup in the language of the file it
  renders: JSON for `vm.json`, YAML for `catalog-info.yaml` (`after/queries/jinja/injections.scm`).
- **linted by Nunjucks**, set up as `fetch:template` sets it up (`${{ }}` for values):
  - Nunjucks syntax, as errors: a `{% for %}` without its `{% endfor %}`, an unclosed `${{`, an
    expression that doesn't parse.
  - For JSON, the rendered JSON, as warnings. Every value is a placeholder: the string `"0"` when
    it's printed, a list of two when it's looped over. Each loop runs twice, so the commas between
    items (`{% if not loop.last %},{% endif %}`) are checked too, and the warning is put on the
    template's line. A placeholder equals hardly anything, so `{% if values.vendor == 'Infosys' %}` is
    false: this checks one way through the ifs, not every one.

Nunjucks and jsonc-parser are installed by npm in `~/.local/share/nvim/nunjucks` the first time a
template is opened, and need Node on the `PATH`.

The `template.yaml` gets Backstage's schema (SchemaStore's `catalog-info.json`, which has the
Template kind) from yamlls, as does any YAML with a `backstage.io` `apiVersion`. That's by
content, not by name: by name, SchemaStore gives every `template.yaml` AWS SAM's schema, which
rejects `apiVersion`, `kind`, `metadata` and `spec`.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| No checkstyle diagnostics in a Maven project | Is the file under `src/main/java`, and does `:Mason` show checkstyle? |
| ESLint shows nothing for a file | `npx eslint <file>` in the project: it may be ignored there too |
| "Unable to find ESLint library" | `npm install` in the project |
| No Stylelint or html-validate diagnostics | `npm install` (they're the project's own); then save the file |
| A Backstage template shows no Nunjucks diagnostics | Is `~/.local/share/nvim/nunjucks/node_modules` there? Delete it and restart nvim: opening a template installs it again |
| A file in `skeleton/` is still `json` or `yaml` | It has no `${{` or `{%` in its first 500 lines, so it renders as is |
| `:checkhealth vim.lsp` lists eslint but nothing shows | The LSP log, at the "Log path" `:checkhealth vim.lsp` shows (`~/.local/state/nvim/logs/lsp.log`): the project's config may fail to load |

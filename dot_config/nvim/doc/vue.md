# Vue development

`.vue` files get two language servers: [vue_ls](https://github.com/vuejs/language-tools) for the
template and styles, and `ts_ls` for everything TypeScript (types, hover, `gd`, completion), in
`<script>` and in template expressions alike. `ts_ls` understands `.vue` files through the
`@vue/typescript-plugin` that ships with vue-language-server, and vue_ls forwards its own
TypeScript questions (component names, props) to `ts_ls`. Vue 3 only: Vue's language tools
dropped Vue 2. The config lives in:

| File | What it does |
| --- | --- |
| `lua/fuentastic/config/lsp.lua` | Mason installs `vue-language-server`; `ts_ls` loads the Vue plugin |
| `lua/fuentastic/plugins/treesitter.lua` | `vue`, `javascript`, `typescript`, `tsx`, `css`, `scss` parsers |
| `lua/fuentastic/plugins/conform.lua` | Formats with prettier in projects that configure it |

## Requirements

- Node 22+ (`npm`), for the language servers.
- The project's dependencies installed (`npm install`): `ts_ls` uses the project's TypeScript.
- `tree-sitter` CLI and a C compiler, to build the treesitter parsers.

Mason installs vue-language-server on first start, but `ts_ls` only loads the Vue plugin when it
starts. So after that very first install run `:LspRestart` once (or restart nvim).

## Which TypeScript is used

The project's `node_modules/typescript`, otherwise the TypeScript 6 that Mason installs with
`ts_ls`. TypeScript 7 ships no `tsserver`, so a project on TypeScript 7 gets Mason's 6 in the
editor. Projects following TypeScript's migration path (`typescript` aliased to
`@typescript/typescript6`) keep their own.

## Formatting

| The project has | `<leader>f` and on save use |
| --- | --- |
| A prettier config (`.prettierrc*`, `prettier.config.*`, or `"prettier"` in `package.json`) | prettier, the project's own from `node_modules` |
| None | `ts_ls` / `vue_ls` |

That applies to `.vue`, `.js`, `.ts`, `.css` and `.scss` files. The language servers ignore a
prettier config, so without this a `semi: false, singleQuote: true` project would get
semicolons and double quotes on every save.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| Template and styles work, but no types or hover in `<script>` | `:LspRestart` (vue-language-server installed after `ts_ls` started) |
| "Could not find `ts_ls`... required by `vue_ls`" | `ts_ls` isn't attached; check `:checkhealth vim.lsp` |
| Errors about missing types everywhere | `npm install` in the project, then `:LspRestart` |
| Formatting doesn't run, or prettier isn't used | `:ConformInfo` |
| No highlighting inside `<script>` / `<style>` | `:checkhealth nvim-treesitter` |

# GitLab

GitLab CI files get three layers of checking, and one key shows where the current branch is at:
its merge request and its pipeline. The config lives in:

| File | What it does |
| --- | --- |
| `ftdetect/gitlab-ci.lua` | GitLab CI files are `yaml.gitlab` |
| `lua/fuentastic/config/lsp.lua` | yamlls with GitLab's CI schema, and gitlab-ci-ls |
| `lua/fuentastic/gitlab.lua` | `glab ci lint`, the status window, the pipeline TUI |
| `lua/fuentastic/config/gitlab.lua` | The commands and keymaps |

## Requirements

- [glab](https://gitlab.com/gitlab-org/cli), logged in to your GitLab: `brew install glab`, then
  `glab auth login`. For the lint against GitLab and for the status window.
- Rust 1.85+ (`rustup update stable`): Mason builds gitlab-ci-ls with cargo.
- Node, for yaml-language-server.

Mason installs both language servers on first start; gitlab-ci-ls takes a few minutes to build.

## Which files

`.gitlab-ci.yml`, any `*.gitlab-ci.yml` and anything under `.gitlab/ci/` are `yaml.gitlab`: YAML
for everything else (treesitter, indent, `gc`), and the GitLab CI language server attaches to them.
For a file the pipeline includes from elsewhere, set the filetype by hand: `:set ft=yaml.gitlab`.

Other YAML files get yaml-language-server too, checked against whatever schema
[SchemaStore](https://www.schemastore.org) has for them: docker-compose files, GitHub workflows,
OpenAPI specs and the like.

## Linting

| Layer | Checks | When |
| --- | --- | --- |
| yaml-language-server, with GitLab's CI schema | Unknown keys, wrong types, invalid values; completion and hover docs for every keyword | As you type |
| gitlab-ci-ls | `extends` to a job that doesn't exist, a `stage` nobody declared, `needs`, components | As you type |
| `glab ci lint` on GitLab | The whole config as GitLab merges it, includes and components resolved | `<leader>gl` |

GitLab's `!reference` tag is declared to yamlls, so it isn't flagged. YAML isn't formatted on save:
yamlls would rewrite CI files.

`<leader>gl` sends the repo's root `.gitlab-ci.yml` to GitLab, unsaved changes included, and puts
any errors in the quickfix list at the job they're about. Included files are read from GitLab
itself, so changes to them count once they're pushed. `<leader>gL` also simulates creating a
pipeline for the current branch (`--dry-run`), which runs the `rules`; the branch has to be pushed.

### Navigating

The usual LSP keys work across the included files:

| Keys | On | Does |
| --- | --- | --- |
| `gd` | `extends: .base`, `include: local: ...`, `needs: - job: build` | Opens the job or file |
| `gr` | A job | Where it's extended or needed |
| `K` | A job | The job as GitLab sees it, `extends` merged in |
| `<leader>rn` | A job | Renames it everywhere |

`gd` on a plain `needs: - build` entry finds nothing; write it `- job: build` to jump.

## Branch, merge request and pipeline

`<leader>gs` opens a window with:

- the branch, and how far it is ahead of or behind its remote;
- its merge request: draft or not, the target branch, approvals, conflicts, unresolved threads and
  whether it can merge;
- its pipeline (the merge request's, if there is one): the status and every job by stage, with
  durations. While the pipeline runs, the window refreshes every 10 seconds.

| Key in the window | Does |
| --- | --- |
| `o` | Opens the job, pipeline or merge request under the cursor in the browser |
| `l` | The job's log, in a float |
| `v` | `glab ci view`: the pipeline TUI, to retry or cancel jobs |
| `r` | Refreshes |
| `q` | Closes |

`;p` opens the pipeline TUI straight away, next to `;g` for lazygit.

## Commands

| Command | Keys |
| --- | --- |
| `:GitlabStatus` | `<leader>gs` |
| `:GitlabCiLint` / `:GitlabCiLint!` | `<leader>gl` / `<leader>gL` |
| `:GitlabPipeline` | `;p` |

## Troubleshooting

| Symptom | Try |
| --- | --- |
| "None of the git remotes ... point to a known GitLab host" | `glab auth login` for that host |
| No gitlab-ci-ls diagnostics | `:Mason`: did gitlab-ci-ls build? It needs Rust 1.85+ |
| "Unresolved tag" errors | The file isn't `yaml.gitlab`; `:set ft=yaml.gitlab` |
| Linting an included file says it's fine, but the pipeline fails | Push it: `<leader>gl` reads included files from GitLab |

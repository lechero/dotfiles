# AEM

What the config does for an AEM project, from the component scripts to the repository. No language
server understands HTL, JSP's AEM tags or JCR content XML, so most of this is the config's own
(`lua/fuentastic/htl.lua`, `jsp.lua`, `jcr.lua`, all resolving through `refs.lua`).

| In | `gd` opens | Guide |
| --- | --- | --- |
| HTL (`.html` under `jcr_root/`) | Components, `data-sly-use` classes and scripts, templates, clientlibs | [htl.md](htl.md) |
| JSP | Includes, classes, tag files, `cq:include` scripts, components, clientlibs | [jsp.md](jsp.md) |
| `.content.xml` | Components, repository nodes, clientlibs | Below |

## `gd` in content XML

| On the line | `gd` opens |
| --- | --- |
| `sling:resourceSuperType="site/components/page"`, `sling:resourceType="..."` | The component's `page.html` (or `.jsp`, else its `.content.xml`) |
| `cq:template="/apps/site/templates/home"`, any other absolute path | That node: its `.content.xml`, or the file |
| `extraClientlibs="[site.author]"`, `embed="[...]"`, `dependencies="[...]"` | The clientlib folder with that category |

Repository names are the FileVault way on disk: `cq:dialog` is the folder `_cq_dialog`. Granite,
Core Components and other `/libs` types aren't in the repo, so `gd` says "Not in this repo".

## Clientlibs

A category resolves to the `.content.xml` of the `cq:ClientLibraryFolder` that lists it in its
`categories`, found once per repo with `git grep`. That works from HTL
(`${clientlib.css @ categories='site.base'}`, a list too), JSP (`<cq:includeClientLib
categories="...">`) and content XML. A clientlib added since nvim started needs a restart to be
found.

## Open in AEM

`<leader>oa` (`:AemOpen`) opens the current file's node on the local author: a page (a `cq:Page`
under `/content`) in the page editor, anything else in CRXDE at that node. The author is the root
`pom.xml`'s `aem.host` and `aem.port`, else `localhost:4502`.

## Content XML

`.content.xml` and other XML get lemminx: well-formedness errors, closing tags, and a `pom.xml`
validated against its schema. It doesn't format: see [linting.md](linting.md#xml-and-json).

## The rest of the workflow

| What | Where |
| --- | --- |
| Java: Sling models, OSGi services, tests, debugging | [java.md](java.md) |
| checkstyle, ESLint, Stylelint and html-validate as CI runs them | [linting.md](linting.md) |
| Jest tests | [testing.md](testing.md) |
| SonarQube or SonarQube Cloud issues with the project's rules | [sonarqube.md](sonarqube.md) |
| The project's GitLab pipeline and merge requests | [gitlab.md](gitlab.md) |

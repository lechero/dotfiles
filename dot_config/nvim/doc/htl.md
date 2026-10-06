# HTL (AEM components)

No language server understands HTL, so `.html` files in an AEM content package (any path with a
`jcr_root/` folder) get their own `gd`. It opens what the cursor line references; on a line with
several references it takes the one under the cursor, else the first.

| On the line | `gd` opens |
| --- | --- |
| `resourceType='nlgeneric/components/content/title'` (or `/apps/...`) | The component's `title/title.html` (or `title.jsp`) |
| `data-sly-use.model="com.example.core.Model"` | `Model.java`, in any module's `src/main/java` |
| `data-sly-use.tpl="templates/card.html"`, `data-sly-use="${'helper.js'}"` | That script |
| `data-sly-include="partials/header.html"` | That script |
| `data-sly-call="${tpl.content}"` | The file `data-sly-use.tpl` loads, at `data-sly-template.content` |
| `data-sly-call="${content}"` | `data-sly-template.content` in this file |
| `${clientlib.css @ categories='site.base'}` (or a list) | The clientlib with that category ([aem.md](aem.md#clientlibs)) |
| Nothing of the above | Vim's own `gd` |

Paths resolve like Sling resolves them: relative to the current script first, then under `/apps` and
`/libs` in every content package of the repo (`*/src/main/content/jcr_root`). `/libs` components
that come from AEM itself, like the Core Components, aren't in the repo, so `gd` says "Not in this
repo". `<C-o>` jumps back. The code is in `lua/fuentastic/htl.lua`; resolving paths and classes is
shared with JSP's `gd` ([jsp.md](jsp.md)) in `lua/fuentastic/refs.lua`.

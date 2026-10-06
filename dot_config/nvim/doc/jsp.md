# JSP development

No language server understands JSP, and nothing upstream parses it for treesitter. So this config
brings its own: a small JSP grammar that hands the markup to the html parser and the Java in
scriptlets to the java one, a `gd` that opens what a line includes, imports or uses, an indent
that follows both the HTML and the Java, and JSP comments for `gc`. The config lives in:

| File | What it does |
| --- | --- |
| `tree-sitter-jsp/` | The JSP grammar: tags, directives, comments, EL; its generated parser in `src/` |
| `queries/jsp/` | Highlights, the html and java injections, the comment strings for `gc` |
| `lua/fuentastic/plugins/treesitter.lua` | Builds the grammar; JSP buffers get the indent below |
| `lua/fuentastic/jsp_indent.lua` | Indent for markup and scriptlet lines |
| `lua/fuentastic/jsp.lua` | `gd` (paths and classes resolve in `lua/fuentastic/refs.lua`, shared with HTL) |
| `after/ftplugin/jsp.lua` | `gd`, and JSP comments as the comment string |
| `ftdetect/jsp.lua` | `.jspf` fragments and `.tag` files are JSP; `.jspx` and `.tagx` are XML |

## Requirements

- `tree-sitter` CLI and a C compiler, to build the parsers (the same as for every other language).

nvim-treesitter builds the JSP parser from `tree-sitter-jsp/` on first start, along with the
`html` and `java` parsers it needs. Until it's done, JSP files get Vim's own JSP syntax.

## Highlighting

The markup is highlighted as HTML, JSTL and other custom tags included, and the Java in `<% %>`,
`<%= %>` and `<%! %>` as Java. Scriptlets are parsed together as one Java program, the way they
end up in the servlet, so an `if` one scriptlet opens and another one closes highlights right.
Directives, JSP comments and EL (`${...}`, `#{...}`) get highlights of their own, also inside
attribute values.

## `gd`

It opens what the cursor line references; on a line with several references it takes the one
under the cursor, else the first.

| On the line | `gd` opens |
| --- | --- |
| `<%@ include file="header.jspf" %>`, `<jsp:include page="...">`, `<jsp:forward page="...">`, `<c:import url="...">` | That page |
| `<%@ page import="java.util.List, com.example.Foo" %>` | The class under the cursor; for `com.example.*` the package's folder |
| `<jsp:useBean class="com.example.Bean">` (or `type=`) | `Bean.java` |
| `Foo` or `com.example.Foo` in Java code | `Foo.java`, through the page's imports |
| `<t:layout>`, from `<%@ taglib prefix="t" tagdir="/WEB-INF/tags" %>` | `WEB-INF/tags/layout.tag` |
| `<app:hello>` or `${app:upper(x)}`, from `<%@ taglib prefix="app" uri="..." %>` | The tag's or function's class, else its entry in the `.tld` |
| `<%@ taglib uri="..." %>` | The `.tld` |
| `<cq:include script="header.jsp">`, `<sling:call script="...">` | That script |
| `resourceType="site/components/title"` | The component's `title.html` or `title.jsp` |
| `<sling:adaptTo adaptTo="com.example.Model">` | `Model.java` |
| `<cq:includeClientLib categories="site.base">` | The clientlib with that category ([aem.md](aem.md#clientlibs)) |
| Nothing of the above | Vim's own `gd` |

Paths resolve like the server resolves them. A relative one starts next to the current page; an
absolute one at the webapp's root (the folder holding `WEB-INF`, `src/main/webapp` in a Maven
war). In an AEM repo, scripts are also looked up under `/apps` and `/libs` in every content
package, like HTL's `gd` ([htl.md](htl.md)). Classes are found in any module's `src/main/java`, and
tag libraries in `WEB-INF` and `src/main/resources/META-INF`.

What isn't in the repo, like the JDK's classes, JSTL's tags or AEM's own `/libs`, gives "Not in
this repo". Tags of a library only an included page declares (`global.jsp` in AEM) and values the
page computes (`page="${view}"`) aren't followed. `<C-o>` jumps back.

## Indent

Markup is indented by its HTML nesting and Java by its braces, a level below the `<%` it's in:

```jsp
<c:forEach items="${names}" var="n">
  <%
    if (n.isEmpty()) {
      continue;
    }
  %>
  <% if (n.length() > 3) { %>
    <b>${n}</b>
  <% } %>
</c:forEach>
```

Markup inside a one-line tag's block, like `<b>` above, is indented under that tag. Markup after a
scriptlet of several lines stays at the scriptlet's level, whatever block it leaves open.

`=` leaves alone what it can't indent better than you: JSP comments and directives over several
lines, attributes continued on the next line, and Java statements continued on the next line
(chained calls, long conditions, an `else` on its own line). Inside `<script>` and `<style>` it
indents like in `.html` files.

New JSP files are indented with 4 spaces, like Java (JSP's ftplugin pulls in Java's settings);
vim-sleuth picks up an existing file's indent.

## Commenting

`gc` comments out with JSP comments, `<%-- --%>`: the server drops those, while an HTML comment
still runs the tags and scriptlets inside it. On Java lines inside a scriptlet it uses `//`.

## Formatting

There's no JSP formatter: saving doesn't format, and `<leader>cf` finds nothing to run. `gg=G`
re-indents a page.

## Changing the grammar

After editing `tree-sitter-jsp/grammar.js`, regenerate the parser and test it in that folder,
then rebuild it in nvim:

```sh
tree-sitter generate --no-bindings
tree-sitter test
```

```vim
:TSInstall! jsp
```

Commit `src/` along with `grammar.js`: nvim-treesitter compiles that, without generating anything.
The test cases are in `test/corpus/`.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| No highlighting, or Vim's old JSP syntax | `:checkhealth nvim-treesitter`; `:TSInstall! jsp` |
| `gd` says "Not in this repo" for a class that is | It's looked up as `<module>/src/main/java/...` at most two folders below the git root |
| `gd` doesn't find a module or `.tld` added since nvim started | Restart nvim: the repo's modules are looked up once |
| `=` leaves a line where it was | See [Indent](#indent): that kind of line keeps its indent |

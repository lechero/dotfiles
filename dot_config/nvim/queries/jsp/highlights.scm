; The markup and the Java in JSP tags are highlighted by the html and java parsers (injections.scm).
(comment) @comment @spell

[
  "<%@"
  "<%!"
  "<%="
  "<%"
  "%>"
] @tag.delimiter

(directive_name) @keyword.directive

(attribute_name) @tag.attribute

(attribute
  "=" @operator)

(attribute_value) @string

; EL
(el_expression
  [
    "${"
    "#{"
    "}"
  ] @punctuation.special)

(identifier) @variable

((identifier) @variable.builtin
  (#any-of? @variable.builtin
    "pageContext" "pageScope" "requestScope" "sessionScope" "applicationScope" "param" "paramValues"
    "header" "headerValues" "cookie" "initParam"))

("."
  .
  (identifier) @variable.member)

((identifier) @function.call
  .
  "(")

; fn:escapeXml(...)
((identifier) @module
  .
  ":"
  .
  (identifier) @function.call)

(string) @string

(number) @number

(boolean) @boolean

(null) @constant.builtin

(keyword_operator) @keyword.operator

(operator) @operator

[
  "."
  ","
  ":"
] @punctuation.delimiter

[
  "("
  ")"
  "["
  "]"
] @punctuation.bracket

(el_braces
  [
    "{"
    "}"
  ] @punctuation.bracket)

; gc comments out with JSP comments: the server drops those, while an HTML comment still runs the
; tags inside it. Lines of Java in a scriptlet get //.
((document
  (_) @_jsp)
  (#set! bo.commentstring "<%-- %s --%>"))

([
  (scriptlet
    (code) @_java)
  (declaration
    (code) @_java)
]
  (#set! bo.commentstring "// %s"))

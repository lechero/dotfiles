((content) @injection.content
  (#set! injection.language "html")
  (#set! injection.combined))

; Scriptlets and declarations end up in one servlet class, so they're parsed as one Java program:
; an `if` one scriptlet opens, another one closes.
([
  (scriptlet
    (code) @injection.content)
  (declaration
    (code) @injection.content)
]
  (#set! injection.language "java")
  (#set! injection.combined))

((expression
  (code) @injection.content)
  (#set! injection.language "java"))

; extends

; The text around a Backstage template's markup, in the language of the file it renders
; (ftdetect/backstage.lua), parsed as one document: an object a {% for %} opens, its {% endfor %}
; closes.
((content) @injection.content
  (#set-template-lang!)
  (#set! injection.combined))

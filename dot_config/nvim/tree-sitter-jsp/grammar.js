/**
 * JSP: template text with JSP tags (<%@ %>, <% %>, <%= %>, <%! %>, <%-- --%>) and EL (${...})
 * in it. The Java in the tags and the HTML around them are left to the java and html parsers
 * (queries/jsp/injections.scm in the nvim config).
 *
 * After changing this file: `tree-sitter generate --no-bindings` here, then `:TSInstall! jsp`.
 */

const ws = /\s+/;

module.exports = grammar({
  name: 'jsp',

  // Whitespace is part of the template text, so nothing is skipped between tokens.
  extras: (_) => [],

  rules: {
    document: ($) => repeat($._node),

    _node: ($) => choice($.comment, $.directive, $.declaration, $.expression, $.scriptlet, $.el_expression, $.content),

    // Everything up to the next JSP tag or EL expression. \${ and \#{ are literal text.
    content: (_) => prec.right(repeat1(choice(/[^<$#\\]+/, '<', '$', '#', '\\', '\\$', '\\#'))),

    // Ends at the first --%>, whatever is in between (JSP tags included).
    comment: (_) => seq('<%--', repeat(choice(/[^-]+/, '-')), '--%>'),

    // <%@ page import="java.util.List" %>
    directive: ($) =>
      seq('<%@', optional(ws), field('name', $.directive_name), repeat(seq(ws, $.attribute)), optional(ws), '%>'),
    directive_name: (_) => /[a-zA-Z]+/,
    attribute: ($) => seq(field('name', $.attribute_name), optional(ws), '=', optional(ws), field('value', $.attribute_value)),
    attribute_name: (_) => /[a-zA-Z_][\w.:-]*/,
    attribute_value: (_) => choice(/"[^"]*"/, /'[^']*'/),

    declaration: ($) => seq('<%!', optional($.code), '%>'), // fields and methods of the servlet
    expression: ($) => seq('<%=', optional($.code), '%>'), // printed
    scriptlet: ($) => seq('<%', optional($.code), '%>'), // statements

    // Java, up to the first %>: JSP ends the tag there even inside a string (%\> escapes it).
    code: (_) => repeat1(choice(/[^%]+/, '%')),

    // ${...} is evaluated when the page renders, #{...} later (deferred, JSF).
    el_expression: ($) => seq(choice('${', '#{'), repeat($._el), '}'),

    _el: ($) =>
      choice(
        ws,
        $.identifier,
        $.string,
        $.number,
        $.boolean,
        $.null,
        $.keyword_operator,
        $.operator,
        $.el_braces,
        '.',
        ',',
        ':',
        '(',
        ')',
        '[',
        ']',
      ),
    el_braces: ($) => seq('{', repeat($._el), '}'), // map and set literals, lambda bodies
    identifier: (_) => /[a-zA-Z_$][\w$]*/,
    string: (_) => choice(/"([^"\\]|\\.)*"/, /'([^'\\]|\\.)*'/),
    number: (_) => /\d+(\.\d+)?([eE][+-]?\d+)?|\.\d+([eE][+-]?\d+)?/,
    boolean: (_) => choice('true', 'false'),
    null: (_) => 'null',
    keyword_operator: (_) =>
      choice('empty', 'not', 'and', 'or', 'div', 'mod', 'eq', 'ne', 'lt', 'gt', 'le', 'ge', 'instanceof'),
    operator: (_) =>
      choice('==', '!=', '<=', '>=', '<', '>', '&&', '||', '!', '+', '-', '*', '/', '%', '?', '=', '+=', '->', ';'),
  },
});

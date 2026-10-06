#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 68
#define LARGE_STATE_COUNT 13
#define SYMBOL_COUNT 95
#define ALIAS_COUNT 0
#define TOKEN_COUNT 71
#define EXTERNAL_TOKEN_COUNT 0
#define FIELD_COUNT 2
#define MAX_ALIAS_SEQUENCE_LENGTH 6
#define PRODUCTION_ID_COUNT 6

enum ts_symbol_identifiers {
  aux_sym_content_token1 = 1,
  anon_sym_LT = 2,
  anon_sym_DOLLAR = 3,
  anon_sym_POUND = 4,
  anon_sym_BSLASH = 5,
  anon_sym_BSLASH_DOLLAR = 6,
  anon_sym_BSLASH_POUND = 7,
  anon_sym_LT_PERCENT_DASH_DASH = 8,
  aux_sym_comment_token1 = 9,
  anon_sym_DASH = 10,
  anon_sym_DASH_DASH_PERCENT_GT = 11,
  anon_sym_LT_PERCENT_AT = 12,
  aux_sym_directive_token1 = 13,
  anon_sym_PERCENT_GT = 14,
  sym_directive_name = 15,
  anon_sym_EQ = 16,
  sym_attribute_name = 17,
  aux_sym_attribute_value_token1 = 18,
  aux_sym_attribute_value_token2 = 19,
  anon_sym_LT_PERCENT_BANG = 20,
  anon_sym_LT_PERCENT_EQ = 21,
  anon_sym_LT_PERCENT = 22,
  aux_sym_code_token1 = 23,
  anon_sym_PERCENT = 24,
  anon_sym_DOLLAR_LBRACE = 25,
  anon_sym_POUND_LBRACE = 26,
  anon_sym_RBRACE = 27,
  anon_sym_DOT = 28,
  anon_sym_COMMA = 29,
  anon_sym_COLON = 30,
  anon_sym_LPAREN = 31,
  anon_sym_RPAREN = 32,
  anon_sym_LBRACK = 33,
  anon_sym_RBRACK = 34,
  anon_sym_LBRACE = 35,
  sym_identifier = 36,
  aux_sym_string_token1 = 37,
  aux_sym_string_token2 = 38,
  sym_number = 39,
  anon_sym_true = 40,
  anon_sym_false = 41,
  sym_null = 42,
  anon_sym_empty = 43,
  anon_sym_not = 44,
  anon_sym_and = 45,
  anon_sym_or = 46,
  anon_sym_div = 47,
  anon_sym_mod = 48,
  anon_sym_eq = 49,
  anon_sym_ne = 50,
  anon_sym_lt = 51,
  anon_sym_gt = 52,
  anon_sym_le = 53,
  anon_sym_ge = 54,
  anon_sym_instanceof = 55,
  anon_sym_EQ_EQ = 56,
  anon_sym_BANG_EQ = 57,
  anon_sym_LT_EQ = 58,
  anon_sym_GT_EQ = 59,
  anon_sym_GT = 60,
  anon_sym_AMP_AMP = 61,
  anon_sym_PIPE_PIPE = 62,
  anon_sym_BANG = 63,
  anon_sym_PLUS = 64,
  anon_sym_STAR = 65,
  anon_sym_SLASH = 66,
  anon_sym_QMARK = 67,
  anon_sym_PLUS_EQ = 68,
  anon_sym_DASH_GT = 69,
  anon_sym_SEMI = 70,
  sym_document = 71,
  sym__node = 72,
  sym_content = 73,
  sym_comment = 74,
  sym_directive = 75,
  sym_attribute = 76,
  sym_attribute_value = 77,
  sym_declaration = 78,
  sym_expression = 79,
  sym_scriptlet = 80,
  sym_code = 81,
  sym_el_expression = 82,
  sym__el = 83,
  sym_el_braces = 84,
  sym_string = 85,
  sym_boolean = 86,
  sym_keyword_operator = 87,
  sym_operator = 88,
  aux_sym_document_repeat1 = 89,
  aux_sym_content_repeat1 = 90,
  aux_sym_comment_repeat1 = 91,
  aux_sym_directive_repeat1 = 92,
  aux_sym_code_repeat1 = 93,
  aux_sym_el_expression_repeat1 = 94,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [aux_sym_content_token1] = "content_token1",
  [anon_sym_LT] = "<",
  [anon_sym_DOLLAR] = "$",
  [anon_sym_POUND] = "#",
  [anon_sym_BSLASH] = "\\",
  [anon_sym_BSLASH_DOLLAR] = "\\$",
  [anon_sym_BSLASH_POUND] = "\\#",
  [anon_sym_LT_PERCENT_DASH_DASH] = "<%--",
  [aux_sym_comment_token1] = "comment_token1",
  [anon_sym_DASH] = "-",
  [anon_sym_DASH_DASH_PERCENT_GT] = "--%>",
  [anon_sym_LT_PERCENT_AT] = "<%@",
  [aux_sym_directive_token1] = "directive_token1",
  [anon_sym_PERCENT_GT] = "%>",
  [sym_directive_name] = "directive_name",
  [anon_sym_EQ] = "=",
  [sym_attribute_name] = "attribute_name",
  [aux_sym_attribute_value_token1] = "attribute_value_token1",
  [aux_sym_attribute_value_token2] = "attribute_value_token2",
  [anon_sym_LT_PERCENT_BANG] = "<%!",
  [anon_sym_LT_PERCENT_EQ] = "<%=",
  [anon_sym_LT_PERCENT] = "<%",
  [aux_sym_code_token1] = "code_token1",
  [anon_sym_PERCENT] = "%",
  [anon_sym_DOLLAR_LBRACE] = "${",
  [anon_sym_POUND_LBRACE] = "#{",
  [anon_sym_RBRACE] = "}",
  [anon_sym_DOT] = ".",
  [anon_sym_COMMA] = ",",
  [anon_sym_COLON] = ":",
  [anon_sym_LPAREN] = "(",
  [anon_sym_RPAREN] = ")",
  [anon_sym_LBRACK] = "[",
  [anon_sym_RBRACK] = "]",
  [anon_sym_LBRACE] = "{",
  [sym_identifier] = "identifier",
  [aux_sym_string_token1] = "string_token1",
  [aux_sym_string_token2] = "string_token2",
  [sym_number] = "number",
  [anon_sym_true] = "true",
  [anon_sym_false] = "false",
  [sym_null] = "null",
  [anon_sym_empty] = "empty",
  [anon_sym_not] = "not",
  [anon_sym_and] = "and",
  [anon_sym_or] = "or",
  [anon_sym_div] = "div",
  [anon_sym_mod] = "mod",
  [anon_sym_eq] = "eq",
  [anon_sym_ne] = "ne",
  [anon_sym_lt] = "lt",
  [anon_sym_gt] = "gt",
  [anon_sym_le] = "le",
  [anon_sym_ge] = "ge",
  [anon_sym_instanceof] = "instanceof",
  [anon_sym_EQ_EQ] = "==",
  [anon_sym_BANG_EQ] = "!=",
  [anon_sym_LT_EQ] = "<=",
  [anon_sym_GT_EQ] = ">=",
  [anon_sym_GT] = ">",
  [anon_sym_AMP_AMP] = "&&",
  [anon_sym_PIPE_PIPE] = "||",
  [anon_sym_BANG] = "!",
  [anon_sym_PLUS] = "+",
  [anon_sym_STAR] = "*",
  [anon_sym_SLASH] = "/",
  [anon_sym_QMARK] = "\?",
  [anon_sym_PLUS_EQ] = "+=",
  [anon_sym_DASH_GT] = "->",
  [anon_sym_SEMI] = ";",
  [sym_document] = "document",
  [sym__node] = "_node",
  [sym_content] = "content",
  [sym_comment] = "comment",
  [sym_directive] = "directive",
  [sym_attribute] = "attribute",
  [sym_attribute_value] = "attribute_value",
  [sym_declaration] = "declaration",
  [sym_expression] = "expression",
  [sym_scriptlet] = "scriptlet",
  [sym_code] = "code",
  [sym_el_expression] = "el_expression",
  [sym__el] = "_el",
  [sym_el_braces] = "el_braces",
  [sym_string] = "string",
  [sym_boolean] = "boolean",
  [sym_keyword_operator] = "keyword_operator",
  [sym_operator] = "operator",
  [aux_sym_document_repeat1] = "document_repeat1",
  [aux_sym_content_repeat1] = "content_repeat1",
  [aux_sym_comment_repeat1] = "comment_repeat1",
  [aux_sym_directive_repeat1] = "directive_repeat1",
  [aux_sym_code_repeat1] = "code_repeat1",
  [aux_sym_el_expression_repeat1] = "el_expression_repeat1",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [aux_sym_content_token1] = aux_sym_content_token1,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_DOLLAR] = anon_sym_DOLLAR,
  [anon_sym_POUND] = anon_sym_POUND,
  [anon_sym_BSLASH] = anon_sym_BSLASH,
  [anon_sym_BSLASH_DOLLAR] = anon_sym_BSLASH_DOLLAR,
  [anon_sym_BSLASH_POUND] = anon_sym_BSLASH_POUND,
  [anon_sym_LT_PERCENT_DASH_DASH] = anon_sym_LT_PERCENT_DASH_DASH,
  [aux_sym_comment_token1] = aux_sym_comment_token1,
  [anon_sym_DASH] = anon_sym_DASH,
  [anon_sym_DASH_DASH_PERCENT_GT] = anon_sym_DASH_DASH_PERCENT_GT,
  [anon_sym_LT_PERCENT_AT] = anon_sym_LT_PERCENT_AT,
  [aux_sym_directive_token1] = aux_sym_directive_token1,
  [anon_sym_PERCENT_GT] = anon_sym_PERCENT_GT,
  [sym_directive_name] = sym_directive_name,
  [anon_sym_EQ] = anon_sym_EQ,
  [sym_attribute_name] = sym_attribute_name,
  [aux_sym_attribute_value_token1] = aux_sym_attribute_value_token1,
  [aux_sym_attribute_value_token2] = aux_sym_attribute_value_token2,
  [anon_sym_LT_PERCENT_BANG] = anon_sym_LT_PERCENT_BANG,
  [anon_sym_LT_PERCENT_EQ] = anon_sym_LT_PERCENT_EQ,
  [anon_sym_LT_PERCENT] = anon_sym_LT_PERCENT,
  [aux_sym_code_token1] = aux_sym_code_token1,
  [anon_sym_PERCENT] = anon_sym_PERCENT,
  [anon_sym_DOLLAR_LBRACE] = anon_sym_DOLLAR_LBRACE,
  [anon_sym_POUND_LBRACE] = anon_sym_POUND_LBRACE,
  [anon_sym_RBRACE] = anon_sym_RBRACE,
  [anon_sym_DOT] = anon_sym_DOT,
  [anon_sym_COMMA] = anon_sym_COMMA,
  [anon_sym_COLON] = anon_sym_COLON,
  [anon_sym_LPAREN] = anon_sym_LPAREN,
  [anon_sym_RPAREN] = anon_sym_RPAREN,
  [anon_sym_LBRACK] = anon_sym_LBRACK,
  [anon_sym_RBRACK] = anon_sym_RBRACK,
  [anon_sym_LBRACE] = anon_sym_LBRACE,
  [sym_identifier] = sym_identifier,
  [aux_sym_string_token1] = aux_sym_string_token1,
  [aux_sym_string_token2] = aux_sym_string_token2,
  [sym_number] = sym_number,
  [anon_sym_true] = anon_sym_true,
  [anon_sym_false] = anon_sym_false,
  [sym_null] = sym_null,
  [anon_sym_empty] = anon_sym_empty,
  [anon_sym_not] = anon_sym_not,
  [anon_sym_and] = anon_sym_and,
  [anon_sym_or] = anon_sym_or,
  [anon_sym_div] = anon_sym_div,
  [anon_sym_mod] = anon_sym_mod,
  [anon_sym_eq] = anon_sym_eq,
  [anon_sym_ne] = anon_sym_ne,
  [anon_sym_lt] = anon_sym_lt,
  [anon_sym_gt] = anon_sym_gt,
  [anon_sym_le] = anon_sym_le,
  [anon_sym_ge] = anon_sym_ge,
  [anon_sym_instanceof] = anon_sym_instanceof,
  [anon_sym_EQ_EQ] = anon_sym_EQ_EQ,
  [anon_sym_BANG_EQ] = anon_sym_BANG_EQ,
  [anon_sym_LT_EQ] = anon_sym_LT_EQ,
  [anon_sym_GT_EQ] = anon_sym_GT_EQ,
  [anon_sym_GT] = anon_sym_GT,
  [anon_sym_AMP_AMP] = anon_sym_AMP_AMP,
  [anon_sym_PIPE_PIPE] = anon_sym_PIPE_PIPE,
  [anon_sym_BANG] = anon_sym_BANG,
  [anon_sym_PLUS] = anon_sym_PLUS,
  [anon_sym_STAR] = anon_sym_STAR,
  [anon_sym_SLASH] = anon_sym_SLASH,
  [anon_sym_QMARK] = anon_sym_QMARK,
  [anon_sym_PLUS_EQ] = anon_sym_PLUS_EQ,
  [anon_sym_DASH_GT] = anon_sym_DASH_GT,
  [anon_sym_SEMI] = anon_sym_SEMI,
  [sym_document] = sym_document,
  [sym__node] = sym__node,
  [sym_content] = sym_content,
  [sym_comment] = sym_comment,
  [sym_directive] = sym_directive,
  [sym_attribute] = sym_attribute,
  [sym_attribute_value] = sym_attribute_value,
  [sym_declaration] = sym_declaration,
  [sym_expression] = sym_expression,
  [sym_scriptlet] = sym_scriptlet,
  [sym_code] = sym_code,
  [sym_el_expression] = sym_el_expression,
  [sym__el] = sym__el,
  [sym_el_braces] = sym_el_braces,
  [sym_string] = sym_string,
  [sym_boolean] = sym_boolean,
  [sym_keyword_operator] = sym_keyword_operator,
  [sym_operator] = sym_operator,
  [aux_sym_document_repeat1] = aux_sym_document_repeat1,
  [aux_sym_content_repeat1] = aux_sym_content_repeat1,
  [aux_sym_comment_repeat1] = aux_sym_comment_repeat1,
  [aux_sym_directive_repeat1] = aux_sym_directive_repeat1,
  [aux_sym_code_repeat1] = aux_sym_code_repeat1,
  [aux_sym_el_expression_repeat1] = aux_sym_el_expression_repeat1,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [aux_sym_content_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_LT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DOLLAR] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUND] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BSLASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BSLASH_DOLLAR] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BSLASH_POUND] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_PERCENT_DASH_DASH] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_comment_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_DASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DASH_DASH_PERCENT_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_PERCENT_AT] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_directive_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_PERCENT_GT] = {
    .visible = true,
    .named = false,
  },
  [sym_directive_name] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_EQ] = {
    .visible = true,
    .named = false,
  },
  [sym_attribute_name] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_attribute_value_token1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_attribute_value_token2] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_LT_PERCENT_BANG] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_PERCENT_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_PERCENT] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_code_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_PERCENT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DOLLAR_LBRACE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUND_LBRACE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RBRACE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DOT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_COMMA] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_COLON] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LPAREN] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RPAREN] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LBRACK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RBRACK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LBRACE] = {
    .visible = true,
    .named = false,
  },
  [sym_identifier] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_string_token1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_string_token2] = {
    .visible = false,
    .named = false,
  },
  [sym_number] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_true] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_false] = {
    .visible = true,
    .named = false,
  },
  [sym_null] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_empty] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_not] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_and] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_or] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_div] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_mod] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_eq] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ne] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_lt] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_gt] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_le] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ge] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_instanceof] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_EQ_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BANG_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_AMP_AMP] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PIPE_PIPE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BANG] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PLUS] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_STAR] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SLASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_QMARK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PLUS_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DASH_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SEMI] = {
    .visible = true,
    .named = false,
  },
  [sym_document] = {
    .visible = true,
    .named = true,
  },
  [sym__node] = {
    .visible = false,
    .named = true,
  },
  [sym_content] = {
    .visible = true,
    .named = true,
  },
  [sym_comment] = {
    .visible = true,
    .named = true,
  },
  [sym_directive] = {
    .visible = true,
    .named = true,
  },
  [sym_attribute] = {
    .visible = true,
    .named = true,
  },
  [sym_attribute_value] = {
    .visible = true,
    .named = true,
  },
  [sym_declaration] = {
    .visible = true,
    .named = true,
  },
  [sym_expression] = {
    .visible = true,
    .named = true,
  },
  [sym_scriptlet] = {
    .visible = true,
    .named = true,
  },
  [sym_code] = {
    .visible = true,
    .named = true,
  },
  [sym_el_expression] = {
    .visible = true,
    .named = true,
  },
  [sym__el] = {
    .visible = false,
    .named = true,
  },
  [sym_el_braces] = {
    .visible = true,
    .named = true,
  },
  [sym_string] = {
    .visible = true,
    .named = true,
  },
  [sym_boolean] = {
    .visible = true,
    .named = true,
  },
  [sym_keyword_operator] = {
    .visible = true,
    .named = true,
  },
  [sym_operator] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_document_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_content_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_comment_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_directive_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_code_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_el_expression_repeat1] = {
    .visible = false,
    .named = false,
  },
};

enum ts_field_identifiers {
  field_name = 1,
  field_value = 2,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_name] = "name",
  [field_value] = "value",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
  [2] = {.index = 1, .length = 1},
  [3] = {.index = 2, .length = 2},
  [4] = {.index = 4, .length = 2},
  [5] = {.index = 6, .length = 2},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_name, 1},
  [1] =
    {field_name, 2},
  [2] =
    {field_name, 0},
    {field_value, 2},
  [4] =
    {field_name, 0},
    {field_value, 3},
  [6] =
    {field_name, 0},
    {field_value, 4},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
};

static const uint16_t ts_non_terminal_alias_map[] = {
  0,
};

static const TSStateId ts_primary_state_ids[STATE_COUNT] = {
  [0] = 0,
  [1] = 1,
  [2] = 2,
  [3] = 3,
  [4] = 4,
  [5] = 5,
  [6] = 6,
  [7] = 7,
  [8] = 8,
  [9] = 9,
  [10] = 10,
  [11] = 11,
  [12] = 12,
  [13] = 13,
  [14] = 14,
  [15] = 15,
  [16] = 16,
  [17] = 17,
  [18] = 18,
  [19] = 19,
  [20] = 20,
  [21] = 21,
  [22] = 22,
  [23] = 23,
  [24] = 24,
  [25] = 25,
  [26] = 26,
  [27] = 27,
  [28] = 28,
  [29] = 29,
  [30] = 30,
  [31] = 31,
  [32] = 32,
  [33] = 33,
  [34] = 34,
  [35] = 35,
  [36] = 36,
  [37] = 37,
  [38] = 38,
  [39] = 39,
  [40] = 40,
  [41] = 41,
  [42] = 42,
  [43] = 43,
  [44] = 44,
  [45] = 45,
  [46] = 46,
  [47] = 47,
  [48] = 48,
  [49] = 49,
  [50] = 50,
  [51] = 51,
  [52] = 52,
  [53] = 53,
  [54] = 54,
  [55] = 55,
  [56] = 56,
  [57] = 57,
  [58] = 58,
  [59] = 59,
  [60] = 60,
  [61] = 61,
  [62] = 62,
  [63] = 63,
  [64] = 64,
  [65] = 65,
  [66] = 66,
  [67] = 67,
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(59);
      ADVANCE_MAP(
        '!', 181,
        '"', 4,
        '#', 65,
        '$', 64,
        '%', 91,
        '&', 10,
        '\'', 11,
        '(', 98,
        ')', 99,
        '*', 183,
        '+', 182,
        ',', 96,
        '-', 72,
        '.', 95,
        '/', 184,
        ':', 97,
        ';', 188,
        '<', 62,
        '=', 80,
        '>', 178,
        '?', 185,
        '[', 100,
        '\\', 66,
        ']', 101,
        'a', 35,
        'd', 30,
        'e', 34,
        'f', 18,
        'g', 23,
        'i', 36,
        'l', 24,
        'm', 39,
        'n', 25,
        'o', 41,
        't', 42,
        '{', 102,
        '|', 51,
        '}', 94,
      );
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(76);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(139);
      END_STATE();
    case 1:
      if (lookahead == '\n') ADVANCE(5);
      if (lookahead == '"') ADVANCE(83);
      if (lookahead != 0) ADVANCE(4);
      END_STATE();
    case 2:
      if (lookahead == '\n') ADVANCE(12);
      if (lookahead == '\'') ADVANCE(85);
      if (lookahead != 0) ADVANCE(11);
      END_STATE();
    case 3:
      ADVANCE_MAP(
        '!', 181,
        '"', 7,
        '%', 90,
        '&', 10,
        '\'', 13,
        '(', 98,
        ')', 99,
        '*', 183,
        '+', 182,
        ',', 96,
        '-', 73,
        '.', 95,
        '/', 184,
        ':', 97,
        ';', 188,
        '<', 63,
        '=', 80,
        '>', 178,
        '?', 185,
        '[', 100,
        ']', 101,
        'a', 120,
        'd', 115,
        'e', 119,
        'f', 103,
        'g', 108,
        'i', 121,
        'l', 109,
        'm', 124,
        'n', 110,
        'o', 126,
        't', 127,
        '{', 102,
        '|', 51,
        '}', 94,
      );
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(76);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(139);
      if (lookahead == '$' ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('b' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 4:
      if (lookahead == '"') ADVANCE(82);
      if (lookahead == '\\') ADVANCE(1);
      if (lookahead != 0) ADVANCE(4);
      END_STATE();
    case 5:
      if (lookahead == '"') ADVANCE(82);
      if (lookahead != 0) ADVANCE(5);
      END_STATE();
    case 6:
      if (lookahead == '"') ADVANCE(5);
      if (lookahead == '%') ADVANCE(16);
      if (lookahead == '\'') ADVANCE(12);
      if (lookahead == '=') ADVANCE(79);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(76);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(81);
      END_STATE();
    case 7:
      if (lookahead == '"') ADVANCE(137);
      if (lookahead == '\\') ADVANCE(56);
      if (lookahead != 0) ADVANCE(7);
      END_STATE();
    case 8:
      if (lookahead == '%') ADVANCE(91);
      if (lookahead != 0) ADVANCE(89);
      END_STATE();
    case 9:
      if (lookahead == '%') ADVANCE(17);
      END_STATE();
    case 10:
      if (lookahead == '&') ADVANCE(179);
      END_STATE();
    case 11:
      if (lookahead == '\'') ADVANCE(84);
      if (lookahead == '\\') ADVANCE(2);
      if (lookahead != 0) ADVANCE(11);
      END_STATE();
    case 12:
      if (lookahead == '\'') ADVANCE(84);
      if (lookahead != 0) ADVANCE(12);
      END_STATE();
    case 13:
      if (lookahead == '\'') ADVANCE(138);
      if (lookahead == '\\') ADVANCE(57);
      if (lookahead != 0) ADVANCE(13);
      END_STATE();
    case 14:
      if (lookahead == '-') ADVANCE(69);
      END_STATE();
    case 15:
      if (lookahead == '-') ADVANCE(71);
      if (lookahead != 0) ADVANCE(70);
      END_STATE();
    case 16:
      if (lookahead == '>') ADVANCE(77);
      END_STATE();
    case 17:
      if (lookahead == '>') ADVANCE(74);
      END_STATE();
    case 18:
      if (lookahead == 'a') ADVANCE(32);
      END_STATE();
    case 19:
      if (lookahead == 'a') ADVANCE(37);
      END_STATE();
    case 20:
      if (lookahead == 'c') ADVANCE(28);
      END_STATE();
    case 21:
      if (lookahead == 'd') ADVANCE(152);
      END_STATE();
    case 22:
      if (lookahead == 'd') ADVANCE(158);
      END_STATE();
    case 23:
      if (lookahead == 'e') ADVANCE(170);
      if (lookahead == 't') ADVANCE(166);
      END_STATE();
    case 24:
      if (lookahead == 'e') ADVANCE(168);
      if (lookahead == 't') ADVANCE(164);
      END_STATE();
    case 25:
      if (lookahead == 'e') ADVANCE(162);
      if (lookahead == 'o') ADVANCE(45);
      if (lookahead == 'u') ADVANCE(33);
      END_STATE();
    case 26:
      if (lookahead == 'e') ADVANCE(142);
      END_STATE();
    case 27:
      if (lookahead == 'e') ADVANCE(144);
      END_STATE();
    case 28:
      if (lookahead == 'e') ADVANCE(38);
      END_STATE();
    case 29:
      if (lookahead == 'f') ADVANCE(172);
      END_STATE();
    case 30:
      if (lookahead == 'i') ADVANCE(49);
      END_STATE();
    case 31:
      if (lookahead == 'l') ADVANCE(146);
      END_STATE();
    case 32:
      if (lookahead == 'l') ADVANCE(43);
      END_STATE();
    case 33:
      if (lookahead == 'l') ADVANCE(31);
      END_STATE();
    case 34:
      if (lookahead == 'm') ADVANCE(40);
      if (lookahead == 'q') ADVANCE(160);
      END_STATE();
    case 35:
      if (lookahead == 'n') ADVANCE(21);
      END_STATE();
    case 36:
      if (lookahead == 'n') ADVANCE(44);
      END_STATE();
    case 37:
      if (lookahead == 'n') ADVANCE(20);
      END_STATE();
    case 38:
      if (lookahead == 'o') ADVANCE(29);
      END_STATE();
    case 39:
      if (lookahead == 'o') ADVANCE(22);
      END_STATE();
    case 40:
      if (lookahead == 'p') ADVANCE(46);
      END_STATE();
    case 41:
      if (lookahead == 'r') ADVANCE(154);
      END_STATE();
    case 42:
      if (lookahead == 'r') ADVANCE(48);
      END_STATE();
    case 43:
      if (lookahead == 's') ADVANCE(27);
      END_STATE();
    case 44:
      if (lookahead == 's') ADVANCE(47);
      END_STATE();
    case 45:
      if (lookahead == 't') ADVANCE(150);
      END_STATE();
    case 46:
      if (lookahead == 't') ADVANCE(50);
      END_STATE();
    case 47:
      if (lookahead == 't') ADVANCE(19);
      END_STATE();
    case 48:
      if (lookahead == 'u') ADVANCE(26);
      END_STATE();
    case 49:
      if (lookahead == 'v') ADVANCE(156);
      END_STATE();
    case 50:
      if (lookahead == 'y') ADVANCE(148);
      END_STATE();
    case 51:
      if (lookahead == '|') ADVANCE(180);
      END_STATE();
    case 52:
      if (lookahead == '+' ||
          lookahead == '-') ADVANCE(55);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(141);
      END_STATE();
    case 53:
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(76);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(78);
      END_STATE();
    case 54:
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(140);
      END_STATE();
    case 55:
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(141);
      END_STATE();
    case 56:
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(7);
      END_STATE();
    case 57:
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(13);
      END_STATE();
    case 58:
      if (eof) ADVANCE(59);
      if (lookahead == '#') ADVANCE(65);
      if (lookahead == '$') ADVANCE(64);
      if (lookahead == '<') ADVANCE(61);
      if (lookahead == '\\') ADVANCE(66);
      if (lookahead != 0) ADVANCE(60);
      END_STATE();
    case 59:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 60:
      ACCEPT_TOKEN(aux_sym_content_token1);
      if (lookahead != 0 &&
          lookahead != '#' &&
          lookahead != '$' &&
          lookahead != '<' &&
          lookahead != '\\') ADVANCE(60);
      END_STATE();
    case 61:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '%') ADVANCE(88);
      END_STATE();
    case 62:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '%') ADVANCE(88);
      if (lookahead == '=') ADVANCE(176);
      END_STATE();
    case 63:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '=') ADVANCE(176);
      END_STATE();
    case 64:
      ACCEPT_TOKEN(anon_sym_DOLLAR);
      if (lookahead == '{') ADVANCE(92);
      END_STATE();
    case 65:
      ACCEPT_TOKEN(anon_sym_POUND);
      if (lookahead == '{') ADVANCE(93);
      END_STATE();
    case 66:
      ACCEPT_TOKEN(anon_sym_BSLASH);
      if (lookahead == '#') ADVANCE(68);
      if (lookahead == '$') ADVANCE(67);
      END_STATE();
    case 67:
      ACCEPT_TOKEN(anon_sym_BSLASH_DOLLAR);
      END_STATE();
    case 68:
      ACCEPT_TOKEN(anon_sym_BSLASH_POUND);
      END_STATE();
    case 69:
      ACCEPT_TOKEN(anon_sym_LT_PERCENT_DASH_DASH);
      END_STATE();
    case 70:
      ACCEPT_TOKEN(aux_sym_comment_token1);
      if (lookahead != 0 &&
          lookahead != '-') ADVANCE(70);
      END_STATE();
    case 71:
      ACCEPT_TOKEN(anon_sym_DASH);
      if (lookahead == '-') ADVANCE(9);
      END_STATE();
    case 72:
      ACCEPT_TOKEN(anon_sym_DASH);
      if (lookahead == '-') ADVANCE(9);
      if (lookahead == '>') ADVANCE(187);
      END_STATE();
    case 73:
      ACCEPT_TOKEN(anon_sym_DASH);
      if (lookahead == '>') ADVANCE(187);
      END_STATE();
    case 74:
      ACCEPT_TOKEN(anon_sym_DASH_DASH_PERCENT_GT);
      END_STATE();
    case 75:
      ACCEPT_TOKEN(anon_sym_LT_PERCENT_AT);
      END_STATE();
    case 76:
      ACCEPT_TOKEN(aux_sym_directive_token1);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(76);
      END_STATE();
    case 77:
      ACCEPT_TOKEN(anon_sym_PERCENT_GT);
      END_STATE();
    case 78:
      ACCEPT_TOKEN(sym_directive_name);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(78);
      END_STATE();
    case 79:
      ACCEPT_TOKEN(anon_sym_EQ);
      END_STATE();
    case 80:
      ACCEPT_TOKEN(anon_sym_EQ);
      if (lookahead == '=') ADVANCE(174);
      END_STATE();
    case 81:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(81);
      END_STATE();
    case 82:
      ACCEPT_TOKEN(aux_sym_attribute_value_token1);
      END_STATE();
    case 83:
      ACCEPT_TOKEN(aux_sym_attribute_value_token1);
      if (lookahead == '"') ADVANCE(137);
      if (lookahead == '\\') ADVANCE(56);
      if (lookahead != 0) ADVANCE(7);
      END_STATE();
    case 84:
      ACCEPT_TOKEN(aux_sym_attribute_value_token2);
      END_STATE();
    case 85:
      ACCEPT_TOKEN(aux_sym_attribute_value_token2);
      if (lookahead == '\'') ADVANCE(138);
      if (lookahead == '\\') ADVANCE(57);
      if (lookahead != 0) ADVANCE(13);
      END_STATE();
    case 86:
      ACCEPT_TOKEN(anon_sym_LT_PERCENT_BANG);
      END_STATE();
    case 87:
      ACCEPT_TOKEN(anon_sym_LT_PERCENT_EQ);
      END_STATE();
    case 88:
      ACCEPT_TOKEN(anon_sym_LT_PERCENT);
      if (lookahead == '!') ADVANCE(86);
      if (lookahead == '-') ADVANCE(14);
      if (lookahead == '=') ADVANCE(87);
      if (lookahead == '@') ADVANCE(75);
      END_STATE();
    case 89:
      ACCEPT_TOKEN(aux_sym_code_token1);
      if (lookahead != 0 &&
          lookahead != '%') ADVANCE(89);
      END_STATE();
    case 90:
      ACCEPT_TOKEN(anon_sym_PERCENT);
      END_STATE();
    case 91:
      ACCEPT_TOKEN(anon_sym_PERCENT);
      if (lookahead == '>') ADVANCE(77);
      END_STATE();
    case 92:
      ACCEPT_TOKEN(anon_sym_DOLLAR_LBRACE);
      END_STATE();
    case 93:
      ACCEPT_TOKEN(anon_sym_POUND_LBRACE);
      END_STATE();
    case 94:
      ACCEPT_TOKEN(anon_sym_RBRACE);
      END_STATE();
    case 95:
      ACCEPT_TOKEN(anon_sym_DOT);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(140);
      END_STATE();
    case 96:
      ACCEPT_TOKEN(anon_sym_COMMA);
      END_STATE();
    case 97:
      ACCEPT_TOKEN(anon_sym_COLON);
      END_STATE();
    case 98:
      ACCEPT_TOKEN(anon_sym_LPAREN);
      END_STATE();
    case 99:
      ACCEPT_TOKEN(anon_sym_RPAREN);
      END_STATE();
    case 100:
      ACCEPT_TOKEN(anon_sym_LBRACK);
      END_STATE();
    case 101:
      ACCEPT_TOKEN(anon_sym_RBRACK);
      END_STATE();
    case 102:
      ACCEPT_TOKEN(anon_sym_LBRACE);
      END_STATE();
    case 103:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'a') ADVANCE(117);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('b' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 104:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'a') ADVANCE(122);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('b' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 105:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'c') ADVANCE(113);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 106:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'd') ADVANCE(153);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 107:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'd') ADVANCE(159);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 108:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(171);
      if (lookahead == 't') ADVANCE(167);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 109:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(169);
      if (lookahead == 't') ADVANCE(165);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 110:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(163);
      if (lookahead == 'o') ADVANCE(130);
      if (lookahead == 'u') ADVANCE(118);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 111:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(143);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 112:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(145);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 113:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'e') ADVANCE(123);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 114:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'f') ADVANCE(173);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 115:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'i') ADVANCE(134);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 116:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'l') ADVANCE(147);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 117:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'l') ADVANCE(128);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 118:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'l') ADVANCE(116);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 119:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'm') ADVANCE(125);
      if (lookahead == 'q') ADVANCE(161);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 120:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'n') ADVANCE(106);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 121:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'n') ADVANCE(129);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 122:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'n') ADVANCE(105);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 123:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'o') ADVANCE(114);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 124:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'o') ADVANCE(107);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 125:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'p') ADVANCE(131);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 126:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'r') ADVANCE(155);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 127:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'r') ADVANCE(133);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 128:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 's') ADVANCE(112);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 129:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 's') ADVANCE(132);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 130:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 't') ADVANCE(151);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 131:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 't') ADVANCE(135);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 132:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 't') ADVANCE(104);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 133:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'u') ADVANCE(111);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 134:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'v') ADVANCE(157);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 135:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == 'y') ADVANCE(149);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 136:
      ACCEPT_TOKEN(sym_identifier);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 137:
      ACCEPT_TOKEN(aux_sym_string_token1);
      END_STATE();
    case 138:
      ACCEPT_TOKEN(aux_sym_string_token2);
      END_STATE();
    case 139:
      ACCEPT_TOKEN(sym_number);
      if (lookahead == '.') ADVANCE(54);
      if (lookahead == 'E' ||
          lookahead == 'e') ADVANCE(52);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(139);
      END_STATE();
    case 140:
      ACCEPT_TOKEN(sym_number);
      if (lookahead == 'E' ||
          lookahead == 'e') ADVANCE(52);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(140);
      END_STATE();
    case 141:
      ACCEPT_TOKEN(sym_number);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(141);
      END_STATE();
    case 142:
      ACCEPT_TOKEN(anon_sym_true);
      END_STATE();
    case 143:
      ACCEPT_TOKEN(anon_sym_true);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 144:
      ACCEPT_TOKEN(anon_sym_false);
      END_STATE();
    case 145:
      ACCEPT_TOKEN(anon_sym_false);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 146:
      ACCEPT_TOKEN(sym_null);
      END_STATE();
    case 147:
      ACCEPT_TOKEN(sym_null);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 148:
      ACCEPT_TOKEN(anon_sym_empty);
      END_STATE();
    case 149:
      ACCEPT_TOKEN(anon_sym_empty);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 150:
      ACCEPT_TOKEN(anon_sym_not);
      END_STATE();
    case 151:
      ACCEPT_TOKEN(anon_sym_not);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 152:
      ACCEPT_TOKEN(anon_sym_and);
      END_STATE();
    case 153:
      ACCEPT_TOKEN(anon_sym_and);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 154:
      ACCEPT_TOKEN(anon_sym_or);
      END_STATE();
    case 155:
      ACCEPT_TOKEN(anon_sym_or);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 156:
      ACCEPT_TOKEN(anon_sym_div);
      END_STATE();
    case 157:
      ACCEPT_TOKEN(anon_sym_div);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 158:
      ACCEPT_TOKEN(anon_sym_mod);
      END_STATE();
    case 159:
      ACCEPT_TOKEN(anon_sym_mod);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 160:
      ACCEPT_TOKEN(anon_sym_eq);
      END_STATE();
    case 161:
      ACCEPT_TOKEN(anon_sym_eq);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 162:
      ACCEPT_TOKEN(anon_sym_ne);
      END_STATE();
    case 163:
      ACCEPT_TOKEN(anon_sym_ne);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 164:
      ACCEPT_TOKEN(anon_sym_lt);
      END_STATE();
    case 165:
      ACCEPT_TOKEN(anon_sym_lt);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 166:
      ACCEPT_TOKEN(anon_sym_gt);
      END_STATE();
    case 167:
      ACCEPT_TOKEN(anon_sym_gt);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 168:
      ACCEPT_TOKEN(anon_sym_le);
      END_STATE();
    case 169:
      ACCEPT_TOKEN(anon_sym_le);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 170:
      ACCEPT_TOKEN(anon_sym_ge);
      END_STATE();
    case 171:
      ACCEPT_TOKEN(anon_sym_ge);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 172:
      ACCEPT_TOKEN(anon_sym_instanceof);
      END_STATE();
    case 173:
      ACCEPT_TOKEN(anon_sym_instanceof);
      if (lookahead == '$' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(136);
      END_STATE();
    case 174:
      ACCEPT_TOKEN(anon_sym_EQ_EQ);
      END_STATE();
    case 175:
      ACCEPT_TOKEN(anon_sym_BANG_EQ);
      END_STATE();
    case 176:
      ACCEPT_TOKEN(anon_sym_LT_EQ);
      END_STATE();
    case 177:
      ACCEPT_TOKEN(anon_sym_GT_EQ);
      END_STATE();
    case 178:
      ACCEPT_TOKEN(anon_sym_GT);
      if (lookahead == '=') ADVANCE(177);
      END_STATE();
    case 179:
      ACCEPT_TOKEN(anon_sym_AMP_AMP);
      END_STATE();
    case 180:
      ACCEPT_TOKEN(anon_sym_PIPE_PIPE);
      END_STATE();
    case 181:
      ACCEPT_TOKEN(anon_sym_BANG);
      if (lookahead == '=') ADVANCE(175);
      END_STATE();
    case 182:
      ACCEPT_TOKEN(anon_sym_PLUS);
      if (lookahead == '=') ADVANCE(186);
      END_STATE();
    case 183:
      ACCEPT_TOKEN(anon_sym_STAR);
      END_STATE();
    case 184:
      ACCEPT_TOKEN(anon_sym_SLASH);
      END_STATE();
    case 185:
      ACCEPT_TOKEN(anon_sym_QMARK);
      END_STATE();
    case 186:
      ACCEPT_TOKEN(anon_sym_PLUS_EQ);
      END_STATE();
    case 187:
      ACCEPT_TOKEN(anon_sym_DASH_GT);
      END_STATE();
    case 188:
      ACCEPT_TOKEN(anon_sym_SEMI);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0},
  [1] = {.lex_state = 58},
  [2] = {.lex_state = 3},
  [3] = {.lex_state = 3},
  [4] = {.lex_state = 3},
  [5] = {.lex_state = 3},
  [6] = {.lex_state = 3},
  [7] = {.lex_state = 3},
  [8] = {.lex_state = 3},
  [9] = {.lex_state = 3},
  [10] = {.lex_state = 3},
  [11] = {.lex_state = 3},
  [12] = {.lex_state = 3},
  [13] = {.lex_state = 58},
  [14] = {.lex_state = 58},
  [15] = {.lex_state = 58},
  [16] = {.lex_state = 58},
  [17] = {.lex_state = 58},
  [18] = {.lex_state = 58},
  [19] = {.lex_state = 58},
  [20] = {.lex_state = 58},
  [21] = {.lex_state = 58},
  [22] = {.lex_state = 58},
  [23] = {.lex_state = 58},
  [24] = {.lex_state = 58},
  [25] = {.lex_state = 58},
  [26] = {.lex_state = 58},
  [27] = {.lex_state = 58},
  [28] = {.lex_state = 58},
  [29] = {.lex_state = 58},
  [30] = {.lex_state = 58},
  [31] = {.lex_state = 58},
  [32] = {.lex_state = 58},
  [33] = {.lex_state = 8},
  [34] = {.lex_state = 8},
  [35] = {.lex_state = 8},
  [36] = {.lex_state = 6},
  [37] = {.lex_state = 15},
  [38] = {.lex_state = 8},
  [39] = {.lex_state = 15},
  [40] = {.lex_state = 8},
  [41] = {.lex_state = 15},
  [42] = {.lex_state = 6},
  [43] = {.lex_state = 0},
  [44] = {.lex_state = 6},
  [45] = {.lex_state = 6},
  [46] = {.lex_state = 6},
  [47] = {.lex_state = 0},
  [48] = {.lex_state = 6},
  [49] = {.lex_state = 6},
  [50] = {.lex_state = 0},
  [51] = {.lex_state = 6},
  [52] = {.lex_state = 0},
  [53] = {.lex_state = 0},
  [54] = {.lex_state = 0},
  [55] = {.lex_state = 53},
  [56] = {.lex_state = 6},
  [57] = {.lex_state = 0},
  [58] = {.lex_state = 0},
  [59] = {.lex_state = 6},
  [60] = {.lex_state = 0},
  [61] = {.lex_state = 0},
  [62] = {.lex_state = 0},
  [63] = {.lex_state = 6},
  [64] = {.lex_state = 0},
  [65] = {.lex_state = 53},
  [66] = {.lex_state = 0},
  [67] = {.lex_state = 0},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_DOLLAR] = ACTIONS(1),
    [anon_sym_POUND] = ACTIONS(1),
    [anon_sym_BSLASH] = ACTIONS(1),
    [anon_sym_BSLASH_DOLLAR] = ACTIONS(1),
    [anon_sym_BSLASH_POUND] = ACTIONS(1),
    [anon_sym_LT_PERCENT_DASH_DASH] = ACTIONS(1),
    [anon_sym_DASH] = ACTIONS(1),
    [anon_sym_DASH_DASH_PERCENT_GT] = ACTIONS(1),
    [anon_sym_LT_PERCENT_AT] = ACTIONS(1),
    [aux_sym_directive_token1] = ACTIONS(1),
    [anon_sym_PERCENT_GT] = ACTIONS(1),
    [anon_sym_EQ] = ACTIONS(1),
    [aux_sym_attribute_value_token1] = ACTIONS(1),
    [aux_sym_attribute_value_token2] = ACTIONS(1),
    [anon_sym_LT_PERCENT_BANG] = ACTIONS(1),
    [anon_sym_LT_PERCENT_EQ] = ACTIONS(1),
    [anon_sym_LT_PERCENT] = ACTIONS(1),
    [anon_sym_PERCENT] = ACTIONS(1),
    [anon_sym_DOLLAR_LBRACE] = ACTIONS(1),
    [anon_sym_POUND_LBRACE] = ACTIONS(1),
    [anon_sym_RBRACE] = ACTIONS(1),
    [anon_sym_DOT] = ACTIONS(1),
    [anon_sym_COMMA] = ACTIONS(1),
    [anon_sym_COLON] = ACTIONS(1),
    [anon_sym_LPAREN] = ACTIONS(1),
    [anon_sym_RPAREN] = ACTIONS(1),
    [anon_sym_LBRACK] = ACTIONS(1),
    [anon_sym_RBRACK] = ACTIONS(1),
    [anon_sym_LBRACE] = ACTIONS(1),
    [aux_sym_string_token1] = ACTIONS(1),
    [aux_sym_string_token2] = ACTIONS(1),
    [sym_number] = ACTIONS(1),
    [anon_sym_true] = ACTIONS(1),
    [anon_sym_false] = ACTIONS(1),
    [sym_null] = ACTIONS(1),
    [anon_sym_empty] = ACTIONS(1),
    [anon_sym_not] = ACTIONS(1),
    [anon_sym_and] = ACTIONS(1),
    [anon_sym_or] = ACTIONS(1),
    [anon_sym_div] = ACTIONS(1),
    [anon_sym_mod] = ACTIONS(1),
    [anon_sym_eq] = ACTIONS(1),
    [anon_sym_ne] = ACTIONS(1),
    [anon_sym_lt] = ACTIONS(1),
    [anon_sym_gt] = ACTIONS(1),
    [anon_sym_le] = ACTIONS(1),
    [anon_sym_ge] = ACTIONS(1),
    [anon_sym_instanceof] = ACTIONS(1),
    [anon_sym_EQ_EQ] = ACTIONS(1),
    [anon_sym_BANG_EQ] = ACTIONS(1),
    [anon_sym_LT_EQ] = ACTIONS(1),
    [anon_sym_GT_EQ] = ACTIONS(1),
    [anon_sym_GT] = ACTIONS(1),
    [anon_sym_AMP_AMP] = ACTIONS(1),
    [anon_sym_PIPE_PIPE] = ACTIONS(1),
    [anon_sym_BANG] = ACTIONS(1),
    [anon_sym_PLUS] = ACTIONS(1),
    [anon_sym_STAR] = ACTIONS(1),
    [anon_sym_SLASH] = ACTIONS(1),
    [anon_sym_QMARK] = ACTIONS(1),
    [anon_sym_PLUS_EQ] = ACTIONS(1),
    [anon_sym_DASH_GT] = ACTIONS(1),
    [anon_sym_SEMI] = ACTIONS(1),
  },
  [1] = {
    [sym_document] = STATE(66),
    [sym__node] = STATE(14),
    [sym_content] = STATE(14),
    [sym_comment] = STATE(14),
    [sym_directive] = STATE(14),
    [sym_declaration] = STATE(14),
    [sym_expression] = STATE(14),
    [sym_scriptlet] = STATE(14),
    [sym_el_expression] = STATE(14),
    [aux_sym_document_repeat1] = STATE(14),
    [aux_sym_content_repeat1] = STATE(16),
    [ts_builtin_sym_end] = ACTIONS(3),
    [aux_sym_content_token1] = ACTIONS(5),
    [anon_sym_LT] = ACTIONS(7),
    [anon_sym_DOLLAR] = ACTIONS(7),
    [anon_sym_POUND] = ACTIONS(7),
    [anon_sym_BSLASH] = ACTIONS(7),
    [anon_sym_BSLASH_DOLLAR] = ACTIONS(5),
    [anon_sym_BSLASH_POUND] = ACTIONS(5),
    [anon_sym_LT_PERCENT_DASH_DASH] = ACTIONS(9),
    [anon_sym_LT_PERCENT_AT] = ACTIONS(11),
    [anon_sym_LT_PERCENT_BANG] = ACTIONS(13),
    [anon_sym_LT_PERCENT_EQ] = ACTIONS(15),
    [anon_sym_LT_PERCENT] = ACTIONS(17),
    [anon_sym_DOLLAR_LBRACE] = ACTIONS(19),
    [anon_sym_POUND_LBRACE] = ACTIONS(19),
  },
  [2] = {
    [sym__el] = STATE(2),
    [sym_el_braces] = STATE(2),
    [sym_string] = STATE(2),
    [sym_boolean] = STATE(2),
    [sym_keyword_operator] = STATE(2),
    [sym_operator] = STATE(2),
    [aux_sym_el_expression_repeat1] = STATE(2),
    [anon_sym_LT] = ACTIONS(21),
    [anon_sym_DASH] = ACTIONS(21),
    [aux_sym_directive_token1] = ACTIONS(24),
    [anon_sym_EQ] = ACTIONS(21),
    [anon_sym_PERCENT] = ACTIONS(27),
    [anon_sym_RBRACE] = ACTIONS(30),
    [anon_sym_DOT] = ACTIONS(32),
    [anon_sym_COMMA] = ACTIONS(24),
    [anon_sym_COLON] = ACTIONS(24),
    [anon_sym_LPAREN] = ACTIONS(24),
    [anon_sym_RPAREN] = ACTIONS(24),
    [anon_sym_LBRACK] = ACTIONS(24),
    [anon_sym_RBRACK] = ACTIONS(24),
    [anon_sym_LBRACE] = ACTIONS(35),
    [sym_identifier] = ACTIONS(32),
    [aux_sym_string_token1] = ACTIONS(38),
    [aux_sym_string_token2] = ACTIONS(38),
    [sym_number] = ACTIONS(24),
    [anon_sym_true] = ACTIONS(41),
    [anon_sym_false] = ACTIONS(41),
    [sym_null] = ACTIONS(32),
    [anon_sym_empty] = ACTIONS(44),
    [anon_sym_not] = ACTIONS(44),
    [anon_sym_and] = ACTIONS(44),
    [anon_sym_or] = ACTIONS(44),
    [anon_sym_div] = ACTIONS(44),
    [anon_sym_mod] = ACTIONS(44),
    [anon_sym_eq] = ACTIONS(44),
    [anon_sym_ne] = ACTIONS(44),
    [anon_sym_lt] = ACTIONS(44),
    [anon_sym_gt] = ACTIONS(44),
    [anon_sym_le] = ACTIONS(44),
    [anon_sym_ge] = ACTIONS(44),
    [anon_sym_instanceof] = ACTIONS(44),
    [anon_sym_EQ_EQ] = ACTIONS(27),
    [anon_sym_BANG_EQ] = ACTIONS(27),
    [anon_sym_LT_EQ] = ACTIONS(27),
    [anon_sym_GT_EQ] = ACTIONS(27),
    [anon_sym_GT] = ACTIONS(21),
    [anon_sym_AMP_AMP] = ACTIONS(27),
    [anon_sym_PIPE_PIPE] = ACTIONS(27),
    [anon_sym_BANG] = ACTIONS(21),
    [anon_sym_PLUS] = ACTIONS(21),
    [anon_sym_STAR] = ACTIONS(27),
    [anon_sym_SLASH] = ACTIONS(27),
    [anon_sym_QMARK] = ACTIONS(27),
    [anon_sym_PLUS_EQ] = ACTIONS(27),
    [anon_sym_DASH_GT] = ACTIONS(27),
    [anon_sym_SEMI] = ACTIONS(27),
  },
  [3] = {
    [sym__el] = STATE(2),
    [sym_el_braces] = STATE(2),
    [sym_string] = STATE(2),
    [sym_boolean] = STATE(2),
    [sym_keyword_operator] = STATE(2),
    [sym_operator] = STATE(2),
    [aux_sym_el_expression_repeat1] = STATE(2),
    [anon_sym_LT] = ACTIONS(47),
    [anon_sym_DASH] = ACTIONS(47),
    [aux_sym_directive_token1] = ACTIONS(49),
    [anon_sym_EQ] = ACTIONS(47),
    [anon_sym_PERCENT] = ACTIONS(51),
    [anon_sym_RBRACE] = ACTIONS(53),
    [anon_sym_DOT] = ACTIONS(55),
    [anon_sym_COMMA] = ACTIONS(49),
    [anon_sym_COLON] = ACTIONS(49),
    [anon_sym_LPAREN] = ACTIONS(49),
    [anon_sym_RPAREN] = ACTIONS(49),
    [anon_sym_LBRACK] = ACTIONS(49),
    [anon_sym_RBRACK] = ACTIONS(49),
    [anon_sym_LBRACE] = ACTIONS(57),
    [sym_identifier] = ACTIONS(55),
    [aux_sym_string_token1] = ACTIONS(59),
    [aux_sym_string_token2] = ACTIONS(59),
    [sym_number] = ACTIONS(49),
    [anon_sym_true] = ACTIONS(61),
    [anon_sym_false] = ACTIONS(61),
    [sym_null] = ACTIONS(55),
    [anon_sym_empty] = ACTIONS(63),
    [anon_sym_not] = ACTIONS(63),
    [anon_sym_and] = ACTIONS(63),
    [anon_sym_or] = ACTIONS(63),
    [anon_sym_div] = ACTIONS(63),
    [anon_sym_mod] = ACTIONS(63),
    [anon_sym_eq] = ACTIONS(63),
    [anon_sym_ne] = ACTIONS(63),
    [anon_sym_lt] = ACTIONS(63),
    [anon_sym_gt] = ACTIONS(63),
    [anon_sym_le] = ACTIONS(63),
    [anon_sym_ge] = ACTIONS(63),
    [anon_sym_instanceof] = ACTIONS(63),
    [anon_sym_EQ_EQ] = ACTIONS(51),
    [anon_sym_BANG_EQ] = ACTIONS(51),
    [anon_sym_LT_EQ] = ACTIONS(51),
    [anon_sym_GT_EQ] = ACTIONS(51),
    [anon_sym_GT] = ACTIONS(47),
    [anon_sym_AMP_AMP] = ACTIONS(51),
    [anon_sym_PIPE_PIPE] = ACTIONS(51),
    [anon_sym_BANG] = ACTIONS(47),
    [anon_sym_PLUS] = ACTIONS(47),
    [anon_sym_STAR] = ACTIONS(51),
    [anon_sym_SLASH] = ACTIONS(51),
    [anon_sym_QMARK] = ACTIONS(51),
    [anon_sym_PLUS_EQ] = ACTIONS(51),
    [anon_sym_DASH_GT] = ACTIONS(51),
    [anon_sym_SEMI] = ACTIONS(51),
  },
  [4] = {
    [sym__el] = STATE(5),
    [sym_el_braces] = STATE(5),
    [sym_string] = STATE(5),
    [sym_boolean] = STATE(5),
    [sym_keyword_operator] = STATE(5),
    [sym_operator] = STATE(5),
    [aux_sym_el_expression_repeat1] = STATE(5),
    [anon_sym_LT] = ACTIONS(47),
    [anon_sym_DASH] = ACTIONS(47),
    [aux_sym_directive_token1] = ACTIONS(65),
    [anon_sym_EQ] = ACTIONS(47),
    [anon_sym_PERCENT] = ACTIONS(51),
    [anon_sym_RBRACE] = ACTIONS(67),
    [anon_sym_DOT] = ACTIONS(69),
    [anon_sym_COMMA] = ACTIONS(65),
    [anon_sym_COLON] = ACTIONS(65),
    [anon_sym_LPAREN] = ACTIONS(65),
    [anon_sym_RPAREN] = ACTIONS(65),
    [anon_sym_LBRACK] = ACTIONS(65),
    [anon_sym_RBRACK] = ACTIONS(65),
    [anon_sym_LBRACE] = ACTIONS(57),
    [sym_identifier] = ACTIONS(69),
    [aux_sym_string_token1] = ACTIONS(59),
    [aux_sym_string_token2] = ACTIONS(59),
    [sym_number] = ACTIONS(65),
    [anon_sym_true] = ACTIONS(61),
    [anon_sym_false] = ACTIONS(61),
    [sym_null] = ACTIONS(69),
    [anon_sym_empty] = ACTIONS(63),
    [anon_sym_not] = ACTIONS(63),
    [anon_sym_and] = ACTIONS(63),
    [anon_sym_or] = ACTIONS(63),
    [anon_sym_div] = ACTIONS(63),
    [anon_sym_mod] = ACTIONS(63),
    [anon_sym_eq] = ACTIONS(63),
    [anon_sym_ne] = ACTIONS(63),
    [anon_sym_lt] = ACTIONS(63),
    [anon_sym_gt] = ACTIONS(63),
    [anon_sym_le] = ACTIONS(63),
    [anon_sym_ge] = ACTIONS(63),
    [anon_sym_instanceof] = ACTIONS(63),
    [anon_sym_EQ_EQ] = ACTIONS(51),
    [anon_sym_BANG_EQ] = ACTIONS(51),
    [anon_sym_LT_EQ] = ACTIONS(51),
    [anon_sym_GT_EQ] = ACTIONS(51),
    [anon_sym_GT] = ACTIONS(47),
    [anon_sym_AMP_AMP] = ACTIONS(51),
    [anon_sym_PIPE_PIPE] = ACTIONS(51),
    [anon_sym_BANG] = ACTIONS(47),
    [anon_sym_PLUS] = ACTIONS(47),
    [anon_sym_STAR] = ACTIONS(51),
    [anon_sym_SLASH] = ACTIONS(51),
    [anon_sym_QMARK] = ACTIONS(51),
    [anon_sym_PLUS_EQ] = ACTIONS(51),
    [anon_sym_DASH_GT] = ACTIONS(51),
    [anon_sym_SEMI] = ACTIONS(51),
  },
  [5] = {
    [sym__el] = STATE(2),
    [sym_el_braces] = STATE(2),
    [sym_string] = STATE(2),
    [sym_boolean] = STATE(2),
    [sym_keyword_operator] = STATE(2),
    [sym_operator] = STATE(2),
    [aux_sym_el_expression_repeat1] = STATE(2),
    [anon_sym_LT] = ACTIONS(47),
    [anon_sym_DASH] = ACTIONS(47),
    [aux_sym_directive_token1] = ACTIONS(49),
    [anon_sym_EQ] = ACTIONS(47),
    [anon_sym_PERCENT] = ACTIONS(51),
    [anon_sym_RBRACE] = ACTIONS(71),
    [anon_sym_DOT] = ACTIONS(55),
    [anon_sym_COMMA] = ACTIONS(49),
    [anon_sym_COLON] = ACTIONS(49),
    [anon_sym_LPAREN] = ACTIONS(49),
    [anon_sym_RPAREN] = ACTIONS(49),
    [anon_sym_LBRACK] = ACTIONS(49),
    [anon_sym_RBRACK] = ACTIONS(49),
    [anon_sym_LBRACE] = ACTIONS(57),
    [sym_identifier] = ACTIONS(55),
    [aux_sym_string_token1] = ACTIONS(59),
    [aux_sym_string_token2] = ACTIONS(59),
    [sym_number] = ACTIONS(49),
    [anon_sym_true] = ACTIONS(61),
    [anon_sym_false] = ACTIONS(61),
    [sym_null] = ACTIONS(55),
    [anon_sym_empty] = ACTIONS(63),
    [anon_sym_not] = ACTIONS(63),
    [anon_sym_and] = ACTIONS(63),
    [anon_sym_or] = ACTIONS(63),
    [anon_sym_div] = ACTIONS(63),
    [anon_sym_mod] = ACTIONS(63),
    [anon_sym_eq] = ACTIONS(63),
    [anon_sym_ne] = ACTIONS(63),
    [anon_sym_lt] = ACTIONS(63),
    [anon_sym_gt] = ACTIONS(63),
    [anon_sym_le] = ACTIONS(63),
    [anon_sym_ge] = ACTIONS(63),
    [anon_sym_instanceof] = ACTIONS(63),
    [anon_sym_EQ_EQ] = ACTIONS(51),
    [anon_sym_BANG_EQ] = ACTIONS(51),
    [anon_sym_LT_EQ] = ACTIONS(51),
    [anon_sym_GT_EQ] = ACTIONS(51),
    [anon_sym_GT] = ACTIONS(47),
    [anon_sym_AMP_AMP] = ACTIONS(51),
    [anon_sym_PIPE_PIPE] = ACTIONS(51),
    [anon_sym_BANG] = ACTIONS(47),
    [anon_sym_PLUS] = ACTIONS(47),
    [anon_sym_STAR] = ACTIONS(51),
    [anon_sym_SLASH] = ACTIONS(51),
    [anon_sym_QMARK] = ACTIONS(51),
    [anon_sym_PLUS_EQ] = ACTIONS(51),
    [anon_sym_DASH_GT] = ACTIONS(51),
    [anon_sym_SEMI] = ACTIONS(51),
  },
  [6] = {
    [sym__el] = STATE(3),
    [sym_el_braces] = STATE(3),
    [sym_string] = STATE(3),
    [sym_boolean] = STATE(3),
    [sym_keyword_operator] = STATE(3),
    [sym_operator] = STATE(3),
    [aux_sym_el_expression_repeat1] = STATE(3),
    [anon_sym_LT] = ACTIONS(47),
    [anon_sym_DASH] = ACTIONS(47),
    [aux_sym_directive_token1] = ACTIONS(73),
    [anon_sym_EQ] = ACTIONS(47),
    [anon_sym_PERCENT] = ACTIONS(51),
    [anon_sym_RBRACE] = ACTIONS(75),
    [anon_sym_DOT] = ACTIONS(77),
    [anon_sym_COMMA] = ACTIONS(73),
    [anon_sym_COLON] = ACTIONS(73),
    [anon_sym_LPAREN] = ACTIONS(73),
    [anon_sym_RPAREN] = ACTIONS(73),
    [anon_sym_LBRACK] = ACTIONS(73),
    [anon_sym_RBRACK] = ACTIONS(73),
    [anon_sym_LBRACE] = ACTIONS(57),
    [sym_identifier] = ACTIONS(77),
    [aux_sym_string_token1] = ACTIONS(59),
    [aux_sym_string_token2] = ACTIONS(59),
    [sym_number] = ACTIONS(73),
    [anon_sym_true] = ACTIONS(61),
    [anon_sym_false] = ACTIONS(61),
    [sym_null] = ACTIONS(77),
    [anon_sym_empty] = ACTIONS(63),
    [anon_sym_not] = ACTIONS(63),
    [anon_sym_and] = ACTIONS(63),
    [anon_sym_or] = ACTIONS(63),
    [anon_sym_div] = ACTIONS(63),
    [anon_sym_mod] = ACTIONS(63),
    [anon_sym_eq] = ACTIONS(63),
    [anon_sym_ne] = ACTIONS(63),
    [anon_sym_lt] = ACTIONS(63),
    [anon_sym_gt] = ACTIONS(63),
    [anon_sym_le] = ACTIONS(63),
    [anon_sym_ge] = ACTIONS(63),
    [anon_sym_instanceof] = ACTIONS(63),
    [anon_sym_EQ_EQ] = ACTIONS(51),
    [anon_sym_BANG_EQ] = ACTIONS(51),
    [anon_sym_LT_EQ] = ACTIONS(51),
    [anon_sym_GT_EQ] = ACTIONS(51),
    [anon_sym_GT] = ACTIONS(47),
    [anon_sym_AMP_AMP] = ACTIONS(51),
    [anon_sym_PIPE_PIPE] = ACTIONS(51),
    [anon_sym_BANG] = ACTIONS(47),
    [anon_sym_PLUS] = ACTIONS(47),
    [anon_sym_STAR] = ACTIONS(51),
    [anon_sym_SLASH] = ACTIONS(51),
    [anon_sym_QMARK] = ACTIONS(51),
    [anon_sym_PLUS_EQ] = ACTIONS(51),
    [anon_sym_DASH_GT] = ACTIONS(51),
    [anon_sym_SEMI] = ACTIONS(51),
  },
  [7] = {
    [anon_sym_LT] = ACTIONS(79),
    [anon_sym_DASH] = ACTIONS(79),
    [aux_sym_directive_token1] = ACTIONS(81),
    [anon_sym_EQ] = ACTIONS(79),
    [anon_sym_PERCENT] = ACTIONS(81),
    [anon_sym_RBRACE] = ACTIONS(81),
    [anon_sym_DOT] = ACTIONS(79),
    [anon_sym_COMMA] = ACTIONS(81),
    [anon_sym_COLON] = ACTIONS(81),
    [anon_sym_LPAREN] = ACTIONS(81),
    [anon_sym_RPAREN] = ACTIONS(81),
    [anon_sym_LBRACK] = ACTIONS(81),
    [anon_sym_RBRACK] = ACTIONS(81),
    [anon_sym_LBRACE] = ACTIONS(81),
    [sym_identifier] = ACTIONS(79),
    [aux_sym_string_token1] = ACTIONS(81),
    [aux_sym_string_token2] = ACTIONS(81),
    [sym_number] = ACTIONS(81),
    [anon_sym_true] = ACTIONS(79),
    [anon_sym_false] = ACTIONS(79),
    [sym_null] = ACTIONS(79),
    [anon_sym_empty] = ACTIONS(79),
    [anon_sym_not] = ACTIONS(79),
    [anon_sym_and] = ACTIONS(79),
    [anon_sym_or] = ACTIONS(79),
    [anon_sym_div] = ACTIONS(79),
    [anon_sym_mod] = ACTIONS(79),
    [anon_sym_eq] = ACTIONS(79),
    [anon_sym_ne] = ACTIONS(79),
    [anon_sym_lt] = ACTIONS(79),
    [anon_sym_gt] = ACTIONS(79),
    [anon_sym_le] = ACTIONS(79),
    [anon_sym_ge] = ACTIONS(79),
    [anon_sym_instanceof] = ACTIONS(79),
    [anon_sym_EQ_EQ] = ACTIONS(81),
    [anon_sym_BANG_EQ] = ACTIONS(81),
    [anon_sym_LT_EQ] = ACTIONS(81),
    [anon_sym_GT_EQ] = ACTIONS(81),
    [anon_sym_GT] = ACTIONS(79),
    [anon_sym_AMP_AMP] = ACTIONS(81),
    [anon_sym_PIPE_PIPE] = ACTIONS(81),
    [anon_sym_BANG] = ACTIONS(79),
    [anon_sym_PLUS] = ACTIONS(79),
    [anon_sym_STAR] = ACTIONS(81),
    [anon_sym_SLASH] = ACTIONS(81),
    [anon_sym_QMARK] = ACTIONS(81),
    [anon_sym_PLUS_EQ] = ACTIONS(81),
    [anon_sym_DASH_GT] = ACTIONS(81),
    [anon_sym_SEMI] = ACTIONS(81),
  },
  [8] = {
    [anon_sym_LT] = ACTIONS(83),
    [anon_sym_DASH] = ACTIONS(83),
    [aux_sym_directive_token1] = ACTIONS(85),
    [anon_sym_EQ] = ACTIONS(83),
    [anon_sym_PERCENT] = ACTIONS(85),
    [anon_sym_RBRACE] = ACTIONS(85),
    [anon_sym_DOT] = ACTIONS(83),
    [anon_sym_COMMA] = ACTIONS(85),
    [anon_sym_COLON] = ACTIONS(85),
    [anon_sym_LPAREN] = ACTIONS(85),
    [anon_sym_RPAREN] = ACTIONS(85),
    [anon_sym_LBRACK] = ACTIONS(85),
    [anon_sym_RBRACK] = ACTIONS(85),
    [anon_sym_LBRACE] = ACTIONS(85),
    [sym_identifier] = ACTIONS(83),
    [aux_sym_string_token1] = ACTIONS(85),
    [aux_sym_string_token2] = ACTIONS(85),
    [sym_number] = ACTIONS(85),
    [anon_sym_true] = ACTIONS(83),
    [anon_sym_false] = ACTIONS(83),
    [sym_null] = ACTIONS(83),
    [anon_sym_empty] = ACTIONS(83),
    [anon_sym_not] = ACTIONS(83),
    [anon_sym_and] = ACTIONS(83),
    [anon_sym_or] = ACTIONS(83),
    [anon_sym_div] = ACTIONS(83),
    [anon_sym_mod] = ACTIONS(83),
    [anon_sym_eq] = ACTIONS(83),
    [anon_sym_ne] = ACTIONS(83),
    [anon_sym_lt] = ACTIONS(83),
    [anon_sym_gt] = ACTIONS(83),
    [anon_sym_le] = ACTIONS(83),
    [anon_sym_ge] = ACTIONS(83),
    [anon_sym_instanceof] = ACTIONS(83),
    [anon_sym_EQ_EQ] = ACTIONS(85),
    [anon_sym_BANG_EQ] = ACTIONS(85),
    [anon_sym_LT_EQ] = ACTIONS(85),
    [anon_sym_GT_EQ] = ACTIONS(85),
    [anon_sym_GT] = ACTIONS(83),
    [anon_sym_AMP_AMP] = ACTIONS(85),
    [anon_sym_PIPE_PIPE] = ACTIONS(85),
    [anon_sym_BANG] = ACTIONS(83),
    [anon_sym_PLUS] = ACTIONS(83),
    [anon_sym_STAR] = ACTIONS(85),
    [anon_sym_SLASH] = ACTIONS(85),
    [anon_sym_QMARK] = ACTIONS(85),
    [anon_sym_PLUS_EQ] = ACTIONS(85),
    [anon_sym_DASH_GT] = ACTIONS(85),
    [anon_sym_SEMI] = ACTIONS(85),
  },
  [9] = {
    [anon_sym_LT] = ACTIONS(87),
    [anon_sym_DASH] = ACTIONS(87),
    [aux_sym_directive_token1] = ACTIONS(89),
    [anon_sym_EQ] = ACTIONS(87),
    [anon_sym_PERCENT] = ACTIONS(89),
    [anon_sym_RBRACE] = ACTIONS(89),
    [anon_sym_DOT] = ACTIONS(87),
    [anon_sym_COMMA] = ACTIONS(89),
    [anon_sym_COLON] = ACTIONS(89),
    [anon_sym_LPAREN] = ACTIONS(89),
    [anon_sym_RPAREN] = ACTIONS(89),
    [anon_sym_LBRACK] = ACTIONS(89),
    [anon_sym_RBRACK] = ACTIONS(89),
    [anon_sym_LBRACE] = ACTIONS(89),
    [sym_identifier] = ACTIONS(87),
    [aux_sym_string_token1] = ACTIONS(89),
    [aux_sym_string_token2] = ACTIONS(89),
    [sym_number] = ACTIONS(89),
    [anon_sym_true] = ACTIONS(87),
    [anon_sym_false] = ACTIONS(87),
    [sym_null] = ACTIONS(87),
    [anon_sym_empty] = ACTIONS(87),
    [anon_sym_not] = ACTIONS(87),
    [anon_sym_and] = ACTIONS(87),
    [anon_sym_or] = ACTIONS(87),
    [anon_sym_div] = ACTIONS(87),
    [anon_sym_mod] = ACTIONS(87),
    [anon_sym_eq] = ACTIONS(87),
    [anon_sym_ne] = ACTIONS(87),
    [anon_sym_lt] = ACTIONS(87),
    [anon_sym_gt] = ACTIONS(87),
    [anon_sym_le] = ACTIONS(87),
    [anon_sym_ge] = ACTIONS(87),
    [anon_sym_instanceof] = ACTIONS(87),
    [anon_sym_EQ_EQ] = ACTIONS(89),
    [anon_sym_BANG_EQ] = ACTIONS(89),
    [anon_sym_LT_EQ] = ACTIONS(89),
    [anon_sym_GT_EQ] = ACTIONS(89),
    [anon_sym_GT] = ACTIONS(87),
    [anon_sym_AMP_AMP] = ACTIONS(89),
    [anon_sym_PIPE_PIPE] = ACTIONS(89),
    [anon_sym_BANG] = ACTIONS(87),
    [anon_sym_PLUS] = ACTIONS(87),
    [anon_sym_STAR] = ACTIONS(89),
    [anon_sym_SLASH] = ACTIONS(89),
    [anon_sym_QMARK] = ACTIONS(89),
    [anon_sym_PLUS_EQ] = ACTIONS(89),
    [anon_sym_DASH_GT] = ACTIONS(89),
    [anon_sym_SEMI] = ACTIONS(89),
  },
  [10] = {
    [anon_sym_LT] = ACTIONS(91),
    [anon_sym_DASH] = ACTIONS(91),
    [aux_sym_directive_token1] = ACTIONS(93),
    [anon_sym_EQ] = ACTIONS(91),
    [anon_sym_PERCENT] = ACTIONS(93),
    [anon_sym_RBRACE] = ACTIONS(93),
    [anon_sym_DOT] = ACTIONS(91),
    [anon_sym_COMMA] = ACTIONS(93),
    [anon_sym_COLON] = ACTIONS(93),
    [anon_sym_LPAREN] = ACTIONS(93),
    [anon_sym_RPAREN] = ACTIONS(93),
    [anon_sym_LBRACK] = ACTIONS(93),
    [anon_sym_RBRACK] = ACTIONS(93),
    [anon_sym_LBRACE] = ACTIONS(93),
    [sym_identifier] = ACTIONS(91),
    [aux_sym_string_token1] = ACTIONS(93),
    [aux_sym_string_token2] = ACTIONS(93),
    [sym_number] = ACTIONS(93),
    [anon_sym_true] = ACTIONS(91),
    [anon_sym_false] = ACTIONS(91),
    [sym_null] = ACTIONS(91),
    [anon_sym_empty] = ACTIONS(91),
    [anon_sym_not] = ACTIONS(91),
    [anon_sym_and] = ACTIONS(91),
    [anon_sym_or] = ACTIONS(91),
    [anon_sym_div] = ACTIONS(91),
    [anon_sym_mod] = ACTIONS(91),
    [anon_sym_eq] = ACTIONS(91),
    [anon_sym_ne] = ACTIONS(91),
    [anon_sym_lt] = ACTIONS(91),
    [anon_sym_gt] = ACTIONS(91),
    [anon_sym_le] = ACTIONS(91),
    [anon_sym_ge] = ACTIONS(91),
    [anon_sym_instanceof] = ACTIONS(91),
    [anon_sym_EQ_EQ] = ACTIONS(93),
    [anon_sym_BANG_EQ] = ACTIONS(93),
    [anon_sym_LT_EQ] = ACTIONS(93),
    [anon_sym_GT_EQ] = ACTIONS(93),
    [anon_sym_GT] = ACTIONS(91),
    [anon_sym_AMP_AMP] = ACTIONS(93),
    [anon_sym_PIPE_PIPE] = ACTIONS(93),
    [anon_sym_BANG] = ACTIONS(91),
    [anon_sym_PLUS] = ACTIONS(91),
    [anon_sym_STAR] = ACTIONS(93),
    [anon_sym_SLASH] = ACTIONS(93),
    [anon_sym_QMARK] = ACTIONS(93),
    [anon_sym_PLUS_EQ] = ACTIONS(93),
    [anon_sym_DASH_GT] = ACTIONS(93),
    [anon_sym_SEMI] = ACTIONS(93),
  },
  [11] = {
    [anon_sym_LT] = ACTIONS(95),
    [anon_sym_DASH] = ACTIONS(95),
    [aux_sym_directive_token1] = ACTIONS(97),
    [anon_sym_EQ] = ACTIONS(95),
    [anon_sym_PERCENT] = ACTIONS(97),
    [anon_sym_RBRACE] = ACTIONS(97),
    [anon_sym_DOT] = ACTIONS(95),
    [anon_sym_COMMA] = ACTIONS(97),
    [anon_sym_COLON] = ACTIONS(97),
    [anon_sym_LPAREN] = ACTIONS(97),
    [anon_sym_RPAREN] = ACTIONS(97),
    [anon_sym_LBRACK] = ACTIONS(97),
    [anon_sym_RBRACK] = ACTIONS(97),
    [anon_sym_LBRACE] = ACTIONS(97),
    [sym_identifier] = ACTIONS(95),
    [aux_sym_string_token1] = ACTIONS(97),
    [aux_sym_string_token2] = ACTIONS(97),
    [sym_number] = ACTIONS(97),
    [anon_sym_true] = ACTIONS(95),
    [anon_sym_false] = ACTIONS(95),
    [sym_null] = ACTIONS(95),
    [anon_sym_empty] = ACTIONS(95),
    [anon_sym_not] = ACTIONS(95),
    [anon_sym_and] = ACTIONS(95),
    [anon_sym_or] = ACTIONS(95),
    [anon_sym_div] = ACTIONS(95),
    [anon_sym_mod] = ACTIONS(95),
    [anon_sym_eq] = ACTIONS(95),
    [anon_sym_ne] = ACTIONS(95),
    [anon_sym_lt] = ACTIONS(95),
    [anon_sym_gt] = ACTIONS(95),
    [anon_sym_le] = ACTIONS(95),
    [anon_sym_ge] = ACTIONS(95),
    [anon_sym_instanceof] = ACTIONS(95),
    [anon_sym_EQ_EQ] = ACTIONS(97),
    [anon_sym_BANG_EQ] = ACTIONS(97),
    [anon_sym_LT_EQ] = ACTIONS(97),
    [anon_sym_GT_EQ] = ACTIONS(97),
    [anon_sym_GT] = ACTIONS(95),
    [anon_sym_AMP_AMP] = ACTIONS(97),
    [anon_sym_PIPE_PIPE] = ACTIONS(97),
    [anon_sym_BANG] = ACTIONS(95),
    [anon_sym_PLUS] = ACTIONS(95),
    [anon_sym_STAR] = ACTIONS(97),
    [anon_sym_SLASH] = ACTIONS(97),
    [anon_sym_QMARK] = ACTIONS(97),
    [anon_sym_PLUS_EQ] = ACTIONS(97),
    [anon_sym_DASH_GT] = ACTIONS(97),
    [anon_sym_SEMI] = ACTIONS(97),
  },
  [12] = {
    [anon_sym_LT] = ACTIONS(99),
    [anon_sym_DASH] = ACTIONS(99),
    [aux_sym_directive_token1] = ACTIONS(101),
    [anon_sym_EQ] = ACTIONS(99),
    [anon_sym_PERCENT] = ACTIONS(101),
    [anon_sym_RBRACE] = ACTIONS(101),
    [anon_sym_DOT] = ACTIONS(99),
    [anon_sym_COMMA] = ACTIONS(101),
    [anon_sym_COLON] = ACTIONS(101),
    [anon_sym_LPAREN] = ACTIONS(101),
    [anon_sym_RPAREN] = ACTIONS(101),
    [anon_sym_LBRACK] = ACTIONS(101),
    [anon_sym_RBRACK] = ACTIONS(101),
    [anon_sym_LBRACE] = ACTIONS(101),
    [sym_identifier] = ACTIONS(99),
    [aux_sym_string_token1] = ACTIONS(101),
    [aux_sym_string_token2] = ACTIONS(101),
    [sym_number] = ACTIONS(101),
    [anon_sym_true] = ACTIONS(99),
    [anon_sym_false] = ACTIONS(99),
    [sym_null] = ACTIONS(99),
    [anon_sym_empty] = ACTIONS(99),
    [anon_sym_not] = ACTIONS(99),
    [anon_sym_and] = ACTIONS(99),
    [anon_sym_or] = ACTIONS(99),
    [anon_sym_div] = ACTIONS(99),
    [anon_sym_mod] = ACTIONS(99),
    [anon_sym_eq] = ACTIONS(99),
    [anon_sym_ne] = ACTIONS(99),
    [anon_sym_lt] = ACTIONS(99),
    [anon_sym_gt] = ACTIONS(99),
    [anon_sym_le] = ACTIONS(99),
    [anon_sym_ge] = ACTIONS(99),
    [anon_sym_instanceof] = ACTIONS(99),
    [anon_sym_EQ_EQ] = ACTIONS(101),
    [anon_sym_BANG_EQ] = ACTIONS(101),
    [anon_sym_LT_EQ] = ACTIONS(101),
    [anon_sym_GT_EQ] = ACTIONS(101),
    [anon_sym_GT] = ACTIONS(99),
    [anon_sym_AMP_AMP] = ACTIONS(101),
    [anon_sym_PIPE_PIPE] = ACTIONS(101),
    [anon_sym_BANG] = ACTIONS(99),
    [anon_sym_PLUS] = ACTIONS(99),
    [anon_sym_STAR] = ACTIONS(101),
    [anon_sym_SLASH] = ACTIONS(101),
    [anon_sym_QMARK] = ACTIONS(101),
    [anon_sym_PLUS_EQ] = ACTIONS(101),
    [anon_sym_DASH_GT] = ACTIONS(101),
    [anon_sym_SEMI] = ACTIONS(101),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 11,
    ACTIONS(103), 1,
      ts_builtin_sym_end,
    ACTIONS(111), 1,
      anon_sym_LT_PERCENT_DASH_DASH,
    ACTIONS(114), 1,
      anon_sym_LT_PERCENT_AT,
    ACTIONS(117), 1,
      anon_sym_LT_PERCENT_BANG,
    ACTIONS(120), 1,
      anon_sym_LT_PERCENT_EQ,
    ACTIONS(123), 1,
      anon_sym_LT_PERCENT,
    STATE(16), 1,
      aux_sym_content_repeat1,
    ACTIONS(126), 2,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
    ACTIONS(105), 3,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
    ACTIONS(108), 4,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
    STATE(13), 9,
      sym__node,
      sym_content,
      sym_comment,
      sym_directive,
      sym_declaration,
      sym_expression,
      sym_scriptlet,
      sym_el_expression,
      aux_sym_document_repeat1,
  [48] = 11,
    ACTIONS(9), 1,
      anon_sym_LT_PERCENT_DASH_DASH,
    ACTIONS(11), 1,
      anon_sym_LT_PERCENT_AT,
    ACTIONS(13), 1,
      anon_sym_LT_PERCENT_BANG,
    ACTIONS(15), 1,
      anon_sym_LT_PERCENT_EQ,
    ACTIONS(17), 1,
      anon_sym_LT_PERCENT,
    ACTIONS(129), 1,
      ts_builtin_sym_end,
    STATE(16), 1,
      aux_sym_content_repeat1,
    ACTIONS(19), 2,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
    ACTIONS(5), 3,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
    ACTIONS(7), 4,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
    STATE(13), 9,
      sym__node,
      sym_content,
      sym_comment,
      sym_directive,
      sym_declaration,
      sym_expression,
      sym_scriptlet,
      sym_el_expression,
      aux_sym_document_repeat1,
  [96] = 5,
    ACTIONS(139), 1,
      anon_sym_LT_PERCENT,
    STATE(15), 1,
      aux_sym_content_repeat1,
    ACTIONS(133), 3,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
    ACTIONS(136), 4,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
    ACTIONS(131), 7,
      ts_builtin_sym_end,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [123] = 5,
    ACTIONS(147), 1,
      anon_sym_LT_PERCENT,
    STATE(15), 1,
      aux_sym_content_repeat1,
    ACTIONS(143), 3,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
    ACTIONS(145), 4,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
    ACTIONS(141), 7,
      ts_builtin_sym_end,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [150] = 2,
    ACTIONS(151), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(149), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [170] = 2,
    ACTIONS(155), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(153), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [190] = 2,
    ACTIONS(159), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(157), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [210] = 2,
    ACTIONS(163), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(161), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [230] = 2,
    ACTIONS(167), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(165), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [250] = 2,
    ACTIONS(171), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(169), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [270] = 2,
    ACTIONS(175), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(173), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [290] = 2,
    ACTIONS(179), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(177), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [310] = 2,
    ACTIONS(183), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(181), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [330] = 2,
    ACTIONS(187), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(185), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [350] = 2,
    ACTIONS(191), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(189), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [370] = 2,
    ACTIONS(195), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(193), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [390] = 2,
    ACTIONS(199), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(197), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [410] = 2,
    ACTIONS(203), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(201), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [430] = 2,
    ACTIONS(207), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(205), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [450] = 2,
    ACTIONS(211), 5,
      anon_sym_LT,
      anon_sym_DOLLAR,
      anon_sym_POUND,
      anon_sym_BSLASH,
      anon_sym_LT_PERCENT,
    ACTIONS(209), 10,
      ts_builtin_sym_end,
      aux_sym_content_token1,
      anon_sym_BSLASH_DOLLAR,
      anon_sym_BSLASH_POUND,
      anon_sym_LT_PERCENT_DASH_DASH,
      anon_sym_LT_PERCENT_AT,
      anon_sym_LT_PERCENT_BANG,
      anon_sym_LT_PERCENT_EQ,
      anon_sym_DOLLAR_LBRACE,
      anon_sym_POUND_LBRACE,
  [470] = 5,
    ACTIONS(213), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(215), 1,
      aux_sym_code_token1,
    ACTIONS(217), 1,
      anon_sym_PERCENT,
    STATE(40), 1,
      aux_sym_code_repeat1,
    STATE(64), 1,
      sym_code,
  [486] = 5,
    ACTIONS(215), 1,
      aux_sym_code_token1,
    ACTIONS(217), 1,
      anon_sym_PERCENT,
    ACTIONS(219), 1,
      anon_sym_PERCENT_GT,
    STATE(40), 1,
      aux_sym_code_repeat1,
    STATE(62), 1,
      sym_code,
  [502] = 5,
    ACTIONS(215), 1,
      aux_sym_code_token1,
    ACTIONS(217), 1,
      anon_sym_PERCENT,
    ACTIONS(221), 1,
      anon_sym_PERCENT_GT,
    STATE(40), 1,
      aux_sym_code_repeat1,
    STATE(67), 1,
      sym_code,
  [518] = 3,
    ACTIONS(223), 1,
      aux_sym_directive_token1,
    STATE(58), 1,
      sym_attribute_value,
    ACTIONS(225), 2,
      aux_sym_attribute_value_token1,
      aux_sym_attribute_value_token2,
  [529] = 4,
    ACTIONS(227), 1,
      aux_sym_comment_token1,
    ACTIONS(230), 1,
      anon_sym_DASH,
    ACTIONS(233), 1,
      anon_sym_DASH_DASH_PERCENT_GT,
    STATE(37), 1,
      aux_sym_comment_repeat1,
  [542] = 4,
    ACTIONS(235), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(237), 1,
      aux_sym_code_token1,
    ACTIONS(240), 1,
      anon_sym_PERCENT,
    STATE(38), 1,
      aux_sym_code_repeat1,
  [555] = 4,
    ACTIONS(243), 1,
      aux_sym_comment_token1,
    ACTIONS(245), 1,
      anon_sym_DASH,
    ACTIONS(247), 1,
      anon_sym_DASH_DASH_PERCENT_GT,
    STATE(41), 1,
      aux_sym_comment_repeat1,
  [568] = 4,
    ACTIONS(249), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(251), 1,
      aux_sym_code_token1,
    ACTIONS(253), 1,
      anon_sym_PERCENT,
    STATE(38), 1,
      aux_sym_code_repeat1,
  [581] = 4,
    ACTIONS(255), 1,
      aux_sym_comment_token1,
    ACTIONS(257), 1,
      anon_sym_DASH,
    ACTIONS(259), 1,
      anon_sym_DASH_DASH_PERCENT_GT,
    STATE(37), 1,
      aux_sym_comment_repeat1,
  [594] = 3,
    ACTIONS(261), 1,
      aux_sym_directive_token1,
    STATE(60), 1,
      sym_attribute_value,
    ACTIONS(225), 2,
      aux_sym_attribute_value_token1,
      aux_sym_attribute_value_token2,
  [605] = 3,
    ACTIONS(263), 1,
      aux_sym_directive_token1,
    ACTIONS(265), 1,
      anon_sym_PERCENT_GT,
    STATE(47), 1,
      aux_sym_directive_repeat1,
  [615] = 3,
    ACTIONS(267), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(269), 1,
      sym_attribute_name,
    STATE(54), 1,
      sym_attribute,
  [625] = 3,
    ACTIONS(269), 1,
      sym_attribute_name,
    ACTIONS(271), 1,
      anon_sym_PERCENT_GT,
    STATE(54), 1,
      sym_attribute,
  [635] = 3,
    ACTIONS(269), 1,
      sym_attribute_name,
    ACTIONS(273), 1,
      anon_sym_PERCENT_GT,
    STATE(54), 1,
      sym_attribute,
  [645] = 3,
    ACTIONS(271), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(275), 1,
      aux_sym_directive_token1,
    STATE(52), 1,
      aux_sym_directive_repeat1,
  [655] = 2,
    STATE(61), 1,
      sym_attribute_value,
    ACTIONS(225), 2,
      aux_sym_attribute_value_token1,
      aux_sym_attribute_value_token2,
  [663] = 2,
    STATE(60), 1,
      sym_attribute_value,
    ACTIONS(225), 2,
      aux_sym_attribute_value_token1,
      aux_sym_attribute_value_token2,
  [671] = 3,
    ACTIONS(273), 1,
      anon_sym_PERCENT_GT,
    ACTIONS(277), 1,
      aux_sym_directive_token1,
    STATE(52), 1,
      aux_sym_directive_repeat1,
  [681] = 3,
    ACTIONS(269), 1,
      sym_attribute_name,
    ACTIONS(279), 1,
      anon_sym_PERCENT_GT,
    STATE(54), 1,
      sym_attribute,
  [691] = 3,
    ACTIONS(281), 1,
      aux_sym_directive_token1,
    ACTIONS(284), 1,
      anon_sym_PERCENT_GT,
    STATE(52), 1,
      aux_sym_directive_repeat1,
  [701] = 3,
    ACTIONS(286), 1,
      aux_sym_directive_token1,
    ACTIONS(288), 1,
      anon_sym_PERCENT_GT,
    STATE(50), 1,
      aux_sym_directive_repeat1,
  [711] = 1,
    ACTIONS(284), 2,
      aux_sym_directive_token1,
      anon_sym_PERCENT_GT,
  [716] = 2,
    ACTIONS(290), 1,
      aux_sym_directive_token1,
    ACTIONS(292), 1,
      sym_directive_name,
  [723] = 2,
    ACTIONS(269), 1,
      sym_attribute_name,
    STATE(54), 1,
      sym_attribute,
  [730] = 1,
    ACTIONS(294), 2,
      aux_sym_directive_token1,
      anon_sym_PERCENT_GT,
  [735] = 1,
    ACTIONS(296), 2,
      aux_sym_directive_token1,
      anon_sym_PERCENT_GT,
  [740] = 2,
    ACTIONS(298), 1,
      aux_sym_directive_token1,
    ACTIONS(300), 1,
      anon_sym_EQ,
  [747] = 1,
    ACTIONS(302), 2,
      aux_sym_directive_token1,
      anon_sym_PERCENT_GT,
  [752] = 1,
    ACTIONS(304), 2,
      aux_sym_directive_token1,
      anon_sym_PERCENT_GT,
  [757] = 1,
    ACTIONS(306), 1,
      anon_sym_PERCENT_GT,
  [761] = 1,
    ACTIONS(308), 1,
      anon_sym_EQ,
  [765] = 1,
    ACTIONS(310), 1,
      anon_sym_PERCENT_GT,
  [769] = 1,
    ACTIONS(312), 1,
      sym_directive_name,
  [773] = 1,
    ACTIONS(314), 1,
      ts_builtin_sym_end,
  [777] = 1,
    ACTIONS(316), 1,
      anon_sym_PERCENT_GT,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(13)] = 0,
  [SMALL_STATE(14)] = 48,
  [SMALL_STATE(15)] = 96,
  [SMALL_STATE(16)] = 123,
  [SMALL_STATE(17)] = 150,
  [SMALL_STATE(18)] = 170,
  [SMALL_STATE(19)] = 190,
  [SMALL_STATE(20)] = 210,
  [SMALL_STATE(21)] = 230,
  [SMALL_STATE(22)] = 250,
  [SMALL_STATE(23)] = 270,
  [SMALL_STATE(24)] = 290,
  [SMALL_STATE(25)] = 310,
  [SMALL_STATE(26)] = 330,
  [SMALL_STATE(27)] = 350,
  [SMALL_STATE(28)] = 370,
  [SMALL_STATE(29)] = 390,
  [SMALL_STATE(30)] = 410,
  [SMALL_STATE(31)] = 430,
  [SMALL_STATE(32)] = 450,
  [SMALL_STATE(33)] = 470,
  [SMALL_STATE(34)] = 486,
  [SMALL_STATE(35)] = 502,
  [SMALL_STATE(36)] = 518,
  [SMALL_STATE(37)] = 529,
  [SMALL_STATE(38)] = 542,
  [SMALL_STATE(39)] = 555,
  [SMALL_STATE(40)] = 568,
  [SMALL_STATE(41)] = 581,
  [SMALL_STATE(42)] = 594,
  [SMALL_STATE(43)] = 605,
  [SMALL_STATE(44)] = 615,
  [SMALL_STATE(45)] = 625,
  [SMALL_STATE(46)] = 635,
  [SMALL_STATE(47)] = 645,
  [SMALL_STATE(48)] = 655,
  [SMALL_STATE(49)] = 663,
  [SMALL_STATE(50)] = 671,
  [SMALL_STATE(51)] = 681,
  [SMALL_STATE(52)] = 691,
  [SMALL_STATE(53)] = 701,
  [SMALL_STATE(54)] = 711,
  [SMALL_STATE(55)] = 716,
  [SMALL_STATE(56)] = 723,
  [SMALL_STATE(57)] = 730,
  [SMALL_STATE(58)] = 735,
  [SMALL_STATE(59)] = 740,
  [SMALL_STATE(60)] = 747,
  [SMALL_STATE(61)] = 752,
  [SMALL_STATE(62)] = 757,
  [SMALL_STATE(63)] = 761,
  [SMALL_STATE(64)] = 765,
  [SMALL_STATE(65)] = 769,
  [SMALL_STATE(66)] = 773,
  [SMALL_STATE(67)] = 777,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 0, 0, 0),
  [5] = {.entry = {.count = 1, .reusable = true}}, SHIFT(16),
  [7] = {.entry = {.count = 1, .reusable = false}}, SHIFT(16),
  [9] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [11] = {.entry = {.count = 1, .reusable = true}}, SHIFT(55),
  [13] = {.entry = {.count = 1, .reusable = true}}, SHIFT(33),
  [15] = {.entry = {.count = 1, .reusable = true}}, SHIFT(34),
  [17] = {.entry = {.count = 1, .reusable = false}}, SHIFT(35),
  [19] = {.entry = {.count = 1, .reusable = true}}, SHIFT(4),
  [21] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(12),
  [24] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(2),
  [27] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(12),
  [30] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0),
  [32] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(2),
  [35] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(6),
  [38] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(11),
  [41] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(10),
  [44] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_el_expression_repeat1, 2, 0, 0), SHIFT_REPEAT(9),
  [47] = {.entry = {.count = 1, .reusable = false}}, SHIFT(12),
  [49] = {.entry = {.count = 1, .reusable = true}}, SHIFT(2),
  [51] = {.entry = {.count = 1, .reusable = true}}, SHIFT(12),
  [53] = {.entry = {.count = 1, .reusable = true}}, SHIFT(7),
  [55] = {.entry = {.count = 1, .reusable = false}}, SHIFT(2),
  [57] = {.entry = {.count = 1, .reusable = true}}, SHIFT(6),
  [59] = {.entry = {.count = 1, .reusable = true}}, SHIFT(11),
  [61] = {.entry = {.count = 1, .reusable = false}}, SHIFT(10),
  [63] = {.entry = {.count = 1, .reusable = false}}, SHIFT(9),
  [65] = {.entry = {.count = 1, .reusable = true}}, SHIFT(5),
  [67] = {.entry = {.count = 1, .reusable = true}}, SHIFT(23),
  [69] = {.entry = {.count = 1, .reusable = false}}, SHIFT(5),
  [71] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [73] = {.entry = {.count = 1, .reusable = true}}, SHIFT(3),
  [75] = {.entry = {.count = 1, .reusable = true}}, SHIFT(8),
  [77] = {.entry = {.count = 1, .reusable = false}}, SHIFT(3),
  [79] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_el_braces, 3, 0, 0),
  [81] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_el_braces, 3, 0, 0),
  [83] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_el_braces, 2, 0, 0),
  [85] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_el_braces, 2, 0, 0),
  [87] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_keyword_operator, 1, 0, 0),
  [89] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_keyword_operator, 1, 0, 0),
  [91] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_boolean, 1, 0, 0),
  [93] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_boolean, 1, 0, 0),
  [95] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_string, 1, 0, 0),
  [97] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_string, 1, 0, 0),
  [99] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_operator, 1, 0, 0),
  [101] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_operator, 1, 0, 0),
  [103] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0),
  [105] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(16),
  [108] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(16),
  [111] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(39),
  [114] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(55),
  [117] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(33),
  [120] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(34),
  [123] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(35),
  [126] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(4),
  [129] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 1, 0, 0),
  [131] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_content_repeat1, 2, 0, 0),
  [133] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_content_repeat1, 2, 0, 0), SHIFT_REPEAT(15),
  [136] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_content_repeat1, 2, 0, 0), SHIFT_REPEAT(15),
  [139] = {.entry = {.count = 1, .reusable = false}}, REDUCE(aux_sym_content_repeat1, 2, 0, 0),
  [141] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_content, 1, 0, 0),
  [143] = {.entry = {.count = 1, .reusable = true}}, SHIFT(15),
  [145] = {.entry = {.count = 1, .reusable = false}}, SHIFT(15),
  [147] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_content, 1, 0, 0),
  [149] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 3, 0, 1),
  [151] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 3, 0, 1),
  [153] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_scriptlet, 3, 0, 0),
  [155] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_scriptlet, 3, 0, 0),
  [157] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_expression, 2, 0, 0),
  [159] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_expression, 2, 0, 0),
  [161] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_scriptlet, 2, 0, 0),
  [163] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_scriptlet, 2, 0, 0),
  [165] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 4, 0, 1),
  [167] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 4, 0, 1),
  [169] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_el_expression, 3, 0, 0),
  [171] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_el_expression, 3, 0, 0),
  [173] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_el_expression, 2, 0, 0),
  [175] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_el_expression, 2, 0, 0),
  [177] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 6, 0, 2),
  [179] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 6, 0, 2),
  [181] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 5, 0, 1),
  [183] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 5, 0, 1),
  [185] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment, 2, 0, 0),
  [187] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment, 2, 0, 0),
  [189] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 5, 0, 2),
  [191] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 5, 0, 2),
  [193] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 4, 0, 2),
  [195] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 4, 0, 2),
  [197] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_declaration, 3, 0, 0),
  [199] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_declaration, 3, 0, 0),
  [201] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_declaration, 2, 0, 0),
  [203] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_declaration, 2, 0, 0),
  [205] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment, 3, 0, 0),
  [207] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment, 3, 0, 0),
  [209] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_expression, 3, 0, 0),
  [211] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_expression, 3, 0, 0),
  [213] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [215] = {.entry = {.count = 1, .reusable = true}}, SHIFT(40),
  [217] = {.entry = {.count = 1, .reusable = false}}, SHIFT(40),
  [219] = {.entry = {.count = 1, .reusable = true}}, SHIFT(19),
  [221] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [223] = {.entry = {.count = 1, .reusable = true}}, SHIFT(49),
  [225] = {.entry = {.count = 1, .reusable = true}}, SHIFT(57),
  [227] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_comment_repeat1, 2, 0, 0), SHIFT_REPEAT(37),
  [230] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_comment_repeat1, 2, 0, 0), SHIFT_REPEAT(37),
  [233] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_comment_repeat1, 2, 0, 0),
  [235] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_code_repeat1, 2, 0, 0),
  [237] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_code_repeat1, 2, 0, 0), SHIFT_REPEAT(38),
  [240] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_code_repeat1, 2, 0, 0), SHIFT_REPEAT(38),
  [243] = {.entry = {.count = 1, .reusable = true}}, SHIFT(41),
  [245] = {.entry = {.count = 1, .reusable = false}}, SHIFT(41),
  [247] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [249] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_code, 1, 0, 0),
  [251] = {.entry = {.count = 1, .reusable = true}}, SHIFT(38),
  [253] = {.entry = {.count = 1, .reusable = false}}, SHIFT(38),
  [255] = {.entry = {.count = 1, .reusable = true}}, SHIFT(37),
  [257] = {.entry = {.count = 1, .reusable = false}}, SHIFT(37),
  [259] = {.entry = {.count = 1, .reusable = true}}, SHIFT(31),
  [261] = {.entry = {.count = 1, .reusable = true}}, SHIFT(48),
  [263] = {.entry = {.count = 1, .reusable = true}}, SHIFT(45),
  [265] = {.entry = {.count = 1, .reusable = true}}, SHIFT(28),
  [267] = {.entry = {.count = 1, .reusable = true}}, SHIFT(24),
  [269] = {.entry = {.count = 1, .reusable = true}}, SHIFT(59),
  [271] = {.entry = {.count = 1, .reusable = true}}, SHIFT(27),
  [273] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [275] = {.entry = {.count = 1, .reusable = true}}, SHIFT(44),
  [277] = {.entry = {.count = 1, .reusable = true}}, SHIFT(51),
  [279] = {.entry = {.count = 1, .reusable = true}}, SHIFT(25),
  [281] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_directive_repeat1, 2, 0, 0), SHIFT_REPEAT(56),
  [284] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_directive_repeat1, 2, 0, 0),
  [286] = {.entry = {.count = 1, .reusable = true}}, SHIFT(46),
  [288] = {.entry = {.count = 1, .reusable = true}}, SHIFT(17),
  [290] = {.entry = {.count = 1, .reusable = true}}, SHIFT(65),
  [292] = {.entry = {.count = 1, .reusable = true}}, SHIFT(53),
  [294] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute_value, 1, 0, 0),
  [296] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 3, 0, 3),
  [298] = {.entry = {.count = 1, .reusable = true}}, SHIFT(63),
  [300] = {.entry = {.count = 1, .reusable = true}}, SHIFT(36),
  [302] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 4, 0, 4),
  [304] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 5, 0, 5),
  [306] = {.entry = {.count = 1, .reusable = true}}, SHIFT(32),
  [308] = {.entry = {.count = 1, .reusable = true}}, SHIFT(42),
  [310] = {.entry = {.count = 1, .reusable = true}}, SHIFT(29),
  [312] = {.entry = {.count = 1, .reusable = true}}, SHIFT(43),
  [314] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
  [316] = {.entry = {.count = 1, .reusable = true}}, SHIFT(18),
};

#ifdef __cplusplus
extern "C" {
#endif
#ifdef TREE_SITTER_HIDE_SYMBOLS
#define TS_PUBLIC
#elif defined(_WIN32)
#define TS_PUBLIC __declspec(dllexport)
#else
#define TS_PUBLIC __attribute__((visibility("default")))
#endif

TS_PUBLIC const TSLanguage *tree_sitter_jsp(void) {
  static const TSLanguage language = {
    .version = LANGUAGE_VERSION,
    .symbol_count = SYMBOL_COUNT,
    .alias_count = ALIAS_COUNT,
    .token_count = TOKEN_COUNT,
    .external_token_count = EXTERNAL_TOKEN_COUNT,
    .state_count = STATE_COUNT,
    .large_state_count = LARGE_STATE_COUNT,
    .production_id_count = PRODUCTION_ID_COUNT,
    .field_count = FIELD_COUNT,
    .max_alias_sequence_length = MAX_ALIAS_SEQUENCE_LENGTH,
    .parse_table = &ts_parse_table[0][0],
    .small_parse_table = ts_small_parse_table,
    .small_parse_table_map = ts_small_parse_table_map,
    .parse_actions = ts_parse_actions,
    .symbol_names = ts_symbol_names,
    .field_names = ts_field_names,
    .field_map_slices = ts_field_map_slices,
    .field_map_entries = ts_field_map_entries,
    .symbol_metadata = ts_symbol_metadata,
    .public_symbol_map = ts_symbol_map,
    .alias_map = ts_non_terminal_alias_map,
    .alias_sequences = &ts_alias_sequences[0][0],
    .lex_modes = ts_lex_modes,
    .lex_fn = ts_lex,
    .primary_state_ids = ts_primary_state_ids,
  };
  return &language;
}
#ifdef __cplusplus
}
#endif

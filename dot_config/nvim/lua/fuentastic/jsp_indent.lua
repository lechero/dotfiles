-- indentexpr for JSP. nvim-treesitter's indent goes by the smallest tree around a line, which
-- around scriptlets is often the Java one for a line of HTML, or the other way round. Here markup
-- is indented by its HTML nesting and scriptlet lines by their Java braces; -1 keeps a line's indent.
local M = {}

-- Lines inside these are a level deeper than the line the node starts on.
local java_blocks = {
  block = true,
  class_body = true,
  interface_body = true,
  enum_body = true,
  constructor_body = true,
  annotation_type_body = true,
  switch_block = true,
  switch_block_statement_group = true, -- statements under a `case x:`
  array_initializer = true,
  element_value_array_initializer = true,
}

local html_elements = { element = true, script_element = true, style_element = true }

-- Elements without an end tag.
local void = {}
for _, name in ipairs { 'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'param', 'source', 'track', 'wbr' } do
  void[name] = true
end

-- The root of the `lang` tree that spans the position. The combined trees (all the markup, all the
-- scriptlets) also span what's between their parts, which a markup line between two scriptlets
-- needs: it can be inside a Java block.
local function root(parser, lang, row, col)
  local child = parser:children()[lang]
  local found
  for _, tree in pairs(child and child:trees() or {}) do
    local node = tree:root()
    if vim.treesitter.is_in_node_range(node, row, col) and (not found or node:byte_length() > found:byte_length()) then
      found = node
    end
  end
  return found
end

-- Java nesting, counted from the scriptlet's own column zero: a level below its <% line, or that
-- line itself when the code starts on it (`<% if (x) {`).
local function java_indent(parser, tag, row, col, text)
  local tag_row = tag:start()
  local _, code_col = tag:named_child(0):start()
  local opener = vim.fn.getline(tag_row + 1)
  local base = vim.fn.indent(tag_row + 1) + (opener:sub(code_col + 1):match '^%s*$' and vim.fn.shiftwidth() or 0)

  local java = root(parser, 'java', row, col)
  local node = java and java:descendant_for_range(row, col, row, col)
  if not node then
    return base -- before the scriptlet's first statement or after its last
  end
  if node:type():match 'comment' or node:type() == 'string_fragment' then
    return -1 -- inside a /* */ comment or a text block
  end

  local closing = text:sub(col + 1, col + 1) == '}'
  local depth = 0
  local statement
  while node do
    local parent = node:parent()
    if not statement and (not parent or java_blocks[parent:type()] or parent:type() == 'program') then
      statement = node
    end
    if java_blocks[node:type()] and node:start() < row then
      if closing and node:end_() == row then
        closing = false -- the block this line closes
      else
        depth = depth + 1
      end
    end
    node = parent
  end
  -- A statement continued from an earlier line (a chained call, a long condition, `else` on its
  -- own line) keeps the indent it was given.
  if text:match '%S' and statement and statement:start() < row and text:sub(col + 1, col + 1) ~= '}' then
    return -1
  end
  return base + depth * vim.fn.shiftwidth()
end

-- The Java block a one-line tag opens around this line: `<% if (x) { %>` ... `<% } %>`. Not one
-- opened inside a scriptlet of several lines: the markup after it stays at the tag's level.
local function tag_block(parser, row, col)
  local java = root(parser, 'java', row, col)
  local node = java and java:descendant_for_range(row, col, row, col)
  while node do
    if java_blocks[node:type()] and node:start() < row and node:end_() > row then
      if vim.fn.getline(node:start() + 1):match '^%s*<%%' then
        return node
      end
    end
    node = node:parent()
  end
end

-- Markup: a level below what the line is in, the closest of an HTML element and a tag's Java block.
-- An end tag goes at its element's indent.
local function html_indent(parser, row, col, text)
  local html = root(parser, 'html', row, col)
  local node = html and html:descendant_for_range(row, col, row, col)
  if node and node:type() == 'raw_text' then
    return require('nvim-treesitter').indentexpr() -- inside <script> or <style>: the JS or CSS tree's
  end
  local element
  while node and not element do
    if (node:type() == 'start_tag' or node:type() == 'self_closing_tag') and node:start() < row then
      return -1 -- attributes over several lines keep their alignment
    end
    if html_elements[node:type()] and node:start() < row then
      element = node
    end
    node = node:parent()
  end

  local block = tag_block(parser, row, col)
  if block and (not element or block:start() > element:start()) then
    return vim.fn.indent(block:start() + 1) + vim.fn.shiftwidth()
  end
  if not element then
    return -1
  end
  local last = element:child(element:child_count() - 1)
  local closing = text:sub(col + 1, col + 2) == '</' and last and last:type() == 'end_tag' and last:start() == row
  return vim.fn.indent(element:start() + 1) + (closing and 0 or vim.fn.shiftwidth())
end

-- Whether a line ends opening something its next line goes in: a JSP tag (`<%`), a Java block
-- (`if (x) {`, `<% if (x) { %>`), an HTML element (`<ul>`).
local function opens(parser, lnum)
  local text = vim.fn.getline(lnum):gsub('%s+$', '')
  if text:match '<%%!?$' or text:match '{%s*%%>$' then
    return true
  end
  if text:match '{$' then
    local node = parser:trees()[1]:root():descendant_for_range(lnum - 1, #text - 1, lnum - 1, #text - 1)
    return node:type() ~= 'content' -- Java, not CSS
  end
  local tag = text:match '<(%a[%w:%-]*)[^<>]*>$'
  return tag ~= nil and not text:match '/>$' and not void[tag:lower()] and not text:find('</' .. tag .. '>', 1, true)
end

-- The indent for the line at `row`, from what's at `col`, its first non-blank (0 when it's blank).
local function indent_at(parser, row, col, text)
  -- The JSP tag the line is inside of, if it started on an earlier line.
  local node = parser:trees()[1]:root():descendant_for_range(row, col, row, col)
  while node and node:parent() and node:parent():type() ~= 'document' do
    node = node:parent()
  end
  if node and node:type() ~= 'content' and node:type() ~= 'document' and node:start() < row then
    if text:sub(col + 1, col + 2) == '%>' then
      return vim.fn.indent(node:start() + 1)
    end
    local code = node:named_child(0)
    local in_code = code and code:type() == 'code' and vim.treesitter.is_in_node_range(code, row, col)
    if in_code and (node:type() == 'scriptlet' or node:type() == 'declaration') then
      return java_indent(parser, node, row, col, text)
    end
    return -1 -- a comment, an expression or a directive over several lines
  end
  return html_indent(parser, row, col, text)
end

-- The line the last call indented. `=` asks for one line after the other and only changes the
-- previous line's indent in between, which leaves the trees right (edits move their nodes along):
-- reparsing the page for each line made `gg=G` quadratic.
local last = {}

function M.indentexpr()
  local lnum = vim.v.lnum
  local parser = vim.treesitter.get_parser(0, 'jsp', { error = false })
  if not parser then
    return -1
  end
  local buf, tick = vim.api.nvim_get_current_buf(), vim.b.changedtick
  if not (last.buf == buf and last.lnum == lnum - 1 and tick - last.tick <= 1) then
    parser:parse(true)
  end
  last = { buf = buf, lnum = lnum, tick = tick }

  local text = vim.fn.getline(lnum)
  local blank = not text:match '%S'
  local indent = indent_at(parser, lnum - 1, blank and 0 or #text:match '^%s*', text)
  -- A blank line, as `o` makes: the trees don't know about what the previous line has just opened
  -- while nothing closes it yet.
  local prev = vim.fn.prevnonblank(lnum - 1)
  if blank and prev > 0 and indent <= vim.fn.indent(prev) and opens(parser, prev) then
    return vim.fn.indent(prev) + vim.fn.shiftwidth()
  end
  return indent
end

return M

// Checks a file of a Backstage template's skeleton the way fetch:template renders it: Nunjucks, with
// ${{ }} for variables (Backstage's SecureTemplater). nvim-lint runs it (lua/fuentastic/plugins/lint.lua).
//
//   node nunjucks-lint.js <folder Nunjucks is installed in> <language of the rendered file> < template
//
// Prints a JSON diagnostic per line: { line, col, severity, message }, 1-based.
//
// Nunjucks syntax is checked by compiling the template. A JSON template is also rendered, every value
// a placeholder that's a string ("0") to print, a list of two to loop over, an object of more
// placeholders to look into: each loop runs twice, so the commas between items are checked too.
// That's one way through the ifs, not all of them, so the rendered JSON's problems are warnings.
'use strict';

const path = require('path');
const vm = require('vm');

const [dir, lang] = process.argv.slice(2);
const nunjucks = require(path.join(dir, 'node_modules', 'nunjucks'));
const jsonc = require(path.join(dir, 'node_modules', 'jsonc-parser'));

const source = require('fs').readFileSync(0, 'utf8');
const opts = { autoescape: false, tags: { variableStart: '${{', variableEnd: '}}' } };

function report(line, col, severity, message) {
  console.log(JSON.stringify({ line: line || 1, col: col || 1, severity, message }));
}

try {
  nunjucks.compiler.compile(source, [], [], 'template', opts);
} catch (err) {
  // "unexpected end of file" (a block left open) has no position: it's where the file ends
  report(err.lineno || source.trimEnd().split('\n').length, err.colno, 'error', err.message);
  process.exit(0);
}

if (lang !== 'json') {
  process.exit(0);
}

// Each line of text starts with a marker of its line number, so a problem in the rendered JSON can
// be put on the template's line. Not inside tags: there it would be part of the expression.
const OPEN = '\uE000';
const CLOSE = '\uE001';
let marked = '';
let inside = null; // the end of the tag the scan is in
source.split('\n').forEach((text, i) => {
  if (i > 0) marked += '\n';
  if (!inside) marked += OPEN + (i + 1) + CLOSE;
  for (let c = 0; c < text.length; c++) {
    if (inside && text.startsWith(inside, c)) {
      c += inside.length - 1;
      inside = null;
    } else if (!inside) {
      inside = text.startsWith('${{', c) ? '}}' : text.startsWith('{%', c) ? '%}' : text.startsWith('{#', c) ? '#}' : null;
    }
  }
  marked += text;
});

// Not a function: Nunjucks would print the wrapper it puts around one.
const placeholder = new Proxy(
  {},
  {
    get(_, key) {
      if (key === Symbol.toPrimitive || key === 'toString' || key === 'valueOf') return () => '0';
      if (key === 'toJSON') return () => '0';
      if (key === 'length') return 2;
      if (key === Symbol.iterator) return () => [placeholder, placeholder][Symbol.iterator]();
      if (typeof key === 'symbol' || key === 'then') return undefined;
      // so split, replace and the like work on it as a string
      if (typeof String.prototype[key] === 'function') return String.prototype[key].bind('0');
      return placeholder;
    },
    has: () => true,
    // `for key, value in object`, `object | length`
    ownKeys: () => ['a', 'b'],
    getOwnPropertyDescriptor: (_, key) =>
      key === 'a' || key === 'b' ? { value: placeholder, enumerable: true, configurable: true, writable: true } : undefined,
  },
);

const env = new nunjucks.Environment(null, opts);
// Backstage's own filters (parseRepoUrl, pick...) and those of the Backstage it runs in
const getFilter = env.getFilter.bind(env);
env.getFilter = (name) => {
  try {
    return getFilter(name);
  } catch {
    return () => placeholder;
  }
};

let rendered;
try {
  const render = () => env.renderString(marked, { values: placeholder });
  rendered = vm.runInNewContext('render()', { render }, { timeout: 2000 });
} catch {
  // a global of the Backstage it runs in, or a value the placeholder can't stand in for
  process.exit(0);
}

let json = '';
const lines = []; // the template's line of each character of json
let line = 1;
for (let i = 0; i < rendered.length; i++) {
  if (rendered[i] === OPEN) {
    const end = rendered.indexOf(CLOSE, i);
    line = Number(rendered.slice(i + 1, end));
    i = end;
  } else {
    json += rendered[i];
    lines.push(line);
  }
}

const errors = [];
jsonc.parse(json, errors, { disallowComments: true, allowTrailingComma: false });
if (errors.length > 0) {
  const { error, offset } = errors[0];
  const problem = jsonc
    .printParseErrorCode(error)
    .replace(/(?<=[a-z])(?=[A-Z])/g, ' ')
    .toLowerCase();
  const near = json
    .slice(Math.max(0, offset - 20), offset + 20)
    .replace(/\s+/g, ' ')
    .trim();
  // On the line before the problem: a missing comma is noticed at the next item, a trailing one at
  // the bracket after it.
  let before = offset - 1;
  while (before > 0 && /\s/.test(json[before])) before--;
  report(lines[Math.max(0, Math.min(before, lines.length - 1))], 1, 'warning', `Renders invalid JSON (${problem}) near: ${near}`);
}

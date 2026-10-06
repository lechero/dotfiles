# Testing

Java tests run through jdtls and the debugger ([java.md](java.md#tests)); Jest tests run through
[neotest](https://github.com/nvim-neotest/neotest) and neotest-jest, on the same keys. The config
lives in `lua/fuentastic/plugins/neotest.lua`.

## Jest

In a test file (`*.test.js`, `*.spec.ts`, `__tests__/`...) of a package that depends on Jest:

| Keys | Does |
| --- | --- |
| `<leader>tm` | Run the test under the cursor |
| `<leader>tc` | Run the whole file |
| `<leader>tl` | Run the last run again |
| `<leader>to` | The output of the test under the cursor, in a float |
| `<leader>ts` | The summary: every test file, with results; run from there too |
| `<leader>tw` | Watch the file: rerun when it changes |

Each test gets a pass or fail sign, and a failure shows as a diagnostic where it failed.

Tests run the way the project runs them: through its `npm test` script when that runs Jest (so its
environment, `TZ=Europe/Amsterdam` or `NODE_ENV=testing` for one, applies), without coverage, from
the folder of the nearest `package.json`. A package without a Jest `test` script uses its
`node_modules/.bin/jest`.

## Troubleshooting

| Symptom | Try |
| --- | --- |
| The keys do nothing | Is the file a test file of a package with `jest` in its dependencies? |
| Every test fails at once | `<leader>to`: usually `npm install`, or a config error in the output |
| A test passes in the shell but not here | Compare with `npm test -- <file>`, which is what runs |

-- GitLab CI files, and the files they include, are yaml.gitlab: gitlab-ci-ls attaches to that
-- filetype only, and everything YAML (treesitter, yamlls, indent, comments) still applies.
vim.filetype.add {
  pattern = {
    ['.*%.gitlab%-ci%.ya?ml'] = 'yaml.gitlab',
    ['.*/%.gitlab/ci/.*%.ya?ml'] = 'yaml.gitlab',
  },
}

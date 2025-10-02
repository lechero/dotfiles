#!/usr/bin/env bash
set -euo pipefail

# Checks whether any JavaScript/TypeScript source files have changed between two refs.
# Excludes common non-source directories (e.g., .github, node_modules, build, dist).
#
# Usage:
#   scripts/source-changed.sh [<base_ref> [<head_ref>]]
#
# Defaults:
#   base_ref = merge-base(HEAD, origin/main) if available; otherwise origin/master; otherwise HEAD~1
#   head_ref = HEAD
#
# Exit codes:
#   0 => JS/TS sources changed
#   1 => No JS/TS source changes
#   2 => Error

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
if [[ -z "${repo_root}" ]]; then
  echo "Error: not inside a git repository" >&2
  exit 2
fi

cd "${repo_root}"

base_ref=""
head_ref="HEAD"

if [[ $# -ge 1 ]]; then
  base_ref="$1"
fi
if [[ $# -ge 2 ]]; then
  head_ref="$2"
fi

if [[ -z "${base_ref}" ]]; then
  # Determine a sane default base ref
  if git rev-parse --verify --quiet origin/main >/dev/null; then
    base_ref=$(git merge-base HEAD origin/main)
  elif git rev-parse --verify --quiet origin/master >/dev/null; then
    base_ref=$(git merge-base HEAD origin/master)
  else
    # Fallback to previous commit
    base_ref="HEAD~1"
  fi
fi

# Query changed files limited to JS/TS and excluding non-source paths using git pathspec.
changed=$(git diff --name-only -z "${base_ref}..${head_ref}" -- \
  ':(glob)**/*.js' \
  ':(glob)**/*.jsx' \
  ':(glob)**/*.ts' \
  ':(glob)**/*.tsx' \
  ':(exclude).github/**' \
  ':(exclude)**/node_modules/**' \
  ':(exclude)build/**' \
  ':(exclude)dist/**' \
  ':(exclude)out/**' \
  ':(exclude)coverage/**' || true)

if [[ -n "${changed}" ]]; then
  # Print changed files (NUL-delimited -> newline for readability)
  printf "%s\n" "JavaScript/TypeScript source changes detected between ${base_ref} and ${head_ref}:"
  printf '%s' "${changed}" | tr '\0' '\n'
  exit 0
else
  echo "No JavaScript/TypeScript source changes between ${base_ref} and ${head_ref}."
  exit 1
fi


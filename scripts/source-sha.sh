#!/usr/bin/env bash
set -euo pipefail

# Computes a single stable SHA over all tracked JavaScript/TypeScript source files
# in the repository, excluding common non-source directories (e.g., .github, node_modules, build, dist).
#
# Usage:
#   scripts/source-sha.sh
#
# Output:
#   Prints a single SHA-256 string representing current JS source state.

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
if [[ -z "${repo_root}" ]]; then
  echo "Error: not inside a git repository" >&2
  exit 2
fi

cd "${repo_root}"

# Pick a hashing command: prefer sha256sum, else fallback to shasum -a 256
if command -v sha256sum >/dev/null 2>&1; then
  HASH_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  HASH_CMD="shasum -a 256"
else
  echo "Error: neither sha256sum nor shasum found in PATH" >&2
  exit 3
fi

# Build the file list using git pathspecs for stable, tracked files only.
mapfile -d '' files < <(git ls-files -z -- \
  ':(glob)**/*.js' \
  ':(glob)**/*.jsx' \
  ':(glob)**/*.ts' \
  ':(glob)**/*.tsx' \
  ':(exclude).github/**' \
  ':(exclude)**/node_modules/**' \
  ':(exclude)build/**' \
  ':(exclude)dist/**' \
  ':(exclude)out/**' \
  ':(exclude)coverage/**')

if (( ${#files[@]} == 0 )); then
  echo "0000000000000000000000000000000000000000000000000000000000000000"
  exit 0
fi

# Hash per-file contents, then hash that list for a stable combined digest.
printf '%s\0' "${files[@]}" | xargs -0 $HASH_CMD | $HASH_CMD | awk '{print $1}'

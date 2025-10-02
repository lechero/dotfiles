#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# update-codex.sh
# Install Codex CLI globally as `codex` (nightly by default).
#
# Channel order (default nightly): nightly -> prerelease/alpha -> stable
# OS/arch auto-detect; archive handling (tar.gz/zip/plain); brew conflict handling.
#
# Usage:
#   update-codex.sh            # nightly (default)
#   update-codex.sh stable
#
# Env overrides:
#   REPO="owner/name"                 # default: openai/codex
#   ASSET_NAME_HINT="codex"           # expected binary name inside archives
#   INSTALL_DIR="/desired/bin"        # directory to install into (must be writable)
#   INSTALL_PATH="/usr/local/bin/codex" # explicit full path (overrides INSTALL_DIR)
#   BREW_UNLINK=1                     # also run `brew unlink codex` if installed
#   GITHUB_TOKEN="..."                # optional, avoid GitHub API rate limits
# ==============================================================================

REPO="${REPO:-openai/codex}"
CHANNEL="${1:-nightly}"                 # nightly (default) | stable
ASSET_NAME_HINT="${ASSET_NAME_HINT:-codex}"

# ---------- OS/ARCH ----------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"        # darwin | linux
RAW_ARCH="$(uname -m | tr '[:upper:]' '[:lower:]')"  # x86_64|arm64|aarch64|...
case "$RAW_ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) ARCH="$RAW_ARCH" ;;
esac

# ---------- Install dir pick (prefer brew & system bins; skip pnpm/npm) ----------
default_install_dir() {
  if [[ -n "${INSTALL_DIR:-}" ]]; then
    echo "$INSTALL_DIR"; return
  fi
  if [[ "$OS" == "darwin" ]] && command -v brew >/dev/null 2>&1; then
    local bp
    bp="$(brew --prefix 2>/dev/null || true)"
    if [[ -n "$bp" && -d "$bp/bin" && -w "$bp/bin" ]]; then
      echo "$bp/bin"; return
    fi
  fi
  for d in /opt/homebrew/bin /usr/local/bin /usr/bin; do
    [[ -d "$d" && -w "$d" ]] && { echo "$d"; return; }
  done
  IFS=':' read -r -a PATH_DIRS <<< "$PATH"
  for d in "${PATH_DIRS[@]}"; do
    [[ ! -d "$d" || ! -w "$d" ]] && continue
    if echo "$d" | grep -qiE 'pnpm|npm|node|nvm|pyenv|asdf|cargo|\.local/bin'; then
      continue
    fi
    echo "$d"; return
  done
  echo "/usr/local/bin"
}

# If INSTALL_PATH set, respect it; else compute MANAGED_PATH in INSTALL_DIR
if [[ -n "${INSTALL_PATH:-}" ]]; then
  MANAGED_PATH="$INSTALL_PATH"
  INSTALL_DIR="$(dirname "$INSTALL_PATH")"
else
  INSTALL_DIR="$(default_install_dir)"
  MANAGED_BASENAME="codex-managed"
  MANAGED_PATH="$INSTALL_DIR/$MANAGED_BASENAME"
fi

LINK_CODEX="$INSTALL_DIR/codex"

# ---------- GitHub API ----------
gh_api() {
  local url="$1"
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    curl -sfL -H "Authorization: Bearer $GITHUB_TOKEN" -H "Accept: application/vnd.github+json" "$url"
  else
    curl -sfL -H "Accept: application/vnd.github+json" "$url"
  fi
}

# ---------- Matching helpers ----------
match_patterns() {
  if [[ "$OS" == "darwin" && "$ARCH" == "arm64" ]]; then
cat <<EOF
aarch64-apple-darwin
arm64-apple-darwin
darwin-arm64
apple-darwin
darwin
EOF
  elif [[ "$OS" == "darwin" && "$ARCH" == "amd64" ]]; then
cat <<EOF
x86_64-apple-darwin
amd64-apple-darwin
darwin-amd64
apple-darwin
darwin
EOF
  elif [[ "$OS" == "linux" && "$ARCH" == "arm64" ]]; then
cat <<EOF
aarch64-unknown-linux-gnu
linux-arm64
linux-aarch64
linux
EOF
  else
cat <<EOF
x86_64-unknown-linux-gnu
linux-amd64
linux
darwin
EOF
  fi
}

choose_best_asset() {
  local urls; urls="$(cat)"; [[ -z "$urls" ]] && return 1
  local best
  # Prefer OS/ARCH archives FIRST
  while read -r pat; do
    best="$(echo "$urls" | grep -Ei "$pat" | grep -Ei '\.(tar\.gz|tgz|zip)$' | head -n1 || true)"
    [[ -n "$best" ]] && { echo "$best"; return 0; }
  done < <(match_patterns)
  # Then OS/ARCH any
  while read -r pat; do
    best="$(echo "$urls" | grep -Ei "$pat" | head -n1 || true)"
    [[ -n "$best" ]] && { echo "$best"; return 0; }
  done < <(match_patterns)
  # Then plain 'codex' bootstrap
  best="$(echo "$urls" | grep -E '/releases/.*/codex$' | head -n1 || true)"
  [[ -n "$best" ]] && { echo "$best"; return 0; }
  # Fallback
  echo "$urls" | head -n1
}

find_asset_url() {
  local rel="$1"
  local json urls
  json="$(gh_api "https://api.github.com/repos/$REPO/$rel")" || return 1
  urls="$(echo "$json" | grep -Eo '"browser_download_url":\s*"[^"]+"' | cut -d'"' -f4)"
  echo "$urls" | choose_best_asset
}

find_prerelease_asset_url() {
  command -v jq >/dev/null 2>&1 || return 1
  local json urls
  json="$(gh_api "https://api.github.com/repos/$REPO/releases?per_page=30")" || return 1
  urls="$(echo "$json" | jq -r '.[] | select(.prerelease==true) | .assets[].browser_download_url')"
  [[ -z "$urls" ]] && return 1
  echo "$urls" | choose_best_asset
}

# ---------- Download & install ----------
download_and_install() {
  local url="$1"
  echo "→ Downloading: $url"
  local tmpdir file target_bin
  tmpdir="$(mktemp -d)"
  trap '[[ -n "${tmpdir:-}" ]] && rm -rf "$tmpdir"' EXIT

  file="$tmpdir/asset"
  curl -sfL "$url" -o "$file"

  mkdir -p "$tmpdir/unpack"
  case "$url" in
    *.tar.gz|*.tgz) tar -xzf "$file" -C "$tmpdir/unpack" ;;
    *.zip)          unzip -q "$file" -d "$tmpdir/unpack" ;;
    *)              cp "$file" "$tmpdir/unpack/$ASSET_NAME_HINT" ;;
  esac

  target_bin=""
  for guess in "$ASSET_NAME_HINT" "codex" "codex-cli"; do
    if found="$(find "$tmpdir/unpack" -type f -iname "$guess" -maxdepth 2 2>/dev/null | head -n1)"; then
      target_bin="$found"; break
    fi
  done
  if [[ -z "$target_bin" ]]; then
    target_bin="$(find "$tmpdir/unpack" -type f -maxdepth 2 -exec ls -l {} \; | awk '{print $5, $9}' | sort -nr | head -n1 | awk '{print $2}')"
  fi
  [[ -z "$target_bin" ]] && { echo "❌ No executable found in asset."; exit 1; }

  chmod +x "$target_bin" || true
  mkdir -p "$INSTALL_DIR"

  echo "→ Installing managed binary to: $MANAGED_PATH"
  if cp "$target_bin" "$MANAGED_PATH" 2>/dev/null; then :; else
    echo "⚠️  Need sudo to write to $INSTALL_DIR"
    echo "   sudo cp \"$target_bin\" \"$MANAGED_PATH\" && sudo chmod +x \"$MANAGED_PATH\""
    exit 1
  fi
  chmod +x "$MANAGED_PATH" || true

  # Create/refresh global symlink only for `codex`
  rm -f "$LINK_CODEX" 2>/dev/null || true
  ln -s "$MANAGED_PATH" "$LINK_CODEX" 2>/dev/null || {
    echo "ℹ️  Could not create $LINK_CODEX; try: sudo ln -sf \"$MANAGED_PATH\" \"$LINK_CODEX\""
  }
}

# ---------- Main ----------
case "$CHANNEL" in
  stable)  echo "Channel: stable" ;;
  nightly) echo "Channel: nightly (default)" ;;
  *) echo "Usage: $0 [stable|nightly]"; exit 1 ;;
esac

STABLE_REL="releases/latest"
NIGHTLY_REL="releases/tags/nightly"

asset_url=""
if [[ "$CHANNEL" == "nightly" ]]; then
  asset_url="$(find_asset_url "$NIGHTLY_REL" || true)"
  [[ -z "$asset_url" ]] && asset_url="$(find_prerelease_asset_url || true)"
  [[ -z "$asset_url" ]] && asset_url="$(find_asset_url "$STABLE_REL" || true)"
else
  asset_url="$(find_asset_url "$STABLE_REL" || true)"
fi
[[ -z "$asset_url" ]] && { echo "❌ No downloadable asset for $REPO."; exit 1; }

download_and_install "$asset_url"

hash -r 2>/dev/null || true

echo "→ Final check"
command -v codex >/dev/null 2>&1 && echo "codex -> $(command -v codex)" || echo "codex not on PATH"
echo "✅ Managed binary: $MANAGED_PATH"

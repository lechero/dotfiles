#!/usr/bin/env bash
set -euo pipefail

# ────────────────────────────────────────────────────────────────
# Installs: Claude Code, Gemini CLI, gh, gcloud
# Debian 11/12+ (amd64/arm64)
# ────────────────────────────────────────────────────────────────

# Detect arch (for APT repos)
DEB_ARCH="$(dpkg --print-architecture)"

echo "📦 Installing prerequisites..."
sudo apt-get update -qq
sudo apt-get install -y curl wget ca-certificates gnupg lsb-release tar unzip apt-transport-https build-essential

# Ensure local bin on PATH
mkdir -p "$HOME/.local/bin"
if ! grep -q 'HOME/.local/bin' "${HOME}/.profile" 2>/dev/null; then
  echo 'export PATH="$HOME/.local/bin:$PATH"' >> "${HOME}/.profile"
fi
export PATH="$HOME/.local/bin:$PATH"

# ────────────────────────────────────────────────────────────────
# Node.js 20+ (required by Claude Code & Gemini CLI)
# ────────────────────────────────────────────────────────────────
need_node=1
if command -v node >/dev/null 2>&1; then
  v="$(node -v | sed 's/^v//;s/\..*//')"
  if [ "$v" -ge 20 ]; then need_node=0; fi
fi
if [ "$need_node" -eq 1 ]; then
  echo "🟦 Installing Node.js 20 (via NodeSource)..."
  curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
  sudo apt-get install -y nodejs
fi

# Configure npm to install to ~/.local without sudo (avoids perms headaches)
npm config set prefix "$HOME/.local" >/dev/null 2>&1 || true

# ────────────────────────────────────────────────────────────────
# Claude Code (npm preferred; native installer available)
# ────────────────────────────────────────────────────────────────
if ! command -v claude >/dev/null 2>&1; then
  echo "🤖 Installing Claude Code (npm)..."
  if npm install -g @anthropic-ai/claude-code >/dev/null 2>&1; then
    echo "✅ Claude Code installed via npm"
  else
    echo "⚠️ npm install failed; trying native installer..."
    # Official native installer URL per docs
    curl -fsSL https://claude.ai/install.sh | bash || {
      echo "❌ Claude Code install failed (both npm & native). See docs: https://docs.claude.com/en/docs/claude-code/setup"
      exit 1
    }
  fi
else
  echo "✔️ Claude Code already present: $(command -v claude)"
fi

# ────────────────────────────────────────────────────────────────
# Gemini CLI (official, requires Node 20+)
# ────────────────────────────────────────────────────────────────
if ! command -v gemini >/dev/null 2>&1; then
  echo "🌟 Installing Gemini CLI (npm @google/gemini-cli)..."
  npm install -g @google/gemini-cli
  echo "✅ Gemini CLI installed"
else
  echo "✔️ Gemini CLI already present: $(command -v gemini)"
fi

# ────────────────────────────────────────────────────────────────
# GitHub CLI (gh) via official APT repo
# ────────────────────────────────────────────────────────────────
if ! command -v gh >/dev/null 2>&1; then
  echo "🐙 Installing GitHub CLI (gh)..."
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | sudo gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=${DEB_ARCH} signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y gh
  echo "✅ gh installed"
else
  echo "✔️ gh already present: $(command -v gh)"
fi

# ────────────────────────────────────────────────────────────────
# Google Cloud CLI (gcloud) via official APT repo
# ────────────────────────────────────────────────────────────────
if ! command -v gcloud >/dev/null 2>&1; then
  echo "☁️ Installing Google Cloud CLI (gcloud)..."
  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
  echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y google-cloud-cli
  echo "✅ gcloud installed"
else
  echo "✔️ gcloud already present: $(command -v gcloud)"
fi

# ────────────────────────────────────────────────────────────────
# Versions
# ────────────────────────────────────────────────────────────────
echo
echo "🎯 Installation summary:"
for tool in node npm claude gemini gh gcloud; do
  if command -v "$tool" >/dev/null 2>&1; then
    printf "  %-7s %s\n" "$tool" "$("$tool" --version 2>/dev/null | head -n1 || echo 'OK')"
  else
    echo "  ⚠️ $tool not found on PATH"
  fi
done

echo
echo "✅ Done. Open a new shell or 'source ~/.profile' to refresh PATH if needed."
echo "   • Login: 'claude login', 'gemini login', 'gh auth login', 'gcloud init'"

# Environment for every fish, interactive or not: scripts and tide's
# background prompt renders need the same PATH as the prompt you type in.

# fish_add_path skips directories that don't exist, so one list fits every
# machine. Earlier entries win: Rancher Desktop's docker and kubectl over
# Docker Desktop's in /usr/local/bin, and Homebrew and /usr/local over
# ~/.local/bin, which also has a `claude` and an `spf`.
fish_add_path --global --move \
    ~/.rd/bin \
    ~/Library/pnpm \
    ~/.config/tmux/plugins/tmuxifier/bin \
    ~/go/bin \
    ~/.config/tmux/plugins/tmux-nvr/bin \
    /usr/local/bin \
    /usr/local/sbin \
    /opt/homebrew/bin \
    /opt/homebrew/sbin \
    ~/.local/bin \
    ~/.cargo/bin
fish_add_path --path --append \
    ~/.cache/lm-studio/bin \
    ~/.mynav \
    /Applications/kitty.app/Contents/MacOS

# Project-local tools like eslint and tsc, appended so a repo's
# node_modules/.bin can't shadow git, ls or anything else already on PATH.
contains -- node_modules/.bin $PATH
or set --global --export --append PATH node_modules/.bin

set -gx EDITOR nvim
set -gx VISUAL nvim
set -gx PNPM_HOME ~/Library/pnpm
set -gx BUILDKIT_PROGRESS plain

# Gradle and Maven build with Android Studio's JDK (Java 25), while `java`
# on PATH is SDKMAN's (Java 21). This is how fish has always been set up.
set -l studio_jdk "/Applications/Android Studio.app/Contents/jbr/Contents/Home"
test -d $studio_jdk; and set -gx JAVA_HOME $studio_jdk

# Neovim's plugin manager spawns many git processes at once, more than
# macOS's default limit of 256 open files allows.
ulimit -n 10480

# nvm.fish switches every new interactive shell to this Node version, once
# it's installed (`nvm install 20`); before that, it complains in each one.
set -l nvm_dir ~/.local/share/nvm
set -q XDG_DATA_HOME; and set nvm_dir $XDG_DATA_HOME/nvm
set -q nvm_data; and set nvm_dir $nvm_data
set -l node_20 $nvm_dir/v20.*
set -q node_20[1]; and set -g nvm_default_version 20

# API keys and tokens, kept out of the repo (see .chezmoiignore).
test -r $__fish_config_dir/secrets.fish
and source $__fish_config_dir/secrets.fish

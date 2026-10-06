# Abbreviations expand as you type, so history keeps the real command.
# Commands that wrap another command or hold logic are in functions/ instead.
status is-interactive; or exit

# Navigation: `..` goes up one directory, `...` two, and so on
abbr -a dotdot --regex '^\.\.+$' --function multicd
abbr -a -- - 'cd -'
abbr -a cd- 'cd -'

# `!!` is the previous command, as in bash: `sudo !!`
abbr -a !! --position anywhere --function last_history_item

# git
abbr -a gd 'git diff'
abbr -a gl 'git pull'
abbr -a gp 'git push'
abbr -a gs 'git status'

# Search
abbr -a a rg
abbr -a g rg
abbr -a r 'rg --hidden -C 2'

# tmux
abbr -a tm tmux
abbr -a tl 'tmux list-sessions'
abbr -a ts tmux-sessionizer
abbr -a tw tmux-windowizer

# Tools
abbr -a c 'bat --paging=never --style=plain'
abbr -a cls clear
abbr -a cm chezmoi
abbr -a fns functions
abbr -a icat 'kitten icat'
abbr -a jl jless
abbr -a k kubectl
abbr -a lg lazygit
abbr -a lsq lazysql
abbr -a lz lazydocker
abbr -a mn mynav
abbr -a ng 'npm install -g'
abbr -a nu 'nvm use && clear'
abbr -a p python
abbr -a p3 python3
abbr -a pwc 'pwd | pbcopy'
abbr -a qq exit
abbr -a t tmuxifier
abbr -a tt task
abbr -a upv '~/.dotfiles/bash/update_neovim.sh'
command -q nvim; and abbr -a vim nvim

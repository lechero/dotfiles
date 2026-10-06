# A new terminal window goes straight into tmux: attach to the most recent
# session, or start one if there's none. Shells already in tmux, terminals
# inside editors and IDEs, and anything without a real terminal stay plain fish.
status is-interactive; or exit
set -q TMUX; and exit
test -t 0 -a -t 1; or exit
command -q tmux; or exit
fish_is_root_user; and exit
test -n "$INSIDE_EMACS$EMACS$VIM$NVIM$VSCODE_RESOLVING_ENVIRONMENT"; and exit
test "$TERM_PROGRAM" = vscode -o "$TERMINAL_EMULATOR" = JetBrains-JediTerm; and exit

if tmux has-session 2>/dev/null
    exec tmux attach
end
exec tmux new-session

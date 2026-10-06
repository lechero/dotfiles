function _cached_source --description "Source a command's output, cached until the command is upgraded"
    # Usage: _cached_source zoxide init fish
    # Running these init commands at every shell start costs a few
    # milliseconds each; sourcing the saved output is nearly free.
    set -l bin (command -s $argv[1]); or return
    set -q XDG_CACHE_HOME; or set -l XDG_CACHE_HOME ~/.cache
    set -l cache $XDG_CACHE_HOME/fish/cached_source/(string join _ -- $argv | string replace -a / _).fish

    if not test -s $cache; or test (path mtime -- $bin) -gt (path mtime -- $cache)
        mkdir -p (path dirname -- $cache)
        # Write to a temporary file first: tmux can start several shells at once.
        $argv >$cache.$fish_pid
        and mv -f $cache.$fish_pid $cache
        or begin
            rm -f $cache.$fish_pid
            return 1
        end
    end

    source $cache
end

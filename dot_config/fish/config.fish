# Interactive setup. PATH and environment variables are in conf.d/00-env.fish,
# plugin loading in conf.d/10-plugins.fish, abbreviations in conf.d/abbr.fish
# and colors in conf.d/colors.fish.

if status is-interactive
    set -g fish_greeting
    set -g fish_key_bindings fish_vi_key_bindings
    set -g fish_cursor_default block

    # The prompt is tide unless `prompt-switch` picked starship. tide needs no
    # setup here: its fish_prompt autoloads from plugins/.
    if test "$prompt_engine" = starship; and command -q starship
        _cached_source starship init fish --print-full-init
        enable_transience
    end

    # docker and lazydocker use the engine `dock --default` saved, unless this
    # shell inherited a DOCKER_HOST from the one that started it. Only here,
    # not in conf.d: tide renders the prompt in non-interactive shells, which
    # must show this shell's engine, not the default.
    if set -q dock_default; and not set -q DOCKER_HOST
        set -gx DOCKER_HOST $dock_default
    end

    _cached_source zoxide init fish
    _cached_source jump shell fish

    # fzf.fish: ctrl-f finds files and ctrl-alt-v variables; the other
    # searches keep their default keys.
    fzf_configure_bindings --directory=ctrl-f --variables=ctrl-alt-v

    # Switch Node to the project's version on entering a directory with an .nvmrc.
    function _nvm_use_nvmrc --on-variable PWD
        status is-command-substitution; and return
        test -r .nvmrc; and nvm use
    end
end

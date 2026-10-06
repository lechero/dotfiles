function prompt-switch --argument-names engine --description 'Switch the prompt between tide and starship'
    # With no argument, switch to whichever prompt isn't active.
    if test -z "$engine"
        test "$prompt_engine" = starship; and set engine tide; or set engine starship
    end

    if not contains -- $engine tide starship
        echo "prompt-switch: unknown prompt '$engine', expected tide or starship" >&2
        return 1
    end
    if test $engine = starship; and not command -q starship
        echo 'prompt-switch: starship is not installed (brew install starship)' >&2
        return 1
    end

    # Universal, so the choice sticks for every new shell on this machine.
    set --universal prompt_engine $engine

    # Both prompts install event handlers and key bindings, so a fresh shell
    # is the clean way to swap. Don't throw away running jobs to get one.
    if jobs -q
        echo "prompt-switch: $engine is the prompt in new shells. This one has jobs running; run `exec fish` when they're done."
        return
    end
    exec fish
end

function fish_user_key_bindings
    # Vi mode is set in config.fish. Bind in both normal and insert mode.
    for mode in default insert
        # Search history for the word under the cursor. shift-up/down are for
        # terminals that don't pass ctrl-up/down through, like Emacs's vterm.
        bind -M $mode ctrl-up history-token-search-backward
        bind -M $mode ctrl-down history-token-search-forward
        bind -M $mode shift-up history-token-search-backward
        bind -M $mode shift-down history-token-search-forward

        bind -M $mode ctrl-t transpose-chars
        # Accept the autosuggestion and run it
        bind -M $mode ctrl-s accept-autosuggestion execute
    end
end

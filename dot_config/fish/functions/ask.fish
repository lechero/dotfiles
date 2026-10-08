function ask --description 'Ask Copilot for a shell command'
    argparse --name=ask c/commandline h/help -- $argv; or return

    if set -q _flag_help
        echo 'Usage: ask QUESTION...
       ask --commandline

Asks Copilot for one fish command that does what QUESTION describes, and
prints it. alt-a does the same with the question typed at the prompt, and
puts the command there in its place: check it, then run it with enter.

Copilot answers with gpt-4.1 unless ask_model names another model.'
        return
    end

    # alt-a: the question is on the command line, and the command replaces it.
    if set -q _flag_commandline
        set -l question (commandline | string join ' ' | string trim)
        test -n "$question"; or return
        echo
        set_color brblack
        echo 'Asking Copilot...'
        set_color normal
        set -l suggestion (ask -- $question)
        and commandline --replace -- (string join \n -- $suggestion)
        commandline -f repaint
        return
    end

    set -l question (string join ' ' -- $argv)
    if test -z "$question"
        echo 'ask: what should the command do? (ask --help)' >&2
        return 2
    end
    set -q ask_model; or set -l ask_model gpt-4.1

    # No tools, so a question can't make it read or run anything. Leaving one
    # harmless tool available is how to switch the rest off: an empty list, or
    # --deny-tool, still lets it read files and run read-only shell commands.
    set -l options -s --model $ask_model --no-ask-user --no-custom-instructions \
        --available-tools=fetch_copilot_cli_documentation --disable-builtin-mcps --log-level none
    # Nor does it need the MCP servers you've added, which it would connect to.
    command -q jq
    and for server in (jq -r '.mcpServers // {} | keys[]' ~/.copilot/mcp-config.json 2>/dev/null)
        set -a options --disable-mcp-server $server
    end

    # Copilot tells the model which files are in the folder it runs in, so it
    # runs in an empty one. Through sh, as a cd here would move this shell.
    set -l empty (mktemp -d -t ask)
    set -l prompt "Reply with only one shell command for fish $version on macOS (BSD tools, Homebrew), with no explanation and no code fences. If it takes several steps, chain them on one line. The task: $question"
    set -l answer (sh -c 'cd "$1" && shift && exec "$@"' sh $empty copilot -p $prompt $options 2>&1)
    set -l result $status
    command rm -rf $empty

    if test $result -ne 0
        printf '%s\n' $answer >&2
        return $result
    end
    # Models sometimes wrap the command in a code fence or a $ prompt anyway.
    set -l suggestion (string match -rv '^\s*```' -- $answer | string replace -r '^\s*\$\s+' '' | string match -rv '^\s*$')
    if test -z "$suggestion"
        echo 'ask: Copilot answered without a command' >&2
        return 1
    end
    printf '%s\n' $suggestion
end

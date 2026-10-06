function y --description 'Ask GitHub Copilot for a shell command'
    if not set -q argv[1]
        echo 'Usage: y <what you want to do>' >&2
        return 1
    end
    copilot -p "@terminal Give me a shell command only, with no explanation: $argv"
end

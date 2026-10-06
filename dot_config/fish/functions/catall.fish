function catall --description 'Print every file below the current directory, each after its path'
    find . -type f -exec sh -c 'echo "$1"; cat "$1"' _ {} \;
end

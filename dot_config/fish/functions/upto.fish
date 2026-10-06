function upto --argument-names name --description 'cd up to the parent directory with this name'
    set -l dirs (string split / -- $PWD)
    set -l index (contains --index -- $name $dirs); or return 1
    cd (string join / -- $dirs[1..$index])
end

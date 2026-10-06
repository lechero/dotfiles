function mkdir-cd --argument-names dir --description 'Make a directory, with its parents, and cd into it'
    mkdir -p -- $dir; and cd -- $dir
end

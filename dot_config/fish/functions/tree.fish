function tree --wraps tree --description 'tree without the file and directory count at the end'
    command tree --noreport $argv
end

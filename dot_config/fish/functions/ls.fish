function ls --wraps ls --description 'ls in color, with a / after directory names'
    command ls -p -G $argv
end

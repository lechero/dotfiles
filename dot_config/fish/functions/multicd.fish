function multicd --description 'Expand .. to cd ../, ... to cd ../../ and so on'
    echo cd (string repeat -n (math (string length -- $argv[1]) - 1) ../)
end

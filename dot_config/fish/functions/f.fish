function f --wraps fzf --description 'fzf with a bat preview of the highlighted file'
    fzf --preview 'bat --color=always {}' $argv
end

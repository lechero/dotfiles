function h --description 'Graph of the last 50 commits on all branches'
    git log --color --decorate --graph --all --max-count=50 \
        --pretty=format:'%C(auto)%h %Cgreen(%ar)%Creset %s %C(yellow)%d %C(cyan)%an' $argv
end

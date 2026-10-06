function l --wraps eza --description 'Long listing with git status and icons, including hidden files'
    eza -lh --git --all --icons=auto $argv
end

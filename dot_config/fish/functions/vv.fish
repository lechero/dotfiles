function vv --description 'Pick files with fzf and open them in Neovim'
    # nvim gets the file names as arguments so it keeps the terminal as its input.
    set -l files (fd --type f --hidden --exclude .git | fzf-tmux -p --multi)
    and nvim $files
end

function v --wraps nvim --description 'Neovim, ignoring a VIMRUNTIME inherited from an outer Neovim'
    env -u VIMRUNTIME nvim $argv
end

function lv --wraps nvim --description 'Neovim with the LazyVim config'
    env -u NVIM_LISTEN_ADDRESS NVIM_APPNAME=LazyVim nvim $argv
end

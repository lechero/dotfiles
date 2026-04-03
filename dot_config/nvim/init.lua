local config_dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h')
vim.opt.runtimepath:prepend(config_dir)

vim.g.mapleader = ' '
vim.g.maplocalleader = ' '

vim.g.have_nerd_font = true

require('fuentastic.config.options')
require('fuentastic.config.keymaps')
require('fuentastic.config.autocmds')
require('fuentastic.config.winmove')
require('fuentastic.config.lazy')

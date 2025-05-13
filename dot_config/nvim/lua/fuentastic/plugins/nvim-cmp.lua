-- from wookayin’s Reddit reply
local has_words_before = function()
	if vim.bo[0].buftype == "prompt" then
		return false
	end
	-- support Lua 5.1 and 5.2+
	local unpack = unpack or table.unpack
	local line, col = unpack(vim.api.nvim_win_get_cursor(0))
	return col ~= 0 and vim.api.nvim_buf_get_lines(0, line - 1, line, true)[1]:sub(col, col):match("%s") == nil
end

return {
	"hrsh7th/nvim-cmp",
	event = "InsertEnter",
	dependencies = {
		{
			"L3MON4D3/LuaSnip",
			build = (function()
				if vim.fn.has("win32") == 1 or vim.fn.executable("make") == 0 then
					return
				end
				return "make install_jsregexp"
			end)(),
			dependencies = {
				-- `friendly-snippets` contains a variety of premade snippets.
				--    See the README about individual language/framework/plugin snippets:
				--    https://github.com/rafamadriz/friendly-snippets
				-- {
				--   'rafamadriz/friendly-snippets',
				--   config = function()
				--     require('luasnip.loaders.from_vscode').lazy_load()
				--   end,
				-- },
			},
		},
		"saadparwaiz1/cmp_luasnip",

		-- Adds other completion capabilities.
		--  nvim-cmp does not ship with all sources by default. They are split
		--  into multiple repos for maintenance purposes.
		"hrsh7th/cmp-nvim-lsp",
		"hrsh7th/cmp-path",
	},
	config = function()
		-- See `:help cmp`
		local cmp = require("cmp")
		local luasnip = require("luasnip")
		luasnip.config.setup({})

		cmp.setup({
			snippet = {
				expand = function(args)
					luasnip.lsp_expand(args.body)
				end,
			},
			completion = { completeopt = "menu,menuone,noinsert" },

			mapping = {
				["<C-Space>"] = require("cmp").mapping.complete(), -- manually open completion menu
				["<C-n>"] = require("cmp").mapping.select_next_item({ behavior = require("cmp").SelectBehavior.Insert }),
				["<C-p>"] = require("cmp").mapping.select_prev_item({ behavior = require("cmp").SelectBehavior.Insert }),
				["<CR>"] = require("cmp").mapping.confirm({ select = false }), -- confirm selection with Enter
				["<Tab>"] = require("cmp").mapping(function(fallback)
					local cmp = require("cmp")
					local copilot_suggestion = require("copilot.suggestion")
					if copilot_suggestion.is_visible() then
						-- If a Copilot inline suggestion is showing, accept it
						copilot_suggestion.accept()
					elseif cmp.visible() and has_words_before() then
						-- If completion menu is open and cursor is after a character, select next item [oai_citation:14‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=mapping%20%3D%20%7B%20%5B,end%20end%29%2C)
						cmp.select_next_item({ behavior = cmp.SelectBehavior.Select })
					elseif require("luasnip").expand_or_jumpable() then
						-- If inside a snippet, jump to next snippet placeholder
						require("luasnip").expand_or_jump()
					else
						fallback() -- otherwise, insert a literal tab (indent)
					end
				end, { "i", "s" }),
				["<S-Tab>"] = require("cmp").mapping(function(fallback) -- Shift-Tab for snippet jump backwards
					if require("luasnip").jumpable(-1) then
						require("luasnip").jump(-1)
					else
						fallback()
					end
				end, { "i", "s" }),
			},
			sources = {
				{ name = "copilot" }, -- GitHub Copilot source (AI suggestions)
				{ name = "nvim_lsp" }, -- LSP completions
				{ name = "luasnip" }, -- snippet completions
				{ name = "buffer" },
				{ name = "path" },
				{ name = "lazydev" }, -- Lua dev completions (if editing Neovim config)
			},
			sorting = {
				priority_weight = 2, -- boost the effect of source priority [oai_citation:11‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
				comparators = {
					require("copilot_cmp.comparators").prioritize, -- custom: prioritize Copilot [oai_citation:12‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
					require("cmp.config.compare").exact, -- ensure exact LSP matches aren’t overshadowed [oai_citation:13‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
					-- … (include other default comparators)
					require("cmp.config.compare").score,
					require("cmp.config.compare").recently_used,
					require("cmp.config.compare").locality,
					require("cmp.config.compare").kind,
					require("cmp.config.compare").sort_text,
					require("cmp.config.compare").length,
					require("cmp.config.compare").order,
				},
			},
		})

		cmp.setup.cmdline("/", {
			mapping = cmp.mapping.preset.cmdline(),
			sources = {
				{ name = "buffer" },
			},
		})

		cmp.setup.cmdline(":", {
			mapping = cmp.mapping.preset.cmdline(),
			sources = cmp.config.sources({
				{ name = "path" },
			}, {
				{ name = "cmdline", option = {
					ignore_cmds = { "Man", "!" },
				} },
			}),
		})
	end,
}

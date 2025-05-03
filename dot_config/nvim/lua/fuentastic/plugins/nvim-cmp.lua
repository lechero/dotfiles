-- lua/fuentastic/plugins/nvim-cmp.lua
return {
	"hrsh7th/nvim-cmp",
	event = "InsertEnter",
	dependencies = {
		"L3MON4D3/LuaSnip",
		"saadparwaiz1/cmp_luasnip",
		"hrsh7th/cmp-nvim-lsp",
		"hrsh7th/cmp-path",
	},
	config = function()
		local cmp = require("cmp")
		local luasnip = require("luasnip")

		-- Helper to know if there's text before the cursor
		local has_words_before = function()
			local row, col = unpack(vim.api.nvim_win_get_cursor(0))
			return col ~= 0 and vim.api.nvim_get_current_line():sub(col, col):match("%s") == nil
		end

		luasnip.config.setup({})

		cmp.setup({
			snippet = {
				expand = function(args)
					luasnip.lsp_expand(args.body)
				end,
			},

			-- show full snippet in a bordered floating window
			window = {
				documentation = cmp.config.window.bordered(),
			},

			mapping = {
				["<C-Space>"] = cmp.mapping.complete(),
				["<C-n>"] = cmp.mapping.select_next_item({ behavior = cmp.SelectBehavior.Insert }),
				["<C-p>"] = cmp.mapping.select_prev_item({ behavior = cmp.SelectBehavior.Insert }),
				["<CR>"] = cmp.mapping.confirm({ select = false }),

				["<Tab>"] = cmp.mapping(function(fallback)
					local copilot = require("copilot.suggestion")
					if copilot.is_visible() then
						copilot.accept()
					elseif cmp.visible() then
						cmp.select_next_item({ behavior = cmp.SelectBehavior.Select })
					elseif has_words_before() then
						cmp.complete()
					else
						fallback()
					end
				end, { "i", "s" }),

				["<S-Tab>"] = cmp.mapping(function(fallback)
					if cmp.visible() then
						cmp.select_prev_item({ behavior = cmp.SelectBehavior.Select })
					elseif luasnip.jumpable(-1) then
						luasnip.jump(-1)
					else
						fallback()
					end
				end, { "i", "s" }),
			},

			sources = {
				{ name = "copilot", group_index = 2 },
				-- Other Sources
				{ name = "nvim_lsp", group_index = 2 },
				{ name = "path", group_index = 2 },
				{ name = "luasnip", group_index = 2 },
				{ name = "buffer", group_index = 2 },
				{ name = "lazydev", group_index = 2 },
				-- { name = "copilot" },
				-- { name = "nvim_lsp" },
				-- { name = "luasnip" },
				-- { name = "path" },
				-- { name = "buffer" },
				-- { name = "lazydev" },
			},

			-- sorting = {
			-- 	priority_weight = 2,
			-- 	comparators = {
			-- 		require("copilot_cmp.comparators").prioritize,
			-- 		cmp.config.compare.exact,
			-- 		cmp.config.compare.score,
			-- 		cmp.config.compare.recently_used,
			-- 		cmp.config.compare.locality,
			-- 		cmp.config.compare.kind,
			-- 		cmp.config.compare.sort_text,
			-- 		cmp.config.compare.length,
			-- 		cmp.config.compare.order,
			-- 	},
			-- },
		})

		-- Optional: border for cmdline completions
		cmp.setup.cmdline("/", {
			mapping = cmp.mapping.preset.cmdline(),
			sources = { { name = "buffer" } },
		})
		cmp.setup.cmdline(":", {
			mapping = cmp.mapping.preset.cmdline(),
			sources = cmp.config.sources({ { name = "path" } }, { { name = "cmdline" } }),
		})
	end,
}
-- return {
-- 	"hrsh7th/nvim-cmp",
-- 	event = "InsertEnter",
-- 	dependencies = {
-- 		{
-- 			"L3MON4D3/LuaSnip",
-- 			build = (function()
-- 				if vim.fn.has("win32") == 1 or vim.fn.executable("make") == 0 then
-- 					return
-- 				end
-- 				return "make install_jsregexp"
-- 			end)(),
-- 			dependencies = {
-- 				-- `friendly-snippets` contains a variety of premade snippets.
-- 				--    See the README about individual language/framework/plugin snippets:
-- 				--    https://github.com/rafamadriz/friendly-snippets
-- 				-- {
-- 				--   'rafamadriz/friendly-snippets',
-- 				--   config = function()
-- 				--     require('luasnip.loaders.from_vscode').lazy_load()
-- 				--   end,
-- 				-- },
-- 			},
-- 		},
-- 		"saadparwaiz1/cmp_luasnip",
--
-- 		-- Adds other completion capabilities.
-- 		--  nvim-cmp does not ship with all sources by default. They are split
-- 		--  into multiple repos for maintenance purposes.
-- 		"hrsh7th/cmp-nvim-lsp",
-- 		"hrsh7th/cmp-path",
-- 	},
-- 	config = function()
-- 		-- See `:help cmp`
-- 		local cmp = require("cmp")
-- 		local luasnip = require("luasnip")
--
-- 		local has_words_before = function()
-- 			local row, col = unpack(vim.api.nvim_win_get_cursor(0))
-- 			return col ~= 0 and vim.api.nvim_get_current_line():sub(col, col):match("%s") == nil
-- 		end
--
-- 		luasnip.config.setup({})
--
-- 		cmp.setup({
-- 			snippet = {
-- 				expand = function(args)
-- 					luasnip.lsp_expand(args.body)
-- 				end,
-- 			},
-- 			completion = { completeopt = "menu,menuone,noinsert" },
--
-- 			window = {
-- 				documentation = cmp.config.window.bordered(), -- show full snippet on hover
-- 			},
-- 			mapping = {
-- 				["<C-Space>"] = require("cmp").mapping.complete(), -- manually open completion menu
-- 				["<C-n>"] = require("cmp").mapping.select_next_item({ behavior = require("cmp").SelectBehavior.Insert }),
-- 				["<C-p>"] = require("cmp").mapping.select_prev_item({ behavior = require("cmp").SelectBehavior.Insert }),
-- 				["<CR>"] = require("cmp").mapping.confirm({ select = false }), -- confirm selection with Enter
-- 				["<Tab>"] = cmp.mapping(function(fallback)
-- 					local copilot = require("copilot.suggestion")
-- 					if copilot.is_visible() then
-- 						copilot.accept()
-- 					elseif cmp.visible() then
-- 						cmp.select_next_item({ behavior = cmp.SelectBehavior.Select })
-- 					elseif has_words_before() then
-- 						cmp.complete()
-- 					else
-- 						fallback()
-- 					end
-- 				end, { "i", "s" }),
-- 				-- ["<Tab>"] = require("cmp").mapping(function(fallback)
-- 				-- 	local cmp = require("cmp")
-- 				-- 	local copilot_suggestion = require("copilot.suggestion")
-- 				-- 	if copilot_suggestion.is_visible() then
-- 				-- 		-- If a Copilot inline suggestion is showing, accept it
-- 				-- 		copilot_suggestion.accept()
-- 				-- 	elseif cmp.visible() and require("cmp.utils.misc").has_words_before() then
-- 				-- 		-- If completion menu is open and cursor is after a character, select next item [oai_citation:14‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=mapping%20%3D%20%7B%20%5B,end%20end%29%2C)
-- 				-- 		cmp.select_next_item({ behavior = cmp.SelectBehavior.Select })
-- 				-- 	elseif require("luasnip").expand_or_jumpable() then
-- 				-- 		-- If inside a snippet, jump to next snippet placeholder
-- 				-- 		require("luasnip").expand_or_jump()
-- 				-- 	else
-- 				-- 		fallback() -- otherwise, insert a literal tab (indent)
-- 				-- 	end
-- 				-- end, { "i", "s" }),
-- 				["<S-Tab>"] = require("cmp").mapping(function(fallback) -- Shift-Tab for snippet jump backwards
-- 					if require("luasnip").jumpable(-1) then
-- 						require("luasnip").jump(-1)
-- 					else
-- 						fallback()
-- 					end
-- 				end, { "i", "s" }),
-- 			},
-- 			sources = {
-- 				{ name = "copilot" }, -- GitHub Copilot source (AI suggestions)
-- 				{ name = "nvim_lsp" }, -- LSP completions
-- 				{ name = "luasnip" }, -- snippet completions
-- 				{ name = "buffer" },
-- 				{ name = "path" },
-- 				{ name = "lazydev" }, -- Lua dev completions (if editing Neovim config)
-- 			},
-- 			sorting = {
-- 				priority_weight = 2, -- boost the effect of source priority [oai_citation:11‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
-- 				comparators = {
-- 					require("copilot_cmp.comparators").prioritize, -- custom: prioritize Copilot [oai_citation:12‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
-- 					require("cmp.config.compare").exact, -- ensure exact LSP matches aren’t overshadowed [oai_citation:13‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=One%20custom%20comparitor%20for%20sorting,stuck%20below%20poor%20copilot%20matches)
-- 					-- … (include other default comparators)
-- 					require("cmp.config.compare").score,
-- 					require("cmp.config.compare").recently_used,
-- 					require("cmp.config.compare").locality,
-- 					require("cmp.config.compare").kind,
-- 					require("cmp.config.compare").sort_text,
-- 					require("cmp.config.compare").length,
-- 					require("cmp.config.compare").order,
-- 				},
-- 			},
-- 		})
--
-- 		cmp.setup.cmdline("/", {
-- 			mapping = cmp.mapping.preset.cmdline(),
-- 			sources = {
-- 				{ name = "buffer" },
-- 			},
-- 		})
--
-- 		cmp.setup.cmdline(":", {
-- 			mapping = cmp.mapping.preset.cmdline(),
-- 			sources = cmp.config.sources({
-- 				{ name = "path" },
-- 			}, {
-- 				{ name = "cmdline", option = {
-- 					ignore_cmds = { "Man", "!" },
-- 				} },
-- 			}),
-- 		})
-- 	end,
-- }

return {
	"zbirenbaum/copilot.lua",
	event = "InsertEnter",
	cmd = "Copilot",
	build = ":Copilot auth",
	config = function()
		require("copilot").setup({
			suggestion = { enabled = false },
			panel = { enabled = false },
			server_opts_overrides = {
				trace = "verbose",
				settings = {
					advanced = {
						listCount = 10, -- #completions for panel
						inlineSuggestCount = 3, -- #completions for getCompletions
					},
				},
			},
		})
		-- require("copilot").setup({
		-- 	-- keep the engine on, but hide all of its built-in UI
		-- 	suggestion = {
		-- 		enabled = true, -- ← engine stays running!
		-- 		auto_trigger = false, -- let cmp trigger
		-- 		hide_during_completion = true, -- don’t show ghost text when cmp is open
		-- 	},
		-- 	panel = { enabled = false }, -- turn off the panel
		--
		-- 	server_opts_overrides = {
		-- 		settings = {
		-- 			advanced = {
		-- 				inlineSuggestCount = 3, -- ask for 3
		-- 				listCount = 3, -- same for panel (if ever re-enabled)
		-- 			},
		-- 		},
		-- 	},
		--
		-- 	filetypes = {
		-- 		markdown = true,
		-- 		yaml = false,
		-- 		-- …etc
		-- 	},
		-- })
	end,
}

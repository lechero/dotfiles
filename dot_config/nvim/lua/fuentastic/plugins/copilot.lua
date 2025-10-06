return {
	"zbirenbaum/copilot.lua",
	event = "InsertEnter", -- ensure it loads when you start typing
	build = ":Copilot auth",
	config = function()
		require("copilot").setup({
			suggestion = { enabled = true, auto_trigger = true },
			panel = { enabled = true },
			copilot_node_command = "node", -- or absolute path if needed
			filetypes = {
				markdown = true,
				gitcommit = true,
				["*"] = true,
			},
		})
	end,
}

return {
	"sindrets/diffview.nvim",
	dependencies = { "nvim-lua/plenary.nvim" },
	cmd = {
		"DiffviewOpen",
		"DiffviewClose",
		"DiffviewToggleFiles",
		"DiffviewFocusFiles",
		"DiffviewFileHistory",
	},
	keys = {
		{
			"<leader>gdo",
			function()
				require("diffview").open()
			end,
			desc = "Diffview: Open repo changes",
		},
		{
			"<leader>gdc",
			"<cmd>DiffviewClose<CR>",
			desc = "Diffview: Close view",
		},
		{
			"<leader>gdf",
			function()
				require("diffview").file_history("%")
			end,
			desc = "Diffview: Current file history",
		},
		{
			"<leader>gdp",
			"<cmd>DiffviewToggleFiles<CR>",
			desc = "Diffview: Toggle file panel",
		},
	},
	opts = {
		enhanced_diff_hl = true,
		view = {
			default = {
				winbar_info = true,
			},
		},
		file_panel = {
			win_config = { width = 35 },
		},
	},
}

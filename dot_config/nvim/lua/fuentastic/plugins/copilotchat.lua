-- lua/plugins/copilotchat.lua
return {
	{
		"CopilotC-Nvim/CopilotChat.nvim",
		dependencies = {
			"github/copilot.vim", -- or 'zbirenbaum/copilot.lua'
			"nvim-lua/plenary.nvim", -- for async, curl, logging, etc.
		},
		build = "make tiktoken", -- only on macOS/Linux if you want tiktoken support
		opts = {
			-- === Window & UI ===
			window = {
				layout = "float", -- put it in a floating window
				relative = "cursor", -- float near the cursor
				width = 0.75, -- 75% of editor width
				height = 0.4, -- 40% of editor height
				border = "rounded",
			},
			show_help = true, -- virtual-text help
			highlight_selection = true,

			-- === Key mappings inside the chat ===
			mappings = {
				complete = { insert = "<Tab>" },
				close = { normal = "q", insert = "<C-c>" },
				reset = { normal = "<C-l>", insert = "<C-l>" },
				submit_prompt = { normal = "<CR>", insert = "<C-s>" },
			},

			-- === Prompts & Sticky Prompts ===
			prompts = {
				Explain = {
					prompt = "Write an explanation for the selected code.",
					system_prompt = "COPILOT_EXPLAIN",
				},
				Review = {
					prompt = "Review the selected code for potential issues.",
					system_prompt = "COPILOT_REVIEW",
				},
			},
			sticky = {
				"@models Using gpt-4o", -- lock in GPT-4o
				"#files", -- always include file listing context
			},

			-- === Model / Agent ===
			model = "gpt-4o", -- default AI model
			agent = "copilot", -- uncomment to use a custom agent

			-- === Optional Providers & Contexts ===
			-- (you can remove or extend these as you like)
			providers = {
				copilot = {},
				github_models = {},
				copilot_embeddings = {},
			},
			contexts = {
				buffer = {},
				git = {},
				url = {},
			},
		},
		config = function(_, opts)
			require("CopilotChat").setup(opts)
			-- integrate with Telescope for all the :CopilotChatPrompts, -Models, etc.
			require("telescope").load_extension("ui-select")
		end,
	},
}

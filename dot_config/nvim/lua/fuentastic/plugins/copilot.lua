return {
	-- 1. Copilot.lua plugin (GitHub Copilot client)
	"zbirenbaum/copilot.lua",
	event = "InsertEnter", -- delay loading until Insert mode for efficiency [oai_citation:1‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=Because%20the%20copilot%20server%20takes,For%20example)
	cmd = "Copilot", -- also allow manual `:Copilot` commands
	build = ":Copilot auth", -- (optional) automatically authenticate on first install
	opts = {
		suggestion = {
			enabled = true,
			auto_trigger = true, -- auto-show suggestions as you type [oai_citation:2‡lazyvim.github.io](https://lazyvim.github.io/extras/ai/copilot#:~:text=opts%20%3D%20,)
			debounce = 75,
			keymap = {
				accept = "<Tab>", -- use <Tab> to accept suggestion [oai_citation:3‡gist.github.com](https://gist.github.com/haandol/c2ff89706fa8af6edf1bb2bb0c1fe3ba#:~:text=suggestion%20%3D%20,)
				accept_word = false,
				accept_line = false,
				next = "<M-]>",
				prev = "<M-[>",
				dismiss = "<C-]>",
			},
			-- hide suggestion text when completion menu is open (to avoid overlap)
			hide_during_completion = true,
		},
		panel = { enabled = false }, -- disable Copilot panel (not used in this setup)
		filetypes = {
			markdown = true, -- enable in Markdown (disabled by default) [oai_citation:4‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=require%28,env%20files%20return%20false)
			yaml = false, -- (example) keep disabled for YAML files
			-- add any other filetypes overrides if needed; by default most languages are enabled
		},
		-- Optionally, set NodeJS path if your system Node is < 20:
		-- copilot_node_command = "/path/to/node>=20"  [oai_citation:5‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=Use%20this%20field%20to%20provide,must%20be%2020%20or%20newer)
	},
}
-- return {
-- 	"zbirenbaum/copilot.lua",
-- 	event = "InsertEnter",
-- 	opts = {
-- 		panel = { enabled = true },
-- 		suggestion = {
-- 			enabled = true,
-- 			auto_trigger = true,
-- 			keymap = {
-- 				accept = "<Tab>",
-- 				next = "<C-]>",
-- 				prev = "<S-Tab>",
-- 				dismiss = "<C-e>",
-- 			},
-- 			-- keymap = {
-- 			-- 	accept = "<C-l>", -- accept suggestion
-- 			-- 	next = "<C-]>", -- next suggestion
-- 			-- 	prev = "<C-[>", -- previous suggestion
-- 			-- 	dismiss = "<C-_>", -- dismiss suggestion
-- 			-- },
-- 		},
-- 		filetypes = { ["*"] = true },
-- 	},
-- }

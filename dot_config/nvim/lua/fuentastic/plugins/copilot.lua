-- lua/fuentastic/plugins/copilot.lua
return {
	"zbirenbaum/copilot.lua",
	event = "InsertEnter",
	cmd = "Copilot",
	build = ":Copilot auth",
	opts = {
		-- disable Copilot.lua's ghost text and panel (they'd conflict with cmp)
		suggestion = { enabled = false }, --  [oai_citation:0‡GitHub](https://github.com/zbirenbaum/copilot-cmp)
		panel = { enabled = false },

		-- ask the LSP to return up to 3 inline suggestions at once
		server_opts_overrides = {
			settings = {
				advanced = {
					inlineSuggestCount = 3, --  [oai_citation:1‡GitHub](https://github.com/zbirenbaum/copilot.lua)
				},
			},
		},

		-- your existing filetype overrides…
		filetypes = {
			markdown = true,
			yaml = true,
			-- etc…
		},
	},
}

-- return {
-- 	-- 1. Copilot.lua plugin (GitHub Copilot client)
-- 	"zbirenbaum/copilot.lua",
-- 	event = "InsertEnter", -- delay loading until Insert mode for efficiency [oai_citation:1‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=Because%20the%20copilot%20server%20takes,For%20example)
-- 	cmd = "Copilot", -- also allow manual `:Copilot` commands
-- 	build = ":Copilot auth", -- (optional) automatically authenticate on first install
-- 	opts = {
-- 		suggestion = {
-- 			enabled = true,
-- 			auto_trigger = true, -- auto-show suggestions as you type [oai_citation:2‡lazyvim.github.io](https://lazyvim.github.io/extras/ai/copilot#:~:text=opts%20%3D%20,)
-- 			debounce = 75,
-- 			keymap = {
-- 				accept = "<Tab>", -- use <Tab> to accept suggestion [oai_citation:3‡gist.github.com](https://gist.github.com/haandol/c2ff89706fa8af6edf1bb2bb0c1fe3ba#:~:text=suggestion%20%3D%20,)
-- 				accept_word = false,
-- 				accept_line = false,
-- 				next = "<M-]>",
-- 				prev = "<M-[>",
-- 				dismiss = "<C-]>",
-- 			},
-- 			-- hide suggestion text when completion menu is open (to avoid overlap)
-- 			hide_during_completion = true,
-- 		},
-- 		server_opts_overrides = {
-- 			settings = {
-- 				advanced = {
-- 					inlineSuggestCount = 3, -- e.g. get up to 3 suggestions  [oai_citation:1‡GitHub](https://github.com/zbirenbaum/copilot.lua)
-- 				},
-- 			},
-- 		},
-- 		panel = { enabled = false, auto_refresh = true }, -- disable Copilot panel (not used in this setup)
-- 		filetypes = {
-- 			markdown = true, -- enable in Markdown (disabled by default) [oai_citation:4‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=require%28,env%20files%20return%20false)
-- 			yaml = true, -- (example) keep disabled for YAML files
-- 			-- add any other filetypes overrides if needed; by default most languages are enabled
-- 		},
-- 		-- Optionally, set NodeJS path if your system Node is < 20:
-- 		-- copilot_node_command = "/path/to/node>=20"  [oai_citation:5‡neovimcraft.com](https://neovimcraft.com/plugin/zbirenbaum/copilot.lua/#:~:text=Use%20this%20field%20to%20provide,must%20be%2020%20or%20newer)
-- 	},
-- }

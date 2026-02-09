return {
	"epwalsh/obsidian.nvim",
	version = "*", -- Use the latest release
	lazy = true,
	ft = "markdown",
	dependencies = {
		"nvim-lua/plenary.nvim",
		"nvim-telescope/telescope.nvim",
		"nvim-treesitter/nvim-treesitter",
	},
	opts = {
		workspaces = {
			{
				name = "fuentastic",
				path = "~/fuentastic",
			},
		},

		-- Where to put new notes
		notes_subdir = "inbox",
		new_notes_location = "notes_subdir",

		-- Mapping Obsidian concepts to Neovim
		daily_notes = {
			folder = "dailies",
			date_format = "%Y-%m-%d",
			alias_format = "%B %-d, %Y",
			template = "daily_template.md", -- Assumes this is in your templates folder
		},

		-- Completion of [[links]]
		completion = {
			nvim_cmp = true,
			min_chars = 2,
		},

		-- Keybindings inside the plugin
		mappings = {
			-- Overrides the 'gf' (go to file) to work with [[wikilinks]]
			["gf"] = {
				action = function()
					return require("obsidian").util.gf_passthrough()
				end,
				opts = { noremap = false, expr = true, buffer = true },
			},
			-- Toggle check-boxes
			["<leader>ch"] = {
				action = function()
					return require("obsidian").util.toggle_checkbox()
				end,
				opts = { buffer = true },
			},
			-- Smart action (follow link, opens URL, etc.)
			["<cr>"] = {
				action = function()
					return require("obsidian").util.smart_action()
				end,
				opts = { buffer = true, expr = true },
			},
		},

		-- UI customization (similar to Obsidian's appearance)
		ui = {
			enable = true,
			update_debounce = 200,
			checkboxes = {
				[" "] = { char = "󰄱", hl_group = "ObsidianTodo" },
				["x"] = { char = "", hl_group = "ObsidianDone" },
			},
			external_link_icon = { char = "", hl_group = "ObsidianExtLinkIcon" },
			reference_text = { hl_group = "ObsidianRefText" },
			highlight_text = { hl_group = "ObsidianHighlightText" },
		},

		-- How notes are named
		note_id_func = function(title)
			-- If title is given, use it as the filename (Obsidian style)
			-- Otherwise, use a timestamp
			if title ~= nil then
				return title
			else
				return tostring(os.time())
			end
		end,
	},
	config = function(_, opts)
		require("obsidian").setup(opts)

		-- Custom Keymaps for searching and navigating the vault
		vim.keymap.set("n", "<leader>on", ":ObsidianNew<cr>", { desc = "New Obsidian Note" })
		vim.keymap.set("n", "<leader>os", ":ObsidianSearch<cr>", { desc = "Search Vault (Grep)" })
		vim.keymap.set("n", "<leader>og", ":ObsidianSearch<cr>", { desc = "Search Vault (Grep)" })
		vim.keymap.set("n", "<leader>oo", ":ObsidianQuickSwitch<cr>", { desc = "Open Note (Telescope)" })
		vim.keymap.set("n", "<leader>ob", ":ObsidianBacklinks<cr>", { desc = "Show Backlinks" })
		vim.keymap.set("n", "<leader>ot", ":ObsidianTags<cr>", { desc = "Search Tags" })
		vim.keymap.set("n", "<leader>od", ":ObsidianToday<cr>", { desc = "Today's Note" })
	end,
}
-- return {
--   'epwalsh/obsidian.nvim',
--   version = '*', -- recommended, use latest release instead of latest commit
--   lazy = true,
--   ft = 'markdown',
--   -- Replace the above line with this if you only want to load obsidian.nvim for markdown files in your vault:
--   -- event = {
--   --   -- If you want to use the home shortcut '~' here you need to call 'vim.fn.expand'.
--   --   -- E.g. "BufReadPre " .. vim.fn.expand "~" .. "/my-vault/*.md"
--   --   -- refer to `:h file-pattern` for more examples
--   --   "BufReadPre path/to/my-vault/*.md",
--   --   "BufNewFile path/to/my-vault/*.md",
--   -- },
--   dependencies = {
--     'nvim-lua/plenary.nvim',
--   },
--   opts = {
--     workspaces = {
--       {
--         name = 'personal',
--         path = '~/fuentastic',
--       },
--     },
--
--     -- see below for full list of options 👇
--   },
-- }

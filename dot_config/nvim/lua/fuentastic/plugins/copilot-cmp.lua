return {
	-- 2. copilot-cmp plugin (bridge between Copilot and nvim-cmp)
	"zbirenbaum/copilot-cmp",
	dependencies = "zbirenbaum/copilot.lua",
	event = { "InsertEnter", "LspAttach" }, -- load source when entering Insert or attaching LSP [oai_citation:6‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=%7B%20event%20%3D%20%7B%20,fix_pairs%20%3D%20true%2C)
	config = function()
		require("copilot_cmp").setup({
			fix_pairs = true, -- resolves bracket/quote pairing issues [oai_citation:7‡github.com](https://github.com/zbirenbaum/copilot-cmp#:~:text=)
		})
	end,
}

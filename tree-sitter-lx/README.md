# tree-sitter-lx

Tree-sitter grammar for the `lx` pattern language.

## Status

This grammar is intended for editor integration first, especially Neovim.
It tracks the current `lx` syntax, including:

- variable definitions with `$name = ...`
- `pattern:` sections
- captures as `NAME { ... }`
- groups `(...)`
- quantifiers like `1`, `1..`, and `1..5`
- builtins `@name`
- variables `$name`
- strings, comments, and character classes

The grammar is intentionally a bit permissive around incomplete edits so it
remains useful while typing.

## Generate

```sh
tree-sitter generate
tree-sitter test
```

## Neovim

You can point `nvim-treesitter` at this directory as a local parser source and
reuse the bundled `queries/highlights.scm`.

Example `init.lua` setup:

```lua
vim.filetype.add({
  extension = {
    lx = "lx",
  },
})

local parser_config = require("nvim-treesitter.parsers").get_parser_configs()

parser_config.lx = {
  install_info = {
    url = "/Volumes/Au/code/litexc/tree-sitter-lx",
    files = { "src/parser.c" },
    generate_requires_npm = false,
  },
  filetype = "lx",
}

require("nvim-treesitter.configs").setup({
  ensure_installed = {},
  highlight = {
    enable = true,
  },
})
```

Then run:

```vim
:TSInstall lx
```

If you want a portable config, replace the hardcoded path with your local clone
path for `tree-sitter-lx`.

# Litex

`litex` is a small pattern language and CLI that compiles readable `.lx` rules into regular expressions.

It is built for cases where raw regex is too dense to write or review directly, but you still want regex-compatible matching, captures, search, and replacement.

## What It Does

- Defines patterns in a whitespace-oriented DSL.
- Compiles `.lx` source to a Go regular expression.
- Supports reusable variables, named captures, alternation, classes, quantifiers, and builtin patterns.
- Exposes a CLI for compile, find, find-all, test, and replace workflows.
- Includes a `tree-sitter` grammar in [`tree-sitter-lx`](./tree-sitter-lx) for editor integration.

## Project Layout

- [`main.go`](./main.go): CLI entry point.
- [`engine`](./engine): lexer, parser, resolver, compiler, and builtins.
- [`tree-sitter-lx`](./tree-sitter-lx): Tree-sitter grammar for `.lx` files.
- [`tmp`](./tmp): example and scratch `.lx` files used during development.

## Install

Requirements:

- Go 1.21.4+.

Run without installing:

```sh
go run . help
```

Build a local binary:

```sh
go build -o lx .
./lx help
```

## CLI

```text
lx compile [-f pattern.lx | <pattern>] [--name=value ...]
lx find [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
lx findall [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
lx test [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
lx replace [-f pattern.lx | <pattern>] <replacement> [-f content.txt | <content>] [--name=value ...]
```

Notes:

- Inline patterns are treated as the body of `pattern:` automatically.
- Use `-f` when you want to pass a full `.lx` source file as-is.
- Content can be inline text or loaded from a file with `-f`.
- `replaceall` is also implemented even though it is not listed in `help`.

## Quick Examples

Compile an inline pattern:

```sh
go run . compile 'digit 4 "-" digit 4'
```

Test whether text matches:

```sh
go run . test 'digit 4' 1234
```

Find all matches:

```sh
go run . findall '@email' 'a@b.com x y@test.dev'
```

Compile a file-backed pattern:

```sh
go run . compile -f tmp/email.lx
```

Use a required CLI variable from a file-backed pattern:

```sh
go run . test -f pattern.lx hello --value=hello
```

## Language Overview

An `.lx` program has two parts:

1. Optional variable definitions at the top.
2. A required `pattern:` section.

Minimal file:

```lx
pattern:
digit 4
```

Variables:

```lx
$sep = ("-" | whitespace)

pattern:
digit 4
$sep 0..1
digit 4
```

Required CLI variables:

```lx
$value = require value

pattern:
$value
```

Run it with:

```sh
go run . test -f pattern.lx hello --value=hello
```

Captures:

```lx
pattern:
UserId {
  letter 2..4
  digit 3
}
```

Alternation and grouping:

```lx
pattern:
("yes" | "no" | "maybe")
```

Character classes:

```lx
pattern:
[letter digit _ -] 3..12
```

Negation:

```lx
pattern:
not whitespace 1..
```

## Core Syntax

### Builtin identifiers

These compile directly to regex fragments:

- `digit`
- `letter`
- `whitespace`
- `tab`
- `space`
- `newline`
- `upper`
- `lower`
- `linestart`
- `lineend`
- `anychar`

### Quantifiers

Quantifiers are written after the target expression:

- `1` means exactly once.
- `0..1` means optional.
- `0..` means zero or more.
- `1..` means one or more.
- `2..5` means between 2 and 5 times.

Examples:

```lx
digit 4
letter 1..
"ab" 2..3
```

### Strings

Use double quotes for literal strings:

```lx
"@"
"https://"
```

### Comments

`#` starts a comment that runs to the end of the line.

### Builtin patterns

The engine includes reusable higher-level patterns:

- `@email`
- `@phone`
- `@creditcard`
- `@url`
- `@domain`

Example:

```lx
pattern:
@email
```

## Example: Email Pattern

[`tmp/email.lx`](./tmp/email.lx) shows a non-trivial pattern with variables and nested captures:

```lx
$alwaysValid = [letter digit _ % +]

pattern:

Email {
  Email__localPart {
    (
      $alwaysValid
      (
        [letter digit _ % + - .] 0..
        $alwaysValid
      ) 0..1
    ) 1..64
  }
  "@"
  Email__domain { @domain }
}
```

Compiling it:

```sh
go run . compile -f tmp/email.lx
```

## Current Constraints

- Definitions must appear before `pattern:`.
- Defined variables must be used, or resolution fails.
- Required variables are injected as literal strings from CLI flags.
- Builtins cannot be used more than once in a single pattern.
- Capture names must be unique.
- Anchors like `linestart` and `lineend` cannot be quantified.

## Tree-sitter

The repo includes a separate Tree-sitter grammar in [`tree-sitter-lx`](./tree-sitter-lx) for syntax highlighting and editor support.

See [`tree-sitter-lx/README.md`](./tree-sitter-lx/README.md) for setup details.

## Status

This repo is still small and evolving. The language and CLI are usable now, but the surface area is intentionally narrow and a few behaviors are still implementation-shaped rather than polished product conventions.

package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var identCharRegex = regexp.MustCompile("[a-zA-Z0-9_]")
var identRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
var numberCharRegex = regexp.MustCompile("[0-9]")
var numberRegex = regexp.MustCompile("^[1-9][0-9]*$")
var whitespaceCharRegex = regexp.MustCompile(`\s`)
var upperCharRegex = regexp.MustCompile(`[A-Z]`)
var lowerCharRegex = regexp.MustCompile(`[a-z]`)

var builtins = map[string]string{ // strings pulled from builtins/
	"email": `capture email ( "EMAIL" )`,
	"phone": ``,
}

var idents = map[string]struct{}{
	"linestart":  {},
	"lineend":    {},
	"whitespace": {},
	"tab":        {},
	"space":      {},
	"newline":    {},
	"digit":      {},
	"anychar":    {},
	"upper":      {},
	"lower":      {},
}

type position struct {
	line   int
	column int
	offset int
}

type lxError struct {
	msg string
	pos position
}

func printError(src string, err lxError) {
	lines := strings.Split(src, "\n")

	if err.pos.line <= 0 || err.pos.line > len(lines) {
		fmt.Println("Error:", err.msg)
		return
	}

	line := lines[err.pos.line-1]

	fmt.Printf("Error: %s\n", err.msg)
	fmt.Printf(" %d | %s\n", err.pos.line, line)

	// caret line
	fmt.Print("   | ")

	for i := 1; i < err.pos.column; i++ {
		if i-1 < len(line) && line[i-1] == '\t' {
			fmt.Print("    ")
		} else {
			fmt.Print(" ")
		}
	}

	fmt.Println(" ^")
}

func Run(mode, pattern, content, replacement string, vars map[string]string) string {
	lexer := newLexer(pattern)
	tokens, err := lexer.lex()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	} else {
		// printTokens(tokens)
	}

	parser := newParser(tokens)
	ast, err := parser.parse()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}

	resolver := newResolver(ast, vars)
	resolvedAst, err := resolver.resolve()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}

	resolvedAst.print()

	// nfa := Compile(ast)
	return ""
	/*
		switch mode {
		case "find":
			return Find(nfa, content)
		case "findall":
			return FindAll(nfa, content)
		case "test":
			return Test(nfa, content)
		}
	*/
}

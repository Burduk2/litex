package engine

import (
	"fmt"
	"strings"
)

var builtins = map[string]struct{}{
	"email": {},
	"phone": {},
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
	fmt.Printf(" %d:%d | %s\n", err.pos.line, err.pos.column, line)

	// caret line
	fmt.Print("     | ")

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
	} else {
		//printTokens(tokens)
	}

	parser := newParser(tokens)
	ast, err := parser.parse()
	if err != nil {
		printError(pattern, *err)
	} else {
		ast.print()
	}

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

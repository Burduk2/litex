package engine

import (
	"os"
	"regexp"
	"strings"
)

type RunnerMode int

const (
	CompileMode RunnerMode = iota
	FindMode
	FindAllMode
	TestMode
	ReplaceMode
	ReplaceAllMode
)

type RunnerOptions struct {
	Mode        RunnerMode
	Pattern     string
	Content     string
	Replacement string
	Vars        map[string]string
}

var identCharRegex = regexp.MustCompile("[a-zA-Z0-9_]")
var identRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
var numberCharRegex = regexp.MustCompile("[0-9]")
var numberRegex = regexp.MustCompile("^[1-9][0-9]*$")
var whitespaceCharRegex = regexp.MustCompile(`\s`)
var upperCharRegex = regexp.MustCompile(`[A-Z]`)
var lowerCharRegex = regexp.MustCompile(`[a-z]`)

var runeIdents = map[string]struct{}{
	"digit":      {},
	"letter":     {},
	"whitespace": {},
	"tab":        {},
	"space":      {},
	"newline":    {},
	"upper":      {},
	"lower":      {},
}

var wildcardIdents = map[string]struct{}{
	"linestart": {},
	"lineend":   {},
	"anychar":   {},
}

type position struct {
	line   int
	column int
	offset int
}

func Run(options RunnerOptions) string {
	pattern := options.Pattern
	vars := options.Vars
	mode := options.Mode
	content := options.Content

	lexer := newLexer(pattern)
	tokens, err := lexer.lex()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}
	// printTokens(tokens)

	parser := newParser(tokens)
	ast, err := parser.parse()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}
	// ast.print()

	resolver := newResolver(ast, vars)
	resolvedAst, err := resolver.resolve()
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}
	// resolvedAst.print()

	compiled, err := Compile(resolvedAst)
	if err != nil {
		printError(pattern, *err)
		os.Exit(1)
	}

	switch mode {
	case CompileMode:
		return compiled.pattern
	case FindMode:
		return compiled.regex.FindString(content)
	case FindAllMode:
		return strings.Join(compiled.regex.FindAllString(content, -1), "\n")
	case ReplaceMode:
		return replaceFirst(compiled.regex, content, options.Replacement)
	case ReplaceAllMode:
		return compiled.regex.ReplaceAllString(content, options.Replacement)
	case TestMode:
		if compiled.regex.MatchString(content) {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func replaceFirst(re *regexp.Regexp, content, replacement string) string {
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content
	}
	return content[:loc[0]] + replacement + content[loc[1]:]
}

package engine

import (
	"fmt"
	"main/engine/builtin"
	"regexp"
	"strings"
)

type compiledProgram struct {
	pattern string
	regex   *regexp.Regexp
}

type compiler struct{}

func Compile(program program) (compiledProgram, *lxError) {
	compiledPattern, err := newCompiler().compileSequence(program.expressions)
	if err != nil {
		return compiledProgram{}, err
	}

	regex, compileErr := regexp.Compile(compiledPattern)
	if compileErr != nil {
		return compiledProgram{}, &lxError{
			msg: "invalid compiled regex: " + compileErr.Error(),
		}
	}

	return compiledProgram{
		pattern: compiledPattern,
		regex:   regex,
	}, nil
}

func newCompiler() *compiler {
	return &compiler{}
}

func (compiler *compiler) compileSequence(expressions []expr) (string, *lxError) {
	var builder strings.Builder

	for _, expression := range expressions {
		compiledExpr, err := compiler.compileExpr(expression)
		if err != nil {
			return "", err
		}
		builder.WriteString(compiledExpr)
	}

	return builder.String(), nil
}

func (compiler *compiler) compileExpr(expression expr) (string, *lxError) {
	switch current := expression.(type) {
	case identExpr:
		return compileIdentRegex(current)
	case builtinExpr:
		pattern, ok := builtin.Builtins[current.name]
		if !ok {
			return "", &lxError{
				msg: "builtin is not compilable to regex: @" + current.name,
				pos: current.pos,
			}
		}
		return pattern, nil
	case literalValueExpr:
		quoted := regexp.QuoteMeta(current.value)
		if len([]rune(current.value)) > 1 {
			return "(?:" + quoted + ")", nil
		}
		return quoted, nil
	case classExpr:
		content, err := compiler.compileClassContent(current)
		if err != nil {
			return "", err
		}
		return "[" + content + "]", nil
	case groupExpr:
		return compiler.compileGroup(current)
	case captureExpr:
		content, err := compiler.compileGroup(current.group)
		if err != nil {
			return "", err
		}
		return "(?P<" + current.name + ">" + unwrapNonCapturingGroup(content) + ")", nil
	case quantifierExpr:
		return compiler.compileQuantifier(current)
	case notExpr:
		content, err := compiler.compileNegatedTarget(current.target)
		if err != nil {
			return "", err
		}
		return "[^" + content + "]", nil
	default:
		return "", &lxError{
			msg: "expression is not compilable to regex",
			pos: expression.exprPos(),
		}
	}
}

func (compiler *compiler) compileGroup(group groupExpr) (string, *lxError) {
	branches := make([]string, 0, len(group.branches))

	for _, branch := range group.branches {
		compiledBranch, err := compiler.compileSequence(branch)
		if err != nil {
			return "", err
		}
		branches = append(branches, compiledBranch)
	}

	if len(branches) == 1 {
		return "(?:" + branches[0] + ")", nil
	}

	return "(?:" + strings.Join(branches, "|") + ")", nil
}

func (compiler *compiler) compileQuantifier(quantifier quantifierExpr) (string, *lxError) {
	if ident, ok := quantifier.target.(identExpr); ok {
		if ident.name == "linestart" || ident.name == "lineend" {
			return "", &lxError{
				msg: "anchors cannot be quantified",
				pos: quantifier.pos,
			}
		}
	}

	target, err := compiler.compileExpr(quantifier.target)
	if err != nil {
		return "", err
	}

	var suffix string
	switch {
	case quantifier.max != nil && quantifier.min == 0 && *quantifier.max == 1:
		suffix = "?"
	case quantifier.max == nil && quantifier.min == 0:
		suffix = "*"
	case quantifier.max == nil && quantifier.min == 1:
		suffix = "+"
	case quantifier.max == nil:
		suffix = fmt.Sprintf("{%d,}", quantifier.min)
	case *quantifier.max == quantifier.min:
		suffix = fmt.Sprintf("{%d}", quantifier.min)
	default:
		suffix = fmt.Sprintf("{%d,%d}", quantifier.min, *quantifier.max)
	}

	return compiledQuantifierTarget(quantifier.target, target) + suffix, nil
}

func (compiler *compiler) compileClassContent(class classExpr) (string, *lxError) {
	var builder strings.Builder

	for _, item := range class.items {
		compiledItem, err := compiler.compileClassItem(item)
		if err != nil {
			return "", err
		}
		builder.WriteString(compiledItem)
	}

	return builder.String(), nil
}

func (compiler *compiler) compileClassItem(item expr) (string, *lxError) {
	switch current := item.(type) {
	case classChar:
		return escapeCharClassRune(current.value), nil
	case classRange:
		return escapeCharClassRune(current.from) + "-" + escapeCharClassRune(current.to), nil
	case identExpr:
		return compileClassIdent(current)
	default:
		return "", &lxError{
			msg: "character class item is not compilable to regex",
			pos: item.exprPos(),
		}
	}
}

func (compiler *compiler) compileNegatedTarget(target expr) (string, *lxError) {
	switch current := target.(type) {
	case identExpr:
		return compileClassIdent(current)
	case classExpr:
		return compiler.compileClassContent(current)
	default:
		return "", &lxError{
			msg: "'not' can only compile identifiers and character classes",
			pos: target.exprPos(),
		}
	}
}

func compileIdentRegex(ident identExpr) (string, *lxError) {
	switch ident.name {
	case "digit":
		return `\d`, nil
	case "letter":
		return `[A-Za-z]`, nil
	case "whitespace":
		return `\s`, nil
	case "tab":
		return `\t`, nil
	case "space":
		return ` `, nil
	case "newline":
		return `\n`, nil
	case "upper":
		return `[A-Z]`, nil
	case "lower":
		return `[a-z]`, nil
	case "linestart":
		return `^`, nil
	case "lineend":
		return `$`, nil
	case "anychar":
		return `.`, nil
	default:
		return "", &lxError{
			msg: "identifier is not compilable to regex: " + ident.name,
			pos: ident.pos,
		}
	}
}

func compileClassIdent(ident identExpr) (string, *lxError) {
	switch ident.name {
	case "digit":
		return `\d`, nil
	case "letter":
		return `A-Za-z`, nil
	case "whitespace":
		return `\s`, nil
	case "tab":
		return `\t`, nil
	case "space":
		return ` `, nil
	case "newline":
		return `\n`, nil
	case "upper":
		return `A-Z`, nil
	case "lower":
		return `a-z`, nil
	default:
		return "", &lxError{
			msg: "identifier is not valid inside a regex character class: " + ident.name,
			pos: ident.pos,
		}
	}
}

func escapeCharClassRune(value rune) string {
	switch value {
	case '\\', '-', ']', '^':
		return `\` + string(value)
	default:
		return string(value)
	}
}

func unwrapNonCapturingGroup(pattern string) string {
	if strings.HasPrefix(pattern, "(?:") && strings.HasSuffix(pattern, ")") {
		return pattern[3 : len(pattern)-1]
	}
	return pattern
}

func compiledQuantifierTarget(targetExpr expr, compiledTarget string) string {
	switch current := targetExpr.(type) {
	case literalValueExpr:
		if len([]rune(current.value)) <= 1 {
			return compiledTarget
		}
	case identExpr, classExpr, notExpr:
		return compiledTarget
	}

	if strings.HasPrefix(compiledTarget, "(?:") && strings.HasSuffix(compiledTarget, ")") {
		return compiledTarget
	}
	return "(?:" + compiledTarget + ")"
}

package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// --------------- PARSE
func (p program) print() {
	var parts []string
	for _, e := range p.expressions {
		parts = append(parts, exprString(e))
	}
	fmt.Printf("program[\n  " + strings.Join(parts, "\n  ") + "\n]")
}

func exprString(e expr) string {
	switch v := e.(type) {
	case identExpr:
		return "ident(" + v.name + ")"
	case variableExpr:
		return "var($" + v.name + ")"
	case requiredVarExpr:
		return "require(" + v.name + ")"
	case notExpr:
		return "not(" + exprString(v.target) + ")"
	case builtinExpr:
		return "builtin(@" + v.name + ")"
	case literalValueExpr:
		return strconv.Quote(v.value)
	case groupExpr:
		return "group(" + joinBranches(v.branches) + ")"
	case captureExpr:
		return "capture(" + v.name + ", " + exprString(v.group) + ")"
	case defineExpr:
		return "define($" + v.name + " = " + exprString(v.value) + ")"
	case patternSectionExpr:
		return "\n  pattern:"
	case quantifierExpr:
		if v.max == nil {
			return fmt.Sprintf("quant(%s, %d, inf)", exprString(v.target), v.min)
		}
		return fmt.Sprintf("quant(%s, %d, %d)", exprString(v.target), v.min, *v.max)
	case classExpr:
		content := []string{}
		for _, item := range v.items {
			switch itemType := item.(type) {
			case classChar:
				content = append(content, fmt.Sprintf("char(%s)", string(itemType.value)))
			case classRange:
				content = append(content, fmt.Sprintf("range(%s-%s)", string(itemType.from), string(itemType.to)))
			default:
				content = append(content, exprString(item))
			}
		}
		return "class(" + strings.Join(content, ", ") + ")"
	default:
		return "<unknown expr>"
	}
}

func joinExprs(exprs []expr) string {
	var parts []string
	for _, e := range exprs {
		parts = append(parts, exprString(e))
	}
	return strings.Join(parts, ", ")
}

func joinBranches(branches [][]expr) string {
	var parts []string
	for _, branch := range branches {
		parts = append(parts, joinExprs(branch))
	}
	return strings.Join(parts, " | ")
}

// ---------------- LEX
func (t tokenType) String() string {
	switch t {
	case identToken:
		return "ident"
	case notToken:
		return "not"
	case captureToken:
		return "capture"
	case requireToken:
		return "require"
	case variableToken:
		return "variable"
	case builtinToken:
		return "builtin"
	case literalToken:
		return "string"
	case classToken:
		return "class"
	case numberToken:
		return "number"
	case lparenToken:
		return "lparen"
	case rparenToken:
		return "rparen"
	case lbracketToken:
		return "lbracket"
	case rbracketToken:
		return "rbracket"
	case starToken:
		return "star"
	case plusToken:
		return "plus"
	case qmarkToken:
		return "qmark"
	case pipeToken:
		return "pipe"
	case dashToken:
		return "dash"
	case equalsToken:
		return "equals"
	case colonToken:
		return "colon"
	case eofToken:
		return "eof"
	default:
		return "unknown"
	}
}

func printTokens(tokens []token) {
	for i, t := range tokens {
		fmt.Printf(
			"%3d  %v  %-10q  (%d:%d)\n",
			i,
			t.typ,
			t.val,
			t.pos.line,
			t.pos.column,
		)
	}
}

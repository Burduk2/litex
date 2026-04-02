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
	case builtinExpr:
		return "builtin(@" + v.name + ")"
	case literalValueExpr:
		return strconv.Quote(v.value)
	case numberExpr:
		return "number(" + v.value + ")"
	case groupExpr:
		return "group(" + joinExprs(v.expressions) + ")"
	case captureExpr:
		return "capture(" + v.name + ", " + exprString(v.group) + ")"
	case orExpr:
		return "or(" + exprString(v.group) + ")"
	case defineExpr:
		return "define($" + v.name + " = " + exprString(v.value) + ")"
	case quantifierExpr:
		if v.max == nil {
			return fmt.Sprintf("quant(%s, %d, inf)", exprString(v.target), v.min)
		}
		return fmt.Sprintf("quant(%s, %d, %d)", exprString(v.target), v.min, *v.max)
	case classExpr:
		return "class(...)"
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

// ---------------- LEX
func (t tokenType) String() string {
	switch t {
	case ident:
		return "ident"
	case variable:
		return "variable"
	case builtin:
		return "builtin"
	case literal:
		return "string"
	case class:
		return "class"
	case number:
		return "number"
	case lparen:
		return "lparen"
	case rparen:
		return "rparen"
	case lbracket:
		return "lbracket"
	case rbracket:
		return "rbracket"
	case star:
		return "star"
	case plus:
		return "plus"
	case qmark:
		return "qmark"
	case dash:
		return "dash"
	case equals:
		return "equals"
	case colon:
		return "colon"
	case eof:
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

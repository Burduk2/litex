package engine

import (
	"fmt"
	"main/engine/builtin"
	"slices"
)

type program struct {
	expressions []expr
}

type expr interface {
	exprNode()
	exprPos() position
}

type identExpr struct {
	pos  position
	name string
}

func (identExpr) exprNode()              {}
func (expr identExpr) exprPos() position { return expr.pos }

type variableExpr struct {
	pos  position
	name string
}

func (variableExpr) exprNode()              {}
func (expr variableExpr) exprPos() position { return expr.pos }

type requiredVarExpr struct {
	pos  position
	name string
}

func (requiredVarExpr) exprNode()              {}
func (expr requiredVarExpr) exprPos() position { return expr.pos }

type notExpr struct {
	pos    position
	target expr
}

func (notExpr) exprNode()              {}
func (expr notExpr) exprPos() position { return expr.pos }

type builtinExpr struct {
	pos  position
	name string
}

func (builtinExpr) exprNode()              {}
func (expr builtinExpr) exprPos() position { return expr.pos }

type literalValueExpr struct {
	pos   position
	value string
}

func (literalValueExpr) exprNode()              {}
func (expr literalValueExpr) exprPos() position { return expr.pos }

type classExpr struct {
	pos   position
	items []expr
}

func (classExpr) exprNode()              {}
func (expr classExpr) exprPos() position { return expr.pos }

type groupExpr struct {
	pos      position
	branches [][]expr
}

func (groupExpr) exprNode()              {}
func (expr groupExpr) exprPos() position { return expr.pos }

type captureExpr struct {
	pos   position
	name  string
	group groupExpr
}

func (captureExpr) exprNode()              {}
func (expr captureExpr) exprPos() position { return expr.pos }

type quantifierExpr struct {
	pos    position
	target expr
	min    int
	max    *int
}

func (quantifierExpr) exprNode()              {}
func (expr quantifierExpr) exprPos() position { return expr.pos }

type defineExpr struct {
	pos   position
	name  string
	value expr
}

func (defineExpr) exprNode()              {}
func (expr defineExpr) exprPos() position { return expr.pos }

type patternSectionExpr struct {
	pos position
}

func (patternSectionExpr) exprNode()              {}
func (expr patternSectionExpr) exprPos() position { return expr.pos }

type classChar struct {
	pos   position
	value rune
}
type classRange struct {
	pos  position
	from rune
	to   rune
}

func (classChar) exprNode()               {}
func (classRange) exprNode()              {}
func (expr classChar) exprPos() position  { return expr.pos }
func (expr classRange) exprPos() position { return expr.pos }

type parser struct {
	tokens             []token
	pos                int
	line               int
	column             int
	seenPatternSection bool
	inDefinitionValue  int
}

func newParser(tokens []token) *parser {
	return &parser{
		tokens: tokens,
		pos:    0,
		line:   1,
		column: 1,
	}
}

func (parser *parser) peek() token {
	if parser.pos >= len(parser.tokens) {
		return token{typ: eofToken}
	}
	return parser.tokens[parser.pos]
}

func (parser *parser) advance() token {
	currentToken := parser.peek()
	parser.pos++
	return currentToken
}

func (parser *parser) match(expected tokenType) bool {
	if parser.peek().typ == expected {
		parser.advance()
		return true
	}
	return false
}

func (parser *parser) expect(expected tokenType) (token, *lxError) {
	currentToken := parser.peek()

	if currentToken.typ != expected {
		return currentToken, &lxError{
			msg: "unexpected token: expected " + expected.String() + ", got " + currentToken.typ.String(),
			pos: currentToken.pos,
		}
	}

	return parser.advance(), nil
}

func (parser *parser) parse() (program, *lxError) {
	expressions, err := parser.parseSequence(eofToken)
	if err != nil {
		return program{}, err
	}
	if !parser.seenPatternSection {
		return program{}, &lxError{
			msg: "pattern entry point ('pattern:') not found",
			pos: parser.peek().pos,
		}
	}

	return program{expressions: expressions}, nil
}

func (parser *parser) parseExpr() (expr, *lxError) {
	if parser.startsPatternSection() {
		return parser.parsePatternSection()
	}

	if parser.peek().typ == variableToken && parser.lookahead(1).typ == equalsToken {
		return parser.parseDefinition()
	}

	if !parser.seenPatternSection && parser.inDefinitionValue == 0 {
		return nil, &lxError{
			msg: "only definitions are allowed before 'pattern:'",
			pos: parser.peek().pos,
		}
	}

	primary, err := parser.parsePrimary()
	if err != nil {
		return nil, err
	}

	return parser.parseQuantifier(primary)
}

func (parser *parser) parsePrimary() (expr, *lxError) {
	currentToken := parser.peek()

	switch currentToken.typ {
	case identToken:
		parser.advance()
		return identExpr{pos: currentToken.pos, name: currentToken.val}, nil
	case notToken:
		return parser.parseNotExpr()
	case captureToken:
		return parser.parseCapture()
	case requireToken:
		return parser.parseRequireExpr()
	case variableToken:
		parser.advance()
		return variableExpr{pos: currentToken.pos, name: currentToken.val}, nil
	case builtinToken:
		parser.advance()
		return builtinExpr{pos: currentToken.pos, name: currentToken.val}, nil
	case literalToken:
		parser.advance()
		return literalValueExpr{pos: currentToken.pos, value: currentToken.val}, nil
	case classToken:
		parser.advance()
		items, err := parseClassItems(currentToken)
		if err != nil {
			return nil, err
		}
		return classExpr{pos: currentToken.pos, items: items}, nil
	case lparenToken:
		return parser.parseGroup()
	case pipeToken:
		return nil, &lxError{
			msg: "unexpected alternative separator",
			pos: currentToken.pos,
		}
	case numberToken:
		return nil, &lxError{
			msg: "quantifiers cannot be quantified",
			pos: currentToken.pos,
		}
	case rangeSepToken:
		return nil, &lxError{
			msg: "invalid quantifier",
			pos: currentToken.pos,
		}
	default:
		return nil, &lxError{
			msg: "unexpected token: " + currentToken.typ.String(),
			pos: currentToken.pos,
		}
	}
}

func (parser *parser) parseDefinition() (expr, *lxError) {
	if parser.seenPatternSection {
		return nil, &lxError{
			msg: "definitions are not allowed after 'pattern:'",
			pos: parser.peek().pos,
		}
	}

	nameToken, err := parser.expect(variableToken)
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(equalsToken); err != nil {
		return nil, err
	}

	expressions, err := parser.parseDefinitionValue()
	if err != nil {
		return nil, err
	}

	return defineExpr{
		pos:   nameToken.pos,
		name:  nameToken.val,
		value: groupExpr{pos: nameToken.pos, branches: [][]expr{expressions}},
	}, nil
}

func (parser *parser) parsePatternSection() (expr, *lxError) {
	if parser.seenPatternSection {
		return nil, &lxError{
			msg: "'pattern:' can appear only once per program",
			pos: parser.peek().pos,
		}
	}

	patternToken, err := parser.expect(identToken)
	if err != nil {
		return nil, err
	}
	if patternToken.val != "pattern" {
		return nil, &lxError{
			msg: "unexpected section label: " + patternToken.val,
			pos: patternToken.pos,
		}
	}

	if _, err := parser.expect(colonToken); err != nil {
		return nil, err
	}

	parser.seenPatternSection = true
	return patternSectionExpr{pos: patternToken.pos}, nil
}

func (parser *parser) parseDefinitionValue() ([]expr, *lxError) {
	var expressions []expr
	parser.inDefinitionValue++
	defer func() { parser.inDefinitionValue-- }()

	for {
		currentToken := parser.peek()
		if currentToken.typ == eofToken || parser.startsDefinition() || parser.startsPatternSection() {
			return expressions, nil
		}

		expression, err := parser.parseExpr()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)

		if _, ok := expression.(requiredVarExpr); ok {
			return expressions, nil
		}
	}
}

func (parser *parser) parseGroup() (expr, *lxError) {
	startToken, err := parser.expect(lparenToken)
	if err != nil {
		return nil, err
	}

	group, err := parser.parseGroupBody()
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(rparenToken); err != nil {
		return nil, err
	}

	group.pos = startToken.pos
	return group, nil
}

func (parser *parser) parseCapture() (expr, *lxError) {
	_, err := parser.expect(captureToken)
	if err != nil {
		return nil, err
	}

	nameToken, err := parser.expect(identToken)
	if err != nil {
		return nil, err
	}
	if _, exists := builtin.GroupNames[nameToken.val]; exists {
		return nil, &lxError{
			msg: fmt.Sprintf("capture group name '%s' is reserved by a builtin, use a different name", nameToken.val),
			pos: nameToken.pos,
		}
	}

	group, err := parser.parseRequiredGroup("capture")
	if err != nil {
		return nil, err
	}
	if isEmptyLiteralGroup(group) {
		return nil, &lxError{
			msg: "capture group cannot be an empty string literal",
			pos: group.pos,
		}
	}

	return captureExpr{pos: nameToken.pos, name: nameToken.val, group: group}, nil
}

func (parser *parser) parseRequireExpr() (expr, *lxError) {
	_, err := parser.expect(requireToken)
	if err != nil {
		return nil, err
	}

	nameToken, err := parser.expect(identToken)
	if err != nil {
		return nil, &lxError{
			msg: "require expects an identifier",
			pos: parser.peek().pos,
		}
	}

	return requiredVarExpr{pos: nameToken.pos, name: nameToken.val}, nil
}

func (parser *parser) parseNotExpr() (expr, *lxError) {
	notTok, err := parser.expect(notToken)
	if err != nil {
		return nil, err
	}

	switch parser.peek().typ {
	case identToken:
		targetToken := parser.advance()
		if _, exists := runeIdents[targetToken.val]; !exists {
			return nil, &lxError{
				msg: "invalid identifier in not expression: " + targetToken.val,
				pos: targetToken.pos,
			}
		}
		return notExpr{
			pos:    notTok.pos,
			target: identExpr{pos: targetToken.pos, name: targetToken.val},
		}, nil
	case classToken:
		targetToken := parser.advance()
		items, err := parseClassItems(targetToken)
		if err != nil {
			return nil, err
		}
		return notExpr{
			pos:    notTok.pos,
			target: classExpr{pos: targetToken.pos, items: items},
		}, nil
	default:
		return nil, &lxError{
			msg: "'not' expects an identifier or character class",
			pos: notTok.pos,
		}
	}
}

func (parser *parser) parseRequiredGroup(context string) (groupExpr, *lxError) {
	if parser.peek().typ != lparenToken {
		return groupExpr{}, &lxError{
			msg: context + " expects a () group",
			pos: parser.peek().pos,
		}
	}

	group, err := parser.parseGroup()
	if err != nil {
		return groupExpr{}, err
	}

	parsedGroup, ok := group.(groupExpr)
	if !ok {
		return groupExpr{}, &lxError{
			msg: context + " expects a () group",
			pos: parser.peek().pos,
		}
	}

	return parsedGroup, nil
}

func (parser *parser) parseGroupBody() (groupExpr, *lxError) {
	var branches [][]expr
	leadingPipe := parser.match(pipeToken)

	for {
		if parser.peek().typ == rparenToken {
			if len(branches) == 0 {
				return groupExpr{}, &lxError{
					msg: "empty group is not allowed",
					pos: parser.peek().pos,
				}
			}
			if leadingPipe {
				return groupExpr{}, &lxError{
					msg: "empty alternative is not allowed",
					pos: parser.peek().pos,
				}
			}
			return groupExpr{branches: branches}, nil
		}

		branch, err := parser.parseBranch(pipeToken, rparenToken)
		if err != nil {
			return groupExpr{}, err
		}
		if len(branch) == 0 {
			return groupExpr{}, &lxError{
				msg: "empty alternative is not allowed",
				pos: parser.peek().pos,
			}
		}
		branches = append(branches, branch)

		if !parser.match(pipeToken) {
			return groupExpr{branches: branches}, nil
		}
	}
}

func (parser *parser) parseBranch(stop ...tokenType) ([]expr, *lxError) {
	var expressions []expr

	for {
		currentToken := parser.peek()
		if tokenIn(currentToken.typ, stop...) {
			return expressions, nil
		}
		if currentToken.typ == eofToken {
			return nil, &lxError{
				msg: "unexpected end of input",
				pos: currentToken.pos,
			}
		}

		expression, err := parser.parseExpr()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
	}
}

func (parser *parser) parseSequence(stop ...tokenType) ([]expr, *lxError) {
	var expressions []expr

	for {
		currentToken := parser.peek()
		if tokenIn(currentToken.typ, stop...) {
			if len(expressions) == 0 {
				return nil, &lxError{
					msg: "a group has no children",
					pos: currentToken.pos,
				}
			}
			return expressions, nil
		}
		if currentToken.typ == eofToken {
			return nil, &lxError{
				msg: "unexpected end of input",
				pos: currentToken.pos,
			}
		}

		expression, err := parser.parseExpr()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
	}
}

func (parser *parser) parseQuantifier(target expr) (expr, *lxError) {
	currentToken := parser.peek()

	if _, alreadyQuantified := target.(quantifierExpr); alreadyQuantified {
		if tokenIn(currentToken.typ, numberToken) {
			return nil, &lxError{
				msg: "quantifiers cannot be quantified",
				pos: currentToken.pos,
			}
		}
		return target, nil
	}

	if !isQuantifiable(target) {
		if tokenIn(currentToken.typ, numberToken) {
			return nil, &lxError{
				msg: "expression cannot be quantified",
				pos: currentToken.pos,
			}
		}
		return target, nil
	}
	if isEmptyQuantifierTarget(target) && currentToken.typ == numberToken {
		return nil, &lxError{
			msg: "empty string literals cannot be quantified",
			pos: target.exprPos(),
		}
	}

	switch currentToken.typ {
	case numberToken:
		return parser.parseNumericQuantifier(target)
	default:
		return target, nil
	}
}

func (parser *parser) parseNumericQuantifier(target expr) (expr, *lxError) {
	firstToken, err := parser.expect(numberToken)
	if err != nil {
		return nil, err
	}

	firstValue, convErr := parseInt(firstToken)
	if convErr != nil {
		return nil, convErr
	}

	if !parser.match(rangeSepToken) {
		if firstValue == 0 {
			return nil, &lxError{
				msg: "invalid quantifier: use 0.. or 0..x instead of 0",
				pos: firstToken.pos,
			}
		}
		max := firstValue
		return quantifierExpr{pos: target.exprPos(), target: target, min: firstValue, max: &max}, nil
	}

	if parser.peek().typ != numberToken {
		return quantifierExpr{pos: target.exprPos(), target: target, min: firstValue, max: nil}, nil
	}

	secondToken, err := parser.expect(numberToken)
	if err != nil {
		return nil, err
	}

	secondValue, convErr := parseInt(secondToken)
	if convErr != nil {
		return nil, convErr
	}
	if secondValue == firstValue {
		return nil, &lxError{
			msg: fmt.Sprintf("redundant quantifier range: use %d instead of %d..%d", firstValue, firstValue, secondValue),
			pos: secondToken.pos,
		}
	}
	if secondValue < firstValue {
		return nil, &lxError{
			msg: "invalid quantifier range",
			pos: secondToken.pos,
		}
	}

	return quantifierExpr{pos: target.exprPos(), target: target, min: firstValue, max: &secondValue}, nil
}

func (parser *parser) startsDefinition() bool {
	return parser.peek().typ == variableToken && parser.lookahead(1).typ == equalsToken
}

func (parser *parser) startsPatternSection() bool {
	return parser.peek().typ == identToken &&
		parser.peek().val == "pattern" &&
		parser.lookahead(1).typ == colonToken
}

func (parser *parser) lookahead(offset int) token {
	index := parser.pos + offset
	if index >= len(parser.tokens) {
		return token{typ: eofToken}
	}
	return parser.tokens[index]
}

func tokenIn(candidate tokenType, expected ...tokenType) bool {
	return slices.Contains(expected, candidate)
}

func isQuantifiable(target expr) bool {
	switch target.(type) {
	case builtinExpr, captureExpr, quantifierExpr, requiredVarExpr:
		return false
	default:
		return true
	}
}

func isEmptyQuantifierTarget(target expr) bool {
	switch current := target.(type) {
	case literalValueExpr:
		return current.value == ""
	case groupExpr:
		return isEmptyLiteralGroup(current)
	default:
		return false
	}
}

func isEmptyLiteralGroup(group groupExpr) bool {
	if len(group.branches) != 1 || len(group.branches[0]) != 1 {
		return false
	}

	return isEmptyStringExpr(group.branches[0][0])
}

func isEmptyStringExpr(expression expr) bool {
	switch current := expression.(type) {
	case literalValueExpr:
		return current.value == ""
	case groupExpr:
		return isEmptyLiteralGroup(current)
	default:
		return false
	}
}

func parseInt(tok token) (int, *lxError) {
	value := 0
	for _, ch := range tok.val {
		value = value*10 + int(ch-'0')
	}
	return value, nil
}

func getClassRangeError(tok token, item []rune, start int) *lxError {
	return &lxError{
		msg: "invalid range in character class: " + string(item),
		pos: position{
			line:   tok.pos.line,
			column: tok.pos.column + start,
		},
	}
}
func parseClassItems(tok token) ([]expr, *lxError) {
	var items []expr
	content := []rune(tok.val)

	for i := 0; i < len(content); {
		if isWhitespace(content[i]) {
			i++
			continue
		}

		start := i
		for i < len(content) && !isWhitespace(content[i]) {
			i++
		}
		item := content[start:i]
		length := len(item)

		itemPos := position{
			line:   tok.pos.line,
			column: tok.pos.column + start,
		}

		if length == 1 {
			items = append(items, classChar{pos: itemPos, value: item[0]})
		} else if length == 4 && item[1] == '.' && item[2] == '.' {
			from, to := item[0], item[3]
			if numberCharRegex.MatchString(string(from)) && numberCharRegex.MatchString(string(to)) {
				if from >= to {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{pos: itemPos, from: from, to: to})
			} else if lowerCharRegex.MatchString(string(from)) && lowerCharRegex.MatchString(string(to)) {
				if int(from) >= int(to) {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{pos: itemPos, from: from, to: to})
			} else if upperCharRegex.MatchString(string(from)) && upperCharRegex.MatchString(string(to)) {
				if int(from) >= int(to) {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{pos: itemPos, from: from, to: to})
			} else {
				return nil, &lxError{
					msg: "invalid range in character class: " + string(item),
					pos: position{
						line:   tok.pos.line,
						column: tok.pos.column + start,
					},
				}
			}
		} else {
			str := string(item)
			if _, exists := runeIdents[str]; !exists {
				return nil, &lxError{
					msg: "invalid identifier in character class: " + str,
					pos: position{
						line:   tok.pos.line,
						column: tok.pos.column + start,
					},
				}
			}
			items = append(items, identExpr{pos: itemPos, name: str})
		}
	}

	return items, nil
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

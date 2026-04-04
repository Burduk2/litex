package engine

import (
	"slices"
)

type program struct {
	expressions []expr
}

type expr interface{ exprNode() }

type identExpr struct {
	name string
}

func (identExpr) exprNode() {}

type variableExpr struct {
	name string
}

func (variableExpr) exprNode() {}

type requiredVarExpr struct {
	name string
}

func (requiredVarExpr) exprNode() {}

type notExpr struct {
	target expr
}

func (notExpr) exprNode() {}

type builtinExpr struct {
	name string
}

func (builtinExpr) exprNode() {}

type literalValueExpr struct {
	value string
}

func (literalValueExpr) exprNode() {}

type classExpr struct {
	items []expr
}

func (classExpr) exprNode() {}

type groupExpr struct {
	branches [][]expr
}

func (groupExpr) exprNode() {}

type captureExpr struct {
	name  string
	group groupExpr
}

func (captureExpr) exprNode() {}

type quantifierExpr struct {
	target expr
	min    int
	max    *int
}

func (quantifierExpr) exprNode() {}

type defineExpr struct {
	name  string
	value expr
}

func (defineExpr) exprNode() {}

type patternSectionExpr struct{}

func (patternSectionExpr) exprNode() {}

type classChar struct {
	value rune
}
type classRange struct {
	from rune
	to   rune
}

func (classChar) exprNode()  {}
func (classRange) exprNode() {}

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
		return identExpr{name: currentToken.val}, nil
	case notToken:
		return parser.parseNotExpr()
	case captureToken:
		return parser.parseCapture()
	case requireToken:
		return parser.parseRequireExpr()
	case variableToken:
		parser.advance()
		return variableExpr{name: currentToken.val}, nil
	case builtinToken:
		parser.advance()
		return builtinExpr{name: currentToken.val}, nil
	case literalToken:
		parser.advance()
		return literalValueExpr{value: currentToken.val}, nil
	case classToken:
		parser.advance()
		items, err := parseClassItems(currentToken)
		if err != nil {
			return nil, err
		}
		return classExpr{items: items}, nil
	case lparenToken:
		return parser.parseGroup()
	case pipeToken:
		return nil, &lxError{
			msg: "unexpected alternative separator",
			pos: currentToken.pos,
		}
	case starToken, plusToken, qmarkToken, numberToken:
		return nil, &lxError{
			msg: "quantifiers cannot be quantified",
			pos: currentToken.pos,
		}
	case dashToken:
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
		name:  nameToken.val,
		value: groupExpr{branches: [][]expr{expressions}},
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
	return patternSectionExpr{}, nil
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
	if _, err := parser.expect(lparenToken); err != nil {
		return nil, err
	}

	group, err := parser.parseGroupBody()
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(rparenToken); err != nil {
		return nil, err
	}

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

	group, err := parser.parseRequiredGroup("capture")
	if err != nil {
		return nil, err
	}

	return captureExpr{name: nameToken.val, group: group}, nil
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

	return requiredVarExpr{name: nameToken.val}, nil
}

func (parser *parser) parseNotExpr() (expr, *lxError) {
	notTok, err := parser.expect(notToken)
	if err != nil {
		return nil, err
	}

	switch parser.peek().typ {
	case identToken:
		targetToken := parser.advance()
		return notExpr{target: identExpr{name: targetToken.val}}, nil
	case classToken:
		targetToken := parser.advance()
		items, err := parseClassItems(targetToken)
		if err != nil {
			return nil, err
		}
		return notExpr{target: classExpr{items: items}}, nil
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
		if tokenIn(currentToken.typ, starToken, plusToken, qmarkToken, numberToken) {
			return nil, &lxError{
				msg: "quantifiers cannot be quantified",
				pos: currentToken.pos,
			}
		}
		return target, nil
	}

	if !isQuantifiable(target) {
		if tokenIn(currentToken.typ, starToken, plusToken, qmarkToken, numberToken) {
			return nil, &lxError{
				msg: "expression cannot be quantified",
				pos: currentToken.pos,
			}
		}
		return target, nil
	}

	switch currentToken.typ {
	case starToken:
		parser.advance()
		return quantifierExpr{target: target, min: 0, max: nil}, nil
	case plusToken:
		parser.advance()
		return quantifierExpr{target: target, min: 1, max: nil}, nil
	case qmarkToken:
		parser.advance()
		max := 1
		return quantifierExpr{target: target, min: 0, max: &max}, nil
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

	if !parser.match(dashToken) {
		max := firstValue
		return quantifierExpr{target: target, min: firstValue, max: &max}, nil
	}

	if tokenIn(parser.peek().typ, eofToken, rparenToken, pipeToken) || parser.startsDefinition() {
		return quantifierExpr{target: target, min: firstValue, max: nil}, nil
	}

	secondToken, err := parser.expect(numberToken)
	if err != nil {
		return nil, err
	}

	secondValue, convErr := parseInt(secondToken)
	if convErr != nil {
		return nil, convErr
	}
	if secondValue < firstValue {
		return nil, &lxError{
			msg: "invalid quantifier range",
			pos: secondToken.pos,
		}
	}

	return quantifierExpr{target: target, min: firstValue, max: &secondValue}, nil
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

		if length == 1 {
			items = append(items, classChar{value: item[0]})
		} else if length == 3 && item[1] == '-' {
			from, to := item[0], item[2]
			if numberCharRegex.MatchString(string(from)) && numberCharRegex.MatchString(string(to)) {
				if from >= to {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{from: from, to: to})
			} else if lowerCharRegex.MatchString(string(from)) && lowerCharRegex.MatchString(string(to)) {
				if int(from) >= int(to) {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{from: from, to: to})
			} else if upperCharRegex.MatchString(string(from)) && upperCharRegex.MatchString(string(to)) {
				if int(from) >= int(to) {
					return nil, getClassRangeError(tok, item, start)
				}
				items = append(items, classRange{from: from, to: to})
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
			if _, exists := idents[str]; !exists || str == "anychar" {
				return nil, &lxError{
					msg: "invalid ident in character class: " + str,
					pos: position{
						line:   tok.pos.line,
						column: tok.pos.column + start,
					},
				}
			}
			items = append(items, identExpr{name: str})
		}
	}

	return items, nil
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

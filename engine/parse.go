package engine

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

type builtinExpr struct {
	name string
}

func (builtinExpr) exprNode() {}

type numberExpr struct {
	value string
}

func (numberExpr) exprNode() {}

type literalValueExpr struct {
	value string
}

func (literalValueExpr) exprNode() {}

type classExpr struct {
	items []classItemExpr
}

func (classExpr) exprNode() {}

type groupExpr struct {
	expressions []expr
}

func (groupExpr) exprNode() {}

type captureExpr struct {
	name  string
	group groupExpr
}

func (captureExpr) exprNode() {}

type orExpr struct {
	group groupExpr
}

func (orExpr) exprNode() {}

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

type classItemExpr interface {
	classItemNode()
}

type literalExpr struct {
	value rune
}

func (literalExpr) classItemNode() {}

type atom struct {
	name string
}

func (atom) classItemNode() {}

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
		return token{typ: eof}
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
	expressions, err := parser.parseSequence(eof)
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

	if parser.peek().typ == variable && parser.lookahead(1).typ == equals {
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
	case ident:
		if currentToken.val == "capture" {
			return parser.parseCapture()
		}
		if currentToken.val == "or" {
			return parser.parseOr()
		}
		parser.advance()
		return identExpr{name: currentToken.val}, nil
	case variable:
		parser.advance()
		return variableExpr{name: currentToken.val}, nil
	case builtin:
		parser.advance()
		return builtinExpr{name: currentToken.val}, nil
	case literal:
		parser.advance()
		return literalValueExpr{value: currentToken.val}, nil
	case number:
		parser.advance()
		return numberExpr{value: currentToken.val}, nil
	case class:
		parser.advance()
		items, err := parseClassItems(currentToken)
		if err != nil {
			return nil, err
		}
		return classExpr{items: items}, nil
	case lparen:
		return parser.parseGroup()
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

	nameToken, err := parser.expect(variable)
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(equals); err != nil {
		return nil, err
	}

	expressions, err := parser.parseDefinitionValue()
	if err != nil {
		return nil, err
	}

	return defineExpr{
		name:  nameToken.val,
		value: groupExpr{expressions: expressions},
	}, nil
}

func (parser *parser) parsePatternSection() (expr, *lxError) {
	if parser.seenPatternSection {
		return nil, &lxError{
			msg: "'pattern:' can appear only once per program",
			pos: parser.peek().pos,
		}
	}

	patternToken, err := parser.expect(ident)
	if err != nil {
		return nil, err
	}
	if patternToken.val != "pattern" {
		return nil, &lxError{
			msg: "unexpected section label: " + patternToken.val,
			pos: patternToken.pos,
		}
	}

	if _, err := parser.expect(colon); err != nil {
		return nil, err
	}

	parser.seenPatternSection = true
	return patternSectionExpr{}, nil
}

func (parser *parser) parseDefinitionValue() ([]expr, *lxError) {
	var expressions []expr
	parser.inDefinitionValue++
	defer func() {
		parser.inDefinitionValue--
	}()

	for {
		currentToken := parser.peek()
		if currentToken.typ == eof || parser.startsDefinition() || parser.startsPatternSection() {
			return expressions, nil
		}

		expression, err := parser.parseExpr()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
	}
}

func (parser *parser) parseGroup() (expr, *lxError) {
	if _, err := parser.expect(lparen); err != nil {
		return nil, err
	}

	expressions, err := parser.parseSequence(rparen)
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(rparen); err != nil {
		return nil, err
	}

	return groupExpr{expressions: expressions}, nil
}

func (parser *parser) parseCapture() (expr, *lxError) {
	captureToken, err := parser.expect(ident)
	if err != nil {
		return nil, err
	}
	if captureToken.val != "capture" {
		return nil, &lxError{
			msg: "unexpected identifier: " + captureToken.val,
			pos: captureToken.pos,
		}
	}

	nameToken, err := parser.expect(ident)
	if err != nil {
		return nil, err
	}

	group, err := parser.parseRequiredGroup("capture")
	if err != nil {
		return nil, err
	}

	return captureExpr{name: nameToken.val, group: group}, nil
}

func (parser *parser) parseOr() (expr, *lxError) {
	orToken, err := parser.expect(ident)
	if err != nil {
		return nil, err
	}
	if orToken.val != "or" {
		return nil, &lxError{
			msg: "unexpected identifier: " + orToken.val,
			pos: orToken.pos,
		}
	}

	group, err := parser.parseRequiredGroup("or")
	if err != nil {
		return nil, err
	}

	return orExpr{group: group}, nil
}

func (parser *parser) parseRequiredGroup(context string) (groupExpr, *lxError) {
	if parser.peek().typ != lparen {
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

func (parser *parser) parseSequence(stop ...tokenType) ([]expr, *lxError) {
	var expressions []expr

	for {
		currentToken := parser.peek()
		if tokenIn(currentToken.typ, stop...) {
			return expressions, nil
		}
		if currentToken.typ == eof {
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

	if !isQuantifiable(target) {
		if tokenIn(currentToken.typ, star, plus, qmark, number) {
			return nil, &lxError{
				msg: "expression cannot be quantified",
				pos: currentToken.pos,
			}
		}
		return target, nil
	}

	switch currentToken.typ {
	case star:
		parser.advance()
		return quantifierExpr{target: target, min: 0, max: nil}, nil
	case plus:
		parser.advance()
		return quantifierExpr{target: target, min: 1, max: nil}, nil
	case qmark:
		parser.advance()
		max := 1
		return quantifierExpr{target: target, min: 0, max: &max}, nil
	case number:
		return parser.parseNumericQuantifier(target)
	default:
		return target, nil
	}
}

func (parser *parser) parseNumericQuantifier(target expr) (expr, *lxError) {
	firstToken, err := parser.expect(number)
	if err != nil {
		return nil, err
	}

	firstValue, convErr := parseInt(firstToken)
	if convErr != nil {
		return nil, convErr
	}

	if !parser.match(dash) {
		max := firstValue
		return quantifierExpr{target: target, min: firstValue, max: &max}, nil
	}

	if tokenIn(parser.peek().typ, eof, rparen) || parser.startsDefinition() {
		return quantifierExpr{target: target, min: firstValue, max: nil}, nil
	}

	secondToken, err := parser.expect(number)
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
	return parser.peek().typ == variable && parser.lookahead(1).typ == equals
}

func (parser *parser) startsPatternSection() bool {
	return parser.peek().typ == ident &&
		parser.peek().val == "pattern" &&
		parser.lookahead(1).typ == colon
}

func (parser *parser) lookahead(offset int) token {
	index := parser.pos + offset
	if index >= len(parser.tokens) {
		return token{typ: eof}
	}
	return parser.tokens[index]
}

func tokenIn(candidate tokenType, expected ...tokenType) bool {
	for _, item := range expected {
		if candidate == item {
			return true
		}
	}
	return false
}

func isQuantifiable(target expr) bool {
	switch target.(type) {
	case builtinExpr, captureExpr:
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

func parseClassItems(tok token) ([]classItemExpr, *lxError) {
	var items []classItemExpr
	content := []rune(tok.val)

	for i := 0; i < len(content); {
		if isClassWhitespace(content[i]) {
			i++
			continue
		}

		if content[i] == '\'' {
			i++
			for i < len(content) && content[i] != '\'' {
				items = append(items, literalExpr{value: content[i]})
				i++
			}
			if i >= len(content) {
				return nil, &lxError{
					msg: "unterminated quoted class literal",
					pos: tok.pos,
				}
			}
			i++
			continue
		}

		start := i
		for i < len(content) && !isClassWhitespace(content[i]) {
			i++
		}
		items = append(items, atom{name: string(content[start:i])})
	}

	return items, nil
}

func isClassWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

package engine

import (
	"regexp"
)

type tokenType int

const (
	identToken tokenType = iota
	notToken
	captureToken
	requireToken
	variableToken
	builtinToken
	literalToken
	classToken
	numberToken

	lparenToken
	rparenToken
	lbracketToken
	rbracketToken

	starToken
	plusToken
	qmarkToken
	pipeToken
	dashToken
	equalsToken
	colonToken

	eofToken
)

type token struct {
	typ tokenType
	val string
	pos position
}

type lexer struct {
	input []rune
	pos   int

	line   int
	column int
}

func newLexer(input string) *lexer {
	return &lexer{
		input:  []rune(input),
		line:   1,
		column: 1,
	}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	ch := l.peek()
	l.pos++

	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	return ch
}

func (l *lexer) makePos() position {
	return position{
		line:   l.line,
		column: l.column,
		offset: l.pos,
	}
}

func (l *lexer) lex() ([]token, *lxError) {
	identCharRegex := regexp.MustCompile("[a-zA-Z0-9_]")
	identRegex := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
	numberCharRegex := regexp.MustCompile("[0-9]")
	numberRegex := regexp.MustCompile("^[1-9][0-9]*$")
	whitespaceCharRegex := regexp.MustCompile(`\s`)

	var tokens []token

	for l.pos < len(l.input) {
		ch := l.peek()

		if ch == '#' {
			for {
				ch := l.peek()
				if ch == '\n' || ch == 0 {
					break
				}
				l.advance()
			}
			continue
		}

		// whitespace
		if whitespaceCharRegex.MatchString(string(ch)) {
			l.advance()
			continue
		}

		// number
		if numberCharRegex.MatchString(string(ch)) {
			start := l.makePos()

			for numberCharRegex.MatchString(string(l.peek())) {
				l.advance()
			}

			val := string(l.input[start.offset:l.pos])
			if !numberRegex.MatchString(val) {
				err := &lxError{
					msg: "invalid number: " + val,
					pos: start,
				}
				return tokens, err
			}

			tokens = append(tokens, token{
				typ: numberToken,
				val: val,
				pos: start,
			})
			continue
		}

		// ident
		if identCharRegex.MatchString(string(ch)) {
			start := l.makePos()

			for identCharRegex.MatchString(string(l.peek())) {
				l.advance()
			}

			val := string(l.input[start.offset:l.pos])
			if !identRegex.MatchString(val) {
				err := &lxError{
					msg: "invalid identifier: " + val,
					pos: start,
				}
				return tokens, err
			}

			tokens = append(tokens, token{
				typ: identToken,
				val: val,
				pos: start,
			})
			switch val {
			case "not":
				tokens[len(tokens)-1].typ = notToken
			case "capture":
				tokens[len(tokens)-1].typ = captureToken
			case "require":
				tokens[len(tokens)-1].typ = requireToken
			}
			continue
		}

		// variable $name
		if ch == '$' {
			l.advance()
			start := l.makePos()

			for identCharRegex.MatchString(string(l.peek())) {
				l.advance()
			}

			val := string(l.input[start.offset:l.pos])
			if !identRegex.MatchString(val) {
				err := &lxError{
					msg: "invalid variable name: " + val,
					pos: start,
				}
				return tokens, err
			}

			tokens = append(tokens, token{
				typ: variableToken,
				val: val,
				pos: start,
			})
			continue
		}

		// builtin @name
		if ch == '@' {
			l.advance()
			start := l.makePos()

			for identCharRegex.MatchString(string(l.peek())) {
				l.advance()
			}

			val := string(l.input[start.offset:l.pos])

			if _, ok := builtins[val]; !ok {
				err := &lxError{
					msg: "unknown builtin: " + val,
					pos: start,
				}
				return tokens, err
			}

			tokens = append(tokens, token{
				typ: builtinToken,
				val: val,
				pos: start,
			})
			continue
		}

		// string "..."
		if ch == '"' {
			l.advance()
			start := l.makePos()

			var val []rune

			for {
				ch := l.peek()
				if ch == 0 {
					err := &lxError{
						msg: "unterminated string literal",
						pos: l.makePos(),
					}
					return tokens, err
				}

				if ch == '"' {
					l.advance()
					break
				}

				if ch == '\\' {
					l.advance()
					esc := l.peek()

					switch esc {
					case '"':
						val = append(val, '"')
					case '\\':
						val = append(val, '\\')
					case 'n':
						val = append(val, '\n')
					case 't':
						val = append(val, '\t')
					case 'r':
						val = append(val, '\r')
					default:
						err := &lxError{
							msg: "invalid escape sequence: \\" + string(esc),
							pos: l.makePos(),
						}
						return tokens, err
					}

					l.advance()
					continue
				}

				val = append(val, ch)
				l.advance()
			}

			tokens = append(tokens, token{
				typ: literalToken,
				val: string(val),
				pos: start,
			})
			continue
		}

		// class
		if ch == '[' {
			l.advance()
			start := l.makePos()

			var val []rune

			for {
				ch := l.peek()
				if ch == 0 {
					err := &lxError{
						msg: "unterminated character class",
						pos: l.makePos(),
					}
					return tokens, err
				}

				if ch == ']' {
					l.advance()
					break
				}

				if ch == '\\' {
					l.advance()
					esc := l.peek()

					switch esc {
					case '[':
						val = append(val, '[')
					case ']':
						val = append(val, ']')
					case '\\':
						val = append(val, '\\')
					default:
						err := &lxError{
							msg: "invalid escape sequence in character class: \\" + string(esc),
							pos: l.makePos(),
						}
						return tokens, err
					}

					l.advance()
					continue
				}

				val = append(val, ch)
				l.advance()
			}

			tokens = append(tokens, token{
				typ: classToken,
				val: string(val),
				pos: start,
			})
			continue
		}

		// single char tokens
		start := l.makePos()

		switch ch {
		case '(':
			tokens = append(tokens, token{typ: lparenToken, pos: start})
		case ')':
			tokens = append(tokens, token{typ: rparenToken, pos: start})
		case '[':
			tokens = append(tokens, token{typ: lbracketToken, pos: start})
		case ']':
			tokens = append(tokens, token{typ: rbracketToken, pos: start})
		case '*':
			tokens = append(tokens, token{typ: starToken, pos: start})
		case '+':
			tokens = append(tokens, token{typ: plusToken, pos: start})
		case '?':
			tokens = append(tokens, token{typ: qmarkToken, pos: start})
		case '|':
			tokens = append(tokens, token{typ: pipeToken, pos: start})
		case '-':
			tokens = append(tokens, token{typ: dashToken, pos: start})
		case '=':
			tokens = append(tokens, token{typ: equalsToken, pos: start})
		case ':':
			tokens = append(tokens, token{typ: colonToken, pos: start})
		default:
			err := &lxError{
				msg: "unexpected character: " + string(ch),
				pos: l.makePos(),
			}
			return tokens, err
		}

		l.advance()
	}

	tokens = append(tokens, token{
		typ: eofToken,
		pos: l.makePos(),
	})

	return tokens, nil
}

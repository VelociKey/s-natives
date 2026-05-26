package lifecycle

const LexerGo = `package snparser

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError
	TokenIdent
	TokenString
	TokenNumber
	TokenEqual
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenSemicolon
	TokenSlash
	TokenMinus
	TokenAnd
	TokenNot
	TokenColon
	TokenComma
	TokenDocBlock
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

type Lexer struct {
	input     string
	pos       int
	width     int
	line      int
	lineStart int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, line: 1}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	start := l.pos

	r := l.next()
	if r == -1 {
		return Token{Type: TokenEOF, Line: l.line, Col: l.col(start)}
	}

	switch r {
	case '=': return l.emit(TokenEqual, start)
	case '{': return l.emit(TokenLBrace, start)
	case '}': return l.emit(TokenRBrace, start)
	case '[': return l.emit(TokenLBracket, start)
	case ']': return l.emit(TokenRBracket, start)
	case '(': return l.emit(TokenLParen, start)
	case ')': return l.emit(TokenRParen, start)
	case ';': return l.emit(TokenSemicolon, start)
	case ':': return l.emit(TokenColon, start)
	case ',': return l.emit(TokenComma, start)
	case '&': return l.emit(TokenAnd, start)
	case '!': return l.emit(TokenNot, start)
	case '^': return l.emit(TokenIdent, start)
	case '/':
		if l.peek() == '*' {
			return lexDocBlock(l, start)
		}
		return l.emit(TokenSlash, start)
	case '-': return l.emit(TokenMinus, start)
	case '"', '\'':
		return lexString(l, r, start)
	default:
		if unicode.IsDigit(r) {
			return lexNumber(l, start)
		}
		if isLetter(r) {
			return lexIdentifier(l, start)
		}
		return l.errorf("unexpected character: %q", r)
	}
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return -1
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += w
	if r == '\n' {
		l.line++
		l.lineStart = l.pos
	}
	return r
}

func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *Lexer) backup() {
	l.pos -= l.width
}

func (l *Lexer) emit(t TokenType, start int) Token {
	return Token{
		Type:    t,
		Literal: l.input[start:l.pos],
		Line:    l.line,
		Col:     l.col(start),
	}
}

func (l *Lexer) col(pos int) int {
	return pos - l.lineStart + 1
}

func (l *Lexer) skipWhitespace() {
	for {
		r := l.next()
		if r == -1 { return }
		if r == '#' { // Line comment
			for {
				r2 := l.next()
				if r2 == -1 || r2 == '\n' { break }
			}
			continue
		}
		if r == '/' && l.peek() == '/' { // // Line comment
			l.next() // consume second '/'
			for {
				r2 := l.next()
				if r2 == -1 || r2 == '\n' { break }
			}
			continue
		}
		if r == ';' {
			// Check if ';' is the first non-whitespace character on this line
			isComment := true
			limit := l.pos - l.width
			for i := l.lineStart; i < limit; i++ {
				c := l.input[i]
				if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
					isComment = false
					break
				}
			}
			if isComment {
				// Only treat as comment if there is some non-whitespace content after it on the same line
				hasContent := false
				for i := l.pos; i < len(l.input); i++ {
					c := l.input[i]
					if c == '\n' {
						break
					}
					if c != ' ' && c != '\t' && c != '\r' {
						hasContent = true
						break
					}
				}
				if hasContent {
					for {
						r2 := l.next()
						if r2 == -1 || r2 == '\n' { break }
					}
					continue
				}
			}
		}
		if !unicode.IsSpace(r) { l.backup(); return }
	}
}

func (l *Lexer) errorf(format string, args ...interface{}) Token {
	return Token{
		Type:    TokenError,
		Literal: fmt.Sprintf(format, args...),
		Line:    l.line,
		Col:     l.pos - l.lineStart + 1,
	}
}

func lexIdentifier(l *Lexer, start int) Token {
	for {
		r := l.next()
		if r == -1 { break }
		if !isLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' && r != '/' && r != '@' {
			l.backup()
			break
		}
	}
	return l.emit(TokenIdent, start)
}

func lexNumber(l *Lexer, start int) Token {
	for {
		r := l.next()
		if r == -1 { break }
		if !unicode.IsDigit(r) {
			l.backup()
			break
		}
	}
	return l.emit(TokenNumber, start)
}

func lexString(l *Lexer, quote rune, start int) Token {
	for {
		r := l.next()
		if r == -1 { return l.errorf("unterminated string") }
		if r == '\\' { l.next(); continue }
		if r == quote { break }
	}
	return l.emit(TokenString, start)
}

func lexDocBlock(l *Lexer, start int) Token {
	l.next() // consume '*'
	for {
		r := l.next()
		if r == -1 { return l.errorf("unterminated doc block") }
		if r == '*' && l.peek() == '/' {
			l.next() // consume '/'
			break
		}
	}
	return l.emit(TokenDocBlock, start)
}

func isLetter(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
`

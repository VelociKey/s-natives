package lifecycle

const ParserGo = `package snparser

import (
	"fmt"
)

type Parser struct {
	lexer *Lexer
	curr  Token
	peek  Token
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{lexer: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curr = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) Parse() (*GrammarNode, error) {
	g := &GrammarNode{Rules: make(map[string]*RuleNode)}
	
	// Handle Header
	if p.curr.Type == TokenColon {
		p.nextToken()
		for p.curr.Type == TokenIdent {
			g.Header = append(g.Header, p.curr.Literal)
			p.nextToken()
			if p.curr.Type == TokenColon { p.nextToken() } else { break }
		}
	}

	for p.curr.Type != TokenEOF {
		if p.curr.Type == TokenSemicolon {
			p.nextToken()
			continue
		}

		// Handle Directives: e.g. import, export, legal, or @
		if p.curr.Type == TokenIdent && (p.curr.Literal == "import" || p.curr.Literal == "export" || p.curr.Literal == "legal" || p.curr.Literal == "@") {
			// Consume until semicolon
			for p.curr.Type != TokenSemicolon && p.curr.Type != TokenEOF {
				if p.curr.Type == TokenError {
					return nil, fmt.Errorf("lexical error in directive at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
				}
				p.nextToken()
			}
			if p.curr.Type == TokenSemicolon {
				p.nextToken()
			}
			continue
		}

		var doc string
		if p.curr.Type == TokenDocBlock {
			doc = p.curr.Literal
			p.nextToken()
		}

		if p.curr.Type == TokenIdent {
			name := p.curr.Literal
			p.nextToken()
			if p.curr.Type == TokenEqual {
				p.nextToken()
				var expr Node
				var err error
				if p.isFlatBlock() {
					expr, err = p.parseFlatBlock()
				} else {
					expr, err = p.parseExpression()
				}
				if err != nil { return nil, err }
				g.Rules[name] = &RuleNode{Name: name, Doc: doc, Expr: expr}
				if p.curr.Type == TokenSemicolon { p.nextToken() }
			} else {
				return nil, fmt.Errorf("expected '=' after rule identifier %q, got %q at line %d, col %d", name, p.curr.Literal, p.curr.Line, p.curr.Col)
			}
		} else {
			if p.curr.Type == TokenError {
				return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
			}
			return nil, fmt.Errorf("unexpected token %q of type %d at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
		}
	}
	return g, nil
}

func (p *Parser) isFlatBlock() bool {
	if p.curr.Type != TokenLBrace {
		return false
	}
	lexerCopy := *p.lexer
	tempParser := &Parser{
		lexer: &lexerCopy,
		curr:  p.curr,
		peek:  p.peek,
	}
	braceDepth := 1
	for {
		tempParser.nextToken()
		t := tempParser.curr
		if t.Type == TokenEOF || t.Type == TokenError {
			break
		}
		if t.Type == TokenLBrace {
			braceDepth++
		} else if t.Type == TokenRBrace {
			braceDepth--
			if braceDepth == 0 {
				break
			}
		} else if braceDepth == 1 {
			if t.Type == TokenEqual || t.Type == TokenSemicolon {
				return true
			}
		}
	}
	return false
}

func (p *Parser) parseFlatBlock() (Node, error) {
	p.nextToken() // {
	var rules []Node
	for p.curr.Type != TokenRBrace && p.curr.Type != TokenEOF {
		if p.curr.Type == TokenError {
			return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
		}
		var doc string
		if p.curr.Type == TokenDocBlock {
			doc = p.curr.Literal
			p.nextToken()
		}
		if p.curr.Type == TokenIdent {
			name := p.curr.Literal
			p.nextToken()
			if p.curr.Type == TokenEqual {
				p.nextToken()
				expr, err := p.parseExpression()
				if err != nil { return nil, err }
				rules = append(rules, &RuleNode{Name: name, Doc: doc, Expr: expr})
				if p.curr.Type == TokenSemicolon { p.nextToken() }
			} else {
				return nil, fmt.Errorf("expected '=' after identifier %q in FlatBlock, got %q at line %d, col %d", name, p.curr.Literal, p.curr.Line, p.curr.Col)
			}
		} else {
			if p.curr.Type == TokenError {
				return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
			}
			return nil, fmt.Errorf("unexpected token %q of type %d in FlatBlock at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
		}
	}
	if p.curr.Type != TokenRBrace { return nil, fmt.Errorf("missing } in FlatBlock") }
	p.nextToken()
	return &SequenceNode{Elements: rules}, nil
}
`

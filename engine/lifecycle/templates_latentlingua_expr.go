package lifecycle

const ExprGo = `package snparser

import (
	"fmt"
	"strconv"
	"unicode"
)

func (p *Parser) parseExpression() (Node, error) {
	type parserFrameType int

	const (
		frameChoice parserFrameType = iota
		frameSequence
		frameGroup
		frameOptional
		frameRepetition
		framePredicate
	)

	type parserFrame struct {
		kind      parserFrameType
		nodes     []Node
		positive  bool
		startLine int
		startCol  int
	}

	var stack []*parserFrame

	// Initially, we push a Choice frame (the root expression) and a Sequence frame.
	stack = append(stack, &parserFrame{
		kind:      frameChoice,
		startLine: p.curr.Line,
		startCol:  p.curr.Col,
	})
	stack = append(stack, &parserFrame{
		kind:      frameSequence,
		startLine: p.curr.Line,
		startCol:  p.curr.Col,
	})

	for {
		if p.curr.Type == TokenError {
			return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
		}

		// Check if we can parse a factor
		isFactor := false
		switch p.curr.Type {
		case TokenIdent, TokenString, TokenLParen, TokenLBracket, TokenLBrace, TokenAnd, TokenNot:
			isFactor = true
		}

		if isFactor {
			var completedNode Node
			switch p.curr.Type {
			case TokenIdent:
				completedNode = &TerminalNode{Value: p.curr.Literal, IsRef: true}
				p.nextToken()
			case TokenString:
				completedNode = &TerminalNode{Value: p.curr.Literal, IsRef: false}
				p.nextToken()
			case TokenAnd, TokenNot:
				stack = append(stack, &parserFrame{
					kind:      framePredicate,
					positive:  p.curr.Type == TokenAnd,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				continue
			case TokenLParen:
				stack = append(stack, &parserFrame{
					kind:      frameGroup,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				stack = append(stack, &parserFrame{
					kind:      frameChoice,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			case TokenLBrace:
				stack = append(stack, &parserFrame{
					kind:      frameRepetition,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				stack = append(stack, &parserFrame{
					kind:      frameChoice,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			case TokenLBracket:
				p.nextToken() // consume '['
				negated := false
				if p.curr.Type == TokenIdent && p.curr.Literal == "^" {
					negated = true
					p.nextToken() // consume '^'
				}

				isRange := false
				if p.curr.Type == TokenIdent {
					if len(p.curr.Literal) == 3 && p.curr.Literal[1] == '-' {
						isRange = true
					} else if len(p.curr.Literal) == 1 {
						r := rune(p.curr.Literal[0])
						if unicode.IsLower(r) || unicode.IsDigit(r) {
							isRange = true
						}
					}
				} else if p.curr.Type == TokenNumber {
					if p.peek.Type == TokenMinus || p.peek.Type == TokenRBracket {
						isRange = true
					}
				}

				if isRange {
					var start, end rune
					if p.curr.Type == TokenIdent && len(p.curr.Literal) == 3 && p.curr.Literal[1] == '-' {
						start = rune(p.curr.Literal[0])
						end = rune(p.curr.Literal[2])
						p.nextToken()
					} else if p.curr.Type == TokenNumber && p.peek.Type == TokenMinus {
						start = rune(p.curr.Literal[0])
						p.nextToken() // consume start digit
						p.nextToken() // consume '-'
						if p.curr.Type != TokenNumber && p.curr.Type != TokenIdent {
							return nil, fmt.Errorf("expected end of range after '-', got %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
						}
						end = rune(p.curr.Literal[0])
						p.nextToken() // consume end
					} else if p.curr.Type == TokenIdent && len(p.curr.Literal) == 1 {
						start = rune(p.curr.Literal[0])
						end = start
						p.nextToken()
					} else if p.curr.Type == TokenNumber {
						start = rune(p.curr.Literal[0])
						end = start
						p.nextToken()
					}

					if p.curr.Type != TokenRBracket {
						return nil, fmt.Errorf("missing ] in character range, got %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
					}
					p.nextToken() // consume ']'
					completedNode = &RangeNode{Start: start, End: end, Negated: negated}
				} else {
					if negated {
						return nil, fmt.Errorf("unexpected '^' at start of optional expression at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					stack = append(stack, &parserFrame{
						kind:      frameOptional,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					stack = append(stack, &parserFrame{
						kind:      frameChoice,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					stack = append(stack, &parserFrame{
						kind:      frameSequence,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					continue
				}
			}

			// Reduce the completed factor node:
			for {
				if len(stack) == 0 {
					return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
				}
				top := stack[len(stack)-1]
				if top.kind == framePredicate {
					stack = stack[:len(stack)-1]
					completedNode = &PredicateNode{
						Expr:     completedNode,
						Positive: top.positive,
					}
					continue
				}
				if top.kind == frameSequence {
					top.nodes = append(top.nodes, completedNode)
					break
				}
				return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
			}

		} else {
			// Current token is not a factor-starting token.
			// The current sequence is complete.
			if len(stack) == 0 {
				return nil, fmt.Errorf("internal parser error: empty stack when closing sequence at line %d, col %d", p.curr.Line, p.curr.Col)
			}
			seqFrame := stack[len(stack)-1]
			if seqFrame.kind != frameSequence {
				return nil, fmt.Errorf("internal parser error: expected sequence frame on top of stack, got %d at line %d, col %d", seqFrame.kind, p.curr.Line, p.curr.Col)
			}
			stack = stack[:len(stack)-1]

			var seqNode Node
			if len(seqFrame.nodes) == 1 {
				seqNode = seqFrame.nodes[0]
			} else {
				seqNode = &SequenceNode{Elements: seqFrame.nodes}
			}

			// Look at parent frame
			if len(stack) == 0 {
				return nil, fmt.Errorf("internal parser error: empty stack after popping sequence at line %d, col %d", p.curr.Line, p.curr.Col)
			}
			parentFrame := stack[len(stack)-1]
			if parentFrame.kind != frameChoice {
				return nil, fmt.Errorf("internal parser error: expected choice frame below sequence, got %d at line %d, col %d", parentFrame.kind, p.curr.Line, p.curr.Col)
			}

			parentFrame.nodes = append(parentFrame.nodes, seqNode)

			if p.curr.Type == TokenSlash {
				p.nextToken() // consume '/'
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			}

			// Choice is complete! Pop choice frame.
			stack = stack[:len(stack)-1]

			var choiceNode Node
			if len(parentFrame.nodes) == 1 {
				choiceNode = parentFrame.nodes[0]
			} else {
				choiceNode = &ChoiceNode{Options: parentFrame.nodes}
			}

			if len(stack) == 0 {
				// Completed the root choice.
				// Semicolon, EOF, or RBrace are valid terminators.
				if p.curr.Type == TokenSemicolon || p.curr.Type == TokenEOF || p.curr.Type == TokenRBrace {
					return choiceNode, nil
				}
				if p.curr.Type == TokenRParen || p.curr.Type == TokenRBracket {
					return nil, fmt.Errorf("unmatched closing delimiter %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
				}
				return nil, fmt.Errorf("unexpected token %q of type %d at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
			}

			grandparentFrame := stack[len(stack)-1]
			switch grandparentFrame.kind {
			case frameGroup:
				if p.curr.Type != TokenRParen {
					return nil, fmt.Errorf("missing )")
				}
				p.nextToken() // consume ')'
				stack = stack[:len(stack)-1] // pop frameGroup

				completedNode := choiceNode
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			case frameOptional:
				if p.curr.Type != TokenRBracket {
					return nil, fmt.Errorf("missing ] in optional expression, got %q", p.curr.Literal)
				}
				p.nextToken() // consume ']'
				stack = stack[:len(stack)-1] // pop frameOptional

				var completedNode Node = &OptionalNode{Expr: choiceNode}
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			case frameRepetition:
				if p.curr.Type != TokenRBrace {
					return nil, fmt.Errorf("missing }")
				}
				p.nextToken() // consume '}'
				stack = stack[:len(stack)-1] // pop frameRepetition

				min, max := 0, -1
				if p.curr.Type == TokenNumber {
					min, _ = strconv.Atoi(p.curr.Literal)
					p.nextToken()
					if p.curr.Type == TokenComma {
						p.nextToken()
						if p.curr.Type == TokenNumber {
							max, _ = strconv.Atoi(p.curr.Literal)
							p.nextToken()
						}
					} else {
						max = min
					}
				}

				var completedNode Node = &RepetitionNode{Expr: choiceNode, Min: min, Max: max}
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			default:
				return nil, fmt.Errorf("internal parser error: unexpected grandparent frame type %d at line %d, col %d", grandparentFrame.kind, p.curr.Line, p.curr.Col)
			}
		}
	}
}
`

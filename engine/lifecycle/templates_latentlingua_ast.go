package lifecycle

const AstGo = `package snparser

// AST Nodes
type Node interface { Kind() string }

type GrammarNode struct {
	Header []string
	Rules  map[string]*RuleNode
}
func (g *GrammarNode) Kind() string { return "Grammar" }

type RuleNode struct {
	Name string
	Doc  string
	Expr Node
}
func (r *RuleNode) Kind() string { return "Rule" }

type SequenceNode struct { Elements []Node }
func (s *SequenceNode) Kind() string { return "Sequence" }

type ChoiceNode struct { Options []Node }
func (c *ChoiceNode) Kind() string { return "Choice" }

type RepetitionNode struct {
	Expr Node
	Min  int
	Max  int
}
func (r *RepetitionNode) Kind() string { return "Repetition" }

type OptionalNode struct { Expr Node }
func (o *OptionalNode) Kind() string { return "Optional" }

type TerminalNode struct {
	Value string
	IsRef bool
}
func (t *TerminalNode) Kind() string { return "Terminal" }

type RangeNode struct {
	Start   rune
	End     rune
	Negated bool
}
func (r *RangeNode) Kind() string { return "Range" }

type PredicateNode struct {
	Expr     Node
	Positive bool
}
func (p *PredicateNode) Kind() string { return "Predicate" }
`

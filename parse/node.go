package parse

import "fmt"

type Node interface {
	Type() NodeType
	Position() *Pos
	// Print(w io.Writer) error // Recursive, formatted printing
	String() string // A non-recursive string representation
	Children() []Node
}

type NodeType int

func (t NodeType) Type() NodeType { return t }

func (p *Pos) Position() *Pos { return p }

const (
	NodeBool NodeType = iota
	NodeCharacter
	NodeComment
	NodeDeref
	NodeKeyword
	NodeList
	NodeMap
	NodeMetadata
	NodeNil
	NodeNumber
	NodeQuote // 'form
	NodeString
	NodeSymbol
	NodeSyntaxQuote   // `form
	NodeUnquote       // ~form
	NodeUnquoteSplice // ~@form
	NodeVector

	// Dispatch macro forms.
	NodeFnLiteral  // #(...)
	NodeIgnoreForm // #_(...)
	NodeRegex
	NodeSet
	NodeVarQuote
	NodeTag // #foo/bar
)

// Nodes

type BoolNode struct {
	NodeType
	*Pos
	Val bool
}

func (b *BoolNode) String() string {
	if b.Val {
		return "true"
	}
	return "false"
}

func (b *BoolNode) Children() []Node { return nil }

type CharacterNode struct {
	NodeType
	*Pos
	Val rune
}

func (c *CharacterNode) String() string   { return fmt.Sprintf("char(%q)", c.Val) }
func (c *CharacterNode) Children() []Node { return nil }

type DerefNode struct {
	NodeType
	*Pos
	Node Node
}

func (d *DerefNode) String() string   { return "deref" }
func (d *DerefNode) Children() []Node { return []Node{d.Node} }

type ListNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (l *ListNode) String() string   { return fmt.Sprintf("list(length=%d)", len(l.Nodes)) }
func (l *ListNode) Children() []Node { return l.Nodes }

type NilNode struct {
	NodeType
	*Pos
}

func (n *NilNode) String() string   { return "nil" }
func (n *NilNode) Children() []Node { return nil }

type SymbolNode struct {
	NodeType
	*Pos
	Val string
}

func (s *SymbolNode) String() string   { return "sym(" + s.Val + ")" }
func (s *SymbolNode) Children() []Node { return nil }

type QuoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (d *QuoteNode) String() string   { return "quote" }
func (d *QuoteNode) Children() []Node { return []Node{d.Node} }

type SyntaxQuoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (d *SyntaxQuoteNode) String() string   { return "syntax quote" }
func (d *SyntaxQuoteNode) Children() []Node { return []Node{d.Node} }

type UnquoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (d *UnquoteNode) String() string   { return "unquote" }
func (d *UnquoteNode) Children() []Node { return []Node{d.Node} }

type UnquoteSpliceNode struct {
	NodeType
	*Pos
	Node Node
}

func (d *UnquoteSpliceNode) String() string   { return "unquote splice" }
func (d *UnquoteSpliceNode) Children() []Node { return []Node{d.Node} }

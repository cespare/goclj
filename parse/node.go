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
	NodeMetadata // ^form
	NodeNil
	NodeNumber
	NodeQuote // 'form
	NodeString
	NodeSymbol
	NodeSyntaxQuote   // `form
	NodeUnquote       // ~form
	NodeUnquoteSplice // ~@form
	NodeVector

	NodeLambdaArg

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

func (n *BoolNode) String() string {
	if n.Val {
		return "true"
	}
	return "false"
}

func (n *BoolNode) Children() []Node { return nil }

type CharacterNode struct {
	NodeType
	*Pos
	Val rune
}

func (n *CharacterNode) String() string   { return fmt.Sprintf("char(%q)", n.Val) }
func (n *CharacterNode) Children() []Node { return nil }

type DerefNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *DerefNode) String() string   { return "deref" }
func (n *DerefNode) Children() []Node { return []Node{n.Node} }

type KeywordNode struct {
	NodeType
	*Pos
	Val string
}

func (n *KeywordNode) String() string   { return fmt.Sprintf("keyword(%s)", n.Val) }
func (n *KeywordNode) Children() []Node { return nil }

type ListNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (n *ListNode) String() string   { return fmt.Sprintf("list(length=%d)", len(n.Nodes)) }
func (n *ListNode) Children() []Node { return n.Nodes }

type MapNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (n *MapNode) String() string   { return fmt.Sprintf("map(length=%d)", len(n.Nodes)/2) }
func (n *MapNode) Children() []Node { return n.Nodes }

type MetadataNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *MetadataNode) String() string   { return "metadata" }
func (n *MetadataNode) Children() []Node { return []Node{n.Node} }

type NilNode struct {
	NodeType
	*Pos
}

func (n *NilNode) String() string   { return "nil" }
func (n *NilNode) Children() []Node { return nil }

type NumberNode struct {
	NodeType
	*Pos
	Val string
}

func (n *NumberNode) String() string   { return fmt.Sprintf("num(%s)", n.Val) }
func (n *NumberNode) Children() []Node { return nil }

type SymbolNode struct {
	NodeType
	*Pos
	Val string
}

func (n *SymbolNode) String() string   { return "sym(" + n.Val + ")" }
func (n *SymbolNode) Children() []Node { return nil }

type QuoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *QuoteNode) String() string   { return "quote" }
func (n *QuoteNode) Children() []Node { return []Node{n.Node} }

type StringNode struct {
	NodeType
	*Pos
	Val string
}

func (n *StringNode) String() string   { return fmt.Sprintf("string(%q)", n.Val) }
func (n *StringNode) Children() []Node { return nil }

type SyntaxQuoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *SyntaxQuoteNode) String() string   { return "syntax quote" }
func (n *SyntaxQuoteNode) Children() []Node { return []Node{n.Node} }

type UnquoteNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *UnquoteNode) String() string   { return "unquote" }
func (n *UnquoteNode) Children() []Node { return []Node{n.Node} }

type UnquoteSpliceNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *UnquoteSpliceNode) String() string   { return "unquote splice" }
func (n *UnquoteSpliceNode) Children() []Node { return []Node{n.Node} }

type VectorNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (n *VectorNode) String() string   { return fmt.Sprintf("vector(length=%d)", len(n.Nodes)) }
func (n *VectorNode) Children() []Node { return n.Nodes }

type LambdaArgNode struct {
	NodeType
	*Pos
	Val string
}

func (n *LambdaArgNode) String() string   { return fmt.Sprintf("lambda-arg(%s)", n.Val) }
func (n *LambdaArgNode) Children() []Node { return nil }

type FnLiteralNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (n *FnLiteralNode) String() string   { return fmt.Sprintf("lambda(length=%d)", len(n.Nodes)) }
func (n *FnLiteralNode) Children() []Node { return n.Nodes }

type IgnoreFormNode struct {
	NodeType
	*Pos
	Node Node
}

func (n *IgnoreFormNode) String() string   { return "ignore" }
func (n *IgnoreFormNode) Children() []Node { return []Node{n.Node} }

type RegexNode struct {
	NodeType
	*Pos
	Val string
}

func (n *RegexNode) String() string   { return fmt.Sprintf("regex(%q)", n.Val) }
func (n *RegexNode) Children() []Node { return nil }

type SetNode struct {
	NodeType
	*Pos
	Nodes []Node
}

func (n *SetNode) String() string   { return fmt.Sprintf("set(length=%d)", len(n.Nodes)) }
func (n *SetNode) Children() []Node { return n.Nodes }

type VarQuoteNode struct {
	NodeType
	*Pos
	Val string
}

func (n *VarQuoteNode) String() string   { return fmt.Sprintf("varquote(%s)", n.Val) }
func (n *VarQuoteNode) Children() []Node { return nil }

type TagNode struct {
	NodeType
	*Pos
	Val string
}

func (n *TagNode) String() string   { return fmt.Sprintf("tag(%s)", n.Val) }
func (n *TagNode) Children() []Node { return nil }

package parse

type Node interface {
	Type() NodeType
	Pos() Pos
	String() string
}

type NodeType int

func (t NodeType) Type() NodeType { return t }

func (p Pos) Pos() Pos { return p }

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
	Pos
	Val bool
}

func (b *BoolNode) String() string {
	if b.Val {
		return "true"
	}
	return "false"
}

type CharacterNode struct {
	NodeType
	Pos
	Val rune
	Text string
}

func (c *CharacterNode) String() string {
	return c.Text
}

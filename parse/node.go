package parse

import "fmt"

/*
BoolNode
CharacterNode
CommentNode
DerefNode
FnLiteralNode
IgnoreFormNode
KeywordNode
ListNode
MapNode
MetadataNode
NewlineNode
NilNode
NumberNode
QuoteNode
RegexNode
SetNode
StringNode
SymbolNode
SyntaxQuoteNode
TagNode
UnquoteNode
UnquoteSpliceNode
VarQuoteNode
VectorNode
*/

type Node interface {
	Position() *Pos
	String() string // A non-recursive string representation
	Children() []Node
	SetChildren([]Node)
}

func (p *Pos) Position() *Pos { return p }

type BoolNode struct {
	*Pos
	Val bool
}

func (n *BoolNode) String() string {
	if n.Val {
		return "true"
	}
	return "false"
}

func (n *BoolNode) Children() []Node   { return nil }
func (n *BoolNode) SetChildren([]Node) { panic("SetChildren called on BoolNode") }

type CharacterNode struct {
	*Pos
	Val  rune
	Text string
}

func (n *CharacterNode) String() string     { return fmt.Sprintf("char(%q)", n.Val) }
func (n *CharacterNode) Children() []Node   { return nil }
func (n *CharacterNode) SetChildren([]Node) { panic("SetChildren called on CharacterNode") }

type CommentNode struct {
	*Pos
	Text string
}

func (n *CommentNode) String() string     { return fmt.Sprintf("comment(%q)", n.Text) }
func (n *CommentNode) Children() []Node   { return nil }
func (n *CommentNode) SetChildren([]Node) { panic("SetChildren called on CommentNode") }

type DerefNode struct {
	*Pos
	Node Node
}

func (n *DerefNode) String() string   { return "deref" }
func (n *DerefNode) Children() []Node { return []Node{n.Node} }
func (n *DerefNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on DerefNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type KeywordNode struct {
	*Pos
	Val string
}

func (n *KeywordNode) String() string     { return fmt.Sprintf("keyword(%s)", n.Val) }
func (n *KeywordNode) Children() []Node   { return nil }
func (n *KeywordNode) SetChildren([]Node) { panic("SetChildren called on KeywordNode") }

type ListNode struct {
	*Pos
	Nodes []Node
}

func (n *ListNode) String() string {
	return fmt.Sprintf("list(length=%d)", countSemantic(n.Nodes))
}
func (n *ListNode) Children() []Node         { return n.Nodes }
func (n *ListNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type MapNode struct {
	*Pos
	Nodes []Node
}

func (n *MapNode) String() string {
	semanticNodes := countSemantic(n.Nodes)
	return fmt.Sprintf("map(length=%d)", semanticNodes/2)
}
func (n *MapNode) Children() []Node         { return n.Nodes }
func (n *MapNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type MetadataNode struct {
	*Pos
	Node Node
}

func (n *MetadataNode) String() string   { return "metadata" }
func (n *MetadataNode) Children() []Node { return []Node{n.Node} }
func (n *MetadataNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on MetadataNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type NewlineNode struct {
	*Pos
}

func (n *NewlineNode) String() string     { return "newline" }
func (n *NewlineNode) Children() []Node   { return nil }
func (n *NewlineNode) SetChildren([]Node) { panic("SetChildren called on NewlineNode") }

type NilNode struct {
	*Pos
}

func (n *NilNode) String() string     { return "nil" }
func (n *NilNode) Children() []Node   { return nil }
func (n *NilNode) SetChildren([]Node) { panic("SetChildren called on NilNode") }

type NumberNode struct {
	*Pos
	Val string
}

func (n *NumberNode) String() string     { return fmt.Sprintf("num(%s)", n.Val) }
func (n *NumberNode) Children() []Node   { return nil }
func (n *NumberNode) SetChildren([]Node) { panic("SetChildren called on NumberNode") }

type SymbolNode struct {
	*Pos
	Val string
}

func (n *SymbolNode) String() string     { return "sym(" + n.Val + ")" }
func (n *SymbolNode) Children() []Node   { return nil }
func (n *SymbolNode) SetChildren([]Node) { panic("SetChildren called on SymbolNode") }

type QuoteNode struct {
	*Pos
	Node Node
}

func (n *QuoteNode) String() string   { return "quote" }
func (n *QuoteNode) Children() []Node { return []Node{n.Node} }
func (n *QuoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on QuoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type StringNode struct {
	*Pos
	Val string
}

func (n *StringNode) String() string     { return fmt.Sprintf("string(%q)", n.Val) }
func (n *StringNode) Children() []Node   { return nil }
func (n *StringNode) SetChildren([]Node) { panic("SetChildren called on StringNode") }

type SyntaxQuoteNode struct {
	*Pos
	Node Node
}

func (n *SyntaxQuoteNode) String() string   { return "syntax quote" }
func (n *SyntaxQuoteNode) Children() []Node { return []Node{n.Node} }
func (n *SyntaxQuoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on SyntaxQuoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type UnquoteNode struct {
	*Pos
	Node Node
}

func (n *UnquoteNode) String() string   { return "unquote" }
func (n *UnquoteNode) Children() []Node { return []Node{n.Node} }
func (n *UnquoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on UnquoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type UnquoteSpliceNode struct {
	*Pos
	Node Node
}

func (n *UnquoteSpliceNode) String() string   { return "unquote splice" }
func (n *UnquoteSpliceNode) Children() []Node { return []Node{n.Node} }
func (n *UnquoteSpliceNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on UnquoteSpliceNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type VectorNode struct {
	*Pos
	Nodes []Node
}

func (n *VectorNode) String() string {
	return fmt.Sprintf("vector(length=%d)", countSemantic(n.Nodes))
}
func (n *VectorNode) Children() []Node         { return n.Nodes }
func (n *VectorNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type FnLiteralNode struct {
	*Pos
	Nodes []Node
}

func (n *FnLiteralNode) String() string {
	return fmt.Sprintf("lambda(length=%d)", countSemantic(n.Nodes))
}
func (n *FnLiteralNode) Children() []Node         { return n.Nodes }
func (n *FnLiteralNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type IgnoreFormNode struct {
	*Pos
	Node Node
}

func (n *IgnoreFormNode) String() string   { return "ignore" }
func (n *IgnoreFormNode) Children() []Node { return []Node{n.Node} }
func (n *IgnoreFormNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on IgnoreFormNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type RegexNode struct {
	*Pos
	Val string
}

func (n *RegexNode) String() string     { return fmt.Sprintf("regex(%q)", n.Val) }
func (n *RegexNode) Children() []Node   { return nil }
func (n *RegexNode) SetChildren([]Node) { panic("SetChildren called on RegexNode") }

type SetNode struct {
	*Pos
	Nodes []Node
}

func (n *SetNode) String() string {
	return fmt.Sprintf("set(length=%d)", countSemantic(n.Nodes))
}
func (n *SetNode) Children() []Node         { return n.Nodes }
func (n *SetNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type VarQuoteNode struct {
	*Pos
	Val string
}

func (n *VarQuoteNode) String() string     { return fmt.Sprintf("varquote(%s)", n.Val) }
func (n *VarQuoteNode) Children() []Node   { return nil }
func (n *VarQuoteNode) SetChildren([]Node) { panic("SetChildren called on VarQuoteNode") }

type TagNode struct {
	*Pos
	Val string
}

func (n *TagNode) String() string     { return fmt.Sprintf("tag(%s)", n.Val) }
func (n *TagNode) Children() []Node   { return nil }
func (n *TagNode) SetChildren([]Node) { panic("SetChildren called on TagNode") }

func isSemantic(n Node) bool {
	switch n.(type) {
	case *CommentNode, *NewlineNode:
		return false
	}
	return true
}

func countSemantic(nodes []Node) int {
	count := 0
	for _, node := range nodes {
		if isSemantic(node) {
			count++
		}
	}
	return count
}

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

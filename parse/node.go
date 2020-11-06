package parse

import "fmt"

type Node interface {
	Position() *Pos
	String() string // A non-recursive string representation
	Parent() Node   // nil if Node is a root node
	SetParent(Node)
	Children() []Node
	SetChildren([]Node)
}

func (p *Pos) Position() *Pos { return p }

type BoolNode struct {
	*Pos
	parent Node
	Val    bool
}

func (n *BoolNode) String() string {
	if n.Val {
		return "true"
	}
	return "false"
}

func (n *BoolNode) Parent() Node       { return n.parent }
func (n *BoolNode) SetParent(p Node)   { n.parent = p }
func (n *BoolNode) Children() []Node   { return nil }
func (n *BoolNode) SetChildren([]Node) { panic("SetChildren called on BoolNode") }

type CharacterNode struct {
	*Pos
	parent Node
	Val    rune
	Text   string
}

func (n *CharacterNode) String() string     { return fmt.Sprintf("char(%q)", n.Val) }
func (n *CharacterNode) Parent() Node       { return n.parent }
func (n *CharacterNode) SetParent(p Node)   { n.parent = p }
func (n *CharacterNode) Children() []Node   { return nil }
func (n *CharacterNode) SetChildren([]Node) { panic("SetChildren called on CharacterNode") }

type CommentNode struct {
	*Pos
	parent Node
	Text   string
}

func (n *CommentNode) String() string     { return fmt.Sprintf("comment(%q)", n.Text) }
func (n *CommentNode) Parent() Node       { return n.parent }
func (n *CommentNode) SetParent(p Node)   { n.parent = p }
func (n *CommentNode) Children() []Node   { return nil }
func (n *CommentNode) SetChildren([]Node) { panic("SetChildren called on CommentNode") }

type DerefNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *DerefNode) String() string   { return "deref" }
func (n *DerefNode) Parent() Node     { return n.parent }
func (n *DerefNode) SetParent(p Node) { n.parent = p }
func (n *DerefNode) Children() []Node { return []Node{n.Node} }
func (n *DerefNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on DerefNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type KeywordNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *KeywordNode) String() string     { return fmt.Sprintf("keyword(%s)", n.Val) }
func (n *KeywordNode) Parent() Node       { return n.parent }
func (n *KeywordNode) SetParent(p Node)   { n.parent = p }
func (n *KeywordNode) Children() []Node   { return nil }
func (n *KeywordNode) SetChildren([]Node) { panic("SetChildren called on KeywordNode") }

type ListNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *ListNode) String() string {
	return fmt.Sprintf("list(length=%d)", countSemantic(n.Nodes))
}
func (n *ListNode) Parent() Node             { return n.parent }
func (n *ListNode) SetParent(p Node)         { n.parent = p }
func (n *ListNode) Children() []Node         { return n.Nodes }
func (n *ListNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type MapNode struct {
	*Pos
	parent    Node
	Namespace string // empty unless the map has a namespace: #:ns{:x 1}
	Nodes     []Node
}

func (n *MapNode) String() string {
	var ns string
	if n.Namespace != "" {
		ns = fmt.Sprintf("ns=%s, ", n.Namespace)
	}
	semanticNodes := countSemantic(n.Nodes)
	return fmt.Sprintf("map(%slength=%d)", ns, semanticNodes/2)
}
func (n *MapNode) Parent() Node             { return n.parent }
func (n *MapNode) SetParent(p Node)         { n.parent = p }
func (n *MapNode) Children() []Node         { return n.Nodes }
func (n *MapNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type MetadataNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *MetadataNode) String() string   { return "metadata" }
func (n *MetadataNode) Parent() Node     { return n.parent }
func (n *MetadataNode) SetParent(p Node) { n.parent = p }
func (n *MetadataNode) Children() []Node { return []Node{n.Node} }
func (n *MetadataNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on MetadataNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type NewlineNode struct {
	*Pos
	parent Node
}

func (n *NewlineNode) String() string     { return "newline" }
func (n *NewlineNode) Parent() Node       { return n.parent }
func (n *NewlineNode) SetParent(p Node)   { n.parent = p }
func (n *NewlineNode) Children() []Node   { return nil }
func (n *NewlineNode) SetChildren([]Node) { panic("SetChildren called on NewlineNode") }

type NilNode struct {
	*Pos
	parent Node
}

func (n *NilNode) String() string     { return "nil" }
func (n *NilNode) Parent() Node       { return n.parent }
func (n *NilNode) SetParent(p Node)   { n.parent = p }
func (n *NilNode) Children() []Node   { return nil }
func (n *NilNode) SetChildren([]Node) { panic("SetChildren called on NilNode") }

type NumberNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *NumberNode) String() string     { return fmt.Sprintf("num(%s)", n.Val) }
func (n *NumberNode) Parent() Node       { return n.parent }
func (n *NumberNode) SetParent(p Node)   { n.parent = p }
func (n *NumberNode) Children() []Node   { return nil }
func (n *NumberNode) SetChildren([]Node) { panic("SetChildren called on NumberNode") }

type SymbolNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *SymbolNode) String() string     { return "sym(" + n.Val + ")" }
func (n *SymbolNode) Parent() Node       { return n.parent }
func (n *SymbolNode) SetParent(p Node)   { n.parent = p }
func (n *SymbolNode) Children() []Node   { return nil }
func (n *SymbolNode) SetChildren([]Node) { panic("SetChildren called on SymbolNode") }

type QuoteNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *QuoteNode) String() string   { return "quote" }
func (n *QuoteNode) Parent() Node     { return n.parent }
func (n *QuoteNode) SetParent(p Node) { n.parent = p }
func (n *QuoteNode) Children() []Node { return []Node{n.Node} }
func (n *QuoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on QuoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type StringNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *StringNode) String() string     { return fmt.Sprintf("string(%q)", n.Val) }
func (n *StringNode) Parent() Node       { return n.parent }
func (n *StringNode) SetParent(p Node)   { n.parent = p }
func (n *StringNode) Children() []Node   { return nil }
func (n *StringNode) SetChildren([]Node) { panic("SetChildren called on StringNode") }

type SyntaxQuoteNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *SyntaxQuoteNode) String() string   { return "syntax quote" }
func (n *SyntaxQuoteNode) Parent() Node     { return n.parent }
func (n *SyntaxQuoteNode) SetParent(p Node) { n.parent = p }
func (n *SyntaxQuoteNode) Children() []Node { return []Node{n.Node} }
func (n *SyntaxQuoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on SyntaxQuoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type UnquoteNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *UnquoteNode) String() string   { return "unquote" }
func (n *UnquoteNode) Parent() Node     { return n.parent }
func (n *UnquoteNode) SetParent(p Node) { n.parent = p }
func (n *UnquoteNode) Children() []Node { return []Node{n.Node} }
func (n *UnquoteNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on UnquoteNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type UnquoteSpliceNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *UnquoteSpliceNode) String() string   { return "unquote splice" }
func (n *UnquoteSpliceNode) Parent() Node     { return n.parent }
func (n *UnquoteSpliceNode) SetParent(p Node) { n.parent = p }
func (n *UnquoteSpliceNode) Children() []Node { return []Node{n.Node} }
func (n *UnquoteSpliceNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on UnquoteSpliceNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type VectorNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *VectorNode) String() string {
	return fmt.Sprintf("vector(length=%d)", countSemantic(n.Nodes))
}
func (n *VectorNode) Parent() Node             { return n.parent }
func (n *VectorNode) SetParent(p Node)         { n.parent = p }
func (n *VectorNode) Children() []Node         { return n.Nodes }
func (n *VectorNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type FnLiteralNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *FnLiteralNode) String() string {
	return fmt.Sprintf("lambda(length=%d)", countSemantic(n.Nodes))
}
func (n *FnLiteralNode) Parent() Node             { return n.parent }
func (n *FnLiteralNode) SetParent(p Node)         { n.parent = p }
func (n *FnLiteralNode) Children() []Node         { return n.Nodes }
func (n *FnLiteralNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type ReaderCondNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *ReaderCondNode) String() string {
	return fmt.Sprintf("reader-cond(length=%d)", len(n.Nodes))
}
func (n *ReaderCondNode) Parent() Node             { return n.parent }
func (n *ReaderCondNode) SetParent(p Node)         { n.parent = p }
func (n *ReaderCondNode) Children() []Node         { return n.Nodes }
func (n *ReaderCondNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type ReaderCondSpliceNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *ReaderCondSpliceNode) String() string {
	return fmt.Sprintf("reader-cond-splice(length=%d)", len(n.Nodes))
}
func (n *ReaderCondSpliceNode) Parent() Node             { return n.parent }
func (n *ReaderCondSpliceNode) SetParent(p Node)         { n.parent = p }
func (n *ReaderCondSpliceNode) Children() []Node         { return n.Nodes }
func (n *ReaderCondSpliceNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type ReaderDiscardNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *ReaderDiscardNode) String() string   { return "discard" }
func (n *ReaderDiscardNode) Parent() Node     { return n.parent }
func (n *ReaderDiscardNode) SetParent(p Node) { n.parent = p }
func (n *ReaderDiscardNode) Children() []Node { return []Node{n.Node} }
func (n *ReaderDiscardNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on ReaderDiscardNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type ReaderEvalNode struct {
	*Pos
	parent Node
	Node   Node
}

func (n *ReaderEvalNode) String() string   { return "eval" }
func (n *ReaderEvalNode) Parent() Node     { return n.parent }
func (n *ReaderEvalNode) SetParent(p Node) { n.parent = p }
func (n *ReaderEvalNode) Children() []Node { return []Node{n.Node} }
func (n *ReaderEvalNode) SetChildren(nodes []Node) {
	if len(nodes) != 1 {
		panicf("SetChildren called on ReaderEvalNode with %d nodes", len(nodes))
	}
	n.Node = nodes[0]
}

type RegexNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *RegexNode) String() string     { return fmt.Sprintf("regex(%q)", n.Val) }
func (n *RegexNode) Parent() Node       { return n.parent }
func (n *RegexNode) SetParent(p Node)   { n.parent = p }
func (n *RegexNode) Children() []Node   { return nil }
func (n *RegexNode) SetChildren([]Node) { panic("SetChildren called on RegexNode") }

type SetNode struct {
	*Pos
	parent Node
	Nodes  []Node
}

func (n *SetNode) String() string {
	return fmt.Sprintf("set(length=%d)", countSemantic(n.Nodes))
}
func (n *SetNode) Parent() Node             { return n.parent }
func (n *SetNode) SetParent(p Node)         { n.parent = p }
func (n *SetNode) Children() []Node         { return n.Nodes }
func (n *SetNode) SetChildren(nodes []Node) { n.Nodes = nodes }

type VarQuoteNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *VarQuoteNode) String() string     { return fmt.Sprintf("varquote(%s)", n.Val) }
func (n *VarQuoteNode) Parent() Node       { return n.parent }
func (n *VarQuoteNode) SetParent(p Node)   { n.parent = p }
func (n *VarQuoteNode) Children() []Node   { return nil }
func (n *VarQuoteNode) SetChildren([]Node) { panic("SetChildren called on VarQuoteNode") }

type TagNode struct {
	*Pos
	parent Node
	Val    string
}

func (n *TagNode) String() string     { return fmt.Sprintf("tag(%s)", n.Val) }
func (n *TagNode) Parent() Node       { return n.parent }
func (n *TagNode) SetParent(p Node)   { n.parent = p }
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

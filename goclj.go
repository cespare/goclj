package goclj

import "github.com/cespare/goclj/parse"

func FnFormSymbol(node parse.Node, sym ...string) bool {
	list, ok := node.(*parse.ListNode)
	if !ok {
		return false
	}
	children := list.Children()
	if len(children) == 0 {
		return false
	}
	s, ok := children[0].(*parse.SymbolNode)
	if !ok {
		return false
	}
	if len(sym) == 0 {
		return true
	}
	for _, name := range sym {
		if s.Val == name {
			return true
		}
	}
	return false
}

func FnFormKeyword(node parse.Node, kw ...string) bool {
	list, ok := node.(*parse.ListNode)
	if !ok {
		return false
	}
	children := list.Children()
	if len(children) == 0 {
		return false
	}
	k, ok := children[0].(*parse.KeywordNode)
	if !ok {
		return false
	}
	if len(kw) == 0 {
		return true
	}
	for _, name := range kw {
		if k.Val == name {
			return true
		}
	}
	return false
}

func Symbol(node parse.Node) bool {
	_, ok := node.(*parse.SymbolNode)
	return ok
}

func Newline(node parse.Node) bool {
	_, ok := node.(*parse.NewlineNode)
	return ok
}

func Vector(node parse.Node) bool {
	_, ok := node.(*parse.VectorNode)
	return ok
}

func Keyword(node parse.Node) bool {
	_, ok := node.(*parse.KeywordNode)
	return ok
}

func Comment(node parse.Node) bool {
	_, ok := node.(*parse.CommentNode)
	return ok
}

// Semantic returns whether a node changes the semantics of the code.
// NOTE: right now this is only used for let indenting.
// It might have to be adjusted if used for other purposes.
func Semantic(node parse.Node) bool {
	switch node.(type) {
	case *parse.NewlineNode, *parse.CommentNode, *parse.MetadataNode, *parse.TagNode:
		return false
	}
	return true
}

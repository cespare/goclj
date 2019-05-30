package format

import "github.com/cespare/goclj/parse"

func (p *Printer) markThreadFirsts(n parse.Node) {
	var nodes []parse.Node
	switch n := n.(type) {
	case *parse.ListNode:
		nodes = n.Nodes
	case *parse.FnLiteralNode:
		nodes = n.Nodes
	}
	if len(nodes) > 0 {
		if sym, ok := nodes[0].(*parse.SymbolNode); ok {
			if style, ok := p.threadFirstStyles[sym.Val]; ok {
				p.markThreadFirstStyle(n, style)
			}
		}
	}
	for _, node := range n.Children() {
		p.markThreadFirsts(node)
	}
}

func (p *Printer) markThreadFirstStyle(form parse.Node, style ThreadFirstStyle) {
	begin := 2
	if _, ok := p.threadFirst[form]; ok {
		begin = 1 // nested thread-first forms
	}
	idxSemantic := 0
	for _, node := range form.Children() {
		switch n := node.(type) {
		case *parse.CommentNode, *parse.NewlineNode:
			continue
		case *parse.ListNode:
			if idxSemantic >= begin {
				switch style {
				case ThreadFirstNormal:
					p.threadFirst[n] = struct{}{}
				case ThreadFirstCondArrow:
					if (begin-idxSemantic)&1 > 0 {
						// Only apply to the second of a pair.
						p.threadFirst[n] = struct{}{}
					}
				}
			}
		}
		idxSemantic++
	}
}

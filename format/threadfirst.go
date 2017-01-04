package format

import "github.com/cespare/goclj/parse"

func (p *Printer) markThreadFirsts(n parse.Node) {
	if list, ok := n.(*parse.ListNode); ok {
		if len(list.Nodes) > 0 {
			if sym, ok := list.Nodes[0].(*parse.SymbolNode); ok {
				if style, ok := p.threadFirstStyles[sym.Val]; ok {
					p.markThreadFirstStyle(list, style)
				}
			}
		}
	}
	for _, node := range n.Children() {
		p.markThreadFirsts(node)
	}
}

func (p *Printer) markThreadFirstStyle(form *parse.ListNode, style ThreadFirstStyle) {
	begin := 2
	if _, ok := p.threadFirst[form]; ok {
		begin = 1 // nested thread-first forms
	}
	idxSemantic := 0
	for _, node := range form.Nodes {
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

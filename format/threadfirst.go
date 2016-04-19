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
	i := 2
	if _, ok := p.threadFirst[form]; ok {
		i = 1 // nested thread-first forms
	}
	condArrowFirst := true
	for ; i < len(form.Nodes); i++ {
		n, ok := form.Nodes[i].(*parse.ListNode)
		if !ok {
			continue
		}
		switch style {
		case ThreadFirstNormal:
			p.threadFirst[n] = struct{}{}
		case ThreadFirstCondArrow:
			if !condArrowFirst {
				p.threadFirst[n] = struct{}{}
			}
		}
		condArrowFirst = !condArrowFirst
	}
}

package format

import (
	"sort"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

type require struct {
	name     string
	as       map[string]struct{}
	referAll bool
	refer    map[string]struct{}
}

type requireList struct {
	m map[string]*require
	// unrecognized semantic nodes
	extraRequire []parse.Node
	extraUse     []parse.Node
}

func newRequireList() *requireList {
	return &requireList{m: make(map[string]*require)}
}

func (rl *requireList) merge(r *require) {
	r2, ok := rl.m[r.name]
	if !ok {
		rl.m[r.name] = r
		return
	}
	if r.as != nil {
		if r2.as == nil {
			r2.as = make(map[string]struct{})
		}
		for s := range r.as {
			r2.as[s] = struct{}{}
		}
	}
	r2.referAll = r.referAll || r2.referAll
	if r.refer != nil {
		if r2.refer == nil {
			r2.refer = make(map[string]struct{})
		}
		for s := range r.refer {
			r2.refer[s] = struct{}{}
		}
	}
}

func (rl *requireList) parseRequire(n *parse.ListNode) {
	for _, node := range n.Children()[1:] {
		switch node.(type) {
		case *parse.CommentNode, *parse.NewlineNode:
		default:
			if r, ok := parseRequire(node); ok {
				rl.merge(r)
			} else {
				rl.extraRequire = append(rl.extraRequire, node)
			}
		}
	}
}

func (rl *requireList) parseUse(n *parse.ListNode) {
	for _, node := range n.Children()[1:] {
		switch node.(type) {
		case *parse.CommentNode, *parse.NewlineNode:
		default:
			if r, ok := parseUse(node); ok {
				rl.merge(r)
			} else {
				rl.extraUse = append(rl.extraUse, node)
			}
		}
	}
}

func (rl *requireList) render() []parse.Node {
	nodes := []parse.Node{
		&parse.KeywordNode{Val: ":require"},
	}
	for _, r := range rl.m {
		parts := []parse.Node{&parse.SymbolNode{Val: r.name}}
		as := sortStringSet(r.as)
		// If there are multiple :as definitions, emit a separate
		// require for the first n-1 of them.
		for len(as) > 1 {
			n := &parse.VectorNode{
				Nodes: []parse.Node{
					&parse.SymbolNode{Val: r.name},
					&parse.KeywordNode{Val: ":as"},
					&parse.SymbolNode{Val: as[0]},
				},
			}
			nodes = append(nodes, n, &parse.NewlineNode{})
			as = as[1:]
		}
		if len(as) > 0 {
			parts = append(parts,
				&parse.KeywordNode{Val: ":as"},
				&parse.SymbolNode{Val: as[0]})
		}
		if r.referAll {
			parts = append(parts,
				&parse.KeywordNode{Val: ":refer"},
				&parse.KeywordNode{Val: ":all"})
		} else if len(r.refer) > 0 {
			var refs []parse.Node
			for _, s := range sortStringSet(r.refer) {
				refs = append(refs, &parse.SymbolNode{Val: s})
			}
			parts = append(parts,
				&parse.KeywordNode{Val: ":refer"},
				&parse.VectorNode{Nodes: refs})
		}
		n := &parse.VectorNode{Nodes: parts}
		nodes = append(nodes, n, &parse.NewlineNode{})
	}
	nodes = append(nodes, rl.extraRequire...)

	list := []parse.Node{
		&parse.ListNode{Nodes: nodes},
		&parse.NewlineNode{},
	}
	if len(rl.extraUse) > 0 {
		extra := []parse.Node{
			&parse.KeywordNode{Val: ":use"},
		}
		for _, n := range rl.extraUse {
			extra = append(extra, n, &parse.NewlineNode{})
		}
		list = append(list,
			&parse.ListNode{Nodes: extra},
			&parse.NewlineNode{},
		)
	}
	return list
}

func sortStringSet(set map[string]struct{}) []string {
	var ss []string
	for s := range set {
		ss = append(ss, s)
	}
	sort.Strings(ss)
	return ss
}

func parseRequire(n parse.Node) (r *require, ok bool) {
	switch n := n.(type) {
	case *parse.SymbolNode:
		return &require{name: n.Val}, true
	case *parse.ListNode, *parse.VectorNode:
		return parseRequireSeq(n.Children())
	default:
		return nil, false
	}
}

func parseRequireSeq(nodes []parse.Node) (r *require, ok bool) {
	if len(nodes) == 0 || !goclj.Symbol(nodes[0]) {
		return nil, false
	}
	r = &require{name: nodes[0].(*parse.SymbolNode).Val}
	var as string
	var refer []string
	numPairs := (len(nodes) - 1) / 2
	for i := 0; i < numPairs; i++ {
		k, v := nodes[i*2+1], nodes[i*2+2]
		kw, ok := k.(*parse.KeywordNode)
		if !ok {
			return nil, false
		}
		// If there are multiple :as or :refers in a require, like
		//   (require '[a :as b :as c])
		// then only the last one takes effect.
		switch kw.Val {
		case ":as":
			vs, ok := v.(*parse.SymbolNode)
			if !ok {
				return nil, false
			}
			as = vs.Val
		case ":refer":
			switch v.(type) {
			case *parse.ListNode, *parse.VectorNode:
				refer = nil
				for _, n := range v.Children() {
					ref, ok := n.(*parse.SymbolNode)
					if !ok {
						return nil, false
					}
					refer = append(refer, ref.Val)
				}
			default:
				return nil, false
			}
		default:
			return nil, false
		}
	}
	if as != "" {
		r.as = map[string]struct{}{as: struct{}{}}
	}
	r.refer = make(map[string]struct{})
	for _, s := range refer {
		r.refer[s] = struct{}{}
	}
	return r, true
}

func parseUse(n parse.Node) (r *require, ok bool) {
	switch n := n.(type) {
	case *parse.SymbolNode:
		return &require{name: n.Val, referAll: true}, true
	case *parse.ListNode, *parse.VectorNode:
		return parseUseSeq(n.Children())
	default:
		return nil, false
	}
}

func parseUseSeq(nodes []parse.Node) (r *require, ok bool) {
	if len(nodes) == 0 || !goclj.Symbol(nodes[0]) {
		return nil, false
	}
	r = &require{name: nodes[0].(*parse.SymbolNode).Val}
	switch len(nodes) {
	case 1:
		r.referAll = true
	case 3:
		kw, ok := nodes[1].(*parse.KeywordNode)
		if !ok {
			return nil, false
		}
		if kw.Val != ":only" {
			return nil, false
		}
		switch nodes[2].(type) {
		case *parse.ListNode, *parse.VectorNode:
		default:
			return nil, false
		}
		r.refer = make(map[string]struct{})
		for _, n := range nodes[2].Children() {
			sym, ok := n.(*parse.SymbolNode)
			if !ok {
				return nil, false
			}
			r.refer[sym.Val] = struct{}{}
		}
	default:
		return nil, false
	}
	return r, true
}

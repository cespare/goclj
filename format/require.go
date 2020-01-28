package format

import (
	"sort"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

func (p *Printer) markRequires(n parse.Node) {
	if !goclj.FnFormSymbol(n, "ns") {
		return
	}
	for _, n := range n.Children() {
		if !goclj.FnFormKeyword(n, ":require", ":require-macros") {
			continue
		}
		for _, n := range n.Children()[1:] {
			r := parseRequire(n)
			if r == nil {
				continue
			}
			for as := range r.as {
				p.requires[as] = r.name
			}
			for _, ref := range []*referList{&r.refer, &r.referMacros} {
				for _, n := range ref.origRefer {
					if n, ok := n.(*parse.SymbolNode); ok {
						p.refers[n.Val] = r.name
					}
				}
				for ref := range ref.refer {
					p.refers[ref] = r.name
				}
			}
		}
	}
}

// referList represents a list of symbols following either a :refer or
// :refer-macros keyword.
type referList struct {
	// origRefer is the original refer vector/list.
	// It is preserved and reused if possible, but if two refer lists are
	// merged, then refer is used instead (and origRefer is set to nil).
	origRefer []parse.Node
	refer     map[string]struct{}
}

func (rl *referList) merge(rl1 *referList) {
	if rl1 == nil {
		return
	}
	if rl1.refer != nil {
		panic("merge arg has non-nil refer")
	}
	if len(rl1.origRefer) == 0 {
		return
	}
	// If the current list has *no* refers, just copy the origRefer
	// list from the other.
	if rl.origRefer == nil && rl.refer == nil {
		rl.origRefer = rl1.origRefer
		return
	}
	// Otherwise, move everything into rl.refer (if not already moved by
	// a previous merge) and copy each symbol from the other list.
	if rl.origRefer != nil {
		rl.extractOrigRefer()
	}
	for _, n := range rl1.origRefer {
		if n, ok := n.(*parse.SymbolNode); ok {
			rl.refer[n.Val] = struct{}{}
		}
	}
}

// render returns a parse.VectorNode for all symbols in this referList,
// or nil if there are no symbols.
func (rl *referList) render() *parse.VectorNode {
	if rl.origRefer != nil {
		return &parse.VectorNode{Nodes: rl.origRefer}
	}
	if len(rl.refer) == 0 {
		return nil
	}
	var refs []parse.Node
	for _, s := range sortStringSet(rl.refer) {
		refs = append(refs, &parse.SymbolNode{Val: s})
	}
	return &parse.VectorNode{Nodes: refs}
}

// extractOrigRefer moves all symbols in the origRefer slice
// into rl.refer map.
func (rl *referList) extractOrigRefer() {
	rl.refer = make(map[string]struct{})
	for _, n := range rl.origRefer {
		if n, ok := n.(*parse.SymbolNode); ok {
			rl.refer[n.Val] = struct{}{}
		}
	}
	rl.origRefer = nil
}

// removeUnused removes all symbols from this referList that aren't
// present in sc.
func (rl *referList) removeUnused(sc *symbolCache) {
	if rl.origRefer != nil {
		// If origRefer doesn't have any unused elements, leave it
		// alone. Otherwise, rewrite it as a refer and handle below.
		for _, n := range rl.origRefer {
			n, ok := n.(*parse.SymbolNode)
			if !ok {
				continue
			}
			if !sc.usesSym(n.Val) {
				rl.extractOrigRefer()
				break
			}
		}
	}
	for ref := range rl.refer {
		if !sc.usesSym(ref) {
			delete(rl.refer, ref)
		}
	}
}

type require struct {
	name     string
	as       map[string]struct{}
	referAll bool

	refer       referList
	referMacros referList

	comments nodeComments
}

func newRequire(name string) *require {
	return &require{
		name: name,
	}
}

type nodeComments struct {
	commentsAbove        []*parse.CommentNode
	commentBeside        *parse.CommentNode
	commentsBesideMerged bool
}

var newline = &parse.NewlineNode{}

func (nc *nodeComments) attachCommentsAbove(cs []*parse.CommentNode) {
	nc.commentsAbove = append(nc.commentsAbove, cs...)
}

func (nc *nodeComments) attachCommentBeside(c *parse.CommentNode) {
	if nc.commentBeside == nil && !nc.commentsBesideMerged {
		nc.commentBeside = c
		return
	}
	if nc.commentBeside != nil {
		nc.commentsAbove = append(nc.commentsAbove, nc.commentBeside, c)
		nc.commentBeside = nil
		nc.commentsBesideMerged = true
		return
	}
	nc.commentsAbove = append(nc.commentsAbove, c)
}

type nodeWithComments struct {
	n        parse.Node
	comments nodeComments
}

type requireList struct {
	m map[string]*require
	// macros is true if this represents a :require-macros list.
	macros bool

	// unrecognized semantic nodes
	extraRequire []*nodeWithComments
	extraUse     []*nodeWithComments

	commentsBelow []*parse.CommentNode
}

func newRequireList() *requireList {
	return &requireList{m: make(map[string]*require)}
}

func (rl *requireList) merge(r *require) *require {
	r2, ok := rl.m[r.name]
	if !ok {
		rl.m[r.name] = r
		return r
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
	r2.refer.merge(&r.refer)
	r2.referMacros.merge(&r.referMacros)
	return r2
}

func (rl *requireList) parseRequireUse(nodes []parse.Node, use bool) {
	var (
		parseFn           = parseRequire
		extra             = &rl.extraRequire
		prevComments      *nodeComments
		lineComments      []*parse.CommentNode
		afterSemanticNode = false
	)
	if use {
		parseFn = parseUse
		extra = &rl.extraUse
	}
	switch nodes[0].(*parse.KeywordNode).Val {
	case ":require-macros", ":use-macros":
		rl.macros = true
	}
	for _, node := range nodes[1:] {
		switch node := node.(type) {
		case *parse.CommentNode:
			if afterSemanticNode {
				prevComments.attachCommentBeside(node)
			} else {
				lineComments = append(lineComments, node)
			}
		case *parse.NewlineNode:
			afterSemanticNode = false
		default:
			if r := parseFn(node); r != nil {
				r2 := rl.merge(r)
				prevComments = &r2.comments
			} else {
				nc := &nodeWithComments{n: node}
				*extra = append(*extra, nc)
				prevComments = &nc.comments
			}
			prevComments.attachCommentsAbove(lineComments)
			afterSemanticNode = true
			lineComments = nil
		}
	}
	rl.commentsBelow = append(rl.commentsBelow, lineComments...)
}

func (rl *requireList) render() []parse.Node {
	var nodes []parse.Node
	if rl.macros {
		nodes = append(nodes, &parse.KeywordNode{Val: ":require-macros"})
	} else {
		nodes = append(nodes, &parse.KeywordNode{Val: ":require"})
	}
	for _, r := range rl.m {
		for _, c := range r.comments.commentsAbove {
			nodes = append(nodes, c, newline)
		}
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
			nodes = append(nodes, n, newline)
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
		} else {
			if n := r.refer.render(); n != nil {
				parts = append(parts, &parse.KeywordNode{Val: ":refer"}, n)
			}
			if n := r.referMacros.render(); n != nil {
				parts = append(parts, &parse.KeywordNode{Val: ":refer-macros"}, n)
			}
		}
		nodes = append(nodes, &parse.VectorNode{Nodes: parts})
		if r.comments.commentBeside != nil {
			nodes = append(nodes, r.comments.commentBeside)
		}
		nodes = append(nodes, newline)
	}
	for _, r := range rl.extraRequire {
		for _, c := range r.comments.commentsAbove {
			nodes = append(nodes, c, newline)
		}
		nodes = append(nodes, r.n)
		if r.comments.commentBeside != nil {
			nodes = append(nodes, r.comments.commentBeside)
		}
		nodes = append(nodes, newline)
	}
	for _, c := range rl.commentsBelow {
		nodes = append(nodes, c, newline)
	}

	list := []parse.Node{
		&parse.ListNode{Nodes: nodes},
		newline,
	}
	if len(rl.extraUse) > 0 {
		extra := []parse.Node{
			&parse.KeywordNode{Val: ":use"},
		}
		for _, n := range rl.extraUse {
			for _, c := range n.comments.commentsAbove {
				extra = append(extra, c, newline)
			}
			extra = append(extra, n.n)
			if n.comments.commentBeside != nil {
				extra = append(extra, n.comments.commentBeside)
			}
			extra = append(extra, newline)
		}
		list = append(list,
			&parse.ListNode{Nodes: extra},
			newline,
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

func parseRequire(n parse.Node) *require {
	switch n := n.(type) {
	case *parse.SymbolNode:
		return newRequire(n.Val)
	case *parse.ListNode, *parse.VectorNode:
		return parseRequireSeq(n.Children())
	default:
		return nil
	}
}

func parseRequireSeq(nodes []parse.Node) *require {
	semNodes := make([]parse.Node, 0, len(nodes))
	for _, n := range nodes {
		if goclj.Semantic(n) {
			semNodes = append(semNodes, n)
		}
	}
	if len(semNodes) == 0 || !goclj.Symbol(semNodes[0]) {
		return nil
	}
	r := newRequire(semNodes[0].(*parse.SymbolNode).Val)
	var as string
	var refer []parse.Node
	var referMacros []parse.Node
	if (len(semNodes)-1)%2 != 0 {
		return nil
	}
	numPairs := (len(semNodes) - 1) / 2
	for i := 0; i < numPairs; i++ {
		k, v := semNodes[i*2+1], semNodes[i*2+2]
		kw, ok := k.(*parse.KeywordNode)
		if !ok {
			return nil
		}
		// If there are multiple :as or :refers in a require, like
		//   (require '[a :as b :as c])
		// then only the last one takes effect.
		switch kw.Val {
		case ":as":
			vs, ok := v.(*parse.SymbolNode)
			if !ok {
				return nil
			}
			as = vs.Val
		case ":refer", ":refer-macros":
			if kw.Val == ":refer" {
				refer = v.Children()
			} else {
				referMacros = v.Children()
			}
			switch v.(type) {
			case *parse.ListNode, *parse.VectorNode:
				for _, n := range v.Children() {
					if !goclj.Semantic(n) {
						continue
					}
					if _, ok := n.(*parse.SymbolNode); !ok {
						return nil
					}
				}
			default:
				return nil
			}
		default:
			return nil
		}
	}
	if as != "" {
		r.as = map[string]struct{}{as: {}}
	}
	r.refer.origRefer = refer
	r.referMacros.origRefer = referMacros
	return r
}

func parseUse(n parse.Node) *require {
	switch n := n.(type) {
	case *parse.SymbolNode:
		r := newRequire(n.Val)
		r.referAll = true
		return r
	case *parse.ListNode, *parse.VectorNode:
		return parseUseSeq(n.Children())
	default:
		return nil
	}
}

func parseUseSeq(nodes []parse.Node) *require {
	if len(nodes) == 0 || !goclj.Symbol(nodes[0]) {
		return nil
	}
	r := newRequire(nodes[0].(*parse.SymbolNode).Val)
	switch len(nodes) {
	case 1:
		r.referAll = true
	case 3:
		kw, ok := nodes[1].(*parse.KeywordNode)
		if !ok {
			return nil
		}
		switch kw.Val {
		case ":as":
			n, ok := nodes[2].(*parse.SymbolNode)
			if !ok {
				return nil
			}
			r.as = map[string]struct{}{n.Val: {}}
		case ":only":
			switch nodes[2].(type) {
			case *parse.ListNode, *parse.VectorNode:
			default:
				return nil
			}
			r.refer.origRefer = nodes[2].Children()
			for _, n := range r.refer.origRefer {
				switch n.(type) {
				case *parse.SymbolNode,
					*parse.CommentNode,
					*parse.NewlineNode:
				default:
					return nil
				}
			}
		default:
			return nil
		}
	default:
		return nil
	}
	return r
}

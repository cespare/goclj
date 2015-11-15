package format

import (
	"sort"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

// applyTransforms performs small fixes to the tree t:
//
//   - reorder (sort) imports and requires
//   - remove trailing \n nodes (dangling close parens)
//   - move arg vectors of defns to the same line if appropriate
//   - move dispatch-val of a defmethod to the same line
//   - remove >1 consecutive blank lines
func applyTransforms(t *parse.Tree) {
	for _, root := range t.Roots {
		if goclj.FnFormSymbol(root, "ns") {
			sortNS(root)
		}
		removeTrailingNewlines(root)
		if goclj.FnFormSymbol(root, "defn") {
			fixDefnArglist(root)
		}
		if goclj.FnFormSymbol(root, "defmethod") {
			fixDefmethodDispatchVal(root)
		}
		removeExtraBlankLinesRecursive(root)
	}
	t.Roots = removeExtraBlankLines(t.Roots)
}

func sortNS(ns parse.Node) {
	for _, n := range ns.Children()[1:] {
		if goclj.FnFormKeyword(n, ":require", ":import") {
			sortImportRequire(n.(*parse.ListNode))
		}
	}
}

func sortImportRequire(n *parse.ListNode) {
	var (
		nodes             = n.Children()
		sorted            = make(importRequireList, 0, len(nodes)/2)
		lineComments      []*parse.CommentNode
		afterSemanticNode = false
	)
	for _, node := range nodes[1:] {
		switch node := node.(type) {
		case *parse.CommentNode:
			if afterSemanticNode {
				sorted[len(sorted)-1].CommentBeside = node
			} else {
				lineComments = append(lineComments, node)
			}
		case *parse.NewlineNode:
			afterSemanticNode = false
		default:
			ir := &importRequire{
				CommentsAbove: lineComments,
				Node:          node,
			}
			sorted = append(sorted, ir)
			lineComments = nil
			afterSemanticNode = true
		}
	}
	sort.Sort(sorted)
	newNodes := []parse.Node{nodes[0]}
	for _, ir := range sorted {
		for _, cn := range ir.CommentsAbove {
			newNodes = append(newNodes, cn, &parse.NewlineNode{})
		}
		newNodes = append(newNodes, ir.Node)
		if ir.CommentBeside != nil {
			newNodes = append(newNodes, ir.CommentBeside)
		}
		newNodes = append(newNodes, &parse.NewlineNode{})
	}
	// unattached comments at the bottom
	for _, cn := range lineComments {
		newNodes = append(newNodes, cn, &parse.NewlineNode{})
	}
	// drop trailing newline
	if len(newNodes) >= 2 && !goclj.Comment(newNodes[len(newNodes)-2]) {
		newNodes = newNodes[:len(newNodes)-1]
	}
	n.SetChildren(newNodes)
}

func removeTrailingNewlines(n parse.Node) {
	nodes := n.Children()
	if len(nodes) == 0 {
		return
	}
	switch n.(type) {
	case *parse.ListNode, *parse.MapNode, *parse.VectorNode, *parse.FnLiteralNode, *parse.SetNode:
		for ; len(nodes) > 0; nodes = nodes[:len(nodes)-1] {
			if len(nodes) >= 2 && goclj.Comment(nodes[len(nodes)-2]) {
				break
			}
			if !goclj.Newline(nodes[len(nodes)-1]) {
				break
			}
		}
		n.SetChildren(nodes)
	}
	for _, node := range nodes {
		removeTrailingNewlines(node)
	}
}

func fixDefnArglist(defn parse.Node) {
	// For defns, change
	//   (defn foo
	//     [x] ...)
	// to
	//   (defn foo [x]
	//     ...)
	// if there's no newline after the arg list.
	nodes := defn.Children()
	if len(nodes) < 5 {
		return
	}
	if !goclj.Newline(nodes[2]) || goclj.Newline(nodes[4]) {
		return
	}
	if !goclj.Vector(nodes[3]) {
		return
	}
	// Move the newline to be after the arglist.
	nodes[2], nodes[3] = nodes[3], nodes[2]
	defn.SetChildren(nodes)
}

func fixDefmethodDispatchVal(defmethod parse.Node) {
	// For defmethods, change
	//   (defmethod foo
	//     :bar
	//     [x] ...)
	// to
	//   (defmethod foo :bar
	//     [x] ...)
	nodes := defmethod.Children()
	if len(nodes) < 5 {
		return
	}
	if !goclj.Newline(nodes[2]) {
		return
	}
	if !goclj.Keyword(nodes[3]) {
		return
	}
	// Move the dispatch-val up to the same line.
	// Insert a newline after if there wasn't one already.
	if goclj.Newline(nodes[4]) {
		nodes = append(nodes[:2], nodes[3:]...)
	} else {
		nodes[2], nodes[3] = nodes[3], nodes[2]
	}
	defmethod.SetChildren(nodes)
}

func removeExtraBlankLinesRecursive(n parse.Node) {
	nodes := n.Children()
	if len(nodes) == 0 {
		return
	}
	if len(nodes) > 2 {
		nodes = removeExtraBlankLines(nodes)
		n.SetChildren(nodes)
	}
	for _, node := range nodes {
		removeExtraBlankLinesRecursive(node)
	}
}

func removeExtraBlankLines(nodes []parse.Node) []parse.Node {
	newNodes := make([]parse.Node, 0, len(nodes))
	newlines := 0
	for _, node := range nodes {
		if goclj.Newline(node) {
			newlines++
		} else {
			newlines = 0
		}
		if newlines <= 2 {
			newNodes = append(newNodes, node)
		}
	}
	return newNodes
}

// An importRequire is an import/require with associated comment nodes.
type importRequire struct {
	CommentsAbove []*parse.CommentNode
	CommentBeside *parse.CommentNode
	Node          parse.Node
}

type importRequireList []*importRequire

func (l importRequireList) Len() int      { return len(l) }
func (l importRequireList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l importRequireList) Less(i, j int) bool {
	n1, n2 := l[i].Node, l[j].Node
	if s1, ok := n1.(*parse.SymbolNode); ok {
		if s2, ok := n2.(*parse.SymbolNode); ok {
			return s1.Val < s2.Val
		}
		if goclj.Vector(n2) {
			return true
		}
		return true
	}
	if v1, ok := n1.(*parse.VectorNode); ok {
		if v2, ok := n2.(*parse.VectorNode); ok {
			if len(v1.Nodes) == 0 {
				return true
			}
			if len(v2.Nodes) == 0 {
				return false
			}
			if p1, ok := v1.Nodes[0].(*parse.SymbolNode); ok {
				if p2, ok := v2.Nodes[0].(*parse.SymbolNode); ok {
					return p1.Val < p2.Val
				}
				return true
			}
			return false
		}
		return false
	}
	return false
}

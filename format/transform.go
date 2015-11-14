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

func sortImportRequire(node *parse.ListNode) {
	children := node.Children()
	sorted := make([]parse.Node, 0, len(children)/2)
	for _, child := range children[1:] {
		if goclj.Newline(child) {
			continue
		}
		sorted = append(sorted, child)
	}
	sort.Sort(importRequireList(sorted))
	node.Nodes = []parse.Node{children[0]}
	for i, n := range sorted {
		node.Nodes = append(node.Nodes, n)
		if i < len(sorted)-1 {
			node.Nodes = append(node.Nodes, &parse.NewlineNode{})
		}
	}
}

func removeTrailingNewlines(n parse.Node) {
	nodes := n.Children()
	if len(nodes) == 0 {
		return
	}
	switch n.(type) {
	case *parse.ListNode, *parse.MapNode, *parse.VectorNode, *parse.FnLiteralNode, *parse.SetNode:
		for ; len(nodes) > 0; nodes = nodes[:len(nodes)-1] {
			if _, ok := nodes[len(nodes)-1].(*parse.NewlineNode); !ok {
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
	if _, ok := nodes[3].(*parse.VectorNode); !ok {
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
	if _, ok := nodes[3].(*parse.KeywordNode); !ok {
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

type importRequireList []parse.Node

func (l importRequireList) Len() int      { return len(l) }
func (l importRequireList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l importRequireList) Less(i, j int) bool {
	n1, n2 := l[i], l[j]
	if s1, ok := n1.(*parse.SymbolNode); ok {
		if s2, ok := n2.(*parse.SymbolNode); ok {
			return s1.Val < s2.Val
		}
		if _, ok := n2.(*parse.VectorNode); ok {
			return false
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
		return true
	}
	return false
}

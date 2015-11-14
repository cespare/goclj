package format

import (
	"sort"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

// applyTransforms performs small fixes to the tree t.
// So far it only reorders imports and requires.
func applyTransforms(t *parse.Tree) {
	for _, root := range t.Roots {
		if goclj.FnFormSymbol(root, "ns") {
			for _, node := range root.Children()[1:] {
				if goclj.FnFormKeyword(node, ":require", ":import") {
					SortImportRequire(node.(*parse.ListNode))
				}
			}
		}
	}
}

func SortImportRequire(node *parse.ListNode) {
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

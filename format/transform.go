package format

import (
	"sort"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

// A Transform is some tree transformation that can be applied after parsing and
// before printing. Some Transforms may use some heuristics that cause them to
// change code semantics in certain cases; these are clearly indicated and none
// of these are enabled by default.
type Transform int

const (
	// TransformSortImportRequire sorts :import and :require declarations
	// in ns blocks.
	TransformSortImportRequire Transform = iota

	// TransformRemoveTrailingNewlines removes extra newlines following
	// sequence-like forms, so that parentheses are written on the same
	// line. For example,
	//   (foo bar
	//    )
	// becomes
	//   (foo bar)
	TransformRemoveTrailingNewlines

	// TransformFixDefnArglistNewline moves the arg vector of defns to the
	// same line, if appropriate:
	//   (defn foo
	//     [x] ...)
	// becomes
	//   (defn foo [x]
	//     ...)
	// if there's no newline after the arg list.
	TransformFixDefnArglistNewline

	// TransformFixDefmethodDispatchValNewline moves the dispatch-val of a
	// defmethod to the same line, so that
	//   (defmethod foo
	//     :bar
	//     [x] ...)
	// becomes
	//   (defmethod foo :bar
	//     [x] ...)
	TransformFixDefmethodDispatchValNewline

	// TransformRemoveExtraBlankLines consolidates consecutive blank lines
	// into a single blank line.
	TransformRemoveExtraBlankLines

	// TransformUseToRequire consolidates :require and :use blocks inside ns
	// declarations, rewriting them using :require if possible.
	// It is not enabled by default.
	TransformUseToRequire

	// TransformRemoveUnusedRequires uses some simple heuristics to remove
	// some unused :require statements:
	//   [foo :as x] ; if there is no x/y in the ns, this is removed
	//   [foo :refer [x]] ; if x does not appear in the ns, this is removed
	TransformRemoveUnusedRequires
)

var DefaultTransforms = map[Transform]bool{
	TransformSortImportRequire:              true,
	TransformRemoveTrailingNewlines:         true,
	TransformFixDefnArglistNewline:          true,
	TransformFixDefmethodDispatchValNewline: true,
	TransformRemoveExtraBlankLines:          true,
}

func applyTransforms(t *parse.Tree, transforms map[Transform]bool) {
	var syms *symbolCache
	if transforms[TransformRemoveUnusedRequires] {
		syms = findSymbols(t.Roots)
	}
	for _, root := range t.Roots {
		if goclj.FnFormSymbol(root, "ns") {
			if transforms[TransformUseToRequire] {
				useToRequire(root)
			}
			if transforms[TransformRemoveUnusedRequires] {
				removeUnusedRequires(root, syms)
			}
			if transforms[TransformSortImportRequire] {
				sortNS(root)
			}
		}
		if transforms[TransformRemoveTrailingNewlines] {
			removeTrailingNewlines(root)
		}
		if transforms[TransformFixDefnArglistNewline] &&
			goclj.FnFormSymbol(root, "defn") {
			fixDefnArglist(root)
		}
		if transforms[TransformFixDefmethodDispatchValNewline] &&
			goclj.FnFormSymbol(root, "defmethod") {
			fixDefmethodDispatchVal(root)
		}
		if transforms[TransformRemoveExtraBlankLines] {
			removeExtraBlankLinesRecursive(root)
		}
	}
	if transforms[TransformRemoveExtraBlankLines] {
		t.Roots = removeExtraBlankLines(t.Roots)
	}
}

func useToRequire(ns parse.Node) {
	rl := newRequireList()
	insertIndex := -1
	prevSkipped := false
	nodes := []parse.Node{&parse.SymbolNode{Val: "ns"}}
	i := 0
	for _, n := range ns.Children()[1:] {
		if prevSkipped && goclj.Newline(n) {
			prevSkipped = false
			continue
		}
		prevSkipped = false
		if goclj.FnFormKeyword(n, ":require", ":use") {
			ln := n.(*parse.ListNode)
			name := ln.Nodes[0].(*parse.KeywordNode).Val
			rl.parseRequireUse(ln, name == ":use")
			if insertIndex == -1 {
				insertIndex = i + 1
			}
			prevSkipped = true
			continue
		}
		nodes = append(nodes, n)
		i++
	}
	if insertIndex != -1 {
		nodes = append(nodes[:insertIndex],
			append(rl.render(), nodes[insertIndex:]...)...)
	}
	ns.SetChildren(nodes)
}

func removeUnusedRequires(ns parse.Node, syms *symbolCache) {
	nodes := ns.Children()[:0]
	for _, n := range ns.Children() {
		if !goclj.FnFormKeyword(n, ":require") {
			nodes = append(nodes, n)
			continue
		}
		rl := newRequireList()
		rl.parseRequireUse(n.(*parse.ListNode), false)
		for name, r := range rl.m {
			if syms.unused(r) {
				delete(rl.m, name)
			}
		}
		requires := rl.render()[0]
		// If all that's left is (:require), drop it.
		if len(requires.Children()) > 1 {
			nodes = append(nodes, requires)
		}
	}
	ns.SetChildren(nodes)
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
				sorted[len(sorted)-1].commentBeside = node
			} else {
				lineComments = append(lineComments, node)
			}
		case *parse.NewlineNode:
			afterSemanticNode = false
		default:
			ir := &importRequire{
				commentsAbove: lineComments,
				node:          node,
			}
			sorted = append(sorted, ir)
			lineComments = nil
			afterSemanticNode = true
		}
	}
	sort.Stable(sorted)
	newNodes := []parse.Node{nodes[0]}
	for _, ir := range sorted {
		for _, cn := range ir.commentsAbove {
			newNodes = append(newNodes, cn, &parse.NewlineNode{})
		}
		newNodes = append(newNodes, ir.node)
		if ir.commentBeside != nil {
			newNodes = append(newNodes, ir.commentBeside)
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
	commentsAbove []*parse.CommentNode
	commentBeside *parse.CommentNode
	node          parse.Node
}

type importRequireList []*importRequire

func (l importRequireList) Len() int      { return len(l) }
func (l importRequireList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l importRequireList) Less(i, j int) bool {
	// We only consider nodes comparable if they are symbols or
	// lists/vectors with a symbol as a first child. Everything else
	// compares as greater than one of these (and equal to one another).
	k0, ok0 := getImportRequireSortKey(l[i].node)
	k1, ok1 := getImportRequireSortKey(l[j].node)
	if ok0 {
		if ok1 {
			return k0 < k1
		}
		return true // valid < junk
	}
	return false // junk == junk, junk > valid
}

func getImportRequireSortKey(n parse.Node) (key string, ok bool) {
	switch n := n.(type) {
	case *parse.SymbolNode:
		return n.Val, true
	case *parse.ListNode, *parse.VectorNode:
		children := n.Children()
		if len(children) == 0 {
			return "", false
		}
		sym, ok := children[0].(*parse.SymbolNode)
		if !ok {
			return "", false
		}
		return sym.Val, true
	default:
		return "", false
	}
}

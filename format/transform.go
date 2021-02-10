package format

import (
	"sort"
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

// A Transform is some tree transformation that can be applied after parsing and
// before printing. Some Transforms may use some heuristics that cause them to
// change code semantics in certain cases; these are clearly indicated and none
// of these are enabled by default.
type Transform int

const (
	// TransformSortImportRequire sorts :import, :require, and :require-macros
	// declarations in ns blocks.
	TransformSortImportRequire Transform = iota

	// TransformEnforceNSStyle applies a few common ns style rules based on
	// "How to ns". See the README for a list of the rules.
	//
	// TODO: add more "How to ns" conventions such as sorting the vectors
	// within a :require clause. See
	// https://github.com/cespare/goclj/pull/85#issuecomment-777754824
	// for a discussion about this.
	TransformEnforceNSStyle

	// TransformRemoveTrailingNewlines removes extra newlines following
	// sequence-like forms, so that parentheses are written on the same
	// line. For example,
	//
	//   (foo bar
	//    )
	//
	// becomes
	//
	//   (foo bar)
	//
	TransformRemoveTrailingNewlines

	// TransformFixDefnArglistNewline moves the arg vector of defns to the
	// same line, if appropriate:
	//
	//   (defn foo
	//     [x] ...)
	//
	// becomes
	//
	//   (defn foo [x]
	//     ...)
	//
	// if there's no newline after the arg list.
	TransformFixDefnArglistNewline

	// TransformFixDefmethodDispatchValNewline moves the dispatch-val of a
	// defmethod to the same line, so that
	//
	//   (defmethod foo
	//     :bar
	//     [x] ...)
	//
	// becomes
	//
	//   (defmethod foo :bar
	//     [x] ...)
	//
	TransformFixDefmethodDispatchValNewline

	// TransformRemoveExtraBlankLines consolidates consecutive blank lines
	// into a single blank line.
	TransformRemoveExtraBlankLines

	// TransformFixIfNewlineConsistency ensures that if one arm of an if
	// expression is preceded by a newline, the other arm is as well.
	// Both of these:
	//
	//   (if foo? a
	//     b)
	//   (if foo?
	//     a b)
	//
	// become
	//
	//   (if foo?
	//     a
	//     b)
	//
	// This applies to similar forms and macros such as if-not, if-let, and
	// so on. One-line expressions such as (if foo? a b) are not affected.
	TransformFixIfNewlineConsistency

	// TransformUseToRequire consolidates :require and :use blocks inside ns
	// declarations, rewriting them using :require if possible.
	//
	// It is not enabled by default.
	TransformUseToRequire

	// TransformRemoveUnusedRequires uses some simple heuristics to remove
	// some unused :require statements:
	//
	//   [foo :as x] ; if there is no x/y in the ns, this is removed
	//   [foo :refer [x]] ; if x does not appear in the ns, this is removed
	//
	// It is not enabled by default.
	TransformRemoveUnusedRequires
)

var DefaultTransforms = map[Transform]bool{
	TransformSortImportRequire:              true,
	TransformEnforceNSStyle:                 true,
	TransformRemoveTrailingNewlines:         true,
	TransformFixDefnArglistNewline:          true,
	TransformFixDefmethodDispatchValNewline: true,
	TransformRemoveExtraBlankLines:          true,
	TransformFixIfNewlineConsistency:        true,
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
			if transforms[TransformEnforceNSStyle] {
				enforceNSStyle(root)
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
			removeExtraBlankLinesRec(root)
		}
		if transforms[TransformFixIfNewlineConsistency] {
			enforceConsistentIfNewlinesRec(root)
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
			children := n.Children()
			name := children[0].(*parse.KeywordNode).Val
			rl.parseRequireUse(children, name == ":use")
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
	children := ns.Children()
	nodes := children[:0]
	for i := 0; i < len(children); i++ {
		n := children[i]
		if !goclj.FnFormKeyword(n, ":require", ":require-macros") {
			nodes = append(nodes, n)
			continue
		}
		rl := newRequireList()
		rl.parseRequireUse(n.Children(), false)
		for name, r := range rl.m {
			if syms.unused(r) {
				delete(rl.m, name)
			}
		}
		requires := rl.render()[0]
		if len(requires.Children()) <= 2 {
			// If all that's left is (:require), drop it.
			// If there's a newline afterwards, drop that too.
			if i < len(children)-1 && goclj.Newline(children[i+1]) {
				i++
			}
		} else {
			nodes = append(nodes, requires)
		}
	}
	ns.SetChildren(nodes)
}

func enforceNSStyle(ns parse.Node) {
	children := ns.Children()
	for i := 1; i < len(children); i++ {
		n := children[i]
		var isVec bool
		switch n.(type) {
		case *parse.ListNode:
		case *parse.VectorNode:
			isVec = true
		default:
			continue
		}
		clauseChildren := n.Children()
		if len(clauseChildren) == 0 {
			continue
		}
		var clause string
		var isSym bool
		switch c := clauseChildren[0].(type) {
		case *parse.KeywordNode:
			clause = c.Val[1:]
		case *parse.SymbolNode:
			clause = c.Val
			isSym = true
		default:
			continue
		}
		multipleArgs := false
		switch clause {
		case "refer-clojure":
		case "require", "require-macros", "use", "import", "load", "gen-class":
			multipleArgs = true
		default:
			continue
		}
		if isSym {
			clauseChildren[0] = &parse.KeywordNode{Val: ":" + clause}
		}
		if multipleArgs && len(clauseChildren) >= 2 && goclj.Semantic(clauseChildren[1]) {
			clauseChildren = append(
				[]parse.Node{clauseChildren[0], newline},
				clauseChildren[1:]...,
			)
		}
		switch clause {
		case "require", "require-macros":
			enforceRequireStyle(clauseChildren)
		case "import":
			enforceImportStyle(clauseChildren)
		}
		n.SetChildren(clauseChildren)
		if isVec {
			n = &parse.ListNode{Nodes: n.Children()}
		}
		children[i] = n
	}
	ns.SetChildren(children)
}

func enforceRequireStyle(nodes []parse.Node) {
	for i, n := range nodes {
		var v *parse.VectorNode
		switch n := n.(type) {
		case *parse.ListNode:
			v = &parse.VectorNode{Nodes: n.Nodes}
		case *parse.SymbolNode:
			v = &parse.VectorNode{Nodes: []parse.Node{n}}
		case *parse.VectorNode:
			v = n
		}
		if v == nil {
			continue
		}
		for i, child := range v.Nodes {
			if l, ok := child.(*parse.ListNode); ok {
				v.Nodes[i] = &parse.VectorNode{Nodes: l.Nodes}
			}
		}
		nodes[i] = v
	}
}

func enforceImportStyle(nodes []parse.Node) {
	for i, n := range nodes {
		switch n := n.(type) {
		case *parse.VectorNode:
			nodes[i] = &parse.ListNode{Nodes: n.Nodes}
		case *parse.SymbolNode:
			j := strings.LastIndexByte(n.Val, '.')
			if j < 0 {
				break
			}
			nodes[i] = &parse.ListNode{
				Nodes: []parse.Node{
					&parse.SymbolNode{Val: n.Val[:j]},
					&parse.SymbolNode{Val: n.Val[j+1:]},
				},
			}
		}
	}
}

func sortNS(ns parse.Node) {
	for _, n := range ns.Children()[1:] {
		if goclj.FnFormKeyword(n, ":require", ":require-macros", ":import") {
			sortImportRequire(n.(*parse.ListNode))
		}
	}
}

func sortImportRequire(n *parse.ListNode) {
	var (
		nodes             = n.Children()
		sorted            = make(importRequireList, 0, len(nodes)/2)
		lineComments      []*parse.CommentNode
		initialNewline    = false
		afterSemanticNode = false
	)
	for i, node := range nodes[1:] {
		switch node := node.(type) {
		case *parse.CommentNode:
			if afterSemanticNode {
				sorted[len(sorted)-1].commentBeside = node
			} else {
				lineComments = append(lineComments, node)
			}
		case *parse.NewlineNode:
			if i == 0 {
				initialNewline = true
			}
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
	if initialNewline {
		newNodes = append(newNodes, newline)
	}
	for _, ir := range sorted {
		for _, cn := range ir.commentsAbove {
			newNodes = append(newNodes, cn, newline)
		}
		newNodes = append(newNodes, ir.node)
		if ir.commentBeside != nil {
			newNodes = append(newNodes, ir.commentBeside)
		}
		newNodes = append(newNodes, newline)
	}
	// unattached comments at the bottom
	for _, cn := range lineComments {
		newNodes = append(newNodes, cn, newline)
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

func removeExtraBlankLinesRec(n parse.Node) {
	nodes := n.Children()
	if len(nodes) == 0 {
		return
	}
	if len(nodes) > 2 {
		nodes = removeExtraBlankLines(nodes)
		n.SetChildren(nodes)
	}
	for _, node := range nodes {
		removeExtraBlankLinesRec(node)
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

func enforceConsistentIfNewlinesRec(n parse.Node) {
	if goclj.FnFormSymbol(n, "if", "if-not", "if-some", "if-let") {
		n.SetChildren(enforceConsistentIfNewlines(n.Children()))
	}
	for _, child := range n.Children() {
		enforceConsistentIfNewlinesRec(child)
	}
}

func enforceConsistentIfNewlines(nodes []parse.Node) []parse.Node {
	var arm0, arm1 int
	var newlineBeforeArm0, newlineBeforeArm1 bool
	var foundTest bool
	i := 1
	for ; i < len(nodes); i++ {
		n := nodes[i]
		if !goclj.Semantic(n) {
			if goclj.Newline(n) {
				newlineBeforeArm0 = true
			}
			continue
		}
		if !foundTest {
			foundTest = true
			continue
		}
		arm0 = i
		break
	}
	i++
	for ; i < len(nodes); i++ {
		n := nodes[i]
		if !goclj.Semantic(n) {
			if goclj.Newline(n) {
				newlineBeforeArm1 = true
			}
			continue
		}
		arm1 = i
		break
	}
	if arm1 == 0 { // only one arm
		return nodes
	}
	if newlineBeforeArm0 && !newlineBeforeArm1 {
		// (if x?
		//   a b)
		return append(
			nodes[:arm1],
			append([]parse.Node{new(parse.NewlineNode)}, nodes[arm1:]...)...,
		)
	}
	if !newlineBeforeArm0 && newlineBeforeArm1 {
		// (if x? a
		//   b)
		return append(
			nodes[:arm0],
			append([]parse.Node{new(parse.NewlineNode)}, nodes[arm0:]...)...,
		)
	}
	return nodes
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

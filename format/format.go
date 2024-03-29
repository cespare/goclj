package format

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

// A Printer writes a parse tree with proper indentation.
type Printer struct {
	*bufWriter
	// IndentChar is the character used for indentation
	// (by default, ' ' is used).
	IndentChar rune
	// IndentOverrides allow setting specific indentation styles for forms.
	IndentOverrides map[string]IndentStyle
	// ThreadFirstStyleOverrides allow specifying custom thread-first
	// macros.
	ThreadFirstStyleOverrides map[string]ThreadFirstStyle

	// Transforms toggles the set of transformations to apply.
	// This map overrides values in DefaultTransforms.
	Transforms map[Transform]bool

	// indentStyles is the union of defaultIndents and IndentOverrides.
	indentStyles map[string]IndentStyle
	// threadFirstStyles is the union of defaultThreadFirstStyles and
	// ThreadFirstStyleOverrides.
	threadFirstStyles map[string]ThreadFirstStyle
	specialIndent     map[parse.Node]IndentStyle
	threadFirst       map[parse.Node]struct{}
	docstrings        map[*parse.StringNode]struct{}

	// The requires and refers maps track all the require aliases and
	// referred names.
	// Example: given the require [x :as y :refer [z]]:
	requires map[string]string // y -> x
	refers   map[string]string // z -> x
}

// NewPrinter creates a printer to the given writer.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{
		bufWriter:     &bufWriter{bufio.NewWriter(w)},
		IndentChar:    ' ',
		specialIndent: make(map[parse.Node]IndentStyle),
		threadFirst:   make(map[parse.Node]struct{}),
		docstrings:    make(map[*parse.StringNode]struct{}),
		requires:      make(map[string]string),
		refers:        make(map[string]string),
	}
}

// PrintTree writes t to p's writer.
func (p *Printer) PrintTree(t *parse.Tree) (err error) {
	p.indentStyles = make(map[string]IndentStyle)
	for k, v := range defaultIndents {
		p.indentStyles[k] = v
	}
	for k, v := range p.IndentOverrides {
		p.indentStyles[k] = v
	}
	p.threadFirstStyles = make(map[string]ThreadFirstStyle)
	for k, v := range defaultThreadFirstStyles {
		p.threadFirstStyles[k] = v
	}
	for k, v := range p.ThreadFirstStyleOverrides {
		p.threadFirstStyles[k] = v
	}
	if p.Transforms == nil {
		p.Transforms = DefaultTransforms
	} else {
		for k, v := range DefaultTransforms {
			if _, ok := p.Transforms[k]; !ok {
				p.Transforms[k] = v
			}
		}
	}
	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case bufErr:
				err = e
			case fmtErr:
				err = e
			default:
				panic(e)
			}
		}
	}()
	applyTransforms(t, p.Transforms)
	for _, node := range t.Roots {
		p.markDocstrings(node)
		p.markThreadFirsts(node)
		p.markRequires(node)
	}
	p.printSequence(t.Roots, 0, IndentNormal)
	return p.bw.Flush()
}

// printNode prints a representation of node using w, the given indent level
// as a baseline. It returns the new indent.
func (p *Printer) printNode(node parse.Node, w int) int {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			return w + p.writeString("true")
		} else {
			return w + p.writeString("false")
		}
	case *parse.CharacterNode:
		return w + p.writeString(node.Text)
	case *parse.CommentNode:
		return w + p.writeString(node.Text)
	case *parse.DerefNode:
		w += p.writeByte('@')
		return p.printNode(node.Node, w)
	case *parse.FnLiteralNode:
		w += p.writeString("#(")
		w = p.printSequence(node.Nodes, w, p.chooseIndent(node.Nodes))
		return w + p.writeString(")")
	case *parse.ReaderCondNode:
		w += p.writeString("#?(")
		w = p.printSequence(node.Nodes, w, indentBindings)
		return w + p.writeString(")")
	case *parse.ReaderCondSpliceNode:
		w += p.writeString("#?@(")
		w = p.printSequence(node.Nodes, w, indentBindings)
		return w + p.writeString(")")
	case *parse.ReaderDiscardNode:
		w += p.writeString("#_")
		return p.printNode(node.Node, w)
	case *parse.ReaderEvalNode:
		w += p.writeString("#=")
		return p.printNode(node.Node, w)
	case *parse.KeywordNode:
		return w + p.writeString(node.Val)
	case *parse.ListNode:
		p.applySpecialIndentRules(node)
		var style IndentStyle
		var ok bool
		if style, ok = p.specialIndent[node]; ok {
			delete(p.specialIndent, node)
		} else {
			style = p.chooseIndent(node.Nodes)
		}
		if _, ok := p.threadFirst[node]; ok {
			style = style.threadFirstTransform()
		}
		w += p.writeString("(")
		w = p.printSequence(node.Nodes, w, style)
		return w + p.writeString(")")
	case *parse.MapNode:
		if node.Namespace != "" {
			w += p.writeString("#")
			w += p.writeString(node.Namespace)
		}
		w += p.writeString("{")
		w = p.printSequence(node.Nodes, w, indentBindings)
		return w + p.writeString("}")
	case *parse.MetadataNode:
		w += p.writeByte('^')
		return p.printNode(node.Node, w)
	case *parse.NewlineNode:
		panic("should not happen")
	case *parse.NilNode:
		return w + p.writeString("nil")
	case *parse.NumberNode:
		return w + p.writeString(node.Val)
	case *parse.QuoteNode:
		w += p.writeByte('\'')
		return p.printNode(node.Node, w)
	case *parse.RegexNode:
		return w + p.writeString(`#"`+node.Val+`"`)
	case *parse.SetNode:
		w += p.writeString("#{")
		w = p.printSequence(node.Nodes, w, IndentNormal)
		return w + p.writeString("}")
	case *parse.StringNode:
		val := node.Val
		if _, ok := p.docstrings[node]; ok {
			val = p.alignDocstring(val, w)
			delete(p.docstrings, node)
		}
		return w + p.writeString(`"`+val+`"`)
	case *parse.SymbolNode:
		return w + p.writeString(node.Val)
	case *parse.SyntaxQuoteNode:
		w += p.writeByte('`')
		return p.printNode(node.Node, w)
	case *parse.TagNode:
		return w + p.writeString("#"+node.Val)
	case *parse.UnquoteNode:
		w += p.writeByte('~')
		return p.printNode(node.Node, w)
	case *parse.UnquoteSpliceNode:
		w += p.writeString("~@")
		return p.printNode(node.Node, w)
	case *parse.VarQuoteNode:
		return w + p.writeString("#'"+node.Val)
	case *parse.VectorNode:
		style, ok := p.specialIndent[node]
		if ok {
			delete(p.specialIndent, node)
		} else {
			style = IndentNormal
		}
		w += p.writeString("[")
		w = p.printSequence(node.Nodes, w, style)
		return w + p.writeString("]")
	default:
		fmtErrf("%s: unhandled node type %T", node.Position(), node)
	}
	return 0
}

// TODO: Create a simple rules interface or something to easily specify the
// special rules below.

func (p *Printer) applySpecialIndentRules(node *parse.ListNode) {
	if len(node.Nodes) == 0 {
		return
	}
	s, ok := node.Nodes[0].(*parse.SymbolNode)
	if !ok {
		return
	}
	if style, ok := p.indentStyleForSymbol(s.Val); ok {
		switch style {
		case IndentLet:
			p.applySpecialLet(node.Nodes)
		case IndentLetfn:
			p.applySpecialLetfn(node.Nodes)
		case IndentFor:
			p.applySpecialFor(node.Nodes)
		case IndentDeftype:
			p.applySpecialDeftype(node.Nodes)
		}
	}
}

func (p *Printer) applySpecialForLet(nodes []parse.Node) {
	i := -1
	prevLet := false
	for _, node := range nodes {
		if !goclj.Semantic(node) {
			continue
		}
		if prevLet {
			if v, ok := node.(*parse.VectorNode); ok {
				p.specialIndent[v] = indentBindings
			}
		}
		prevLet = false
		i++
		if i%2 != 0 {
			continue
		}
		prevLet = isKeywordNode(node, ":let")
	}
}

func (p *Printer) applySpecialFor(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if !goclj.Semantic(node) {
			continue
		}
		if v, ok := node.(*parse.VectorNode); ok {
			p.specialIndent[v] = indentBindings
			p.applySpecialForLet(v.Nodes)
		}
		return
	}
}

func (p *Printer) applySpecialLet(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if !goclj.Semantic(node) {
			continue
		}
		if v, ok := node.(*parse.VectorNode); ok {
			p.specialIndent[v] = indentBindings
		}
		return
	}
}

func (p *Printer) applySpecialLetfn(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if !goclj.Semantic(node) {
			continue
		}
		v, ok := node.(*parse.VectorNode)
		if !ok {
			return
		}
		for _, n := range v.Nodes {
			if fn, ok := n.(*parse.ListNode); ok {
				p.specialIndent[fn] = IndentListBody
			}
		}
	}
}

func (p *Printer) applySpecialDeftype(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if fn, ok := node.(*parse.ListNode); ok {
			p.specialIndent[fn] = IndentListBody
		}
	}
}

func (p *Printer) chooseIndent(nodes []parse.Node) IndentStyle {
	if len(nodes) == 0 {
		return IndentNormal
	}
	switch node := nodes[0].(type) {
	case *parse.KeywordNode:
		return IndentList
	case *parse.SymbolNode:
		return p.chooseListIndent(node.Val)
	}
	return IndentNormal
}

func (p *Printer) chooseListIndent(name string) IndentStyle {
	if style, ok := p.indentStyleForSymbol(name); ok {
		return style
	}
	name = symbolName(name)
	for _, prefix := range []string{"def", "let", "with-", "when-"} {
		if strings.HasPrefix(name, prefix) {
			return IndentListBody
		}
	}
	return IndentList
}

func (p *Printer) indentStyleForSymbol(name string) (IndentStyle, bool) {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		ns := name[:i]
		unqualified := name[i+1:]
		if req, ok := p.requires[ns]; ok {
			if style, ok := p.indentStyles[req+"/"+unqualified]; ok {
				return style, true
			}
		}
		if style, ok := p.indentStyles[unqualified]; ok {
			return style, true
		}
	} else {
		if req, ok := p.refers[name]; ok {
			if style, ok := p.indentStyles[req+"/"+name]; ok {
				return style, true
			}
		}
	}
	// Fall back to any rule for the name as written.
	style, ok := p.indentStyles[name]
	return style, ok
}

func symbolName(sym string) string {
	if i := strings.LastIndex(sym, "/"); i >= 0 {
		return sym[i+1:]
	}
	return sym
}

// A ThreadFirst style represents a variety of thread-first macro.
type ThreadFirstStyle int

const (
	// ThreadFirstNormal is for thread-first macros that take one argument
	// and thread through all remaining forms. -> and some-> follow this
	// pattern.
	ThreadFirstNormal ThreadFirstStyle = iota
	// ThreadFirstCondArrow is the style used by cond->, which takes one
	// argument and then threads through every other form thereafter.
	ThreadFirstCondArrow
)

var defaultThreadFirstStyles = map[string]ThreadFirstStyle{
	"->":     ThreadFirstNormal,
	"cond->": ThreadFirstCondArrow,
	"some->": ThreadFirstNormal,
}

// An IndentStyle represents the indentation strategy
// used for formatting a sequence of values.
type IndentStyle int

const (
	// IndentNormal is for sequences that introduce no special indentation.
	//   [1
	//    2]
	IndentNormal IndentStyle = iota
	// IndentList is the default list indentation.
	//   (foo bar
	//        baz)
	IndentList
	// IndentListBody is for list forms which have bodies. For these forms,
	// subsequent lines are indented two spaces, rather than being aligned.
	// Forms like this include many language functions and macros like def
	// and defn.
	//   (def x
	//     3)
	//   (defn foo []
	//     bar)
	IndentListBody
	// IndentLet is for let-like forms. This is like IndentListBody, except
	// that the first parameter consists of let-style bindings (the
	// even-numbered ones are indented).
	//   (let [foo
	//           bar])
	IndentLet
	// IndentFor is for for-like forms. This is like IndentLet, except
	// that the let-style bindings can contain ":let" introducing another
	// let-style binding (the even-numbered ones are indented).
	//   (for [foo
	//           bar
	//         :let [x
	//                 baz])
	IndentFor
	// indentBindings is for the paired bindings (usually inside a vector
	// form) of a form indented using IndentLet. It is also used for maps.
	indentBindings
	// IndentLetfn is for letfn or anything that looks like it, where the
	// binding vector contains function bodies that should be themselves
	// indented using IndentListBody.
	//   (letfn [(twice [x]
	//              (* x 2))
	//           (six-times [y]
	//              (* (twice y) 3))]
	//     (println "Twice 15 =" (twice 15))
	//     (println "Six times 15 =" (six-times 15)))
	IndentLetfn
	// IndentDeftype is used for macros similar to deftype that define
	// functions/methods that themselves should be indented using
	// IndentListBody.
	//   (defrecord Foo [x y z]
	//     Xer
	//     (foobar [this]
	//       this)
	//     (baz [this a b c]
	//       (+ a b c)))
	IndentDeftype
	// IndentCond0 is like IndentListBody but the even-numbered arguments
	// are further indented by two.
	//   (cond
	//     (> a 10)
	//       foo
	//     (> a 5)
	//       bar)
	// As a special case, to account for condp, when the symbol :>> occurs
	// in an even-numbered position, the parity of the subsequent arguments
	// is offset by one.
	IndentCond0
	// IndentCond1 is like IndentCond0 except that it ignores 1 body
	// parameter.
	//   (case x
	//     "one"
	//       1
	//     "two"
	//       2)
	IndentCond1
	// IndentCond2 is like IndentCond0 except that it ignores 2 body
	// parameters.
	//   (condp = value
	//     1
	//       "one"
	//     2
	//       "two"
	//     3
	//       "three")
	IndentCond2
	// IndentCond4 is like IndentCond0 except that it ignores 4 body
	// parameters.
	IndentCond4
)

var defaultIndents = map[string]IndentStyle{
	"as->":            IndentListBody,
	"assoc":           IndentCond1,
	"binding":         IndentLet,
	"bound-fn":        IndentListBody,
	"case":            IndentCond1,
	"catch":           IndentListBody,
	"cond":            IndentCond0,
	"cond->":          IndentCond1,
	"cond->>":         IndentCond1,
	"condp":           IndentCond2,
	"def":             IndentListBody,
	"definline":       IndentListBody,
	"definterface":    IndentDeftype,
	"defmacro":        IndentListBody,
	"defmethod":       IndentListBody,
	"defmulti":        IndentListBody,
	"defn":            IndentListBody,
	"defn-":           IndentListBody,
	"defonce":         IndentListBody,
	"defproject":      IndentCond2,
	"defprotocol":     IndentDeftype,
	"defrecord":       IndentDeftype,
	"defstruct":       IndentListBody,
	"deftest":         IndentListBody,
	"deftest-":        IndentListBody,
	"deftype":         IndentDeftype,
	"doseq":           IndentFor,
	"dotimes":         IndentLet,
	"doto":            IndentListBody,
	"extend":          IndentListBody,
	"extend-protocol": IndentDeftype,
	"extend-type":     IndentDeftype,
	"fn":              IndentListBody,
	"for":             IndentFor,
	"if":              IndentListBody,
	"if-let":          IndentLet,
	"if-not":          IndentListBody,
	"if-some":         IndentLet,
	"let":             IndentLet,
	"letfn":           IndentLetfn,
	"locking":         IndentListBody,
	"loop":            IndentLet,
	"ns":              IndentListBody,
	"proxy":           IndentDeftype,
	"reify":           IndentDeftype,
	"send-off":        IndentListBody,
	"set-test":        IndentListBody,
	"testing":         IndentListBody,
	"update":          IndentListBody,
	"update-in":       IndentListBody,
	"when":            IndentListBody,
	"when-first":      IndentLet,
	"when-let":        IndentLet,
	"when-not":        IndentListBody,
	"when-some":       IndentLet,
	"while":           IndentListBody,
	"with-bindings":   IndentListBody,
	"with-in-str":     IndentListBody,
	"with-local-vars": IndentLet,
	"with-open":       IndentLet,
	"with-precision":  IndentListBody,
	"with-redefs":     IndentLet,
	"with-redefs-fn":  IndentListBody,
	"with-test":       IndentListBody,
}

var indentExtraOffsets = [...]int{
	indentBindings: 0,
	IndentCond0:    1,
	IndentCond1:    2,
	IndentCond2:    3,
	IndentCond4:    5,
}

func (style IndentStyle) threadFirstTransform() IndentStyle {
	switch style {
	case IndentCond1:
		return IndentCond0
	case IndentCond2:
		return IndentCond1
	}
	return style
}

// indentListMaxCommentAlign is the maximum length of a list form name that will
// still cause the arguments to be aligned using IndentList default style if the
// element with which they're being aligned with is a comment.
// For example:
// (foobar ; len(foobar) < indentListMaxCommentAlign
//         1
//         2)
//
// but
// (foobar-blah-blah-blah ; len(foobar-blah-blah-blah) > indentListMaxCommentAlign
//   1
//   2)
const indentListMaxCommentAlign = 12

func (p *Printer) printSequence(nodes []parse.Node, w int, style IndentStyle) int {
	var (
		w2         = w
		needSpace  = false
		needIndent = false

		// firstIndent and firstLen are used for IndentList and IndentCondX
		// for tracking indent based on nodes[0].
		firstIndent int
		firstLen    int

		// pairStartIdx says how many nodes precede the paired elements
		// in forms such as cond.
		pairStartIdx = indentExtraOffsets[style]

		// semanticIdx counts the number of semantic nodes we've
		// printed.
		semanticIdx int

		// pairIdx is used for forms with paired elements (IndentList
		// and IndentCondX). It indicates the index of paired elements
		// after we reach them (once semanticIdx reaches pairStartIdx).
		// Before that point, it is -1.
		//
		// As a special case, the symbol :>> (used by condp) is part of
		// a triple (rather than a pair) of nodes.
		//
		// Here are example values of pairIdx:
		//
		// (condp         ; -1
		//    some        ; -1
		//    [1 2 3 4]   ; -1
		//    #{0 6 7}    ; 0
		//      :>>       ; 1
		//      inc       ; 2
		//    #{4 5 9}    ; 0
		//      123)      ; 1
		pairIdx int

		// If we had a newline during a paired element, extraIndent
		// indicates this so that we can remove the indent afterward.
		extraIndent = false
	)
	if pairStartIdx > 0 {
		pairIdx = -1
	}
	for i, n := range nodes {
		if goclj.Newline(n) {
			switch style {
			case IndentList,
				IndentListBody,
				IndentLet,
				IndentLetfn,
				IndentFor,
				IndentDeftype:
				if i == 1 {
					w++
				}
			case indentBindings,
				IndentCond0,
				IndentCond1,
				IndentCond2,
				IndentCond4:
				if pairIdx > 0 {
					w += 2
					extraIndent = true
				} else if i == 1 && semanticIdx == 1 {
					w++
				}
			}
			w2 = w
			p.writeByte('\n')
			needIndent = true
			needSpace = false
			continue
		}

		semantic := goclj.Semantic(n)

		switch style {
		case IndentList, IndentCond0:
			if i == 1 {
				if !semantic && firstLen > indentListMaxCommentAlign {
					w++
				} else {
					w = firstIndent + 1
				}
			}
		case IndentNormal, indentBindings:
		default:
			if i == 1 {
				w++
			}
		}
		if needIndent {
			p.writeString(strings.Repeat(string(p.IndentChar), w))
		}
		if needSpace {
			w2 += p.writeByte(' ')
		}
		w2 = p.printNode(n, w2)
		if i == 0 {
			firstIndent = w2
			firstLen = w2 - w
		}
		needIndent = false
		needSpace = true
		if extraIndent {
			w -= 2
			extraIndent = false
		}

		if !semantic {
			continue
		}
		semanticIdx++
		if pairIdx >= 0 {
			pairIdx++
		}

		if semanticIdx == pairStartIdx ||
			(pairIdx == 2 && !isKeywordNode(n, ":>>")) || pairIdx > 2 {
			pairIdx = 0
		}
	}
	// We need to put in a trailing indent here; the next token cannot be a
	// newline (it will need to be the closing delimiter for this sequence).
	if needIndent {
		p.writeString(strings.Repeat(string(p.IndentChar), w))
	}
	return w2
}

func isKeywordNode(n parse.Node, kw string) bool {
	kn, ok := n.(*parse.KeywordNode)
	if !ok {
		return false
	}
	return kn.Val == kw
}

type bufWriter struct {
	bw *bufio.Writer
}

type bufErr struct{ error }

func (bw *bufWriter) writeString(s string) int {
	n, err := bw.bw.WriteString(s)
	if err != nil {
		panic(bufErr{err})
	}
	return n
}
func (bw *bufWriter) writeByte(b byte) int {
	if err := bw.bw.WriteByte(b); err != nil {
		panic(bufErr{err})
	}
	return 1
}

type fmtErr string

func (e fmtErr) Error() string { return string(e) }

func fmtErrf(format string, args ...interface{}) {
	panic(fmtErr(fmt.Sprintf(format, args...)))
}

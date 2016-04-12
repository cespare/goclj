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
	// IndentSpecial are extra names that are given two-space indentation
	// regardless of length (the same way that defn, let, with* and many
	// others are).
	IndentSpecial []string

	// indentSpecialSet is the union of indentSpecialDefaults and
	// IndentSpecial.
	indentSpecialSet map[string]struct{}
	specialIndent    map[parse.Node]indentStyle
	docstrings       map[*parse.StringNode]struct{}
}

// NewPrinter creates a printer to the given writer.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{
		bufWriter:     &bufWriter{bufio.NewWriter(w)},
		IndentChar:    ' ',
		specialIndent: make(map[parse.Node]indentStyle),
		docstrings:    make(map[*parse.StringNode]struct{}),
	}
}

// PrintTree writes t to p's writer.
func (p *Printer) PrintTree(t *parse.Tree) (err error) {
	p.indentSpecialSet = make(map[string]struct{})
	for _, s := range indentSpecialDefaults {
		p.indentSpecialSet[s] = struct{}{}
	}
	for _, s := range p.IndentSpecial {
		p.indentSpecialSet[s] = struct{}{}
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
	applyTransforms(t)
	for _, node := range t.Roots {
		p.markDocstrings(node)
	}
	p.printSequence(t.Roots, 0, indentNormal)
	return p.bw.Flush()
}

// printNode prints a representation of node using w, the given indent level
// as a baseline. It returns the new indent.
func (p *Printer) printNode(node parse.Node, w int) int {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			return w + p.WriteString("true")
		} else {
			return w + p.WriteString("false")
		}
	case *parse.CharacterNode:
		return w + p.WriteString(node.Text)
	case *parse.CommentNode:
		return w + p.WriteString(node.Text)
	case *parse.DerefNode:
		w += p.WriteByte('@')
		return p.printNode(node.Node, w)
	case *parse.FnLiteralNode:
		w += p.WriteString("#(")
		w = p.printSequence(node.Nodes, w, p.chooseIndent(node.Nodes))
		return w + p.WriteString(")")
	case *parse.IgnoreFormNode:
		w += p.WriteString("#_")
		return p.printNode(node.Node, w)
	case *parse.KeywordNode:
		return w + p.WriteString(node.Val)
	case *parse.ListNode:
		p.applySpecialIndentRules(node)
		var style indentStyle
		var ok bool
		if style, ok = p.specialIndent[node]; ok {
			delete(p.specialIndent, node)
		} else {
			style = p.chooseIndent(node.Nodes)
		}
		w += p.WriteString("(")
		w = p.printSequence(node.Nodes, w, style)
		return w + p.WriteString(")")
	case *parse.MapNode:
		w += p.WriteString("{")
		w = p.printSequence(node.Nodes, w, indentNormal)
		return w + p.WriteString("}")
	case *parse.MetadataNode:
		w += p.WriteByte('^')
		return p.printNode(node.Node, w)
	case *parse.NewlineNode:
		panic("should not happen")
	case *parse.NilNode:
		return w + p.WriteString("nil")
	case *parse.NumberNode:
		return w + p.WriteString(node.Val)
	case *parse.QuoteNode:
		w += p.WriteByte('\'')
		return p.printNode(node.Node, w)
	case *parse.RegexNode:
		return w + p.WriteString(`#"`+node.Val+`"`)
	case *parse.SetNode:
		w += p.WriteString("#{")
		w = p.printSequence(node.Nodes, w, indentNormal)
		return w + p.WriteString("}")
	case *parse.StringNode:
		val := node.Val
		if _, ok := p.docstrings[node]; ok {
			val = p.alignDocstring(val, w)
			delete(p.docstrings, node)
		}
		return w + p.WriteString(`"`+val+`"`)
	case *parse.SymbolNode:
		return w + p.WriteString(node.Val)
	case *parse.SyntaxQuoteNode:
		w += p.WriteByte('`')
		return p.printNode(node.Node, w)
	case *parse.TagNode:
		return w + p.WriteString("#"+node.Val)
	case *parse.UnquoteNode:
		w += p.WriteByte('~')
		return p.printNode(node.Node, w)
	case *parse.UnquoteSpliceNode:
		w += p.WriteString("~@")
		return p.printNode(node.Node, w)
	case *parse.VarQuoteNode:
		return w + p.WriteString("#'"+node.Val)
	case *parse.VectorNode:
		style, ok := p.specialIndent[node]
		if ok {
			delete(p.specialIndent, node)
		} else {
			style = indentNormal
		}
		w += p.WriteString("[")
		w = p.printSequence(node.Nodes, w, style)
		return w + p.WriteString("]")
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
	switch symbolName(s.Val) {
	case "let":
		p.applySpecialLet(node.Nodes)
	case "letfn":
		p.applySpecialLetfn(node.Nodes)
	case "deftype", "defrecord":
		p.applySpecialDeftype(node.Nodes)
	}
}

func (p *Printer) applySpecialLet(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if goclj.Newline(node) {
			continue
		}
		if v, ok := node.(*parse.VectorNode); ok {
			p.specialIndent[v] = indentLet
		}
		return
	}
}

func (p *Printer) applySpecialLetfn(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if goclj.Newline(node) {
			continue
		}
		v, ok := node.(*parse.VectorNode)
		if !ok {
			return
		}
		for _, n := range v.Nodes {
			if fn, ok := n.(*parse.ListNode); ok {
				p.specialIndent[fn] = indentListSpecial
			}
		}
	}
}

func (p *Printer) applySpecialDeftype(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if fn, ok := node.(*parse.ListNode); ok {
			p.specialIndent[fn] = indentListSpecial
		}
	}
}

func (p *Printer) chooseIndent(nodes []parse.Node) indentStyle {
	if len(nodes) == 0 {
		return indentNormal
	}
	switch node := nodes[0].(type) {
	case *parse.KeywordNode:
		return indentList
	case *parse.SymbolNode:
		switch node.Val {
		case "cond":
			return indentCond
		case "case", "cond->", "cond->>":
			return indentCond2
		case "condp":
			return indentCondp
		}
		if p.special(node) {
			return indentListSpecial
		}
		return indentList
	}
	return indentNormal
}

var (
	// TODO(caleb): I wish I had written down where I got this list...
	indentSpecialDefaults = []string{
		"as->", "binding", "bound-fn", "case", "catch", "cond->", "cond->>",
		"condp", "def", "definline", "definterface", "defmacro", "defmethod",
		"defmulti", "defn", "defn-", "defonce", "defprotocol", "defrecord",
		"defstruct", "deftest", "deftest-", "deftype", "doseq", "dotimes", "doto",
		"extend", "extend-protocol", "extend-type", "fn", "for", "if", "if-let",
		"if-not", "if-some", "let", "letfn", "locking", "loop", "ns", "proxy",
		"reify", "set-test", "testing", "when", "when-first", "when-let",
		"when-not", "when-some", "while", "with-bindings", "with-in-str",
		"with-local-vars", "with-open", "with-precision", "with-redefs",
		"with-redefs-fn", "with-test",
	}
	indentSpecialPrefixes = []string{
		"def", "let", "send", "with", "when",
	}
)

func (p *Printer) special(node *parse.SymbolNode) bool {
	name := symbolName(node.Val)
	if _, ok := p.indentSpecialSet[name]; ok {
		return true
	}
	for _, prefix := range indentSpecialPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func symbolName(sym string) string {
	if i := strings.LastIndex(sym, "/"); i >= 0 {
		return sym[i+1:]
	}
	return sym
}

type indentStyle int

const (
	indentNormal      indentStyle = iota // [1\n2] ; 2 is below 1
	indentList                           // (foo bar\nbaz) ; baz is below bar
	indentListSpecial                    // (defn foo []\nbar) ; bar is indented 2
	indentLet                            // (let [foo\nbar]) ; bar is indented two beyond foo
	indentCond                           // like indentLet but starting on the second element
	indentCond2                          // like indentLet but starting on the third element
	indentCondp                          // like indentLet but starting on the fourth element
)

var indentExtraOffsets = [...]int{
	indentLet:   0,
	indentCond:  1,
	indentCond2: 2,
	indentCondp: 3,
}

func (p *Printer) printSequence(nodes []parse.Node, w int, style indentStyle) int {
	var (
		w2         = w
		needSpace  = false
		needIndent = false

		// used for IndentList, for tracking indent based on nodes[0]
		firstIndent int

		// used by IndentLet, indexCond, indexCase, and indexCondp
		// for counting semantic tokens
		idxSemantic int
		extraIndent = false
	)
	for i, n := range nodes {
		if goclj.Newline(n) {
			switch style {
			case indentList, indentListSpecial:
				if i == 1 {
					w++
				}
			case indentLet, indentCond, indentCond2, indentCondp:
				off := indentExtraOffsets[style]
				if idxSemantic >= 1 && idxSemantic <= off {
					// Fall back to indentListSpecial in this case.
					// Example:
					// (case
					//    foo
					//    "a" b)
					// The 'foo' is indented like indentListSpecial.
					if i == 1 {
						w++
					}
				}
				if idxSemantic > off && (idxSemantic-off)%2 == 1 {
					w += 2
					extraIndent = true
				}
			}
			w2 = w
			p.WriteByte('\n')
			needIndent = true
			needSpace = false
			continue
		}
		if goclj.Semantic(n) {
			idxSemantic++
		}
		switch style {
		case indentList:
			if i == 1 {
				w = firstIndent + 1
			}
		case indentListSpecial, indentCond, indentCond2, indentCondp:
			if i == 1 {
				w++
			}
		}
		if needIndent {
			p.WriteString(strings.Repeat(string(p.IndentChar), w))
		}
		if needSpace {
			w2 += p.WriteByte(' ')
		}
		w2 = p.printNode(n, w2)
		if i == 0 {
			firstIndent = w2
		}
		needIndent = false
		needSpace = true
		if extraIndent {
			w -= 2
			extraIndent = false
		}
	}
	// We need to put in a trailing indent here; the next token cannot be a
	// newline (it will need to be the closing delimiter for this sequence).
	if needIndent {
		p.WriteString(strings.Repeat(string(p.IndentChar), w))
	}
	return w2
}

type bufWriter struct {
	bw *bufio.Writer
}

type bufErr struct{ error }

func (bw *bufWriter) Write(b []byte) (int, error) {
	n, err := bw.bw.Write(b)
	if err != nil {
		panic(bufErr{err})
	}
	return n, nil
}

func (bw *bufWriter) WriteString(s string) int {
	n, err := bw.bw.WriteString(s)
	if err != nil {
		panic(bufErr{err})
	}
	return n
}
func (bw *bufWriter) WriteByte(b byte) int {
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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/cespare/diff"
	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

type Printer struct {
	*bufWriter
	IndentChar byte

	specialIndent map[parse.Node]IndentStyle
}

func NewPrinter(w io.Writer) *Printer {
	return &Printer{
		bufWriter:     &bufWriter{bufio.NewWriter(w)},
		IndentChar:    ' ',
		specialIndent: make(map[parse.Node]IndentStyle),
	}
}

func (p *Printer) PrintTree(t *parse.Tree) (err error) {
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
	ApplyTransforms(t)
	p.PrintSequence(t.Roots, 0, IndentNormal)
	return p.bw.Flush()
}

// ApplyTransforms performs small fixes to the tree t:
//
//  - Reordering imports and requires
//
func ApplyTransforms(t *parse.Tree) {
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

// PrintNode prints a representation of node using w, the given indent level, as a baseline.
// It returns the new indent.
func (p *Printer) PrintNode(node parse.Node, w int) int {
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
		return p.PrintNode(node.Node, w)
	case *parse.FnLiteralNode:
		w += p.WriteString("#(")
		w = p.PrintSequence(node.Nodes, w, chooseIndent(node.Nodes))
		return w + p.WriteString(")")
	case *parse.IgnoreFormNode:
		w += p.WriteString("#_")
		return p.PrintNode(node.Node, w)
	case *parse.KeywordNode:
		return w + p.WriteString(node.Val)
	case *parse.ListNode:
		p.applySpecialIndentRules(node)
		var style IndentStyle
		var ok bool
		if style, ok = p.specialIndent[node]; ok {
			delete(p.specialIndent, node)
		} else {
			style = chooseIndent(node.Nodes)
		}
		w += p.WriteString("(")
		w = p.PrintSequence(node.Nodes, w, style)
		return w + p.WriteString(")")
	case *parse.MapNode:
		w += p.WriteString("{")
		w = p.PrintSequence(node.Nodes, w, IndentNormal)
		return w + p.WriteString("}")
	case *parse.MetadataNode:
		w += p.WriteByte('^')
		return p.PrintNode(node.Node, w)
	case *parse.NewlineNode:
		panic("should not happen")
	case *parse.NilNode:
		return w + p.WriteString("nil")
	case *parse.NumberNode:
		return w + p.WriteString(node.Val)
	case *parse.QuoteNode:
		w += p.WriteByte('\'')
		return p.PrintNode(node.Node, w)
	case *parse.RegexNode:
		return w + p.WriteString(`#"`+node.Val+`"`)
	case *parse.SetNode:
		w += p.WriteString("#{")
		w = p.PrintSequence(node.Nodes, w, IndentNormal)
		return w + p.WriteString("}")
	case *parse.StringNode:
		return w + p.WriteString(`"`+node.Val+`"`)
	case *parse.SymbolNode:
		return w + p.WriteString(node.Val)
	case *parse.SyntaxQuoteNode:
		w += p.WriteByte('`')
		return p.PrintNode(node.Node, w)
	case *parse.TagNode:
		return w + p.WriteString("#"+node.Val)
	case *parse.UnquoteNode:
		w += p.WriteByte('~')
		return p.PrintNode(node.Node, w)
	case *parse.UnquoteSpliceNode:
		w += p.WriteString("~@")
		return p.PrintNode(node.Node, w)
	case *parse.VarQuoteNode:
		return w + p.WriteString("#'"+node.Val)
	case *parse.VectorNode:
		style, ok := p.specialIndent[node]
		if ok {
			delete(p.specialIndent, node)
		} else {
			style = IndentNormal
		}
		w += p.WriteString("[")
		w = p.PrintSequence(node.Nodes, w, style)
		return w + p.WriteString("]")
	default:
		FmtErrf("%s: unhandled node type %T", node.Position(), node)
	}
	return 0
}

// TODO: Create a simple rules interface or something to easily specify the special rules below.

func (p *Printer) applySpecialIndentRules(node *parse.ListNode) {
	if len(node.Nodes) == 0 {
		return
	}
	s, ok := node.Nodes[0].(*parse.SymbolNode)
	if !ok {
		return
	}
	switch s.Val {
	case "let":
		p.applySpecialLet(node.Nodes)
	case "letfn":
		p.applySpecialLetfn(node.Nodes)
	case "deftype":
		p.applySpecialDeftype(node.Nodes)
	}
}

func (p *Printer) applySpecialLet(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if goclj.Newline(node) {
			continue
		}
		if v, ok := node.(*parse.VectorNode); ok {
			p.specialIndent[v] = IndentLet
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
				p.specialIndent[fn] = IndentListSpecial
			}
		}
	}
}

func (p *Printer) applySpecialDeftype(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if fn, ok := node.(*parse.ListNode); ok {
			p.specialIndent[fn] = IndentListSpecial
		}
	}
}

func chooseIndent(nodes []parse.Node) IndentStyle {
	if len(nodes) == 0 {
		return IndentNormal
	}
	switch node := nodes[0].(type) {
	case *parse.KeywordNode:
		return IndentList
	case *parse.SymbolNode:
		if special(node) {
			return IndentListSpecial
		}
		return IndentList
	}
	return IndentNormal
}

var indentSpecialRegex = regexp.MustCompile(
	`^(def.*|let.*|send.*|with.*)$`,
)

var indentSpecial = make(map[string]struct{})

func init() {
	for _, word := range []string{
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
	} {
		indentSpecial[word] = struct{}{}
	}
}

func special(node *parse.SymbolNode) bool {
	if _, ok := indentSpecial[node.Val]; ok {
		return true
	}
	return indentSpecialRegex.MatchString(node.Val)
}

type IndentStyle int

const (
	IndentNormal      IndentStyle = iota // [1\n2] ; 2 is below 1
	IndentList                           // (foo bar\nbaz) ; baz is below bar
	IndentListSpecial                    // (defn foo []\nbar) ; bar is indented 2
	IndentLet                            // (let [foo\nbar]) ; bar is indented two beyond foo
)

func (p *Printer) PrintSequence(nodes []parse.Node, w int, indentStyle IndentStyle) int {
	var (
		w2         = w
		needSpace  = false
		needIndent = false

		// used for IndentList, for tracking indent based on nodes[0]
		firstIndent int

		// used for IndentLet, for counting semantic tokens
		idxSemantic int
		letIndent   = false
	)
	for i, n := range nodes {
		if goclj.Newline(n) {
			switch indentStyle {
			case IndentList, IndentListSpecial:
				if i == 1 {
					w++
				}
			case IndentLet:
				if idxSemantic%2 == 1 {
					w += 2
					letIndent = true
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
		switch indentStyle {
		case IndentList:
			if i == 1 {
				w = firstIndent + 1
			}
		case IndentListSpecial:
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
		w2 = p.PrintNode(n, w2)
		if i == 0 {
			firstIndent = w2
		}
		needIndent = false
		needSpace = true
		if letIndent {
			w -= 2
			letIndent = false
		}
	}
	// We need to put in a trailing indent here; the next token cannot be a newline
	// (it will need to be the closing delimiter for this sequence).
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
	bw.Write([]byte(s))
	return len(s)
}
func (bw *bufWriter) WriteByte(b byte) int {
	bw.Write([]byte{b})
	return 1
}

type fmtErr string

func (e fmtErr) Error() string { return string(e) }

func FmtErrf(format string, args ...interface{}) {
	panic(fmtErr(fmt.Sprintf(format, args...)))
}

func main() {
	var (
		indentCharFlag = flag.String("indentchar", " ", "character to use for indenting")
		listDifferent  = flag.Bool("l", false, "print files whose formatting differs from cljfmt's")
		writeFile      = flag.Bool("w", false, "write result to (source) file instead of stdout")
	)
	flag.Parse()
	if len(*indentCharFlag) != 1 {
		fatalf("-indentchar arg must have length 1")
	}
	indentChar := (*indentCharFlag)[0]
	if flag.NArg() < 1 {
		usage()
	}
	if *listDifferent || *writeFile {
		for _, filename := range flag.Args() {
			if err := writeFormatted(filename, indentChar, *listDifferent, *writeFile); err != nil {
				fatal(err)
			}
		}
		return
	}

	if flag.NArg() > 1 {
		fatalf("must provide a single file unless -l or -w are given")
	}

	t, err := parse.File(flag.Arg(0), true)
	if err != nil {
		fatal(err)
	}
	p := NewPrinter(os.Stdout)
	p.IndentChar = indentChar
	if err := p.PrintTree(t); err != nil {
		fatal(err)
	}
}

func writeFormatted(filename string, indentChar byte, listDifferent, writeFile bool) error {
	tw, err := ioutil.TempFile("", "cljfmt-")
	if err != nil {
		return err
	}
	defer os.Remove(tw.Name())
	defer tw.Close()

	t, err := parse.File(filename, true)
	if err != nil {
		return err
	}
	p := NewPrinter(tw)
	p.IndentChar = indentChar
	if err := p.PrintTree(t); err != nil {
		return err
	}
	tw.Close()
	different, err := diff.Files(filename, tw.Name())
	if err != nil {
		return err
	}
	if !different {
		return nil
	}
	if listDifferent {
		fmt.Println(filename)
	}
	if writeFile {
		if err := os.Rename(tw.Name(), filename); err != nil {
			return err
		}
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [flags] [path ...]\n", os.Args[0])
	os.Exit(1)
}

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

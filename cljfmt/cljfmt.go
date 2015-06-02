package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/cespare/diff"
	"github.com/cespare/goclj/parse"
)

type Printer struct {
	*bufWriter
	IndentChar byte

	specialIndent map[*parse.ListNode]struct{}
}

func NewPrinter(w io.Writer) *Printer {
	return &Printer{
		bufWriter:     &bufWriter{bufio.NewWriter(w)},
		IndentChar:    ' ',
		specialIndent: make(map[*parse.ListNode]struct{}),
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
	p.PrintSequence(t.Roots, 0, IndentNormal)
	return p.bw.Flush()
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
		if _, ok := p.specialIndent[node]; ok {
			style = IndentListSpecial
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
		w += p.WriteString("[")
		w = p.PrintSequence(node.Nodes, w, IndentNormal)
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
	case "letfn":
		p.applySpecialLetfn(node.Nodes)
	case "deftype":
		p.applySpecialDeftype(node.Nodes)
	}
}

func (p *Printer) applySpecialLetfn(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if isNewline(node) {
			continue
		}
		v, ok := node.(*parse.VectorNode)
		if !ok {
			return
		}
		for _, n := range v.Nodes {
			if fn, ok := n.(*parse.ListNode); ok {
				p.specialIndent[fn] = struct{}{}
			}
		}
	}
}

func (p *Printer) applySpecialDeftype(nodes []parse.Node) {
	for _, node := range nodes[1:] {
		if fn, ok := node.(*parse.ListNode); ok {
			p.specialIndent[fn] = struct{}{}
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
)

func (p *Printer) PrintSequence(nodes []parse.Node, w int, indentStyle IndentStyle) int {
	var (
		w2          = w
		needSpace   = false
		needIndent  = false
		firstIndent int // used for IndentList, for tracking indent based on nodes[0]
	)
	for i, n := range nodes {
		if isNewline(n) {
			if indentStyle != IndentNormal && i == 1 {
				w++
			}
			w2 = w
			p.WriteByte('\n')
			needIndent = true
			needSpace = false
			continue
		}
		if i == 1 {
			switch indentStyle {
			case IndentList:
				w = firstIndent + 1
			case IndentListSpecial:
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
	}
	// We need to put in a trailing indent here; the next token cannot be a newline
	// (it will need to be the closing delimiter for this sequence).
	if needIndent {
		p.WriteString(strings.Repeat(string(p.IndentChar), w))
	}
	return w2
}

func isNewline(node parse.Node) bool {
	_, ok := node.(*parse.NewlineNode)
	return ok
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

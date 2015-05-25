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
}

func NewPrinter(w io.Writer) *Printer {
	return &Printer{
		bufWriter:  &bufWriter{bufio.NewWriter(w)},
		IndentChar: ' ',
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
	p.PrintSequence(t.Roots, 0, false)
	return p.bw.Flush()
}

func (p *Printer) PrintNode(node parse.Node, indent int) {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			p.WriteString("true")
		} else {
			p.WriteString("false")
		}
	case *parse.CharacterNode:
		p.WriteString(node.Text)
	case *parse.CommentNode:
		p.WriteString(node.Text)
	case *parse.DerefNode:
		p.WriteByte('@')
		p.PrintNode(node.Node, indent+1)
	case *parse.FnLiteralNode:
		p.WriteString("#(")
		p.PrintSequence(node.Nodes, indent+2, true)
		p.WriteString(")")
	case *parse.IgnoreFormNode:
		p.WriteString("#_")
		p.PrintNode(node.Node, indent+2)
	case *parse.KeywordNode:
		p.WriteString(node.Val)
	case *parse.ListNode:
		p.WriteString("(")
		p.PrintSequence(node.Nodes, indent+1, true)
		p.WriteString(")")
	case *parse.MapNode:
		p.WriteString("{")
		p.PrintSequence(node.Nodes, indent+1, false)
		p.WriteString("}")
	case *parse.MetadataNode:
		p.WriteByte('^')
		p.PrintNode(node.Node, indent+1)
	case *parse.NewlineNode:
		panic("should not happen")
	case *parse.NilNode:
		p.WriteString("nil")
	case *parse.NumberNode:
		p.WriteString(node.Val)
	case *parse.QuoteNode:
		p.WriteByte('\'')
		p.PrintNode(node.Node, indent+1)
	case *parse.RegexNode:
		p.WriteString(`#"` + node.Val + `"`)
	case *parse.SetNode:
		p.WriteString("#{")
		p.PrintSequence(node.Nodes, indent+2, false)
		p.WriteString("}")
	case *parse.StringNode:
		p.WriteString(`"` + node.Val + `"`)
	case *parse.SymbolNode:
		p.WriteString(node.Val)
	case *parse.SyntaxQuoteNode:
		p.WriteByte('`')
		p.PrintNode(node.Node, indent+1)
	case *parse.TagNode:
		p.WriteString("#" + node.Val)
	case *parse.UnquoteNode:
		p.WriteByte('~')
		p.PrintNode(node.Node, indent+1)
	case *parse.UnquoteSpliceNode:
		p.WriteString("~@")
		p.PrintNode(node.Node, indent+2)
	case *parse.VarQuoteNode:
		p.WriteString("#'" + node.Val)
	case *parse.VectorNode:
		p.WriteString("[")
		p.PrintSequence(node.Nodes, indent+1, false)
		p.WriteString("]")
	default:
		FmtErrf("%s: unhandled node type %T", node.Position(), node)
	}
}

func (p *Printer) PrintSequence(nodes []parse.Node, indent int, listIndent bool) {
	prevNewline := false
	subIndent := indent
	for i, n := range nodes {
		if _, ok := n.(*parse.NewlineNode); ok {
			if listIndent && i == 1 {
				indent++
			}
			subIndent = indent
			p.WriteByte('\n')
			p.WriteString(strings.Repeat(string(p.IndentChar), indent))
			prevNewline = true
			continue
		}
		if listIndent && i == 1 {
			indent += ListIndentWidth(nodes[0])
		}
		if !prevNewline && i > 0 {
			p.WriteByte(' ')
		}
		p.PrintNode(n, subIndent)
		subIndent += IndentWidth(n)
		prevNewline = false
	}
}

// IndentWidth is the width of a form for the purposes of indenting the next line.
// For 'simple' forms (symbols, keywords, ...) the width includes one extra
// at the end for the following space.
func IndentWidth(node parse.Node) int {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			return 5
		}
		return 6
	case *parse.CharacterNode:
		return 2 // Not going to worry about multiwidth chars
	case *parse.CommentNode:
		return 0
	case *parse.DerefNode:
		return 1 + IndentWidth(node.Node)
	case *parse.KeywordNode:
		return len(node.Val) + 1
	case *parse.ListNode:
		return 2
	case *parse.MapNode:
		return 2
	case *parse.MetadataNode:
		return 1 + IndentWidth(node.Node)
	case *parse.NewlineNode:
		return 0
	case *parse.NilNode:
		return 4
	case *parse.NumberNode:
		return len(node.Val) + 1
	case *parse.SymbolNode:
		return len(node.Val) + 1
	case *parse.QuoteNode:
		return 1 + IndentWidth(node.Node)
	case *parse.StringNode:
		return 3 + len(node.Val)
	case *parse.SyntaxQuoteNode:
		return 1 + IndentWidth(node.Node)
	case *parse.UnquoteNode:
		return 1 + IndentWidth(node.Node)
	case *parse.UnquoteSpliceNode:
		return 1 + IndentWidth(node.Node)
	case *parse.VectorNode:
		return 2
	case *parse.FnLiteralNode:
		return 2
	case *parse.IgnoreFormNode:
		return 2 + IndentWidth(node.Node)
	case *parse.RegexNode:
		return 4 + len(node.Val)
	case *parse.SetNode:
		return 2
	case *parse.VarQuoteNode:
		return 1 + len(node.Val)
	case *parse.TagNode:
		return 1 + len(node.Val)
	}
	panic("unreached")
}

var indentSpecial = regexp.MustCompile(
	`^(def.*|if.*|let.*|send.*|when.*|with.*)$`,
)

func ListIndentWidth(node parse.Node) int {
	if node, ok := node.(*parse.SymbolNode); ok {
		switch node.Val {
		case "binding", "catch", "doseq", "doto", "fn", "for", "loop", "ns", "update":
			return 1
		}
		if indentSpecial.MatchString(node.Val) {
			return 1
		}
	}
	return IndentWidth(node)
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

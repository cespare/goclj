package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cespare/diff"
	"github.com/cespare/goclj/parse"
)

// TODO: Finish unhandled node types
// TODO: indent special fns/forms (e.g. ns, defn, ...) only two
// TODO: Indent paired vec forms (like let) correctly
// TODO: Split up lines of doc comments and indent properly

func PrintTree(w io.Writer, t *parse.Tree) (err error) {
	bw := &bufWriter{bufio.NewWriter(w)}
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
	PrintSequence(bw, t.Roots, 0, false)
	return bw.bw.Flush()
}

func PrintNode(w *bufWriter, node parse.Node, indent int) {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
	case *parse.CharacterNode:
		w.WriteString(node.Text)
	case *parse.CommentNode:
		w.WriteString(node.Text)
	case *parse.DerefNode:
		w.WriteByte('@')
		PrintNode(w, node.Node, indent+1)
	case *parse.FnLiteralNode:
	case *parse.IgnoreFormNode:
	case *parse.KeywordNode:
		w.WriteString(node.Val)
	case *parse.ListNode:
		w.WriteString("(")
		PrintSequence(w, node.Nodes, indent+1, true)
		w.WriteString(")")
	case *parse.MapNode:
		w.WriteString("{")
		PrintSequence(w, node.Nodes, indent+1, false)
		w.WriteString("}")
	case *parse.MetadataNode:
	case *parse.NewlineNode:
		panic("should not happen")
	case *parse.NilNode:
	case *parse.NumberNode:
		w.WriteString(node.Val)
	case *parse.QuoteNode:
	case *parse.RegexNode:
	case *parse.SetNode:
	case *parse.StringNode:
		w.WriteString(`"` + node.Val + `"`)
	case *parse.SymbolNode:
		w.WriteString(node.Val)
	case *parse.SyntaxQuoteNode:
	case *parse.TagNode:
	case *parse.UnquoteNode:
	case *parse.UnquoteSpliceNode:
	case *parse.VarQuoteNode:
	case *parse.VectorNode:
		w.WriteString("[")
		PrintSequence(w, node.Nodes, indent+1, false)
		w.WriteString("]")
	default:
		FmtErrf("%s: unhandled node type %T", node.Position(), node)
	}
}

// -1 indent => don't indent
func PrintSequence(w *bufWriter, nodes []parse.Node, indent int, listIndent bool) {
	newline := false
	for i, n := range nodes {
		if _, ok := n.(*parse.NewlineNode); ok {
			w.WriteByte('\n')
			newline = true
			continue
		}
		if listIndent && i == 1 {
			indent += PrintWidth(nodes[0]) + 1
		}
		if newline {
			w.WriteString(strings.Repeat(indentChar, indent))
			newline = false
		} else if i > 0 {
			w.WriteByte(' ')
		}
		PrintNode(w, n, indent)
	}
}

func PrintWidth(node parse.Node) int {
	switch node := node.(type) {
	case *parse.BoolNode:
		if node.Val {
			return 4
		}
		return 5
	case *parse.CharacterNode:
		return 1 // Not going to worry about multiwidth chars
	case *parse.CommentNode:
		return 0
	case *parse.DerefNode:
		return 1 + PrintWidth(node.Node)
	case *parse.KeywordNode:
		return len(node.Val)
	case *parse.ListNode:
		return 2
	case *parse.MapNode:
		return 2
	case *parse.MetadataNode:
		return 1 + PrintWidth(node.Node)
	case *parse.NewlineNode:
		return 0
	case *parse.NilNode:
		return 3
	case *parse.NumberNode:
		return len(node.Val)
	case *parse.SymbolNode:
		return len(node.Val)
	case *parse.QuoteNode:
		return 1 + PrintWidth(node.Node)
	case *parse.StringNode:
		return 2 + len(node.Val)
	case *parse.SyntaxQuoteNode:
		return 1 + PrintWidth(node.Node)
	case *parse.UnquoteNode:
		return 1 + PrintWidth(node.Node)
	case *parse.UnquoteSpliceNode:
		return 1 + PrintWidth(node.Node)
	case *parse.VectorNode:
		return 2
	case *parse.FnLiteralNode:
		return 2
	case *parse.IgnoreFormNode:
		return 2 + PrintWidth(node.Node)
	case *parse.RegexNode:
		return 3 + len(node.Val)
	case *parse.SetNode:
		return 2
	case *parse.VarQuoteNode:
		return 1 + len(node.Val)
	case *parse.TagNode:
		return 1 + len(node.Val)
	}
	panic("unreached")
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

var (
	indentChar string
)

func main() {
	flag.StringVar(&indentChar, "indent-char", " ", "character to use for indenting")
	listDifferent := flag.Bool("l", false, "print files whose formatting differs from cljfmt's")
	writeFile := flag.Bool("w", false, "write result to (source) file instead of stdout")
	flag.Parse()
	if len(indentChar) != 1 {
		fatalf("-indent-char arg must have length 1")
	}
	if flag.NArg() < 1 {
		usage()
	}
	if *listDifferent || *writeFile {
		for _, filename := range flag.Args() {
			if err := writeFormatted(filename, *listDifferent, *writeFile); err != nil {
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
	if err := PrintTree(os.Stdout, t); err != nil {
		fatal(err)
	}
}

func writeFormatted(filename string, listDifferent, writeFile bool) error {
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
	if err := PrintTree(tw, t); err != nil {
		return err
	}
	tw.Close()
	identical, err := diff.Files(filename, tw.Name())
	if err != nil {
		return err
	}
	if identical {
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

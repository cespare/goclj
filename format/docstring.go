package format

import (
	"strings"

	"github.com/cespare/goclj"
	"github.com/cespare/goclj/parse"
)

func (p *Printer) markDocstrings(n parse.Node) {
	if !goclj.FnFormSymbol(n, "ns", "defmulti", "def", "defmacro", "defn") {
		return
	}
	nodes := n.Children()
	if len(nodes) < 3 {
		return
	}
	if _, ok := nodes[1].(*parse.SymbolNode); !ok {
		return
	}
	for _, node := range nodes[2:] {
		if s, ok := node.(*parse.StringNode); ok {
			p.docstrings[s] = struct{}{}
			return
		}
		if !goclj.Newline(node) {
			return
		}
	}
}

func (p *Printer) alignDocstring(docstring string, w int) string {
	var (
		lines   = strings.Split(docstring, "\n")
		aligned = []string{lines[0]}
		indent  = strings.Repeat(string(p.IndentChar), w)
	)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line != "" {
			line = indent + line
		}
		aligned = append(aligned, line)
	}
	return strings.Join(aligned, "\n")
}

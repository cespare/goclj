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
	if !goclj.Symbol(nodes[1]) {
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
		prefix := indent
		n := strings.IndexFunc(line, func(r rune) bool { return r != ' ' })
		if n > w {
			prefix += strings.Repeat(" ", n-w)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			line = prefix + line
		}
		aligned = append(aligned, line)
	}
	return strings.Join(aligned, "\n")
}

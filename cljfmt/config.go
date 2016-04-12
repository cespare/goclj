package main

import (
	"fmt"
	"io"

	"github.com/cespare/goclj/parse"
)

type unexpectedNodeError struct {
	parse.Node
}

func (e unexpectedNodeError) Error() string {
	return fmt.Sprintf("found unexpected node (%T) at %s",
		e.Node, e.Node.Position())
}

func parseDotConfig(r io.Reader, name string) (indentSpecial []string, err error) {
	// We don't ask the parser for non-semantic nodes, so we don't need to
	// prune out comments.
	tree, err := parse.Reader(r, name, 0)
	if err != nil {
		return nil, err
	}
	if len(tree.Roots) == 0 {
		// I guess this is fine.
		return nil, nil
	}
	if len(tree.Roots) > 1 {
		return nil, unexpectedNodeError{tree.Roots[1]}
	}
	m, ok := tree.Roots[0].(*parse.MapNode)
	if !ok {
		return nil, unexpectedNodeError{tree.Roots[0]}
	}
	if len(m.Nodes)%2 != 0 {
		return nil, fmt.Errorf("map value at %s has odd number of children", m.Position())
	}
	for i := 0; i < len(m.Nodes); i += 2 {
		k := m.Nodes[i]
		sym, ok := k.(*parse.KeywordNode)
		if !ok || sym.Val != ":indent-special" {
			continue
		}
		indentSpecial, err = parseStringVector(m.Nodes[i+1])
		if err != nil {
			return nil, err
		}
	}
	return indentSpecial, nil
}

func parseStringVector(node parse.Node) ([]string, error) {
	vec, ok := node.(*parse.VectorNode)
	if !ok {
		return nil, unexpectedNodeError{node}
	}
	ss := make([]string, len(vec.Nodes))
	for i, v := range vec.Nodes {
		s, ok := v.(*parse.StringNode)
		if !ok {
			return nil, unexpectedNodeError{v}
		}
		ss[i] = s.Val
	}
	return ss, nil
}

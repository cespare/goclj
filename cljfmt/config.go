package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/cespare/goclj/format"
	"github.com/cespare/goclj/parse"
)

type unexpectedNodeError struct {
	parse.Node
}

func (e unexpectedNodeError) Error() string {
	return fmt.Sprintf("found unexpected node (%T) at %s",
		e.Node, e.Node.Position())
}

func parseDotConfig(r io.Reader, name string) (map[string]format.IndentStyle, error) {
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
		if !ok || sym.Val != ":indent-overrides" {
			continue
		}
		seq, err := sequence(m.Nodes[i+1])
		if err != nil {
			return nil, err
		}
		return parseIndentOverrides(seq)
	}
	return nil, nil
}

func parseIndentOverrides(nodes []parse.Node) (map[string]format.IndentStyle, error) {
	if len(nodes)%2 != 0 {
		return nil, errors.New(":indent-overrides value has odd number of children")
	}
	overrides := make(map[string]format.IndentStyle)
	for i := 0; i < len(nodes); i += 2 {
		var names []string
		seq, err := sequence(nodes[i])
		if err == nil {
			for _, n := range seq {
				s, err := stringNode(n)
				if err != nil {
					return nil, err
				}
				names = append(names, s)
			}
		} else {
			s, err := stringNode(nodes[i])
			if err != nil {
				return nil, err
			}
			names = []string{s}
		}
		kw, ok := nodes[i+1].(*parse.KeywordNode)
		if !ok {
			return nil, unexpectedNodeError{nodes[i+1]}
		}
		style, ok := indentStyles[kw.Val]
		if !ok {
			return nil, fmt.Errorf("unknown indent style %q", kw.Val)
		}
		for _, name := range names {
			overrides[name] = style
		}
	}
	return overrides, nil
}

func sequence(node parse.Node) ([]parse.Node, error) {
	switch node.(type) {
	case *parse.ListNode, *parse.VectorNode:
		return node.Children(), nil
	}
	return nil, unexpectedNodeError{node}
}

func stringNode(node parse.Node) (string, error) {
	sn, ok := node.(*parse.StringNode)
	if !ok {
		return "", unexpectedNodeError{node}
	}
	return sn.Val, nil
}

var indentStyles = map[string]format.IndentStyle{
	":normal":    format.IndentNormal,
	":list":      format.IndentList,
	":list-body": format.IndentListBody,
	":let":       format.IndentLet,
	":letfn":     format.IndentLetfn,
	":deftype":   format.IndentDeftype,
	":cond0":     format.IndentCond0,
	":cond1":     format.IndentCond1,
	":cond2":     format.IndentCond2,
}

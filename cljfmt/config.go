package main

import (
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

func (c *config) parseDotConfig(r io.Reader, name string) error {
	// We don't ask the parser for non-semantic nodes, so we don't need to
	// prune out comments.
	tree, err := parse.Reader(r, name, 0)
	if err != nil {
		return err
	}
	if len(tree.Roots) == 0 {
		// I guess this is fine.
		return nil
	}
	if len(tree.Roots) > 1 {
		return unexpectedNodeError{tree.Roots[1]}
	}
	m, ok := tree.Roots[0].(*parse.MapNode)
	if !ok {
		return unexpectedNodeError{tree.Roots[0]}
	}
	if len(m.Nodes)%2 != 0 {
		return fmt.Errorf("map value at %s has odd number of children", m.Position())
	}
	for i := 0; i < len(m.Nodes); i += 2 {
		k := m.Nodes[i]
		sym, ok := k.(*parse.KeywordNode)
		if !ok {
			continue
		}
		switch sym.Val {
		case ":extensions":
			c.extensions = make(map[string]struct{})
			seq, err := sequence(m.Nodes[i+1])
			if err != nil {
				return err
			}
			for _, n := range seq {
				ext, err := stringNode(n)
				if err != nil {
					return err
				}
				c.extensions[ext] = struct{}{}
			}
		case ":indent-overrides", ":thread-first-overrides":
			seq, err := sequence(m.Nodes[i+1])
			if err != nil {
				return err
			}
			overrides, err := parseOverrides(seq, sym.Val)
			if err != nil {
				return err
			}
			switch sym.Val {
			case ":indent-overrides":
				c.indentOverrides = make(map[string]format.IndentStyle)
				for k, v := range overrides {
					style, ok := indentStyles[v]
					if !ok {
						return fmt.Errorf("unknown indent style %q", v)
					}
					c.indentOverrides[k] = style
				}
			case ":thread-first-overrides":
				c.threadFirstOverrides = make(map[string]format.ThreadFirstStyle)
				for k, v := range overrides {
					style, ok := threadFirstStyles[v]
					if !ok {
						return fmt.Errorf("unknown thread-first style %q", v)
					}
					c.threadFirstOverrides[k] = style
				}
			}
		default:
			return fmt.Errorf("unknown configuration key %q", sym.Val)
		}
	}
	return nil
}

func parseOverrides(nodes []parse.Node, name string) (map[string]string, error) {
	if len(nodes)%2 != 0 {
		return nil, fmt.Errorf("%s value has odd number of children", name)
	}
	overrides := make(map[string]string)
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
		for _, s := range names {
			overrides[s] = kw.Val
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
	":for":       format.IndentFor,
	":deftype":   format.IndentDeftype,
	":cond0":     format.IndentCond0,
	":cond1":     format.IndentCond1,
	":cond2":     format.IndentCond2,
	":cond4":     format.IndentCond4,
}

var threadFirstStyles = map[string]format.ThreadFirstStyle{
	":normal": format.ThreadFirstNormal,
	":cond->": format.ThreadFirstCondArrow,
}

package parse

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var testCases = []struct {
	s    string
	want string
}{
	// all types
	{"true", "true"},
	{`\s`, "char('s')"},
	{"; comment!", `comment("; comment!")`},
	{"@foo", "deref"},
	{"#(+ % 3)", "lambda(length=3)"},
	{"#_(a b c)", "discard"},
	{":foobar", "keyword(:foobar)"},
	{"(foo bar baz)", "list(length=3)"},
	{"{:a b :c d}", "map(length=2)"},
	{"#:foo{:a 1}", "map(ns=:foo, length=1)"},
	{"#::{:b 1234}", "map(ns=::, length=1)"},
	{`^String`, "metadata"},
	{"nil", "nil"},
	{"123.456", "num(123.456)"},
	{"foo", "sym(foo)"},
	{"'(foobar)", "quote"},
	{`#"^asdf"`, `regex("^asdf")`},
	{"#{1 2 3}", "set(length=3)"},
	{"#?(:clj 1)", "reader-cond(length=2)"},
	{"#?@(:clj :a :default :b)", "reader-cond-splice(length=4)"},
	{`"foo"`, `string("foo")`},
	{"`(1 2 3)", "syntax quote"},
	{"#foo", "tag(foo)"},
	{"~foo", "unquote"},
	{"~@foo", "unquote splice"},
	{"#'asdf", "varquote(asdf)"},
	{"[a b c]", "vector(length=3)"},

	// issue 13
	{"#_foobar", "discard"},

	// issue 32
	{"#=foo", "eval"},
	{"#^foo", "metadata"},
	{"#! hello!", `comment("#! hello!")`},

	// issue 35
	{"a%b%", "sym(a%b%)"},
	{":100%>50%", "keyword(:100%>50%)"},
}

func TestAll(t *testing.T) {
	for _, tc := range testCases {
		tree, err := Reader(strings.NewReader(tc.s), "temp", IncludeNonSemantic)
		if err != nil {
			t.Fatalf("error parsing %q: %s", tc.s, err)
		}
		if len(tree.Roots) != 1 {
			t.Errorf("got %d roots for %q; want 1", len(tree.Roots), tc.s)
			continue
		}
		got := tree.Roots[0].String()
		if got != tc.want {
			t.Errorf("for %q: got %s; want %s", tc.s, got, tc.want)
			continue
		}
	}
}

func TestParentPointers(t *testing.T) {
	for _, tc := range []struct {
		s     string
		child string
		want  string
	}{
		{"a", "sym(a)", "<nil>"},
		{"(a b)", "sym(b)", "list(length=2)"},
		{"(a {b c})", "sym(b)", "map(length=1)"},
		{"'a", "sym(a)", "quote"},
	} {
		tree, err := Reader(strings.NewReader(tc.s), "temp", IncludeNonSemantic)
		if err != nil {
			t.Fatalf("error parsing %q: %s", tc.s, err)
		}
		child := walkFindNode(tree.Roots, tc.child)
		if child == nil {
			t.Errorf("for %q: child %s not found", tc.s, tc.child)
			continue
		}
		if got := fmt.Sprint(child.Parent()); got != tc.want {
			t.Errorf("for %q: got parent %s, want %s", tc.s, got, tc.want)
			continue
		}
	}
}

func walkFindNode(nodes []Node, target string) Node {
	for _, n := range nodes {
		if n.String() == target {
			return n
		}
		if m := walkFindNode(n.Children(), target); m != nil {
			return m
		}
	}
	return nil
}

// Issue 32.
func TestUnreadable(t *testing.T) {
	_, err := Reader(strings.NewReader("#<X Y Z>"), "temp", IncludeNonSemantic)
	if err == nil {
		t.Fatal("got nil error on unreadable dispatch macro")
	}
	if !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("for unreadable dispatch macro, got wrong error %s", err)
	}
}

// Issue 33.
func TestCommentCarriageReturn(t *testing.T) {
	const input = "3;a\r4"
	tree, err := Reader(strings.NewReader(input), "temp", IncludeNonSemantic)
	if err != nil {
		t.Fatalf("error parsing %q: %s", input, err)
	}
	got := tree.flatStrings()
	want := []string{"num(3)", `comment(";a")`, "num(4)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("for %q, got %v; want %v", input, got, want)
	}
}

// Issue 37.
func TestInternalNewlines(t *testing.T) {
	const input = "[3\n4]"
	tree, err := Reader(strings.NewReader(input), "temp", IncludeNonSemantic)
	if err != nil {
		t.Fatalf("error parsing %q: %s", input, err)
	}
	got := tree.flatStrings()
	want := []string{"vector(length=2)", "num(3)", "newline", "num(4)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("for %q, got %v; want %v", input, got, want)
	}
}

// Issue 48.
func TestUnterminatedQuotes(t *testing.T) {
	for _, input := range []string{"@", "'", "`", "~", "~@"} {
		_, err := Reader(strings.NewReader(input), "temp", IncludeNonSemantic)
		if !strings.HasSuffix(err.Error(), "unexpected EOF") {
			t.Errorf("for %q, got err=%v; want unexpected EOF", input, err)
		}
	}

	const input = "';hello\na"
	tree, err := Reader(strings.NewReader(input), "temp", IncludeNonSemantic)
	if err != nil {
		t.Fatalf("error parsing %q: %s", input, err)
	}
	got := tree.flatStrings()
	want := []string{"quote", "sym(a)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("for %q: got %v; want %v", input, got, want)
	}
}

// flatStrings gives a flattened string representation of t by calling String on
// each node in the tree in a depth-first traversal.
func (t *Tree) flatStrings() []string {
	var nodes []string
	var visit func(n Node)
	visit = func(n Node) {
		nodes = append(nodes, n.String())
		for _, child := range n.Children() {
			visit(child)
		}
	}
	for _, root := range t.Roots {
		visit(root)
	}
	return nodes
}

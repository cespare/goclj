package parse

import (
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
	{`^String`, "metadata"},
	{"nil", "nil"},
	{"123.456", "num(123.456)"},
	{"'(foobar)", "quote"},
	{`#"^asdf"`, `regex("^asdf")`},
	{"#{1 2 3}", "set(length=3)"},
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
}

func TestAll(t *testing.T) {
	for _, tc := range testCases {
		tree, err := Reader(strings.NewReader(tc.s), "temp", IncludeNonSemantic)
		if err != nil {
			t.Fatalf("error parsing %q: %s", tc.s, err)
		}
		if len(tree.Roots) != 1 {
			t.Fatalf("got %d roots for %q; want 1", len(tree.Roots), tc.s)
		}
		got := tree.Roots[0].String()
		if got != tc.want {
			t.Fatalf("for %q: got %s; want %s", tc.s, got, tc.want)
		}
	}
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
	got := make([]string, len(tree.Roots))
	for i, node := range tree.Roots {
		got[i] = node.String()
	}
	want := []string{"num(3)", `comment(";a")`, "num(4)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("for %q, got %v; want %v", input, got, want)
	}
}

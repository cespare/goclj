package parse

import (
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
			t.Fatalf("Error parsing %q: %s", tc.s, err)
		}
		if len(tree.Roots) != 1 {
			t.Fatalf("Got %d roots for %q; expected 1", len(tree.Roots), tc.s)
		}
		got := tree.Roots[0].String()
		if got != tc.want {
			t.Fatalf("For %q: got %s; want %s", tc.s, got, tc.want)
		}
	}
}

// See issue 32.
func TestUnreadable(t *testing.T) {
	_, err := Reader(strings.NewReader("#<X Y Z>"), "temp", IncludeNonSemantic)
	if err == nil {
		t.Fatal("got nil error on unreadable dispatch macro")
	}
	if !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("for unreadable dispatch macro, got wrong error %s", err)
	}
}

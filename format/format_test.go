package format

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cespare/goclj/parse"
)

func TestFixture(t *testing.T) {
	for _, fixture := range []string{
		"simple1",
		"let",
		"map",
		"nsmap",
		"deftype",
		"listbody",
		"threadfirst",
		"cond",
		"indent",
		"readercond",
		"issue5",
		"issue8",
		"issue9",
		"issue14",
		"issue15",
		"issue16",
		"issue17",
		"issue18",
		"issue19",
		"issue21",
		"issue23",
		"issue49",
		"issue55",
	} {
		t.Run(fixture, func(t *testing.T) {
			testFixture(t, fixture+".clj")
		})
	}
}

func TestChange(t *testing.T) {
	for _, fixture := range []string{
		"styleguide",
		"newline",
		"require",
		"issue6",
		"issue7",
		"issue25",
		"issue26",
		"issue32",
		"issue37",
	} {
		t.Run(fixture, func(t *testing.T) {
			testChange(t, fixture+"_before.clj", fixture+"_after.clj")
		})
	}
}

func TestTransformsUseToRequire(t *testing.T) {
	testChangeTransforms(
		t,
		"transform/use2require_before.clj",
		"transform/use2require_after.clj",
		map[Transform]bool{TransformUseToRequire: true},
	)
}

func TestTransformsRemoveUnusedRequires(t *testing.T) {
	testChangeTransforms(
		t,
		"transform/unusedrequires_before.clj",
		"transform/unusedrequires_after.clj",
		map[Transform]bool{
			TransformUseToRequire:         true,
			TransformRemoveUnusedRequires: true,
		},
	)
}

func TestTransformsRemoveUnusedRequiresEmpty(t *testing.T) {
	testChangeTransforms(
		t,
		"transform/unusedrequiresempty_before.clj",
		"transform/unusedrequiresempty_after.clj",
		map[Transform]bool{TransformRemoveUnusedRequires: true},
	)
}

func TestCustomIndent(t *testing.T) {
	const file0 = "indent1.clj"
	const file1 = "indent1_custom.clj"
	testFixture(t, file0)

	f := func(p *Printer) {
		p.IndentOverrides = map[string]IndentStyle{
			"delete": IndentListBody,
			"up":     IndentListBody,
		}
	}
	testChangeCustom(t, file0, file1, f)
}

func TestCustomTransforms(t *testing.T) {
	const before = "transforms_before.clj"
	const after = "transforms_after.clj"
	testChangeTransforms(t, before, after, map[Transform]bool{
		TransformSortImportRequire:     false,
		TransformFixDefnArglistNewline: false,
	})
}

func TestIssue41(t *testing.T) {
	const file = "issue41.clj"
	f := func(p *Printer) {
		p.IndentOverrides = map[string]IndentStyle{
			"cond-blah-blah-blah": IndentCond0,
		}
	}
	testChangeCustom(t, file, file, f)
}

func testFixture(t *testing.T, filename string) {
	testChange(t, filename, filename)
}

func testChange(t *testing.T, before, after string) {
	testChangeCustom(t, before, after, func(p *Printer) {})
}

func testChangeTransforms(t *testing.T, before, after string, transforms map[Transform]bool) {
	t.Helper()
	f := func(p *Printer) { p.Transforms = transforms }
	testChangeCustom(t, before, after, f)
}

func testChangeCustom(t *testing.T, before, after string, f func(p *Printer)) {
	t.Helper()
	tree := parseFile(t, before)
	var buf bytes.Buffer
	p := NewPrinter(&buf)
	f(p)
	if err := p.PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	want := readFile(t, after)
	check(t, before, buf.String(), string(want))
}

func parseFile(t *testing.T, name string) *parse.Tree {
	tree, err := parse.File(filepath.Join("testdata", name), parse.IncludeNonSemantic)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func readFile(t *testing.T, name string) []byte {
	b, err := ioutil.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func check(t *testing.T, desc, got, want string) {
	t.Helper()
	if got != want {
		gotFormatted := formatLines(got)
		wantFormatted := formatLines(want)
		t.Errorf("formatted %s incorrectly: got\n%swant\n%s",
			desc, gotFormatted, wantFormatted)
	}
}

func formatLines(contents string) string {
	lines := strings.Split(contents, "\n")
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	for i, line := range lines {
		numWidth := int(math.Log10(float64(len(lines)))) + 1
		fmt.Fprintf(&b, "  %*d ", numWidth, i+1)
		prefix := true
		for _, c := range line {
			if prefix && c == ' ' {
				b.WriteRune('·')
			} else {
				prefix = false
				b.WriteRune(c)
			}
		}
		b.WriteRune('\n')
	}
	return b.String()
}

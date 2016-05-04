package format

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"

	"github.com/cespare/goclj/parse"
)

// TODO: Rewrite using subtests after Go 1.7.

func TestSimpleFile(t *testing.T)  { testFixture(t, "simple1.clj") }
func TestLet(t *testing.T)         { testFixture(t, "let.clj") }
func TestDeftype(t *testing.T)     { testFixture(t, "deftype.clj") }
func TestListbody(t *testing.T)    { testFixture(t, "listbody.clj") }
func TestThreadFirst(t *testing.T) { testFixture(t, "threadfirst.clj") }
func TestCond(t *testing.T)        { testFixture(t, "cond.clj") }

func TestStyleGuide(t *testing.T) { testChange(t, "styleguide_bad.clj", "styleguide_good.clj") }
func TestNewline(t *testing.T)    { testChange(t, "newline_before.clj", "newline_after.clj") }
func TestRequire(t *testing.T)    { testChange(t, "require_before.clj", "require_after.clj") }

func TestIndent(t *testing.T)  { testFixture(t, "indent.clj") }
func TestIssue5(t *testing.T)  { testFixture(t, "issue5.clj") }
func TestIssue8(t *testing.T)  { testFixture(t, "issue8.clj") }
func TestIssue9(t *testing.T)  { testFixture(t, "issue9.clj") }
func TestIssue14(t *testing.T) { testFixture(t, "issue14.clj") }
func TestIssue15(t *testing.T) { testFixture(t, "issue15.clj") }
func TestIssue16(t *testing.T) { testFixture(t, "issue16.clj") }
func TestIssue17(t *testing.T) { testFixture(t, "issue17.clj") }
func TestIssue18(t *testing.T) { testFixture(t, "issue18.clj") }
func TestIssue19(t *testing.T) { testFixture(t, "issue19.clj") }
func TestIssue21(t *testing.T) { testFixture(t, "issue21.clj") }
func TestIssue23(t *testing.T) { testFixture(t, "issue23.clj") }

func TestIssue6(t *testing.T)  { testChange(t, "issue6_before.clj", "issue6_after.clj") }
func TestIssue7(t *testing.T)  { testChange(t, "issue7_before.clj", "issue7_after.clj") }
func TestIssue25(t *testing.T) { testChange(t, "issue25_before.clj", "issue25_after.clj") }
func TestIssue26(t *testing.T) { testChange(t, "issue26_before.clj", "issue26_after.clj") }
func TestIssue32(t *testing.T) { testChange(t, "issue32_before.clj", "issue32_after.clj") }

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
	f := func(p *Printer) { p.Transforms = transforms }
	testChangeCustom(t, before, after, f)
}

func testChangeCustom(t *testing.T, before, after string, f func(p *Printer)) {
	tree := parseFile(t, before)
	var buf bytes.Buffer
	p := NewPrinter(&buf)
	f(p)
	if err := p.PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	want := readFile(t, after)
	check(t, before, buf.Bytes(), want)
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

func check(t *testing.T, desc string, got, want []byte) {
	if !bytes.Equal(got, want) {
		gotFormatted := formatLines(got)
		wantFormatted := formatLines(want)
		t.Errorf("formatted %s incorrectly: got\n%swant\n%s",
			desc, gotFormatted, wantFormatted)
	}
}

func formatLines(contents []byte) []byte {
	lines := bytes.Split(contents, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	var result []byte
	for i, line := range lines {
		numWidth := int(math.Log10(float64(len(lines)))) + 1
		formatted := []byte(fmt.Sprintf("  %*d ", numWidth, i+1))
		prefix := true
		for _, c := range line {
			if prefix && c == ' ' {
				formatted = append(formatted, '.')
			} else {
				prefix = false
				formatted = append(formatted, c)
			}
		}
		formatted = append(formatted, '\n')
		result = append(result, formatted...)
	}
	return result
}

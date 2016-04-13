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

func TestSimpleFile(t *testing.T) { testFixture(t, "simple1.clj") }
func TestLet(t *testing.T)        { testFixture(t, "let.clj") }
func TestDeftype(t *testing.T)    { testFixture(t, "deftype.clj") }
func TestListbody(t *testing.T)   { testFixture(t, "listbody.clj") }

func TestStyleGuide(t *testing.T) { testTransform(t, "styleguide_bad.clj", "styleguide_good.clj") }
func TestNewline(t *testing.T)    { testTransform(t, "newline_before.clj", "newline_after.clj") }
func TestRequire(t *testing.T)    { testTransform(t, "require_before.clj", "require_after.clj") }

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

func TestIssue6(t *testing.T)  { testTransform(t, "issue6_before.clj", "issue6_after.clj") }
func TestIssue7(t *testing.T)  { testTransform(t, "issue7_before.clj", "issue7_after.clj") }
func TestIssue25(t *testing.T) { testTransform(t, "issue25_before.clj", "issue25_after.clj") }
func TestIssue26(t *testing.T) { testTransform(t, "issue26_before.clj", "issue26_after.clj") }
func TestIssue32(t *testing.T) { testTransform(t, "issue32_before.clj", "issue32_after.clj") }

func TestCustomIndent(t *testing.T) {
	const file0 = "indent1.clj"
	const file1 = "indent1_custom.clj"
	tree := parseFile(t, file0)
	var buf bytes.Buffer
	if err := NewPrinter(&buf).PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	want := readFile(t, file0)
	check(t, file0, buf.Bytes(), want)

	buf.Reset()
	p := NewPrinter(&buf)
	p.IndentOverrides = map[string]IndentStyle{
		"delete": IndentListBody,
		"up":     IndentListBody,
	}
	if err := p.PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	want = readFile(t, file1)
	check(t, file1, buf.Bytes(), want)
}

func testFixture(t *testing.T, filename string) {
	testTransform(t, filename, filename)
}

func testTransform(t *testing.T, before, after string) {
	tree := parseFile(t, before)
	var buf bytes.Buffer
	if err := NewPrinter(&buf).PrintTree(tree); err != nil {
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

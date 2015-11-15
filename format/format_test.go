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

func TestSimpleFile(t *testing.T) { testFixture(t, "simple1.clj") }

func TestStyleGuide(t *testing.T) {
	testTransform(t, "styleguide_bad.clj", "styleguide_good.clj")
}

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
func TestIssue23(t *testing.T) { testFixture(t, "issue23.clj") }

func TestIssue6(t *testing.T)  { testTransform(t, "issue6_before.clj", "issue6_after.clj") }
func TestIssue7(t *testing.T)  { testTransform(t, "issue7_before.clj", "issue7_after.clj") }
func TestIssue25(t *testing.T) { testTransform(t, "issue25_before.clj", "issue25_after.clj") }
func TestIssue26(t *testing.T) { testTransform(t, "issue26_before.clj", "issue26_after.clj") }

func testFixture(t *testing.T, filename string) {
	testTransform(t, filename, filename)
}

func testTransform(t *testing.T, before, after string) {
	sourceFile := filepath.Join("testdata", before)
	tree, err := parse.File(sourceFile, true)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := NewPrinter(&buf).PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	want, err := ioutil.ReadFile(filepath.Join("testdata", after))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		gotFormatted := formatLines(buf.Bytes())
		wantFormatted := formatLines(want)
		t.Fatalf("Formatted %s incorrectly: got\n%swant\n%s", sourceFile, gotFormatted, wantFormatted)
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

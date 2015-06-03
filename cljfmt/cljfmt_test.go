package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"testing"

	"github.com/cespare/goclj/parse"
)

func TestSimpleFile(t *testing.T) { testFixture(t, "testdata/simple1.clj") }
func TestIndent(t *testing.T)     { testFixture(t, "testdata/indent.clj") }
func TestIssue5(t *testing.T)     { testFixture(t, "testdata/issue5.clj") }
func TestIssue8(t *testing.T)     { testFixture(t, "testdata/issue8.clj") }
func TestIssue9(t *testing.T)     { testFixture(t, "testdata/issue9.clj") }
func TestIssue10(t *testing.T)    { testFixture(t, "testdata/issue10.clj") }
func TestIssue14(t *testing.T)    { testFixture(t, "testdata/issue14.clj") }
func TestIssue15(t *testing.T)    { testFixture(t, "testdata/issue15.clj") }
func TestIssue16(t *testing.T)    { testFixture(t, "testdata/issue16.clj") }
func TestIssue17(t *testing.T)    { testFixture(t, "testdata/issue17.clj") }
func TestIssue18(t *testing.T)    { testFixture(t, "testdata/issue18.clj") }

func testFixture(t *testing.T, filename string) {
	tree, err := parse.File(filename, true)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := NewPrinter(&buf).PrintTree(tree); err != nil {
		t.Fatal(err)
	}
	original, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), original) {
		gotFormatted := formatLines(buf.Bytes())
		wantFormatted := formatLines(original)
		t.Fatalf("Formatted %s incorrectly: got\n%soriginal was\n%s", filename, gotFormatted, wantFormatted)
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

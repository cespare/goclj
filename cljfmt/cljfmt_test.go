package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"

	"github.com/cespare/goclj/parse"
)

func TestFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/*.clj")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		testFixture(t, file)
	}
}

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

package format

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/cespare/goclj/parse"
)

func TestFixture(t *testing.T) {
	fixtures, _ := loadFixtures(t)
	if len(fixtures) < 5 {
		t.Fatal("failed to load fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			testFixture(t, fixture+".clj")
		})
	}
}

func TestChange(t *testing.T) {
	_, fixtures := loadFixtures(t)
	if len(fixtures) < 5 {
		t.Fatal("failed to load fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			testChange(t, fixture+"_before.clj", fixture+"_after.clj")
		})
	}
}

func TestTransformsEnforceNSStyle(t *testing.T) {
	testChangeTransforms(
		t,
		"transform/nsstyle_before.clj",
		"transform/nsstyle_after.clj",
		map[Transform]bool{TransformEnforceNSStyle: true},
	)
}

func TestTransformsUseToRequire(t *testing.T) {
	testChangeTransforms(
		t,
		"custom/use2require_before.clj",
		"custom/use2require_after.clj",
		map[Transform]bool{TransformUseToRequire: true},
	)
}

func TestTransformsRemoveUnusedRequires(t *testing.T) {
	testChangeTransforms(
		t,
		"custom/unusedrequires_before.clj",
		"custom/unusedrequires_after.clj",
		map[Transform]bool{
			TransformUseToRequire:         true,
			TransformRemoveUnusedRequires: true,
		},
	)
}

func TestTransformsRemoveUnusedRequiresEmpty(t *testing.T) {
	testChangeTransforms(
		t,
		"custom/unusedrequiresempty_before.clj",
		"custom/unusedrequiresempty_after.clj",
		map[Transform]bool{TransformRemoveUnusedRequires: true},
	)
}

func TestTransformsFixIfNewlineConsistency(t *testing.T) {
	testChangeTransforms(
		t,
		"custom/ifnewlines_before.clj",
		"custom/ifnewlines_after.clj",
		map[Transform]bool{TransformFixIfNewlineConsistency: true},
	)
}

func TestCustomIndent(t *testing.T) {
	const file0 = "custom/indent1.clj"
	const file1 = "custom/indent1_custom.clj"
	testFixture(t, file0)

	f := func(p *Printer) {
		p.IndentOverrides = map[string]IndentStyle{
			"delete":            IndentListBody,
			"up":                IndentListBody,
			"org.lib1/f":        IndentListBody,
			"org.lib3/mycond":   IndentCond0,
			"org.lib3/mymacro1": IndentLet,
			"org.lib4/mymacro2": IndentLet,
		}
	}
	testChangeCustom(t, file0, file1, f)
}

func TestCustomTransforms(t *testing.T) {
	testChangeTransforms(
		t,
		"custom/transforms_before.clj",
		"custom/transforms_after.clj",
		map[Transform]bool{
			TransformSortImportRequire:     false,
			TransformFixDefnArglistNewline: false,
		},
	)
}

func TestIssue41(t *testing.T) {
	const file = "custom/issue41.clj"
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

var (
	fixturesOnce   sync.Once
	fixturesErr    error
	singleFixtures []string
	changeFixtures []string
)

func loadFixtures(t *testing.T) (singles, changes []string) {
	t.Helper()
	fixturesOnce.Do(func() { fixturesErr = loadFixturesOnce() })
	if fixturesErr != nil {
		t.Fatalf("error loading fixtures: %s", fixturesErr)
	}
	return singleFixtures, changeFixtures
}

func loadFixturesOnce() error {
	paths, err := filepath.Glob("testdata/*.clj")
	if err != nil {
		return err
	}
	befores := make(map[string]struct{})
	afters := make(map[string]struct{})
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".clj")
		if n := strings.TrimSuffix(name, "_before"); n != name {
			befores[n] = struct{}{}
			continue
		}
		if n := strings.TrimSuffix(name, "_after"); n != name {
			afters[n] = struct{}{}
			continue
		}
		singleFixtures = append(singleFixtures, name)
	}
	for name := range befores {
		if _, ok := afters[name]; !ok {
			return fmt.Errorf("found %s_before.clj but no %[1]s_after.clj", name)
		}
		delete(afters, name)
		changeFixtures = append(changeFixtures, name)
	}
	for name := range afters {
		return fmt.Errorf("found %s_after.clj but no %[1]s_before.clj", name)
	}

	sort.Strings(singleFixtures)
	sort.Strings(changeFixtures)
	return nil
}

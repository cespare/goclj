package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cespare/goclj/format"
	"github.com/cespare/goclj/parse"
)

func usage() {
	fmt.Fprintf(os.Stderr, `usage: %s [flags] [paths...]
Any directories given will be recursively walked. If no paths are provided,
cljfmt reads from standard input.

Flags:
`, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, `
See the goclj README for more documentation of the available transforms.`)
}

type config struct {
	indentOverrides      map[string]format.IndentStyle
	threadFirstOverrides map[string]format.ThreadFirstStyle
	transforms           map[format.Transform]bool
	list                 bool
	write                bool
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("clfmt: ")
	var configFile pathFlag
	if home, ok := os.LookupEnv("HOME"); ok {
		configFile.p = filepath.Join(home, ".cljfmt")
	}
	conf := config{
		transforms: make(map[format.Transform]bool),
	}
	flag.Var(&configFile, "c", "path to config file")
	flag.BoolVar(&conf.list, "l", false,
		"print files whose formatting differs from cljfmt's")
	flag.BoolVar(&conf.write, "w", false,
		"write result to (source) file instead of stdout")
	flag.Var(transformFlag{conf.transforms, true}, "enable-transform",
		"turn on the named transform")
	flag.Var(transformFlag{conf.transforms, false}, "disable-transform",
		"turn off the named transform")
	flag.Usage = usage
	flag.Parse()

	conf.parseDotConfigFile(configFile)

	if flag.NArg() == 0 {
		if conf.write {
			log.Fatal("cannot use -w with standard input")
		}
		conf.list = false
		if err := conf.processFile("<stdin>", os.Stdin); err != nil {
			log.Fatal(err)
		}
		return
	}

	for _, path := range flag.Args() {
		stat, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}
		if stat.IsDir() {
			conf.walkDir(path)
			continue
		}
		if err := conf.processFile(path, nil); err != nil {
			log.Fatal(err)
		}
	}
}

type transformFlag struct {
	m map[format.Transform]bool
	b bool
}

func (tf transformFlag) Set(v string) error {
	var t format.Transform
	switch v {
	case "sort-import-require":
		t = format.TransformSortImportRequire
	case "enforce-ns-style":
		t = format.TransformEnforceNSStyle
	case "remove-trailing-newlines":
		t = format.TransformRemoveTrailingNewlines
	case "fix-defn-arglist-newline":
		t = format.TransformFixDefnArglistNewline
	case "fix-defmethod-dispatch-val-newline":
		t = format.TransformFixDefmethodDispatchValNewline
	case "remove-extra-blank-lines":
		t = format.TransformRemoveExtraBlankLines
	case "fix-if-newline-consistency":
		t = format.TransformFixIfNewlineConsistency
	case "use-to-require":
		t = format.TransformUseToRequire
	case "remove-unused-requires":
		t = format.TransformRemoveUnusedRequires
	default:
		return fmt.Errorf("unrecognized transform %q", v)
	}
	tf.m[t] = tf.b
	return nil
}

func (tf transformFlag) String() string {
	return "none"
}

type pathFlag struct {
	p   string
	set bool
}

func (pf *pathFlag) Set(v string) error {
	pf.p = v
	pf.set = true
	return nil
}

func (pf *pathFlag) String() string {
	return pf.p
}

func (c *config) parseDotConfigFile(pf pathFlag) {
	if pf.p == "" {
		return
	}
	f, err := os.Open(pf.p)
	if err != nil {
		if !os.IsNotExist(err) || pf.set {
			log.Println("warning: could not open config", err)
		}
		return
	}
	defer f.Close()
	if err := c.parseDotConfig(f, pf.p); err != nil {
		log.Fatalf("error parsing config %s: %s", pf.p, err)
	}
}

var (
	buf1 bytes.Buffer
	buf2 bytes.Buffer
)

// processFile formats the given file.
// If in == nil, the input is the file of the given name.
func (c *config) processFile(filename string, in io.Reader) error {
	var perm os.FileMode = 0644
	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			return err
		}
		perm = stat.Mode().Perm()
		in = f
	}

	buf1.Reset()
	buf2.Reset()

	if _, err := io.Copy(&buf1, in); err != nil {
		return err
	}
	r := bytes.NewReader(buf1.Bytes())
	t, err := parse.Reader(r, filename, parse.IncludeNonSemantic)
	if err != nil {
		return err
	}

	p := format.NewPrinter(&buf2)
	p.IndentChar = ' '
	p.IndentOverrides = c.indentOverrides
	p.ThreadFirstStyleOverrides = c.threadFirstOverrides
	p.Transforms = c.transforms
	if err := p.PrintTree(t); err != nil {
		return err
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		if c.list {
			fmt.Println(filename)
		}
		if c.write {
			if err := ioutil.WriteFile(filename, buf2.Bytes(), perm); err != nil {
				return err
			}
		}
	}
	if !c.list && !c.write {
		io.Copy(os.Stdout, &buf2)
	}
	return nil
}

func (c *config) walkDir(path string) {
	walk := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		for _, ext := range []string{".clj", ".cljs", ".cljc"} {
			if strings.HasSuffix(name, ext) {
				return c.processFile(path, nil)
			}
		}
		return nil // not a Clojure file
	}
	if err := filepath.Walk(path, walk); err != nil {
		log.Fatal(err)
	}
}

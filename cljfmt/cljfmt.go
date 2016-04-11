package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

Flags:`, os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var (
		list  = flag.Bool("l", false, "print files whose formatting differs from cljfmt's")
		write = flag.Bool("w", false, "write result to (source) file instead of stdout")
	)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		if *write {
			fatal("cannot use -w with standard input")
		}
		if err := processFile("<stdin>", os.Stdin, false, false); err != nil {
			fatal(err)
		}
		return
	}

	for _, path := range flag.Args() {
		stat, err := os.Stat(path)
		if err != nil {
			fatal(err)
		}
		if stat.IsDir() {
			walkDir(path, *list, *write)
			continue
		}
		if err := processFile(path, nil, *list, *write); err != nil {
			fatal(err)
		}
	}
}

var (
	buf1 bytes.Buffer
	buf2 bytes.Buffer
)

// processFile formats the given file.
// If in == nil, the input is the file of the given name.
func processFile(filename string, in io.Reader, list, write bool) error {
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
	t, err := parse.Reader(bytes.NewReader(buf1.Bytes()), filename, true)
	if err != nil {
		return err
	}

	p := format.NewPrinter(&buf2)
	p.IndentChar = ' '
	if err := p.PrintTree(t); err != nil {
		return err
	}
	if bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		return nil
	}
	if list {
		fmt.Println(filename)
	}
	if write {
		if err := ioutil.WriteFile(filename, buf2.Bytes(), perm); err != nil {
			return err
		}
	}
	if !list && !write {
		io.Copy(os.Stdout, &buf2)
	}
	return nil
}

func walkDir(path string, list, write bool) {
	walk := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		name := f.Name()
		if strings.HasPrefix(name, ".") ||
			!strings.HasSuffix(name, ".clj") {
			return nil
		}
		return processFile(path, nil, list, write)
	}
	if err := filepath.Walk(path, walk); err != nil {
		fatal(err)
	}
}

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

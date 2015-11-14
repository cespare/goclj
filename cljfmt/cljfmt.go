package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cespare/diff"
	"github.com/cespare/goclj/format"
	"github.com/cespare/goclj/parse"
)

func main() {
	var (
		indentCharFlag = flag.String("indentchar", " ", "character to use for indenting")
		listDifferent  = flag.Bool("l", false, "print files whose formatting differs from cljfmt's")
		writeFile      = flag.Bool("w", false, "write result to (source) file instead of stdout")
	)
	flag.Parse()
	if len(*indentCharFlag) != 1 {
		fatalf("-indentchar arg must have length 1")
	}
	indentChar := (*indentCharFlag)[0]
	if flag.NArg() < 1 {
		usage()
	}
	if *listDifferent || *writeFile {
		for _, filename := range flag.Args() {
			if err := writeFormatted(filename, indentChar, *listDifferent, *writeFile); err != nil {
				fatal(err)
			}
		}
		return
	}

	if flag.NArg() > 1 {
		fatalf("must provide a single file unless -l or -w are given")
	}

	t, err := parse.File(flag.Arg(0), true)
	if err != nil {
		fatal(err)
	}
	p := format.NewPrinter(os.Stdout)
	p.IndentChar = indentChar
	if err := p.PrintTree(t); err != nil {
		fatal(err)
	}
}

func writeFormatted(filename string, indentChar byte, listDifferent, writeFile bool) error {
	tw, err := ioutil.TempFile(filepath.Dir(filename), "cljfmt-")
	if err != nil {
		return err
	}
	defer os.Remove(tw.Name())
	defer tw.Close()

	t, err := parse.File(filename, true)
	if err != nil {
		return err
	}
	p := format.NewPrinter(tw)
	p.IndentChar = indentChar
	if err := p.PrintTree(t); err != nil {
		return err
	}
	tw.Close()
	different, err := diff.Files(filename, tw.Name())
	if err != nil {
		return err
	}
	if !different {
		return nil
	}
	if listDifferent {
		fmt.Println(filename)
	}
	if writeFile {
		if err := os.Rename(tw.Name(), filename); err != nil {
			return err
		}
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [flags] [path ...]\n", os.Args[0])
	os.Exit(1)
}

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

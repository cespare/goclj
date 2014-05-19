package parse

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
)

type Tree struct {
	Roots []Node

	// Parser state
	tok       token // single-item lookahead
	peekCount int
	lex       *lexer
	inLambda  bool
}

func (t *Tree) Parse() (err error) {
	defer t.recover(&err)
	for {
		node := t.parse()
		if node == nil {
			break
		}
		t.Roots = append(t.Roots, node)
	}
	return nil
}

func (t *Tree) Write(w io.Writer) error {
	_, err := w.Write([]byte("asdf"))
	return err
}

func (t *Tree) recover(err *error) {
	if e := recover(); e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if e2, ok := e.(error); ok {
			*err = e2
		} else {
			panic(e)
		}
	}
}

func (t *Tree) next() token {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.tok = t.lex.nextToken()
	}
	return t.tok
}

func (t *Tree) backup() token {
	t.peekCount++
	if t.peekCount > 1 {
		panic("backup() called twice consecutively")
	}
}

func (t *Tree) peek() token {
	if t.peekCount == 0 {
		t.peekCount++
		t.tok = t.lex.nextToken()
	}
	return t.tok
}

func (t *Tree) errorf(format string, args ...interface{}) {
	panic(t.lex.pos.FormatError("parse", fmt.Sprintf(format, args...)))
}

func ParseFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	t := &Tree{lex: lex(filename, bufio.NewReader(f))}
	if err := t.Parse(); err != nil {
		log.Fatal(err)
	}
	if err := t.Write(os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// parse parses the next top-level item from the token stream. It returns nil if there are no non-EOF tokens
// left in the stream.
func (t *Tree) parse() Node {
	tok := t.peek()
	switch tok.typ {

	}
}

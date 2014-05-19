package parse

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func LexFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	l := lex(filename, bufio.NewReader(f))
outer:
	for {
		tok := l.nextToken()
		if tok.typ == tokError {
			log.Fatal(tok.AsError())
		}
		fmt.Println(tok)
		if tok.typ == tokEOF {
			break outer
		}
	}
}

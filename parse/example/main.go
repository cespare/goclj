package main

import (
	"log"
	"os"

	"github.com/cespare/goclj/parse"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s FILENAME", os.Args[0])
	}
	parse.LexFile(os.Args[1])
}

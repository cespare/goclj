package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cespare/goclj/parse"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s FILENAME", os.Args[0])
	}
	t, err := parse.ParseFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(t)
}

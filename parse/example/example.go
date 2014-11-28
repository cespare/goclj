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
	t, err := parse.File(os.Args[1], true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(t)
}

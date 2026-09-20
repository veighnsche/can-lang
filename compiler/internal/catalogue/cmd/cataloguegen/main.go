package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
)

func main() {
	check := flag.Bool("check", false, "verify mirrors without writing")
	root := flag.String("root", ".", "checkout root")
	flag.Parse()
	if err := catalogue.Generate(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

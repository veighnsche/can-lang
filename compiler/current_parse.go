package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// runCurrentParse is the current-language grammar boundary. It deliberately
// never enters the predecessor checker/evaluator/emitter. I11 will build the
// current driver on these nodes after the checker and emitter tasks land.
func runCurrentParse(stdout, stderr io.Writer, args []string) int {
	flags := flag.NewFlagSet("parse", flag.ContinueOnError)
	flags.SetOutput(stderr)
	render := flags.Bool("render", false, "render the parsed syntax tree without comments")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: canlc parse [--render] FILE.can")
		return 2
	}
	path := flags.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	file, err := source.New(path, string(data))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	parsed := syntax.Parse(file)
	if !parsed.OK() {
		for _, diagnostic := range parsed.Diagnostics {
			fmt.Fprintln(stderr, diagnostic.Format(file))
		}
		return 1
	}
	if *render {
		_, err = io.WriteString(stdout, syntax.Format(parsed.File))
	} else {
		_, err = fmt.Fprintf(stdout, "parsed %s: package %s, %d declarations\n", path, parsed.File.Header.Name.Text, len(parsed.File.Declarations))
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/veighnsche/can-lang/distribution"
	"os"
)

func main() {
	archive := flag.String("archive", "", "local pinned Bun archive (required)")
	output := flag.String("out", "dist/development", "parent of new version root")
	source := flag.String("source", ".", "Can source checkout")
	version := flag.String("version", "dev", "development distribution version")
	flag.Parse()
	if *archive == "" {
		fmt.Fprintln(os.Stderr, "--archive is required; runtime downloads are never automatic")
		os.Exit(2)
	}
	path, err := distribution.Build(context.Background(), *source, *output, *archive, *version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

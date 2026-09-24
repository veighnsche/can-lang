package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, root, "/private/tmp/can-htmx-503-bundle", filepath.Join(root, ".local-deps/bun-darwin-aarch64.zip"), "htmx-503-probe")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundle)
}

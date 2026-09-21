package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/veighnsche/can-lang/distribution"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "install" {
		install(os.Args[2:])
		return
	}
	archive := flag.String("archive", "", "local pinned Bun archive (required)")
	output := flag.String("out", "dist/development", "parent of new version root")
	source := flag.String("source", ".", "Can source checkout")
	version := flag.String("version", "dev", "development distribution version")
	releaseOut := flag.String("release-out", "", "emit release artifacts here after building (optional)")
	flag.Parse()
	if *archive == "" {
		fmt.Fprintln(os.Stderr, "--archive is required; runtime downloads are never automatic")
		os.Exit(2)
	}
	ctx := context.Background()
	path, err := distribution.Build(ctx, *source, *output, *archive, *version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
	if *releaseOut != "" {
		artifacts, err := distribution.Release(ctx, path, *releaseOut)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(artifacts.Archive)
		fmt.Println(artifacts.SHA256)
		fmt.Println(artifacts.Inspection)
	}
}

func install(args []string) {
	flags := flag.NewFlagSet("install", flag.ExitOnError)
	archive := flags.String("archive", "", "release zip (required)")
	sha := flags.String("sha", "", "detached sha256 record (required)")
	root := flags.String("root", "", "install root (required)")
	update := flags.Bool("update", false, "preserve and report the previous selection")
	flags.Parse(args)
	if *archive == "" || *sha == "" || *root == "" {
		fmt.Fprintln(os.Stderr, "distbuild install --archive <zip> --sha <record> --root <dir> [--update]")
		os.Exit(2)
	}
	ctx := context.Background()
	if *update {
		previous, current, err := distribution.Update(ctx, *archive, *sha, *root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(previous)
		fmt.Println(current)
		return
	}
	path, err := distribution.Install(ctx, *archive, *sha, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

// Package main is the Can compiler launcher and manifest-backed tooling.
// Current emission/publication lives in internal/emit and internal/driver.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
)

func failf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

// version is stamped at build time via:
//
//	go build -ldflags "-X main.version=<v>" ./compiler
//
// Unstamped builds (e.g. plain `go install ...@latest`) report "dev".
var version = "dev"

// Bound by the development bundle builder; an ordinary compiler build has no sidecar.
var bundleManifestSHA256 string

func run(argv []string) int {
	if len(argv) > 0 && argv[0] == "assert" {
		timeoutMs := driver.DefaultAssertTimeoutMs
		rest := argv[1:]
		if len(rest) >= 1 && rest[0] == "--assert-timeout-ms" {
			if len(rest) < 3 {
				fmt.Fprintln(os.Stderr, "usage: canlc assert [--assert-timeout-ms 1..600000] PROJECT [PACKAGE [DECLARATION] ASSERTION]")
				return 2
			}
			parsed, parseErr := driver.ParseAssertTimeoutMs(rest[1])
			if parseErr != nil {
				fmt.Fprintln(os.Stderr, "usage: canlc assert [--assert-timeout-ms 1..600000] PROJECT [PACKAGE [DECLARATION] ASSERTION]")
				fmt.Fprintln(os.Stderr, parseErr)
				return 2
			}
			timeoutMs = parsed
			rest = rest[2:]
		}
		if len(rest) != 1 && len(rest) != 3 && len(rest) != 4 {
			fmt.Fprintln(os.Stderr, "usage: canlc assert [--assert-timeout-ms 1..600000] PROJECT [PACKAGE [DECLARATION] ASSERTION]")
			return 2
		}
		sidecar, err := driver.Resolve(bundleManifestSHA256)
		if err == nil {
			err = sidecar.Assert(context.Background(), rest[0], rest[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr, timeoutMs)
		}
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				if code := exit.ExitCode(); code > 0 {
					return code
				}
				return 1
			}
			if errors.Is(err, driver.ErrAssertionsFailed) {
				return 1
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if len(argv) > 0 && (argv[0] == "build" || argv[0] == "run") {
		if len(argv) < 2 || argv[1] == "" || (argv[0] == "build" && len(argv) != 2) || (argv[0] == "run" && len(argv) > 2 && argv[2] != "--") {
			fmt.Fprintln(os.Stderr, "usage: canlc build PROJECT_DIRECTORY | canlc run PROJECT_DIRECTORY [-- APPLICATION_ARGS...]")
			return 2
		}
		sidecar, err := driver.Resolve(bundleManifestSHA256)
		if err == nil {
			if argv[0] == "build" {
				var report driver.BuildReport
				report, err = sidecar.Build(context.Background(), argv[1])
				if err == nil {
					err = json.NewEncoder(os.Stdout).Encode(report)
				}
			} else {
				var args []string
				if len(argv) > 2 {
					args = argv[3:]
				}
				err = sidecar.Run(context.Background(), argv[1], args, os.Environ(), os.Stdin, os.Stdout, os.Stderr)
			}
		}
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok && argv[0] == "run" {
				if code := exit.ExitCode(); code > 0 {
					return code
				}
				return 1
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if len(argv) > 0 && argv[0] == "inspect-types" {
		return runInspectTypes(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "inspect-project" {
		return runInspectProject(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "parse" {
		return runCurrentParse(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && (argv[0] == "runtime-check" || argv[0] == "catalogue-check") {
		sidecar, err := driver.Resolve(bundleManifestSHA256)
		if err == nil {
			entry := "tools/runtime/check.ts"
			if argv[0] == "catalogue-check" {
				entry = "tools/runtime/catalogue-check.ts"
			}
			err = sidecar.RunTool(context.Background(), entry, argv[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	if len(argv) > 0 && (argv[0] == "--version" || argv[0] == "-version" || argv[0] == "version") {
		fmt.Printf("canlc %s\n", version)
		return 0
	}
	if len(argv) > 0 && argv[0] == "lsp" {
		return runLSP(argv[1:])
	}
	if len(argv) > 0 && (argv[0] == "explain" || argv[0] == "lint" || argv[0] == "baseline" || argv[0] == "normalize") {
		fmt.Fprintf(os.Stderr, "canlc %s was retired with the predecessor toolchain in I44\n", argv[0])
		return 2
	}
	if len(argv) > 0 && argv[0] == "clean" {
		if len(argv) != 2 {
			fmt.Fprintln(os.Stderr, "usage: canlc clean PROJECT_DIRECTORY")
			return 2
		}
		output, err := driver.BeginOutput(argv[1])
		if err == nil {
			defer output.Close()
			err = output.Clean()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(os.Stderr, "usage: canlc parse FILE | inspect-project PROJECT | inspect-types PROJECT | clean PROJECT")
	return 2
}

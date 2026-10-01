// Package main is the Can compiler launcher and manifest-backed tooling.
// Current emission/publication lives in internal/emit and internal/driver.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
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
		jobs := driver.DefaultAssertJobs()
		rest := argv[1:]
		assertUsage := "usage: canlc assert [--assert-timeout-ms 1..600000] [--assert-jobs 1..64] PROJECT [PACKAGE [DECLARATION] ASSERTION]"
		for len(rest) >= 1 && strings.HasPrefix(rest[0], "--") {
			if len(rest) < 3 {
				fmt.Fprintln(os.Stderr, assertUsage)
				return 2
			}
			switch rest[0] {
			case "--assert-timeout-ms":
				parsed, parseErr := driver.ParseAssertTimeoutMs(rest[1])
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, assertUsage)
					fmt.Fprintln(os.Stderr, parseErr)
					return 2
				}
				timeoutMs = parsed
			case "--assert-jobs":
				parsed, parseErr := driver.ParseAssertJobs(rest[1])
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, assertUsage)
					fmt.Fprintln(os.Stderr, parseErr)
					return 2
				}
				jobs = parsed
			default:
				fmt.Fprintln(os.Stderr, assertUsage)
				return 2
			}
			rest = rest[2:]
		}
		if len(rest) != 1 && len(rest) != 3 && len(rest) != 4 {
			fmt.Fprintln(os.Stderr, assertUsage)
			return 2
		}
		sidecar, err := driver.Resolve(bundleManifestSHA256)
		if err == nil {
			err = sidecar.Assert(context.Background(), rest[0], rest[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr, timeoutMs, jobs)
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
		timeoutMs := driver.DefaultAssertTimeoutMs
		jobs := driver.DefaultAssertJobs()
		target := browser.TargetBun
		browserManifest := ""
		rest := argv[1:]
		buildUsage := "usage: canlc build [--target bun|browser] [--assert-timeout-ms 1..600000] [--assert-jobs 1..64] [--browser-manifest FILE] PROJECT_DIRECTORY | canlc run [--assert-timeout-ms 1..600000] [--assert-jobs 1..64] PROJECT_DIRECTORY [-- APPLICATION_ARGS...]"
		for len(rest) >= 1 && strings.HasPrefix(rest[0], "--") {
			if len(rest) < 2 {
				fmt.Fprintln(os.Stderr, buildUsage)
				return 2
			}
			switch rest[0] {
			case "--assert-timeout-ms":
				parsed, parseErr := driver.ParseAssertTimeoutMs(rest[1])
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, buildUsage)
					fmt.Fprintln(os.Stderr, parseErr)
					return 2
				}
				timeoutMs = parsed
			case "--assert-jobs":
				parsed, parseErr := driver.ParseAssertJobs(rest[1])
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, buildUsage)
					fmt.Fprintln(os.Stderr, parseErr)
					return 2
				}
				jobs = parsed
			case "--target":
				if argv[0] != "build" {
					fmt.Fprintln(os.Stderr, buildUsage)
					return 2
				}
				parsed, parseErr := browser.ParseTarget(rest[1])
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, buildUsage)
					fmt.Fprintln(os.Stderr, parseErr)
					return 2
				}
				target = parsed
			case "--browser-manifest":
				if argv[0] != "build" || rest[1] == "" {
					fmt.Fprintln(os.Stderr, buildUsage)
					return 2
				}
				browserManifest = rest[1]
			default:
				fmt.Fprintln(os.Stderr, buildUsage)
				return 2
			}
			rest = rest[2:]
		}
		if argv[0] == "build" && target == browser.TargetBrowser && browserManifest != "" {
			fmt.Fprintln(os.Stderr, buildUsage)
			fmt.Fprintln(os.Stderr, "browser builds do not pair a browser manifest")
			return 2
		}
		if len(rest) < 1 || rest[0] == "" || strings.HasPrefix(rest[0], "--") || (argv[0] == "build" && len(rest) != 1) || (argv[0] == "run" && len(rest) > 1 && rest[1] != "--") {
			fmt.Fprintln(os.Stderr, buildUsage)
			return 2
		}
		sidecar, err := driver.Resolve(bundleManifestSHA256)
		if err == nil {
			if argv[0] == "build" {
				var report driver.BuildReport
				report, err = sidecar.BuildTarget(context.Background(), rest[0], os.Environ(), os.Stdin, os.Stderr, timeoutMs, jobs, target, browserManifest)
				if err == nil {
					err = json.NewEncoder(os.Stdout).Encode(report)
				}
			} else {
				var args []string
				if len(rest) > 1 {
					args = rest[2:]
				}
				err = sidecar.Run(context.Background(), rest[0], args, os.Environ(), os.Stdin, os.Stdout, os.Stderr, timeoutMs, jobs)
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
	if len(argv) > 0 && argv[0] == "test" {
		return runTest(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "check" {
		return runCheck(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "format" {
		return runCurrentFormat(os.Stdout, os.Stderr, argv[1:])
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
	fmt.Fprintln(os.Stderr, "usage: canlc parse FILE | format [--write] FILE | inspect-project PROJECT | inspect-types PROJECT | clean PROJECT")
	return 2
}

// testListSchema is the only --list envelope version P21 emits. Unknown
// schemas refuse before any project load.
const testListSchema = "1"

// testListRoot is one nonexecutingly listed assertion root.
type testListRoot struct {
	Package     string `json:"package"`
	Declaration string `json:"declaration"`
	Name        string `json:"name"`
}

// testListDoc is the versioned --list envelope.
type testListDoc struct {
	Schema    string         `json:"schema"`
	Kind      string         `json:"kind"`
	Project   string         `json:"project"`
	Candidate string         `json:"candidate"`
	Reference string         `json:"reference"`
	Roots     []testListRoot `json:"roots"`
}

const testUsage = "usage: canlc test --candidate PKG --reference PKG [--schema 1] [--list] PROJECT | canlc test --candidate PKG --reference PKG [--schema 1] --reference-toolchain PATH --owner-dir PATH PROJECT"

// runTest implements the narrow P21 test dispatch: explicit candidate and
// reference selection, a nonexecuting --list over the checked candidate
// roots, and presence-gated execution parameters. Plans, retries and
// verdicts stay in Can with N as owner only; live execution refuses until
// the P23 runner exists. It never stages, publishes, or spawns.
func runTest(stdout, stderr io.Writer, argv []string) int {
	var candidate, reference, schema, toolchain, ownerDir string
	schema = testListSchema
	list := false
	var positional []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--list" {
			list = true
			continue
		}
		if arg == "--candidate" || arg == "--reference" || arg == "--schema" || arg == "--reference-toolchain" || arg == "--owner-dir" {
			if i+1 >= len(argv) || argv[i+1] == "" || strings.HasPrefix(argv[i+1], "--") {
				fmt.Fprintln(stderr, testUsage)
				return 2
			}
			i++
			switch arg {
			case "--candidate":
				candidate = argv[i]
			case "--reference":
				reference = argv[i]
			case "--schema":
				schema = argv[i]
			case "--reference-toolchain":
				toolchain = argv[i]
			case "--owner-dir":
				ownerDir = argv[i]
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			fmt.Fprintln(stderr, testUsage)
			return 2
		}
		positional = append(positional, arg)
	}
	if candidate == "" || reference == "" || len(positional) != 1 {
		fmt.Fprintln(stderr, testUsage)
		return 2
	}
	if schema != testListSchema {
		fmt.Fprintln(stderr, testUsage)
		fmt.Fprintf(stderr, "unknown test schema %q: want %q\n", schema, testListSchema)
		return 2
	}
	if list && (toolchain != "" || ownerDir != "") {
		fmt.Fprintln(stderr, testUsage)
		fmt.Fprintln(stderr, "--list takes no execution parameters")
		return 2
	}
	if !list && (toolchain == "" || ownerDir == "") {
		fmt.Fprintln(stderr, testUsage)
		fmt.Fprintln(stderr, "test execution needs --reference-toolchain PATH and --owner-dir PATH")
		return 2
	}
	directory := positional[0]
	graph, err := project.Load(directory)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(graph.Errors) != 0 {
		fmt.Fprintln(stderr, graph.Errors[0])
		return 1
	}
	ids := map[string]string{}
	for _, pkg := range graph.Packages {
		ids[pkg.Name] = pkg.ID
	}
	candidateID, ok := ids[candidate]
	if !ok {
		fmt.Fprintf(stderr, "unknown candidate package %q\n", candidate)
		return 1
	}
	if _, ok := ids[reference]; !ok {
		fmt.Fprintf(stderr, "unknown reference package %q\n", reference)
		return 1
	}
	if !list {
		if info, err := os.Stat(toolchain); err != nil || info.IsDir() {
			fmt.Fprintf(stderr, "reference toolchain absent: %s\n", toolchain)
			return 1
		}
		if info, err := os.Stat(ownerDir); err != nil || !info.IsDir() {
			fmt.Fprintf(stderr, "owner absent: %s\n", ownerDir)
			return 1
		}
		fmt.Fprintln(stderr, "test execution is gated by P23: selection validated, live R/N runner unimplemented")
		return 1
	}
	program, err := check.CheckAssertionProgram(graph)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	doc := testListDoc{Schema: testListSchema, Kind: "can.test.list", Project: directory, Candidate: candidate, Reference: reference, Roots: []testListRoot{}}
	for _, assertion := range program.Assertions {
		if assertion.Root.Package != candidateID {
			continue
		}
		doc.Roots = append(doc.Roots, testListRoot{Package: assertion.Root.Package, Declaration: assertion.Root.Declaration, Name: assertion.Root.Name})
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}

const checkUsage = "usage: canlc check --json [--schema 1] PROJECT"

// runCheck implements the narrow P21 check dispatch: the versioned JSON
// diagnostics document only. It delegates to the inert P20 checker,
// which never emits, publishes, runs, fetches, or mutates.
func runCheck(stdout, stderr io.Writer, argv []string) int {
	asJSON := false
	schema := driver.CheckSchemaVersion
	var positional []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--json" {
			asJSON = true
			continue
		}
		if arg == "--schema" {
			if i+1 >= len(argv) || argv[i+1] == "" || strings.HasPrefix(argv[i+1], "--") {
				fmt.Fprintln(stderr, checkUsage)
				return 2
			}
			i++
			schema = argv[i]
			continue
		}
		if strings.HasPrefix(arg, "--") {
			fmt.Fprintln(stderr, checkUsage)
			return 2
		}
		positional = append(positional, arg)
	}
	if !asJSON || len(positional) != 1 {
		fmt.Fprintln(stderr, checkUsage)
		return 2
	}
	if schema != driver.CheckSchemaVersion {
		fmt.Fprintln(stderr, checkUsage)
		fmt.Fprintf(stderr, "unknown check schema %q: want %q\n", schema, driver.CheckSchemaVersion)
		return 2
	}
	doc, exit := driver.CheckProjectJSON(context.Background(), positional[0])
	fmt.Fprintln(stdout, string(doc))
	return exit
}

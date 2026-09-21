// Package main is the Can compiler launcher and manifest-backed tooling.
// Current emission/publication lives in internal/emit and internal/driver.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	if len(argv) > 0 && argv[0] == "explain" {
		return runExplain(os.Stdout, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "lint" {
		return runLint(os.Stdout, os.Stderr, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "baseline" {
		return runBaseline(argv[1:])
	}
	if len(argv) > 0 && argv[0] == "normalize" {
		if err := runNormalize(os.Stdout, argv[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "canlc FAILED: %v\n", err)
			return 1
		}
		return 0
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

// parseBaselineArgs parses `canlc baseline --out BASE.json [--origin ID]
// file.can [...]`. Unknown flags and missing values are usage errors.
func parseBaselineArgs(argv []string) (out, origin string, args []string, err error) {
	fs := flag.NewFlagSet("canlc baseline", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out, "out", "", "candidate baseline to write (JSON)")
	fs.StringVar(&origin, "origin", "", "origin id recorded in the baseline")
	if err := fs.Parse(argv); err != nil {
		return "", "", nil, err
	}
	return out, origin, fs.Args(), nil
}

func runBaseline(argv []string) int {
	out, origin, args, err := parseBaselineArgs(argv)
	if err != nil || out == "" || len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: canlc baseline --out BASE.json [--origin ID] file.can [...]")
		return 2
	}
	mods, texts, collected, err := legacyParsePaths(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "canlc FAILED: %v\n", err)
		return 1
	}
	prog, collected := checkProgram(mods, texts, collected, nil)
	if err := firstError(collected); err != nil {
		fmt.Fprintf(os.Stderr, "canlc FAILED: %v\n", err)
		return 1
	}
	if origin == "" {
		origin = "candidate"
	}
	if err := WriteBaseline(out, prog, origin); err != nil {
		fmt.Fprintf(os.Stderr, "canlc FAILED: %v\n", err)
		return 1
	}
	fmt.Printf("canlc: candidate baseline for %d declarations written to %s (unaccepted)\n", len(FingerprintProgram(prog)), out)
	return 0
}

// runNormalize implements `canlc normalize file.can [...]`: gate on the
// full suite, then print every decision-table outcome in canonical form,
// one `mod.fn/test => value` line per test, sorted. Expectation mismatches
// still print (the outcome is the artifact); only execution failures —
// missing scripts, unbound names, leftover stubs — fail the command.
// Exit 0 on print: normalize observes, compile gates.
func runNormalize(w io.Writer, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("usage: canlc normalize file.can [...]")
	}
	mods, _, collected, err := legacyParsePaths(paths)
	if err != nil {
		return err
	}
	prog, collected := checkProgram(mods, nil, collected, nil)
	if ferr := firstError(collected); ferr != nil {
		return ferr
	}
	var lines []string
	for _, m := range mods {
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			for _, t := range fn.Tests {
				got, _, err := runTestValue(fn, t, prog, nil)
				if err != nil {
					return err
				}
				lines = append(lines, fmt.Sprintf("%s.%s/%s => %s", m.Mod, fn.Name, t.Name, normalizeValue(got)))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	return nil
}

// Predecessor-only loading. Current projects use project.Load and resolve.Build;
// this historical entry point remains solely for the scheduled old pipeline
// retirement, with no adaptation into current identities or output paths.
// legacyParsePaths reads and parses every path, collecting CAN1000 diagnostics
// for files that do not parse instead of failing fast, so one broken file
// never hides the rest. Raw IO errors still fail immediately.
//
// Identity is the cleaned input path: two inputs with different
// identities are different modules even when their basenames match.
// Output stems stay bare while unique, then disambiguate by directory;
// the same identity twice is an CAN5007 collision, rejected before
// evaluation or writing.
func legacyParsePaths(paths []string) (mods []*Module, texts map[string]string, collected []Diag, err error) {
	texts = map[string]string{}
	seen := map[string]bool{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, nil, nil, err
		}
		m, perr := parseModuleText(p, string(data))
		if perr != nil {
			base := filepath.Base(filepath.Clean(p))
			collected = append(collected, Diag{File: base, Line: diagLine(perr, 1), Sev: "error", Msg: stripLinePrefix(perr), Code: CodeParse})
			continue
		}
		if seen[m.ID] {
			collected = append(collected, Diag{File: m.File, Line: 1, Sev: "error",
				Msg: fmt.Sprintf("module %s inputs twice: one canonical identity per file", m.ID), Code: CodeModuleCollision})
			continue
		}
		seen[m.ID] = true
		mods = append(mods, m)
		texts[m.ID] = string(data)
	}
	legacyAssignStems(mods)
	return mods, texts, collected, nil
}

// legacyAssignStems gives every module an injective output stem. The bare
// stem (filename without extension) wins while unique, so single-file
// and distinct-name inputs emit exactly as before; every sharer of a
// basename takes the sanitized identity path instead (in sorted
// identity order), with numeric suffixes breaking residual ties.
// Deterministic in the input set, never silently merging two owners
// into one artifact.
func legacyAssignStems(mods []*Module) {
	count := map[string]int{}
	for _, m := range mods {
		count[m.Stem]++
	}
	used := map[string]bool{}
	ordered := append([]*Module{}, mods...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	for _, m := range ordered {
		if count[m.Stem] == 1 && !used[m.Stem] {
			used[m.Stem] = true
			continue
		}
		candidate := sanitizeStem(strings.TrimSuffix(m.ID, ".can"))
		for n := 2; used[candidate]; n++ {
			candidate = fmt.Sprintf("%s_%d", sanitizeStem(strings.TrimSuffix(m.ID, ".can")), n)
		}
		m.Stem = candidate
		used[candidate] = true
	}
}

// sanitizeStem maps an identity path to stem characters: every run of
// non-letters-and-digits (separators included) becomes one underscore,
// with leading/trailing underscores trimmed. Underscore itself maps to
// itself, so "a/b" and "a_b" can still collide — legacyAssignStems breaks
// that tie with a numeric suffix.
func sanitizeStem(id string) string {
	var b strings.Builder
	prev := '_'
	for _, r := range id {
		var c rune
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			c = r
		default:
			c = '_'
		}
		if c == '_' && prev == '_' {
			continue
		}
		b.WriteRune(c)
		prev = c
	}
	return strings.Trim(b.String(), "_")
}

// checkProgram runs the full shared suite over parsed modules: static
// checks, world build, semantic checks, exhaustiveness proof, and test
// runs. pass fires per passing test. Execution-dependent phases are
// skipped on a dirty world, mirroring the editor.
func checkProgram(mods []*Module, texts map[string]string, collected []Diag, pass func(mod, fn, test string)) (*Program, []Diag) {
	// G1 expansion first: templates become monomorphic stamps,
	// so every phase below consumes plain checked shapes.
	if diags := expandGenerics(mods, texts); len(diags) > 0 {
		collected = append(collected, diags...)
		for _, d := range diags {
			if d.Sev == "error" {
				return nil, collected
			}
		}
	}
	for _, m := range mods {
		collected = append(collected, checkStatic(m, texts[m.ID])...)
	}
	if len(mods) == 0 {
		return nil, collected
	}
	prog, world := buildWorld(mods[0], mods, texts)
	collected = append(collected, world...)
	if hasErrors(world) {
		return prog, collected
	}
	// a46 S2 barrier: export certificates issue whole-program before
	// any linkage evaluation, so an uncertified refusal can never
	// launder into trusted script evidence.
	collected = append(collected, certifyExports(mods, prog, texts)...)
	// S2 slice plan barrier: asset bridge certificates issue under
	// the same whole-program rule, for the same reason.
	collected = append(collected, certifyAssetBridge(mods, prog, texts)...)

	// World-level termination refusal (a11): cross-file cycles are
	// reported per-line and suppress only execution-dependent checks,
	// per the R10 world-error rule.
	global := checkGlobalCycles(mods, texts, prog)
	collected = append(collected, global...)
	// Finite products only (first cut): record-type cycles fail before
	// tests or output through the same execution gate.
	recCycles := checkRecordCycles(mods, texts)
	collected = append(collected, recCycles...)
	// Invocation-closed cycles refuse termination through the same
	// gate: dynamic dispatch admits no decreases proof.
	invokeCycles := checkInvokeCycles(mods, texts, prog)
	collected = append(collected, invokeCycles...)
	gblocked := hasErrors(global) || hasErrors(recCycles) || hasErrors(invokeCycles)
	// Linked-invoke barrier (CLI mirror of prepareProviders in
	// lsp.go): the checkSem loop checks and RUNS each module in
	// paths order, but cross-file invocation executes provider
	// bodies linked — including multi-arg constructors that only
	// bind after the provider's static phase. Run every module's
	// function-static phase silently first, so no test row
	// anywhere can execute a raw provider body. Diagnostics still
	// surface from each module's own checkSem below; the static
	// phase is idempotent, so the second run is a no-op.
	for _, m := range mods {
		localExtern := map[string]bool{}
		for _, d := range m.Decls {
			if ex, ok := d.(*ExternDecl); ok {
				localExtern[ex.Name] = true
			}
		}
		called := map[string]bool{}
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			_ = checkFnStatic(fn, prog, m, texts[m.ID], localExtern, called)
		}
	}
	for _, m := range mods {
		text := texts[m.ID]
		var hook func(fn, test string)
		if pass != nil {
			mod := m.Mod
			hook = func(fn, test string) { pass(mod, fn, test) }
		}
		collected = append(collected, checkSem(m, text, prog, hook, gblocked)...)
	}
	for _, m := range mods {
		for _, err := range verifyExhaustiveAll([]*Module{m}, prog) {
			d := proofDiag(texts[m.ID], err)
			d.File = m.File
			collected = append(collected, d)
		}
	}
	return prog, collected
}

// hasErrors reports whether any diagnostic is an error.
func hasErrors(diags []Diag) bool {
	return firstError(diags) != nil
}

// firstError returns the first error diagnostic as a file:line: message,
// dropping a redundant filename the message already carries.
func firstError(diags []Diag) error {
	for _, d := range diags {
		if d.Sev == "error" {
			msg := d.Msg
			if d.File != "" {
				msg = strings.TrimPrefix(msg, d.File+": ")
				return fmt.Errorf("%s:%d: %s", d.File, d.Line, msg)
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

// failDiags prints every diagnostic as JSON lines in json mode, then
// returns the first error. Prose mode just returns it; run() prints the
// familiar single line.
func failDiags(collected []Diag, jsonOut bool) error {
	if jsonOut {
		reportDiags(os.Stdout, collected)
	}
	return firstError(collected)
}

// jsonDiag is the machine rendering of a Diag: same facts as the editor
// squiggle plus the stable code. Field order is fixed for golden tests.
type jsonDiag struct {
	Code     string `json:"code"`
	Sev      string `json:"sev"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Msg      string `json:"msg"`
	Expected string `json:"expected,omitempty"`
	Found    string `json:"found,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// reportDiags writes diagnostics sorted file-top to file-bottom, one JSON
// object per line. stdout stays pure JSON: callers must suppress every
// other print in json mode.
func reportDiags(w io.Writer, diags []Diag) {
	cp := append([]Diag(nil), diags...)
	sortDiags(cp)
	for _, d := range cp {
		body, err := json.Marshal(jsonDiag{
			Code: d.Code, Sev: d.Sev, File: d.File, Line: d.Line,
			Start: d.Start, End: d.End, Msg: d.Msg,
			Expected: d.Expected, Found: d.Found, Hint: d.Hint,
		})
		if err != nil {
			continue
		}
		fmt.Fprintln(w, string(body))
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	c := s[0]
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return string(c) + s[1:]
}

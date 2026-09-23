package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// runCurrentFormat renders one source file canonically with trivia
// preserved. Stdout mode is pure syntax; --write additionally resolves
// the containing project and validates both the original and the
// proposed content through overlay snapshots before atomic replacement.
// Anything invalid, stale or inequivalent leaves the file untouched.
func runCurrentFormat(stdout, stderr io.Writer, args []string) int {
	flags := flag.NewFlagSet("format", flag.ContinueOnError)
	flags.SetOutput(stderr)
	write := flags.Bool("write", false, "validate and replace the file in place")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: canlc format [--write] FILE.can")
		return 2
	}
	path := flags.Arg(0)
	before, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var identity os.FileInfo
	if *write {
		identity, err = os.Lstat(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	formatted, err := formatSource(path, string(before))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !*write {
		if _, err := io.WriteString(stdout, formatted); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if err := writeFormatted(path, identity, string(before), formatted); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func formatSource(path, text string) (string, error) {
	file, err := source.New(path, text)
	if err != nil {
		return "", err
	}
	parsed := syntax.Parse(file)
	if !parsed.OK() {
		return "", fmt.Errorf("%s", parsed.Diagnostics[0].Format(file))
	}
	out, err := syntax.FormatTrivia(parsed.File)
	if err != nil {
		return "", fmt.Errorf("%s: %v", path, err)
	}
	// The formatted text must reparse and sit at the fixpoint; anything
	// else proves the renderer moved more than trivia.
	reparsed, err := source.New(path, out)
	if err != nil {
		return "", err
	}
	check := syntax.Parse(reparsed)
	if !check.OK() {
		return "", fmt.Errorf("%s: formatted output fails to parse: %s", path, check.Diagnostics[0].Format(reparsed))
	}
	again, err := syntax.FormatTrivia(check.File)
	if err != nil || again != out {
		return "", fmt.Errorf("%s: formatted output is not stable", path)
	}
	return out, nil
}

// writeFormatted validates the replacement against the containing
// project: the original must check clean, the proposed content must
// check clean through an overlay, and the file must be unchanged since
// it was read. The replacement is atomic; every failure path leaves
// the original bytes in place.
func writeFormatted(path string, identity os.FileInfo, before, formatted string) error {
	if before == formatted {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	root := discoverRoot(real)
	if info, err := os.Lstat(filepath.Join(root, "can.project.json")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s: --write requires an enclosing can.project.json", path)
	}
	original, err := driver.CheckSnapshot(root, real, project.NewOverlay())
	if err != nil {
		return err
	}
	if len(original.Diagnostics) != 0 {
		return fmt.Errorf("%s: cannot format: %s", path, describeDiagnostic(original.Diagnostics[0]))
	}
	overlay := project.NewOverlay()
	if err := overlay.Set(real, 1, formatted); err != nil {
		return err
	}
	proposed, err := driver.CheckSnapshot(root, real, overlay)
	if err != nil {
		return err
	}
	if len(proposed.Diagnostics) != 0 {
		return fmt.Errorf("%s: formatted output fails to check: %s", path, describeDiagnostic(proposed.Diagnostics[0]))
	}
	currentInfo, err := os.Lstat(real)
	if err != nil {
		return err
	}
	if identity == nil || !os.SameFile(identity, currentInfo) {
		return fmt.Errorf("%s: source file was replaced since read; refusing to write", path)
	}
	current, err := os.ReadFile(real)
	if err != nil {
		return err
	}
	if string(current) != before {
		return fmt.Errorf("%s: source changed since read; refusing to write", path)
	}
	replacement, err := os.CreateTemp(filepath.Dir(real), ".can-format-*")
	if err != nil {
		return err
	}
	name := replacement.Name()
	defer os.Remove(name)
	if _, err := io.WriteString(replacement, formatted); err != nil {
		replacement.Close()
		return err
	}
	if err := replacement.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, currentInfo.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(name, real)
}

func describeDiagnostic(diagnostic driver.Diagnostic) string {
	if diagnostic.Code == "" {
		return diagnostic.Message
	}
	return diagnostic.Code + ": " + diagnostic.Message
}

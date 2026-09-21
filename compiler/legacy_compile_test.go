package main

// Historical fixture emitter, excluded from the shipped compiler. Current
// output publication uses the manifest-owned generation driver exclusively.
import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func parseCompileArgs(argv []string) (out, format, baseline string, args []string, err error) {
	fs := flag.NewFlagSet("canlc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out, "out", "", "output directory for emitted modules")
	fs.StringVar(&format, "format", "", "output format (json)")
	fs.StringVar(&baseline, "baseline", "", "accepted revision baseline (JSON)")
	if err := fs.Parse(argv); err != nil {
		return "", "", "", nil, err
	}
	return out, format, baseline, fs.Args(), nil
}

func compile(out string, paths []string) error {
	return compileEx(out, paths, false)
}

// compileEx runs the same full suite the editor runs (checkStatic,
// world, checkSem, proof, tests), so "no squiggles" and "compiles" mean
// the same thing. Prose mode preserves the old fail-fast behavior by
// returning the first error; json mode prints every diagnostic as one
// JSON object per line on stdout and stays silent on success.
func compileEx(out string, paths []string, jsonOut bool) error {
	return compileAll(out, paths, jsonOut, "")
}

// compileBaselined runs the full suite with revision-identity
// enforcement (a77): after the ordinary gate passes, the accepted
// baseline is compared and drift, removals, and additions fail the
// build. An empty baseline path selects no enforcement.
func compileBaselined(out string, paths []string, jsonOut bool, baselinePath string) error {
	return compileAll(out, paths, jsonOut, baselinePath)
}

func compileAll(out string, paths []string, jsonOut bool, baselinePath string) error {
	mods, texts, collected, err := legacyParsePaths(paths)
	if err != nil {
		return err
	}
	pass := func(mod, fn, test string) {}
	if !jsonOut {
		pass = func(mod, fn, test string) {
			fmt.Printf("PASS %s.%s/%s\n", mod, fn, test)
		}
	}
	prog, collected := checkProgram(mods, texts, collected, pass)
	if err := firstError(collected); err != nil {
		return failDiags(collected, jsonOut)
	}
	var base *RevisionBaseline
	if baselinePath != "" {
		var err error
		base, err = LoadBaseline(baselinePath)
		if err != nil {
			return fmt.Errorf("canlc: cannot load baseline: %v", err)
		}
		if idDiags := CheckRevisionIdentity(prog, texts, base); len(idDiags) > 0 {
			for _, d := range idDiags {
				if d.File == "" {
					d.File = mods[0].File
				}
			}
			collected = append(collected, idDiags...)
			return failDiags(collected, jsonOut)
		}
	}
	// a82: verifier activation. Once the ordinary gate passes, every
	// contracted function must satisfy admission and proof:
	// unsupported, unproven, or inconclusive contracts block
	// acceptance. Uncontracted functions keep ordinary status.
	if vcDiags := VerifyContracts(prog, texts); len(vcDiags) > 0 {
		for _, d := range vcDiags {
			if d.File == "" {
				d.File = mods[0].File
			}
		}
		collected = append(collected, vcDiags...)
		return failDiags(collected, jsonOut)
	}
	if !jsonOut {
		printVerificationReport(prog)
	}
	// a87: pinned-row weakening is advisory. It reports only when the
	// world is otherwise clean (same ordering as identity and proof),
	// prints loudly on both output modes, and never blocks emit.
	if base != nil {
		if pinDiags := CheckPinnedRows(prog, texts, base); len(pinDiags) > 0 {
			// Acceptance changes are warnings; inability to canonicalize
			// evidence is an error and must prevent artifact emission.
			if firstError(pinDiags) != nil {
				return failDiags(pinDiags, jsonOut)
			}
			if jsonOut {
				reportDiags(os.Stdout, pinDiags)
			} else {
				for _, d := range pinDiags {
					fmt.Fprintf(os.Stderr, "canlc: warning %s:%d: %s\n", d.File, d.Line, d.Msg)
				}
			}
		}
	}
	for _, m := range mods {
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			seen := map[string]bool{}
			for _, c := range walkCalls(fn.Body) {
				if prog.Uses[c.Fname] && !seen[c.Fname] {
					seen[c.Fname] = true
					fn.UsesHere = append(fn.UsesHere, c.Fname)
				}
			}
		}
	}
	total := 0
	for _, m := range mods {
		for _, d := range m.Decls {
			if fn, ok := d.(*FnDecl); ok {
				total += len(fn.Tests)
			}
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	stemOf := map[string]string{}
	resultOfStem := map[string]string{}
	fnUnions := map[string]string{}
	for _, m := range mods {
		resultOfStem[m.Stem] = capitalize(m.Mod) + "Result"
		for _, d := range m.Decls {
			switch d := d.(type) {
			case *FnDecl:
				stemOf[d.Name] = m.Stem
				// Per-function unions (see fnResultUnion): call
				// temporaries and return annotations carry the
				// callee's own outcomes, never the module-wide
				// union, so strict checkers narrow exactly.
				u, err := fnResultUnion(d, prog)
				if err != nil {
					return err
				}
				fnUnions[d.Name] = u
			case *ExternDecl:
				// b02: externs join the stem table like
				// functions, so shared-extern imports
				// resolve to the declaring stem.
				stemOf[d.Name] = m.Stem
				u, err := externUnion(d, prog)
				if err != nil {
					return err
				}
				fnUnions[d.Name] = u
			case *TypeDecl:
				stemOf[d.Name] = m.Stem
			case *VariantDecl:
				// a74: variant parents join the stem table like
				// records, so cross-module type references import
				// the union from the provider stem.
				stemOf[d.Name] = m.Stem
			}
		}
	}
	for _, m := range mods {
		text, err := emitModule(m, prog, stemOf, resultOfStem, fnUnions)
		if err != nil {
			return err
		}
		name := m.Stem + ".ts"
		if err := os.WriteFile(out+"/"+name, []byte(text), 0o644); err != nil {
			return err
		}
		if !jsonOut {
			fmt.Printf("EMIT %s\n", name)
		}
	}
	catalog, err := json.MarshalIndent(buildCatalog(mods, prog, texts), "", "  ")
	if err != nil {
		return err
	}
	catalog = append(catalog, '\n')
	if err := os.WriteFile(out+"/errors.json", catalog, 0o644); err != nil {
		return err
	}
	if !jsonOut {
		fmt.Printf("canlc: %d tests passed, %d modules emitted to %s\n", total, len(mods), out)
	}
	return nil
}

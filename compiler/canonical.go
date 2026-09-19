package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// canonicalError is an internal fail-closed signal, never a serializable value.
// The baseline/diagnostic boundaries catch only this signal; unrelated compiler
// panics remain bugs. No partially canonicalized record can be written or used.
type canonicalError string

func (e canonicalError) Error() string { return "cannot canonicalize evidence: " + string(e) }

func canonicalDiags(prog *Program, texts map[string]string, out *[]Diag) {
	if r := recover(); r != nil {
		e, ok := r.(canonicalError)
		if !ok {
			panic(r)
		}
		file, line := revisionFallback(prog)
		d := spanDiag(texts[file], line, "error", e.Error(), "mod", CodeRevisionIdentity)
		*out = []Diag{d}
	}
}

func canonType(t string) string { return strings.Join(strings.Fields(t), "") }

func canonTypes(ts []string) string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = canonType(t)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// Encode the label discriminator as well as its spelling: a named "pos"
// argument is not a positional argument. Array order retains evaluation order.
func canonArgs(args []Arg) string {
	type argument struct {
		Named bool
		Name  string
		Value string
	}
	out := make([]argument, 0, len(args))
	for _, a := range args {
		name := ""
		if a.HasName {
			name = a.Name
		}
		out = append(out, argument{a.HasName, name, canonSmall(a.V)})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// Validate executable structure even though bodies must NOT enter interface
// identity. Given scripts are checked separately; they are not executable bodies.
func validateCanonicalProgram(prog *Program) {
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			if fn, ok := d.(*FnDecl); ok {
				canonNode(fn.Body)
				everySmall(fn, func(site smallSite) { canonSmall(site.s) })
			}
		}
	}
}

// A row's structural spelling is separate from the identities it references.
// Fn references bind a resolved owner/revision and interface dependency closure,
// not an unversioned spelling. Bodies remain outside acceptance expectation
// identity: changing implementation without changing the promised value is not
// weakening that expectation, and proofs are freshly checked (no proof cache).
func canonRowDependencies(prog *Program, entries map[string]RevisionEntry, row Test) string {
	deps := map[string]string{}
	var add func(string)
	add = func(key string) {
		if _, seen := deps[key]; seen {
			return
		}
		e, ok := entries[key]
		if !ok {
			panic(canonicalError("missing referenced identity " + key))
		}
		deps[key] = e.Fingerprint
		// Interface drift deliberately ignores a pure dependency re-key;
		// acceptance references retain exact revisions throughout the closure.
		for _, dep := range e.Deps {
			add(dep)
		}
	}
	types := map[string]bool{}
	everySmall(&FnDecl{Tests: []Test{row}}, func(site smallSite) {
		s := site.s
		for _, t := range s.TypeArgs {
			revisionTypeDeps(t, types)
		}
		switch s.Kind {
		case "fnref":
			fn := prog.Fns[s.Fname]
			if fn == nil {
				panic(canonicalError("unresolved function reference " + s.Fname))
			}
			found := false
			for _, m := range prog.Modules {
				if m.ID != prog.FnFile[s.Fname] {
					continue
				}
				key := revisionKey("fn", m.Mod, fn.Name, fn.Rev, true)
				add(key)
				found = true
			}
			if !found {
				panic(canonicalError("missing function owner " + s.Fname))
			}
		case "ctor":
			name := s.Ctor
			if parent := prog.Cases[name]; parent != "" {
				name = parent
			}
			types[name] = true
		case "seal":
			types[s.Seal] = true
		case "seqlit":
			revisionTypeDeps(s.Elem, types)
		}
	})
	for key := range entries {
		_, rest, _ := strings.Cut(key, ":")
		_, name, _ := strings.Cut(rest, ".")
		if i := strings.LastIndex(name, "@"); i >= 0 {
			name = name[:i]
		}
		if types[name] {
			add(key)
		}
	}
	var keys []string
	for key := range deps {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	pairs := make([][2]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, [2]string{key, deps[key]})
	}
	b, _ := json.Marshal(pairs)
	return fmt.Sprintf(" dependencies/v%d:%s", RevisionFormat, b)
}

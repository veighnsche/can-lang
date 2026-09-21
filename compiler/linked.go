package main

// a68: runLinkedPure, the explicitly selected linked-pure
// integration runner (a67 §1.3, verdict B). Committed Go
// tests name an explicit module set, a root function and
// revision, and arguments; the reachable pure graph
// executes with real bodies and no scripts. Ordinary
// given semantics, coverage law, and emission are
// untouched: this path never runs outside tests.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkLinkedGraph admits exactly the linked-executable
// graph: checked CAN source or an admitted deterministic
// kernel, every branch inspected. Externs, state
// operations, effectful functions, and unresolved calls
// refuse before anything executes. The visited set ends
// cycles in the walk; cross-module cycles already failed
// the whole-program precondition, and the runtime depth
// bound backstops the rest.
func checkLinkedGraph(prog *Program, root string) error {
	seen := map[string]bool{}
	queue := []string{root}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		// Intrinsics other than state ops are admitted
		// deterministic kernels; state ops refuse.
		if kind := classifyCallee(prog, "", name); kind.IsIntrinsic() {
			if kind == CalleeStoreOp {
				return fmt.Errorf("linked execution refused: state operation %s", name)
			}
			continue
		}
		if prog.Externs[name] != nil {
			return fmt.Errorf("linked execution refused: extern %s", name)
		}
		fn, ok := prog.Fns[name]
		if !ok {
			return fmt.Errorf("linked execution refused: unresolved call %s", name)
		}
		if len(fn.Effects) > 0 {
			return fmt.Errorf("linked execution refused: effectful function %s", name)
		}
		for _, c := range walkCalls(fn.Body) {
			queue = append(queue, c.Fname)
		}
	}
	return nil
}

// runLinkedPure executes root@rev over files (loaded in
// order) with args, comparing the actual value against
// expect. The ordinary whole-program suite must be fully
// clean first; the root revision must match; each vector
// gets a fresh store and a discarded coverage map, so
// integration traces never satisfy CAN4107. A modeling,
// domain, or resource failure fails the test, never
// trusted evidence.
func runLinkedPure(t *testing.T, files map[string]string, order []string, root string, rev int, args map[string]string, expect string) error {
	t.Helper()
	dir := t.TempDir()
	full := make([]string, 0, len(order))
	for _, p := range order {
		src, ok := files[p]
		if !ok {
			return fmt.Errorf("linked: no source for %s", p)
		}
		fp := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fp, []byte(src), 0o644); err != nil {
			return err
		}
		full = append(full, fp)
	}
	mods, texts, collected, err := legacyParsePaths(full)
	if err != nil {
		return err
	}
	prog, collected := checkProgram(mods, texts, collected, nil)
	for _, d := range collected {
		if d.Sev == "error" {
			return fmt.Errorf("linked precondition failed: %s", d.Msg)
		}
	}
	fn, ok := prog.Fns[root]
	if !ok {
		return fmt.Errorf("linked root %s resolves nowhere", root)
	}
	if fn.Rev != rev {
		return fmt.Errorf("linked root %s is rev %d, want %d", root, fn.Rev, rev)
	}
	if err := checkLinkedGraph(prog, root); err != nil {
		return err
	}
	tmp := &Ctx{Prog: prog, Test: "linked", Linked: true}
	env := map[string]*Value{}
	for _, p := range fn.Params {
		src, ok := args[p[0]]
		if !ok {
			return fmt.Errorf("linked %s: missing arg %s", root, p[0])
		}
		sm, err := parseSmall(src)
		if err != nil {
			return err
		}
		v, err := evSmall(sm, map[string]*Value{}, tmp, fn.Name)
		if err != nil {
			return err
		}
		env[p[0]] = v
	}
	ctx, err := freshExecCtx(prog, "linked", true)
	if err != nil {
		return err
	}
	got, err := evNode(fn.Body, env, ctx, fn.Name)
	if err != nil {
		return fmt.Errorf("linked execution of %s failed: %v", root, err)
	}
	want, err := parseSmall(expect)
	if err != nil {
		return err
	}
	// a92: the expect string parses raw, outside the checked
	// program, so its Ok spellings bind here against the root
	// return — the same want the provider body saw. Faults stay
	// with evaluation (a bad expect is a mismatch, not a diag).
	if want.Kind == "ctor" && want.Ctor == "Ok" {
		bc := newTycker(prog, "", root)
		benv := map[string]string{}
		for _, p := range fn.Params {
			benv[p[0]] = p[1]
		}
		bc.checkCtor(want, fn.Ret, fn.Line, benv, "linked expect for "+root)
	}
	if want.Kind == "ctor" && want.Ctor == "Ok" {
		if got.Kind != "ok" {
			return fmt.Errorf("linked %s: expected Ok, got %s", root, describe(got))
		}
		exp, err := evSmall(want, env, ctx, fn.Name)
		if err != nil {
			return err
		}
		if valueHasFn(got) || valueHasFn(exp) {
			eq, detail, err := expectEq(exp, got)
			if err != nil {
				return err
			}
			if !eq {
				msg := fmt.Sprintf("linked %s: Ok payload mismatch: got %s, want %s", root, describe(got), describe(exp))
				if detail != "" {
					msg += ": " + detail
				}
				return errors.New(msg)
			}
			return nil
		}
		eq, err := vEq(got, exp)
		if err != nil || !eq {
			return fmt.Errorf("linked %s: Ok payload mismatch: got %s, want %s", root, describe(got), describe(exp))
		}
		return nil
	}
	if want.Kind == "ctor" && strings.Contains(want.Ctor, ".") {
		if got.Kind != "err" || got.ErrKind != want.Ctor {
			return fmt.Errorf("linked %s: expected %s, got %s", root, want.Ctor, describe(got))
		}
		exp, err := evSmall(want, env, ctx, fn.Name)
		if err != nil {
			return err
		}
		if valueHasFn(got) || valueHasFn(exp) {
			eq, detail, err := expectEq(exp, got)
			if err != nil {
				return err
			}
			if !eq {
				msg := fmt.Sprintf("linked %s: error payload mismatch: got %s, want %s", root, describe(got), describe(exp))
				if detail != "" {
					msg += ": " + detail
				}
				return errors.New(msg)
			}
			return nil
		}
		eq, err := vEq(got, exp)
		if err != nil || !eq {
			return fmt.Errorf("linked %s: error payload mismatch: got %s, want %s", root, describe(got), describe(exp))
		}
		return nil
	}
	return fmt.Errorf("linked %s: bad expectation shape", root)
}

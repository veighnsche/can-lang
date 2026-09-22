// canlc lsp: minimal Language Server over stdio.
//
// Speaks just enough JSON-RPC to drive editor squiggles: initialize,
// textDocument/didOpen, textDocument/didChange, shutdown/exit. Every
// keystroke re-runs the full diagnosis (parse, naming, uses resolution,
// exhaustiveness proof, signature tests) and publishes diagnostics.
//
// Run: canlc lsp [--stdio] [--baseline BASE.json]   (editors connect stdout/stdin
// with Content-Length framing). With --baseline, every diagnosis also
// runs revision-identity enforcement (a79): the same CAN6013 findings
// the CLI reports, as editor squiggles.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// Diag is one squiggle: 1-based Line, Sev "error"/"warning", human
// message, an optional 0-based UTF-16 [Start, End) token span, a stable
// Code from compiler/code.go, and the File (basename) it belongs to. When
// End <= Start the editor falls back to the whole line, so checks that
// cannot name a token stay line-precise without pretending otherwise.
type Diag struct {
	File       string
	Line       int
	Sev        string
	Msg        string
	Start, End int
	Code       string
	// Expected, Found, and Hint carry machine-actionable
	// payloads (a71, a61 item 1): what the rule wanted, what
	// it saw, and the suggested fix shape. Empty means the
	// code carries no payload; JSON omits empty fields so
	// payload-free lines stay byte-identical.
	Expected string
	Found    string
	Hint     string
}

func sevCode(sev string) int {
	if sev == "warning" {
		return 2
	}
	return 1
}

// diagLine finds the message's own line info, defaulting to the fallback.
func diagLine(err error, fallback int) int {
	var le *LineError
	if errors.As(err, &le) && le.Line > 0 {
		return le.Line
	}
	return fallback
}

// locateLine returns the 1-based line first containing sub, or fallback.
func locateLine(text, sub string, fallback int) int {
	for n, line := range strings.Split(text, "\n") {
		if strings.Contains(line, sub) {
			return n + 1
		}
	}
	return fallback
}

// u16len counts UTF-16 code units: editor columns are UTF-16, Go strings
// are bytes, and the two only agree on ASCII.
func u16len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// u16col converts a byte offset inside line to a UTF-16 column.
func u16col(line string, byteOff int) int {
	if byteOff < 0 {
		byteOff = 0
	}
	if byteOff > len(line) {
		byteOff = len(line)
	}
	return u16len(line[:byteOff])
}

// tokenSpan returns the 0-based UTF-16 [start, end) span of token on the
// given 1-based line. It prefers the applied form (name + "(" hits the
// call, not the mention) then the declared form (name + ":" hits the
// parameter, not a use), then the bare token. ok=false means the token
// is not on that line and the caller must fall back to whole-line.
func tokenSpan(text string, line int, token string) (start, end int, ok bool) {
	lines := strings.Split(text, "\n")
	if line < 1 || line > len(lines) || token == "" {
		return 0, 0, false
	}
	ln := lines[line-1]
	for _, cand := range []string{token + "(", token + ":", token} {
		if i := strings.Index(ln, cand); i >= 0 {
			if cand != token {
				return u16col(ln, i), u16col(ln, i+len(token)), true
			}
			return u16col(ln, i), u16col(ln, i+len(cand)), true
		}
	}
	return 0, 0, false
}

// spanDiag builds a Diag whose squiggle covers token on line, falling
// back to the whole line when the token is not there.
func spanDiag(text string, line int, sev, msg, token, code string) Diag {
	if s, e, ok := tokenSpan(text, line, token); ok {
		return Diag{Line: line, Sev: sev, Msg: msg, Start: s, End: e, Code: code}
	}
	return Diag{Line: line, Sev: sev, Msg: msg, Code: code}
}

// locateLineFrom is locateLine starting at 1-based fromLine: given-table
// entries repeat across retry-nested matches, so each match searches
// downward from itself instead of stealing the first match's row.
func locateLineFrom(text, sub string, fromLine, fallback int) int {
	lines := strings.Split(text, "\n")
	if fromLine < 1 {
		fromLine = 1
	}
	for n := fromLine - 1; n < len(lines); n++ {
		if strings.Contains(lines[n], sub) {
			return n + 1
		}
	}
	return locateLine(text, sub, fallback)
}

// needsSiblingWarn reports whether the sibling-parse warning may fire:
// solely alongside an unresolved-uses error. It keys on the typed
// diagnostic code (CodeUsesResolve), never on message wording, so
// rewording the human message cannot silently detach the suppression.
func needsSiblingWarn(world []Diag) bool {
	for _, d := range world {
		if d.Sev == "error" && d.Code == CodeUsesResolve {
			return true
		}
	}
	return false
}

// usesFallback extends the editor world with provider files for
// uses pins no loaded module satisfies, so an open file sees the
// same world a group CLI build would: the open file first, its
// same-dir siblings next, then provider files found elsewhere
// under the go.mod-anchored tree, appended in sorted path order
// and closed transitively (a loaded provider's own pins resolve
// the same way). Only well-formed name@rev pins trigger the
// search, and only files providing a needed name join — never a
// whole tree at once. Selection is by name; revision matching,
// self-pin rejection, and duplicate reporting stay exactly the
// buildWorld rules, so a pin that fails there fails here
// identically. Without a go.mod ancestor (temp dirs, go.mod-less
// projects) the world stays dir-confined, exactly as before.
func usesFallback(dir string, all []*Module, texts map[string]string) ([]*Module, map[string]string) {
	root := findModuleRoot(dir)
	if root == "" {
		return all, texts
	}
	index := indexRootProviders(root)
	if len(index) == 0 {
		return all, texts
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return all, texts
	}
	loaded := map[string]bool{}
	for _, m := range all {
		loaded[filepath.Clean(filepath.Join(absDir, m.ID))] = true
	}
	// loaded starts with the open file plus every sibling: their
	// keys are dir-relative, so absolutize against dir. A pin a
	// loaded module already satisfies (including a sibling's pin
	// the open file answers) never searches.
	for {
		provided := worldProvidedNames(all)
		var toLoad []string
		seen := map[string]bool{}
		for _, m := range all {
			for _, u := range m.Hdr["uses"] {
				if !strings.Contains(u, "@") {
					continue
				}
				base := pinRe.ReplaceAllString(u, "")
				if base == "" {
					continue
				}
				if owner, ok := provided[base]; ok && owner != m {
					continue
				}
				for _, cand := range index[base] {
					if !loaded[cand] && !seen[cand] {
						seen[cand] = true
						toLoad = append(toLoad, cand)
					}
				}
			}
		}
		if len(toLoad) == 0 {
			return all, texts
		}
		sort.Strings(toLoad)
		for _, abs := range toLoad {
			loaded[abs] = true
			data, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			key, err := filepath.Rel(absDir, abs)
			if err != nil {
				continue
			}
			mod, err := parseModuleText(key, string(data))
			if err != nil {
				continue
			}
			all = append(all, mod)
			texts[key] = string(data)
		}
	}
}

// worldProvidedNames mirrors the buildWorld provider map
// (first declaration wins) over the loaded modules: functions,
// consts, externs, types, variants, and brands all satisfy pins.
func worldProvidedNames(mods []*Module) map[string]*Module {
	p := map[string]*Module{}
	put := func(n string, m *Module) {
		if _, ok := p[n]; !ok {
			p[n] = m
		}
	}
	for _, m := range mods {
		for _, d := range m.Decls {
			switch d := d.(type) {
			case *FnDecl:
				put(d.Name, m)
			case *ConstDecl:
				put(d.Name, m)
			case *ExternDecl:
				put(d.Name, m)
			case *TypeDecl:
				put(d.Name, m)
			case *VariantDecl:
				put(d.Name, m)
			case *BrandDecl:
				put(d.Name, m)
			}
		}
	}
	return p
}

// findModuleRoot walks up from dir to the nearest ancestor holding
// go.mod. "" confines the world to dir: temp test dirs and
// go.mod-less projects diagnose exactly as before, deterministically
// (no sibling-test or scratch-file leakage through shared parents).
func findModuleRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for i := 0; i < 64; i++ {
		if st, err := os.Stat(filepath.Join(abs, "go.mod")); err == nil && !st.IsDir() {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
	return ""
}

// indexRootProviders maps every provided declaration name to the
// sorted absolute paths of the .can files declaring it under root.
// Dot directories and node_modules never scan; unparseable files
// cannot provide and are skipped silently (their pins, if any were
// needed, keep today's error).
func indexRootProviders(root string) map[string][]string {
	index := map[string][]string{}
	add := func(name, abs string) {
		index[name] = append(index[name], abs)
	}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".can") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		m, err := parseModuleText(path, string(data))
		if err != nil {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		for _, dd := range m.Decls {
			switch dd := dd.(type) {
			case *FnDecl:
				add(dd.Name, abs)
			case *ConstDecl:
				add(dd.Name, abs)
			case *ExternDecl:
				add(dd.Name, abs)
			case *TypeDecl:
				add(dd.Name, abs)
			case *VariantDecl:
				add(dd.Name, abs)
			case *BrandDecl:
				add(dd.Name, abs)
			}
		}
		return nil
	})
	for _, v := range index {
		sort.Strings(v)
	}
	return index
}

// diagnose runs every check on the open file (sibling .can files in dir
// provide the uses/provides world, extended by the usesFallback
// provider search above) and returns sorted diagnostics.
// Without a baseline no identity findings report: the editor stays
// quiet exactly as before.
func diagnose(dir, name, text string) []Diag {
	return diagnoseWith(dir, name, text, nil)
}

// diagnoseWith threads an accepted revision baseline through the
// same pipeline: when the world otherwise checks clean, identity
// drift against the baseline appends CAN6013 findings, mirroring
// the CLI's firstError gate so broken programs never gain drift
// noise on top of their real errors.
func diagnoseWith(dir, name, text string, base *RevisionBaseline) []Diag {
	var out []Diag
	var open *Module
	entries, _ := os.ReadDir(dir)
	files := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".can") && e.Name() != name {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	// all is ordered deterministically: the open file first (its
	// declarations win shadowing), then siblings by filename, so repeated
	// diagnoses of the same directory agree with each other.
	all := []*Module{}
	open, err := parseModuleText(name, text)
	if err != nil {
		return append(out, Diag{File: name, Line: diagLine(err, 1), Sev: "error", Msg: stripLinePrefix(err), Code: CodeParse})
	}
	all = append(all, open)
	// An unparseable sibling only matters when it could be the missing
	// provider: the warning fires solely alongside an unresolved uses
	// (CodeUsesResolve), so one broken demo never yellows its whole gallery.
	var sibWarn []Diag
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		m, err := parseModuleText(f, string(data))
		if err != nil {
			sibWarn = append(sibWarn, Diag{File: name, Line: 1, Sev: "warning",
				Msg:  fmt.Sprintf("sibling %s does not parse, uses-checks may over-report: %v", f, err),
				Code: CodeSiblingParse})
			continue
		}
		all = append(all, m)
	}
	texts := map[string]string{name: text}
	for _, f := range files {
		if data, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			texts[f] = string(data)
		}
	}
	all, texts = usesFallback(dir, all, texts)
	// G1 expansion, same position as the CLI pipeline: stamps
	// replace templates before any static check runs.
	if diags := expandGenerics(all, texts); len(diags) > 0 {
		out = append(out, diags...)
		for _, d := range diags {
			if d.Sev == "error" {
				sortDiags(out)
				return withFile(out, name)
			}
		}
	}
	out = append(out, checkStatic(open, text)...)
	prog, world := buildWorld(open, all, texts)
	out = append(out, world...)
	if needsSiblingWarn(world) {
		out = append(out, sibWarn...)
	}
	if len(world) > 0 {
		sortDiags(out)
		return withFile(out, name)
	}
	// a46 S2 barrier: same whole-program certification as the CLI, so
	// "no squiggles" and "compiles" cannot diverge on authority.
	out = append(out, certifyExports(all, prog, texts)...)
	// S2 slice plan barrier: same whole-program rule for asset bridges.
	out = append(out, certifyAssetBridge(all, prog, texts)...)

	for _, err := range verifyExhaustiveAll([]*Module{open}, prog) {
		out = append(out, proofDiag(text, err))
	}
	// World-level termination refusal (a11): cross-file cycles block
	// execution through the same gate, per the R10 world-error rule
	// (report per-line, suppress only execution-dependent checks).
	global := checkGlobalCycles(all, texts, prog)
	out = append(out, global...)
	recCycles := checkRecordCycles(all, texts)
	out = append(out, recCycles...)
	invokeCycles := checkInvokeCycles(all, texts, prog)
	out = append(out, invokeCycles...)
	// Provider bodies prepare silently before any open-file row
	// runs: cross-file invocation executes them, so their static
	// phase (including constructor binding) must complete even
	// though only the open file reports and executes.
	providerBlocked := prepareProviders(all, open, prog, texts)
	out = append(out, checkSem(open, text, prog, nil, hasErrors(global) || hasErrors(recCycles) || hasErrors(invokeCycles) || providerBlocked)...)
	if base != nil && !hasErrors(out) {
		out = append(out, CheckRevisionIdentity(prog, texts, base)...)
	}
	// a87: pinned weakening reports under the same clean-world rule,
	// after identity, so drift noise never stacks atop real errors.
	if base != nil && !hasErrors(out) {
		out = append(out, CheckPinnedRows(prog, texts, base)...)
	}
	// a82: verifier activation in the editor. Like identity
	// enforcement, proof findings append only when the world
	// otherwise checks clean, so broken programs never gain
	// proof noise atop real errors.
	if !hasErrors(out) {
		out = append(out, VerifyContracts(prog, texts)...)
	}
	// Strictness linter (can-idioms C6-C12, CAN3410-3416):
	// arm-reduction findings publish as errors on files the
	// compiler otherwise accepts, under the same clean-world
	// layering as identity, pinned rows, and contracts above —
	// broken programs show their real errors first.
	if !hasErrors(out) {
		out = append(out, lintDiagsFor(name, texts)...)
	}
	sortDiags(out)
	return withFile(out, name)
}

// withFile stamps every unattributed diagnostic with its owner file.
// buildWorld sets File at creation (it knows every module); everything
// else describes the open document.
func withFile(out []Diag, name string) []Diag {
	for i := range out {
		if out[i].File == "" {
			out[i].File = name
		}
	}
	return out
}

// checkStatic runs the checks that need no world: naming, header
// integrity, and decision-table shapes. Shared by the editor and the CLI
// so "no squiggles" and "compiles" cannot diverge.
func checkStatic(open *Module, text string) []Diag {
	var out []Diag
	out = append(out, checkNaming(open, text)...)
	out = append(out, checkModIntegrity(open, text)...)
	for _, d := range open.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		if len(fn.Tests) == 0 {
			// Stamps name their instance: a rowless instance is
			// missing evidence for one shape, not an untested
			// function, and the fix is a row pinning it.
			who := fn.Name
			if base, ok := open.GenericBase[fn.Name]; ok {
				who = describeStamp(fn.Name, base)
			}
			out = append(out, spanDiag(text, fn.Line, "error",
				fmt.Sprintf("%s ships no tests: every function needs its decision table", who), fn.Name, CodeMissingTests))
		}
		out = append(out, resolveTestArgs(fn, text)...)
		out = append(out, checkTestShapes(fn, text)...)
	}
	return withFile(out, open.File)
}

// checkSem runs the world-dependent checks: calls, given, emits, and
// unused items. onPass fires per passing test (the CLI prints PASS; the
// editor passes nil). Only call on a clean world. extBlocked carries a
// world-level refusal (a11: cross-file cycles) into the same
// prove-first gate as the per-module termination proofs.
// checkFnStatic runs one function's static phase: every checkSem
// check except execution and coverage. checkSem runs it for the
// open module; diagnose runs it silently for provider modules so
// cross-file invocation executes prepared bodies (b00 Q2d: the
// open module alone cannot authorize real execution of an
// unprepared provider body).
func checkFnStatic(fn *FnDecl, prog *Program, owner *Module, text string, localExtern, called map[string]bool) []Diag {
	var out []Diag
	for k := range calledFns(fn) {
		called[k] = true
	}
	out = append(out, checkCalls(fn, prog, localExtern, text)...)
	out = append(out, checkEagerScrutinee(fn, text)...)
	out = append(out, checkDecreases(fn, prog, text)...)
	out = append(out, checkEffects(fn, prog, text)...)
	out = append(out, checkGiven(fn, prog, text)...)
	// a92: checkTypes resolves positional construction by
	// mutation, so it runs before anything evaluates
	// (a18's sandbox): mutation-before-eval.
	out = append(out, checkTypes(fn, prog, text)...)
	out = append(out, checkEmits(fn, prog, text)...)
	out = append(out, checkScriptConsistency(fn, prog, text)...)
	out = append(out, checkUnusedParams(fn, text)...)
	out = append(out, checkConstRefs(fn, prog, owner, text)...)
	for k := range usedConsts(fn) {
		called[k] = true
	}
	return out
}

// prepareProviders runs the static phase for every non-open
// module. Findings stay silent (they surface when the provider
// file opens), but any error blocks open-file execution: a
// partial world may report source diagnostics but cannot execute
// an uncertified callback. True means blocked.
func prepareProviders(all []*Module, open *Module, prog *Program, texts map[string]string) bool {
	for _, m := range all {
		if m == open {
			continue
		}
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
			for _, dg := range checkFnStatic(fn, prog, m, texts[m.ID], localExtern, called) {
				if dg.Sev == "error" {
					return true
				}
			}
		}
	}
	return false
}

func checkSem(open *Module, text string, prog *Program, onPass func(fn, test string), extBlocked bool) []Diag {
	var out []Diag
	// Slice 1: resolve const-named patterns to literals before
	// any other per-function check, test run, or proof sees
	// them. Idempotent: rewritten literals are not revisited.
	out = append(out, elaborateConstPatterns(open, prog, text)...)
	called := map[string]bool{}
	localExtern := map[string]bool{}
	for _, d := range open.Decls {
		if ex, ok := d.(*ExternDecl); ok {
			localExtern[ex.Name] = true
		}
	}
	for _, d := range open.Decls {
		switch d := d.(type) {
		case *FnDecl:
			out = append(out, checkFnStatic(d, prog, open, text, localExtern, called)...)
		case *ExternDecl:
			out = append(out, checkExternSig(d, prog, text)...)
		case *BrandDecl:
			out = append(out, checkBrandDecl(d, prog, text)...)
		case *StateDecl:
			out = append(out, checkStateDecl(d, text)...)
		case *TypeDecl:
			out = append(out, checkDeclFields(d.Name, d.Fields, d.Line, prog, text, true)...)
		case *VariantDecl:
			for _, c := range d.Cases {
				out = append(out, checkDeclFields(qualifyCase(d.Name, c.Short), c.Fields, c.Line, prog, text, false)...)
			}
		case *ErrorDecl:
			out = append(out, checkDeclFields(d.Name, d.Fields, d.Line, prog, text, false)...)
		}
	}
	out = append(out, checkConstDecls(open, prog, text)...)
	for k := range prog.ConstUsed[open.ID] {
		called[k] = true
	}
	out = append(out, checkUnusedUses(open, text, called)...)
	out = append(out, checkLocalCycles(open, prog, text)...)
	// A local call cycle would hang test execution: prove acyclic
	// before running anything. Open termination proofs (bad, stale,
	// or unproven decreases, unguarded recursion, cross-file cycles),
	// ill-formed script evidence (outcome-only rows, malformed
	// outcomes, Ok claims the provider body contradicts), and unproven
	// authority (effects) block the same way:
	// the gate is prove-first, run-after.
	blocked := extBlocked
	for _, d := range out {
		if d.Sev == "error" && (d.Code == CodeLocalCycle ||
			d.Code == CodeBadDecreases || d.Code == CodeStaleDecreases ||
			d.Code == CodeNoDecrease || d.Code == CodeNoGuard ||
			d.Code == CodeNoExchange || d.Code == CodeBadStub || d.Code == CodeInconsistentScript ||
			d.Code == CodeUndeclaredEffect || d.Code == CodeStaleEffect) {
			blocked = true
		}
	}
	// Coverage is assessed over green tests module-wide: each test
	// runs with a private map, and only passing runs merge into the
	// shared map, so a helper arm counts caller flow-through without
	// letting a failing test fake coverage. Functions without tests
	// are already flagged elsewhere.
	shared := map[*Node]map[int]bool{}
	merge := func(src map[*Node]map[int]bool) {
		for n, arms := range src {
			dst := shared[n]
			if dst == nil {
				dst = map[int]bool{}
				shared[n] = dst
			}
			for i := range arms {
				dst[i] = true
			}
		}
	}
	failed := map[string]bool{}
	if !blocked {
		for _, d := range open.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			for _, t := range fn.Tests {
				tc := map[*Node]map[int]bool{}
				if err := runTest(fn, t, prog, tc); err != nil {
					var uce *UnknownCallError
					if errors.As(err, &uce) && calleeUnknown(prog, uce.Fname) {
						// a62: the row can only fail on the
						// unknown call checkCalls already
						// reported; suppress the CAN4200 but
						// mark the fn failed so coverage
						// stays silent too.
						failed[fn.Name] = true
						continue
					}
					failed[fn.Name] = true
					out = append(out, spanDiag(text, t.Line, "error",
						fmt.Sprintf("test %s fails: %s", t.Name, stripLinePrefix(err)), t.Name, CodeTestFailed))
				} else {
					merge(tc)
					if onPass != nil {
						onPass(fn.Name, t.Name)
					}
				}
			}
		}
	}
	for _, d := range open.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		if !failed[fn.Name] && !blocked && len(fn.Tests) > 0 {
			out = append(out, checkCoverage(fn, prog, text, shared)...)
		}
	}
	return withFile(out, open.File)
}

// checkCoverage enforces the test-per-arm law: every match arm must
// execute at least once across the function's decision-table run.
// Untaken arms are dead code or missing tests, both compile errors.
// The one exception is a checked identity relay: a bound error arm of
// a local call whose body is exactly the same-kind reconstruction
// with every payload field unchanged. Such an arm is a total,
// transparent re-raise — no behavior remains to witness — so it
// carries a structural certificate instead of an execution one. A
// relay-shaped arm that fails the check (wrong kind, dropped field,
// changed value) is an invalid certificate, not an uncovered arm.
func checkCoverage(fn *FnDecl, prog *Program, text string, cov map[*Node]map[int]bool) []Diag {
	var out []Diag
	for _, n := range matchNodes(fn.Body) {
		for i, a := range n.Arms {
			if cov[n][i] {
				continue
			}
			// First slot describes the arm.
			desc, tok := patDesc(a.Pats[0])
			if ok, reason := relayStatus(prog, fn.Name, n, n.Arms, i); ok {
				if reason == "" {
					continue
				}
				out = append(out, spanDiag(text, a.Line, "error",
					fmt.Sprintf("invalid identity relay %s in %s: %s", desc, fn.Name, reason), tok, CodeInvalidRelay))
				continue
			}
			out = append(out, spanDiag(text, a.Line, "error",
				fmt.Sprintf("no test takes %s in %s", desc, fn.Name), tok, CodeArmUntaken))
		}
	}
	return out
}

// relayStatus checks an untaken arm for the identity-relay shape: a
// bound error arm of a local call whose body rebuilds an error.
// It returns ok=false for anything else (the arm stays under the
// execution law). For a relay shape it returns ok=true with reason=""
// when the certificate verifies (same kind, complete fields, unchanged
// bound values, no other content), or ok=true with a reason naming the
// defect when the certificate is invalid. A shadowed arm is never
// certified: an earlier arm matching the same error kind already
// consumes every value this one could take, so structural evidence
// cannot substitute for the execution the shadowing removed.
func relayStatus(prog *Program, owner string, n *Node, arms []Arm, idx int) (bool, string) {
	if n.Kind != MatchCall {
		return false, ""
	}
	if localCallee(prog, owner, n.Scruts[0].Fname) == nil {
		return false, ""
	}
	a := arms[idx]
	pat := a.Pats[0]
	if pat.Kind != "variant" || pat.Name == "Ok" {
		return false, ""
	}
	for _, prev := range arms[:idx] {
		pp := prev.Pats[0]
		if (pp.Kind == "variant" || pp.Kind == "variantWild") && pp.Name == pat.Name {
			return false, ""
		}
	}
	fields, known := prog.Errors[pat.Name]
	if !known {
		return false, ""
	}
	rhs := a.Rhs
	if rhs == nil || rhs.IsMatch || rhs.Small == nil || rhs.Small.Kind != "ctor" {
		return false, ""
	}
	s := rhs.Small
	if !strings.Contains(s.Ctor, ".") {
		return false, ""
	}
	if s.Ctor != pat.Name {
		return true, fmt.Sprintf("reconstructs %s instead of %s", s.Ctor, pat.Name)
	}
	want := map[string]bool{}
	for _, f := range fields {
		want[f] = true
	}
	seen := map[string]bool{}
	for _, arg := range s.Args {
		if !want[arg.Name] {
			return true, fmt.Sprintf("rebuilds unexpected field %s", arg.Name)
		}
		seen[arg.Name] = true
		v := arg.V
		if v == nil || v.Kind != "ref" || len(v.Ref) != 2 || v.Ref[0] != pat.Var || v.Ref[1] != arg.Name {
			return true, fmt.Sprintf("field %s is not %s.%s", arg.Name, pat.Var, arg.Name)
		}
	}
	for _, f := range fields {
		if !seen[f] {
			return true, fmt.Sprintf("drops field %s", f)
		}
	}
	return true, ""
}

// patDesc renders a pattern the way its arm row reads it, plus the token
// the squiggle should cover on that row.
func patDesc(p Pattern) (desc, tok string) {
	switch p.Kind {
	case "variantWild":
		return "on " + p.Name + " _", p.Name
	case "variant":
		return "on " + p.Name + " " + p.Var, p.Name
	case "bool":
		if p.B {
			return "on true", "true"
		}
		return "on false", "false"
	case "str":
		// a66: an interpreted pattern's decoded value is not
		// searchable in source; locate its verbatim spelling.
		if strings.HasPrefix(p.Raw, `e"`) {
			return "on " + p.Raw, p.Raw
		}
		return "on " + strconv.Quote(p.Str), p.Str
	default:
		return "_", "_"
	}
}

// sortDiags orders squiggles file-top to file-bottom, errors before
// warnings on the same line, for a stable editor presentation.
func sortDiags(out []Diag) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Sev != out[j].Sev {
			return out[i].Sev < out[j].Sev
		}
		return out[i].Msg < out[j].Msg
	})
}

func stripLinePrefix(err error) string {
	var le *LineError
	if errors.As(err, &le) {
		return le.Err.Error()
	}
	return err.Error()
}

// proofDiag anchors an exhaustiveness-proof failure to a token. A stale
// arm names its kind on its own row, so the kind gets the squiggle; every
// other proof failure (missing arm, bool shape, bad pattern) lands on the
// match or arm keyword of its row. The code travels typed on the error
// from the producer (see proofErrf); only the squiggle token and the
// missing-arm hint still read the message, which is data, not identity.
func proofDiag(text string, err error) Diag {
	line := diagLine(err, 1)
	msg := stripLinePrefix(err)
	code, ok := proofCode(err)
	if !ok {
		code = CodeProofOther
	}
	var want, hint string
	switch code {
	case CodeStaleArm:
		// a63: the kind is the first field after the marker;
		// a nesting hint may follow it (see verifyExhaustiveAll).
		if i := strings.Index(msg, "stale match arm "); i >= 0 {
			kind := strings.TrimSpace(msg[i+len("stale match arm "):])
			if j := strings.IndexAny(kind, " ;"); j >= 0 {
				kind = kind[:j]
			}
			return spanDiag(text, line, "error", msg, kind, code)
		}
	case CodeMissingArm:
		if i := strings.Index(msg, "missing "); i >= 0 {
			want = strings.TrimSpace(msg[i+len("missing "):])
			hint = fmt.Sprintf("add an `on %s ...` arm covering the missing outcome", want)
		}
	}
	kw := "match"
	if lines := strings.Split(text, "\n"); line >= 1 && line <= len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[line-1]), "on ") {
			kw = "on"
		}
	}
	d := spanDiag(text, line, "error", msg, kw, code)
	d.Expected = want
	d.Hint = hint
	if found, ok := proofFound(err); ok {
		d.Found = found
	}
	return d
}

// ------------------------------------------------------------- protocol ---

type rpcMsg struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
}

type docID struct {
	URI string `json:"uri"`
}

func pathFromURI(uri string) string {
	parsed, err := url.ParseRequestURI(uri)
	if err != nil || parsed.Scheme != "file" {
		return ""
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return ""
	}
	return path
}

func uriFromPath(path string) string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return "file://" + strings.Join(segments, "/")
}

func writeFrame(w *bufio.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body))
	w.Write(body)
	return w.Flush()
}

// ---------------------------------------------------------------- server ---

// The I41 server keeps the stdio JSON-RPC transport and replaces every
// semantic hook: each keystroke diagnoses an in-memory overlay snapshot
// through the current parse, resolve, and check pipeline, and definition
// requests resolve through file, package, prelude, and import scopes. No
// request builds, runs, asserts, dials out, queries, reads the
// environment, emits files, or mutates registries; the --baseline
// execution hook is gone, and predecessor spellings diagnose as ordinary
// current-syntax errors.
type lspDoc struct {
	path    string
	text    string
	version int64
}

type lspServer struct {
	docs      map[string]*lspDoc
	overlay   *project.Overlay
	published map[string]bool
}

func newLSPServer() *lspServer {
	return &lspServer{docs: map[string]*lspDoc{}, overlay: project.NewOverlay(), published: map[string]bool{}}
}

func parseLSPArgs(argv []string) error {
	for _, arg := range argv {
		if arg == "--baseline" || strings.HasPrefix(arg, "--baseline=") {
			return fmt.Errorf("canlc lsp: --baseline was retired with the baseline-veto handshake; the server now reports live diagnostics")
		}
	}
	fs := flag.NewFlagSet("canlc lsp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("stdio", false, "stdio transport marker from LSP clients (ignored)")
	if err := fs.Parse(argv); err != nil {
		return fmt.Errorf("usage: canlc lsp [--stdio]")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: canlc lsp [--stdio]")
	}
	return nil
}

func runLSP(argv []string) int {
	if err := parseLSPArgs(argv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	serveLSP(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout))
	return 0
}

func serveLSP(in *bufio.Reader, out *bufio.Writer) {
	server := newLSPServer()
	respond := func(id *json.RawMessage, result any) {
		writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	}
	respondErr := func(id *json.RawMessage, code int, message string) {
		writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
	}
	for {
		var length int
		for {
			line, err := in.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			}
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(in, body); err != nil {
			return
		}
		var msg rpcMsg
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}
		switch msg.Method {
		case "initialize":
			respond(msg.ID, map[string]any{"capabilities": map[string]any{"textDocumentSync": 1, "definitionProvider": true}})
		case "initialized", "$/cancelRequest":
		case "textDocument/didOpen":
			var p struct {
				TextDocument struct {
					URI     string `json:"uri"`
					Text    string `json:"text"`
					Version int64  `json:"version"`
				} `json:"textDocument"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				continue
			}
			server.open(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
			server.diagnose(out, p.TextDocument.URI)
		case "textDocument/didChange":
			var p struct {
				TextDocument struct {
					URI     string `json:"uri"`
					Version int64  `json:"version"`
				} `json:"textDocument"`
				Changes []struct {
					Text string `json:"text"`
				} `json:"contentChanges"`
			}
			if json.Unmarshal(msg.Params, &p) != nil || len(p.Changes) == 0 {
				continue
			}
			server.change(p.TextDocument.URI, p.Changes[len(p.Changes)-1].Text, p.TextDocument.Version)
			server.diagnose(out, p.TextDocument.URI)
		case "textDocument/didClose":
			var p struct {
				TextDocument docID `json:"textDocument"`
			}
			if json.Unmarshal(msg.Params, &p) == nil {
				server.close(out, p.TextDocument.URI)
			}
		case "textDocument/definition":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid definition params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.definition(p.TextDocument.URI, p.Position.Line, p.Position.Character))
			}
		case "shutdown":
			respond(msg.ID, nil)
		case "exit":
			return
		default:
			if msg.ID != nil {
				respondErr(msg.ID, -32601, "unknown method "+msg.Method)
			}
		}
	}
}

func (s *lspServer) open(uri, text string, version int64) {
	path := pathFromURI(uri)
	s.docs[uri] = &lspDoc{path: path, text: text, version: version}
	if path == "" {
		return
	}
	if err := s.overlay.Set(path, version, text); err != nil {
		s.overlay.Clear(path)
	}
}

func (s *lspServer) change(uri, text string, version int64) {
	doc, ok := s.docs[uri]
	if !ok {
		doc = &lspDoc{path: pathFromURI(uri)}
		s.docs[uri] = doc
	}
	doc.text, doc.version = text, version
	if doc.path == "" {
		return
	}
	if err := s.overlay.Set(doc.path, version, text); err != nil {
		s.overlay.Clear(doc.path)
	}
}

func (s *lspServer) close(out *bufio.Writer, uri string) {
	doc, ok := s.docs[uri]
	if !ok {
		return
	}
	if doc.path != "" {
		s.overlay.Clear(doc.path)
	}
	delete(s.docs, uri)
	delete(s.published, uri)
	publishBridgeDiagnostics(out, uri, nil, nil)
}

func (s *lspServer) diagnose(out *bufio.Writer, uri string) {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		publishBridgeDiagnostics(out, uri, nil, nil)
		return
	}
	root := discoverRoot(doc.path)
	snapshot, err := driver.CheckSnapshot(root, doc.path, s.overlay)
	if err != nil {
		publishBridgeDiagnostics(out, uri, &doc.version, nil)
		return
	}
	byURI := map[string][]driver.Diagnostic{}
	for _, diagnostic := range snapshot.Diagnostics {
		uri := uriFromPath(diagnostic.File)
		byURI[uri] = append(byURI[uri], diagnostic)
	}
	for openURI, openDoc := range s.docs {
		version := openDoc.version
		publishBridgeDiagnostics(out, openURI, &version, byURI[openURI])
		if len(byURI[openURI]) > 0 {
			s.published[openURI] = true
		} else {
			delete(s.published, openURI)
		}
	}
	for diagnosedURI, diags := range byURI {
		if _, open := s.docs[diagnosedURI]; open {
			continue
		}
		publishBridgeDiagnostics(out, diagnosedURI, nil, diags)
		s.published[diagnosedURI] = true
	}
	for publishedURI := range s.published {
		if _, still := byURI[publishedURI]; still {
			continue
		}
		if _, open := s.docs[publishedURI]; open {
			continue
		}
		publishBridgeDiagnostics(out, publishedURI, nil, nil)
		delete(s.published, publishedURI)
	}
}

func (s *lspServer) definition(uri string, line, character int) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	location, ok, err := driver.Definition(snapshot, doc.path, line, character)
	if err != nil || !ok {
		return nil
	}
	return map[string]any{
		"uri": uriFromPath(location.File),
		"range": map[string]any{
			"start": map[string]any{"line": location.Line, "character": location.Start},
			"end":   map[string]any{"line": location.Line, "character": location.End},
		},
	}
}

// discoverRoot walks up from a source file to the enclosing project
// manifest. Without one, the file's own directory becomes the diagnosis
// root so the bridge reports the missing project instead of silence.
func discoverRoot(path string) string {
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		abs = path
	}
	dir := abs
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for i := 0; i < 256; i++ {
		if info, err := os.Lstat(filepath.Join(dir, "can.project.json")); err == nil && info.Mode().IsRegular() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(abs)
}

func publishBridgeDiagnostics(out *bufio.Writer, uri string, version *int64, diags []driver.Diagnostic) error {
	items := []any{}
	for _, d := range diags {
		line := d.Line
		if line < 0 {
			line = 0
		}
		start, end := d.Start, d.End
		if start < 0 {
			start = 0
		}
		if end < start {
			end = start
		}
		item := map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": line, "character": start},
				"end":   map[string]any{"line": line, "character": end},
			},
			"severity": 1,
			"source":   "canlc",
			"message":  d.Message,
		}
		if d.Code != "" {
			item["code"] = d.Code
		}
		items = append(items, item)
	}
	params := map[string]any{"uri": uri, "diagnostics": items}
	if version != nil {
		params["version"] = *version
	}
	return writeFrame(out, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params":  params,
	})
}

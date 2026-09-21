package main

// a81: proof obligations + z3 boundary. Admitted functions prove
// per-exit obligations (Requires ∧ Path ⟹ Ensures) and call-site
// preconditions through callee summaries verified in this run,
// with z3 deciding QF_LIA queries. Only unsatisfiable negations
// discharge; sat is a counterexample, anything else inconclusive.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// smt is a minimal SMT-LIB term: variables, numerals, Booleans,
// and applications. Structural substitution (no string surgery)
// instantiates callee contracts at call sites.
type smt struct {
	op   string // "var", "num", "bool", "app"
	name string // var name (plain), numeral digits, true/false
	app  string // and or not ite = distinct < <= > >= + - *
	args []*smt
}

func svar(name string) *smt   { return &smt{op: "var", name: name} }
func snum(digits string) *smt { return &smt{op: "num", name: digits} }
func sbool(b bool) *smt {
	return &smt{op: "bool", name: map[bool]string{true: "true", false: "false"}[b]}
}
func sapp(app string, args ...*smt) *smt { return &smt{op: "app", app: app, args: args} }

func smtNum(v string) *smt {
	// SMT-LIB numerals are non-negative: negatives parenthesize.
	if strings.HasPrefix(v, "-") {
		return sapp("-", snum(strings.TrimPrefix(v, "-")))
	}
	return snum(v)
}

// String renders SMT-LIB. Variables quote (CAN names carry dots);
// numerals render bare.
func (s *smt) String() string {
	switch s.op {
	case "var":
		return "|" + s.name + "|"
	case "num", "bool":
		return s.name
	}
	parts := make([]string, 0, len(s.args)+1)
	parts = append(parts, s.app)
	for _, a := range s.args {
		parts = append(parts, a.String())
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// subst replaces variables by name, structurally.
func subst(bind map[string]*smt, s *smt) *smt {
	if s.op == "var" {
		if v, ok := bind[s.name]; ok {
			return v
		}
		return s
	}
	if s.op != "app" {
		return s
	}
	out := &smt{op: "app", app: s.app}
	for _, a := range s.args {
		out.args = append(out.args, subst(bind, a))
	}
	return out
}

func sand(ts ...*smt) *smt {
	var flat []*smt
	for _, t := range ts {
		if t.op == "bool" && t.name == "true" {
			continue
		}
		flat = append(flat, t)
	}
	if len(flat) == 0 {
		return sbool(true)
	}
	if len(flat) == 1 {
		return flat[0]
	}
	return sapp("and", flat...)
}

// symVal is a symbolic value: an int/bool leaf term, or a record
// decomposed into field values. Projections resolve structurally,
// so record equality flattens into field equalities.
type symVal struct {
	term   *smt
	fields map[string]*symVal
}

func leafVal(t *smt) *symVal { return &symVal{term: t} }

// prover generates and discharges obligations for one function.
// It runs only after admission accepts the function with the
// current verified set, so shapes here are trusted-but-verified:
// every converter failure fails closed to inconclusive.
type prover struct {
	a      *admission
	fn     *FnDecl
	name   string
	text   string
	decls  map[string]string // smt var name -> "Int"/"Bool"
	inputs []string          // param leaf vars, for countermodels
	seq    int               // fresh-variable counter
	out    []Diag
}

func newProver(a *admission, fn *FnDecl, text string) *prover {
	name, _ := declNameLine(fn)
	return &prover{a: a, fn: fn, name: name, text: text, decls: map[string]string{}}
}

func (p *prover) fresh(base string) string {
	p.seq++
	return fmt.Sprintf("call%d.%s", p.seq, base)
}

// declare binds a param or payload leaf, recording its sort.
func (p *prover) declare(name, sort string) *smt {
	p.decls[name] = sort
	return svar(name)
}

// bindSort materializes a symbolic value for a proof sort: leaves
// become fresh variables, records decompose through fields.
func (p *prover) bindSort(prefix string, st admitSort) *symVal {
	switch st.kind {
	case "int":
		return leafVal(p.declare(prefix, "Int"))
	case "bool":
		return leafVal(p.declare(prefix, "Bool"))
	case "rec":
		v := &symVal{fields: map[string]*symVal{}}
		names := make([]string, 0, len(st.fields))
		for n := range st.fields {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			v.fields[n] = p.bindSort(prefix+"."+n, st.fields[n])
		}
		return v
	}
	return nil
}

// paramEnv binds every parameter to fresh symbolic inputs.
func (p *prover) paramEnv() map[string]*symVal {
	env := map[string]*symVal{}
	for _, pr := range p.fn.Params {
		v := p.bindSort(pr[0], p.a.sortOfType(pr[1]))
		if v == nil {
			return nil
		}
		env[pr[0]] = v
		collectLeaves(pr[0], v, &p.inputs)
	}
	return env
}

func collectLeaves(prefix string, v *symVal, out *[]string) {
	if v.fields == nil {
		*out = append(*out, prefix)
		return
	}
	names := make([]string, 0, len(v.fields))
	for n := range v.fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		collectLeaves(prefix+"."+n, v.fields[n], out)
	}
}

// resolve follows a reference path through the environment.
func resolve(path []string, env map[string]*symVal) *symVal {
	cur, ok := env[path[0]]
	if !ok {
		return nil
	}
	for _, seg := range path[1:] {
		if cur.fields == nil {
			return nil
		}
		cur, ok = cur.fields[seg]
		if !ok {
			return nil
		}
	}
	return cur
}

// smallVal converts an admitted term to SMT. Records convert only
// through flattening at comparisons; elsewhere they need a leaf.
func (p *prover) smallVal(s *Small, env map[string]*symVal) (*smt, bool) {
	if s == nil {
		return nil, false
	}
	switch s.Kind {
	case "int":
		if s.Num != nil {
			return smtNum(s.Num.String()), true
		}
		return nil, false
	case "bool":
		return sbool(s.B), true
	case "ref":
		v := resolve(s.Ref, env)
		if v == nil || v.term == nil {
			return nil, false
		}
		return v.term, true
	case "binop":
		return p.binopVal(s, env)
	}
	return nil, false
}

func (p *prover) binopVal(s *Small, env map[string]*symVal) (*smt, bool) {
	if s.Op == "==" || s.Op == "!=" {
		eq, ok := p.eqVal(s.L, s.R, env)
		if !ok {
			return nil, false
		}
		if s.Op == "!=" {
			return sapp("not", eq), true
		}
		return eq, true
	}
	l, ok := p.smallVal(s.L, env)
	if !ok {
		return nil, false
	}
	r, ok := p.smallVal(s.R, env)
	if !ok {
		return nil, false
	}
	switch s.Op {
	case "+", "-", "*":
		return sapp(s.Op, l, r), true
	case "<", "<=", ">", ">=":
		return sapp(s.Op, l, r), true
	}
	return nil, false
}

// eqVal flattens record equality into field equalities; leaves
// compare directly.
func (p *prover) eqVal(l, r *Small, env map[string]*symVal) (*smt, bool) {
	lv := p.smallBinding(l, env)
	rv := p.smallBinding(r, env)
	if lv == nil || rv == nil {
		return nil, false
	}
	conj := eqFlat(lv, rv)
	if conj == nil {
		return nil, false
	}
	return sand(conj...), true
}

func eqFlat(l, r *symVal) []*smt {
	if l.fields == nil && r.fields == nil {
		return []*smt{sapp("=", l.term, r.term)}
	}
	if l.fields == nil || r.fields == nil {
		return nil
	}
	var out []*smt
	for name, lf := range l.fields {
		rf, ok := r.fields[name]
		if !ok {
			return nil
		}
		sub := eqFlat(lf, rf)
		if sub == nil {
			return nil
		}
		out = append(out, sub...)
	}
	return out
}

// smallBinding converts a term to a symbolic value, keeping record
// structure for equality and call-argument instantiation.
func (p *prover) smallBinding(s *Small, env map[string]*symVal) *symVal {
	if s == nil {
		return nil
	}
	switch s.Kind {
	case "int":
		if s.Num == nil {
			return nil
		}
		return leafVal(smtNum(s.Num.String()))
	case "bool":
		return leafVal(sbool(s.B))
	case "ref":
		return resolve(s.Ref, env)
	case "proj":
		base := p.smallBinding(s.L, env)
		if base == nil {
			return nil
		}
		return base.fields[s.Field]
	case "binop":
		if v, ok := p.smallVal(s, env); ok {
			return leafVal(v)
		}
		return nil
	case "ctor":
		v := &symVal{fields: map[string]*symVal{}}
		for _, arg := range s.Args {
			if arg.V == nil {
				return nil
			}
			b := p.smallBinding(arg.V, env)
			if b == nil {
				return nil
			}
			v.fields[arg.Name] = b
		}
		return v
	}
	return nil
}

// matchVal converts a Boolean contract match to ite.
func (p *prover) matchVal(node *Node, env map[string]*symVal) (*smt, bool) {
	if node == nil || !node.IsMatch || node.Kind != MatchValue || len(node.Scruts) != 1 {
		return nil, false
	}
	scrut, ok := p.smallVal(node.Scruts[0], env)
	if !ok {
		return nil, false
	}
	var onTrue, onFalse *smt
	for _, armNode := range node.Arms {
		if len(armNode.Pats) != 1 || armNode.Pats[0].Kind != "bool" {
			return nil, false
		}
		var v *smt
		rhs := armNode.Rhs
		if rhs == nil {
			return nil, false
		}
		if rhs.IsMatch {
			v, ok = p.matchVal(rhs, env)
		} else {
			v, ok = p.smallVal(rhs.Small, env)
		}
		if !ok {
			return nil, false
		}
		if armNode.Pats[0].B {
			onTrue = v
		} else {
			onFalse = v
		}
	}
	if onTrue == nil || onFalse == nil {
		return nil, false
	}
	return sapp("ite", scrut, onTrue, onFalse), true
}

// predVal converts one contract predicate row or match block.
func (p *prover) predVal(s *Small, env map[string]*symVal) (*smt, bool) {
	return p.smallVal(s, env)
}

// obligation is one unsatisfiability query: Requires ∧ Path ∧
// ¬Claim must be unsat. Exits claim their outcome predicate;
// call sites claim the callee precondition.
type obligation struct {
	fn         string
	kind       string // "exit on Ok" or "requires of g"
	line       int
	req        []*smt
	path       []*smt
	neg        *smt
	inputs     []string
	viaSummary bool
	want       string // described claim, for Expected
}

// execState is one symbolic path: accumulated facts, the current
// environment, and whether callee summaries were assumed.
type execState struct {
	asserts    []*smt
	env        map[string]*symVal
	viaSummary bool
	line       int
}

// gen builds every obligation for the function. False means
// construction itself failed: the caller fails closed to
// inconclusive rather than trusting a partial set.
func (p *prover) gen() ([]obligation, bool) {
	env := p.paramEnv()
	if env == nil {
		return nil, false
	}
	var req []*smt
	for _, r := range p.fn.Requires {
		v, ok := p.smallVal(r, env)
		if !ok {
			return nil, false
		}
		req = append(req, v)
	}
	var out []obligation
	st := &execState{env: env, line: p.fn.Line}
	if !p.genBody(p.fn.Body, st, req, &out) {
		return nil, false
	}
	return out, true
}

func (p *prover) genBody(node *Node, st *execState, req []*smt, out *[]obligation) bool {
	if node == nil {
		return false
	}
	line := st.line
	if node.Line > 0 {
		line = node.Line
	}
	if node.IsMatch {
		if node.Kind == MatchCall {
			return p.genCall(node, st, req, out, line)
		}
		if len(node.Scruts) != 1 {
			return false
		}
		scrut, ok := p.smallVal(node.Scruts[0], st.env)
		if !ok {
			return false
		}
		// Boolean arms keep the exact complete-case encoding.
		// Integer, range, and wildcard arms encode as value
		// constraints; anything else declines. Admission takes
		// all three (int scrutinees are admitted), so the
		// prover discharges them instead of failing every
		// integer case analysis as inconclusive.
		intCase := true
		for _, armNode := range node.Arms {
			if len(armNode.Pats) != 1 {
				return false
			}
			if armNode.Pats[0].Kind == "bool" {
				intCase = false
			}
		}
		if !intCase {
			for _, armNode := range node.Arms {
				if armNode.Pats[0].Kind != "bool" {
					return false
				}
			}
		}
		for _, armNode := range node.Arms {
			cond := scrutCond(scrut, armNode.Pats[0].B)
			if intCase {
				var ok bool
				cond, ok = armIntCond(scrut, armNode.Pats[0])
				if !ok {
					return false
				}
			}
			branch := &execState{env: st.env, viaSummary: st.viaSummary, line: line}
			branch.asserts = append(append([]*smt{}, st.asserts...), cond)
			if armNode.Line > 0 {
				branch.line = armNode.Line
			}
			if !p.genBody(armNode.Rhs, branch, req, out) {
				return false
			}
		}
		return true
	}
	if node.Small == nil || node.Small.Kind != "ctor" {
		return false
	}
	return p.genExit(node.Small, st, req, out, line)
}

// armIntCond encodes one integer-case arm guard: int literals
// as equalities, ranges as closed intervals, wildcards as true.
// Wild over-approximates (it also covers inputs earlier arms
// take): sound for universal postconditions, at most
// inconclusive where the solver cannot use it. Or-alternatives
// and unresolved bounds decline.
func armIntCond(scrut *smt, pat Pattern) (*smt, bool) {
	switch pat.Kind {
	case "int":
		if pat.Num == nil {
			return nil, false
		}
		return sapp("=", scrut, smtNum(pat.Num.String())), true
	case "range":
		if pat.Num == nil || pat.Hi == nil {
			return nil, false
		}
		lo, hi := pat.Num.String(), pat.Hi.String()
		return sand(sapp("<=", smtNum(lo), scrut), sapp("<=", scrut, smtNum(hi))), true
	case "wild":
		return sbool(true), true
	}
	return nil, false
}

func scrutCond(scrut *smt, takeTrue bool) *smt {
	if takeTrue {
		return scrut
	}
	return sapp("not", scrut)
}

// genExit proves one actual exit against its outcome predicate.
func (p *prover) genExit(ctor *Small, st *execState, req []*smt, out *[]obligation, line int) bool {
	var arm *ContractArm
	for i := range p.fn.Ensures {
		if p.fn.Ensures[i].Outcome == ctor.Ctor {
			arm = &p.fn.Ensures[i]
			break
		}
	}
	if arm == nil {
		return false
	}
	payload := map[string]*symVal{}
	for _, arg := range wholeOkArgs(ctor, recordShapes(p.a.prog.Modules)) {
		b := p.smallBinding(arg.V, st.env)
		if b == nil {
			return false
		}
		payload[arg.Name] = b
	}
	env := copySymEnv(st.env)
	env[arm.Bind] = &symVal{fields: payload}
	var claims []*smt
	for _, pr := range arm.Preds {
		v, ok := p.predVal(pr, env)
		if !ok {
			return false
		}
		claims = append(claims, v)
	}
	for _, mt := range arm.Matches {
		v, ok := p.matchVal(mt, env)
		if !ok {
			return false
		}
		claims = append(claims, v)
	}
	*out = append(*out, obligation{
		fn: p.name, kind: "exit on " + arm.Outcome, line: line,
		req: req, path: append([]*smt{}, st.asserts...),
		neg:        sapp("not", sand(claims...)),
		inputs:     append([]string{}, p.inputs...),
		viaSummary: st.viaSummary,
		want:       "on " + arm.Outcome + ": " + descClaims(arm),
	})
	return true
}

// genCall proves the callee precondition at the site, then extends
// every outcome arm with the verified summary.
func (p *prover) genCall(node *Node, st *execState, req []*smt, out *[]obligation, line int) bool {
	call := node.Scruts[0]
	callee, ok := p.a.fns[call.Fname]
	if !ok || !isContracted(callee) || !p.a.verified[call.Fname] {
		return false
	}
	// Actual arguments instantiate the callee formals, records
	// included: bindings carry structure, not strings. Binding
	// follows the one shared rule (bindSlots): positional args
	// bind by source index, named args bind by name, and the two
	// mix freely — judging the whole call from Args[0] alone
	// misbinds out-of-order mixed calls (a later named arg would
	// land in the wrong formal), proving obligations about the
	// wrong values. A bind failure fails closed to inconclusive:
	// the checker reports the malformed call itself.
	formals := map[string]*symVal{}
	slots, berr := bindSlots(call.Fname, call.Args, callee.Params)
	if berr != nil {
		return false
	}
	bound := make([]*Small, len(callee.Params))
	for i, a := range call.Args {
		bound[slots[i]] = a.V
	}
	for i, pr := range callee.Params {
		arg := bound[i]
		if arg == nil {
			return false
		}
		b := p.smallBinding(arg, st.env)
		if b == nil {
			return false
		}
		formals[pr[0]] = b
	}
	calleeEnv := formals
	var preClaims []*smt
	for _, r := range callee.Requires {
		v, ok := p.smallVal(r, calleeEnv)
		if !ok {
			return false
		}
		preClaims = append(preClaims, v)
	}
	*out = append(*out, obligation{
		fn: p.name, kind: "requires of " + call.Fname, line: line,
		req: req, path: append([]*smt{}, st.asserts...),
		neg:        sapp("not", sand(preClaims...)),
		inputs:     append([]string{}, p.inputs...),
		viaSummary: st.viaSummary,
		want:       call.Fname + " requires: " + descSmalls(callee.Requires),
	})
	for _, armNode := range node.Arms {
		if len(armNode.Pats) != 1 || (armNode.Pats[0].Kind != "variant" && armNode.Pats[0].Kind != "variantWild") {
			return false
		}
		outcome := armNode.Pats[0].Name
		var carm *ContractArm
		for i := range callee.Ensures {
			if callee.Ensures[i].Outcome == outcome {
				carm = &callee.Ensures[i]
				break
			}
		}
		if carm == nil {
			return false
		}
		var shape admitSort
		if outcome == "Ok" {
			shape = p.a.successSort(callee.Ret)
		} else if es, ok := p.a.errorShape(outcome); ok {
			shape = es
		} else {
			return false
		}
		fresh := p.bindSort(p.fresh(callee.Name), shape)
		if fresh == nil {
			return false
		}
		sumEnv := copySymEnv(calleeEnv)
		sumEnv[carm.Bind] = fresh
		branch := &execState{env: copySymEnv(st.env), viaSummary: true, line: line}
		branch.asserts = append(append([]*smt{}, st.asserts...), summaryAsserts(p, carm, sumEnv)...)
		if armNode.Pats[0].Var != "" {
			bound := fresh
			if outcome == "Ok" && len(armNode.Pats[0].TypeArgs) == 1 && p.a.types[callee.Ret] == nil {
				bound = fresh.fields["value"]
			}
			branch.env[armNode.Pats[0].Var] = bound
		}
		if armNode.Line > 0 {
			branch.line = armNode.Line
		}
		if !p.genBody(armNode.Rhs, branch, req, out) {
			return false
		}
	}
	return true
}

func summaryAsserts(p *prover, arm *ContractArm, env map[string]*symVal) []*smt {
	var out []*smt
	for _, pr := range arm.Preds {
		v, ok := p.predVal(pr, env)
		if !ok {
			return nil
		}
		out = append(out, v)
	}
	for _, mt := range arm.Matches {
		v, ok := p.matchVal(mt, env)
		if !ok {
			return nil
		}
		out = append(out, v)
	}
	return out
}

func copySymEnv(env map[string]*symVal) map[string]*symVal {
	out := map[string]*symVal{}
	for k, v := range env {
		out[k] = v
	}
	return out
}

func descSmalls(ts []*Small) string {
	parts := make([]string, 0, len(ts))
	for _, t := range ts {
		parts = append(parts, describeSmall(t))
	}
	if len(parts) == 0 {
		return "true"
	}
	return strings.Join(parts, " and ")
}

func descClaims(arm *ContractArm) string {
	d := descSmalls(arm.Preds)
	for range arm.Matches {
		if d != "" {
			d += " and "
		}
		d += "boolean case"
	}
	if d == "" {
		d = "true"
	}
	return d
}

// script renders one obligation as a QF_LIA query: every input
// and payload variable declared, entry and path facts asserted,
// the negated claim last. Unsat discharges; sat refutes.
func (p *prover) script(ob obligation) string {
	var b strings.Builder
	b.WriteString("(set-logic QF_LIA)\n")
	names := make([]string, 0, len(p.decls))
	for n := range p.decls {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(&b, "(declare-const |%s| %s)\n", n, p.decls[n])
	}
	for _, r := range ob.req {
		fmt.Fprintf(&b, "(assert %s)\n", r.String())
	}
	for _, a := range ob.path {
		fmt.Fprintf(&b, "(assert %s)\n", a.String())
	}
	if ob.neg != nil {
		fmt.Fprintf(&b, "(assert %s)\n", ob.neg.String())
	}
	b.WriteString("(check-sat)\n")
	return b.String()
}

// smtTimeout bounds one solver query: exhaustion rejects as
// inconclusive, never as acceptance.
const smtTimeout = 30 * time.Second

// smtBinary locates the solver. CANLC_Z3 overrides PATH for tests;
// otherwise z3 must resolve normally.
func smtBinary() string {
	if p := os.Getenv("CANLC_Z3"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	}
	if p, err := exec.LookPath("z3"); err == nil {
		return p
	}
	return ""
}

// runSMTQuery sends one script, returning stdout.
func runSMTQuery(script, extra string) (string, error) {
	bin := smtBinary()
	if bin == "" {
		return "", fmt.Errorf("no solver binary: install z3 or set CANLC_Z3")
	}
	ctx, cancel := context.WithTimeout(context.Background(), smtTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-in")
	cmd.Stdin = strings.NewReader(script + extra)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("solver timeout after %s", smtTimeout)
		}
		return "", fmt.Errorf("solver failure: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// parseSMTResult classifies exactly sat, unsat, or unknown from
// the first output line. Anything else is a tooling failure,
// never a proof outcome.
func parseSMTResult(out string) (string, error) {
	line := out
	if i := strings.Index(line, "\n"); i >= 0 {
		line = line[:i]
	}
	switch strings.TrimSpace(line) {
	case "sat", "unsat", "unknown":
		return strings.TrimSpace(line), nil
	}
	return "", fmt.Errorf("unrecognized solver response %q", strings.TrimSpace(line))
}

// parseSMTModel reads (get-value) output back into input values:
// ((|left| 0) (|right| (- 3)) (|flag| false)) maps names to
// mathematical-integer or Boolean spellings. Only requested names
// are returned.
func parseSMTModel(out string, want []string) map[string]string {
	toks := tokenizeSMT(out)
	got := map[string]string{}
	for i := 0; i < toks.length(); {
		// Each pair is ( name value ). A bare (( opens the
		// outer wrapper list, not a pair.
		if toks.at(i) != "(" {
			i++
			continue
		}
		if toks.at(i+1) == "(" {
			i++
			continue
		}
		i++
		if i >= toks.length() {
			break
		}
		name := toks.unquote(i)
		i++
		val, next := toks.valueAt(i)
		if name == "" || val == "" {
			i = next
			continue
		}
		got[name] = val
		i = next
		// Consume the pair's close paren.
		if toks.at(i) == ")" {
			i++
		}
	}
	out2 := map[string]string{}
	for _, w := range want {
		if v, ok := got[w]; ok {
			out2[w] = v
		}
	}
	return out2
}

// smtToks is a tokenized s-expression: parens, quoted |names|
// (kept verbatim), and bare atoms.
type smtToks []string

func tokenizeSMT(out string) smtToks {
	var toks smtToks
	for i := 0; i < len(out); {
		switch c := out[i]; {
		case c == ' ' || c == '\n' || c == '\t' || c == '\r':
			i++
		case c == '(' || c == ')':
			toks = append(toks, string(c))
			i++
		case c == '|':
			j := strings.Index(out[i+1:], "|")
			if j < 0 {
				return toks
			}
			toks = append(toks, "|"+out[i+1:i+1+j]+"|")
			i += j + 2
		default:
			j := i
			for j < len(out) && !strings.ContainsRune(" \n\t\r()", rune(out[j])) {
				j++
			}
			toks = append(toks, out[i:j])
			i = j
		}
	}
	return toks
}

func (t smtToks) length() int { return len(t) }

func (t smtToks) at(i int) string {
	if i < 0 || i >= len(t) {
		return ""
	}
	return t[i]
}

// unquote strips one pair of pipes from a quoted name.
func (t smtToks) unquote(i int) string {
	s := t.at(i)
	if len(s) >= 2 && strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|") {
		return s[1 : len(s)-1]
	}
	return s
}

// valueAt reads one value: a bare atom, or a balanced list
// normalized to SMT-LIB spelling ((- 3) reads back as -3).
func (t smtToks) valueAt(i int) (string, int) {
	if t.at(i) != "(" {
		return t.unquote(i), i + 1
	}
	depth := 0
	j := i
	for j < len(t) {
		if t.at(j) == "(" {
			depth++
		} else if t.at(j) == ")" {
			depth--
			if depth == 0 {
				break
			}
		}
		j++
	}
	if j >= len(t) {
		return "", j
	}
	inner := t[i+1 : j]
	if len(inner) == 2 && inner[0] == "-" {
		return "-" + inner[1], j + 1
	}
	var b strings.Builder
	b.WriteString("(")
	for k, s := range inner {
		if k > 0 {
			b.WriteString(" ")
		}
		b.WriteString(s)
	}
	b.WriteString(")")
	return b.String(), j + 1
}

// prove discharges one obligation: unsat verifies, sat refutes
// with a countermodel over the inputs, anything else is
// inconclusive.
func (p *prover) prove(ob obligation) []Diag {
	script := p.script(ob)
	out, err := runSMTQuery(script, "")
	if err != nil {
		return []Diag{p.inconclusive(ob, err.Error())}
	}
	res, err := parseSMTResult(out)
	if err != nil {
		return []Diag{p.inconclusive(ob, err.Error())}
	}
	switch res {
	case "unsat":
		return nil
	case "sat":
		return []Diag{p.refuted(ob, script)}
	default: // unknown
		return []Diag{p.inconclusive(ob, "solver returned unknown")}
	}
}

func (p *prover) refuted(ob obligation, script string) Diag {
	d := spanDiag(p.text, ob.line, "error",
		fmt.Sprintf("%s %s unproven: the negation is satisfiable", ob.fn, ob.kind),
		shortName(ob.fn), CodeContractUnproven)
	d.Expected = ob.want
	found := "counterexample inputs: " + p.countermodel(script, ob)
	if ob.viaSummary {
		found += " (modulo callee summaries: a weak summary admits inputs no execution reaches)"
	}
	d.Found = found
	d.Hint = "correct the body, or strengthen the relevant explicit summary through reviewed revision changes"
	return d
}

// countermodel replays a satisfiable query with get-value over the
// entry inputs. Best effort: an unparseable model still leaves the
// refutation standing, with the inputs unnamed.
func (p *prover) countermodel(script string, ob obligation) string {
	if len(ob.inputs) == 0 {
		return "none named"
	}
	quoted := make([]string, 0, len(ob.inputs))
	for _, in := range ob.inputs {
		quoted = append(quoted, "|"+in+"|")
	}
	out, err := runSMTQuery(script, "(get-value ("+strings.Join(quoted, " ")+"))\n")
	if err != nil {
		return strings.Join(ob.inputs, "=?, ") + "=? (model unavailable: " + err.Error() + ")"
	}
	model := parseSMTModel(out, ob.inputs)
	parts := make([]string, 0, len(ob.inputs))
	for _, in := range ob.inputs {
		if v, ok := model[in]; ok {
			parts = append(parts, in+"="+v)
		}
	}
	if len(parts) == 0 {
		return "unparseable model"
	}
	return strings.Join(parts, " ")
}

func (p *prover) inconclusive(ob obligation, cause string) Diag {
	d := spanDiag(p.text, ob.line, "error",
		fmt.Sprintf("%s %s inconclusive: %s", ob.fn, ob.kind, cause),
		shortName(ob.fn), CodeContractInconclusive)
	d.Expected = ob.want
	d.Found = cause
	d.Hint = "rerun within the supported policy or reduce proof complexity; no acceptance"
	return d
}

func shortName(qualified string) string {
	if i := strings.LastIndex(qualified, "."); i >= 0 {
		return qualified[i+1:]
	}
	return qualified
}

// checkRows enforces entry admissibility: every test row of a
// contracted function must satisfy requires. Rows bind literals
// (records included, via constructor shape); anything else cannot
// be assessed and stays silent.
func (p *prover) checkRows() []Diag {
	if len(p.fn.Requires) == 0 {
		return nil
	}
	var out []Diag
	for _, row := range p.fn.Tests {
		env := p.paramEnv()
		if env == nil {
			return append(out, p.inconclusive(obligation{fn: p.name, kind: "test rows", line: p.fn.Line, want: descSmalls(p.fn.Requires)}, "obligation construction failed"))
		}
		// Fresh row-local variables: the row binds concrete
		// inputs, not the proof's symbolic ones.
		saved := p.decls
		p.decls = map[string]string{}
		rowEnv := map[string]*symVal{}
		usable := true
		for _, pr := range p.fn.Params {
			var arg *Small
			for _, a := range row.Args {
				if a.Name == pr[0] {
					arg = a.V
					break
				}
			}
			b := p.smallBinding(arg, env)
			if b == nil {
				usable = false
				break
			}
			if !ground(b) {
				usable = false
				break
			}
			rowEnv[pr[0]] = b
		}
		if !usable {
			p.decls = saved
			continue
		}
		var claims []*smt
		ok := true
		for _, r := range p.fn.Requires {
			v, good := p.smallVal(r, rowEnv)
			if !good {
				ok = false
				break
			}
			claims = append(claims, v)
		}
		script := p.script(obligation{req: claims})
		p.decls = saved
		if !ok {
			continue
		}
		res, err := runSMTQuery(script, "")
		if err != nil {
			d := spanDiag(p.text, row.Line, "error",
				fmt.Sprintf("%s test %s inconclusive: %s", p.name, row.Name, err.Error()),
				row.Name, CodeContractInconclusive)
			d.Expected = descSmalls(p.fn.Requires)
			d.Found = err.Error()
			d.Hint = "rerun within the supported policy or reduce proof complexity; no acceptance"
			out = append(out, d)
			continue
		}
		switch r, perr := parseSMTResult(res); {
		case perr != nil:
			d := spanDiag(p.text, row.Line, "error",
				fmt.Sprintf("%s test %s inconclusive: %s", p.name, row.Name, perr.Error()),
				row.Name, CodeContractInconclusive)
			d.Expected = descSmalls(p.fn.Requires)
			d.Found = perr.Error()
			d.Hint = "rerun within the supported policy or reduce proof complexity; no acceptance"
			out = append(out, d)
		case r == "unsat":
			d := spanDiag(p.text, row.Line, "error",
				fmt.Sprintf("%s test %s violates requires: the row admits no input the contract allows", p.name, row.Name),
				row.Name, CodeContractInadmissibleTest)
			d.Expected = descSmalls(p.fn.Requires)
			d.Found = "row " + row.Name + " unsatisfiable under requires"
			d.Hint = "supply an admitted row; do not skip the row or silently narrow expectations"
			out = append(out, d)
		}
	}
	return out
}

// ground reports whether a binding is fully concrete: literals
// and constructors of literals only.
func ground(v *symVal) bool {
	if v == nil {
		return false
	}
	if v.fields == nil {
		return v.term != nil && v.term.op != "var"
	}
	for _, f := range v.fields {
		if !ground(f) {
			return false
		}
	}
	return true
}

// VerifyContracts proves every contracted function callee-first,
// threading the verified set so a callee summary becomes usable
// only after its own obligations discharge in this run. Each run
// re-proves: there is no proof cache, and a saved report is a
// record, not an accepted certificate. Pure classifier, unwired:
// pipeline activation is a later slice.
func VerifyContracts(prog *Program, texts map[string]string) []Diag {
	a := indexAdmission(prog, texts)
	var out []Diag
	recursed := map[string]bool{}
	mark := len(a.out)
	for _, name := range a.checkRecursion() {
		recursed[name] = true
	}
	out = append(out, a.out[mark:]...)
	a.out = a.out[:mark]
	for _, name := range topoContracted(a) {
		if recursed[name] {
			continue
		}
		fn := a.fns[name]
		m := a.ownerOf(fn)
		var text string
		if m != nil {
			text = a.textOf(m)
		}
		base := len(a.out)
		a.checkFn(m, fn)
		diags := append([]Diag{}, a.out[base:]...)
		a.out = a.out[:base]
		out = append(out, diags...)
		if len(diags) > 0 {
			continue
		}
		v := newProver(a, fn, text)
		obs, ok := v.gen()
		if !ok {
			short, line := declNameLine(fn)
			d := spanDiag(text, line, "error",
				fmt.Sprintf("%s obligation construction failed", short),
				short, CodeContractInconclusive)
			d.Expected = "admitted contract and body"
			d.Found = "converter failure on an admitted shape"
			d.Hint = "reduce proof complexity; no acceptance"
			out = append(out, d)
			continue
		}
		rowDiags := v.checkRows()
		out = append(out, rowDiags...)
		failed := len(rowDiags) > 0
		for _, ob := range obs {
			if res := v.prove(ob); len(res) > 0 {
				out = append(out, res...)
				failed = true
			}
		}
		if !failed {
			a.verified[name] = true
		}
	}
	return out
}

// contractIdentities splits function declarations into contracted
// (verified when verification reports no findings) and
// uncontracted (ordinary tested status, never universally
// verified), as qualified name@rev identities in program order.
func contractIdentities(prog *Program) (verified, uncontracted []string) {
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			id := fmt.Sprintf("%s@%d", fn.Name, fn.Rev)
			if isContracted(fn) {
				verified = append(verified, id)
			} else {
				uncontracted = append(uncontracted, id)
			}
		}
	}
	return verified, uncontracted
}

// printVerificationReport renders the verdict position 7 evidence
// record for a passing compilation: what verified, what did not,
// and the scope boundary.
func printVerificationReport(prog *Program) {
	verified, uncontracted := contractIdentities(prog)
	fmt.Println("contract verification: succeeded")
	fmt.Printf("verified declarations: [%s]\n", strings.Join(verified, ", "))
	if len(uncontracted) == 0 {
		fmt.Println("uncontracted declarations: none")
	} else {
		fmt.Printf("uncontracted declarations: not universally verified: [%s]\n", strings.Join(uncontracted, ", "))
	}
	fmt.Println("scope: supported source semantics with verified callees")
}

// topoContracted orders contracted functions callee-first. Cycles
// cannot occur past admission, but the visited set keeps the walk
// total regardless.
func topoContracted(a *admission) []string {
	var names []string
	for name, fn := range a.fns {
		if isContracted(fn) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	visited := map[string]bool{}
	var order []string
	var visit func(string)
	visit = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		fn := a.fns[n]
		if fn != nil {
			collectCalls(fn.Body, func(c string) {
				if t, ok := a.fns[c]; ok && isContracted(t) {
					visit(c)
				}
			})
		}
		order = append(order, n)
	}
	for _, n := range names {
		visit(n)
	}
	return order
}

func recordShapes(mods []*Module) map[string][][2]string {
	recs := map[string][][2]string{}
	for _, b := range builtinTypeDecls() {
		recs[b.Name] = b.Fields
	}
	for _, m := range mods {
		for _, d := range m.Decls {
			if td, ok := d.(*TypeDecl); ok {
				if _, seen := recs[td.Name]; !seen {
					recs[td.Name] = td.Fields
				}
			}
		}
	}
	return recs
}

// variantShapes indexes declared variants by parent name, first
// wins across modules, matching the checker and the evaluator
// (a74). Collisions are rejected at the registry, so first wins is
// deterministic, exactly like records.

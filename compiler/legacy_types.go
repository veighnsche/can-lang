// Predecessor type checker, retained only until the legacy pipeline retirement.
// Current nominal types and compatibility live in internal/types.
package main

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
)

// tycker carries per-function checking state: the world's declared
// record and error shapes plus the value names in scope.
type tycker struct {
	prog   *Program
	text   string
	fn     string
	recs   map[string][][2]string
	errs   map[string][][2]string
	brands map[string]bool
	// variants holds declared variant names (a73): payloads
	// may name variants, but never sequences of them.
	variants map[string]bool
	// cases maps a qualified case name to its check shape
	// (a74): the parent variant is the constructor's nominal
	// type, the fields its exact construction contract.
	cases map[string]variantCase
	// brandFiles maps brand name to declaring file (first wins).
	brandFiles map[string]string
	// brandSeals maps brand name to its declared promotion sources
	// (a26 seals_from; first wins). Empty means str-only minting.
	brandSeals map[string][]string
	// exec is true inside function bodies (executable positions)
	// and false in tests and given rows (checked data positions).
	exec bool
	// cells maps visible cell names to base types (this file only).
	cells map[string]string
	out   []Diag
}

func newTycker(prog *Program, text, fn string) *tycker {
	c := &tycker{prog: prog, text: text, fn: fn,
		recs:       map[string][][2]string{},
		errs:       map[string][][2]string{},
		variants:   map[string]bool{},
		cases:      map[string]variantCase{},
		brands:     map[string]bool{},
		brandFiles: map[string]string{},
		brandSeals: map[string][]string{},
		cells:      map[string]string{}}
	for _, b := range builtinTypeDecls() {
		c.recs[b.Name] = b.Fields
	}
	for _, b := range builtinErrorDecls() {
		c.errs[b.Name] = b.Fields
	}
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			switch d := d.(type) {
			case *TypeDecl:
				if _, ok := c.recs[d.Name]; !ok {
					c.recs[d.Name] = d.Fields
				}
			case *VariantDecl:
				c.variants[d.Name] = true
				// a74: qualified cases enter the construction
				// table with their parent and exact fields.
				// Collisions are rejected at the registry, so
				// first wins here exactly like records.
				for _, vc := range d.Cases {
					q := qualifyCase(d.Name, vc.Short)
					if _, ok := c.cases[q]; !ok {
						c.cases[q] = variantCase{parent: d.Name, fields: vc.Fields}
					}
				}
			case *ErrorDecl:
				if _, ok := c.errs[d.Name]; !ok {
					c.errs[d.Name] = d.Fields
				}
			case *BrandDecl:
				c.brands[d.Name] = true
				if _, ok := c.brandFiles[d.Name]; !ok {
					if f, ok := prog.BrandFile[d.Name]; ok {
						c.brandFiles[d.Name] = f
					}
				}
				if _, ok := c.brandSeals[d.Name]; !ok {
					c.brandSeals[d.Name] = d.SealsFrom
				}
			case *StateDecl:
				// Cells resolve in the checking function's own file;
				// only base types enter (the decl rule owns the rest).
				if prog.FnFile[fn] != m.ID {
					continue
				}
				switch d.Type {
				case "str", "int", "bool", "dec":
					if _, ok := c.cells[d.Name]; !ok {
						c.cells[d.Name] = d.Type
					}
				}
			}
		}
	}
	// Asset bridge grants introduce the named foreign brands into scope
	// so a sink module compiles standalone: params and error fields may
	// name them, but bodies can never seal them (the claimed owner is
	// never the checking file, so the a15 rule keeps failing closed).
	// A real BrandDecl always wins: injection fills only names no loaded
	// module declares, and certifyAssetBridge verifies the claimed owner
	// wherever the owner module loads.
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			g, ok := d.(*AssetBridgeDecl)
			if !ok {
				continue
			}
			for _, name := range []string{g.Asset, g.Policy} {
				if c.brands[name] {
					continue
				}
				c.brands[name] = true
				c.brandFiles[name] = g.Owner
			}
		}
	}
	return c
}

// variantCase is one qualified case's check shape (a74): the
// parent variant it constructs, and the exact declared fields.
type variantCase struct {
	parent string
	fields [][2]string
}

// isCaseType reports whether t is a qualified case name (a75): a
// binder's static identity. Parents are variants, never cases.
func isCaseType(c *tycker, t string) bool {
	_, ok := c.cases[t]
	return ok
}

// seqElemName splits a sequence annotation Seq<T> into its element
// type name (a36 S1). ok=false for anything else, including nested
// sequences: Seq<Seq<str>> is not a v1 shape.
func seqElemName(t string) (string, bool) {
	if !strings.HasPrefix(t, "Seq<") || !strings.HasSuffix(t, ">") {
		return "", false
	}
	elem := t[len("Seq<") : len(t)-1]
	if elem == "" || strings.ContainsAny(elem, "<>") {
		return "", false
	}
	return elem, true
}

// fnTypeShape splits a function-value annotation Fn<A, R, [kinds]>
// into its input type, success type, and error-list text.
// ok=false for anything else, including a non-list third slot.
// Shape only: member validation belongs to expansion.
func fnTypeShape(t string) (string, string, string, bool) {
	if !strings.HasPrefix(t, "Fn<") || !strings.HasSuffix(t, ">") {
		return "", "", "", false
	}
	inner := t[len("Fn<") : len(t)-1]
	var parts []string
	depth, square := 0, 0
	start := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '<':
			depth++
		case '>':
			depth--
		case '[':
			square++
		case ']':
			square--
		case ',':
			if depth == 0 && square == 0 {
				parts = append(parts, inner[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, inner[start:])
	if len(parts) != 3 {
		return "", "", "", false
	}
	a := strings.TrimSpace(parts[0])
	r := strings.TrimSpace(parts[1])
	e := strings.TrimSpace(parts[2])
	if a == "" || r == "" || len(e) < 2 || e[0] != '[' || e[len(e)-1] != ']' {
		return "", "", "", false
	}
	return a, r, e, true
}

// sameType reports whether two type spellings denote the same
// type. Plain spellings compare byte-identical (annotations are
// never re-spaced); Fn heads compare structurally — input,
// success, and error multiset — so one signature written tight
// and one written loose still match. The input recurses (heads
// nest); success is a declared data type.
func sameType(a, b string) bool {
	if a == b {
		return true
	}
	aa, ar, ae, aok := fnTypeShape(a)
	ba, br, be, bok := fnTypeShape(b)
	if !aok || !bok {
		return false
	}
	return sameType(aa, ba) && ar == br && sameFnErrs(ae, be)
}

// sameFnErrs compares two bracketed Fn error lists as multisets:
// canonical order is enforced where heads are written, so any
// residual difference here is spacing, never meaning.
func sameFnErrs(a, b string) bool {
	kinds := func(e string) []string {
		raw := strings.TrimSpace(e[1 : len(e)-1])
		if raw == "" {
			return nil
		}
		var out []string
		for _, k := range strings.Split(raw, ",") {
			out = append(out, strings.TrimSpace(k))
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(kinds(a), kinds(b))
}

// typeHasFn reports whether a type spelling contains a callable
// anywhere: a direct Fn head, a Seq element, a generic argument,
// or a record field transitively. The visited set keeps
// Seq-recursive data graphs terminating. Brands, errors, variants,
// and scalars are leaves: their declarations reject Fn payloads
// at their own positions, so containment never needs to look
// inside them — a violating declaration fails there regardless.
func (c *tycker) typeHasFn(t string) bool {
	return typeContainsFn(t, c.recs)
}

func typeContainsFn(t string, records map[string][][2]string) bool {
	seen := map[string]bool{}
	var has func(x string) bool
	has = func(x string) bool {
		if _, _, _, ok := fnTypeShape(x); ok {
			return true
		}
		if elem, ok := seqElemName(x); ok {
			if seen[x] {
				return false
			}
			seen[x] = true
			return has(elem)
		}
		if strings.HasPrefix(x, "Seq<") && strings.HasSuffix(x, ">") {
			if seen[x] {
				return false
			}
			seen[x] = true
			return has(x[len("Seq<") : len(x)-1])
		}
		if _, args, ok := splitMention(x); ok {
			if seen[x] {
				return false
			}
			seen[x] = true
			for _, a := range args {
				if has(strings.TrimSpace(a)) {
					return true
				}
			}
			return false
		}
		fields, ok := records[x]
		if !ok {
			return false
		}
		if seen[x] {
			return false
		}
		seen[x] = true
		for _, f := range fields {
			if has(f[1]) {
				return true
			}
		}
		return false
	}
	return has(t)
}

// captureDataKind reports whether a Small kind may appear in a
// reference capture: literals, variable/field references, seals,
// and data constructions. Anything executing or computing is
// outside B00, as are nested references and script-only forms.
func captureDataKind(kind string) bool {
	switch kind {
	case "str", "int", "bool", "dec", "float", "ref", "ctor", "seqlit", "seal":
		return true
	}
	return false
}

// checkFnref validates one reference creation: the target resolves
// to an admitted source function under the creator's authority,
// captures bind by name in declaration order leaving exactly one
// parameter unbound, every capture is data of its slot type with
// no function inside, and the denoted input and success carrier
// are data-only. The denoted Fn type lands on s.T; the generic
// want tail compares it against the annotation through typeOf.
func (c *tycker) checkFnref(s *Small, line int, env map[string]string, where string) {
	show := s.Fname
	if base, ok := c.prog.GenericBase[s.Fname]; ok {
		show = base
	}
	structural := func() {
		for _, a := range s.Args {
			c.value(a.V, "", line, env, where)
		}
	}
	tgt, ok := c.prog.Fns[s.Fname]
	if !ok {
		if c.prog.Externs[s.Fname] != nil {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("reference to extern %s refused: only source functions are referenceable", show), "fnref", CodeFnTargetRefused))
		} else if kind := classifyCallee(c.prog, "", s.Fname); kind.IsIntrinsic() {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("reference to %s refused: compiler kernels are not referenceable", show), "fnref", CodeFnTargetRefused))
		} else {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s references unknown function %s", c.fn, show), "fnref", CodeUnknownCall))
		}
		structural()
		return
	}
	// Q3a: a foreign reference needs the creator's own uses pin,
	// exactly like a foreign call; the pin names the base.
	if localCallee(c.prog, c.fn, s.Fname) == nil && !c.prog.Uses[s.Fname] {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("%s references %s which is not in uses: add name@rev to uses", c.fn, show), "fnref", CodeCallNotInUses))
	}
	slots, unbound, berr := bindRefSlots(s.Fname, s.Args, tgt.Params)
	if berr != nil {
		c.out = append(c.out, spanDiag(c.text, line, "error", berr.Error(), "fnref", CodeBadBinding))
		structural()
		c.checkFnrefTarget(s, tgt, show, line)
		return
	}
	if len(unbound) != 1 {
		if len(unbound) == 0 {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("reference to %s binds every parameter: leave exactly one unbound (a reference is not a call)", show), "fnref", CodeFnResidualArity))
		} else {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("reference to %s leaves %d parameters unbound: bind all but exactly one", show, len(unbound)), "fnref", CodeFnResidualArity))
		}
	}
	for i, a := range s.Args {
		p := tgt.Params[slots[i]]
		label := "reference to " + show + " capture " + p[0]
		c.value(a.V, "", line, env, label)
		walkSmallTrees(a.V, func(x *Small) {
			if !captureDataKind(x.Kind) {
				name := p[0]
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("capture %s of reference to %s is computed (%s): captures are literals, refs, seals, and data constructions", name, show, x.Kind), "fnref", CodeFnComputedCapture))
			}
		})
		if c.knownType(p[1]) {
			if got, ok := c.typeOf(a.V, env); ok && !sameType(got, p[1]) {
				c.mismatch(line, label, got, p[1], tokenOf(a.V))
			}
		}
		if got, ok := c.typeOf(a.V, env); ok && c.typeHasFn(got) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("capture %s of reference to %s contains a function value: captures are data-only", p[0], show), "fnref", CodeFnContainment))
		}
	}
	c.checkFnrefTarget(s, tgt, show, line)
	if len(unbound) == 1 {
		a := tgt.Params[unbound[0]][1]
		if c.typeHasFn(a) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("input %s of reference to %s contains a function value: inputs are data-only", a, show), "fnref", CodeFnContainment))
		}
		if c.typeHasFn(tgt.Ret) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("success %s of reference to %s contains a function value: success carriers are data-only", tgt.Ret, show), "fnref", CodeFnContainment))
		}
		if t, ok := c.refFnType(s); ok {
			s.T = t
		}
	}
}

// checkFnrefTarget enforces target admission: the return names a
// supported success type, the reachable graph is linked-pure, and no
// reachable function needs a precondition. Containment is checked separately.
// Captures never discharge any of these.
func (c *tycker) checkFnrefTarget(s *Small, tgt *FnDecl, show string, line int) {
	if _, ok := successFields(tgt.Ret, c.recs, c.variants[tgt.Ret] || c.brands[tgt.Ret]); !ok || !c.knownType(tgt.Ret) {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("reference to %s refused: success %s is not a supported success type", show, tgt.Ret), "fnref", CodeFnTargetRefused))
	}
	if err := checkLinkedGraph(c.prog, s.Fname); err != nil {
		detail := strings.TrimPrefix(err.Error(), "linked execution refused: ")
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("reference to %s refused: %s", show, detail), "fnref", CodeFnTargetRefused))
	}
	if holder := firstRequiresHolder(c.prog, s.Fname); holder != "" {
		disp := holder
		if base, ok := c.prog.GenericBase[holder]; ok {
			disp = base
		}
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("reference to %s refused: %s has a required precondition", show, disp), "fnref", CodeFnTargetRefused))
	}
}

// firstRequiresHolder returns the first function with a required
// precondition reachable from root (itself included) over direct
// calls, or "" when the reachable graph is precondition-free.
// Shared by reference-target admission and invoke-cycle
// admissibility: captures never discharge preconditions in either.
func firstRequiresHolder(prog *Program, root string) string {
	seen := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		fn, ok := prog.Fns[name]
		if !ok {
			continue
		}
		if len(fn.Requires) > 0 {
			return name
		}
		for _, cc := range walkCalls(fn.Body) {
			if !seen[cc.Fname] {
				seen[cc.Fname] = true
				queue = append(queue, cc.Fname)
			}
		}
	}
	return ""
}

// fnSeqDiags reports a sequence element carrying a callable. The
// annotation may be well-formed (a stamped element name); the
// position still refuses function values transitively.
func (c *tycker) fnSeqDiags(t, where string, line int, token string) []Diag {
	if elem, ok := seqElemName(t); ok && c.typeHasFn(elem) {
		return []Diag{spanDiag(c.text, line, "error",
			fmt.Sprintf("sequence element %s of %s contains a function value: sequence elements are data-only", elem, where), token, CodeFnContainment)}
	}
	return nil
}

// successSeqDiags keeps existing sequence element restrictions when the
// sequence is the success itself, not a declared record field.
func (c *tycker) successSeqDiags(ret string, line int) []Diag {
	out := c.fnSeqDiags(ret, "returns", line, ret)
	if elem, ok := seqElemName(ret); ok && c.variants[elem] {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("Seq<%s> is deferred: variant sequences are not admitted", elem), ret, CodeUnknownType))
	}
	return out
}

// refFnType computes the callable type a reference denotes: the
// unbound parameter becomes A, the target return becomes R, and
// the target emits become the canonically ordered error list.
// ok=false means unresolvable; the value rule owns every
// diagnostic, so shape failures stay silent here.
func (c *tycker) refFnType(s *Small) (string, bool) {
	tgt, ok := c.prog.Fns[s.Fname]
	if !ok {
		return "", false
	}
	_, unbound, err := bindRefSlots(s.Fname, s.Args, tgt.Params)
	if err != nil || len(unbound) != 1 {
		return "", false
	}
	if _, ok := successFields(tgt.Ret, c.recs, c.variants[tgt.Ret] || c.brands[tgt.Ret]); !ok || !c.knownType(tgt.Ret) || len(c.successSeqDiags(tgt.Ret, tgt.Line)) != 0 {
		return "", false
	}
	errs := slices.Clone(tgt.Emits)
	slices.Sort(errs)
	return fmt.Sprintf("Fn<%s, %s, [%s]>", tgt.Params[unbound[0]][1], tgt.Ret, strings.Join(errs, ", ")), true
}

// fnHeadDiags enforces b00 deep validity on one Fn annotation that
// shape-accepted: the input names a known type, the success names a
// supported data type, every kind names a declared error, and the kinds are
// distinct and canonically (lexicographically) ordered. Unknown
// input reuses CAN6002 and unknown kinds reuse CAN4002; head-invalid
// shapes (unsupported success, duplicate or unordered kinds) are
// CAN6019. Non-Fn annotations are a silent no-op, so call sites
// need no shape gate. This cut validates the head everywhere it
// is written; transitive containment (Fn inside A or R through
// records) and the sequence/variant/error/const/extern position
// exclusions live in typeHasFn and its call sites.
func (c *tycker) fnHeadDiags(t, where string, line int, token string) []Diag {
	a, r, e, ok := fnTypeShape(t)
	if !ok {
		return nil
	}
	var out []Diag
	if !c.knownType(a) {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("unknown type %s in Fn input of %s", a, where), token, CodeUnknownType))
	}
	if _, ok := successFields(r, c.recs, c.variants[r] || c.brands[r]); !ok {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("Fn success %s of %s is not a supported success type", r, where), token, CodeFnHeadInvalid))
	}
	if _, ok := seqElemName(r); ok && !c.knownType(r) {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("unknown type %s in Fn success of %s", r, where), token, CodeUnknownType))
	}
	out = append(out, c.successSeqDiags(r, line)...)
	// Invocation inputs and success carriers are data-only (b00
	// Q1d): a bearing head would smuggle callbacks through the
	// invocation boundary. The reference side enforces the same
	// rule on denoted types; the head side enforces it here.
	if c.typeHasFn(a) {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("Fn input %s of %s contains a function value: inputs are data-only", a, where), token, CodeFnContainment))
	}
	if c.typeHasFn(r) {
		out = append(out, spanDiag(c.text, line, "error",
			fmt.Sprintf("Fn success %s of %s contains a function value: success carriers are data-only", r, where), token, CodeFnContainment))
	}
	raw := strings.TrimSpace(e[1 : len(e)-1])
	if raw == "" {
		return out
	}
	var kinds []string
	for _, k := range strings.Split(raw, ",") {
		kinds = append(kinds, strings.TrimSpace(k))
	}
	seen := map[string]bool{}
	var checked []string
	for _, k := range kinds {
		if _, ok := c.errs[k]; !ok {
			out = append(out, spanDiag(c.text, line, "error",
				fmt.Sprintf("unknown error kind %s in Fn list of %s", k, where), token, CodeUnknownKind))
			continue
		}
		if seen[k] {
			out = append(out, spanDiag(c.text, line, "error",
				fmt.Sprintf("duplicate error kind %s in Fn list of %s", k, where), token, CodeFnHeadInvalid))
			continue
		}
		seen[k] = true
		checked = append(checked, k)
	}
	want := slices.Clone(checked)
	slices.Sort(want)
	for i := range checked {
		if checked[i] != want[i] {
			out = append(out, spanDiag(c.text, line, "error",
				fmt.Sprintf("error kinds in Fn list of %s are not canonically ordered: want [%s]", where, strings.Join(want, ", ")), token, CodeFnHeadInvalid))
			break
		}
	}
	return out
}

// knownType reports whether a name is a legal annotation: a base type,
// the Bytes primitive, a declared record, a declared brand, a
// sequence over a plain element type, or a function-value head.
// Error kinds are not values.
func (c *tycker) knownType(t string) bool {
	switch t {
	case "str", "int", "bool", "dec", "Bytes":
		return true
	}
	if _, _, _, ok := fnTypeShape(t); ok {
		// The head shape admits the annotation so the callable
		// validators report precise codes (slots, arity, error
		// kinds) instead of drowning in unknown-type noise.
		return true
	}
	if elem, ok := seqElemName(t); ok {
		if _, nested := seqElemName(elem); nested {
			return false
		}
		return c.knownType(elem)
	}
	if _, ok := c.recs[t]; ok {
		return true
	}
	if c.variants[t] {
		return true
	}
	return c.brands[t]
}

type calleeSig struct {
	params [][2]string
	ret    string
}

func (c *tycker) callee(name string) *calleeSig {
	if k, ok := bytesKernels[name]; ok && !k.restricted {
		return &calleeSig{k.params, k.ret}
	}
	if f, ok := c.prog.Fns[name]; ok {
		return &calleeSig{f.Params, f.Ret}
	}
	if e, ok := c.prog.Externs[name]; ok {
		return &calleeSig{e.Params, e.Ret}
	}
	return nil
}

func (c *tycker) mismatch(line int, where, got, want, token string) {
	c.out = append(c.out, spanDiag(c.text, line, "error",
		fmt.Sprintf("%s: got %s, want %s", where, got, want), token, CodeTypeMismatch))
}

// resolveRef types a variable or field path. ok=false means uncheckable;
// the caller (value, which visits each node once) owns the diagnostic.
func (c *tycker) resolveRef(ref []string, env map[string]string) (string, bool) {
	if len(ref) == 0 {
		return "", false
	}
	t, ok := env[ref[0]]
	if !ok || t == "" {
		// Slice 1: mirror the value() exemption so pure
		// type queries resolve constants to their sort.
		if len(ref) == 1 && constNameRe.MatchString(ref[0]) {
			if decl, found := lookupConst(c.prog, ref[0]); found {
				return decl.Type, true
			}
		}
		return "", false
	}
	if strings.HasPrefix(t, "cell:") {
		// Synthetic get payload: exactly .value of the cell type.
		if len(ref) == 2 && ref[1] == "value" {
			return strings.TrimPrefix(t, "cell:"), true
		}
		return "", false
	}
	if t == "parts" {
		// Synthetic observation payload: exactly .coefficient and
		// .scale, both int. Lowercase, so no declared type collides.
		if len(ref) == 2 && (ref[1] == "coefficient" || ref[1] == "scale") {
			return "int", true
		}
		return "", false
	}
	for _, f := range ref[1:] {
		var fields [][2]string
		if strings.HasPrefix(t, "err:") {
			fields = c.errs[strings.TrimPrefix(t, "err:")]
		} else if cc, ok := c.cases[t]; ok {
			// a75: a case binder views its own payload. The
			// binder's static type is the qualified case, so a
			// same-named field of a sibling case never resolves
			// here — projection is case-specific by construction.
			fields = cc.fields
		} else {
			fields = c.recs[t]
			if fields == nil && !c.brands[t] && t != "str" && t != "int" && t != "bool" && t != "dec" {
				return "", false
			}
		}
		if fields == nil {
			return "", false
		}
		found := false
		for _, fd := range fields {
			if fd[0] == f {
				t = fd[1]
				found = true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return t, true
}

// typeOf is the pure half of checking: the static type of a Small, or
// ok=false when the position is unchecked (calls resolve through their
// own rule, stubs through the callee contract, wildcards are patterns).
func (c *tycker) typeOf(s *Small, env map[string]string) (string, bool) {
	switch s.Kind {
	case "str":
		return "str", true
	case "int":
		return "int", true
	case "bool":
		return "bool", true
	case "dec":
		return "dec", true
	case "seal":
		if !c.brands[s.Seal] {
			return "", false
		}
		return s.Seal, true
	case "fnref":
		return c.refFnType(s)
	case "seqlit":
		// A typed literal carries its sequence type outward, exactly
		// like a record constructor carries its record type, so outer
		// positions (fields, args, expectations) check the element
		// identity instead of re-deriving it. Unknown element types
		// stay silent here: value() owns that diagnostic.
		if _, nested := seqElemName(s.Elem); nested {
			return "", false
		}
		if !c.knownType(s.Elem) {
			return "", false
		}
		return "Seq<" + s.Elem + ">", true
	case "ref":
		t, ok := c.resolveRef(s.Ref, env)
		if !ok {
			return "", false
		}
		if strings.HasPrefix(t, "err:") {
			return t, true
		}
		// A variable of undeclared type is unchecked here: the
		// param/field declaration owns the CAN6002, and comparing
		// through it would cascade one typo into many.
		if !c.knownType(t) {
			// a75: a case binder carries its qualified case as
			// its static type. It is not a known annotation, but
			// it must compare — otherwise a binder smuggled into
			// a parent-typed position would pass silently. The
			// case name never equals the parent, so the mismatch
			// fires exactly where whole-union smuggling is tried.
			if _, ok := c.cases[t]; ok {
				return t, true
			}
			return "", false
		}
		return t, true
	case "ctor":
		// A named record constructor carries its record type outward
		// so outer positions (Ok fields, call arguments, test
		// expectations, equality) check the constructor identity, not
		// just the inner fields. Ok and error ctors resolve through
		// their own rules; unknown records stay silent here because
		// checkCtor owns the unknown-record diagnostic.
		if s.Ctor != "Ok" && !strings.Contains(s.Ctor, ".") {
			// a45 S1: the primitive Bytes constructor carries its
			// type outward like a record constructor. checkCtor
			// owns shape and range validation.
			if s.Ctor == "Bytes" {
				return "Bytes", true
			}
			if _, ok := c.recs[s.Ctor]; ok {
				return s.Ctor, true
			}
			// a74: a qualified case constructor carries its
			// parent variant outward, so outer positions check
			// the nominal identity, never the payload shape.
			// checkCtor owns the undeclared-case diagnostic.
			if cc, ok := c.cases[s.Ctor]; ok {
				return cc.parent, true
			}
		}
		return "", false
	case "strlen":
		return "int", true
	case "stridx":
		// a38 S3: a sequence base yields its element type, so a
		// brand member stays branded at check time. Unknown bases
		// keep the historical int; the value rule owns the base
		// diagnostic and preseq code never sees a sequence.
		if lt, ok := c.typeOf(s.L, env); ok {
			if elem, isSeq := seqElemName(lt); isSeq {
				return elem, true
			}
		}
		return "int", true
	case "proj":
		// b03: a projection carries its field's type outward.
		// Unknown bases and non-records stay silent: the value
		// rule owns both diagnostics.
		if bt, ok := c.typeOf(s.L, env); ok {
			if fields, ok := c.recs[bt]; ok {
				for _, fd := range fields {
					if fd[0] == s.Field {
						return fd[1], true
					}
				}
			}
		}
		return "", false
	case "strslice":
		return "str", true
	case "not":
		// Slice 5: negation takes and yields bools; the value
		// rule owns the operand refusal.
		return "bool", true
	case "neg":
		// Slice 6: prefix minus over exact int or dec yields
		// the operand type; the value rule owns the refusal.
		if t, ok := c.typeOf(s.L, env); ok && (t == "int" || t == "dec") {
			return t, true
		}
		return "", false
	case "binop":
		if !isArith(s.Op) {
			return "bool", true
		}
		l, lok := c.typeOf(s.L, env)
		r, rok := c.typeOf(s.R, env)
		// a16: + concatenates strings (construction needs no
		// indexing); - and * stay numeric-only, and brands and
		// bools compute nothing even when both sides agree.
		if s.Op == "+" && lok && rok && l == "str" && r == "str" {
			return "str", true
		}
		// a39 S4: an append carries its sequence type outward,
		// so outer positions check element identity once.
		if s.Op == "+" && lok && rok {
			if le, ok := seqElemName(l); ok && r == le {
				return l, true
			}
		}
		// a42: same-brand str-backed + yields the brand.
		if s.Op == "+" && lok && rok && l == r && c.brands[l] {
			if under, ok := c.prog.Brands[l]; ok && under == "str" {
				return l, true
			}
		}
		// a17: decimal division and remainder have no exact result;
		// keep them untyped so parents stay silent and the value
		// rule reports the one refusal.
		if (s.Op == "/" || s.Op == "%") && lok && rok && l == "dec" && r == "dec" {
			return "", false
		}
		if !lok || !rok || l != r || (l != "int" && l != "dec") {
			return "", false
		}
		return l, true
	}
	return "", false
}

// isArith reports the computing operators: comparisons ask, these do.
// a17 adds / and % (exact Euclidean integer division); dec operands
// for either are refused in the value rule, not here.
func isArith(op string) bool {
	return op == "+" || op == "-" || op == "*" || op == "/" || op == "%"
}

// isOrdering reports the inequality comparisons: <, >, <=, >=.
// Unlike ==/!= (structural over any same-type pair), ordering
// needs an ordered domain.
func isOrdering(op string) bool {
	return op == "<" || op == ">" || op == "<=" || op == ">="
}

// arithVerb names the operator class for mismatch messages, so agents
// do not file arithmetic mistakes under comparison.
func arithVerb(op string) string {
	switch op {
	case "+":
		return "add"
	case "-":
		return "subtract"
	case "/":
		return "divide"
	case "%":
		return "modulo"
	default:
		return "multiply"
	}
}

// tokenOf picks the squiggle token for a value: the applied name for
// calls, the leaf for paths, the operator for comparisons.
func tokenOf(s *Small) string {
	switch s.Kind {
	case "call":
		return s.Fname
	case "ref":
		if len(s.Ref) > 0 {
			return s.Ref[len(s.Ref)-1]
		}
	case "binop":
		return s.Op
	case "not":
		return "not"
	case "neg":
		return "-"
	case "strlen":
		return "#"
	case "stridx":
		return "[]"
	case "proj":
		return s.Field
	case "strslice":
		return "[:]"
	case "seal":
		return s.Seal
	case "seqlit":
		return "Seq"
	case "ctor":
		return s.Ctor
	case "exchange":
		return "exchange"
	case "fnref":
		return "fnref"
	}
	return ""
}

// value checks one Small against its wanted type and recurses into
// children with their own wants. want="" means the position is
// want-free (arm roots producing errors, scrutinees): children are
// still checked structurally, so one broken row never hides the rest.
func (c *tycker) value(s *Small, want string, line int, env map[string]string, where string) {
	if s == nil {
		return
	}
	if s.Kind == "float" {
		msg := fmt.Sprintf("float %s has no spelling: write d\"%s\"", s.Str, s.Str)
		if strings.ContainsAny(s.Str, "eE") {
			msg = fmt.Sprintf("float %s has no spelling: write the value as d\"12.34\"", s.Str)
		}
		c.out = append(c.out, spanDiag(c.text, line, "error", msg, s.Str, CodeFloatLiteral))
		return
	}
	if s.Kind == "seal" {
		if !c.brands[s.Seal] {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("unknown brand %s in seal", s.Seal), s.Seal, CodeUnknownType))
			return
		}
		// a15: executable code mints only its own module's brands.
		// The declaring file owns every executable seal site (grep
		// seal is the audit); tests and given rows may name any
		// declared brand because they are checked data, not code.
		if c.exec {
			if owner, ok := c.brandFiles[s.Seal]; ok && owner != c.prog.FnFile[c.fn] {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("seal %s in %s mints a brand declared in %s: bodies seal only their own module's brands", s.Seal, c.fn, owner), s.Seal, CodeSealForeign))
				return
			}
		}
		// a10: brands erase to strings at runtime; emit reads T.
		s.T = s.Seal
		// a25: seals take string literals or string-typed refs and
		// fields, so decision-tabled constructors can mint computed
		// brands. The file-ownership rule above stays the audit.
		if len(s.Args) != 1 {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("seal %s takes one value", s.Seal), s.Seal, CodeTypeMismatch))
			return
		}
		slabel := fmt.Sprintf("seal %s value", s.Seal)
		c.value(s.Args[0].V, "", line, env, slabel)
		if got, ok := c.typeOf(s.Args[0].V, env); ok && c.brands[got] {
			// a26: explicitly authorized one-way promotion. The
			// destination's seals_from names the admitted source
			// brands; unlisted sources stay CAN6003, and there is
			// no reverse, transitive, or inferred promotion.
			if !slices.Contains(c.brandSeals[s.Seal], got) {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("seal %s cannot promote %s: add %s to seals_from on %s", s.Seal, got, got, s.Seal), s.Seal, CodeTypeMismatch))
				return
			}
			if srcFile, ok := c.brandFiles[got]; ok {
				if dstFile, ok := c.brandFiles[s.Seal]; ok && srcFile != dstFile {
					c.out = append(c.out, spanDiag(c.text, line, "error",
						fmt.Sprintf("seal %s promotes %s from %s: promotions stay inside one module", s.Seal, got, srcFile), s.Seal, CodeSealForeign))
					return
				}
			}
		} else if !ok || got != "str" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("seal %s takes a string", s.Seal), s.Seal, CodeTypeMismatch))
			return
		}
		if want != "" && want != s.Seal {
			c.mismatch(line, where, s.Seal, want, s.Seal)
		}
		return
	}
	if s.Kind == "ref" {
		t, ok := env[s.Ref[0]]
		if !ok {
			// Slice 1: a constant reference carries its
			// declared sort: existence is checkConstRefs'
			// job (CAN2104), so the type checker
			// substitutes instead of reporting unbound.
			if len(s.Ref) > 0 && constNameRe.MatchString(s.Ref[0]) {
				if decl, found := lookupConst(c.prog, s.Ref[0]); found {
					t, ok = decl.Type, true
				}
			}
		}
		if !ok {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("unbound name %s in %s", s.Ref[0], c.fn), s.Ref[0], CodeTypeMismatch))
			return
		}
		if t == "" {
			// Bound to an unknown contract (unknown callee or
			// return): the owner reports it; checking here would
			// cascade one broken line into many.
			return
		}
		if strings.HasPrefix(t, "ok:") && len(s.Ref) == 1 {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s is a value-return Ok payload: select %s.value", s.Ref[0], s.Ref[0]), s.Ref[0], CodeTypeMismatch))
			return
		}
		if t == "empty-ok" {
			// Bound to a put's empty Ok: it carries no fields and
			// no value, so any use is a mistake at this line.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s is the empty Ok of a put: it carries no fields", s.Ref[0]), s.Ref[0], CodeTypeMismatch))
			return
		}
		got, ok := c.resolveRef(s.Ref, env)
		if !ok {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("no field %s on %s", strings.Join(s.Ref[1:], "."), s.Ref[0]), tokenOf(s), CodeTypeMismatch))
			return
		}
		if want != "" && !strings.HasPrefix(got, "err:") && !sameType(got, want) {
			c.mismatch(line, where, got, want, tokenOf(s))
		}
		// a10: record the resolved type for emit's typed dispatch.
		s.T = got
		return
	}
	if want != "" {
		if got, ok := c.typeOf(s, env); ok && !sameType(got, want) {
			c.mismatch(line, where, got, want, tokenOf(s))
		}
	}
	switch s.Kind {
	case "not":
		// Slice 5: prefix negation over one bool operand, no
		// calls inside (those stay CAN3003 outside a match
		// scrutinee), no truthiness: CAN6003 names the type.
		s.T = "bool"
		c.value(s.L, "", line, env, "not")
		t, ok := c.typeOf(s.L, env)
		if !ok {
			return
		}
		if t != "bool" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot not %s: not takes a bool operand", t), "not", CodeTypeMismatch))
		}
		return
	case "neg":
		// Slice 6: prefix minus over exact int or dec. Calls
		// inside stay CAN3003 outside a match scrutinee;
		// anything else is CAN6003. The result carries the
		// operand type for emit's typed dispatch.
		c.value(s.L, "", line, env, "negate")
		t, ok := c.typeOf(s.L, env)
		if !ok {
			return
		}
		if t != "int" && t != "dec" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot negate %s: unary minus takes int or dec", t), "-", CodeTypeMismatch))
			return
		}
		s.T = t
		return
	case "binop":
		where := "comparison"
		if isArith(s.Op) {
			where = arithVerb(s.Op)
		}
		if s.Op == "and" || s.Op == "or" {
			// Slice 5: eager combinators over two bool
			// operands. Calls inside stay CAN3003 (outside
			// a match scrutinee); mistyped sides are
			// CAN6003; the result is always bool.
			s.T = "bool"
			where = s.Op
			c.value(s.L, "", line, env, where)
			c.value(s.R, "", line, env, where)
			l, lok := c.typeOf(s.L, env)
			r, rok := c.typeOf(s.R, env)
			if !lok || !rok {
				return
			}
			if l != "bool" || r != "bool" {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("cannot %s %s with %s: and/or take bool operands", s.Op, l, r), s.Op, CodeTypeMismatch))
			}
			return
		}
		c.value(s.L, "", line, env, where)
		c.value(s.R, "", line, env, where)
		l, lok := c.typeOf(s.L, env)
		r, rok := c.typeOf(s.R, env)
		if !lok || !rok {
			return
		}
		// a10: record the operand type for emit's typed dispatch.
		// Same-type operands are enforced below; emit runs only on
		// success, so the annotation always agrees there.
		if l == r {
			s.T = l
		}
		if isArith(s.Op) {
			// Arithmetic yields the operand type, but only int
			// and dec compute, plus str under + (a16: explicit
			// construction). Same-brand seals and bools do not,
			// even when both sides agree; neither do - and * on
			// strings.
			if l == r && l == "str" && s.Op == "+" {
				return
			}
			// a42: B + B -> B for str-backed brands (fragment
			// assembly). Both operands are already minted, so
			// no seal site is added or bypassed (the grep-seal
			// audit is untouched); the result stays branded
			// (sink rule intact). Eval and emit erase brands
			// already, so this arm is checker-only. Anything
			// else brand-flavored falls through to the
			// refusal below.
			if s.Op == "+" && lok && rok && l == r && c.brands[l] {
				if under, ok := c.prog.Brands[l]; ok && under == "str" {
					s.T = l
					return
				}
			}
			// a39 S4: Seq<T> + T appends, yielding the sequence
			// type. Seq + Seq is a separate concatenation
			// contract (not v1) with its own diagnostic; a
			// member on the left keeps the existing
			// no-conversions refusal below.
			if s.Op == "+" && lok && rok {
				if le, ok := seqElemName(l); ok {
					if _, rok := seqElemName(r); rok {
						c.out = append(c.out, spanDiag(c.text, line, "error",
							fmt.Sprintf("cannot add %s with %s: sequence concatenation is not in v1", l, r), s.Op, CodeTypeMismatch))
						return
					}
					if !sameType(r, le) {
						c.mismatch(line, where, r, le, s.Op)
						return
					}
					s.T = l
					return
				}
			}
			// a17: 1/3 does not terminate, so decimal division and
			// remainder are refused per operation. Integers divide
			// exactly (Euclidean); use them.
			if l == r && l == "dec" && (s.Op == "/" || s.Op == "%") {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("dec %s has no exact result: divide integers, not decimals", s.Op), s.Op, CodeInexactDivision))
				return
			}
			if l != r || (l != "int" && l != "dec") {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("cannot %s %s with %s: no implicit conversions", arithVerb(s.Op), l, r), s.Op, CodeTypeMismatch))
			}
			return
		}
		if l != r {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot compare %s with %s: no implicit conversions", l, r), s.Op, CodeTypeMismatch))
		} else if l == "Bytes" || r == "Bytes" {
			// a45 S1: no direct byte operators. Structural comparison
			// lives in the test evaluator (vEq) only, so record and
			// payload expectations still verify.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot compare %s with %s: Bytes comparison is not in v1", l, r), s.Op, CodeTypeMismatch))
		} else if _, ok := seqElemName(l); ok {
			// a36 S1 promises no sequence equality surface: ==
			// over two sequences is a compile error, not a silent
			// shape. Structural comparison lives in the test
			// evaluator (vEq) only, so expectations still verify.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot compare %s with %s: sequence equality is not in v1", l, r), s.Op, CodeTypeMismatch))
		} else if c.variants[l] || isCaseType(c, l) {
			// a74: cases compare by matching (a75), never by ==.
			// The refusal lands here so no variant operand sails
			// through to a loud emit failure. Structural
			// comparison lives in the test evaluator (vEq) only,
			// so expectations over variant payloads still verify.
			// A future slice may amend this explicitly if it
			// carries its own comparison convention. a75: case
			// binders compare through their case identity, so the
			// refusal covers them too.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot compare %s with %s: variant equality is not in v1", l, r), s.Op, CodeTypeMismatch))
		} else if c.typeHasFn(l) {
			// b00: function values do not compare, directly or
			// through a bearing record. Factory receipt equality
			// stays in the test evaluator (expectEq); the
			// language surface refuses here so no operand sails
			// through to object identity at emit.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot compare %s with %s: function values do not compare", l, r), s.Op, CodeTypeMismatch))
		} else if isOrdering(s.Op) && l != "int" && l != "str" && l != "dec" && !c.brands[l] {
			// Ordering needs an ordered domain: records,
			// bools, cells, and payloads compare for
			// equality only. Without this gate a record <
			// record sails through to a meaningless native
			// comparison at emit. Brands erase to strings,
			// so every brand orders as str.
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot order %s with %s: %s takes int, str, dec, or brand operands", l, r, s.Op), s.Op, CodeTypeMismatch))
		}
	case "strlen":
		c.value(s.L, "", line, env, "length")
		if t, ok := c.typeOf(s.L, env); ok && t != "str" {
			// a37 S2: # counts sequence elements too. Anything
			// else keeps the pinned scalar diagnostic verbatim.
			if _, isSeq := seqElemName(t); !isSeq {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("cannot count scalars of %s: length needs str", t), "#", CodeTypeMismatch))
				return
			}
		}
		s.T = "int"
	case "stridx":
		c.value(s.L, "", line, env, "index base")
		c.value(s.R, "", line, env, "index")
		if t, ok := c.typeOf(s.L, env); ok && t != "str" {
			// a38 S3: a sequence base indexes to its element
			// type. The index stays int; anything else keeps
			// the pinned diagnostic verbatim.
			if elem, isSeq := seqElemName(t); isSeq {
				if it, ok := c.typeOf(s.R, env); ok && it != "int" {
					c.out = append(c.out, spanDiag(c.text, line, "error",
						fmt.Sprintf("cannot index with %s: index must be int", it), "[]", CodeTypeMismatch))
					return
				}
				s.T = elem
				return
			}
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot index into %s: base must be str", t), "[]", CodeTypeMismatch))
			return
		}
		if t, ok := c.typeOf(s.R, env); ok && t != "int" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot index with %s: index must be int", t), "[]", CodeTypeMismatch))
			return
		}
		s.T = "int"
	case "proj":
		// b03: the base must be a record carrying the field.
		// Unknown bases stay silent (the base owns the error);
		// the want tail above compares through s.T.
		c.value(s.L, "", line, env, "projection base")
		bt, ok := c.typeOf(s.L, env)
		if !ok {
			return
		}
		fields, ok := c.recs[bt]
		if !ok {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot project .%s of %s: base must be a record", s.Field, bt), s.Field, CodeTypeMismatch))
			return
		}
		for _, fd := range fields {
			if fd[0] == s.Field {
				s.T = fd[1]
				return
			}
		}
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("no field %s on %s", s.Field, bt), s.Field, CodeTypeMismatch))
	case "strslice":
		c.value(s.L, "", line, env, "slice base")
		c.value(s.R, "", line, env, "slice start")
		c.value(s.Hi, "", line, env, "slice end")
		if t, ok := c.typeOf(s.L, env); ok && t != "str" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("cannot slice %s: base must be str", t), "[:]", CodeTypeMismatch))
			return
		}
		for _, b := range []*Small{s.R, s.Hi} {
			if t, ok := c.typeOf(b, env); ok && t != "int" {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("cannot slice with %s: bounds must be int", t), "[:]", CodeTypeMismatch))
				return
			}
		}
		s.T = "str"
	case "call":
		if isStoreOp(s.Fname) {
			c.checkStoreOp(s, line, env)
			return
		}
		if isDecParts(s.Fname) {
			c.checkDecParts(s, line, env)
			return
		}
		sig := c.callee(s.Fname)
		if sig == nil {
			return
		}
		slots, berr := bindSlots(s.Fname, s.Args, sig.params)
		if berr != nil {
			c.out = append(c.out, spanDiag(c.text, line, "error", berr.Error(), s.Fname, CodeBadBinding))
			// Still check the argument expressions themselves so one
			// bad vector never hides nested errors inside the args.
			for _, a := range s.Args {
				c.value(a.V, "", line, env, "call "+s.Fname+" arg")
			}
			return
		}
		for i, a := range s.Args {
			p := sig.params[slots[i]]
			want, label := p[1], "call "+s.Fname+" arg "+p[0]
			c.value(a.V, "", line, env, label)
			if c.knownType(want) {
				if got, ok := c.typeOf(a.V, env); ok && !sameType(got, want) {
					c.mismatch(line, label, got, want, tokenOf(a.V))
				}
			}
		}
	case "ctor":
		c.checkCtor(s, want, line, env, where)
	case "list":
		// a36 S1: bare [...] is script rows, never a value. Given
		// tables consume the outer list structurally (checkStubs),
		// so this arm only fires in real value positions, where the
		// fix is always an explicitly typed Seq<T>[...] literal.
		c.out = append(c.out, spanDiag(c.text, line, "error",
			"bare [...] is script rows, not a sequence value: write Seq<T>[...] with an explicit element type", "[", CodeSeqLiteral))
		for _, it := range s.Items {
			c.value(it, "", line, env, where)
		}
	case "fnref":
		c.checkFnref(s, line, env, where)
	case "seqlit":
		// a36 S1: values and element checking are one admission
		// boundary. Every member checks against the written element
		// type with no inference and no emptiness waiver; exec is
		// untouched, so seals inside executable literals keep the
		// a15 file-ownership rule (CAN6004) while test and script
		// data keep naming any declared brand.
		if _, nested := seqElemName(s.Elem); nested || !c.knownType(s.Elem) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("unknown type %s in Seq literal", s.Elem), "Seq", CodeUnknownType))
			for _, it := range s.Items {
				c.value(it, "", line, env, where)
			}
			return
		}
		if c.typeHasFn(s.Elem) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("sequence element %s of Seq literal contains a function value: sequence elements are data-only", s.Elem), "Seq", CodeFnContainment))
			for _, it := range s.Items {
				c.value(it, "", line, env, where)
			}
			return
		}
		st := "Seq<" + s.Elem + ">"
		for _, it := range s.Items {
			switch it.Kind {
			case "call", "exchange", "wild", "list":
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("Seq literal takes values, not %s", it.Kind), "Seq", CodeTypeMismatch))
				c.value(it, "", line, env, where)
				continue
			}
			before := len(c.out)
			c.value(it, "", line, env, where)
			if got, ok := c.typeOf(it, env); ok && !sameType(got, s.Elem) {
				c.mismatch(line, where, got, s.Elem, tokenOf(it))
			} else if !ok && len(c.out) == before {
				// Outcome constructors are not element data, even if the
				// literal's declared Seq type matches the success annotation.
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("Seq literal needs a value of type %s", s.Elem), tokenOf(it), CodeTypeMismatch))
			}
		}
		if want != "" && want != st {
			c.mismatch(line, where, st, want, "Seq")
		}
	default:
		for _, a := range s.Args {
			c.value(a.V, "", line, env, where)
		}
	}
}

// checkStoreOp validates a store call against its cell: positional
// args only (one spelling), the cell first, and a put value of
// exactly the cell type. Unknown cells belong to checkEffects and
// stay silent here, like unknown callees.
func (c *tycker) checkStoreOp(s *Small, line int, env map[string]string) {
	for _, a := range s.Args {
		if a.HasName {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("call %s takes positional args", s.Fname), s.Fname, CodeTypeMismatch))
			return
		}
	}
	cell, ok := storeCellName(s)
	if !ok {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("call %s takes the cell first", s.Fname), s.Fname, CodeTypeMismatch))
		return
	}
	want, known := c.cells[cell]
	if !known {
		return
	}
	count := 1
	if s.Fname == "state__put" {
		count = 2
	}
	if len(s.Args) != count {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("call %s takes %d args", s.Fname, count), s.Fname, CodeTypeMismatch))
		return
	}
	if s.Fname == "state__put" {
		label := fmt.Sprintf("call %s value", s.Fname)
		c.value(s.Args[1].V, "", line, env, label)
		if got, ok := c.typeOf(s.Args[1].V, env); ok && got != want {
			c.mismatch(line, label, got, want, tokenOf(s.Args[1].V))
		}
	}
}

// checkDecParts validates the decimal observation kernel: positional
// args only (one spelling), exactly one arg, and a dec operand.
// Anything else is CAN6003, the operand-rule family.
func (c *tycker) checkDecParts(s *Small, line int, env map[string]string) {
	for _, a := range s.Args {
		if a.HasName {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("call %s takes positional args", s.Fname), s.Fname, CodeTypeMismatch))
			return
		}
	}
	if len(s.Args) != 1 {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("call %s takes 1 arg", s.Fname), s.Fname, CodeTypeMismatch))
		return
	}
	label := fmt.Sprintf("call %s value", s.Fname)
	c.value(s.Args[0].V, "", line, env, label)
	if got, ok := c.typeOf(s.Args[0].V, env); ok && got != "dec" {
		c.mismatch(line, label, got, "dec", tokenOf(s.Args[0].V))
	}
}

// nodeStoreArms types a store-op match: a get payload binds its var
// to the cell type through the synthetic value field; a put yields
// empty Ok, so its var binds nothing checkable. The cell type flows
// from the declaration, never from inference.
func (c *tycker) nodeStoreArms(n *Node, env map[string]string, want string) {
	cell, ok := storeCellName(n.Scruts[0])
	t, known := "", false
	if ok {
		t, known = c.cells[cell]
	}
	for _, a := range n.Arms {
		p := a.Pats[0]
		env2 := map[string]string{}
		for k, v := range env {
			env2[k] = v
		}
		armWant := ""
		if p.Kind == "variant" && p.Name == "Ok" && p.Var != "" {
			if n.Scruts[0].Fname == "state__get" && known {
				c.rejectKernelWholeOk(p, a.Line)
				env2[p.Var] = "cell:" + t
			} else if n.Scruts[0].Fname == "state__put" {
				c.rejectKernelWholeOk(p, a.Line)
				env2[p.Var] = "empty-ok"
			} else {
				env2[p.Var] = ""
			}
			armWant = want
		}
		if p.Kind == "variantWild" && p.Name == "Ok" {
			armWant = want
		}
		c.node(a.Rhs, env2, armWant)
	}
}

// nodePartsArms types a dec-observation match: the Ok payload binds
// its var to the synthetic parts shape, whose only fields are the
// int coefficient and scale. The shape is fixed by the kernel, never
// by inference, so no declared record is consulted.
func (c *tycker) nodePartsArms(n *Node, env map[string]string, want string) {
	for _, a := range n.Arms {
		p := a.Pats[0]
		env2 := map[string]string{}
		for k, v := range env {
			env2[k] = v
		}
		armWant := ""
		if p.Kind == "variant" && p.Name == "Ok" && p.Var != "" {
			c.rejectKernelWholeOk(p, a.Line)
			env2[p.Var] = "parts"
			armWant = want
		}
		if p.Kind == "variantWild" && p.Name == "Ok" {
			armWant = want
		}
		c.node(a.Rhs, env2, armWant)
	}
}

// checkCtor validates a construction against its contract: Ok against
// the wanted success shape, dotted errors against their ErrorDecl, named
// records against their TypeDecl. want="" (error-producing positions)
// still checks error fields; unknown contracts belong to other codes,
// so undeclared kinds are skipped, never double-reported. Positional
// args resolve here by mutation (a92), so later phases see names.
func (c *tycker) checkCtor(s *Small, want string, line int, env map[string]string, where string) {
	name := s.Ctor
	var fields [][2]string
	label := where
	if name == "Ok" {
		if len(s.TypeArgs) > 0 {
			c.checkWholeOk(s, want, line, env)
			return
		}
		if want == "" || strings.HasPrefix(want, "err:") {
			for _, a := range s.Args {
				c.value(a.V, "", line, env, where)
			}
			return
		}
		rec, ok := successFields(want, c.recs, c.variants[want] || c.brands[want])
		if !ok {
			// Unsupported/unknown return types have their own diagnostics.
			return
		}
		fields, label = rec, "Ok"
	} else if strings.Contains(name, ".") {
		ed, ok := c.errs[name]
		if !ok {
			for _, a := range s.Args {
				c.value(a.V, "", line, env, where)
			}
			return
		}
		fields, label = ed, name
	} else if name == "Bytes" {
		// a45 S1: literal-only construction. Exactly one positional
		// argument holding an explicit Seq<int> literal whose members
		// are integer-literal AST nodes in 0..255 (big.Int
		// comparison, no narrowing). No conversion, no inference.
		// These are compile diagnostics, not error outcomes.
		if len(s.Args) != 1 || s.Args[0].HasName {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				"Bytes takes one Seq<int> literal with integer members 0..255", name, CodeBytesLiteral))
			for _, a := range s.Args {
				c.value(a.V, "", line, env, where)
			}
			return
		}
		arg := s.Args[0].V
		before := len(c.out)
		c.value(arg, "", line, env, where)
		if len(c.out) != before {
			// The child's own rule (bare list, unbound name,
			// element type, member shape) already explains the
			// failure; no derivative diagnostic.
			return
		}
		if arg.Kind != "seqlit" || arg.Elem != "int" {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				"Bytes takes one Seq<int> literal with integer members 0..255", name, CodeBytesLiteral))
			return
		}
		s.T = "Bytes"
		for i, m := range arg.Items {
			if m.Kind != "int" || m.Num == nil {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("Bytes member %d is not an integer literal: Bytes takes literal members 0..255", i), name, CodeBytesLiteral))
				continue
			}
			if m.Num.Sign() < 0 || m.Num.Cmp(big.NewInt(256)) >= 0 {
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("Bytes member %d is %s: byte members are 0..255", i, m.Num.String()), name, CodeBytesElementRange))
			}
		}
		if want != "" && want != "Bytes" {
			c.mismatch(line, where, "Bytes", want, name)
		}
		return
	} else if cc, ok := c.cases[name]; ok {
		// a74: a qualified case constructs its parent variant
		// with exactly the declared case fields. The shared
		// field loop below checks unknown, repeated, missing,
		// and mistyped fields; the constructor's nominal type
		// is the parent, never the payload shape.
		fields, label = cc.fields, name
		// a75: the nominal type annotates outward like Bytes, so
		// emit's typed dispatch reads the parent for scrutinees
		// built inline. Anything else ignores constructor T.
		s.T = cc.parent
		if want != "" && want != cc.parent {
			c.mismatch(line, where, cc.parent, want, name)
		}
	} else {
		rec, ok := c.recs[name]
		if !ok {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("unknown record %s in construction", name), name, CodeUnknownType))
			return
		}
		fields, label = rec, name
		if want != "" && want != name {
			c.mismatch(line, where, name, want, name)
		}
	}
	byName := map[string]string{}
	for _, f := range fields {
		byName[f[0]] = f[1]
	}
	// a92: positional construction. An unnamed arg at list
	// index i claims field i (the calls rule, bindSlots —
	// not the stricter test-row prefix rule). Resolution
	// assigns names by mutation, so every later phase —
	// runs, proofs, emit — sees one named shape.
	// Idempotent: resolved args are named, so a second run
	// is a no-op. Two faults are CAN6003: a positional past
	// the arity, and a positional landing on a named-claimed
	// field. Faulted args stay unnamed and are skipped
	// below: resolution-owned, already reported.
	claimed := map[int]bool{}
	for _, a := range s.Args {
		if !a.HasName {
			continue
		}
		for j, f := range fields {
			if f[0] == a.Name {
				claimed[j] = true
			}
		}
	}
	bound := map[int]bool{}
	for i := range s.Args {
		a := &s.Args[i]
		if a.HasName {
			continue
		}
		if i >= len(fields) {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s takes %d args for %d fields", label, len(s.Args), len(fields)), label, CodeTypeMismatch))
			continue
		}
		if claimed[i] {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s supplies field %s twice", label, fields[i][0]), fields[i][0], CodeTypeMismatch))
			continue
		}
		claimed[i] = true
		a.Name = fields[i][0]
		a.HasName = true
		bound[i] = true
	}
	seenArg := map[string]bool{}
	for i, a := range s.Args {
		if !a.HasName {
			continue
		}
		if seenArg[a.Name] {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s repeats field %s", label, a.Name), a.Name, CodeTypeMismatch))
		}
		seenArg[a.Name] = true
		ft, ok := byName[a.Name]
		if !ok {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s has no field %s", label, a.Name), a.Name, CodeTypeMismatch))
			c.value(a.V, "", line, env, where)
			continue
		}
		flabel := fmt.Sprintf("%s field %s", label, a.Name)
		c.value(a.V, "", line, env, flabel)
		if c.knownType(ft) {
			if got, ok := c.typeOf(a.V, env); ok && !sameType(got, ft) {
				// a92: a bound name never appears in
				// source, so the squiggle covers the
				// offending value instead of nothing.
				tok := a.Name
				if bound[i] {
					tok = tokenOf(a.V)
				}
				c.mismatch(line, flabel, got, ft, tok)
			} else if !ok && name == "Ok" && c.valueSuccess(want) {
				// Outcome constructors have no data type. They must not
				// sneak into a value payload merely
				// because legacy want-free positions leave them untyped.
				kind := "type"
				if scalarSuccess(want) {
					kind = "scalar"
				}
				if c.variants[want] {
					kind = "variant"
				}
				c.out = append(c.out, spanDiag(c.text, line, "error",
					fmt.Sprintf("%s needs a value of %s %s", flabel, kind, want), tokenOf(a.V), CodeTypeMismatch))
			}
		}
	}
	for _, f := range fields {
		found := false
		for _, a := range s.Args {
			if a.Name == f[0] {
				found = true
				break
			}
		}
		if !found {
			c.out = append(c.out, spanDiag(c.text, line, "error",
				fmt.Sprintf("%s is missing field %s", label, f[0]), f[0], CodeTypeMismatch))
		}
	}
}

// node walks a body threading the wanted type to final-value positions:
// Ok arms and value matches produce the function's return, error arms
// produce errors (checked against their decl, want-free). Bindings copy
// the map down; shadowing stays undiagnosed (open question).
func (c *tycker) node(n *Node, env map[string]string, want string) {
	if n == nil {
		return
	}
	if !n.IsMatch {
		// Non-record values are not outcomes. Bare values must not
		// bypass the Ok envelope just because their type fits.
		if f := c.prog.Fns[c.fn]; f != nil && c.valueSuccess(f.Ret) {
			if s := n.Small; s != nil {
				_, isError := c.errs[s.Ctor]
				if !(s.Kind == "ctor" && (s.Ctor == "Ok" || isError)) {
					kind := "value"
					if scalarSuccess(f.Ret) {
						kind = "scalar"
					}
					if c.variants[f.Ret] {
						kind = "variant"
					}
					c.out = append(c.out, spanDiag(c.text, n.Line, "error",
						kind+" return must use Ok(value) or a declared error", tokenOf(s), CodeTypeMismatch))
				}
			}
		}
		w := want
		if w == "" && n.Small != nil && n.Small.Kind == "ctor" && n.Small.Ctor == "Ok" {
			// a92: an Ok arm top is return-positioned even
			// under error arms (want-free by threading), so
			// check it against the fn success shape and
			// positional args bind. typeOf stays silent for
			// Ok, so no comparison is added: anything else
			// keeps want-free checking.
			if f, ok := c.prog.Fns[c.fn]; ok {
				if _, ok := successFields(f.Ret, c.recs, c.variants[f.Ret] || c.brands[f.Ret]); ok {
					w = f.Ret
				}
			}
		}
		c.value(n.Small, w, n.Line, env, "returns")
		return
	}
	if n.Kind == MatchInvoke {
		c.checkInvoke(n, env, want)
		return
	}
	// Every scrutinee is valued, so a bad reference in any slot is
	// caught here exactly as it is at runtime. Only call matches take
	// arm bindings, and they always carry one scrutinee.
	for _, s := range n.Scruts {
		c.value(s, "", n.Line, env, "match scrutinee")
	}
	if n.Kind == MatchCall {
		ns := n.Scruts[0]
		if isStoreOp(ns.Fname) {
			c.nodeStoreArms(n, env, want)
			return
		}
		if isDecParts(ns.Fname) {
			c.nodePartsArms(n, env, want)
			return
		}
		sig := c.callee(ns.Fname)
		for _, a := range n.Arms {
			p := a.Pats[0]
			env2 := map[string]string{}
			for k, v := range env {
				env2[k] = v
			}
			armWant := ""
			if p.Kind == "variant" && p.Var != "" {
				if p.Name == "Ok" {
					if sig != nil && c.knownType(sig.ret) {
						env2[p.Var] = c.patternSuccessType(p, sig.ret, a.Line)
					} else {
						env2[p.Var] = ""
					}
					armWant = want
				} else if _, ok := c.errs[p.Name]; ok {
					env2[p.Var] = "err:" + p.Name
				} else {
					env2[p.Var] = ""
				}
			}
			if p.Kind == "variantWild" && p.Name == "Ok" {
				armWant = want
			}
			c.node(a.Rhs, env2, armWant)
		}
		return
	}
	// a75: variant elimination. A single scrutinee resolving to a
	// variant parent takes the case path; a case-typed scrutinee
	// (an already-eliminated binder) is refused; any variant slot
	// in a multi match refuses the second scrutinee. Everything
	// else keeps the legacy bool/str/wild rules, with case-shaped
	// patterns rejected as non-variant matches.
	variantSlot, parent := -1, ""
	for i, s := range n.Scruts {
		if st, ok := c.typeOf(s, env); ok && c.variants[st] {
			variantSlot, parent = i, st
			break
		}
	}
	if variantSlot >= 0 && len(n.Scruts) > 1 {
		c.out = append(c.out, spanDiag(c.text, n.Line, "error",
			fmt.Sprintf("match over %s takes exactly one scrutinee", parent), "match", CodeBadArmKind))
		for _, a := range n.Arms {
			c.node(a.Rhs, env, want)
		}
		return
	}
	if len(n.Scruts) == 1 {
		if st, ok := c.typeOf(n.Scruts[0], env); ok && c.variants[st] {
			c.nodeVariantArms(n, env, want, st)
			return
		}
		if st, ok := c.typeOf(n.Scruts[0], env); ok && isCaseType(c, st) {
			c.out = append(c.out, spanDiag(c.text, n.Line, "error",
				fmt.Sprintf("match over %s eliminates unions, not cases: match the union scrutinee", st), "match", CodeBadArmKind))
			for _, a := range n.Arms {
				c.node(a.Rhs, env, want)
			}
			return
		}
	}
	for _, a := range n.Arms {
		for _, p := range a.Pats {
			if p.isCase() {
				c.out = append(c.out, spanDiag(c.text, a.Line, "error",
					fmt.Sprintf("variant pattern on a non-variant match"), "on", CodeVariantOnVal))
				break
			}
		}
		c.node(a.Rhs, env, want)
	}
}

// checkInvoke checks one `match invoke cb with n` against the
// resolved callable signature: the target names an Fn-typed value
// in scope, the argument matches the input type, and arm binders
// thread the success payload and error kinds exactly like a call
// match. Unbound names report through the scrutinee check, while
// known non-callables (including shadowed binders) reuse the
// mismatch code (JEV Q1). Exhaustiveness belongs to the proof.
func (c *tycker) checkInvoke(n *Node, env map[string]string, want string) {
	for _, s := range n.Scruts {
		c.value(s, "", n.Line, env, "match scrutinee")
	}
	if n.InvokeArg != nil {
		c.value(n.InvokeArg, "", n.Line, env, "invoke argument")
	}
	name := ""
	if len(n.Scruts) == 1 {
		if s := n.Scruts[0]; s.Kind == "ref" && len(s.Ref) == 1 {
			name = s.Ref[0]
		}
	}
	sig := n.invokeSig
	if name != "" {
		t, ok := env[name]
		if !ok && constNameRe.MatchString(name) {
			if decl, found := lookupConst(c.prog, name); found {
				t, ok = decl.Type, true
			}
		}
		// A resolved signature still loses to a shadowed
		// binder: the name in scope is what executes. The
		// proof keeps reading the parameter's kinds (it has
		// no env), so only the checker reports the shadow.
		if ok && t != "" && (sig == nil || t != sig.head) {
			c.out = append(c.out, spanDiag(c.text, n.Line, "error",
				fmt.Sprintf("invoke target %s has type %s: want a function value", name, strings.TrimPrefix(t, "err:")), name, CodeTypeMismatch))
		}
	}
	if sig != nil && n.InvokeArg != nil && c.knownType(sig.in) {
		if got, ok := c.typeOf(n.InvokeArg, env); ok && !sameType(got, sig.in) {
			c.mismatch(n.Line, "invoke "+name+" argument", got, sig.in, tokenOf(n.InvokeArg))
		}
	}
	for _, a := range n.Arms {
		p := a.Pats[0]
		env2 := map[string]string{}
		for k, v := range env {
			env2[k] = v
		}
		armWant := ""
		if p.Kind == "variant" && p.Var != "" {
			if p.Name == "Ok" {
				if sig != nil && c.knownType(sig.ret) {
					env2[p.Var] = c.patternSuccessType(p, sig.ret, a.Line)
				} else {
					env2[p.Var] = ""
				}
				armWant = want
			} else if _, ok := c.errs[p.Name]; ok {
				env2[p.Var] = "err:" + p.Name
			} else {
				env2[p.Var] = ""
			}
		}
		if p.Kind == "variantWild" && p.Name == "Ok" {
			armWant = want
		}
		c.node(a.Rhs, env2, armWant)
	}
}

// nodeVariantArms checks one variant elimination (a75): every arm
// carries exactly one case pattern of the scrutinee's union, each
// case exactly once, no other pattern kinds. Binders enter the arm
// env typed as their qualified case, so projection resolves through
// the case's own fields. Missing cases report one diagnostic per
// case, mirroring the call-match missing-outcome shape.
func (c *tycker) nodeVariantArms(n *Node, env map[string]string, want, parent string) {
	seen := map[string]bool{}
	for _, a := range n.Arms {
		if len(a.Pats) != 1 {
			c.out = append(c.out, spanDiag(c.text, a.Line, "error",
				fmt.Sprintf("match arm has %d patterns; this match has 1 scrutinee", len(a.Pats)), "on", CodeBadArmKind))
			c.node(a.Rhs, env, want)
			continue
		}
		p := a.Pats[0]
		if !p.isCase() {
			desc, tok := patDesc(p)
			c.out = append(c.out, spanDiag(c.text, a.Line, "error",
				fmt.Sprintf("match over %s takes case arms only, not %s", parent, desc), tok, CodeBadArmKind))
			c.node(a.Rhs, env, want)
			continue
		}
		cc, ok := c.cases[p.Name]
		if !ok || cc.parent != parent {
			c.out = append(c.out, spanDiag(c.text, a.Line, "error",
				fmt.Sprintf("stale match arm %s", p.Name), p.Name, CodeStaleArm))
			c.node(a.Rhs, env, want)
			continue
		}
		if seen[p.Name] {
			c.out = append(c.out, spanDiag(c.text, a.Line, "error",
				fmt.Sprintf("duplicate match arm %s", p.Name), p.Name, CodeBadArmKind))
		}
		seen[p.Name] = true
		env2 := map[string]string{}
		for k, v := range env {
			env2[k] = v
		}
		if p.Kind == "variant" && p.Var != "" && p.Var != "_" {
			env2[p.Var] = p.Name
		}
		c.node(a.Rhs, env2, want)
	}
	var missing []string
	for q, cc := range c.cases {
		if cc.parent == parent && !seen[q] {
			missing = append(missing, q)
		}
	}
	slices.Sort(missing)
	for _, q := range missing {
		c.out = append(c.out, spanDiag(c.text, n.Line, "error",
			fmt.Sprintf("non-exhaustive match over %s, missing %s", parent, q), "match", CodeMissingArm))
	}
}

// checkStubs validates given-table outcomes against the callee
// contract: Ok shapes against its Ret, error fields against their
// decl. Kinds were already checked; undeclared kinds are skipped here.
// Stubs evaluate in the test's argument environment, so param names
// resolve here exactly as they do at runtime.
func (c *tycker) checkStubs(fn *FnDecl, text string, env map[string]string) {
	for _, m := range matchNodes(fn.Body) {
		if m.Kind != MatchCall || m.Given == nil {
			continue
		}
		ms := m.Scruts[0]
		sig := c.callee(ms.Fname)
		if sig == nil {
			continue
		}
		for key, sm := range m.Given {
			if sm == nil {
				continue
			}
			line := locateLineFrom(text, key+" =>", m.Line, m.Line)
			where := fmt.Sprintf("script %s for %s", key, ms.Fname)
			var items []*Small
			if sm.Kind == "list" {
				items = sm.Items
			} else {
				items = []*Small{sm}
			}
			for _, it := range items {
				if it.Kind != "exchange" {
					continue // check.go owns the row shape
				}
				if it.Outcome.Kind == "ctor" && it.Outcome.Ctor == "Ok" {
					c.checkCtor(it.Outcome, sig.ret, line, env, where)
				} else {
					c.value(it.Outcome, "", line, env, where)
				}
				// Expected args type against callee params, like test
				// args type against function params. Names themselves
				// are proven dynamically at each hit.
				for _, a := range it.Args {
					c.value(a.V, "", line, env, "exchange arg "+a.Name+" for "+ms.Fname)
					want := ""
					for _, p := range sig.params {
						if p[0] == a.Name {
							want = p[1]
						}
					}
					if want != "" && c.knownType(want) {
						if got, ok := c.typeOf(a.V, env); ok && !sameType(got, want) {
							c.mismatch(line, "exchange arg "+a.Name+" for "+ms.Fname, got, want, tokenOf(a.V))
						}
					}
				}
			}
		}
	}
}

// checkTypes enforces annotations on one function: params and ret name
// known types, tests match the signature, stubs match the callee, and
// the body checks internally with the return threaded to Ok positions.
func checkTypes(fn *FnDecl, prog *Program, text string) []Diag {
	c := newTycker(prog, text, fn.Name)
	for _, p := range fn.Params {
		if !c.knownType(p[1]) {
			c.out = append(c.out, spanDiag(text, fn.Line, "error",
				fmt.Sprintf("unknown type %s in param %s", p[1], p[0]), p[0], CodeUnknownType))
		}
		c.out = append(c.out, c.fnHeadDiags(p[1], "param "+p[0], fn.Line, p[0])...)
		c.out = append(c.out, c.fnSeqDiags(p[1], "param "+p[0], fn.Line, p[0])...)
	}
	if !c.knownType(fn.Ret) {
		c.out = append(c.out, spanDiag(text, fn.Line, "error",
			fmt.Sprintf("unknown type %s in returns", fn.Ret), fn.Ret, CodeUnknownType))
	}
	c.out = append(c.out, c.fnHeadDiags(fn.Ret, "returns", fn.Line, fn.Ret)...)
	c.out = append(c.out, c.successSeqDiags(fn.Ret, fn.Line)...)
	env := map[string]string{}
	for _, p := range fn.Params {
		env[p[0]] = p[1]
	}
	for _, t := range fn.Tests {
		for _, a := range t.Args {
			want := ""
			for _, p := range fn.Params {
				if p[0] == a.Name {
					want = p[1]
				}
			}
			label := fmt.Sprintf("test %s arg %s", t.Name, a.Name)
			c.value(a.V, "", t.Line, env, label)
			if want != "" && c.knownType(want) {
				if got, ok := c.typeOf(a.V, env); ok && !sameType(got, want) {
					c.mismatch(t.Line, label, got, want, tokenOf(a.V))
				}
			}
		}
		exp := t.Expected
		label := fmt.Sprintf("test %s expects", t.Name)
		switch {
		case exp.Kind == "ctor" && exp.Ctor == "Ok":
			if c.knownType(fn.Ret) {
				c.checkCtor(exp, fn.Ret, t.Line, env, label)
			} else {
				c.value(exp, "", t.Line, env, label)
			}
		case exp.Kind == "ctor" && strings.Contains(exp.Ctor, "."):
			// Complete error expectations carry the payload and
			// check against the error decl, like constructions.
			if _, ok := c.errs[exp.Ctor]; !ok {
				c.out = append(c.out, spanDiag(text, t.Line, "error",
					fmt.Sprintf("unknown error kind %s in %s", exp.Ctor, label), exp.Ctor, CodeUnknownType))
			} else {
				c.checkCtor(exp, "", t.Line, env, label)
			}
		case exp.Kind == "ref" && len(exp.Ref) > 1:
			// Bare error kinds prove nothing about the payload (a12):
			// expectations must construct the complete error value.
			name := strings.Join(exp.Ref, ".")
			d := spanDiag(text, t.Line, "error",
				fmt.Sprintf("test %s expects bare error kind %s: write the complete error value", t.Name, name), exp.Ref[len(exp.Ref)-1], CodeBareErrorKind)
			d.Found = name
			parts := make([]string, 0, len(c.errs[name]))
			for _, f := range c.errs[name] {
				parts = append(parts, fmt.Sprintf("%s = <%s>", f[0], f[1]))
			}
			d.Expected = name + "(" + strings.Join(parts, ", ") + ")"
			d.Hint = "write the complete error value with all fields"
			c.out = append(c.out, d)
		case exp.Kind == "float":
			c.value(exp, "", t.Line, env, label)
		default:
			c.out = append(c.out, spanDiag(text, t.Line, "error",
				fmt.Sprintf("%s %s: write Ok(...) or an error kind", label, exp.Kind), t.Name, CodeTypeMismatch))
		}
	}
	c.checkStubs(fn, text, env)
	ret := ""
	if c.knownType(fn.Ret) {
		ret = fn.Ret
	}
	// Bodies are executable positions: the seal rule applies from
	// here on. Everything above checked data (tests, scripts).
	c.exec = true
	c.node(fn.Body, env, ret)
	return c.out
}

// checkExternSig validates a foreign import's contract: params and ret
// name known types and remain data-only. Non-record successes use the
// same one-value envelope as source calls; host execution is still trusted.
func checkExternSig(ex *ExternDecl, prog *Program, text string) []Diag {
	c := newTycker(prog, text, ex.Name)
	for _, p := range ex.Params {
		if !c.knownType(p[1]) {
			c.out = append(c.out, spanDiag(text, ex.Line, "error",
				fmt.Sprintf("unknown type %s in param %s", p[1], p[0]), p[0], CodeUnknownType))
		}
		c.out = append(c.out, c.fnHeadDiags(p[1], "param "+p[0], ex.Line, p[0])...)
		if c.typeHasFn(p[1]) {
			c.out = append(c.out, spanDiag(text, ex.Line, "error",
				fmt.Sprintf("param %s of extern %s contains a function value: extern signatures are data-only", p[0], ex.Name), p[0], CodeFnContainment))
		}
	}
	c.out = append(c.out, c.fnHeadDiags(ex.Ret, "returns", ex.Line, ex.Ret)...)
	if c.typeHasFn(ex.Ret) {
		c.out = append(c.out, spanDiag(text, ex.Line, "error",
			fmt.Sprintf("returns of extern %s contains a function value: extern signatures are data-only", ex.Name), ex.Ret, CodeFnContainment))
	}
	if !c.knownType(ex.Ret) {
		c.out = append(c.out, spanDiag(text, ex.Line, "error",
			fmt.Sprintf("unknown type %s in returns", ex.Ret), ex.Ret, CodeUnknownType))
	}
	c.out = append(c.out, c.successSeqDiags(ex.Ret, ex.Line)...)
	// a12: extern manifests are upper bounds like function emits —
	// every entry must name a declared error.
	for _, e := range ex.Emits {
		if _, ok := c.errs[e]; !ok {
			c.out = append(c.out, spanDiag(text, ex.Line, "error",
				fmt.Sprintf("extern %s declares unknown error kind %s in emits", ex.Name, e), e, CodeUnknownKind))
		}
	}
	return c.out
}

// checkDeclFields validates record and error field annotations: every
// field names a known type. Uses of an undeclared field type stay
// silent (typeOf suppresses them), so the declaration owns the error.
func checkDeclFields(name string, fields [][2]string, line int, prog *Program, text string, allowFn bool) []Diag {
	c := newTycker(prog, text, name)
	var out []Diag
	for _, f := range fields {
		if elem, ok := seqElemName(f[1]); ok && c.variants[elem] {
			// Decision 6: variant sequences are deferred.
			// No existing source can contain one (variants
			// are new), so this rejects new surface only.
			out = append(out, spanDiag(text, line, "error",
				fmt.Sprintf("Seq<%s> is deferred in this cut: variant sequences are not admitted", elem), f[0], CodeUnknownType))
			continue
		}
		if !c.knownType(f[1]) {
			out = append(out, spanDiag(text, line, "error",
				fmt.Sprintf("unknown type %s in field %s", f[1], f[0]), f[0], CodeUnknownType))
		}
		out = append(out, c.fnHeadDiags(f[1], "field "+f[0], line, f[0])...)
		if !allowFn && c.typeHasFn(f[1]) {
			out = append(out, spanDiag(text, line, "error",
				fmt.Sprintf("field %s of %s contains a function value: this field position is data-only", f[0], name), f[0], CodeFnContainment))
		} else if allowFn {
			out = append(out, c.fnSeqDiags(f[1], "field "+f[0], line, f[0])...)
		}
	}
	return out
}

// checkBrandDecl enforces the v0 brand boundary: string-backed only.
// int-backed brands wait for a second underlying type with something
// to prove about it. A seals_from source must name a declared brand
// from the same module (a26): promotion authority is explicit and
// owner-local, never inferred across files.
func checkBrandDecl(b *BrandDecl, prog *Program, text string) []Diag {
	if b.Under != "str" {
		return []Diag{spanDiag(text, b.Line, "error",
			fmt.Sprintf("brand %s wraps %s: v0 brands wrap str only", b.Name, b.Under), b.Under, CodeTypeMismatch)}
	}
	var out []Diag
	for _, src := range b.SealsFrom {
		srcFile, ok := prog.BrandFile[src]
		if !ok {
			out = append(out, spanDiag(text, b.Line, "error",
				fmt.Sprintf("brand %s seals_from unknown brand %s", b.Name, src), src, CodeUnknownType))
			continue
		}
		if dstFile, ok := prog.BrandFile[b.Name]; ok && srcFile != dstFile {
			out = append(out, spanDiag(text, b.Line, "error",
				fmt.Sprintf("brand %s seals_from %s from %s: promotions stay inside one module", b.Name, src, srcFile), src, CodeSealForeign))
		}
	}
	return out
}

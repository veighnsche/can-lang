package main

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// a10: ints emit as bigint (unbounded, exact) and decs as strings
// carrying canonical digits (exact via the $canDec helpers below).
// The old number mapping was lossy (0.1+0.2) and is gone.
var tsBase = map[string]string{"str": "string", "int": "bigint", "bool": "boolean", "dec": "string", "Bytes": "Uint8Array"}

func tsType(t string) (string, error) {
	return tsTypeB(t, nil, nil, nil, nil)
}

// recordShapes indexes declared record fields by type name, first
// wins across modules, matching the checker and the evaluator.
func variantShapes(mods []*Module) map[string]*VariantDecl {
	variants := map[string]*VariantDecl{}
	for _, m := range mods {
		for _, d := range m.Decls {
			if vd, ok := d.(*VariantDecl); ok {
				if _, seen := variants[vd.Name]; !seen {
					variants[vd.Name] = vd
				}
			}
		}
	}
	return variants
}

// errorShapes indexes declared error fields by kind name, first wins
// across modules, matching the checker and the evaluator.
func errorShapes(mods []*Module) map[string][][2]string {
	errs := map[string][][2]string{}
	for _, b := range builtinErrorDecls() {
		errs[b.Name] = b.Fields
	}
	for _, m := range mods {
		for _, d := range m.Decls {
			if ed, ok := d.(*ErrorDecl); ok {
				if _, seen := errs[ed.Name]; !seen {
					errs[ed.Name] = ed.Fields
				}
			}
		}
	}
	return errs
}

// tsTypeB maps an can annotation to TS, erasing brands to their
// underlying type. Branding is proof, not runtime: the emit forgets it.
// Declared record names map to their emitted TS type of the same name.
// Declared variant parents (a74) map to their emitted union type of
// the same name, so variant-typed fields and params reference it.
// Callable heads map to unary closure types over the same
// record/error union builders as named functions; errs carries the
// error shapes for the union members (nil fails a head closed).
func tsTypeB(t string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	if out, ok := tsBase[t]; ok {
		return out, nil
	}
	// a36 S1: sequences emit as arrays over the mapped element
	// type. Brands erase through the same rule as scalars, so
	// Seq<M__B> is string[] without hard-coding brand names here.
	if elem, ok := seqElemName(t); ok {
		if variants[elem] != nil || typeContainsFn(elem, recs) {
			return "", fmt.Errorf("cannot map can type to TS: unsupported sequence element %s", elem)
		}
		inner, err := tsTypeB(elem, brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		return inner + "[]", nil
	}
	if a, r, e, ok := fnTypeShape(t); ok {
		return tsFnType(a, r, e, brands, recs, variants, errs)
	}
	if u, ok := brands[t]; ok {
		if out, ok := tsBase[u]; ok {
			return out, nil
		}
	}
	if _, ok := recs[t]; ok {
		return t, nil
	}
	if _, ok := variants[t]; ok {
		return t, nil
	}
	return "", fmt.Errorf("cannot map can type to TS: %s", t)
}

// tsFnType renders one callable head as a unary closure type: the
// mapped input, then the flattened Ok/error union the thunk
// returns, built by the same member builders as named functions.
// Ok fields sort exactly like fnResultUnion so a temporary
// annotated from either side renders identically.
func tsFnType(a, r, e string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	if typeContainsFn(a, recs) || typeContainsFn(r, recs) {
		return "", fmt.Errorf("cannot map can type to TS: Fn input and success must be data-only")
	}
	in, err := tsTypeB(a, brands, recs, variants, errs)
	if err != nil {
		return "", err
	}
	fields, ok := successFields(r, recs, variants[r] != nil || brands[r] != "")
	if !ok {
		return "", fmt.Errorf("cannot map can type to TS: Fn success %s is not a supported success type", r)
	}
	type kv struct{ k, v string }
	var kvs []kv
	for _, f := range fields {
		t, err := tsTypeB(f[1], brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		kvs = append(kvs, kv{f[0], t})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })
	fs := ""
	for _, p := range kvs {
		fs += "; " + p.k + ": " + p.v
	}
	union := okMember(fs)
	raw := strings.TrimSpace(e[1 : len(e)-1])
	if raw != "" {
		for _, k := range strings.Split(raw, ",") {
			kind := strings.TrimSpace(k)
			efs, ok := errs[kind]
			if !ok {
				return "", fmt.Errorf("cannot map can type to TS: Fn error %s is not declared", kind)
			}
			mem, err := tsErrFieldsMember(kind, efs, brands, recs, variants, errs)
			if err != nil {
				return "", err
			}
			union += " | " + mem
		}
	}
	return fmt.Sprintf("(input: %s) => %s", in, union), nil
}

// tsTag is the outcome discriminator key in emitted TypeScript. It is
// unspellable in can (identifiers match \w+, so $ never appears in
// source), keeping metadata disjoint from logical payload fields: a
// field named kind stays data, and the tag can never be overwritten by
// one. Payload fields keep their source names verbatim.
const tsTag = "$can_kind"

// tsField renders one payload field as target storage. Every declared
// field becomes an own data property: ordinarily name: value, except
// __proto__, which an object literal would otherwise install as the
// prototype — the computed-key form defines a real own property.
func tsField(name, value string) string {
	if name == "__proto__" {
		return `["__proto__"]: ` + value
	}
	return name + ": " + value
}

// tsErrMember renders one error kind as a TS union member.
func tsErrMember(ed *ErrorDecl, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	return tsErrFieldsMember(ed.Name, ed.Fields, brands, recs, variants, errs)
}

// tsErrFieldsMember renders one error kind from looked-up fields,
// so callable unions (which resolve kinds through the error shape
// map, not declarations) share the member byte-for-byte.
func tsErrFieldsMember(name string, fields [][2]string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	fs := ""
	for _, f := range fields {
		t, err := tsTypeB(f[1], brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		fs += "; " + f[0] + ": " + t
	}
	return fmt.Sprintf("{ %s: \"%s\"%s }", tsTag, name, fs), nil
}

// tsVariantMember renders one variant case as a TS union member
// (a74): the qualified case name is the discriminant tag, payload
// fields map through their declared types. Nullary cases carry the
// tag only. The shape mirrors tsErrMember; the checker guarantees
// exact fields, so emit trusts the declaration.
func tsVariantMember(parent string, vc VariantCase, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	tag := qualifyCase(parent, vc.Short)
	fs := ""
	for _, f := range vc.Fields {
		t, err := tsTypeB(f[1], brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		fs += "; " + f[0] + ": " + t
	}
	return fmt.Sprintf("{ %s: \"%s\"%s }", tsTag, tag, fs), nil
}

// builtinErrorLookup finds an error declaration by kind: compiler-owned
// first, then source last-wins across modules (matching the loops it
// replaces). A builtin name can never resolve to source: world build
// rejects such redefinitions as primitive shadows.
func builtinErrorLookup(name string, mods []*Module) *ErrorDecl {
	for _, b := range builtinErrorDecls() {
		if b.Name == name {
			return b
		}
	}
	var ed *ErrorDecl
	for _, m := range mods {
		for _, d := range m.Decls {
			if er, ok := d.(*ErrorDecl); ok && er.Name == name {
				ed = er
			}
		}
	}
	return ed
}

// resultUnion renders one TS Result union: an ok member carrying
// the given record fields plus one member per emitted error
// kind, in emits order. externUnion, fnResultUnion, and
// decodeResultUnion differ only in where the ok shape and the
// error declarations come from; the rendering — and the
// unknown-error failure — lives here once, so the three unions
// cannot drift apart. (Per-function unions vs the module-wide
// union stay distinct by design: a14 emit narrowing, not
// suppressions.)
func okMember(fs string) string {
	return fmt.Sprintf("{ %s: \"ok\"%s }", tsTag, fs)
}

// okFieldsMember renders the ok member from raw record fields,
// mapping each field type once. (declaredOkShape instead returns
// already-mapped types for the module-union dedup keys, so the
// per-function union assembles its member from that map without
// a second mapping pass.)
func okFieldsMember(fields [][2]string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, errs map[string][][2]string) (string, error) {
	fs := ""
	for _, f := range fields {
		t, err := tsTypeB(f[1], brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		fs += "; " + f[0] + ": " + t
	}
	return okMember(fs), nil
}

func resultUnion(ok string, emits []string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl, lookupMods []*Module, emitsErr func(e string) error) (string, error) {
	union := ok
	var errs map[string][][2]string
	if lookupMods != nil {
		errs = errorShapes(lookupMods)
	}
	for _, e := range emits {
		ed := builtinErrorLookup(e, lookupMods)
		if ed == nil {
			return "", emitsErr(e)
		}
		mem, err := tsErrMember(ed, brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		union += " | " + mem
	}
	return union, nil
}

// externUnion is the TS Result type of a foreign call: ok carrying the
// declared success shape plus one member per emits kind. The host owns
// the implementation; scripts check this interface, not the host's behavior.
func externUnion(ex *ExternDecl, prog *Program) (string, error) {
	recs := recordShapes(prog.Modules)
	variants := variantShapes(prog.Modules)
	if typeContainsFn(ex.Ret, recs) {
		return "", fmt.Errorf("extern %s: signatures are data-only", ex.Name)
	}
	for _, p := range ex.Params {
		if typeContainsFn(p[1], recs) {
			return "", fmt.Errorf("extern %s: signatures are data-only", ex.Name)
		}
	}
	fields, found := successFields(ex.Ret, recs, variants[ex.Ret] != nil || prog.Brands[ex.Ret] != "")
	if !found {
		return "", fmt.Errorf("extern %s returns unknown type %s", ex.Name, ex.Ret)
	}
	ok, err := okFieldsMember(fields, prog.Brands, recs, variants, errorShapes(prog.Modules))
	if err != nil {
		return "", err
	}
	return resultUnion(ok, ex.Emits, prog.Brands, recs, variants, prog.Modules,
		func(e string) error { return fmt.Errorf("extern %s emits unknown error %s", ex.Name, e) })
}

// fnResultUnion is the TS Result type of one local function: ok
// carrying its declared success shape plus one member per declared emits
// kind, in emits order. Call temporaries and return annotations use
// this instead of the module-wide union so a strict checker narrows
// each handled outcome to its exact payload shape (a14: emit
// narrowing, not suppressions). The module union stays as the general
// published type. Ok fields sort to match the module-union member for
// the same shape.
func fnResultUnion(fn *FnDecl, prog *Program) (string, error) {
	shape, err := declaredOkShape(fn, prog)
	if err != nil {
		return "", err
	}
	var names []string
	for n := range shape {
		names = append(names, n)
	}
	sort.Strings(names)
	fs := ""
	for _, n := range names {
		fs += "; " + n + ": " + shape[n]
	}
	recs := recordShapes(prog.Modules)
	variants := variantShapes(prog.Modules)
	return resultUnion(okMember(fs), fn.Emits, prog.Brands, recs, variants, prog.Modules,
		func(e string) error { return fmt.Errorf("%s emits unknown error %s", fn.Name, e) })
}

// declaredOkShape builds a function's Ok payload shape from its declared
// return record or value envelope, never from body/test literals. The
// checker proves every Ok construction against that shape (unknown,
// missing, and mistyped fields fail before emission), so the declaration
// is the single source of types: changing evidence without changing the
// signature or body cannot change the emitted type, and computed or
// referenced payloads need no literal witness.
func declaredOkShape(fn *FnDecl, prog *Program) (map[string]string, error) {
	recs := recordShapes(prog.Modules)
	variants := variantShapes(prog.Modules)
	fields, ok := successFields(fn.Ret, recs, variants[fn.Ret] != nil || prog.Brands[fn.Ret] != "")
	if !ok {
		return nil, fmt.Errorf("%s returns unknown type %s", fn.Name, fn.Ret)
	}
	shape := map[string]string{}
	errs := errorShapes(prog.Modules)
	for _, f := range fields {
		t, err := tsTypeB(f[1], prog.Brands, recs, variants, errs)
		if err != nil {
			return nil, err
		}
		shape[f[0]] = t
	}
	return shape, nil
}

// leafType reports the static type of a literal operand: kinds carry
// their own types, so emit can dispatch exactly even where the checker
// never annotated (direct unit calls). Refs need checker annotations.
func leafType(s *Small) string {
	switch s.Kind {
	case "int":
		return "int"
	case "dec":
		return "dec"
	case "str":
		return "str"
	case "bool":
		return "bool"
	case "seqlit":
		// Literal operands carry their sequence type without
		// checker annotations, like every other literal kind.
		// Emit runs only after checkSem, so Elem is valid here.
		return "Seq<" + s.Elem + ">"
	case "ctor":
		// The validated Bytes constructor carries its type without
		// checker annotations, like every other literal kind.
		if s.Ctor == "Bytes" {
			return "Bytes"
		}
	}
	return ""
}

// childType resolves one operand's static type: the checker's T
// annotation first, literals second. Emit runs only after checkSem,
// so unknown here is a loud internal, never a silent default.
func childType(s *Small) string {
	if s == nil {
		return ""
	}
	if s.T != "" {
		return s.T
	}
	return leafType(s)
}

// isScalar reports whether an operand type compares exactly with
// native identity: base types (bigint by value, canonical dec
// strings, strings, booleans) and str-backed brands.
func (e *emitter) isScalar(ot string) bool {
	if _, ok := tsBase[ot]; ok {
		// a45 S1: Bytes lowers through tsBase but never compares
		// natively; membership here must not imply ===.
		return ot != "Bytes"
	}
	return e.brands[ot] == "str"
}

// emitEquality lowers ==/!= by resolved operand type. Scalars and
// brands compare natively; records and error payloads compare
// field-by-field over their declared shape, ignoring the outcome
// envelope; cell wrappers compare their .value payloads. Unknown or
// unsupported operand types fail (CAN5005) instead of falling back to
// object identity.
func (e *emitter) emitEquality(op, ot, l, r string) (string, error) {
	ll, rr, shape := l, r, ot
	if strings.HasPrefix(shape, "cell:") {
		// Cell wrappers compare their .value payloads, never the
		// wrapper identity.
		ll += ".value"
		rr += ".value"
		shape = strings.TrimPrefix(shape, "cell:")
	}
	if _, _, _, ok := fnTypeShape(shape); ok {
		return "", fmt.Errorf("cannot emit comparison over %s: function values do not compare (%s)", shape, CodeBadCompare)
	}
	if shape == "Bytes" {
		// a45 S1: direct byte operators are deferred. Nested byte
		// comparison routes through the structural runtime below;
		// never emit === for Bytes here.
		return "", fmt.Errorf("cannot emit comparison over Bytes: byte equality is deferred (%s)", CodeBadCompare)
	}
	native := e.isScalar(shape)
	var expr string
	if native {
		if op == "==" {
			expr = fmt.Sprintf("(%s === %s)", ll, rr)
		} else {
			expr = fmt.Sprintf("(%s !== %s)", ll, rr)
		}
		return expr, nil
	}
	if e.shapeContainsBytes(shape) {
		e.bytesEq = true
	}
	fields, err := e.equalityFields(shape)
	if err != nil {
		return "", err
	}
	var fs []string
	for _, f := range fields {
		fs = append(fs, strconv.Quote(f))
	}
	e.recEq = true
	expr = fmt.Sprintf("$canEqRec(%s, %s, [%s])", ll, rr, strings.Join(fs, ", "))
	if op == "!=" {
		expr = "(!" + expr + ")"
	}
	return expr, nil
}

// equalityFields resolves the declared comparison shape for a record
// or error-payload operand: record names to their type fields, err:
// kinds to their error fields. Anything else is not structurally
// comparable (CAN5005).
func (e *emitter) equalityFields(ot string) ([]string, error) {
	if strings.HasPrefix(ot, "err:") {
		fs, ok := e.errFields[strings.TrimPrefix(ot, "err:")]
		if !ok {
			return nil, fmt.Errorf("cannot emit comparison over %s (%s)", ot, CodeBadCompare)
		}
		return fs, nil
	}
	if fs, ok := e.recs[ot]; ok {
		if e.shapeHasFn(ot, map[string]bool{}) {
			return nil, fmt.Errorf("cannot emit comparison over %s: record contains a function value (%s)", ot, CodeBadCompare)
		}
		var names []string
		for _, f := range fs {
			names = append(names, f[0])
		}
		return names, nil
	}
	return nil, fmt.Errorf("cannot emit comparison over %s (%s)", ot, CodeBadCompare)
}

// shapeHasFn reports whether a record transitively contains a
// callable through its fields. The visited set keeps recursive
// data graphs terminating; error and variant payloads refuse Fn
// at their own positions, so they are leaves here.
func (e *emitter) shapeHasFn(t string, seen map[string]bool) bool {
	if _, _, _, ok := fnTypeShape(t); ok {
		return true
	}
	if elem, ok := seqElemName(t); ok {
		return e.shapeHasFn(elem, seen)
	}
	if strings.HasPrefix(t, "Seq<") && strings.HasSuffix(t, ">") {
		return e.shapeHasFn(t[len("Seq<"):len(t)-1], seen)
	}
	fields, ok := e.recs[t]
	if !ok {
		return false
	}
	if seen[t] {
		return false
	}
	seen[t] = true
	for _, f := range fields {
		if e.shapeHasFn(f[1], seen) {
			return true
		}
	}
	return false
}

// binopOperandType reports the static can type of a binop's operands
// for dispatch. Same-type operands are enforced by the checker, so one
// type describes both sides. Checker annotations win; literal kinds
// are the fallback. Empty means unknown: the caller fails loud.
func binopOperandType(node *Small) string {
	if node.T != "" {
		return node.T
	}
	if t := leafType(node.L); t != "" {
		return t
	}
	return leafType(node.R)
}

func (e *emitter) emitValue(node *Small) (string, error) {
	switch node.Kind {
	case "str":
		// One literal encoder (normStr): the in-memory string is
		// either source-raw (ordinary "..." interprets no
		// escapes) or already decoded (a66 e"..." six escapes),
		// so the target literal must re-escape every special
		// char. Values and patterns share this encoder (see
		// stmtMatch).
		return normStr(node.Str), nil
	case "int":
		return node.Num.String() + "n", nil
	case "dec":
		return strconv.Quote(node.Dec), nil
	case "seal":
		// Sealed values emit as their string: brands erase. The
		// checker owns the one-string-operand rule (a25).
		if len(node.Args) != 1 {
			return "", fmt.Errorf("cannot emit seal: want one value")
		}
		return e.emitValue(node.Args[0].V)
	case "bool":
		if node.B {
			return "true", nil
		}
		return "false", nil
	case "seqlit":
		// a36 S1: sequence values are array literals, members in
		// order. Sealed members erase to their strings through the
		// shared seal arm, so brands need no special case here.
		parts := make([]string, 0, len(node.Items))
		for _, it := range node.Items {
			v, err := e.emitValue(it)
			if err != nil {
				return "", err
			}
			parts = append(parts, v)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case "ref":
		// Slice 1: a constant reference inlines its literal:
		// the checker owns existence (CAN2104) and linkage
		// (CAN2105), so emit substitutes the value. Unknown
		// names emit verbatim and fail downstream as before.
		if len(node.Ref) == 1 && constNameRe.MatchString(node.Ref[0]) {
			if c, ok := e.consts[node.Ref[0]]; ok && c.Value != nil {
				return e.emitValue(c.Value)
			}
		}
		return strings.Join(node.Ref, "."), nil
	case "binop":
		l, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		r, err := e.emitValue(node.R)
		if err != nil {
			return "", err
		}
		// Slice 5: eager combinators lower to strict helper
		// calls (both arguments evaluate before the call, so
		// no bare && short-circuits); the helper emits only
		// when used, following the $canDec* pattern.
		if node.Op == "and" || node.Op == "or" {
			name := "$canBoolAnd"
			key := "and"
			if node.Op == "or" {
				name = "$canBoolOr"
				key = "or"
			}
			e.boolOps[key] = true
			return fmt.Sprintf("%s(%s, %s)", name, l, r), nil
		}
		ot := binopOperandType(node)
		if ot == "" {
			return "", fmt.Errorf("cannot emit %s: operand type unknown (run checkSem first)", node.Op)
		}
		// Equality lowers by resolved operand type, preserving the
		// evaluator's structural semantics: scalars (and brands,
		// erased to strings) are exact natively — bigint compares by
		// value, dec strings are canonical — while records, error
		// payloads, and cell wrappers compare field-by-field over
		// their declared shape, ignoring the outcome envelope.
		// Anything else fails instead of falling back to object
		// identity (CAN5005).
		if node.Op == "==" || node.Op == "!=" {
			eq, err := e.emitEquality(node.Op, ot, l, r)
			if err != nil {
				return "", err
			}
			return eq, nil
		}
		if ot == "dec" {
			// Decs are canonical-digit strings: route through the
			// exact $canDec helpers, never native operators.
			switch node.Op {
			case "+":
				e.decOps["add"] = true
				return fmt.Sprintf("$canDecAdd(%s, %s)", l, r), nil
			case "-":
				e.decOps["sub"] = true
				return fmt.Sprintf("$canDecSub(%s, %s)", l, r), nil
			case "*":
				e.decOps["mul"] = true
				return fmt.Sprintf("$canDecMul(%s, %s)", l, r), nil
			case ">=":
				e.decOps["ge"] = true
				return fmt.Sprintf("$canDecGe(%s, %s)", l, r), nil
			case "<=":
				e.decOps["le"] = true
				return fmt.Sprintf("$canDecLe(%s, %s)", l, r), nil
			case ">":
				e.decOps["gt"] = true
				return fmt.Sprintf("$canDecGt(%s, %s)", l, r), nil
			case "<":
				e.decOps["lt"] = true
				return fmt.Sprintf("$canDecLt(%s, %s)", l, r), nil
			}
			return "", fmt.Errorf("cannot emit op %s", node.Op)
		}
		// Strings and brands order by UTF-8 bytes through the
		// $canStr helpers, matching Go's byte-wise evaluator
		// ordering (native target ordering is UTF-16 code units and
		// disagrees past the BMP); string + concatenates natively
		// and ==/!== stay exact in both runtimes.
		if ot == "str" || e.brands[ot] == "str" {
			switch node.Op {
			case ">=":
				e.strOps["ge"] = true
				return fmt.Sprintf("$canStrGe(%s, %s)", l, r), nil
			case "<=":
				e.strOps["le"] = true
				return fmt.Sprintf("$canStrLe(%s, %s)", l, r), nil
			case ">":
				e.strOps["gt"] = true
				return fmt.Sprintf("$canStrGt(%s, %s)", l, r), nil
			case "<":
				e.strOps["lt"] = true
				return fmt.Sprintf("$canStrLt(%s, %s)", l, r), nil
			}
		}
		// Ints are bigints (native ops exact, except / and %: BigInt
		// truncates toward zero, so Euclidean division rides the
		// $canDivMod helper, emitted inline only when used).
		if ot == "int" && (node.Op == "/" || node.Op == "%") {
			e.divmod = true
			idx := "0"
			if node.Op == "%" {
				idx = "1"
			}
			return fmt.Sprintf("$canDivMod(%s, %s)[%s]", l, r, idx), nil
		}
		// a39 S4: sequence append lowers to spread with precise
		// element types; the checker owns the operand rule
		// (Seq+T only, never Seq+Seq).
		if node.Op == "+" {
			if _, ok := seqElemName(ot); ok {
				return fmt.Sprintf("[...%s, %s]", l, r), nil
			}
		}
		// Ordering past this point is ints only: dec, str, and
		// str-backed brands returned through their helpers
		// above, and anything else (records, bools, cells) has
		// no native ordering — fail like == instead of
		// emitting a meaningless object comparison.
		if isOrdering(node.Op) && ot != "int" {
			return "", fmt.Errorf("cannot emit %s over %s (%s)", node.Op, ot, CodeBadCompare)
		}
		ops := map[string]string{">=": ">=", "<=": "<=", ">": ">", "<": "<", "+": "+", "-": "-", "*": "*"}
		op, ok := ops[node.Op]
		if !ok {
			return "", fmt.Errorf("cannot emit op %s", node.Op)
		}
		return fmt.Sprintf("(%s %s %s)", l, op, r), nil
	case "strlen":
		v, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		// a37 S2: arrays lower through the same spread-length
		// shape as strings; the checker owns the operand rule.
		if ct := childType(node.L); ct != "str" {
			if _, ok := seqElemName(ct); !ok {
				return "", fmt.Errorf("cannot emit #: operand type unknown (run checkSem first)")
			}
		}
		return fmt.Sprintf("(BigInt([...%s].length))", v), nil
	case "not":
		// Slice 5: negation is already strict (the operand
		// evaluates first), so a plain prefix lowers it.
		v, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("!(%s)", v), nil
	case "neg":
		// Slice 6: integers emit (-value) over native
		// bigints. Decimals never emit JavaScript -x
		// (canonical strings): exact subtraction from
		// decimal zero through the shared helper.
		v, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		switch binopOperandType(node) {
		case "int":
			return fmt.Sprintf("(-%s)", v), nil
		case "dec":
			e.decOps["sub"] = true
			return fmt.Sprintf("$canDecSub(\"0.0\", %s)", v), nil
		default:
			return "", fmt.Errorf("cannot emit neg: operand type unknown (run checkSem first)")
		}
	case "stridx":
		b, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		ix, err := e.emitValue(node.R)
		if err != nil {
			return "", err
		}
		if childType(node.L) != "str" {
			// a38 S3: sequence indexing lowers to the generic
			// seq helper; the checker owns the base rule.
			if _, ok := seqElemName(childType(node.L)); !ok {
				return "", fmt.Errorf("cannot emit []: operand type unknown (run checkSem first)")
			}
			e.seqOps["seqat"] = true
			return fmt.Sprintf("$canSeqAt(%s, %s)", b, ix), nil
		}
		e.strOps["at"] = true
		return fmt.Sprintf("$canStrAt(%s, %s)", b, ix), nil
	case "proj":
		// b03: bases are postfix ($canSeqAt/$canStrSlice
		// calls or nested projections) — all safe under one
		// appended property access.
		b, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		return b + "." + node.Field, nil
	case "strslice":
		b, err := e.emitValue(node.L)
		if err != nil {
			return "", err
		}
		lo, err := e.emitValue(node.R)
		if err != nil {
			return "", err
		}
		hi, err := e.emitValue(node.Hi)
		if err != nil {
			return "", err
		}
		if childType(node.L) != "str" {
			return "", fmt.Errorf("cannot emit [:]: operand type unknown (run checkSem first)")
		}
		e.strOps["slice"] = true
		return fmt.Sprintf("$canStrSlice(%s, %s, %s)", b, lo, hi), nil
	case "ctor":
		if node.Ctor == "Ok" && len(node.TypeArgs) > 0 {
			return e.emitWholeOk(node)
		}
		if node.Ctor == "Bytes" {
			// a45 S1: validated byte lowering. Emit checked members
			// as number literals, never through the Seq bigint path:
			// wrapping it would throw on 0n and silently remap -1
			// and 256. Re-validate for direct emitter callers.
			if len(node.Args) != 1 || node.Args[0].HasName {
				return "", fmt.Errorf("cannot emit Bytes: want one Seq<int> literal")
			}
			arg := node.Args[0].V
			if arg.Kind != "seqlit" || arg.Elem != "int" {
				return "", fmt.Errorf("cannot emit Bytes: want one Seq<int> literal")
			}
			nums := make([]string, 0, len(arg.Items))
			for _, it := range arg.Items {
				if it.Kind != "int" || it.Num == nil || it.Num.Sign() < 0 || it.Num.Cmp(big.NewInt(256)) >= 0 {
					return "", fmt.Errorf("cannot emit Bytes: member out of range 0..255")
				}
				nums = append(nums, it.Num.String())
			}
			return "Uint8Array.from([" + strings.Join(nums, ", ") + "])", nil
		}
		var parts []string
		for _, a := range node.Args {
			v, err := e.emitValue(a.V)
			if err != nil {
				return "", err
			}
			parts = append(parts, tsField(a.Name, v))
		}
		inner := strings.Join(parts, ", ")
		if node.Ctor == "Ok" {
			if inner == "" {
				return `{ ` + tsTag + `: "ok" }`, nil
			}
			return `{ ` + tsTag + `: "ok", ` + inner + ` }`, nil
		}
		if strings.Contains(node.Ctor, ".") {
			if inner == "" {
				return `{ ` + tsTag + `: "` + node.Ctor + `" }`, nil
			}
			return `{ ` + tsTag + `: "` + node.Ctor + `", ` + inner + ` }`, nil
		}
		// Declared variant cases in value positions (a74) are
		// tagged data objects like errors: the qualified tag
		// names the case, payload fields follow. Emit runs only
		// after checkSem, so exact fields are guaranteed here.
		if _, ok := e.cases[node.Ctor]; ok {
			if inner == "" {
				return `{ ` + tsTag + `: "` + node.Ctor + `" }`, nil
			}
			return `{ ` + tsTag + `: "` + node.Ctor + `", ` + inner + ` }`, nil
		}
		// Declared records in value positions are plain data objects
		// with the checked field set: no outcome tag, since these
		// values never enter the match protocol. Unknown constructors
		// fail loud instead of mistagging as an error kind.
		if _, ok := e.recs[node.Ctor]; !ok {
			return "", fmt.Errorf("cannot emit unknown constructor %s", node.Ctor)
		}
		return `{ ` + inner + ` }`, nil
	case "fnref":
		return e.emitFnref(node)
	case "call":
		// One binding rule (bindSlots): the call is invoked in the
		// resolved parameter order, not source order, so a reordered
		// named call means the same thing here as in evaluation.
		// Store ops are positional-only with a fixed spelling and keep
		// their source order.
		ordered := node.Args
		if !isStoreOp(node.Fname) {
			params, ok := e.params[node.Fname]
			if !ok {
				return "", fmt.Errorf("no params for callee %s", node.Fname)
			}
			slots, err := bindSlots(node.Fname, node.Args, params)
			if err != nil {
				return "", err
			}
			ordered = make([]Arg, len(node.Args))
			for i, a := range node.Args {
				ordered[slots[i]] = a
			}
		}
		var parts []string
		for _, a := range ordered {
			v, err := e.emitValue(a.V)
			if err != nil {
				return "", err
			}
			parts = append(parts, v)
		}
		return fmt.Sprintf("%s(%s)", node.Fname, strings.Join(parts, ", ")), nil
	}
	return "", fmt.Errorf("cannot emit: %s", node.Kind)
}

// decRuntimeShared implements the representation plumbing every dec
// operation needs: split canonical digits, renormalize, and convert
// between canonical strings and scaled BigInt mantissas. It mirrors
// parseDecParts/decArith/canonDec in eval.go exactly: add/sub align to
// the wider scale, mul sums scales, results renormalize (no trailing
// fractional zeros, -0 folds to 0.0).
var decRuntimeShared = []string{
	"function $canDecSplit(d: string): { neg: boolean; ip: string; fp: string } {",
	"  let neg = false;",
	"  if (d.startsWith(\"-\")) {",
	"    neg = true;",
	"    d = d.slice(1);",
	"  }",
	"  const dot = d.indexOf(\".\");",
	"  return { neg, ip: d.slice(0, dot), fp: d.slice(dot + 1) };",
	"}",
	"function $canDecNorm(ip: string, fp: string, neg: boolean): string {",
	"  ip = ip.replace(/^0+(?=\\d)/, \"\");",
	"  fp = fp.replace(/0+$/, \"\");",
	"  if (fp === \"\") {",
	"    fp = \"0\";",
	"  }",
	"  if (ip === \"0\" && fp === \"0\") {",
	"    return \"0.0\";",
	"  }",
	"  return (neg ? \"-\" : \"\") + ip + \".\" + fp;",
	"}",
	"function $canDecMant(p: { neg: boolean; ip: string; fp: string }, scale: number): bigint {",
	"  let f = p.fp;",
	"  while (f.length < scale) {",
	"    f += \"0\";",
	"  }",
	"  const m = BigInt(p.ip + f);",
	"  return p.neg ? -m : m;",
	"}",
	"function $canDecFromMant(m: bigint, scale: number): string {",
	"  let neg = false;",
	"  if (m < 0n) {",
	"    neg = true;",
	"    m = -m;",
	"  }",
	"  let digits = m.toString();",
	"  while (digits.length < scale + 1) {",
	"    digits = \"0\" + digits;",
	"  }",
	"  return $canDecNorm(digits.slice(0, digits.length - scale), digits.slice(digits.length - scale), neg);",
	"}",
}

// decRuntimeOps holds one exact operation per can operator, in fixed
// order for byte-stable output. Only used entries are emitted.
var decRuntimeOps = []struct {
	key  string
	code []string
}{
	{"add", []string{
		"function $canDecAdd(a: string, b: string): string {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecFromMant($canDecMant(A, s) + $canDecMant(B, s), s);",
		"}",
	}},
	{"sub", []string{
		"function $canDecSub(a: string, b: string): string {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecFromMant($canDecMant(A, s) - $canDecMant(B, s), s);",
		"}",
	}},
	{"mul", []string{
		"function $canDecMul(a: string, b: string): string {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  return $canDecFromMant($canDecMant(A, A.fp.length) * $canDecMant(B, B.fp.length), A.fp.length + B.fp.length);",
		"}",
	}},
	{"ge", []string{
		"function $canDecGe(a: string, b: string): boolean {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecMant(A, s) >= $canDecMant(B, s);",
		"}",
	}},
	{"le", []string{
		"function $canDecLe(a: string, b: string): boolean {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecMant(A, s) <= $canDecMant(B, s);",
		"}",
	}},
	{"gt", []string{
		"function $canDecGt(a: string, b: string): boolean {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecMant(A, s) > $canDecMant(B, s);",
		"}",
	}},
	{"lt", []string{
		"function $canDecLt(a: string, b: string): boolean {",
		"  const A = $canDecSplit(a);",
		"  const B = $canDecSplit(b);",
		"  const s = Math.max(A.fp.length, B.fp.length);",
		"  return $canDecMant(A, s) < $canDecMant(B, s);",
		"}",
	}},
	{"parts", []string{
		"function $canDecParts(d: string): { coefficient: bigint; scale: bigint } {",
		"  const p = $canDecSplit(d);",
		"  return { coefficient: $canDecMant(p, p.fp.length), scale: BigInt(p.fp.length) };",
		"}",
	}},
}

// strRuntimeShared compares strings by UTF-8 bytes: Go orders strings
// byte-wise, while native target ordering is UTF-16 code units and
// disagrees past the BMP (U+E000 vs U+10000 sorts opposite). Encoding
// both sides to UTF-8 and comparing bytes reproduces the evaluator
// exactly on every input.
var strRuntimeShared = []string{
	"function $canStrCmp(a: string, b: string): number {",
	"  const A = new TextEncoder().encode(a);",
	"  const B = new TextEncoder().encode(b);",
	"  const n = Math.min(A.length, B.length);",
	"  for (let i = 0; i < n; i++) {",
	"    if (A[i] !== B[i]) {",
	"      return A[i] < B[i] ? -1 : 1;",
	"    }",
	"  }",
	"  if (A.length === B.length) {",
	"    return 0;",
	"  }",
	"  return A.length < B.length ? -1 : 1;",
	"}",
}

var strRuntimeOps = []struct {
	key  string
	code []string
}{
	{"ge", []string{
		"function $canStrGe(a: string, b: string): boolean {",
		"  return $canStrCmp(a, b) >= 0;",
		"}",
	}},
	{"le", []string{
		"function $canStrLe(a: string, b: string): boolean {",
		"  return $canStrCmp(a, b) <= 0;",
		"}",
	}},
	{"at", []string{
		"function $canStrAt(s: string, i: bigint): bigint {",
		"  const cps = [...s];",
		"  if (i < 0n || i > BigInt(Number.MAX_SAFE_INTEGER)) throw new Error(\"str index out of range\");",
		"  const k = Number(i);",
		"  if (k >= cps.length) throw new Error(\"str index out of range\");",
		"  const cp = cps[k].codePointAt(0);",
		"  if (cp === undefined) throw new Error(\"str index out of range\");",
		"  return BigInt(cp);",
		"}",
	}},
	{"slice", []string{
		"function $canStrSlice(s: string, a: bigint, b: bigint): string {",
		"  const cps = [...s];",
		"  const toIdx = (x: bigint): number => { if (x < 0n || x > BigInt(Number.MAX_SAFE_INTEGER)) throw new Error(\"str slice out of range\"); return Number(x); };",
		"  const lo = toIdx(a), hi = toIdx(b);",
		"  if (lo > hi || hi > cps.length) throw new Error(\"str slice out of range\");",
		"  return cps.slice(lo, hi).join(\"\");",
		"}",
	}},
	{"gt", []string{
		"function $canStrGt(a: string, b: string): boolean {",
		"  return $canStrCmp(a, b) > 0;",
		"}",
	}},
	{"lt", []string{
		"function $canStrLt(a: string, b: string): boolean {",
		"  return $canStrCmp(a, b) < 0;",
		"}",
	}},
}

// Structural equality runtime, composed from shared chunks so
// each rule lives once: $canEqRec compares exactly the declared
// fields (ignoring the outcome envelope, requiring own presence
// on both sides, matching vEq), delegating per-field to
// $canEqVal, which recurses into nested objects and compares
// scalars exactly (bigint by value, canonical dec strings and
// strings by content, booleans natively). Shapes that can
// contain Bytes (a45 S1) prepend the typed-array branch and add
// the $canEqBytes comparison; a typed array never compares equal
// to an ordinary array merely because enumerable keys match.
var eqValHead = []string{
	"function $canEqVal(x: any, y: any): boolean {",
}

var eqValBytesBranch = []string{
	"  if (x instanceof Uint8Array || y instanceof Uint8Array) {",
	"    return $canEqBytes(x, y);",
	"  }",
}

var eqValObjectCore = []string{
	"  if (typeof x === \"object\" && x !== null && typeof y === \"object\" && y !== null) {",
	"    const kx = Object.keys(x);",
	"    if (kx.length !== Object.keys(y).length) {",
	"      return false;",
	"    }",
	"    for (const k of kx) {",
	"      if (!Object.prototype.hasOwnProperty.call(y, k)) {",
	"        return false;",
	"      }",
	"      if (!$canEqVal((x as any)[k], (y as any)[k])) {",
	"        return false;",
	"      }",
	"    }",
	"    return true;",
	"  }",
	"  return x === y;",
	"}",
}

var eqBytesHelper = []string{
	"function $canEqBytes(x: any, y: any): boolean {",
	"  if (!(x instanceof Uint8Array) || !(y instanceof Uint8Array)) {",
	"    return false;",
	"  }",
	"  if (x.length !== y.length) {",
	"    return false;",
	"  }",
	"  for (let i = 0; i < x.length; i++) {",
	"    if (x[i] !== y[i]) {",
	"      return false;",
	"    }",
	"  }",
	"  return true;",
	"}",
}

var eqRecHelper = []string{
	"function $canEqRec(a: any, b: any, fields: string[]): boolean {",
	"  for (const f of fields) {",
	"    if (!Object.prototype.hasOwnProperty.call(a, f)) {",
	"      return false;",
	"    }",
	"    if (!Object.prototype.hasOwnProperty.call(b, f)) {",
	"      return false;",
	"    }",
	"    if (!$canEqVal((a as any)[f], (b as any)[f])) {",
	"      return false;",
	"    }",
	"  }",
	"  return true;",
	"}",
}

func concatLines(chunks ...[]string) []string {
	var out []string
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

// eqHelpers renders the plain structural equality runtime;
// bytesEqHelpers renders the Bytes-capable variant. The two
// bundles are mutually exclusive at emission (recEq with or
// without bytesEq), and both compose the chunks above, so the
// emitted bytes are unchanged while each rule has one source.
var eqHelpers = concatLines(eqValHead, eqValObjectCore, eqRecHelper)

var bytesEqHelpers = concatLines(eqValHead, eqValBytesBranch, eqValObjectCore, eqBytesHelper, eqRecHelper)

// strHelpers renders the byte-order string runtime for exactly the used
// comparisons, shared plumbing first, then ops in fixed order.
func strHelpers(used map[string]bool) []string {
	var out []string
	out = append(out, "// Byte-order string comparison: UTF-8 bytes, matching Go.")
	out = append(out, strRuntimeShared...)
	for _, op := range strRuntimeOps {
		if used[op.key] {
			out = append(out, op.code...)
		}
	}
	return out
}

var seqRuntimeOps = []struct {
	key  string
	code []string
}{
	{"seqat", []string{
		"function $canSeqAt<T>(a: T[], i: bigint): T {",
		"  if (i < 0n || i > BigInt(Number.MAX_SAFE_INTEGER)) throw new Error(\"seq index out of range\");",
		"  const k = Number(i);",
		"  if (k >= a.length) throw new Error(\"seq index out of range\");",
		"  return a[k];",
		"}",
	}},
}

// seqHelpers renders the sequence runtime for exactly the used
// operations, mirroring strHelpers: shared plumbing first (none
// yet), then ops in fixed order.
func seqHelpers(used map[string]bool) []string {
	var out []string
	out = append(out, "// Sequence indexing (a38 S3): bounds throw, matching Go.")
	for _, op := range seqRuntimeOps {
		if used[op.key] {
			out = append(out, op.code...)
		}
	}
	return out
}

// boolHelpers renders the strict boolean runtime for exactly the
// used operators. Calls evaluate both arguments before the body
// runs, so and/or stay eager where a bare && would short-circuit.
func boolHelpers(used map[string]bool) []string {
	var out []string
	out = append(out, "// Strict boolean runtime (slice 5): eager helpers, never bare &&.")
	if used["and"] {
		out = append(out,
			"function $canBoolAnd(a: boolean, b: boolean): boolean {",
			"  return a && b;",
			"}",
		)
	}
	if used["or"] {
		out = append(out,
			"function $canBoolOr(a: boolean, b: boolean): boolean {",
			"  return a || b;",
			"}",
		)
	}
	return out
}

// decHelpers renders the exact-decimal runtime for exactly the used
// operations, shared plumbing first, then ops in fixed order.
func decHelpers(used map[string]bool) []string {
	var out []string
	out = append(out, "// Exact-decimal runtime (a10): canonical-digit strings, BigInt math.")
	out = append(out, decRuntimeShared...)
	for _, op := range decRuntimeOps {
		if used[op.key] {
			out = append(out, op.code...)
		}
	}
	return out
}

type emitter struct {
	tmp       int
	fnUnions  map[string]string      // callee fn -> its TS Result type
	params    map[string][][2]string // callee fn/extern -> params in order
	brands    map[string]string      // brand -> underlying, for erasure
	cellTypes map[string]string      // cell -> TS type, this module only
	tailUnion string
	decOps    map[string]bool         // exact-decimal helpers used by this module
	strOps    map[string]bool         // byte-order string helpers used by this module
	seqOps    map[string]bool         // sequence helpers used by this module (a38 S3)
	boolOps   map[string]bool         // strict boolean helpers used by this module (slice 5)
	recEq     bool                    // structural record comparison used by this module
	bytesEq   bool                    // compared shapes can contain Bytes (a45 S1)
	recs      map[string][][2]string  // record name -> declared fields
	errFields map[string][]string     // error kind -> declared field names
	errTypes  map[string][][2]string  // error kind -> declared typed fields
	variants  map[string]*VariantDecl // variant parent -> declaration (a74)
	cases     map[string]string       // qualified case -> parent variant (a74)
	divmod    bool                    // Euclidean division helper used by this module
	utf8dec   bool                    // strict UTF-8 decode helper used by this module (a50 B6)
	hexenc    bool                    // hex encode helper used by this module (a52 B8)
	hexdec    bool                    // strict hex decode helper used by this module (a55 B10)
	b64enc    bool                    // base64 encode helper used by this module (a58 B12)
	b64dec    bool                    // strict base64 decode helper used by this module (a59 B14)
	consts    map[string]*ConstDecl   // program const table: refs inline their literal (slice 1)
}

// shapeContainsBytes reports whether a comparison operand's declared
// shape can carry Bytes: directly, through Seq<Bytes>, or nested in
// records and error payloads. Recursion is bounded by visited names.
func (e *emitter) shapeContainsBytes(ot string) bool {
	seen := map[string]bool{}
	var rec func(t string) bool
	rec = func(t string) bool {
		if t == "Bytes" {
			return true
		}
		if elem, ok := seqElemName(t); ok {
			return rec(elem)
		}
		if strings.HasPrefix(t, "err:") {
			fs, ok := e.errTypes[strings.TrimPrefix(t, "err:")]
			if !ok {
				return false
			}
			for _, f := range fs {
				if rec(f[1]) {
					return true
				}
			}
			return false
		}
		if seen[t] {
			return false
		}
		seen[t] = true
		fs, ok := e.recs[t]
		if !ok {
			return false
		}
		for _, f := range fs {
			if rec(f[1]) {
				return true
			}
		}
		return false
	}
	return rec(strings.TrimPrefix(ot, "cell:"))
}

// divModHelper renders the Euclidean integer-division runtime: BigInt
// / and % truncate toward zero, so a negative truncated remainder is
// adjusted into 0 <= r < |b|. The helper text is pinned by
// TestDivEmitHelper and the Go-side contract by TestDivModVectors;
// executing the emitted helper against the contract awaits a node
// gate (tsc verification stays suspended per #11).
var divModHelper = []string{
	"// Euclidean integer division (a17): quotient and remainder with",
	"// 0 <= r < |b| on every sign combination.",
	"function $canDivMod(a: bigint, b: bigint): [bigint, bigint] {",
	"  let q: bigint = a / b;",
	"  let r: bigint = a - b * q;",
	"  if (r < 0n) {",
	"    q += b > 0n ? -1n : 1n;",
	"    r = a - b * q;",
	"  }",
	"  return [q, r];",
	"}",
}

// utf8DecodeHelper renders the strict UTF-8 decode runtime (a50 B6):
// validate-then-decode over the input VIEW (indices, never the
// backing buffer). The leading-byte table is the strict grammar Go
// utf8.Valid implements (C0/C1/F5+ excluded, E0/ED/F0/F4 second-byte
// ranges, continuation tails, no tolerant reads); only valid input
// reaches TextDecoder, so malformed bytes become the error VALUE
// without exception handling. fatal:true makes decoding total on
// valid input; ignoreBOM:true preserves a leading U+FEFF, which the
// default decoder would strip.
var utf8DecodeHelper = []string{
	"function $canUtf8Decode(value: Uint8Array): { $can_kind: \"ok\"; value: string } | { $can_kind: \"encoding.invalid_utf8\"; value: Uint8Array } {",
	"  let i = 0;",
	"  const n = value.length;",
	"  let valid = true;",
	"  while (i < n && valid) {",
	"    const b0 = value[i];",
	"    if (b0 < 0x80) { i += 1; continue; }",
	"    let need = 0; let lo = 0x80; let hi = 0xBF;",
	"    if (b0 >= 0xC2 && b0 <= 0xDF) { need = 1; }",
	"    else if (b0 === 0xE0) { need = 2; lo = 0xA0; }",
	"    else if (b0 >= 0xE1 && b0 <= 0xEC) { need = 2; }",
	"    else if (b0 === 0xED) { need = 2; hi = 0x9F; }",
	"    else if (b0 >= 0xEE && b0 <= 0xEF) { need = 2; }",
	"    else if (b0 === 0xF0) { need = 3; lo = 0x90; }",
	"    else if (b0 >= 0xF1 && b0 <= 0xF3) { need = 3; }",
	"    else if (b0 === 0xF4) { need = 3; hi = 0x8F; }",
	"    else { valid = false; break; }",
	"    if (i + need >= n) { valid = false; break; }",
	"    const b1 = value[i + 1];",
	"    if (b1 < lo || b1 > hi) { valid = false; break; }",
	"    for (let k = 2; k <= need; k++) {",
	"      const b = value[i + k];",
	"      if (b < 0x80 || b > 0xBF) { valid = false; break; }",
	"    }",
	"    i += 1 + need;",
	"  }",
	"  if (!valid) return { $can_kind: \"encoding.invalid_utf8\", value: value };",
	"  return { $can_kind: \"ok\", value: new TextDecoder(\"utf-8\", { fatal: true, ignoreBOM: true }).decode(value) };",
	"}",
}

// hexEncodeHelper renders the hex encode runtime (a52 B8): one
// lowercase digit pair per octet, byte-ordered, empty to "". The
// digit table (not arithmetic + case fixups) is what makes
// lowercase structural; indices keep views exact.
var hexEncodeHelper = []string{
	"function $canHexEncode(value: Uint8Array): string {",
	"  const digits = \"0123456789abcdef\";",
	"  let out = \"\";",
	"  for (let i = 0; i < value.length; i++) {",
	"    out += digits[value[i] >> 4] + digits[value[i] & 15];",
	"  }",
	"  return out;",
	"}",
}

// hexDecodeHelper renders the strict hex decode runtime (a55 B10):
// validate-then-decode over UTF-16 code units. Length parity and
// every unit's range membership are checked before any lookup; the
// explicit -1 sentinel makes an unchecked invalid unit
// indistinguishable from nothing (never default-zero into digit 0).
// Fresh output buffer per call; malformed input returns the error
// value with the original string. Allocation and host-contract
// failures stay loud: no catch relabels them as invalid hex.
var hexDecodeHelper = []string{
	"function $canHexVal(c: number): number {",
	"  if (c >= 48 && c <= 57) return c - 48;",
	"  if (c >= 65 && c <= 70) return c - 55;",
	"  if (c >= 97 && c <= 102) return c - 87;",
	"  return -1;",
	"}",
	"function $canHexDecode(value: string): { $can_kind: \"ok\"; value: Uint8Array } | { $can_kind: \"encoding.invalid_hex\"; value: string } {",
	"  if (value.length % 2 !== 0) return { $can_kind: \"encoding.invalid_hex\", value: value };",
	"  const out = new Uint8Array(value.length / 2);",
	"  for (let i = 0; i < value.length; i += 2) {",
	"    const hi = $canHexVal(value.charCodeAt(i));",
	"    const lo = $canHexVal(value.charCodeAt(i + 1));",
	"    if (hi < 0 || lo < 0) return { $can_kind: \"encoding.invalid_hex\", value: value };",
	"    out[i / 2] = hi * 16 + lo;",
	"  }",
	"  return { $can_kind: \"ok\", value: out };",
	"}",
}

// b64EncodeHelper renders the base64 encode runtime (a58 B12):
// standard padded alphabet over the input VIEW, three octets to
// four sextets, byte-ordered, empty to "". The alphabet table (not
// host btoa, whose binary-string contract is a misuse trap) is what
// makes the mapping structural; indices keep views exact.
var b64EncodeHelper = []string{
	"function $canB64Encode(value: Uint8Array): string {",
	"  const alpha = \"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/\";",
	"  let out = \"\";",
	"  let i = 0;",
	"  for (; i + 3 <= value.length; i += 3) {",
	"    const n = (value[i] << 16) | (value[i + 1] << 8) | value[i + 2];",
	"    out += alpha[(n >> 18) & 63] + alpha[(n >> 12) & 63] + alpha[(n >> 6) & 63] + alpha[n & 63];",
	"  }",
	"  const rem = value.length - i;",
	"  if (rem === 1) {",
	"    const n = value[i] << 16;",
	"    out += alpha[(n >> 18) & 63] + alpha[(n >> 12) & 63] + \"==\";",
	"  } else if (rem === 2) {",
	"    const n = (value[i] << 16) | (value[i + 1] << 8);",
	"    out += alpha[(n >> 18) & 63] + alpha[(n >> 12) & 63] + alpha[(n >> 6) & 63] + \"=\";",
	"  }",
	"  return out;",
	"}",
}

// b64DecodeHelper renders the strict base64 decode runtime (a59
// B14): validate-then-decode over UTF-16 code units. Length mod 4,
// per-position alphabet membership, final-quartet padding shape,
// and the exact unused-bit masks (four for DD==, two for DDD=)
// are all established before any sextet lookup; the explicit -1
// sentinel makes an unchecked unit indistinguishable from nothing.
// Fresh output buffer per call; malformed input returns the error
// value with the original string. No atob (forgiving), no catch.
var b64DecodeHelper = []string{
	"function $canB64Val(c: number): number {",
	"  if (c >= 65 && c <= 90) return c - 65;",
	"  if (c >= 97 && c <= 122) return c - 71;",
	"  if (c >= 48 && c <= 57) return c + 4;",
	"  if (c === 43) return 62;",
	"  if (c === 47) return 63;",
	"  return -1;",
	"}",
	"function $canB64Decode(value: string): { $can_kind: \"ok\"; value: Uint8Array } | { $can_kind: \"encoding.invalid_base64\"; value: string } {",
	"  if (value.length % 4 !== 0) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"  const nq = value.length / 4;",
	"  let pad = 0;",
	"  const vals: number[] = new Array(value.length);",
	"  for (let i = 0; i < value.length; i++) {",
	"    const q = (i / 4) | 0;",
	"    const pos = i % 4;",
	"    const c = value.charCodeAt(i);",
	"    if (c === 61) {",
	"      if (q !== nq - 1 || pos < 2) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"      pad++;",
	"      vals[i] = 0;",
	"    } else {",
	"      const v = $canB64Val(c);",
	"      if (v < 0 || pad > 0) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"      vals[i] = v;",
	"    }",
	"  }",
	"  if (pad > 2) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"  if (pad === 2 && (vals[value.length - 3] & 15) !== 0) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"  if (pad === 1 && (vals[value.length - 2] & 3) !== 0) return { $can_kind: \"encoding.invalid_base64\", value: value };",
	"  const out = new Uint8Array((value.length / 4) * 3 - pad);",
	"  for (let q = 0; q < nq; q++) {",
	"    const n = (vals[q * 4] << 18) | (vals[q * 4 + 1] << 12) | (vals[q * 4 + 2] << 6) | vals[q * 4 + 3];",
	"    const base = q * 3;",
	"    if (base < out.length) out[base] = (n >> 16) & 255;",
	"    if (base + 1 < out.length) out[base + 1] = (n >> 8) & 255;",
	"    if (base + 2 < out.length) out[base + 2] = n & 255;",
	"  }",
	"  return { $can_kind: \"ok\", value: out };",
	"}",
}

func (e *emitter) fresh() string {
	e.tmp++
	// Unspellable in can (identifiers match \w+, so $ never appears in
	// source): generated temporaries can never collide with source
	// bindings such as a parameter named _m1.
	return fmt.Sprintf("$can_m%d", e.tmp)
}

func (e *emitter) retLines(rhs *Node, env map[string]string) ([]string, error) {
	if rhs.IsMatch {
		var nested []string
		if err := e.stmtMatch(rhs, &nested); err != nil {
			return nil, err
		}
		return nested, nil
	}
	v, err := e.emitValue(rhs.Small)
	if err != nil {
		return nil, err
	}
	return []string{"return " + v + ";"}, nil
}

func indent(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		out = append(out, "  "+ln)
	}
	return out
}

// emitFnref lowers one reference creation to a self-applying
// thunk: each capture evaluates once into a fresh const, in
// written order, and the returned closure applies the target
// with parameters reconstructed in declaration order, the
// missing input arriving as the closure's parameter. An
// immediately invoked arrow keeps the whole creation an
// expression (usable in call arguments and constructions) while
// giving captures real temporaries instead of re-evaluated
// inline expressions.
func (e *emitter) emitFnref(node *Small) (string, error) {
	params, ok := e.params[node.Fname]
	if !ok {
		return "", fmt.Errorf("no params for reference target %s", node.Fname)
	}
	slots, unbound, err := bindRefSlots(node.Fname, node.Args, params)
	if err != nil {
		return "", err
	}
	if len(unbound) != 1 {
		return "", fmt.Errorf("reference to %s leaves %d parameters unbound", node.Fname, len(unbound))
	}
	var temps []string
	capOf := map[int]string{}
	for i, a := range node.Args {
		v, err := e.emitValue(a.V)
		if err != nil {
			return "", err
		}
		t := e.fresh()
		temps = append(temps, fmt.Sprintf("const %s = %s;", t, v))
		capOf[slots[i]] = t
	}
	parts := make([]string, 0, len(params))
	for i := range params {
		if i == unbound[0] {
			parts = append(parts, "input")
			continue
		}
		t, ok := capOf[i]
		if !ok {
			return "", fmt.Errorf("reference to %s misses capture %s", node.Fname, params[i][0])
		}
		parts = append(parts, t)
	}
	body := strings.Join(temps, " ")
	if body != "" {
		body += " "
	}
	body += fmt.Sprintf("return (input) => %s(%s);", node.Fname, strings.Join(parts, ", "))
	return fmt.Sprintf("(() => { %s })()", body), nil
}

// sigResultUnion renders one resolved invocation signature as its
// flattened Ok/error union: the same sorted Ok fields and error
// members as fnResultUnion, but over the signature's own success
// type and head error list. Invoke temporaries annotate with
// this so strict checkers narrow exactly like calls.
func (e *emitter) sigResultUnion(sig *invokeSig) (string, error) {
	if typeContainsFn(sig.in, e.recs) || typeContainsFn(sig.ret, e.recs) {
		return "", fmt.Errorf("invoke input and success must be data-only")
	}
	fields, ok := successFields(sig.ret, e.recs, e.variants[sig.ret] != nil || e.brands[sig.ret] != "")
	if !ok {
		return "", fmt.Errorf("invoke returns unknown type %s", sig.ret)
	}
	mapped := map[string]string{}
	for _, f := range fields {
		t, err := tsTypeB(f[1], e.brands, e.recs, e.variants, e.errTypes)
		if err != nil {
			return "", err
		}
		mapped[f[0]] = t
	}
	var names []string
	for n := range mapped {
		names = append(names, n)
	}
	sort.Strings(names)
	fs := ""
	for _, n := range names {
		fs += "; " + n + ": " + mapped[n]
	}
	union := okMember(fs)
	for _, kind := range sig.errs {
		efs, ok := e.errTypes[kind]
		if !ok {
			return "", fmt.Errorf("invoke emits unknown error %s", kind)
		}
		mem, err := tsErrFieldsMember(kind, efs, e.brands, e.recs, e.variants, e.errTypes)
		if err != nil {
			return "", err
		}
		union += " | " + mem
	}
	return union, nil
}

func (e *emitter) stmtMatch(node *Node, out *[]string) error {
	if !elaborated(node) {
		// Reaching codegen unelaborated is a compiler bug.
		return fmt.Errorf("match chain reached emit unelaborated")
	}
	if node.Kind == MatchInvoke {
		return e.stmtInvoke(node, out)
	}
	if node.Kind != MatchCall {
		return e.emitValueMatch(node, out)
	}
	scrut := node.Scruts[0]
	// Codegen has no program context, so the classifier answers
	// only the intrinsic question here; every non-intrinsic
	// (local, foreign, unknown) lowers through the union path.
	switch classifyCallee(nil, "", scrut.Fname) {
	case CalleeStoreOp:
		return e.stmtStoreOp(node, scrut, out)
	case CalleeDecParts:
		return e.stmtDecParts(node, scrut, out)
	case CalleeBytesDecode, CalleeBytesHexDecode, CalleeBytesB64Decode:
		return e.stmtBytesDecode(node, scrut, out)
	case CalleeBytesHexEncode:
		return e.stmtBytesHexEncode(node, scrut, out)
	case CalleeBytesB64Encode:
		return e.stmtBytesB64Encode(node, scrut, out)
	case CalleeBytesAsset:
		return e.stmtAssetFields(node, scrut, out)
	case CalleeBytesOther:
		return e.stmtBytesEncode(node, scrut, out)
	}
	union, ok := e.fnUnions[scrut.Fname]
	if !ok {
		return fmt.Errorf("no Result type for callee %s", scrut.Fname)
	}
	tmp := e.fresh()
	call, err := e.emitValue(scrut)
	if err != nil {
		return err
	}
	*out = append(*out, fmt.Sprintf("const %s: %s = %s;", tmp, union, call))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	return e.emitCallArms(node, tmp, out)
}

// stmtInvoke lowers one invocation: the closure applies to the
// emitted argument, and the outcome dispatches through the same
// arm chains as calls. A missing static certificate fails the
// build (never a guessed signature); a shadowed or non-callable
// target fails it the same way.
func (e *emitter) stmtInvoke(node *Node, out *[]string) error {
	if node.invokeSig == nil {
		return fmt.Errorf("invoke without a checked signature")
	}
	if len(node.Scruts) != 1 {
		return fmt.Errorf("invoke without a target")
	}
	s := node.Scruts[0]
	if s.Kind != "ref" || len(s.Ref) != 1 {
		return fmt.Errorf("invoke target is not a name")
	}
	if node.InvokeArg == nil {
		return fmt.Errorf("invoke without an argument")
	}
	union, err := e.sigResultUnion(node.invokeSig)
	if err != nil {
		return err
	}
	arg, err := e.emitValue(node.InvokeArg)
	if err != nil {
		return err
	}
	target, err := e.emitValue(s)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("const %s: %s = %s(%s);", tmp, union, target, arg))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	return e.emitCallArms(node, tmp, out)
}

// emitCallArms lowers every arm of a call match over an already-bound
// discriminated temporary: the generic foreign/local path and the
// fallible-kernel path share it, so decoder matches use ordinary
// success/error binding, never encoder single-success assumptions.
func (e *emitter) emitCallArms(node *Node, tmp string, out *[]string) error {
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		// Every arm body is block-scoped: the same binder name in
		// two arms (or a binder shadowing an outer one) must not
		// collide, and success and error binders lower uniformly
		// as one const bound to the matched union value.
		switch {
		case pat.Kind == "variantWild":
			*out = append(*out, fmt.Sprintf("case \"%s\": {", pat.Name))
			lines, err := e.retLines(arm.Rhs, nil)
			if err != nil {
				return err
			}
			*out = append(*out, indent(lines)...)
			*out = append(*out, "}")
		case pat.Kind == "variant" && pat.Name != "Ok":
			*out = append(*out, fmt.Sprintf("case \"%s\": {", pat.Name))
			*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, tmp))
			lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
			if err != nil {
				return err
			}
			*out = append(*out, indent(lines)...)
			*out = append(*out, "}")
		case pat.Kind == "variant" && pat.Name == "Ok":
			*out = append(*out, `case "ok": {`)
			binding, err := e.wholeSuccessBinding(pat, tmp)
			if err != nil {
				return err
			}
			*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, binding))
			lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
			if err != nil {
				return err
			}
			*out = append(*out, indent(lines)...)
			*out = append(*out, "}")
		default:
			return fmt.Errorf("call-match arm must be an error kind or Ok")
		}
	}
	// The checker proves arm coverage before emission, so default is
	// unreachable; it exists so strict checkers (whose narrowing gives
	// up through deeply nested switches) and readers both see a total
	// switch, and so a future emitter bug throws instead of returning
	// undefined.
	*out = append(*out, "default: {")
	*out = append(*out, "  throw new Error(\"unreachable\");")
	*out = append(*out, "}")
	*out = append(*out, "}")
	return nil
}

// emitValueMatch emits a value table of any arity 1..N: one arm loop
// for every shape. Scrutinee references preserve the legacy layout
// exactly — the single expression inline (no temporary, golden bytes),
// one $can_mN temporary per slot past arity 1 — so the only
// arity-dependent choice is output formatting, never match semantics.
// A conditionless arm owns its whole residual: trailing lines mid-chain
// with later arms dead (the historical wild shape, kept for arity-1
// wild arms too), a bare else at the end. A proved-total final arm
// reads the checker's bit, never a recomputed proof; anything else is a
// tested condition. Exhaustiveness already proven; emit assumes it.
func (e *emitter) emitValueMatch(node *Node, out *[]string) error {
	nslot := len(node.Scruts)
	// a75: variant elimination lowers through a tag switch, never
	// the bool/str condition chain. The checker proves the arms
	// total before emission, so this path assumes it.
	if nslot == 1 {
		if ot := childType(node.Scruts[0]); e.variants[ot] != nil {
			return e.emitVariantMatch(node, out)
		}
	}
	for _, arm := range node.Arms {
		if len(arm.Pats) != nslot {
			return fmt.Errorf("match arm has %d patterns; this match has %d scrutinees", len(arm.Pats), nslot)
		}
		for _, p := range arm.Pats {
			if k := p.Kind; k != "bool" && k != "str" && k != "wild" && k != "int" && k != "range" && k != "or" {
				return fmt.Errorf("variant pattern on a non-call match")
			}
		}
	}
	refs := make([]string, nslot)
	if nslot == 1 {
		sv, err := e.emitValue(node.Scruts[0])
		if err != nil {
			return err
		}
		refs[0] = sv
	} else {
		for i, s := range node.Scruts {
			v, err := e.emitValue(s)
			if err != nil {
				return err
			}
			t := e.fresh()
			*out = append(*out, fmt.Sprintf("const %s = %s;", t, v))
			refs[i] = t
		}
	}
	finalElse := false
	if node.analysis != nil {
		finalElse = node.analysis.emitFinalElse
	}
	known := make([]*bool, nslot)
	for n, arm := range node.Arms {
		last := n == len(node.Arms)-1
		conds, learn, contradiction := e.valueConds(arm, refs, known)
		kw := "if"
		if n > 0 {
			kw = "else if"
		}
		switch {
		case len(conds) == 0 && last && (nslot > 1 || !allWild(arm)):
			// Whole residual at the end: bare else. (Arity-1 wild
			// arms keep the historical trailing-lines shape below.)
			*out = append(*out, "else {")
		case len(conds) == 0:
			// Whole residual mid-chain with later arms dead (the
			// historical wild shape): trailing lines, chain ends.
			lines, err := e.retLines(arm.Rhs, nil)
			if err != nil {
				return err
			}
			*out = append(*out, lines...)
			return nil
		case last && finalElse:
			*out = append(*out, "else {")
		default:
			if contradiction {
				// Static duplicate past the proof: the raw
				// conjunction keeps the chain total; never reached.
				conds, _, _ = e.valueConds(arm, refs, nil)
			}
			*out = append(*out, fmt.Sprintf("%s (%s) {", kw, strings.Join(conds, " && ")))
			if learn >= 0 {
				nb := !arm.Pats[learn].B
				known[learn] = &nb
			}
		}
		lines, err := e.retLines(arm.Rhs, nil)
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	return nil
}

// emitVariantMatch lowers one variant elimination (a75): the
// scrutinee evaluates once into a generated temporary, the switch
// reads that temporary's tag, and each payload arm binds the same
// temporary so field access narrows through it. Case order follows
// source arms. The defensive default mirrors call matches: the
// checker proves totality, so reaching it throws instead of
// returning undefined.
func (e *emitter) emitVariantMatch(node *Node, out *[]string) error {
	sv, err := e.emitValue(node.Scruts[0])
	if err != nil {
		return err
	}
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("const %s = %s;", tmp, sv))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		if len(arm.Pats) != 1 {
			return fmt.Errorf("match arm has %d patterns; this match has 1 scrutinee", len(arm.Pats))
		}
		pat := arm.Pats[0]
		if !pat.isCase() {
			return fmt.Errorf("variant match arm must be a case")
		}
		*out = append(*out, fmt.Sprintf("case \"%s\": {", pat.Name))
		binder := pat.Var
		if pat.Kind == "variant" && binder != "" {
			*out = append(*out, fmt.Sprintf("  const %s = %s;", binder, tmp))
			lines, err := e.retLines(arm.Rhs, map[string]string{binder: binder})
			if err != nil {
				return err
			}
			*out = append(*out, indent(lines)...)
		} else {
			lines, err := e.retLines(arm.Rhs, nil)
			if err != nil {
				return err
			}
			*out = append(*out, indent(lines)...)
		}
		*out = append(*out, "}")
	}
	*out = append(*out, "default: {")
	*out = append(*out, "  throw new Error(\"unreachable\");")
	*out = append(*out, "}")
	*out = append(*out, "}")
	return nil
}

// allWild reports whether every slot pattern is a wildcard.
func allWild(arm Arm) bool {
	for _, p := range arm.Pats {
		if p.Kind != "wild" {
			return false
		}
	}
	return true
}

// valueConds renders one arm's test conjunction over the scrutinee
// references (inline expression or temporaries), dropping bool checks
// implied by failed earlier arms (known). Reports the slot when the
// conjunction is exactly one fresh bool test (its failure fixes that
// slot for later arms), and whether the arm contradicts known facts (a
// static duplicate: raw conditions still render, so the chain stays
// total).
// altCond renders one or-alternative as a boolean test over
// ref: bool, string, and integer renderings match the lone-arm
// shapes exactly, so `1 | 2` reads `(x === 1n || x === 2n)`.
// Unresolvable bounds render false (fail-closed past a world
// error, which already fails); the method takes no learning
// state because alternatives never teach contradictions.
func (e *emitter) altCond(alt Pattern, ref string) string {
	switch alt.Kind {
	case "bool":
		if !alt.B {
			return "!(" + ref + ")"
		}
		return ref
	case "str":
		return fmt.Sprintf("%s === %s", ref, normStr(alt.Str))
	case "int":
		if alt.Num == nil {
			return "false"
		}
		return fmt.Sprintf("%s === %sn", ref, alt.Num.String())
	case "range":
		if alt.Num == nil || alt.Hi == nil {
			return "false"
		}
		return fmt.Sprintf("(%s >= %sn && %s <= %sn)", ref, alt.Num.String(), ref, alt.Hi.String())
	default:
		return "false"
	}
}

func (e *emitter) valueConds(arm Arm, refs []string, known []*bool) (conds []string, learn int, contradiction bool) {
	learn = -1
	for i, p := range arm.Pats {
		switch p.Kind {
		case "wild":
		case "bool":
			if known != nil && known[i] != nil {
				if *known[i] == p.B {
					continue
				}
				contradiction = true
			}
			c := refs[i]
			if !p.B {
				c = "!(" + refs[i] + ")"
			}
			conds = append(conds, c)
			if known != nil && known[i] == nil && len(conds) == 1 {
				learn = i
			} else {
				learn = -1
			}
		case "str":
			conds = append(conds, fmt.Sprintf("%s === %s", refs[i], normStr(p.Str)))
			learn = -1
		case "int":
			// Slice 3: bigint-exact singleton comparison. No
			// contradiction learning: the proof owns overlap
			// and shadowing, emit only tests. Unresolved bounds
			// never survive the proof; false fails closed.
			if p.Num == nil {
				conds = append(conds, "false")
				learn = -1
				break
			}
			conds = append(conds, fmt.Sprintf("%s === %sn", refs[i], p.Num.String()))
			learn = -1
		case "range":
			// Slice 3: closed-interval lowering. Proof results
			// are consumed (final-else separation), never
			// invented here.
			if p.Num == nil || p.Hi == nil {
				conds = append(conds, "false")
				learn = -1
				break
			}
			conds = append(conds, fmt.Sprintf("(%s >= %sn && %s <= %sn)", refs[i], p.Num.String(), refs[i], p.Hi.String()))
			learn = -1
		case "or":
			// Slice 4: one source arm, unioned conditions. No
			// contradiction learning across alternatives: the
			// proof owns alternative usefulness (CAN4112).
			parts := make([]string, 0, len(p.Alts))
			for _, alt := range p.Alts {
				parts = append(parts, e.altCond(alt, refs[i]))
			}
			conds = append(conds, "("+strings.Join(parts, " || ")+")")
			learn = -1
		default:
			// Variant slots never survive the proof; render nothing so
			// the arm reads total rather than crashing the emitter.
			learn = -1
		}
	}
	return conds, learn, contradiction
}

// stmtStoreOp emits a cell operation as plain module state access: a
// get reads the cell into an ok+value union, a put assigns then
// proceeds. No imports: shared numeric code is emitted inline (a10).
// Only Ok-variant arms are legal past the exhaustiveness gate.
func (e *emitter) stmtStoreOp(node *Node, scrut *Small, out *[]string) error {
	cell, ok := storeCellName(scrut)
	t, known := e.cellTypes[cell]
	if !ok || !known {
		return fmt.Errorf("unknown cell in %s", scrut.Fname)
	}
	tmp := e.fresh()
	if scrut.Fname == "state__get" {
		*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\", value: %s } = { "+tsTag+": \"ok\", value: %s };", tmp, t, cell))
	} else {
		if len(scrut.Args) != 2 {
			return fmt.Errorf("cannot emit %s: want cell and value", scrut.Fname)
		}
		v, err := e.emitValue(scrut.Args[1].V)
		if err != nil {
			return err
		}
		*out = append(*out, fmt.Sprintf("%s = %s;", cell, v))
		*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\" } = { "+tsTag+": \"ok\" };", tmp))
	}
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		if pat.Kind != "variant" || pat.Name != "Ok" {
			return fmt.Errorf("store match arm must be Ok")
		}
		*out = append(*out, `case "ok": {`)
		*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, tmp))
		lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	*out = append(*out, "}")
	return nil
}

// stmtDecParts emits a dec-observation match: the operand evaluates
// once through $canDecParts into an ok-tagged pair, then the single
// Ok arm binds it like any other payload. Only Ok-variant arms are
// legal past the exhaustiveness gate.
func (e *emitter) stmtDecParts(node *Node, scrut *Small, out *[]string) error {
	if len(scrut.Args) != 1 {
		return fmt.Errorf("cannot emit %s: want one value", scrut.Fname)
	}
	v, err := e.emitValue(scrut.Args[0].V)
	if err != nil {
		return err
	}
	e.decOps["parts"] = true
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\", coefficient: bigint, scale: bigint } = { "+tsTag+": \"ok\", ...$canDecParts(%s) };", tmp, v))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		if pat.Kind != "variant" || pat.Name != "Ok" {
			return fmt.Errorf("dec__parts match arm must be Ok")
		}
		*out = append(*out, `case "ok": {`)
		*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, tmp))
		lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	*out = append(*out, "}")
	return nil
}

// stmtBytesEncode lowers UTF-8 encoding (a46 S2, a47 S3): the input
// string through TextEncoder into an Ok record of Uint8Array.
// Restricted calls require the certificate annotation; uncertified
// nodes fail loud and never fall through to the ordinary call path.
func (e *emitter) stmtBytesEncode(node *Node, scrut *Small, out *[]string) error {
	if isBytesExport(scrut.Fname) && scrut.ExportBrand == "" {
		return fmt.Errorf("cannot emit %s: no valid exports_utf8 grant certified this call", scrut.Fname)
	}
	var argv *Small
	if !isBytesExport(scrut.Fname) {
		slots, err := bindSlots(scrut.Fname, scrut.Args, bytesKernels[scrut.Fname].params)
		if err != nil {
			return fmt.Errorf("cannot emit %s: %s", scrut.Fname, err.Error())
		}
		for i, s := range slots {
			if s == 0 {
				argv = scrut.Args[i].V
			}
		}
	} else if len(scrut.Args) != 1 {
		return fmt.Errorf("cannot emit %s: want one value", scrut.Fname)
	} else {
		argv = scrut.Args[0].V
	}
	v, err := e.emitValue(argv)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\", value: Uint8Array } = { "+tsTag+": \"ok\", value: new TextEncoder().encode(%s) };", tmp, v))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		if pat.Kind != "variant" || pat.Name != "Ok" {
			return fmt.Errorf("bytes__utf8__export match arm must be Ok")
		}
		binding, err := e.wholeSuccessBinding(pat, tmp)
		if err != nil {
			return err
		}
		*out = append(*out, `case "ok": {`)
		*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, binding))
		lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	*out = append(*out, "}")
	return nil
}

// stmtAssetFields lowers witness projection (S2 slice plan): the asset
// string split into its eight canonical fields, refusing malformed
// witnesses loud, then the ordinary arm lowering over the bound
// Asset__Fields record. The policy handle evaluates (its contract is
// structural) but has no runtime projection yet: S4 binds it.
// Restricted calls require the certificate annotation; uncertified
// nodes fail loud and never fall through to the ordinary call path.
func (e *emitter) stmtAssetFields(node *Node, scrut *Small, out *[]string) error {
	if scrut.ExportBrand == "" {
		return fmt.Errorf("cannot emit %s: no valid asset_bridge grant certified this call", scrut.Fname)
	}
	if len(scrut.Args) != 2 || scrut.Args[0].HasName || scrut.Args[1].HasName {
		return fmt.Errorf("cannot emit %s: want the bare asset and policy values", scrut.Fname)
	}
	av, err := e.emitValue(scrut.Args[0].V)
	if err != nil {
		return err
	}
	pv, err := e.emitValue(scrut.Args[1].V)
	if err != nil {
		return err
	}
	union, err := decodeResultUnion(scrut.Fname, e.brands, e.recs, e.variants)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("void (%s);", pv))
	*out = append(*out, fmt.Sprintf("const %s: %s = ((witness: string): %s => { const parts = witness.split(\"|\"); if (parts.length !== 8) { throw new Error(\"asset witness malformed\"); } return { "+tsTag+": \"ok\", url: parts[2], digest: parts[3], role: parts[4] }; })(%s);", tmp, union, union, av))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	return e.emitCallArms(node, tmp, out)
}

// decodeResultUnion renders one fallible kernel's two-outcome TS
// type from the compiler-owned contracts (never literals): ok
// carrying the kernel's return record plus one member per emitted
// error. The kernel name selects the descriptor; an unknown name
// fails instead of falling back to another kernel's contract.
func decodeResultUnion(kernel string, brands map[string]string, recs map[string][][2]string, variants map[string]*VariantDecl) (string, error) {
	k, ok := bytesKernels[kernel]
	if !ok {
		return "", fmt.Errorf("no fallible contract for %s", kernel)
	}
	var td *TypeDecl
	for _, b := range builtinTypeDecls() {
		if b.Name == k.ret {
			td = b
		}
	}
	if td == nil {
		return "", fmt.Errorf("decode kernel %s returns unknown type %s", kernel, k.ret)
	}
	okm, err := okFieldsMember(td.Fields, brands, recs, variants, nil)
	if err != nil {
		return "", err
	}
	return resultUnion(okm, k.emits, brands, recs, variants, nil,
		func(e string) error { return fmt.Errorf("decode kernel %s emits unknown error %s", kernel, e) })
}

// stmtBytesDecode lowers the fallible kernels (a50 B6, a55 B10):
// the operand through the kernel's own strict helper into that
// kernel's two-outcome union. Matches use ordinary success/error
// binding via the shared arm lowering, never encoder single-success
// assumptions. Helper and flag select per kernel; nothing here may
// assume UTF-8 shapes.
func (e *emitter) stmtBytesDecode(node *Node, scrut *Small, out *[]string) error {
	slots, err := bindSlots(scrut.Fname, scrut.Args, bytesKernels[scrut.Fname].params)
	if err != nil {
		return fmt.Errorf("cannot emit %s: %s", scrut.Fname, err.Error())
	}
	var argv *Small
	for i, s := range slots {
		if s == 0 {
			argv = scrut.Args[i].V
		}
	}
	v, err := e.emitValue(argv)
	if err != nil {
		return err
	}
	var helper string
	switch {
	case isBytesDecode(scrut.Fname):
		helper = "$canUtf8Decode"
		e.utf8dec = true
	case isBytesHexDecode(scrut.Fname):
		helper = "$canHexDecode"
		e.hexdec = true
	case isBytesB64Decode(scrut.Fname):
		helper = "$canB64Decode"
		e.b64dec = true
	default:
		return fmt.Errorf("no fallible lowering for %s", scrut.Fname)
	}
	union, err := decodeResultUnion(scrut.Fname, e.brands, e.recs, e.variants)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	*out = append(*out, fmt.Sprintf("const %s: %s = %s(%s);", tmp, union, helper, v))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	return e.emitCallArms(node, tmp, out)
}

// stmtBytesHexEncode lowers hex encoding (a52 B8): the input Bytes
// through $canHexEncode into an Ok record of lowercase hex. Total
// kernel, so matches take the Ok arm only.
func (e *emitter) stmtBytesHexEncode(node *Node, scrut *Small, out *[]string) error {
	slots, err := bindSlots(scrut.Fname, scrut.Args, bytesKernels[scrut.Fname].params)
	if err != nil {
		return fmt.Errorf("cannot emit %s: %s", scrut.Fname, err.Error())
	}
	var argv *Small
	for i, s := range slots {
		if s == 0 {
			argv = scrut.Args[i].V
		}
	}
	v, err := e.emitValue(argv)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	e.hexenc = true
	*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\", value: string } = { "+tsTag+": \"ok\", value: $canHexEncode(%s) };", tmp, v))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		if pat.Kind != "variant" || pat.Name != "Ok" {
			return fmt.Errorf("bytes__hex__encode match arm must be Ok")
		}
		binding, err := e.wholeSuccessBinding(pat, tmp)
		if err != nil {
			return err
		}
		*out = append(*out, `case "ok": {`)
		*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, binding))
		lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	*out = append(*out, "}")
	return nil
}

// stmtBytesB64Encode lowers base64 encoding (a58 B12): the input
// Bytes through $canB64Encode into an Ok record of padded base64.
// Total kernel, so matches take the Ok arm only.
func (e *emitter) stmtBytesB64Encode(node *Node, scrut *Small, out *[]string) error {
	slots, err := bindSlots(scrut.Fname, scrut.Args, bytesKernels[scrut.Fname].params)
	if err != nil {
		return fmt.Errorf("cannot emit %s: %s", scrut.Fname, err.Error())
	}
	var argv *Small
	for i, s := range slots {
		if s == 0 {
			argv = scrut.Args[i].V
		}
	}
	v, err := e.emitValue(argv)
	if err != nil {
		return err
	}
	tmp := e.fresh()
	e.b64enc = true
	*out = append(*out, fmt.Sprintf("const %s: { "+tsTag+": \"ok\", value: string } = { "+tsTag+": \"ok\", value: $canB64Encode(%s) };", tmp, v))
	*out = append(*out, fmt.Sprintf("switch (%s."+tsTag+") {", tmp))
	for _, arm := range node.Arms {
		pat := arm.Pats[0]
		if pat.Kind != "variant" || pat.Name != "Ok" {
			return fmt.Errorf("bytes__base64__encode match arm must be Ok")
		}
		binding, err := e.wholeSuccessBinding(pat, tmp)
		if err != nil {
			return err
		}
		*out = append(*out, `case "ok": {`)
		*out = append(*out, fmt.Sprintf("  const %s = %s;", pat.Var, binding))
		lines, err := e.retLines(arm.Rhs, map[string]string{pat.Var: pat.Var})
		if err != nil {
			return err
		}
		*out = append(*out, indent(lines)...)
		*out = append(*out, "}")
	}
	*out = append(*out, "}")
	return nil
}

func (e *emitter) fn(fn *FnDecl, union string) ([]string, error) {
	// Per-function temp scope: every fresh() temporary lands as a
	// const inside this body, so numbering restarts at $can_m1 per
	// function. Editing one function no longer renumbers later
	// functions' goldens.
	e.tmp = 0
	var params []string
	for _, p := range fn.Params {
		t, err := tsTypeB(p[1], e.brands, e.recs, e.variants, e.errTypes)
		if err != nil {
			return nil, err
		}
		params = append(params, p[0]+": "+t)
	}
	lines := []string{fmt.Sprintf("export function %s(%s): %s {", fn.Name, strings.Join(params, ", "), union)}
	var body []string
	if fn.Body.IsMatch {
		if err := e.stmtMatch(fn.Body, &body); err != nil {
			return nil, err
		}
	} else {
		v, err := e.emitValue(fn.Body.Small)
		if err != nil {
			return nil, err
		}
		body = append(body, "return "+v+";")
	}
	lines = append(lines, indent(body)...)
	lines = append(lines, "}")
	return lines, nil
}

func emitModule(mod *Module, prog *Program, stemOf, resultOfStem map[string]string, fnUnions map[string]string) (string, error) {
	var L []string
	L = append(L, fmt.Sprintf("// GENERATED from %s by canlc v0.0.0. DO NOT EDIT.", mod.File))
	L = append(L, "// Prod emit: tests + given stripped.")
	// Declared records resolve to their emitted TS type of the same
	// name throughout this module. Declared variant parents (a74)
	// resolve to their emitted union type the same way.
	recs := recordShapes(prog.Modules)
	variants := variantShapes(prog.Modules)
	errs := errorShapes(prog.Modules)
	need := map[string]map[string]bool{}
	wantImport := func(fname string) {
		// b02: shared externs import from the
		// declaring stem's .externs stub below, never
		// from the module itself.
		if prog.Uses[fname] && prog.Externs[fname] == nil {
			if need[stemOf[fname]] == nil {
				need[stemOf[fname]] = map[string]bool{}
			}
			need[stemOf[fname]][fname] = true
		}
	}
	for _, d := range mod.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		for _, c := range walkCalls(fn.Body) {
			wantImport(c.Fname)
		}
		// A thunk calls its target by name, so an address-taken
		// foreign target imports exactly like a called one.
		for _, r := range walkFnrefs(fn.Body) {
			wantImport(r.Fname)
		}
	}
	// Foreign type references (a74): a field, case payload,
	// param, or Ret-record shape naming a type from another stem
	// imports that stem's type. Previously only called functions
	// (plus the Result type) were imported, so a cross-module
	// named type reference emitted unbound and failed strict tsc.
	// Same-stem names, bases, brands, and sequences thereof need
	// nothing: bases lower natively, brands erase, Seq unwraps.
	seenTypeRefs := map[string]bool{}
	var typeRefs func(t string)
	typeRefs = func(t string) {
		if seenTypeRefs[t] {
			return
		}
		seenTypeRefs[t] = true
		base := t
		if elem, ok := seqElemName(base); ok {
			base = elem
		}
		// Callable heads are structural, but their input and
		// success types may name foreign records: descend so
		// the closure annotation imports them.
		if a, r, kinds, ok := fnTypeShape(base); ok {
			typeRefs(a)
			typeRefs(r)
			// Fn results inline record fields and error payloads. Import
			// their named field types too, not only the parent record.
			for _, f := range recs[r] {
				typeRefs(f[1])
			}
			for _, kind := range strings.Split(kinds[1:len(kinds)-1], ",") {
				for _, f := range errs[strings.TrimSpace(kind)] {
					typeRefs(f[1])
				}
			}
			return
		}
		if _, ok := tsBase[base]; ok {
			return
		}
		stem, ok := stemOf[base]
		if !ok || stem == mod.Stem {
			return
		}
		if need[stem] == nil {
			need[stem] = map[string]bool{}
		}
		need[stem]["type "+base] = true
	}
	for _, d := range mod.Decls {
		switch d := d.(type) {
		case *TypeDecl:
			for _, f := range d.Fields {
				typeRefs(f[1])
			}
		case *ErrorDecl:
			for _, f := range d.Fields {
				typeRefs(f[1])
			}
		case *VariantDecl:
			for _, vc := range d.Cases {
				for _, f := range vc.Fields {
					typeRefs(f[1])
				}
			}
		case *FnDecl:
			for _, p := range d.Params {
				typeRefs(p[1])
			}
			if valueSuccess(d.Ret, variants[d.Ret] != nil) && prog.Brands[d.Ret] == "" {
				typeRefs(d.Ret)
			}
			if fs, ok := recs[d.Ret]; ok {
				for _, f := range fs {
					typeRefs(f[1])
				}
			}
		case *ExternDecl:
			for _, p := range d.Params {
				typeRefs(p[1])
			}
			if valueSuccess(d.Ret, variants[d.Ret] != nil) && prog.Brands[d.Ret] == "" {
				typeRefs(d.Ret)
			}
			if fs, ok := recs[d.Ret]; ok {
				for _, f := range fs {
					typeRefs(f[1])
				}
			}
		}
	}
	var stems []string
	for s := range need {
		stems = append(stems, s)
	}
	sort.Strings(stems)
	for _, stem := range stems {
		if stem == mod.Stem {
			continue
		}
		var names []string
		for n := range need[stem] {
			names = append(names, n)
		}
		sort.Strings(names)
		names = append(names, "type "+resultOfStem[stem])
		L = append(L, fmt.Sprintf("import { %s } from \"./%s\";", strings.Join(names, ", "), stem))
	}
	// Foreign imports: every extern this module calls is a typed TS
	// import from its host-provided stub file. Sorted for stability.
	exCalled := map[string]bool{}
	for _, d := range mod.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		for _, c := range walkCalls(fn.Body) {
			if prog.Externs[c.Fname] != nil {
				exCalled[c.Fname] = true
			}
		}
		for _, r := range walkFnrefs(fn.Body) {
			if prog.Externs[r.Fname] != nil {
				exCalled[r.Fname] = true
			}
		}
	}
	// b02: called externs group by declaring stem — own externs
	// keep the historical single line, shared ones import from
	// their owner's stub. One line per stem, sorted.
	exByStem := map[string][]string{}
	for n := range exCalled {
		stem := stemOf[n]
		if stem == "" {
			stem = mod.Stem
		}
		exByStem[stem] = append(exByStem[stem], n)
	}
	var exStems []string
	for s := range exByStem {
		exStems = append(exStems, s)
	}
	sort.Strings(exStems)
	for _, s := range exStems {
		names := exByStem[s]
		sort.Strings(names)
		L = append(L, fmt.Sprintf("import { %s } from \"./%s.externs\";", strings.Join(names, ", "), s))
	}
	cap_ := capitalize(mod.Mod) + "Result"
	var members []string
	for _, d := range mod.Decls {
		ed, ok := d.(*ErrorDecl)
		if !ok {
			continue
		}
		mem, err := tsErrMember(ed, prog.Brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		members = append(members, mem)
	}
	// Compiler-owned errors join the module union exactly when the
	// module's declared function outcomes require them: never
	// unconditionally, or every module's public union would carry
	// unrelated builtin members.
	for _, b := range builtinErrorDecls() {
		needed := false
		for _, d := range mod.Decls {
			switch d := d.(type) {
			case *FnDecl:
				for _, e := range d.Emits {
					if e == b.Name {
						needed = true
					}
				}
			case *ExternDecl:
				for _, e := range d.Emits {
					if e == b.Name {
						needed = true
					}
				}
			}
		}
		if !needed {
			continue
		}
		mem, err := tsErrMember(b, prog.Brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		members = append(members, mem)
	}
	// One ok member per distinct declared Ok shape, sorted for
	// stability. A single-shape module emits exactly the old union; a
	// file that mixes shapes (validators beside their consumers)
	// emits an ok union instead of failing the build. Shapes key on
	// names and types together, so value: bool and value: int stay
	// distinct members instead of collapsing by name.
	shapes := map[string]map[string]string{}
	for _, d := range mod.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		shape, err := declaredOkShape(fn, prog)
		if err != nil {
			return "", err
		}
		var keys []string
		for k := range shape {
			keys = append(keys, k+":"+shape[k])
		}
		sort.Strings(keys)
		key := strings.Join(keys, ",")
		if _, seen := shapes[key]; !seen {
			shapes[key] = shape
		}
	}
	var skeys []string
	for k := range shapes {
		skeys = append(skeys, k)
	}
	sort.Strings(skeys)
	var okMembers []string
	for _, k := range skeys {
		shape := shapes[k]
		var names []string
		for n := range shape {
			names = append(names, n)
		}
		sort.Strings(names)
		okFs := ""
		for _, n := range names {
			okFs += "; " + n + ": " + shape[n]
		}
		okMembers = append(okMembers, fmt.Sprintf("{ "+tsTag+": \"ok\"%s }", okFs))
	}
	union := strings.Join(okMembers, " | ")
	if union == "" {
		union = "{ " + tsTag + ": \"ok\" }"
	}
	for _, m := range members {
		union += " | " + m
	}
	L = append(L, fmt.Sprintf("export type %s = %s;", cap_, union))
	for _, d := range mod.Decls {
		td, ok := d.(*TypeDecl)
		if !ok {
			continue
		}
		var fs []string
		for _, f := range td.Fields {
			t, err := tsTypeB(f[1], prog.Brands, recs, variants, errs)
			if err != nil {
				return "", err
			}
			fs = append(fs, f[0]+": "+t)
		}
		if len(fs) == 0 {
			L = append(L, fmt.Sprintf("export type %s = {};", td.Name))
		} else {
			L = append(L, fmt.Sprintf("export type %s = { %s };", td.Name, strings.Join(fs, "; ")))
		}
	}
	// Variant unions (a74): one exported union type per variant
	// parent, one member per case in declaration order, qualified
	// tags as discriminants. Payload field types map through the
	// same rule as records, so nested variants resolve.
	for _, d := range mod.Decls {
		vd, ok := d.(*VariantDecl)
		if !ok {
			continue
		}
		var members []string
		for _, vc := range vd.Cases {
			mem, err := tsVariantMember(vd.Name, vc, prog.Brands, recs, variants, errs)
			if err != nil {
				return "", err
			}
			members = append(members, mem)
		}
		L = append(L, fmt.Sprintf("export type %s = %s;", vd.Name, strings.Join(members, " | ")))
	}
	// Module state: one mutable let per cell, initialized from the
	// decl literal. Tests prove per-scenario behavior from init;
	// prod shares the cell across calls (documented boundary).
	cellTypes := map[string]string{}
	params := map[string][][2]string{}
	for n, f := range prog.Fns {
		params[n] = f.Params
	}
	for n, ex := range prog.Externs {
		params[n] = ex.Params
	}
	em := &emitter{fnUnions: fnUnions, params: params, brands: prog.Brands, cellTypes: cellTypes, decOps: map[string]bool{}, strOps: map[string]bool{}, seqOps: map[string]bool{}, boolOps: map[string]bool{}, recs: recs, errFields: prog.Errors, errTypes: errorShapes(prog.Modules), variants: variants, cases: prog.Cases, divmod: false, consts: prog.Consts}
	for _, d := range mod.Decls {
		sd, ok := d.(*StateDecl)
		if !ok {
			continue
		}
		t, err := tsTypeB(sd.Type, prog.Brands, recs, variants, errs)
		if err != nil {
			return "", err
		}
		cellTypes[sd.Name] = t
		init, err := em.emitValue(sd.Init)
		if err != nil {
			return "", err
		}
		L = append(L, fmt.Sprintf("let %s: %s = %s;", sd.Name, t, init))
	}
	var fnLines []string
	for _, d := range mod.Decls {
		fn, ok := d.(*FnDecl)
		if !ok {
			continue
		}
		// Return annotations are per-function unions (see
		// fnResultUnion): the module union would lie about which
		// outcomes each function can produce.
		union, ok := fnUnions[fn.Name]
		if !ok {
			return "", fmt.Errorf("no Result type for function %s", fn.Name)
		}
		lines, err := em.fn(fn, union)
		if err != nil {
			return "", err
		}
		fnLines = append(fnLines, lines...)
	}
	// Exact-decimal runtime: emitted inline only when a dec operation
	// or ordering is used, so files without dec arithmetic gain no code.
	// Helpers carry $ prefixes, which can naming (domain__verb) cannot
	// spell, so user code can never collide with them.
	if len(em.decOps) > 0 {
		L = append(L, decHelpers(em.decOps)...)
	}
	// Strict boolean runtime: emitted inline only when and/or is
	// used, so files without boolean operators gain no code.
	if len(em.boolOps) > 0 {
		L = append(L, boolHelpers(em.boolOps)...)
	}
	// Byte-order string runtime: emitted inline only when a string
	// ordering is used, so files without one gain no code.
	if len(em.strOps) > 0 {
		L = append(L, strHelpers(em.strOps)...)
	}
	// Sequence runtime: emitted inline only when sequence indexing
	// is used, so files without one gain no code.
	if len(em.seqOps) > 0 {
		L = append(L, seqHelpers(em.seqOps)...)
	}
	// Structural equality runtime: emitted inline only when a record,
	// error-payload, or cell comparison is used. Shapes that can
	// contain Bytes get the typed-array branch exactly once.
	if em.recEq {
		if em.bytesEq {
			L = append(L, bytesEqHelpers...)
		} else {
			L = append(L, eqHelpers...)
		}
	}
	if em.divmod {
		L = append(L, divModHelper...)
	}
	// Strict UTF-8 decode runtime: emitted inline only when the decode
	// kernel is called, so files without one gain no code.
	if em.utf8dec {
		L = append(L, utf8DecodeHelper...)
	}
	// Hex encode runtime: emitted inline only when the hex encode
	// kernel is called, so files without one gain no code.
	if em.hexenc {
		L = append(L, hexEncodeHelper...)
	}
	// Strict hex decode runtime: emitted inline only when the hex
	// decode kernel is called, so files without one gain no code.
	if em.hexdec {
		L = append(L, hexDecodeHelper...)
	}
	// Base64 encode runtime: emitted inline only when the base64
	// encode kernel is called, so files without one gain no code.
	if em.b64enc {
		L = append(L, b64EncodeHelper...)
	}
	// Strict base64 decode runtime: emitted inline only when the
	// base64 decode kernel is called, so files without one gain
	// no code.
	if em.b64dec {
		L = append(L, b64DecodeHelper...)
	}
	L = append(L, fnLines...)
	// a46 S2: compiler-owned record definitions, emitted exactly when
	// referenced. References reach the output only through tsTypeB,
	// but at too many call sites to track explicitly; the
	// word-boundary scan cannot miss a spelling and cannot
	// false-positive on longer identifiers (__ has no boundary).
	joined := strings.Join(L, "\n")
	for _, b := range builtinTypeDecls() {
		if !regexp.MustCompile(`\b` + b.Name + `\b`).MatchString(joined) {
			continue
		}
		var fs []string
		for _, f := range b.Fields {
			t, err := tsTypeB(f[1], em.brands, recs, em.variants, em.errTypes)
			if err != nil {
				return "", err
			}
			fs = append(fs, f[0]+": "+t)
		}
		L = append(L, fmt.Sprintf("export type %s = { %s };", b.Name, strings.Join(fs, "; ")))
	}
	return strings.Join(L, "\n") + "\n", nil
}

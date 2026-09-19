package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// a77: revision identity enforcement. Fingerprints over resolved
// declarations plus transitive closures; baselines selected by the
// acceptance workflow; same-revision drift rejected. Interface/dependency
// fingerprints and pinned acceptance evidence have distinct inputs. Bodies,
// tests, positions, comments, checker annotations, and given tables never
// enter interface identity. Executable structure is serialized separately;
// proofs are recomputed, not authorized by a persisted executable hash.

// RevisionFormat versions the canonicalization. Incompatible
// baseline formats are refused, never silently rehashed.
const RevisionFormat = 2

// RevisionEntry is one declaration's identity record. Fingerprint
// covers the own canonical form plus the transitive dependency
// closure; Own covers only the declaration itself, so enforcement
// can tell a direct change from a rebound dependency. Fragment is
// the human structural summary carried into diagnostics. Canon
// holds the own canonical form for structural differencing;
// Detail carries the kind-specific structure the differ reads.
type RevisionEntry struct {
	Fingerprint string
	Own         string
	Fragment    string
	Deps        []string
	DepPrints   map[string]string
	Loc         string
	Canon       string
	Detail      RevisionDetail
}

// RevisionDetail is the kind-specific structure under a canonical
// form. Only the fields for the entry's kind are populated.
type RevisionDetail struct {
	Kind     string
	Params   [][2]string
	Ret      string
	Emits    []string
	Effects  []string
	Requires []string
	Ensures  []string
	Cases    []RevisionCase
	Fields   [][2]string
	Under    string
	Seals    []string
	Init     string
	Grant    string
}

// RevisionCase is one variant case in canonical form: qualified
// identity plus payload fields. Parent identity is part of the
// entry, so a moved case changes membership even when its tag
// is unchanged.
type RevisionCase struct {
	Qualified string
	Fields    [][2]string
}

// RevisionBaseline is the accepted-interface manifest. Origin
// names the review base that accepted it; Accepted distinguishes
// a proposal (generation output) from an authority. Scope lists
// the protected modules. The checker reports origin and scope;
// candidate-supplied baselines (Accepted false) are refused
// before any comparison.
type RevisionBaseline struct {
	Format   int                      `json:"format"`
	Origin   string                   `json:"origin"`
	Accepted bool                     `json:"accepted"`
	Scope    []string                 `json:"scope"`
	Entries  map[string]RevisionEntry `json:"entries"`
	// Pinned records trusted-acceptance rows (a87): row key to
	// canonical expectation rendering plus resolved reference dependencies.
	// Format 2 requires this section, including an empty map when there
	// are no pins. Format 1 omitted semantic inputs and is not authority.
	// Pins never enter interface fingerprints; changes warn (CAN6017).
	Pinned map[string]string `json:"pinned"`
}

// revisionKey names one declaration's identity: kind, canonical
// owner (module name), qualified name, and revision where the
// declaration carries one. Errors and cells carry no revision;
// their key omits it, and any change to them is drift.
func revisionKey(kind, mod, name string, rev int, hasRev bool) string {
	if hasRev {
		return fmt.Sprintf("%s:%s.%s@%d", kind, mod, name, rev)
	}
	return fmt.Sprintf("%s:%s.%s", kind, mod, name)
}

func canonStringList(xs []string) string {
	return "[" + strings.Join(xs, ",") + "]"
}

func canonFields(fs [][2]string) string {
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		parts = append(parts, f[0]+":"+canonType(f[1]))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// canonSmall serializes one expression structurally. Decoded
// literal values compare (%q distinguishes a raw backslash from
// an interpreted escape because the parser already decoded
// e-strings); checker annotations (T) and export certificates
// (ExportBrand) are derived data and excluded; argument order is
// significant (conservative: reorder is drift, never silent).
func canonSmall(s *Small) string {
	if s == nil {
		return "nil"
	}
	switch s.Kind {
	case "str":
		return "s:" + strconv.Quote(s.Str)
	case "int":
		if s.Num == nil {
			panic(canonicalError("int literal has no value"))
		}
		return "i:" + s.Num.String()
	case "bool":
		if s.B {
			return "b:1"
		}
		return "b:0"
	case "dec":
		return "d:" + s.Dec
	case "float":
		return "f:" + s.Str
	case "ref":
		return "ref(" + strings.Join(s.Ref, ".") + ")"
	case "seal":
		// The sealed value rides in Args, not Str (the parser
		// leaves Str empty): canon from the operand, or every
		// seal of one brand hashes identically and x->y drifts
		// silently. Hand-built Str seals keep the old shape.
		if len(s.Args) == 1 && s.Args[0].V != nil {
			return "seal(" + s.Seal + ":" + canonSmall(s.Args[0].V) + ")"
		}
		return "seal(" + s.Seal + ":" + strconv.Quote(s.Str) + ")"
	case "seqlit":
		parts := make([]string, 0, len(s.Items))
		for _, it := range s.Items {
			parts = append(parts, canonSmall(it))
		}
		return "seqlit(" + canonType(s.Elem) + ")[" + strings.Join(parts, ",") + "]"
	case "ctor", "call", "fnref":
		name := s.Fname
		if s.Kind == "ctor" {
			name = s.Ctor
		}
		return s.Kind + "(" + strconv.Quote(name) + ")types" + canonTypes(s.TypeArgs) + canonArgs(s.Args)
	case "binop":
		return "binop(" + s.Op + "," + canonSmall(s.L) + "," + canonSmall(s.R) + ")"
	case "strlen":
		return "strlen(" + canonSmall(s.L) + ")"
	case "stridx":
		return "stridx(" + canonSmall(s.L) + "," + canonSmall(s.R) + ")"
	case "proj":
		return "proj(" + canonSmall(s.L) + "," + s.Field + ")"
	case "strslice":
		return "strslice(" + canonSmall(s.L) + "," + canonSmall(s.R) + "," + canonSmall(s.Hi) + ")"
	case "exchange":
		parts := make([]string, 0, len(s.Args))
		for _, a := range s.Args {
			parts = append(parts, a.Name+"="+canonSmall(a.V))
		}
		return "exchange[" + strings.Join(parts, ",") + "]=>" + canonSmall(s.Outcome)
	case "list":
		parts := make([]string, 0, len(s.Items))
		for _, it := range s.Items {
			parts = append(parts, canonSmall(it))
		}
		return "list[" + strings.Join(parts, ",") + "]"
	case "wild":
		return "wild"
	case "not":
		return "not(" + canonSmall(s.L) + ")"
	case "neg":
		return "neg(" + canonSmall(s.L) + ")"
	case "forward":
		return "forward(" + s.Str + ")"
	default:
		panic(canonicalError("unsupported expression kind " + strconv.Quote(s.Kind)))
	}
}

// canonConst serializes a constant by its semantic content (R4
// clarification): the expanded typed value, not its spelling. A
// changed value can never hide behind a rename — renames change
// the entry key, values change the canonical form. Literals only
// (V1), so there are no nominal dependencies to record.
func canonConst(d *ConstDecl) string {
	return "const(" + d.Name + ")type(" + d.Type + ")value(" + canonSmall(d.Value) + ")"
}

// canonPattern serializes one match pattern by decoded meaning:
// raw spellings (Raw) are presentation, Str carries the value.
func canonPattern(p Pattern) string {
	switch p.Kind {
	case "wild":
		return "wild"
	case "bool":
		if p.B {
			return "bool:1"
		}
		return "bool:0"
	case "str":
		return "str:" + strconv.Quote(p.Str)
	case "int":
		if p.Num == nil {
			panic(canonicalError("int pattern has no value"))
		}
		return "int:" + p.Num.String()
	case "range":
		if p.Num == nil || p.Hi == nil {
			panic(canonicalError("unresolved range pattern"))
		}
		return "range:" + p.Num.String() + ".." + p.Hi.String()
	case "or":
		// Slice 4: alternatives canon in source order, each
		// like a lone pattern; elaboration already resolved
		// const and range bounds before identity runs.
		parts := make([]string, 0, len(p.Alts))
		for _, alt := range p.Alts {
			parts = append(parts, canonPattern(alt))
		}
		return "or:(" + strings.Join(parts, "|") + ")"
	case "variant", "variantWild":
		return p.Kind + ":" + strconv.Quote(p.Name) + ":" + strconv.Quote(p.Var) + ":types" + canonTypes(p.TypeArgs)
	default:
		panic(canonicalError("unsupported pattern kind " + strconv.Quote(p.Kind)))
	}
}

// canonNode serializes one body/match tree. Given tables are test
// evidence and excluded; arm order is significant.
func canonNode(n *Node) string {
	if n == nil {
		return "nil"
	}
	if !n.IsMatch {
		return canonSmall(n.Small)
	}
	if n.Kind != MatchValue && n.Kind != MatchCall && n.Kind != MatchInvoke {
		panic(canonicalError(fmt.Sprintf("unsupported/unelaborated match kind %d", n.Kind)))
	}
	parts := make([]string, 0, len(n.Scruts)+len(n.Arms)+1)
	if n.Kind == MatchInvoke {
		if n.InvokeArg == nil {
			panic(canonicalError("invocation has no argument"))
		}
		parts = append(parts, "with="+canonSmall(n.InvokeArg))
	}
	for _, s := range n.Scruts {
		parts = append(parts, canonSmall(s))
	}
	for _, a := range n.Arms {
		pats := make([]string, 0, len(a.Pats))
		for _, p := range a.Pats {
			pats = append(pats, canonPattern(p))
		}
		parts = append(parts, "["+strings.Join(pats, ",")+"]=>"+canonNode(a.Rhs))
	}
	return "match(" + strconv.Itoa(int(n.Kind)) + ")[" + strings.Join(parts, ";") + "]"
}

// canonContractArm serializes one ensures arm: outcome, binder,
// predicates, and Boolean match blocks in full. Dropping a match
// block weakens the contract, so Matches serialize whole.
func canonContractArm(a ContractArm) string {
	preds := make([]string, 0, len(a.Preds))
	for _, p := range a.Preds {
		preds = append(preds, canonSmall(p))
	}
	matches := make([]string, 0, len(a.Matches))
	for _, m := range a.Matches {
		matches = append(matches, canonNode(m))
	}
	return "on(" + a.Outcome + ":" + a.Bind + ")[" + strings.Join(preds, ",") + "][" + strings.Join(matches, ";") + "]"
}

func shaHex(s string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("can-revision/v%d\x00%s", RevisionFormat, s)))
	return hex.EncodeToString(sum[:])
}

// revisionTypeDeps collects the nominal type names inside one
// annotation: bare names verbatim, sequences unwrapped to their
// element, callable input/result/error members recursively traversed. Base
// types resolve to nothing; other names are dependency candidates mapped
// by the indexer to identity keys.
func revisionTypeDeps(t string, out map[string]bool) {
	t = canonType(t)
	if elem, ok := seqElemName(t); ok {
		revisionTypeDeps(elem, out)
		return
	}
	if a, r, es, ok := fnTypeShape(t); ok {
		revisionTypeDeps(a, out)
		revisionTypeDeps(r, out)
		for _, e := range strings.Split(strings.Trim(es, "[]"), ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				out[e] = true
			}
		}
		return
	}
	switch t {
	case "str", "int", "bool", "dec", "Bytes":
		return
	}
	if t == "" {
		return
	}
	out[t] = true
}

// revisionDeclEntries indexes one module's declarations into
// identity keys with their canonical form, fragment, detail, and
// named dependencies. Bodies, tests, positions, comments, and
// derived annotations never enter. First wins across modules,
// mirroring the checker and evaluator.
func revisionDeclEntries(m *Module, out map[string]*revisionWork) {
	for _, d := range m.Decls {
		_, dline := declNameLine(d)
		declLoc := fmt.Sprintf("%s:%d", m.File, dline)
		switch d := d.(type) {
		case *FnDecl:
			key := revisionKey("fn", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			params := make([][2]string, len(d.Params))
			copy(params, d.Params)
			deps := map[string]bool{}
			for _, p := range d.Params {
				revisionTypeDeps(p[1], deps)
			}
			revisionTypeDeps(d.Ret, deps)
			requires := make([]string, 0, len(d.Requires))
			for _, r := range d.Requires {
				requires = append(requires, canonSmall(r))
			}
			ensures := make([]string, 0, len(d.Ensures))
			for _, a := range d.Ensures {
				ensures = append(ensures, canonContractArm(a))
			}
			for _, e := range d.Emits {
				if strings.Contains(e, ".") {
					deps[e] = true
				}
			}
			for _, e := range d.Effects {
				if i := strings.LastIndex(e, "."); i >= 0 {
					deps[e[:i]] = true
				}
			}
			canon := "fn(" + d.Name + ")" +
				"params" + canonFields(params) +
				"ret(" + canonType(d.Ret) + ")" +
				"emits" + canonStringList(d.Emits) +
				"effects" + canonStringList(d.Effects) +
				"requires[" + strings.Join(requires, ",") + "]" +
				"ensures[" + strings.Join(ensures, ";") + "]"
			fragment := fmt.Sprintf("fn %s.%s(%s) -> %s emits %s effects %s requires %s ensures %d arms",
				m.Mod, d.Name, canonFields(params), d.Ret,
				canonStringList(d.Emits), canonStringList(d.Effects),
				presentAbsent(len(requires) > 0), len(ensures))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "fn", Params: params, Ret: d.Ret,
					Emits: append([]string{}, d.Emits...), Effects: append([]string{}, d.Effects...),
					Requires: requires, Ensures: ensures}}
		case *VariantDecl:
			key := revisionKey("variant", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			cases := make([]RevisionCase, 0, len(d.Cases))
			deps := map[string]bool{}
			for _, vc := range d.Cases {
				q := qualifyCase(d.Name, vc.Short)
				fields := make([][2]string, len(vc.Fields))
				copy(fields, vc.Fields)
				for _, f := range vc.Fields {
					revisionTypeDeps(f[1], deps)
				}
				cases = append(cases, RevisionCase{Qualified: q, Fields: fields})
			}
			// Cases are an unordered named set (a72 decisions
			// 1-3): pure reorder is not drift. Everything else
			// keeps declaration order.
			slices.SortFunc(cases, func(a, b RevisionCase) int {
				return strings.Compare(a.Qualified, b.Qualified)
			})
			parts := make([]string, 0, len(cases))
			frags := make([]string, 0, len(cases))
			for _, c := range cases {
				parts = append(parts, c.Qualified+canonFields(c.Fields))
				frags = append(frags, c.Qualified+canonFields(c.Fields))
			}
			canon := "variant(" + d.Name + ")[" + strings.Join(parts, ",") + "]"
			fragment := fmt.Sprintf("variant %s.%s@%d cases [%s]", m.Mod, d.Name, d.Rev, strings.Join(frags, ", "))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "variant", Cases: cases}}
		case *TypeDecl:
			key := revisionKey("type", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			fields := make([][2]string, len(d.Fields))
			copy(fields, d.Fields)
			deps := map[string]bool{}
			for _, f := range d.Fields {
				revisionTypeDeps(f[1], deps)
			}
			canon := "type(" + d.Name + ")" + canonFields(fields)
			fragment := fmt.Sprintf("type %s.%s@%d(%s)", m.Mod, d.Name, d.Rev, canonFields(fields))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "type", Fields: fields}}
		case *ErrorDecl:
			key := revisionKey("error", m.Mod, d.Name, 0, false)
			if _, seen := out[key]; seen {
				continue
			}
			fields := make([][2]string, len(d.Fields))
			copy(fields, d.Fields)
			deps := map[string]bool{}
			for _, f := range d.Fields {
				revisionTypeDeps(f[1], deps)
			}
			canon := "error(" + d.Name + ")" + canonFields(fields)
			fragment := fmt.Sprintf("error %s.%s(%s)", m.Mod, d.Name, canonFields(fields))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "error", Fields: fields}}
		case *BrandDecl:
			key := revisionKey("brand", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			seals := append([]string{}, d.SealsFrom...)
			canon := "brand(" + d.Name + ")under(" + d.Under + ")seals" + canonStringList(seals)
			fragment := fmt.Sprintf("brand %s.%s@%d is %s seals_from %s", m.Mod, d.Name, d.Rev, d.Under, canonStringList(seals))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: depsMap(),
				detail: RevisionDetail{Kind: "brand", Under: d.Under, Seals: seals}}
			revisionTypeDeps(d.Under, out[key].deps)
		case *ExternDecl:
			key := revisionKey("extern", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			params := make([][2]string, len(d.Params))
			copy(params, d.Params)
			deps := map[string]bool{}
			for _, p := range d.Params {
				revisionTypeDeps(p[1], deps)
			}
			revisionTypeDeps(d.Ret, deps)
			for _, e := range d.Emits {
				if strings.Contains(e, ".") {
					deps[e] = true
				}
			}
			canon := "extern(" + d.Name + ")" +
				"params" + canonFields(params) +
				"ret(" + canonType(d.Ret) + ")" +
				"emits" + canonStringList(d.Emits)
			fragment := fmt.Sprintf("extern %s.%s(%s) -> %s emits %s",
				m.Mod, d.Name, canonFields(params), d.Ret, canonStringList(d.Emits))
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "extern", Params: params, Ret: d.Ret, Emits: append([]string{}, d.Emits...)}}
		case *ConstDecl:
			key := revisionKey("const", m.Mod, d.Name, d.Rev, true)
			if _, seen := out[key]; seen {
				continue
			}
			canon := canonConst(d)
			fragment := fmt.Sprintf("const %s.%s@%d: %s", m.Mod, d.Name, d.Rev, d.Type)
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: depsMap(),
				detail: RevisionDetail{Kind: "const", Ret: d.Type, Init: canonSmall(d.Value)}}
		case *StateDecl:
			key := revisionKey("state", m.Mod, d.Name, 0, false)
			if _, seen := out[key]; seen {
				continue
			}
			canon := "state(" + d.Name + ")type(" + d.Type + ")init(" + canonSmall(d.Init) + ")"
			fragment := fmt.Sprintf("state %s.%s: %s", m.Mod, d.Name, d.Type)
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: depsMap(),
				detail: RevisionDetail{Kind: "state", Ret: d.Type, Init: canonSmall(d.Init)}}
		case *Utf8ExportDecl:
			key := "export:" + m.Mod + "." + d.Brand + " via " + d.Function + "@" + strconv.Itoa(d.Revision)
			if _, seen := out[key]; seen {
				continue
			}
			canon := "export(" + d.Brand + " via " + d.Function + "@" + strconv.Itoa(d.Revision) + ")"
			fragment := "exports_utf8 " + d.Brand + " via " + d.Function + "@" + strconv.Itoa(d.Revision)
			deps := map[string]bool{d.Brand: true, d.Function: true}
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "export", Grant: d.Brand + " via " + d.Function + "@" + strconv.Itoa(d.Revision)}}
		case *AssetBridgeDecl:
			key := "bridge:" + m.Mod + "." + d.Asset + "," + d.Policy + " from " + d.Owner + " via " + d.Function + "@" + strconv.Itoa(d.Revision) + " for " + d.Role
			if _, seen := out[key]; seen {
				continue
			}
			canon := "bridge(" + d.Asset + "," + d.Policy + " from " + d.Owner + " via " + d.Function + "@" + strconv.Itoa(d.Revision) + " for " + d.Role + ")"
			fragment := "asset_bridge " + d.Asset + ", " + d.Policy + " from " + d.Owner + " via " + d.Function + "@" + strconv.Itoa(d.Revision) + " for " + d.Role
			deps := map[string]bool{d.Asset: true, d.Policy: true, d.Function: true}
			out[key] = &revisionWork{canon: canon, fragment: fragment, loc: declLoc, deps: deps,
				detail: RevisionDetail{Kind: "bridge", Grant: d.Asset + " via " + d.Function + "@" + strconv.Itoa(d.Revision)}}
		}
	}
}

func depsMap() map[string]bool { return map[string]bool{} }

func presentAbsent(b bool) string {
	if b {
		return "present"
	}
	return "absent"
}

// revisionWork is one indexed declaration before fingerprinting.
type revisionWork struct {
	canon    string
	fragment string
	loc      string
	deps     map[string]bool
	detail   RevisionDetail
}

// FingerprintProgram computes every declaration's identity record:
// the own hash over its canonical form plus the transitive closure
// of referenced identities. Cycles cannot occur in admitted type
// graphs; the visited set still fails closed by naming the key.
func FingerprintProgram(prog *Program) map[string]RevisionEntry {
	work := map[string]*revisionWork{}
	for _, m := range prog.Modules {
		revisionDeclEntries(m, work)
	}
	memo := map[string]string{}
	var full func(key string, stack map[string]bool) string
	depKeysOf := func(w *revisionWork) []string {
		seen := map[string]bool{}
		var depKeys []string
		for name := range w.deps {
			for _, k := range resolveRevisionDep(name, work) {
				if !seen[k] {
					seen[k] = true
					depKeys = append(depKeys, k)
				}
			}
		}
		slices.Sort(depKeys)
		return depKeys
	}
	full = func(key string, stack map[string]bool) string {
		if h, ok := memo[key]; ok {
			return h
		}
		w, ok := work[key]
		if !ok {
			return "missing:" + key
		}
		if stack[key] {
			return "cycle:" + key
		}
		stack[key] = true
		depKeys := depKeysOf(w)
		// Pairs bind revision-stripped bases to content prints:
		// a pure re-key (rev 1 to rev 2, identical content) is
		// silent, while any content move still shifts the hash.
		pairs := make([]string, 0, len(depKeys))
		for _, k := range depKeys {
			pairs = append(pairs, depBaseName(k)+"\x01"+full(k, stack))
		}
		slices.Sort(pairs)
		h := shaHex(w.canon + "\x00" + strings.Join(pairs, ","))
		delete(stack, key)
		memo[key] = h
		return h
	}
	out := map[string]RevisionEntry{}
	for key, w := range work {
		deps := depKeysOf(w)
		prints := map[string]string{}
		for _, k := range deps {
			prints[k] = full(k, map[string]bool{})
		}
		out[key] = RevisionEntry{
			Fingerprint: full(key, map[string]bool{}),
			Own:         shaHex(w.canon),
			Fragment:    w.fragment,
			Deps:        deps,
			DepPrints:   prints,
			Loc:         w.loc,
			Canon:       w.canon,
			Detail:      w.detail,
		}
	}
	return out
}

// revisionAnchor locates a current declaration for an identity
// key by scanning modules in order. First wins, mirroring the
// checker.
func revisionAnchor(prog *Program, key string) (fileID string, line int) {
	kind := key[:strings.Index(key, ":")]
	rest := key[strings.Index(key, ":")+1:]
	base := rest
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		base = rest[:i]
	}
	mod, name, _ := strings.Cut(base, ".")
	short := name
	if kind == "export" {
		// rest is "mod.Brand via fn@rev": anchor the grant row.
		if b, _, ok := strings.Cut(rest, " via "); ok {
			if _, bn, ok := strings.Cut(b, "."); ok {
				short = bn
			}
		}
	}
	for _, m := range prog.Modules {
		if m.Mod != mod {
			continue
		}
		for _, d := range m.Decls {
			if d.declKind() != kind {
				continue
			}
			n, l := declNameLine(d)
			if kind == "export" {
				if ed, ok := d.(*Utf8ExportDecl); ok && ed.Brand == short {
					return m.ID, l
				}
				continue
			}
			if kind == "bridge" {
				if bd, ok := d.(*AssetBridgeDecl); ok && (bd.Asset == short || bd.Function == short) {
					return m.ID, bd.Line
				}
				continue
			}
			// Error and state names are dotted; compare the
			// full declared name against the key's short part.
			if n == short || n == name {
				return m.ID, l
			}
		}
	}
	return revisionFallback(prog)
}

// revisionFallback anchors baseline-level findings (untrusted or
// incompatible baselines, removals without a surviving module) at
// the first module's opening row.
func revisionFallback(prog *Program) (string, int) {
	best := ""
	for _, m := range prog.Modules {
		if best == "" || m.ID < best {
			best = m.ID
		}
	}
	return best, 1
}

// diffRevisionDetail compares two details of one kind, returning
// ordered change phrases (capped with a count remainder).
func diffRevisionDetail(kind string, old, new RevisionDetail) []string {
	var out []string
	fieldChanges := func(label string, o, n [][2]string) {
		om := map[string]string{}
		for _, f := range o {
			om[f[0]] = f[1]
		}
		nm := map[string]string{}
		for _, f := range n {
			nm[f[0]] = f[1]
		}
		for _, f := range o {
			nt, ok := nm[f[0]]
			if !ok {
				out = append(out, label+" removed field "+f[0])
			} else if nt != f[1] {
				out = append(out, label+" field "+f[0]+": "+f[1]+" -> "+nt)
			}
		}
		for _, f := range n {
			if _, ok := om[f[0]]; !ok {
				out = append(out, label+" added field "+f[0])
			}
		}
	}
	setChanges := func(label string, o, n []string) {
		om := map[string]bool{}
		for _, e := range o {
			om[e] = true
		}
		nm := map[string]bool{}
		for _, e := range n {
			nm[e] = true
		}
		var added, removed []string
		for _, e := range n {
			if !om[e] {
				added = append(added, e)
			}
		}
		for _, e := range o {
			if !nm[e] {
				removed = append(removed, e)
			}
		}
		slices.Sort(added)
		slices.Sort(removed)
		for _, e := range added {
			out = append(out, label+" added "+e)
		}
		for _, e := range removed {
			out = append(out, label+" removed "+e)
		}
	}
	switch kind {
	case "variant":
		om := map[string]RevisionCase{}
		for _, c := range old.Cases {
			om[c.Qualified] = c
		}
		nm := map[string]RevisionCase{}
		for _, c := range new.Cases {
			nm[c.Qualified] = c
		}
		var added, removed []string
		for _, c := range new.Cases {
			if _, ok := om[c.Qualified]; !ok {
				added = append(added, c.Qualified)
			}
		}
		for _, c := range old.Cases {
			if _, ok := nm[c.Qualified]; !ok {
				removed = append(removed, c.Qualified)
			}
		}
		slices.Sort(added)
		slices.Sort(removed)
		for _, q := range added {
			out = append(out, "added "+q)
		}
		for _, q := range removed {
			out = append(out, "removed "+q)
		}
		for _, c := range new.Cases {
			if oc, ok := om[c.Qualified]; ok {
				of := map[string]string{}
				for _, f := range oc.Fields {
					of[f[0]] = f[1]
				}
				nf := map[string]string{}
				for _, f := range c.Fields {
					nf[f[0]] = f[1]
				}
				for _, f := range oc.Fields {
					if nt, ok := nf[f[0]]; !ok {
						out = append(out, c.Qualified+" removed field "+f[0])
					} else if nt != f[1] {
						out = append(out, c.Qualified+" field "+f[0]+": "+f[1]+" -> "+nt)
					}
				}
				for _, f := range c.Fields {
					if _, ok := of[f[0]]; !ok {
						out = append(out, c.Qualified+" added field "+f[0])
					}
				}
			}
		}
	case "fn", "extern":
		if len(old.Params) != len(new.Params) {
			out = append(out, fmt.Sprintf("params arity %d -> %d", len(old.Params), len(new.Params)))
		}
		for i := 0; i < len(old.Params) && i < len(new.Params); i++ {
			o, n := old.Params[i], new.Params[i]
			if o[0] != n[0] {
				out = append(out, fmt.Sprintf("param %d renamed %s -> %s", i, o[0], n[0]))
			} else if o[1] != n[1] {
				out = append(out, fmt.Sprintf("param %s: %s -> %s", o[0], o[1], n[1]))
			}
		}
		if old.Ret != new.Ret {
			out = append(out, "returns "+old.Ret+" -> "+new.Ret)
		}
		setChanges("emits", old.Emits, new.Emits)
		if kind == "fn" {
			setChanges("effects", old.Effects, new.Effects)
			ro, rn := len(old.Requires) > 0, len(new.Requires) > 0
			switch {
			case ro && !rn:
				out = append(out, "requires removed")
			case !ro && rn:
				out = append(out, "requires added")
			case strings.Join(old.Requires, "\x00") != strings.Join(new.Requires, "\x00"):
				out = append(out, "requires changed")
			}
			om := map[string]string{}
			for _, a := range old.Ensures {
				om[ensuresOutcome(a)] = a
			}
			nm := map[string]string{}
			for _, a := range new.Ensures {
				nm[ensuresOutcome(a)] = a
			}
			var added, removed []string
			for oc := range nm {
				if _, ok := om[oc]; !ok {
					added = append(added, oc)
				}
			}
			for oc := range om {
				if _, ok := nm[oc]; !ok {
					removed = append(removed, oc)
				}
			}
			slices.Sort(added)
			slices.Sort(removed)
			for _, oc := range added {
				out = append(out, "ensures added "+oc)
			}
			for _, oc := range removed {
				out = append(out, "ensures removed "+oc)
			}
			for oc, na := range nm {
				if oa, ok := om[oc]; ok && oa != na {
					out = append(out, "ensures changed "+oc)
				}
			}
		}
	case "type", "error":
		fieldChanges(kind, old.Fields, new.Fields)
	case "brand":
		if old.Under != new.Under {
			out = append(out, "underlying "+old.Under+" -> "+new.Under)
		}
		setChanges("seals_from", old.Seals, new.Seals)
	case "state":
		if old.Ret != new.Ret {
			out = append(out, "type "+old.Ret+" -> "+new.Ret)
		}
		if old.Init != new.Init {
			out = append(out, "init changed")
		}
	case "const":
		if old.Ret != new.Ret {
			out = append(out, "type "+old.Ret+" -> "+new.Ret)
		}
		if old.Init != new.Init {
			out = append(out, "value changed")
		}
	case "export":
		if old.Grant != new.Grant {
			out = append(out, "grant "+old.Grant+" -> "+new.Grant)
		}
	}
	if len(out) > 4 {
		out = append(out[:4], fmt.Sprintf("and %d more", len(out)-4))
	}
	return out
}

// ensuresOutcome indexes one canonical ensures arm by its outcome:
// the outcome token before the first binder/parenthesis.
func ensuresOutcome(canon string) string {
	rest := strings.TrimPrefix(canon, "on(")
	if i := strings.IndexAny(rest, ":)"); i >= 0 {
		return rest[:i]
	}
	return rest
}

// CheckRevisionIdentity enforces an accepted baseline over a
// resolved program: unknown formats and unaccepted baselines are
// refused before any comparison; shared identities compare
// fingerprints; baseline-only identities report REMOVED and
// candidate-only identities report ADDED. A nil baseline selects
// no enforcement. Diagnostics sort by message for determinism.
func CheckRevisionIdentity(prog *Program, texts map[string]string, base *RevisionBaseline) (out []Diag) {
	defer canonicalDiags(prog, texts, &out)
	if base == nil {
		return nil
	}
	mkspan := func(msg, token, code string) Diag {
		fileID, line := revisionFallback(prog)
		text := ""
		if t, ok := texts[fileID]; ok {
			text = t
		}
		return spanDiag(text, line, "error", msg, token, code)
	}
	if base.Format != RevisionFormat {
		return []Diag{mkspan(fmt.Sprintf("unsupported revision format %d (want %d): generate a new candidate with a compatible canlc, then explicitly review and re-accept",
			base.Format, RevisionFormat), "mod", CodeRevisionIdentity)}
	}
	if !base.Accepted {
		return []Diag{mkspan(fmt.Sprintf("not an accepted baseline (origin %q): generation is not acceptance; select an accepted baseline from the review base",
			base.Origin), "mod", CodeRevisionIdentity)}
	}
	if base.Pinned == nil {
		return []Diag{mkspan("incomplete revision baseline: missing pinned evidence section; regenerate and explicitly review/re-accept", "mod", CodeRevisionIdentity)}
	}
	validateCanonicalProgram(prog)
	current := FingerprintProgram(prog)
	anchor := func(key string) (string, int) {
		fileID, line := revisionAnchor(prog, key)
		if fileID == "" {
			return revisionFallback(prog)
		}
		return fileID, line
	}
	spanFor := func(key, token string) (string, int, string) {
		fileID, line := anchor(key)
		return texts[fileID], line, token
	}
	var keys []string
	for k := range current {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		cur := current[key]
		old, ok := base.Entries[key]
		if !ok {
			text, line, token := spanFor(key, shortRevisionName(key))
			d := spanDiag(text, line, "error",
				fmt.Sprintf("%s is new since accepted baseline %q: accept by updating the baseline", key, base.Origin),
				token, CodeRevisionIdentity)
			d.Expected = fmt.Sprintf("baseline %q scope %s: no such identity", base.Origin, canonStringList(base.Scope))
			d.Found = "ADDED " + cur.Fragment
			d.Hint = "accept the new identity by updating the accepted baseline, or remove the declaration"
			out = append(out, d)
			continue
		}
		if cur.Fingerprint == old.Fingerprint {
			continue
		}
		var changes []string
		if cur.Own != old.Own {
			changes = diffRevisionDetail(cur.Detail.Kind, old.Detail, cur.Detail)
		} else {
			changes = reboundRevisionDeps(old, cur, current)
		}
		if len(changes) == 0 {
			changes = []string{"changed"}
		}
		short := shortRevisionName(key)
		text, line, token := spanFor(key, short)
		d := spanDiag(text, line, "error",
			fmt.Sprintf("%s differs from accepted baseline %q", shortRev(key), base.Origin),
			token, CodeRevisionIdentity)
		d.Expected = fmt.Sprintf("baseline %q: %s", base.Origin, old.Fragment)
		d.Found = strings.Join(changes, "; ")
		d.Hint = "restore the accepted interface, or publish a reviewed new revision and explicitly update affected pins"
		out = append(out, d)
	}
	var oldKeys []string
	for k := range base.Entries {
		if _, ok := current[k]; !ok {
			oldKeys = append(oldKeys, k)
		}
	}
	slices.Sort(oldKeys)
	for _, key := range oldKeys {
		old := base.Entries[key]
		text, line, _ := spanFor(key, "mod")
		d := spanDiag(text, line, "error",
			fmt.Sprintf("%s was removed since accepted baseline %q: accept the removal by updating the baseline", shortRev(key), base.Origin),
			"mod", CodeRevisionIdentity)
		d.Expected = fmt.Sprintf("baseline %q: %s", base.Origin, old.Fragment)
		if old.Loc != "" {
			d.Expected += " at " + old.Loc
		}
		d.Found = "REMOVED " + old.Fragment
		d.Hint = "restore the declaration, or accept the removal by updating the accepted baseline"
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Diag) int {
		if a.Msg != b.Msg {
			return strings.Compare(a.Msg, b.Msg)
		}
		return strings.Compare(a.File, b.File)
	})
	return out
}

// reboundRevisionDeps names the changed dependencies behind a
// fingerprint move whose own form is untouched. Dependencies group
// by revision-stripped base name, so a pure re-key (rev 1 to rev 2
// with identical content) stays silent while content moves report.
func reboundRevisionDeps(old, cur RevisionEntry, current map[string]RevisionEntry) []string {
	group := func(prints map[string]string) map[string]map[string]bool {
		g := map[string]map[string]bool{}
		for k, p := range prints {
			b := depBaseName(k)
			if g[b] == nil {
				g[b] = map[string]bool{}
			}
			g[b][p] = true
		}
		return g
	}
	oldPrints := map[string]string{}
	for k, p := range old.DepPrints {
		oldPrints[k] = p
	}
	newPrints := map[string]string{}
	for _, k := range cur.Deps {
		if now, ok := current[k]; ok {
			newPrints[k] = now.Fingerprint
		}
	}
	oldG, newG := group(oldPrints), group(newPrints)
	seen := map[string]bool{}
	var out []string
	for b := range oldG {
		seen[b] = true
	}
	for b := range newG {
		seen[b] = true
	}
	var bases []string
	for b := range seen {
		bases = append(bases, b)
	}
	slices.Sort(bases)
	for _, b := range bases {
		o, n := oldG[b], newG[b]
		if equalPrintSets(o, n) {
			continue
		}
		short := b
		if _, nm, ok := strings.Cut(b, "."); ok {
			short = nm
		}
		switch {
		case len(o) == 0:
			out = append(out, "rebound dependency added "+short)
		case len(n) == 0:
			out = append(out, "rebound dependency removed "+short)
		default:
			out = append(out, "rebound dependency changed "+short)
		}
	}
	return out
}

func equalPrintSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for p := range a {
		if !b[p] {
			return false
		}
	}
	return true
}

// depBaseName strips kind and revision from an identity key,
// leaving the module-qualified base a dependency groups by.
func depBaseName(key string) string {
	rest := key[strings.Index(key, ":")+1:]
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

func shortRevisionName(key string) string {
	rest := key[strings.Index(key, ":")+1:]
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest = rest[:i]
	}
	if _, n, ok := strings.Cut(rest, "."); ok {
		return n
	}
	return rest
}

func shortRev(key string) string {
	rest := key[strings.Index(key, ":")+1:]
	if i := strings.Index(rest, "."); i >= 0 {
		return rest[i+1:]
	}
	return rest
}

// PinnedRow is one decision-table row with its acceptance status
// (a87): the fn identity key, short owner name, test name, source
// line, canonical rendering, and whether the row carries the marker.
type PinnedRow struct {
	FnKey  string
	FnName string
	Test   string
	Line   int
	Render string
	Pinned bool
	FileID string
	File   string
}

// canonPinnedRow renders one decision-table row canonically: test name,
// argument bindings in declaration order, and the canonical expectation.
// canonSmall already normalizes literals, so formatting churn is not
// drift; argument order stays significant, matching the fingerprint.
func canonPinnedRow(t Test) string {
	parts := make([]string, 0, len(t.Args))
	for i, a := range t.Args {
		name := a.Name
		if !a.HasName {
			name = fmt.Sprintf("#%d", i)
		}
		v := "nil"
		if a.V != nil {
			v = canonSmall(a.V)
		}
		parts = append(parts, name+"="+v)
	}
	return t.Name + "(" + strings.Join(parts, ",") + ")binds" + canonFields(t.TypeBinds) + " => " + canonSmall(t.Expected)
}

// pinRowKey names one acceptance row: the fn identity key plus test
// name. The rev rides inside the fn key, so a rev bump orphans pins
// the same way it revokes identity — the identity error owns that
// case, and the pinned check skips fns absent on either side.
func pinRowKey(fnKey, test string) string {
	return fnKey + "/" + test
}

// pinnedRowWalk collects every decision-table row in the program keyed
// by acceptance key. Only marked rows enter the baseline; unmarked rows
// are proposed evidence whose churn is silent by construction.
func pinnedRowWalk(prog *Program) map[string]PinnedRow {
	rows := map[string]PinnedRow{}
	entries := FingerprintProgram(prog)
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok {
				continue
			}
			fnKey := revisionKey("fn", m.Mod, fn.Name, fn.Rev, true)
			for _, t := range fn.Tests {
				rows[pinRowKey(fnKey, t.Name)] = PinnedRow{
					FnKey: fnKey, FnName: m.Mod + "." + fn.Name,
					Test: t.Name, Line: t.Line, Render: canonPinnedRow(t) + canonRowDependencies(prog, entries, t),
					Pinned: t.Pinned, FileID: m.ID, File: m.File,
				}
			}
		}
	}
	return rows
}

// PinnedRows collects marked rows: acceptance key to canonical
// rendering. Generation stores exactly this map, so regen output is
// deterministic (encoding/json sorts map keys).
func PinnedRows(prog *Program) map[string]string {
	out := map[string]string{}
	for k, r := range pinnedRowWalk(prog) {
		if r.Pinned {
			out[k] = r.Render
		}
	}
	return out
}

// CheckPinnedRows reports A-light weakening (a87): a pinned expectation
// that changed, a pinned row that vanished, or a pin demoted back to
// proposed since the accepted baseline. Advisory only: severity warning,
// not a build error. Canonicalization failures, unlike weakening, fail closed
// with an error. Runs only
// against accepted baselines in the current format — identity owns
// every other complaint. Fns whose identity is absent on either side
// are skipped: added, removed, or revved declarations already report
// through CAN6013, and new pins record silently on regen.
func CheckPinnedRows(prog *Program, texts map[string]string, base *RevisionBaseline) (out []Diag) {
	defer canonicalDiags(prog, texts, &out)
	if base == nil || base.Format != RevisionFormat || !base.Accepted || len(base.Pinned) == 0 {
		return nil
	}
	rows := pinnedRowWalk(prog)
	fnKeys := map[string]bool{}
	for _, m := range prog.Modules {
		for _, d := range m.Decls {
			if fn, ok := d.(*FnDecl); ok {
				fnKeys[revisionKey("fn", m.Mod, fn.Name, fn.Rev, true)] = true
			}
		}
	}
	warn := func(row PinnedRow, line int, verb, expected, found string) {
		d := spanDiag(texts[row.FileID], line, "warning",
			fmt.Sprintf("pinned row %s/%s %s since accepted baseline %q", row.FnName, row.Test, verb, base.Origin),
			row.Test, CodePinnedWeakened)
		d.File = row.File
		d.Expected = expected
		d.Found = found
		d.Hint = "restore the accepted expectation, or re-accept by updating the baseline"
		out = append(out, d)
	}
	var keys []string
	for k := range rows {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		row := rows[k]
		old, ok := base.Pinned[k]
		if ok && row.Pinned && old != row.Render {
			warn(row, row.Line, "weakened", old, row.Render)
			continue
		}
		if ok && !row.Pinned {
			warn(row, row.Line, "demoted to proposed", old, row.Render)
		}
	}
	var oldKeys []string
	for k := range base.Pinned {
		if _, ok := rows[k]; !ok {
			oldKeys = append(oldKeys, k)
		}
	}
	slices.Sort(oldKeys)
	for _, k := range oldKeys {
		sep := strings.LastIndex(k, "/")
		if sep < 0 || !fnKeys[k[:sep]] {
			continue
		}
		fileID, line := revisionAnchor(prog, k[:sep])
		file := fileID
		for _, m := range prog.Modules {
			if m.ID == fileID {
				file = m.File
			}
		}
		d := spanDiag(texts[fileID], line, "warning",
			fmt.Sprintf("pinned row %s weakened since accepted baseline %q: row removed", strings.TrimPrefix(k, "fn:"), base.Origin),
			"mod", CodePinnedWeakened)
		d.File = file
		d.Expected = base.Pinned[k]
		d.Found = "absent"
		d.Hint = "restore the accepted row, or re-accept by updating the baseline"
		out = append(out, d)
	}
	return out
}

// WriteBaseline generates a candidate (unaccepted) baseline from a
// resolved program. Generation is deterministic: sorted modules,
// sorted entry keys. Accepted stays false: generation is not
// acceptance, and the checker refuses unaccepted baselines.
func WriteBaseline(path string, prog *Program, origin string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(canonicalError); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	validateCanonicalProgram(prog)
	mods := map[string]bool{}
	for _, m := range prog.Modules {
		mods[m.Mod] = true
	}
	var scope []string
	for m := range mods {
		scope = append(scope, m)
	}
	slices.Sort(scope)
	base := &RevisionBaseline{
		Format:   RevisionFormat,
		Origin:   origin,
		Accepted: false,
		Scope:    scope,
		Entries:  FingerprintProgram(prog),
		Pinned:   PinnedRows(prog),
	}
	raw, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// LoadBaseline reads a baseline file. Format and acceptance are
// checked by enforcement, not here: loading never validates
// authority.
func LoadBaseline(path string) (*RevisionBaseline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var base RevisionBaseline
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, err
	}
	if base.Entries == nil {
		base.Entries = map[string]RevisionEntry{}
	}
	return &base, nil
}

// resolveRevisionDep maps one dependency name to indexed identity
// keys. Dotted error kinds match error entries verbatim; cell
// capabilities match state entries by cell name; bare type, brand,
// and function names match every same-named declaration. Taking
// all cross-owner matches into the closure can only over- rather
// than under-approximate drift, and first-wins keeps it
// deterministic with the checker.
func resolveRevisionDep(name string, work map[string]*revisionWork) []string {
	seen := map[string]bool{}
	var out []string
	add := func(k string) {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for key := range work {
		kind := key[:strings.Index(key, ":")]
		rest := key[strings.Index(key, ":")+1:]
		base := rest
		if i := strings.LastIndex(rest, "@"); i >= 0 {
			base = rest[:i]
		}
		// base is "mod.Name" (dotted for error kinds).
		if base == name {
			add(key)
			continue
		}
		if _, n, ok := strings.Cut(base, "."); ok && n == name {
			add(key)
			continue
		}
		if kind == "export" {
			// rest is "mod.Brand via fn@rev": match the brand
			// or the granted function by short name.
			if b, f, ok := strings.Cut(rest, " via "); ok {
				if _, bn, ok := strings.Cut(b, "."); ok && bn == name {
					add(key)
					continue
				}
				fn := f
				if i := strings.LastIndex(fn, "@"); i >= 0 {
					fn = fn[:i]
				}
				if fn == name {
					add(key)
				}
			}
		}
	}
	return out
}

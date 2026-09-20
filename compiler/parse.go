package main

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/internal/scan"
)

// ---------------------------------------------------------------- AST ------
type Arg struct {
	Name    string
	HasName bool
	V       *Small
}

type Small struct {
	Kind string // str,int,bool,dec,float,wild,binop,call,ctor,list,ref,seal,exchange,strlen,stridx,strslice,seqlit,forward,not,neg,proj,fnref
	Str  string
	// Outcome holds a scripted result for Kind exchange: the row proves
	// "this request received this permitted response" (a12).
	Outcome *Small
	// Num holds an int literal of arbitrary size (a10: ints are
	// mathematically unbounded, so literals never overflow).
	Num *big.Int
	// T holds the checker's static type for ref/binop/seal nodes
	// (a10: emit reads it to choose bigint-native vs exact-decimal
	// code, so code generation reasons over typed structure).
	T string
	B bool
	// Dec holds canonical decimal digits for Kind dec: -?\d+\.\d+ with
	// no trailing fractional zeros (d"1.50" parses to "1.5").
	Dec string
	// Seal holds the brand name for Kind seal; the sealed value
	// rides in Args[0].V (Str stays empty).
	Seal string
	// Elem holds the element type name for Kind seqlit
	// (a36 S1: typed sequence literals Seq<T>[...]).
	Elem string
	Op   string
	L, R *Small
	// Hi holds the slice end for Kind strslice (base L, start R).
	Hi *Small
	// Field holds the one projected segment for Kind proj (base
	// L): m[i].a.b nests two proj nodes, never a path.
	Field string
	Fname string
	// TypeArgs holds explicit instantiation arguments on a call
	// node (G1): `call f<str>(...)`. Empty for monomorphic
	// calls. Expansion rewrites the call to its stamped copy.
	TypeArgs []string
	Args     []Arg
	Ctor     string
	Items    []*Small
	Ref      []string
	// ExportBrand names the granted brand on a certified
	// bytes__utf8__export call node (a46 S2). Set only by
	// certifyExports after full validation; empty means
	// uncertified, and both evaluator and emitter refuse it.
	ExportBrand string
}

type Pattern struct {
	Kind     string // wild,bool,str,int,range,const,variant,variantWild,or
	B        bool
	Str      string
	Name     string
	Var      string
	TypeArgs []string // case args are erased; typed Ok retains its resolved success annotation
	// Alts holds `|` alternatives for Kind "or", in source
	// order. Credit and coverage union over them; each
	// alternative parses like a lone slot pattern, so ranges
	// bind tighter than `|` and `|` tighter than the comma.
	Alts []Pattern
	// Num holds an int singleton value, or a range lower bound
	// once resolved. Hi holds a range upper bound (nil for
	// singletons). LoS/HiS carry unresolved range bound text
	// (integer literals resolve at parse; anything else resolves
	// in buildWorld, so const bounds see the finished table);
	// resolved ranges have Num/Hi set and empty LoS/HiS.
	Num *big.Int
	Hi  *big.Int
	LoS string
	HiS string

	// Raw is the source spelling of a str pattern (a66): the
	// squiggle locator needs the verbatim token because a
	// decoded interpreted literal is not searchable in source.
	Raw string
}

// isCase reports whether a pattern names a variant case by shape
// (a75): a qualified __ name that is neither Ok nor a dotted
// error kind. Shape only — membership is the checker's job, so
// unknown and wrong-union names still parse. Every phase uses
// this one predicate, so error-protocol patterns (Ok, dotted)
// never leak into the elimination path.
func (p Pattern) isCase() bool {
	if p.Kind != "variant" && p.Kind != "variantWild" {
		return false
	}
	return p.Name != "Ok" && !strings.Contains(p.Name, ".")
}

// ContractArm is one ensures arm (a69): the outcome it
// specifies, the bound result/error name, expression
// predicates, and Boolean match blocks. Stored only;
// no phase proves, checks, or emits contracts yet.
type ContractArm struct {
	Outcome string
	Bind    string
	Preds   []*Small
	Matches []*Node
	Line    int
}

// VariantCase is one case row of a variant declaration
// (a73): the short name as written plus payload fields.
// The qualified constructor name is elaborated by
// qualifyCase, never written in the declaration.
type VariantCase struct {
	Short  string
	Fields [][2]string
	Line   int
}

// VariantDecl is a closed tagged union declaration
// (a73): a nominal parent with a fixed case set.
// Generic parents and their cases are stamped before checking;
// downstream phases consume only monomorphic declarations.
type VariantDecl struct {
	Name       string
	Rev        int
	TypeParams []string
	Cases      []VariantCase
	Line       int
}

// qualifyCase elaborates a declaration-row short name to
// its globally unambiguous qualified constructor: the
// variant's domain (name before its first "__") plus the
// short name. One documented rule, applied everywhere.
func qualifyCase(variant, short string) string {
	domain := variant
	if i := strings.Index(variant, "__"); i >= 0 {
		domain = variant[:i]
	}
	// A stamped parent carries its instance suffix on every case as
	// well: Option__Value$T$int owns Option__Some$T$int, never the
	// case of another instance. Source names cannot contain '$'.
	if i := strings.Index(variant, "$T$"); i >= 0 {
		return domain + "__" + short + variant[i:]
	}
	return domain + "__" + short
}

type Arm struct {
	// Pats holds the arm's patterns, one per match scrutinee: exactly
	// one entry for single-scrutinee arms, two or more for
	// multi-scrutinee value arms (docs/a28, slots bool/str/wild).
	// Always non-empty; the parser rejects empty slots.
	Pats []Pattern
	Rhs  *Node
	Line int
}

type MatchKind uint8

const (
	// MatchValue discriminates ordinary values: one or more scrutinees,
	// per-slot bool/str/wild patterns, no given table.
	MatchValue MatchKind = iota
	// MatchCall discriminates one call's outcome: Ok or emitted error
	// variants, scripted through given. Always a single scrutinee.
	MatchCall
	// MatchChain is sequential fallible composition (a86): ordered
	// call steps binding Ok payloads with a shared failure arm. The
	// checker elaborates it into nested MatchCall nodes before any
	// proof, run, or emit sees it, so downstream phases only ever
	// meet the two shapes above; an unelaborated chain reaching them
	// is a compiler bug and fails closed wherever Kind switches.
	MatchChain
	// MatchInvoke discriminates one function-value invocation: a
	// bare-name reference plus a single `with` argument, Ok or
	// emitted error variants (B00). Stage 1 parses the shape and
	// refuses it statically; proof, run, and emit fail closed on
	// the kind until invocation lands.
	MatchInvoke
)

// ChainStep is one chain link: a call, its Ok-payload binder, and an
// optional boolean guard over bound values. Guard holds parsed Small
// or nil; calls and forward are rejected inside guards at parse.
type ChainStep struct {
	Call   *Small
	Binder string
	Guard  *Small
	// Given scripts a foreign step call with the ordinary table
	// grammar; nil scripts nothing. The elaborated call node
	// carries it untouched.
	Given map[string]*Small
	Line  int
}

type Node struct {
	IsMatch bool
	// Kind names the match family, decided once at parse: call outcome
	// vs value table. Downstream phases switch on Kind, never on
	// arity or scrutinee shape.
	Kind MatchKind
	// Scruts holds the match scrutinees: exactly one for MatchCall,
	// one or more for MatchValue, the bare-name reference for
	// MatchInvoke (whose single argument rides InvokeArg).
	Scruts []*Small
	Arms   []Arm
	// InvokeArg holds the one `with` argument of a MatchInvoke;
	// nil for every other kind.
	InvokeArg *Small
	Given     map[string]*Small // nil value node = retired "-" row, rejected in checkGiven (a91)
	Small     *Small
	Line      int
	// ChainSteps holds a86 chain links; ChainTail continues on full
	// success and ChainElseText is the shared failure source text,
	// parsed fresh per level at elaboration so no Small is aliased.
	// Meaningful only when Kind is MatchChain, empty otherwise.
	ChainSteps    []ChainStep
	ChainTail     *Node
	ChainElse     string
	ChainElseLine int
	// analysis carries the value-table proof from verification to
	// emit (nil until verifyValueMatch proves the table). Unexported:
	// invisible to any serialization, meaningful only post-proof.
	analysis *valueMatchAnalysis
	// invokeSig carries the resolved callable signature for a
	// MatchInvoke (nil until resolveInvokeSites runs in
	// buildWorld, and nil forever when the target names no
	// Fn-typed parameter). Check, proof, forward elaboration,
	// run, and emit consume it; every consumer fails closed on
	// nil. Unexported like analysis.
	invokeSig *invokeSig
}

type Test struct {
	Name string
	// TypeBinds pins one complete instantiation per row on a
	// generic fn (G1): `row<T=str>(...)`. Param=arg pairs in
	// source order; empty for monomorphic rows. Expansion
	// routes each row to its stamped copy.
	TypeBinds [][2]string
	Args      []Arg
	Expected  *Small
	Line      int
	// Pinned marks a row as trusted acceptance (a87): weakening its
	// expectation against an accepted baseline is reported loudly.
	// Rows without the marker are proposed evidence and churn freely.
	Pinned bool
}

type Decl interface{ declKind() string }

type ErrorDecl struct {
	Name   string
	Fields [][2]string
	Line   int
}

func (d *ErrorDecl) declKind() string { return "error" }

type TypeDecl struct {
	Name string
	Rev  int
	// TypeParams names the declared type parameters (G2):
	// empty for monomorphic records. Expansion stamps one
	// monomorphic copy per distinct instantiation; the
	// template itself never reaches checking.
	TypeParams []string
	Fields     [][2]string
	Line       int
}

func (d *TypeDecl) declKind() string    { return "type" }
func (d *VariantDecl) declKind() string { return "variant" }

type FnDecl struct {
	Name string
	Rev  int
	// TypeParams names the declared type parameters (G1): empty
	// for monomorphic functions. Expansion stamps one
	// monomorphic copy per distinct instantiation; the template
	// itself never reaches checking.
	TypeParams []string
	Params     [][2]string
	Ret        string
	Emits      []string
	Tests      []Test
	Body       *Node
	UsesHere   []string
	// Effects lists the cell capabilities this function may use,
	// e.g. Count__total.read. Set from the effects metadata line.
	Effects []string
	// DecNames names the params proven to shrink on every self-call
	// (empty, none), and DecSchema names the admitted recursion shape:
	// "" is the unit loop (site passes p - 1), "euclid" is the
	// Euclidean step (site passes (b, a % b)), "narrowing" is binary
	// search (site passes (lo, mid) or (mid, hi) with mid (lo+hi)/2).
	// Set from the decreases metadata line; a19 owns the theorems.
	DecNames  []string
	DecSchema string
	// Requires holds requires-block predicates and Ensures the
	// outcome-indexed ensures arms (a69). Parsed and stored
	// only; no phase enforces them yet.
	Requires []*Small
	Ensures  []ContractArm
	Line     int
}

func (d *FnDecl) declKind() string { return "fn" }

// Utf8ExportDecl authorizes one function revision to disclose one
// brand's representation as UTF-8 Bytes (a46 S2). It defines no value
// or function: certifyExports validates it whole-program and annotates
// the exact permitted call site. Never in provides.
type Utf8ExportDecl struct {
	Brand    string
	Function string
	Revision int
	Line     int
}

func (d *Utf8ExportDecl) declKind() string { return "export" }

// AssetBridgeDecl authorizes one sink function revision to consume one
// schema-owned approval witness (S2 slice plan). Dual authorization: the
// named owner module releases the asset and policy brands, the granting
// module certifies the sink. It defines no value or function:
// certifyAssetBridge validates it whole-program and annotates the exact
// permitted kernel call site (reusing the ExportBrand certificate field).
// Never in provides.
type AssetBridgeDecl struct {
	Asset    string
	Policy   string
	Owner    string
	Function string
	Revision int
	Role     string
	Line     int
}

func (d *AssetBridgeDecl) declKind() string { return "bridge" }

// BrandDecl is a nominal string wrapper: brand Name is str rev N.
// An optional seals_from [B, ...] clause authorizes explicit one-way
// promotion seals from those same-module brands (a26); without it the
// brand mints from str only. Branding is proof, not runtime; the
// emitter forgets every brand.
type BrandDecl struct {
	Name      string
	Under     string
	Rev       int
	SealsFrom []string
	Line      int
}

// ConstDecl is a named scalar-literal constant (slice 1): const
// Name: TYPE rev N = literal. V1 admits int, str, dec, bool
// literals only; the checker rejects anything else (CAN6016).
// References resolve lazily at each consumer through the
// program const table, so termination stays syntactic.
type ConstDecl struct {
	Name  string
	Type  string
	Rev   int
	Value *Small
	Line  int
}

func (d *ConstDecl) declKind() string { return "const" }

func (d *BrandDecl) declKind() string { return "brand" }

// ExternDecl is a foreign function: declared, never defined. Calls to it
// are scripted through given tables like can calls; the host provides
// the implementation and the TS emit imports it.
type ExternDecl struct {
	Name   string
	Rev    int
	Params [][2]string
	Ret    string
	Emits  []string
	Line   int
}

func (d *ExternDecl) declKind() string { return "extern" }

// StateDecl is module-private named storage for one base-type value:
// state Name: T = lit. Cells never appear in provides, take no pins,
// and are visible only in their own file.
type StateDecl struct {
	Name string
	Type string
	Init *Small
	Line int
}

func (d *StateDecl) declKind() string { return "state" }

type Module struct {
	File string
	// ID is the canonical source identity: the cleaned input path as
	// passed. Two inputs with different IDs are different modules even
	// when their basenames (File) match; File stays the display name.
	ID    string
	Stem  string
	Mod   string
	Hdr   map[string][]string
	Decls []Decl
	// GenericBase maps stamped names to their generic base
	// (G1 expansion). Nil for modules without generics.
	GenericBase map[string]string
}

// ---------------------------------------------------------- scanning -------
// The R1 brace scan lives in internal/scan (one implementation shared
// with modcheck's repo gate); stripComment below tracks strings the
// same way.
func stripComment(line string) string {
	var out strings.Builder
	inStr := false
	for i := 0; i < len(line); {
		ch := line[i]
		if inStr {
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(line) {
				out.WriteByte(line[i+1])
				i++
			} else if ch == '"' {
				inStr = false
			}
		} else {
			if ch == '"' {
				inStr = true
				out.WriteByte(ch)
			} else if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
				break
			} else {
				out.WriteByte(ch)
			}
		}
		i++
	}
	return out.String()
}

func splitTop(s string, sep rune) []string {
	return splitTopInner(s, sep, false, false, false)
}

// isCallGenericHead reports whether s[i] opens a generic call's
// argument list: `<` immediately following a name in `call NAME<`
// position. Comparisons (`a<b`) and Seq heads (`Seq<str>`) never
// match: the name must sit directly behind the bracket with a
// whole-word `call` before it.
func isCallGenericHead(s string, i int) bool {
	j := i - 1
	for j >= 0 && (s[j] == '_' || s[j] >= '0' && s[j] <= '9' || s[j] >= 'a' && s[j] <= 'z' || s[j] >= 'A' && s[j] <= 'Z') {
		j--
	}
	if j == i-1 {
		return false
	}
	k := j
	for k >= 0 && (s[k] == ' ' || s[k] == '\t') {
		k--
	}
	if k < 3 || s[k-3:k+1] != "call" {
		return false
	}
	return k-4 < 0 || !(s[k-4] == '_' || s[k-4] >= '0' && s[k-4] <= '9' || s[k-4] >= 'a' && s[k-4] <= 'z' || s[k-4] >= 'A' && s[k-4] <= 'Z')
}

// splitTopInner is splitTop with an empty-slot policy: drop trims and
// drops empties (argument lists, where trailing commas are tolerated),
// keep preserves every slot so callers can reject them. callAware
// additionally protects generic call argument lists (`call f<A,B>`)
// from the separator; comparisons and Seq heads are untouched.
// Only scrutinee lists opt in — G1 generic calls are valid in
// scrutinee position alone, and argument-list splitting keeps its
// exact existing meaning everywhere else. patternAware protects case
// type arguments on arm lists only, never on expression comparisons.
func splitTopInner(s string, sep rune, keepEmpty bool, callAware bool, patternAware bool) []string {
	var parts []string
	var cur strings.Builder
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			cur.WriteByte(ch)
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
			cur.WriteByte(ch)
		} else if ch == '<' && (genericHeadLT(s, i) || (len(stack) > 0 && stack[len(stack)-1] == '>') || (callAware && isCallGenericHead(s, i)) || (patternAware && genericCaseHeadLT(s, i))) {
			stack = append(stack, '>')
			cur.WriteByte(ch)
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
			cur.WriteByte(ch)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
			cur.WriteByte(ch)
		} else if len(stack) == 0 && rune(ch) == sep {
			parts = append(parts, cur.String())
			cur.Reset()
		} else {
			cur.WriteByte(ch)
		}
	}
	parts = append(parts, cur.String())
	var keep []string
	for _, p := range parts {
		if keepEmpty || strings.TrimSpace(p) != "" {
			keep = append(keep, strings.TrimSpace(p))
		}
	}
	return keep
}

// splitMatchList splits a top-level comma list from a match line or arm,
// rejecting empty slots: `match x,` and `true,, false` are malformed
// syntax, not short rows. String- and paren-aware like splitTop.
func splitMatchList(s string) ([]string, error) {
	return splitMatchParts(s, false)
}

func splitMatchParts(s string, patterns bool) ([]string, error) {
	raw := splitTopInner(s, ',', true, true, patterns)
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p == "" {
			return nil, fmt.Errorf("empty slot in match list: %q", s)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("match with no scrutinee")
	}
	return out, nil
}

// findTop returns the index and matched op of the first top-level occurrence.
// topBracket finds the first [ outside strings and any paren or
// bracket depth: the start of a postfix index/slice group. Index 0
// is never reported, so list literals keep their own branch.
func topBracket(s string) (int, bool) {
	pdepth, bdepth := 0, 0
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if ch == '(' {
			pdepth++
		} else if ch == ')' && pdepth > 0 {
			pdepth--
		} else if ch == '[' {
			if pdepth == 0 && bdepth == 0 {
				if i > 0 {
					return i, true
				}
				return -1, false
			}
			bdepth++
		} else if ch == ']' && bdepth > 0 {
			bdepth--
		}
	}
	return -1, false
}

// topColons lists every : outside strings and any paren or bracket
// depth. Empty sides are significant (s[1:] keeps them); splitTop
// drops empties, which would silently turn s[1:] into s[1].
func topColons(s string) []int {
	var out []int
	pdepth, bdepth := 0, 0
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if ch == '(' {
			pdepth++
		} else if ch == ')' && pdepth > 0 {
			pdepth--
		} else if ch == '[' {
			bdepth++
		} else if ch == ']' && bdepth > 0 {
			bdepth--
		} else if ch == ':' && pdepth == 0 && bdepth == 0 {
			out = append(out, i)
		}
	}
	return out
}

// leadingBinOpLen reports the length of a binary operator opening s:
// two-char comparisons first, then the arithmetic ops. Zero means s
// does not start with one (= alone is binding syntax, not an op).
func leadingBinOpLen(s string) int {
	for _, op := range []string{"==", ">=", "<=", "!=", ">", "<", "+", "-", "*", "/", "%"} {
		if strings.HasPrefix(s, op) {
			return len(op)
		}
	}
	return 0
}

func findTop(s string, ops []string) (int, string) {
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		} else if len(stack) == 0 {
			// a36 S1: a sequence head is one atom. Its <, >,
			// and member commas never split an outer operator.
			if strings.HasPrefix(s[i:], "Seq<") {
				if end := seqHeadEnd(s[i:]); end > 0 {
					i += end
					continue
				}
			}
			// G2: a generic head is one atom too.
			if n := genericHeadLen(s[i:]); n > 0 {
				i += n - 1
				continue
			}
			for _, op := range ops {
				if strings.HasPrefix(s[i:], op) {
					return i, op
				}
			}
		}
	}
	return -1, ""
}

// isWordChar reports identifier constituents for operator-word
// boundaries: and/or/not never split inside a longer name.
func isWordChar(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

// findTopWord is findTop for whole-word operators (and/or): the
// first depth-zero occurrence outside strings whose neighbors
// are not word characters, so `or` splits `a or b` but never
// `orig`, `error`, or `"a or b"`.
func findTopWord(s string, words []string) (int, string) {
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		} else if len(stack) == 0 {
			if strings.HasPrefix(s[i:], "Seq<") {
				if end := seqHeadEnd(s[i:]); end > 0 {
					i += end
					continue
				}
			}
			// G2: a generic head is one atom too.
			if n := genericHeadLen(s[i:]); n > 0 {
				i += n - 1
				continue
			}
			for _, w := range words {
				if !strings.HasPrefix(s[i:], w) {
					continue
				}
				if i > 0 && isWordChar(s[i-1]) {
					continue
				}
				if j := i + len(w); j < len(s) && isWordChar(s[j]) {
					continue
				}
				return i, w
			}
		}
	}
	return -1, ""
}

// cutWordPrefix strips a leading operator word (not) with the
// same boundary rule: `not x` and `not(x)` claim, `nothing`
// never does.
func cutWordPrefix(s, w string) (string, bool) {
	if !strings.HasPrefix(s, w) {
		return "", false
	}
	if rest := s[len(w):]; rest == "" {
		return "", true
	} else if !isWordChar(rest[0]) {
		return strings.TrimSpace(rest), true
	}
	return "", false
}

// findLastTop is findTop keeping the last top-level occurrence instead
// of the first: splitting there makes chains associate left.
func findLastTop(s string, ops []string) (int, string) {
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	best, bestOp := -1, ""
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		} else if len(stack) == 0 {
			// a36 S1: a sequence head is one atom (see findTop).
			if strings.HasPrefix(s[i:], "Seq<") {
				if end := seqHeadEnd(s[i:]); end > 0 {
					i += end
					continue
				}
			}
			// G2: a generic head is one atom too.
			if n := genericHeadLen(s[i:]); n > 0 {
				i += n - 1
				continue
			}
			for _, op := range ops {
				if strings.HasPrefix(s[i:], op) {
					best, bestOp = i, op
				}
			}
		}
	}
	return best, bestOp
}

func balanced(s string, openI int) (int, error) {
	closers := map[byte]byte{'(': ')', '[': ']'}
	want := closers[s[openI]]
	depth := 0
	inStr, esc := false, false
	for i := openI; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if ch == s[openI] {
			depth++
		} else if ch == want {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("unbalanced %c in: %s", s[openI], s)
}

// ------------------------------------------------------- small exprs -------
var (
	reInt  = regexp.MustCompile(`^-?\d+$`)
	reWord = regexp.MustCompile(`^[\w.]+$`)
	// reProjSeg matches one .field segment in the postfix loop
	// (b03): the leading dot plus a strict identifier.
	reProjSeg = regexp.MustCompile(`^\.([A-Za-z_]\w*)`)
	reName    = regexp.MustCompile(`^\w+$`)
	reDec     = regexp.MustCompile(`^d"([^"]*)"$`)
	reFloat   = regexp.MustCompile(`^-?(\d+\.\d*|\.\d+|\d+[eE][+-]?\d+)$`)
	reDecNM   = regexp.MustCompile(`^(-?)(\d+)\.(\d+)$`)
	reSeal    = regexp.MustCompile(`^seal\s+(\w+)\((.*)\)$`)
	// reSeqElem admits one plain element type name inside Seq<...>.
	// No angle brackets: nested sequences are not a v1 shape, so a
	// second < fails here with a precise diagnostic instead of a
	// binop cascade downstream.
	reSeqElem = regexp.MustCompile(`^[A-Za-z_][\w.]*$`)
	// reGenericElem admits one instantiated element type inside
	// Seq<...> (G2): Base<args> with a non-Seq base. Nested user
	// args and Seq<Seq<..>> still fail at reSeqElem's message.
	reGenericElem = regexp.MustCompile(`^([A-Za-z_][\w.]*)<(.+)>$`)
)

// canonDec normalizes dec digits to canonical form: no leading integer
// zeros, no trailing fractional zeros, -0 folded to 0. The shape stays
// dotted (d"1.0", never d"1") so every dec reads as fractional.
func canonDec(raw string) (string, error) {
	m := reDecNM.FindStringSubmatch(raw)
	if m == nil {
		return "", fmt.Errorf("bad dec digits %q: want d\"12.34\"", raw)
	}
	ip := strings.TrimLeft(m[2], "0")
	if ip == "" {
		ip = "0"
	}
	fp := strings.TrimRight(m[3], "0")
	if fp == "" {
		fp = "0"
	}
	if ip == "0" && fp == "0" {
		return "0.0", nil
	}
	return m[1] + ip + "." + fp, nil
}

// decodeEscapes interprets the six e"..." escapes once, left to
// right: \" \\ \n \r \t \0. The result is data, never input to
// another pass (e"\\n" is backslash+n; e"\01" is NUL followed
// by '1', not an octal escape). Unknown escapes, a dangling
// backslash, and non-escape content all fail: ordinary "..."
// already provides the preservation mechanism, so e"..."
// rejects instead of passing through.
func decodeEscapes(raw string) (string, error) {
	if strings.IndexByte(raw, '\\') < 0 {
		return raw, nil
	}
	var out strings.Builder
	out.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			out.WriteByte(c)
			continue
		}
		i++
		if i >= len(raw) {
			return "", fmt.Errorf("dangling backslash in interpreted string")
		}
		switch raw[i] {
		case '"':
			out.WriteByte('"')
		case '\\':
			out.WriteByte('\\')
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case '0':
			out.WriteByte(0)
		default:
			return "", fmt.Errorf("unsupported escape \\%c in interpreted string: want one of \\\" \\\\ \\n \\r \\t \\0", raw[i])
		}
	}
	return out.String(), nil
}

// escClose finds the closing quote of a literal opening at s[0],
// skipping backslash-escaped bytes. -1 means unterminated.
func escClose(s string) int {
	esc := false
	for k := 1; k < len(s); k++ {
		c := s[k]
		if esc {
			esc = false
		} else if c == '\\' {
			esc = true
		} else if c == '"' {
			return k
		}
	}
	return -1
}

// parseCallHead parses a complete call expression, plain or
// generic (`call f<str>(...)`). Callers check this before any
// operator cascade: a generic argument list's brackets must
// never reach the comparison splitter as less-than. matched
// distinguishes shape ("not a call", fall through to the
// cascade) from contents (a malformed call reports here, so
// argument errors keep their existing messages).
func parseCallHead(s string) (node *Small, err error, matched bool) {
	if !strings.HasPrefix(s, "call ") {
		return nil, nil, false
	}
	m := regexp.MustCompile(`^call\s+(\w+)(?:<(.+?)>)?\((.*)\)$`).FindStringSubmatch(s)
	if m == nil {
		return nil, nil, false
	}
	tyargs, err := splitTypeArgs(m[2])
	if err != nil {
		return nil, err, true
	}
	args, err := parseArgs(m[3])
	if err != nil {
		return nil, err, true
	}
	return &Small{Kind: "call", Fname: m[1], TypeArgs: tyargs, Args: args}, nil, true
}

// parseFnrefHead parses one function reference `fnref
// target<T>(name = value, ...)` (B00 stage 1): the parens hold
// captures, not invocation arguments, so every capture is named.
// A distinct Kind keeps reference semantics out of the call,
// lint, and graph passes. Malformed heads report here, never
// fall through to comparison parsing.
func parseFnrefHead(s string) (node *Small, err error, matched bool) {
	if !strings.HasPrefix(s, "fnref ") {
		return nil, nil, false
	}
	m := regexp.MustCompile(`^fnref\s+(\w+)(?:<(.+?)>)?\((.*)\)$`).FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("bad fnref: want fnref target<T>(name = value, ...) , got %s", s), true
	}
	tyargs, err := splitTypeArgs(m[2])
	if err != nil {
		return nil, err, true
	}
	args, err := parseArgs(m[3])
	if err != nil {
		return nil, err, true
	}
	for _, a := range args {
		if !a.HasName {
			return nil, fmt.Errorf("bad fnref capture: every capture is named, got positional"), true
		}
	}
	return &Small{Kind: "fnref", Fname: m[1], TypeArgs: tyargs, Args: args}, nil, true
}

func parseSmall(s string) (*Small, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}
	if c, err, matched := parseCallHead(s); matched {
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	if r, err, matched := parseFnrefHead(s); matched {
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	// a66: interpreted string literals e"...". Same str kind and
	// runtime representation as ordinary literals; only the six
	// escapes decode. The e prefix is inert to every splitter
	// (they track "..." regions uniformly), so embedding and
	// postfix reuse the ordinary rule below: a bare literal, or
	// fall-through to generic binop/bracket parsing.
	if strings.HasPrefix(s, `e"`) {
		j := escClose(s[1:])
		if j < 0 {
			return nil, fmt.Errorf("unterminated interpreted string: %s", s)
		}
		j++ // account for the e prefix
		decoded, err := decodeEscapes(s[2:j])
		if err != nil {
			return nil, err
		}
		if j == len(s)-1 {
			return &Small{Kind: "str", Str: decoded}, nil
		}
		rest := strings.TrimSpace(s[j+1:])
		embedded := false
		if oplen := leadingBinOpLen(rest); oplen > 0 {
			after := strings.TrimSpace(rest[oplen:])
			if after != "" && !strings.HasPrefix(after, `"`) && !strings.HasPrefix(after, `e"`) {
				embedded = true
			}
		} else if strings.HasPrefix(rest, "[") {
			embedded = true
		}
		if !embedded {
			if len(s) < 3 || !strings.HasSuffix(s, `"`) {
				return nil, fmt.Errorf("bad string: %s", s)
			}
			inner, err := decodeEscapes(s[2 : len(s)-1])
			if err != nil {
				return nil, err
			}
			return &Small{Kind: "str", Str: inner}, nil
		}
	}
	if strings.HasPrefix(s, `"`) {
		// A trailing index/slice group belongs to the postfix branch
		// below ("ABC"[0:1]), not to the literal: fall through when
		// the closing quote is followed by [. A literal-left binary
		// expression ("-" + tail) falls through too, unless the right
		// operand is itself quoted ("a" + "b" stays one swallowed
		// literal, as ever). Anything else keeps the old behavior.
		j := -1
		esc := false
		for k := 1; k < len(s); k++ {
			c := s[k]
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				j = k
				break
			}
		}
		if j == len(s)-1 {
			return &Small{Kind: "str", Str: s[1:j]}, nil
		}
		oldPath := true
		if j > 0 {
			rest := strings.TrimSpace(s[j+1:])
			if oplen := leadingBinOpLen(rest); oplen > 0 {
				after := strings.TrimSpace(rest[oplen:])
				if after != "" && !strings.HasPrefix(after, `"`) {
					oldPath = false
				}
			} else if strings.HasPrefix(rest, "[") {
				oldPath = false
			}
		}
		if oldPath {
			if len(s) < 2 || !strings.HasSuffix(s, `"`) {
				return nil, fmt.Errorf("bad string: %s", s)
			}
			return &Small{Kind: "str", Str: s[1 : len(s)-1]}, nil
		}
	}
	if reInt.MatchString(s) {
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return nil, fmt.Errorf("bad int literal %s", s)
		}
		return &Small{Kind: "int", Num: n}, nil
	}
	// Dec precedes float: d"1.5" is the one legal dotted spelling.
	if m := reDec.FindStringSubmatch(s); m != nil {
		canon, err := canonDec(m[1])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "dec", Dec: canon}, nil
	}
	// Bare dotted numbers parse (as float nodes) so checkStatic can
	// point at them with CAN6001; they never evaluate.
	if reFloat.MatchString(s) {
		return &Small{Kind: "float", Str: s}, nil
	}
	if s == "true" || s == "false" {
		return &Small{Kind: "bool", B: s == "true"}, nil
	}
	if s == "_" {
		return &Small{Kind: "wild"}, nil
	}
	// Seal precedes binops so a literal containing == stays intact.
	// The inner expression is kept raw: checkTypes enforces the
	// string-literal rule with CAN6003, and eval enforces str.
	if m := reSeal.FindStringSubmatch(s); m != nil {
		if idx := strings.Index(s, "("); idx >= 0 {
			if end, err := balanced(s, idx); err == nil && end == len(s)-1 {
				inner, err := parseSmall(m[2])
				if err != nil {
					return nil, err
				}
				return &Small{Kind: "seal", Seal: m[1], Args: []Arg{{V: inner}}}, nil
			}
		}
	}
	// Exchange precedes binops for the same reason: args bind with =
	// and outcomes may contain comparisons. One spelling per meaning:
	// every script row is `exchange args (...) outcome ...` (a12).
	if s == "exchange" || strings.HasPrefix(s, "exchange ") || strings.HasPrefix(s, "exchange\t") {
		return parseExchange(s)
	}
	// a36 S1: typed sequence literals Seq<T>[...]. The element type
	// is always written; bare [...] keeps its script-row meaning.
	// This rule precedes binop splitting so the < and > never read
	// as comparisons. Anything starting with Seq< is claimed here:
	// malformed shapes fail with a precise error, never a binop
	// cascade (`Seq < x` with a space is unaffected and still parses
	// as a comparison).
	if strings.HasPrefix(s, "Seq<") {
		end := seqHeadEnd(s)
		if end < 0 {
			return nil, fmt.Errorf("bad sequence literal %s: want Seq<T>[...]", s)
		}
		if end == len(s)-1 {
			return parseSeqLit(s)
		}
		// A valid head followed by more text: the literal is one
		// operand inside a larger expression. Reparse with the
		// head parenthesized so the normal cascade below splits
		// operators with its usual precedence; the inner parse
		// still owns malformed heads precisely.
		head := s[:end+1]
		if _, err := parseSeqLit(head); err != nil {
			return nil, err
		}
		return parseSmall("(" + head + ")" + s[end+1:])
	}
	// G2: generic constructions Box<str>(...). Claimed like
	// sequence heads so the <> never reads as comparison; a
	// valid head with trailing text parenthesizes and reparses
	// the same way.
	if n := genericHeadLen(s); n > 0 {
		rest := s[n:]
		end, err := balanced(rest, 0)
		if err != nil {
			return nil, fmt.Errorf("bad generic construction %s: unbalanced (...)", s)
		}
		if end == len(rest)-1 {
			return parseGenericCtor(s)
		}
		head := s[:n+end+1]
		if _, err := parseGenericCtor(head); err != nil {
			return nil, err
		}
		return parseSmall("(" + head + ")" + s[n+end+1:])
	}
	// A bare instantiated mention in value position is never a
	// comparison (`a<b>` without a trailing `>` keeps its old
	// reading): fail precisely instead of cascading.
	if m := reGenericElem.FindStringSubmatch(s); m != nil && !strings.ContainsAny(m[2], " \t") {
		return nil, fmt.Errorf("type %s is not a value: construct it with %s(...)", s, s)
	}
	// Slice 5: eager boolean operators, loosest precedence: or,
	// then and, each splitting at the first top-level whole
	// word so chains associate left through recursion.
	if i, op := findTopWord(s, []string{"or"}); i >= 0 {
		l, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		r, err := parseSmall(s[i+len(op):])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "binop", Op: op, L: l, R: r}, nil
	}
	if i, op := findTopWord(s, []string{"and"}); i >= 0 {
		l, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		r, err := parseSmall(s[i+len(op):])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "binop", Op: op, L: l, R: r}, nil
	}
	return parseSmallCmp(s)
}

// parseSmallCmp parses comparison-level expressions and
// tighter: prefix not, comparisons, arithmetic, postfix, and
// primaries. Splitting not out of parseSmall keeps `not a and
// b` reading `(not a) and b`: the operand never spans an
// and/or, while parenthesized operands still take the full
// cascade through parseSmall.
func parseSmallCmp(s string) (*Small, error) {
	// G1: `not` strips to a bare call (`not call f<str>(x)`),
	// so the call head check repeats here past the prefix.
	if c, err, matched := parseCallHead(s); matched {
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	if r, err, matched := parseFnrefHead(s); matched {
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	// Slice 5: prefix not binds tighter than comparisons
	// (`not a == b` is `not (a == b)`), nesting freely.
	if rest, ok := cutWordPrefix(s, "not"); ok {
		if strings.TrimSpace(rest) == "" {
			return nil, fmt.Errorf("empty expression")
		}
		v, err := parseSmallCmp(rest)
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "not", L: v}, nil
	}
	if i, op := findTop(s, []string{"==", ">=", "<=", ">", "<", "!="}); i >= 0 {
		l, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		r, err := parseSmall(s[i+len(op):])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "binop", Op: op, L: l, R: r}, nil
	}
	// Arithmetic binds tighter than comparisons. + and - split before
	// *, /, % (lower precedence splits first); each level splits at
	// the LAST top-level occurrence so chains associate left:
	// 10 - 3 - 2 is (10-3)-2. Prefix minus binds tighter than *
	// and binary +/- (slice 6), so the addition level delegates.
	// / and % share * precedence (a17: exact Euclidean integer
	// division; dec operands refused in checkSem).
	return parseSmallAdd(s)
}

// findLastBinAddSub splits + and - like findLastTop, except a +/-
// in unary position never splits: the previous non-space
// character must end an operand (word character, closing quote,
// paren, or bracket). A leading or operator-preceded minus is
// unary, so a * -b keeps its minus and a - -b splits at the
// binary one; chains still associate left through recursion.
func findLastBinAddSub(s string) (int, string) {
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	best, bestOp := -1, ""
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		} else if len(stack) == 0 && (ch == '+' || ch == '-') {
			j := i - 1
			for j >= 0 && (s[j] == ' ' || s[j] == '\t') {
				j--
			}
			if j < 0 {
				continue
			}
			pc := s[j]
			if pc == '"' || pc == ')' || pc == ']' || isWordChar(pc) {
				best, bestOp = i, string(ch)
			}
		}
	}
	return best, bestOp
}

// parseSmallAdd parses addition-level expressions: binary +/-
// splits, else multiplication level. A minus in unary position
// never splits here; it descends for the prefix rule below.
func parseSmallAdd(s string) (*Small, error) {
	if i, op := findLastBinAddSub(s); i >= 0 {
		l, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		r, err := parseSmall(s[i+len(op):])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "binop", Op: op, L: l, R: r}, nil
	}
	return parseSmallMul(s)
}

// parseNegAtom resolves one atomic operand spelling for prefix
// minus: integer and decimal literals fold to their negated
// literal nodes, floats parse so the float ban owns them, and
// bools pass through for the operand refusal. ok=false means
// rest is composite: refs, calls, brackets, and parens descend
// at mul precedence instead. Seal, exchange, and Seq<>
// spellings stay composite-path parse errors (nonsense either
// way); -not x is out of scope (not binds looser than prefix
// minus, so it needs parens: -(not x)).
func parseNegAtom(rest string) (*Small, bool) {
	if reInt.MatchString(rest) {
		n, ok := new(big.Int).SetString(rest, 10)
		if !ok {
			return nil, false
		}
		return &Small{Kind: "int", Num: new(big.Int).Neg(n)}, true
	}
	if m := reDec.FindStringSubmatch(rest); m != nil {
		canon, err := canonDec(m[1])
		if err != nil {
			return nil, false
		}
		if strings.HasPrefix(canon, "-") {
			return &Small{Kind: "dec", Dec: canon[1:]}, true
		}
		neg, err := canonDec("-" + canon)
		if err != nil {
			return nil, false
		}
		return &Small{Kind: "dec", Dec: neg}, true
	}
	if reFloat.MatchString(rest) {
		return &Small{Kind: "float", Str: rest}, true
	}
	if rest == "true" || rest == "false" {
		return &Small{Kind: "bool", B: rest == "true"}, true
	}
	if strings.HasPrefix(rest, `e"`) {
		j := escClose(rest[1:])
		if j < 0 {
			return nil, false
		}
		j++
		if j != len(rest)-1 {
			return nil, false
		}
		decoded, err := decodeEscapes(rest[2:j])
		if err != nil {
			return nil, false
		}
		return &Small{Kind: "str", Str: decoded}, true
	}
	return nil, false
}

// parseSmallMul parses multiplication-level expressions and
// tighter: * / % splits, then prefix minus, then the postfix
// and primary chain below. Trying * first keeps the minus
// tight: -a * b splits at *, leaving -a for the rule, so the
// tree reads (-a) * b.
func parseSmallMul(s string) (*Small, error) {
	if i, op := findLastTop(s, []string{"*", "/", "%"}); i > 0 {
		l, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		r, err := parseSmall(s[i+len(op):])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "binop", Op: op, L: l, R: r}, nil
	}
	if len(s) > 1 && s[0] == '-' {
		rest := strings.TrimSpace(s[1:])
		if rest == "" {
			return nil, fmt.Errorf("empty expression")
		}
		// Slice 6: atomic spellings first. Literals live in
		// parseSmall's head, below this level, so - 3, -d"0.5",
		// -3.5, and -true resolve here: ints and decs fold
		// back to literal nodes (emitted bytes never change),
		// while floats and bools pass through so CAN6001 and
		// CAN6003 own them downstream. Anything else (refs,
		// calls, brackets, parens) descends; reaching here
		// past the * split keeps the minus tight.
		if lit, ok := parseNegAtom(rest); ok {
			return lit, nil
		}
		v, err := parseSmallMul(rest)
		if err != nil {
			return nil, err
		}
		// Parenthesized literals normalize too: -(3) is -3.
		if v.Kind == "int" && v.Num != nil {
			return &Small{Kind: "int", Num: new(big.Int).Neg(v.Num)}, nil
		}
		if v.Kind == "dec" {
			if strings.HasPrefix(v.Dec, "-") {
				return &Small{Kind: "dec", Dec: v.Dec[1:]}, nil
			}
			canon, err := canonDec("-" + v.Dec)
			if err != nil {
				return nil, err
			}
			return &Small{Kind: "dec", Dec: canon}, nil
		}
		return &Small{Kind: "neg", L: v}, nil
	}
	// a20: scalar text operators. # binds tightest (this branch runs
	// only when no looser split matched, so the operand is atomic);
	// s[i] and s[a:b] are postfix at the same level, chaining left.
	if strings.HasPrefix(s, "#") {
		v, err := parseSmall(s[1:])
		if err != nil {
			return nil, err
		}
		return &Small{Kind: "strlen", L: v}, nil
	}
	if i, ok := topBracket(s); ok {
		base, err := parseSmall(s[:i])
		if err != nil {
			return nil, err
		}
		rest := s[i:]
		for len(rest) > 0 {
			// b03: a .field group projects off the bracket
			// result, alternating with further brackets.
			if rest[0] == '.' {
				m := reProjSeg.FindStringSubmatch(rest)
				if m == nil {
					return nil, fmt.Errorf("bad projection %q: want .field after ]", rest)
				}
				base = &Small{Kind: "proj", L: base, Field: m[1]}
				rest = strings.TrimSpace(rest[len(m[0]):])
				continue
			}
			if rest[0] != '[' {
				return nil, fmt.Errorf("unexpected %q after ]", rest)
			}
			end, err := balanced(rest, 0)
			if err != nil {
				return nil, err
			}
			inner := rest[1:end]
			colons := topColons(inner)
			if len(colons) > 1 {
				return nil, fmt.Errorf("bad slice: %s", rest[:end+1])
			}
			if len(colons) == 0 {
				ix, err := parseSmall(inner)
				if err != nil {
					return nil, err
				}
				base = &Small{Kind: "stridx", L: base, R: ix}
				rest = strings.TrimSpace(rest[end+1:])
				continue
			}
			lo, err := parseSmall(inner[:colons[0]])
			if err != nil {
				return nil, err
			}
			hi, err := parseSmall(inner[colons[0]+1:])
			if err != nil {
				return nil, err
			}
			base = &Small{Kind: "strslice", L: base, R: lo, Hi: hi}
			rest = strings.TrimSpace(rest[end+1:])
		}
		return base, nil
	}
	// Calls parse at the cascade heads (parseCallHead): reaching
	// here with a `call` prefix means the head check declined a
	// malformed call, so report it rather than mis-splitting.
	if strings.HasPrefix(s, "call ") {
		return nil, fmt.Errorf("bad call syntax: %s", s)
	}
	if strings.HasPrefix(s, "(") {
		if end, err := balanced(s, 0); err == nil && end == len(s)-1 {
			return parseSmall(s[1 : len(s)-1])
		}
	}
	if strings.HasPrefix(s, "[") {
		if end, err := balanced(s, 0); err == nil && end == len(s)-1 {
			var items []*Small
			for _, p := range splitTop(s[1:len(s)-1], ',') {
				it, err := parseSmall(p)
				if err != nil {
					return nil, err
				}
				items = append(items, it)
			}
			return &Small{Kind: "list", Items: items}, nil
		}
	}
	if m := regexp.MustCompile(`^([\w.]+)\((.*)\)$`).FindStringSubmatch(s); m != nil {
		if idx := strings.Index(s, "("); idx >= 0 {
			if end, err := balanced(s, idx); err == nil && end == len(s)-1 {
				args, err := parseArgs(m[2])
				if err != nil {
					return nil, err
				}
				return &Small{Kind: "ctor", Ctor: m[1], Args: args}, nil
			}
		}
	}
	if reWord.MatchString(s) {
		return &Small{Kind: "ref", Ref: strings.Split(s, ".")}, nil
	}
	return nil, fmt.Errorf("cannot parse expression: %s", s)
}

// angleEnd returns the index of the `>` balancing the `<` at
// s[start], or -1 when unbalanced. Quotes are not special:
// heads never contain them.
func angleEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// genericHeadLen reports the length of the generic head opening
// at s[0] (`Word<balanced>` immediately followed by `(`), or 0
// when s holds no head there. The paren itself stays for normal
// depth handling; only the `<>` span is skipped, so its commas
// never split an outer list and its brackets never read as
// comparisons. `w<x>(y)` is therefore always a generic head now,
// never a chained comparison (zero corpus occurrences); `w<x>[y]`
// and spaced forms keep their old comparison readings.
func genericHeadLen(s string) int {
	i := 0
	for i < len(s) && (isWordChar(s[i]) || s[i] == '.') {
		i++
	}
	if i == 0 || i >= len(s) || s[i] != '<' {
		return 0
	}
	if c := s[0]; !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return 0
	}
	end := angleEnd(s, i)
	if end < 0 || end+1 >= len(s) || s[end+1] != '(' {
		return 0
	}
	return end + 1
}

// genericHeadLT reports whether the `<` at s[i] opens a generic
// head: word characters immediately before, a balanced `<>`
// span, and `(` immediately after — or `[` after a head that
// itself nests angles, which is a Seq literal over an
// instantiated element type (Seq<M__Pair<K,V>>[...]). The
// backward twin of genericHeadLen for scanners positioned at
// the bracket. The nesting requirement keeps plain Seq<str>[
// heads and <= comparisons exactly as unprotected as before:
// only a comma-bearing head can need the protection.
func genericHeadLT(s string, i int) bool {
	j := i - 1
	for j >= 0 && (isWordChar(s[j]) || s[j] == '.') {
		j--
	}
	j++
	if j >= i {
		return false
	}
	if c := s[j]; !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return false
	}
	end := angleEnd(s, i)
	if end < 0 || end+1 >= len(s) {
		return false
	}
	if s[end+1] == '(' {
		return true
	}
	if s[end+1] != '[' {
		return false
	}
	for k := i + 1; k < end; k++ {
		if s[k] == '<' {
			return true
		}
	}
	return false
}

// genericCaseHeadLT protects commas inside explicit case patterns only.
// Expression scanners do not opt in, so comparison syntax is unchanged.
func genericCaseHeadLT(s string, i int) bool {
	j := i - 1
	for j >= 0 && isWordChar(s[j]) {
		j--
	}
	base := s[j+1 : i]
	end := angleEnd(s, i)
	return reName.MatchString(base) && (base == "Ok" || strings.Contains(base, "__")) && end >= 0 && end+1 < len(s) && (s[end+1] == ' ' || s[end+1] == '\t')
}

// parseGenericCtor parses one generic construction `Base<args>(...)`
// (G2): the base names the record template, the args instantiate
// it. Semantic validation (known base, arity, closed args) belongs
// to expansion, not parsing — like splitTypeArgs for calls.
func parseGenericCtor(s string) (*Small, error) {
	lt := strings.IndexByte(s, '<')
	if lt <= 0 {
		return nil, fmt.Errorf("bad generic construction %s: want Name<...>(...)", s)
	}
	base := s[:lt]
	gt := angleEnd(s, lt)
	if gt < 0 {
		return nil, fmt.Errorf("bad generic construction %s: want %s<...>(...)", s, base)
	}
	tyargs, err := splitTypeArgs(s[lt+1 : gt])
	if err != nil {
		return nil, err
	}
	if base == "Ok" && len(tyargs) != 1 {
		return nil, fmt.Errorf("typed Ok takes exactly one success type")
	}
	rest := s[gt+1:]
	end, err := balanced(rest, 0)
	if err != nil || end != len(rest)-1 {
		return nil, fmt.Errorf("bad generic construction %s: want %s<...>(...)", s, base)
	}
	args, err := parseArgs(rest[1:end])
	if err != nil {
		return nil, err
	}
	return &Small{Kind: "ctor", Ctor: base, TypeArgs: tyargs, Args: args}, nil
}

// seqHeadEnd ends the Seq<T>[...] head starting at s[0]: the index
// of the closing bracket, or -1 when s holds no valid head there.
// Whitespace between > and [ is tolerated, like everywhere else.
func seqHeadEnd(s string) int {
	close := angleEnd(s, len("Seq<")-1)
	if close < 0 {
		close = strings.Index(s, ">")
	}
	if close < 0 {
		return -1
	}
	rest := s[close+1:]
	lead := len(rest) - len(strings.TrimLeft(rest, " \t"))
	rest = strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(rest, "[") {
		return -1
	}
	end, err := balanced(rest, 0)
	if err != nil {
		return -1
	}
	return close + 1 + lead + end
}

// parseSeqLit parses one typed sequence literal Seq<T>[v, ...]
// (a36 S1). The element type is one plain name: no nesting, no
// elision. Members are full expressions split top-level-comma aware,
// so members containing commas parse; an empty bracket is the empty
// sequence. Anything else starting with Seq< fails here precisely.
func parseSeqLit(s string) (*Small, error) {
	close := angleEnd(s, len("Seq<")-1)
	if close < 0 {
		close = strings.Index(s, ">")
	}
	if close < 0 {
		return nil, fmt.Errorf("bad sequence literal %s: want Seq<T>[...]", s)
	}
	elem := s[len("Seq<"):close]
	if !reSeqElem.MatchString(elem) {
		// G2: one instantiated element type. Nested user args
		// parse so expansion owns the nesting rejection;
		// Seq<Seq<..>> keeps its precise parse error.
		if m := reGenericElem.FindStringSubmatch(elem); m == nil || m[1] == "Seq" {
			return nil, fmt.Errorf("bad sequence element type %q: want one plain type name, no nesting", elem)
		}
	}
	rest := strings.TrimSpace(s[close+1:])
	if !strings.HasPrefix(rest, "[") {
		return nil, fmt.Errorf("bad sequence literal %s: want Seq<%s>[...]", s, elem)
	}
	end, err := balanced(rest, 0)
	if err != nil || end != len(rest)-1 {
		return nil, fmt.Errorf("bad sequence literal %s: want Seq<%s>[...]", s, elem)
	}
	inner := strings.TrimSpace(rest[1:end])
	var items []*Small
	if inner != "" {
		for _, p := range splitTop(inner, ',') {
			it, err := parseSmall(p)
			if err != nil {
				return nil, err
			}
			items = append(items, it)
		}
	}
	return &Small{Kind: "seqlit", Elem: elem, Items: items}, nil
}

// parseExchange parses one script row: exchange args (...) outcome ....
// The args group is located by balanced parens (depth- and
// string-aware like every other grouping rule); arg values are full
// expressions. Expected args must be named: an unnamed expectation
// cannot say which parameter it pins.
func parseExchange(s string) (*Small, error) {
	rest := strings.TrimSpace(s[len("exchange"):])
	if !strings.HasPrefix(rest, "args") {
		return nil, fmt.Errorf("bad exchange row (want exchange args (...) outcome ...): %s", s)
	}
	rest = strings.TrimSpace(rest[len("args"):])
	if !strings.HasPrefix(rest, "(") {
		return nil, fmt.Errorf("bad exchange row (want exchange args (...) outcome ...): %s", s)
	}
	end, err := balanced(rest, 0)
	if err != nil {
		return nil, err
	}
	args, err := parseArgs(rest[1:end])
	if err != nil {
		return nil, err
	}
	for _, a := range args {
		if !a.HasName {
			return nil, fmt.Errorf("exchange args must be named (k = v): %s", s)
		}
	}
	rest = strings.TrimSpace(rest[end+1:])
	if !strings.HasPrefix(rest, "outcome") {
		return nil, fmt.Errorf("bad exchange row (want exchange args (...) outcome ...): %s", s)
	}
	rest = strings.TrimSpace(rest[len("outcome"):])
	if rest == "" {
		return nil, fmt.Errorf("bad exchange row (missing outcome): %s", s)
	}
	out, err := parseSmall(rest)
	if err != nil {
		return nil, err
	}
	return &Small{Kind: "exchange", Args: args, Outcome: out}, nil
}

func parseArgs(s string) ([]Arg, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []Arg
	for _, part := range splitTop(s, ',') {
		i, op := findTop(part, []string{"==", ">=", "<=", "!=", "="})
		if op == "=" {
			name := strings.TrimSpace(part[:i])
			if !reName.MatchString(name) {
				return nil, fmt.Errorf("bad kwarg: %s", part)
			}
			v, err := parseSmall(part[i+1:])
			if err != nil {
				return nil, err
			}
			out = append(out, Arg{Name: name, HasName: true, V: v})
		} else {
			v, err := parseSmall(part)
			if err != nil {
				return nil, err
			}
			out = append(out, Arg{V: v})
		}
	}
	return out, nil
}

func isKwargList(args []Arg) bool {
	for _, a := range args {
		if !a.HasName {
			return false
		}
	}
	return true
}

// ------------------------------------------------------------ files --------
type row struct {
	indent int
	code   string
	line   int // 1-based source line
}

// LineError carries a 1-based source line for editor diagnostics
// (and `file:line:` CLI messages). at() attaches it once, innermost first.
type LineError struct {
	Line int
	Err  error
}

func (e *LineError) Error() string { return fmt.Sprintf("line %d: %v", e.Line, e.Err) }
func (e *LineError) Unwrap() error { return e.Err }

func at(line int, err error) error {
	if err == nil {
		return nil
	}
	var le *LineError
	if errors.As(err, &le) {
		return err
	}
	return &LineError{Line: line, Err: err}
}

var (
	reHdrLine = regexp.MustCompile(`^(provides|uses|emits)\s*\[(.*)\]$`)
	reError   = regexp.MustCompile(`^error\s+([\w.]+)\((.*)\)$`)
	reType    = regexp.MustCompile(`^type\s+(\w+)(?:<([\w\s,]+)>)?\s+rev\s+(\d+)\s*\($`)
	// reVariant mirrors reType; reVariantCase heads a case row
	// with kwargs payload fields (empty parens = nullary).
	reVariant     = regexp.MustCompile(`^variant\s+(\w+)(?:<([\w\s,]+)>)?\s+rev\s+(\d+)\s*\($`)
	reVariantCase = regexp.MustCompile(`^case\s+(\w+)\((.*)\)$`)
	reBrand       = regexp.MustCompile(`^brand\s+(\w+)\s+is\s+(\w+)\s+rev\s+(\d+)(\s+seals_from\s+\[([^\]]*)\])?$`)
	reConst       = regexp.MustCompile(`^const\s+(\w+)\s*:\s*(\w+(?:<.+>)?)\s+rev\s+(\d+)\s*=\s*(.+)$`)
	reExtern      = regexp.MustCompile(`^extern\s+(\w+)\((.*)\)\s*->\s*(\w+(?:<.+>)?)\s+rev\s+(\d+)$`)
	reFn          = regexp.MustCompile(`^fn\s+(\w+)(?:<([\w\s,]+)>)?\((.*)\)\s*->\s*(\w+(?:<.+>)?)\s+rev\s+(\d+)$`)
	reExport      = regexp.MustCompile(`^exports_utf8\s+(\w+)\s+via\s+(\w+)@(\d+)$`)
	reBridge      = regexp.MustCompile(`^asset_bridge\s+(\w+)\s*,\s*(\w+)\s+from\s+(\w+)\s+via\s+(\w+)@(\d+)\s+for\s+(\w+)$`)
	reField       = regexp.MustCompile(`^(\w+)\s*:\s*(\w+(?:<.+>)?)$`)
	reTest        = regexp.MustCompile(`^(\w+)(?:<(.+?)>)?\((.*)\)\s*=>\s*(.+)$`)
	reGiven       = regexp.MustCompile(`^(\w+)\s*=>\s*(.+)$`)
	reArm         = regexp.MustCompile(`^(?:on\s+)?(.+?)\s*=>\s*(.*)$`)
	// reContractArm heads an ensures arm: outcome plus bound name,
	// no => (predicates follow as rows). a69 owns the shape.
	reContractArm     = regexp.MustCompile(`^on\s+([A-Za-z][\w.]*)\s+(\w+)$`)
	reDecreases       = regexp.MustCompile(`^decreases\s+(\w+)$`)
	reDecreasesSchema = regexp.MustCompile(`^decreases\s+(\w+)\s*,\s*(\w+)\s+by\s+(euclid|narrowing)$`)
	reEffects         = regexp.MustCompile(`^effects\s*\[(.*)\]$`)
	// G2: the state sort admits the <> shape so expansion
	// rejects generic instances with a naming message; the
	// base-only rule itself is unchanged.
	reState     = regexp.MustCompile(`^state\s+(\w+)\s*:\s*(\w+(?:<.+>)?)\s*=\s*(.+)$`)
	reRevWord   = regexp.MustCompile(`\brev\b`)
	reTypeParam = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	rePatVar    = regexp.MustCompile(`^([\w.]+)\s+(\w+)$`)
	rePatWild   = regexp.MustCompile(`^([\w.]+)\s+_$`)
)

func parseFields(s, what string) ([][2]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out [][2]string
	for _, part := range splitFieldList(s) {
		m := reField.FindStringSubmatch(part)
		if m == nil {
			return nil, fmt.Errorf("bad %s field: %s", what, part)
		}
		out = append(out, [2]string{m[1], m[2]})
	}
	return out, nil
}

// splitFieldList splits a declaration field list (`name: Type`
// pairs) on top-level commas. Unlike splitTop it treats every
// `<` as a type-argument opener: field lists hold annotations,
// never comparisons, so `Fn<int, M__O, [m.err]>` survives whole
// while expression splitting keeps its exact existing meaning.
func splitFieldList(s string) []string {
	var parts []string
	var cur strings.Builder
	angle := 0
	var stack []byte
	pairs := map[byte]byte{'(': ')', '[': ']'}
	inStr, esc := false, false
	flush := func() {
		parts = append(parts, cur.String())
		cur.Reset()
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			cur.WriteByte(ch)
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
		} else if ch == '"' {
			inStr = true
			cur.WriteByte(ch)
		} else if ch == '<' {
			angle++
			cur.WriteByte(ch)
		} else if ch == '>' && angle > 0 {
			angle--
			cur.WriteByte(ch)
		} else if closer, ok := pairs[ch]; ok {
			stack = append(stack, closer)
			cur.WriteByte(ch)
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
			cur.WriteByte(ch)
		} else if angle == 0 && len(stack) == 0 && ch == ',' {
			flush()
		} else {
			cur.WriteByte(ch)
		}
	}
	flush()
	var keep []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			keep = append(keep, strings.TrimSpace(p))
		}
	}
	return keep
}

// dupParam names the first repeated parameter in a fn/extern
// signature, or "" when all are distinct. A repeated name
// leaves binding ambiguous (the name map keeps one slot while
// positional calls fill both), so the declaration fails here
// instead of confusing every call site downstream.
func dupParam(params [][2]string) string {
	seen := make(map[string]bool, len(params))
	for _, p := range params {
		if seen[p[0]] {
			return p[0]
		}
		seen[p[0]] = true
	}
	return ""
}

// splitTypeArgs splits a type-argument list on top-level commas,
// tracking angle-bracket depth so Seq<str> survives whole. Empty
// pieces are rejected: `f<str,>` is malformed, not short. Used for
// call-site args and row binds; semantic validation (known types,
// no nesting in G1) belongs to expansion, not parsing.
func splitTypeArgs(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("bad type arguments: unbalanced > in %s", s)
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("bad type arguments: unbalanced < in %s", s)
	}
	parts = append(parts, s[start:])
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("bad type arguments: empty slot in %s", s)
		}
		out = append(out, p)
	}
	return out, nil
}

// parseTypeParams validates a fn decl's parameter list: names are
// [A-Z][A-Za-z0-9]*, duplicates rejected like value params.
func parseTypeParams(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range splitTop(s, ',') {
		if !reTypeParam.MatchString(p) {
			return nil, fmt.Errorf("bad type parameter %s: want [A-Z][A-Za-z0-9]*", p)
		}
		if seen[p] {
			return nil, fmt.Errorf("duplicate type parameter %s", p)
		}
		seen[p] = true
		out = append(out, p)
	}
	return out, nil
}

// parseTypeBinds validates a test row's instantiation pins:
// P=E pairs in source order. Names are checked against the
// decl's parameters at expansion, where the full program is
// visible; here only the pair shape is enforced.
func parseTypeBinds(s string) ([][2]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	pieces, err := splitTypeArgs(s)
	if err != nil {
		return nil, err
	}
	var out [][2]string
	for _, p := range pieces {
		eq := strings.Index(p, "=")
		if eq <= 0 || eq == len(p)-1 {
			return nil, fmt.Errorf("bad type bind %s: want P=Type", p)
		}
		name := strings.TrimSpace(p[:eq])
		arg := strings.TrimSpace(p[eq+1:])
		if !reTypeParam.MatchString(name) {
			return nil, fmt.Errorf("bad type bind %s: param name wants [A-Z][A-Za-z0-9]*", p)
		}
		out = append(out, [2]string{name, arg})
	}
	return out, nil
}

func parseModule(path string) (*Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	m, err := parseModuleText(base, string(data))
	if err != nil {
		var le *LineError
		if errors.As(err, &le) {
			return nil, fmt.Errorf("%s:%d: %v", path, le.Line, le.Err)
		}
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return m, nil
}

func parseModuleText(name, text string) (*Module, error) {
	path := name
	// Row 4: source decoding enforces valid UTF-8 before anything
	// else runs. The pipeline cannot carry malformed bytes
	// faithfully (the emitter substitutes), so fail closed at the
	// door rather than repairing silently downstream.
	if !utf8.ValidString(text) {
		return nil, at(1, fmt.Errorf("source is not valid UTF-8: decode the file as UTF-8 before compiling"))
	}
	// R1 (a45): braces are delimiters nowhere, but data inside
	// string literals is not delimiters either, so the ban scans
	// string-aware. Comments stay banned.
	for n, raw := range strings.Split(text, "\n") {
		if scan.LegacyBraceOutsideString(raw) {
			return nil, at(n+1, fmt.Errorf("curly braces are banned outside string literals, use () records"))
		}
	}
	var rows []row
	for n, raw := range strings.Split(text, "\n") {
		code := strings.TrimRight(stripComment(raw), " \t")
		if strings.TrimSpace(code) == "" {
			continue
		}
		indent := len(code) - len(strings.TrimLeft(code, " "))
		rows = append(rows, row{indent, strings.TrimSpace(code), n + 1})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: empty file", path)
	}
	parts := strings.Fields(rows[0].code)
	if len(parts) != 2 || parts[0] != "mod" {
		return nil, at(rows[0].line, fmt.Errorf("file must open with mod <domain>"))
	}
	mod := &Module{Hdr: map[string][]string{}}
	mod.Mod = parts[1]
	base := path[strings.LastIndex(path, "/")+1:]
	mod.Stem = strings.TrimSuffix(base, ".can")
	i := 1
	for i < len(rows) && rows[i].indent > 0 {
		m := reHdrLine.FindStringSubmatch(rows[i].code)
		if m == nil {
			return nil, at(rows[i].line, fmt.Errorf("bad mod header line: %s", rows[i].code))
		}
		mod.Hdr[m[1]] = splitTop(m[2], ',')
		i++
	}
	for _, key := range []string{"provides", "uses", "emits"} {
		if _, ok := mod.Hdr[key]; !ok {
			mod.Hdr[key] = nil
		}
	}
	for i < len(rows) {
		indent, code := rows[i].indent, rows[i].code
		if indent != 0 {
			return nil, at(rows[i].line, fmt.Errorf("top-level decl must start at column 0: %s", code))
		}
		declLine := rows[i].line
		switch {
		case strings.HasPrefix(code, "error "):
			m := reError.FindStringSubmatch(code)
			if m == nil {
				return nil, at(declLine, fmt.Errorf("bad error decl: %s", code))
			}
			fields, err := parseFields(m[2], "error")
			if err != nil {
				return nil, at(declLine, err)
			}
			mod.Decls = append(mod.Decls, &ErrorDecl{Name: m[1], Fields: fields, Line: declLine})
			i++
		case strings.HasPrefix(code, "type "):
			m := reType.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad type decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[3])
			typarams, err := parseTypeParams(m[2])
			if err != nil {
				return nil, at(declLine, err)
			}
			decl := &TypeDecl{Name: m[1], Rev: rev, TypeParams: typarams, Line: declLine}
			i++
			for i < len(rows) && rows[i].indent > 0 {
				fs, err := parseFields(strings.TrimSuffix(rows[i].code, ","), "type")
				if err != nil {
					return nil, at(rows[i].line, err)
				}
				decl.Fields = append(decl.Fields, fs...)
				i++
			}
			if i >= len(rows) || rows[i].code != ")" {
				return nil, at(declLine, fmt.Errorf("type %s missing closing )", decl.Name))
			}
			i++
			mod.Decls = append(mod.Decls, decl)
		case strings.HasPrefix(code, "variant "):
			m := reVariant.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad variant decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[3])
			typarams, err := parseTypeParams(m[2])
			if err != nil {
				return nil, at(declLine, err)
			}
			decl := &VariantDecl{Name: m[1], Rev: rev, TypeParams: typarams, Line: declLine}
			i++
			for i < len(rows) && rows[i].indent > 0 {
				cm := reVariantCase.FindStringSubmatch(strings.TrimSuffix(rows[i].code, ","))
				if cm == nil {
					return nil, at(rows[i].line, fmt.Errorf("bad variant case: %s", rows[i].code))
				}
				if strings.Contains(cm[1], "__") {
					return nil, at(rows[i].line, fmt.Errorf("write the short case name, not %q: qualification is elaborated", cm[1]))
				}
				fs, err := parseFields(cm[2], "case")
				if err != nil {
					return nil, at(rows[i].line, err)
				}
				decl.Cases = append(decl.Cases, VariantCase{Short: cm[1], Fields: fs, Line: rows[i].line})
				i++
			}
			if len(decl.Cases) == 0 {
				return nil, at(declLine, fmt.Errorf("variant %s declares no cases: at least one case is required", decl.Name))
			}
			if i >= len(rows) || rows[i].code != ")" {
				return nil, at(declLine, fmt.Errorf("variant %s missing closing )", decl.Name))
			}
			i++
			mod.Decls = append(mod.Decls, decl)
		case strings.HasPrefix(code, "brand "):
			m := reBrand.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad brand decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[3])
			var from []string
			if m[4] != "" {
				for _, name := range strings.Split(m[5], ",") {
					if name = strings.TrimSpace(name); name != "" {
						from = append(from, name)
					}
				}
			}
			mod.Decls = append(mod.Decls, &BrandDecl{Name: m[1], Under: m[2], Rev: rev, SealsFrom: from, Line: declLine})
			i++
		case strings.HasPrefix(code, "const "):
			m := reConst.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad const decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[3])
			val, err := parseSmall(strings.TrimSpace(m[4]))
			if err != nil {
				return nil, at(declLine, fmt.Errorf("bad const value: %v", err))
			}
			mod.Decls = append(mod.Decls, &ConstDecl{Name: m[1], Type: m[2], Rev: rev, Value: val, Line: declLine})
			i++
		case strings.HasPrefix(code, "exports_utf8 "):
			m := reExport.FindStringSubmatch(code)
			if m == nil {
				if !strings.Contains(code, "@") {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad exports_utf8 decl: %s", code))
			}
			rev, err := strconv.Atoi(m[3])
			if err != nil {
				return nil, at(declLine, fmt.Errorf("bad exports_utf8 decl: rev out of range: %s", m[3]))
			}
			mod.Decls = append(mod.Decls, &Utf8ExportDecl{Brand: m[1], Function: m[2], Revision: rev, Line: declLine})
			i++
		case strings.HasPrefix(code, "asset_bridge "):
			m := reBridge.FindStringSubmatch(code)
			if m == nil {
				if !strings.Contains(code, "@") {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad asset_bridge decl: %s", code))
			}
			rev, err := strconv.Atoi(m[5])
			if err != nil {
				return nil, at(declLine, fmt.Errorf("bad asset_bridge decl: rev out of range: %s", m[5]))
			}
			mod.Decls = append(mod.Decls, &AssetBridgeDecl{Asset: m[1], Policy: m[2], Owner: m[3], Function: m[4], Revision: rev, Role: m[6], Line: declLine})
			i++
		case strings.HasPrefix(code, "state "):
			m := reState.FindStringSubmatch(code)
			if m == nil {
				return nil, at(declLine, fmt.Errorf("bad state decl: %s", code))
			}
			init, err := parseSmall(m[3])
			if err != nil {
				return nil, at(declLine, err)
			}
			mod.Decls = append(mod.Decls, &StateDecl{Name: m[1], Type: m[2], Init: init, Line: declLine})
			i++
		case strings.HasPrefix(code, "extern "):
			m := reExtern.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad extern decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[4])
			params, err := parseFields(m[2], "param")
			if err != nil {
				return nil, at(declLine, err)
			}
			if dup := dupParam(params); dup != "" {
				return nil, at(declLine, fmt.Errorf("duplicate param %s in extern %s", dup, m[1]))
			}
			ex := &ExternDecl{Name: m[1], Rev: rev, Params: params, Ret: m[3], Line: declLine}
			i++
			for i < len(rows) && rows[i].indent > 0 {
				mm := regexp.MustCompile(`^emits\s*\[(.*)\]$`).FindStringSubmatch(rows[i].code)
				if mm == nil {
					return nil, at(rows[i].line, fmt.Errorf("unexpected in extern %s: %s", ex.Name, rows[i].code))
				}
				ex.Emits = splitTop(mm[1], ',')
				i++
			}
			mod.Decls = append(mod.Decls, ex)
		case strings.HasPrefix(code, "fn "):
			m := reFn.FindStringSubmatch(code)
			if m == nil {
				if !reRevWord.MatchString(code) {
					return nil, at(declLine, fmt.Errorf("missing rev N: versioning is mandatory"))
				}
				return nil, at(declLine, fmt.Errorf("bad fn decl: %s", code))
			}
			rev, _ := strconv.Atoi(m[5])
			typarams, err := parseTypeParams(m[2])
			if err != nil {
				return nil, at(declLine, err)
			}
			params, err := parseFields(m[3], "param")
			if err != nil {
				return nil, at(declLine, err)
			}
			if dup := dupParam(params); dup != "" {
				return nil, at(declLine, fmt.Errorf("duplicate param %s in fn %s", dup, m[1]))
			}
			fn := &FnDecl{Name: m[1], Rev: rev, TypeParams: typarams, Params: params, Ret: m[4], Line: declLine}
			i++
			for i < len(rows) && rows[i].indent > 0 && isMetaHead(rows[i].code) {
				ind, c := rows[i].indent, rows[i].code
				metaLine := rows[i].line
				switch {
				case strings.HasPrefix(c, "emits "):
					mm := regexp.MustCompile(`^emits\s*\[(.*)\]$`).FindStringSubmatch(c)
					if mm == nil {
						return nil, at(metaLine, fmt.Errorf("bad emits: %s", c))
					}
					fn.Emits = splitTop(mm[1], ',')
					i++
				case strings.HasPrefix(c, "decreases"):
					if fn.DecNames != nil {
						return nil, at(metaLine, fmt.Errorf("duplicate decreases line"))
					}
					if mm := reDecreases.FindStringSubmatch(c); mm != nil {
						fn.DecNames = []string{mm[1]}
						i++
						break
					}
					if mm := reDecreasesSchema.FindStringSubmatch(c); mm != nil {
						fn.DecNames = []string{mm[1], mm[2]}
						fn.DecSchema = mm[3]
						i++
						break
					}
					return nil, at(metaLine, fmt.Errorf("bad decreases line: %s", c))
				case strings.HasPrefix(c, "effects "):
					mm := reEffects.FindStringSubmatch(c)
					if mm == nil {
						return nil, at(metaLine, fmt.Errorf("bad effects: %s", c))
					}
					fn.Effects = splitTop(mm[1], ',')
					i++
				case c == "requires":
					if fn.Requires != nil {
						return nil, at(metaLine, fmt.Errorf("duplicate requires block"))
					}
					i++
					for i < len(rows) && rows[i].indent > ind {
						sm, err := parseSmall(rows[i].code)
						if err != nil {
							return nil, at(rows[i].line, err)
						}
						fn.Requires = append(fn.Requires, sm)
						i++
					}
					if len(fn.Requires) == 0 {
						return nil, at(metaLine, fmt.Errorf("requires with no predicates"))
					}
				case c == "ensures":
					if fn.Ensures != nil {
						return nil, at(metaLine, fmt.Errorf("duplicate ensures block"))
					}
					i++
					for i < len(rows) && rows[i].indent > ind {
						aind, ac := rows[i].indent, rows[i].code
						aline := rows[i].line
						m := reContractArm.FindStringSubmatch(ac)
						if m == nil {
							return nil, at(aline, fmt.Errorf("bad ensures arm: %s", ac))
						}
						arm := ContractArm{Outcome: m[1], Bind: m[2], Line: aline}
						i++
						for i < len(rows) && rows[i].indent > aind {
							pc, pline := rows[i].code, rows[i].line
							if strings.HasPrefix(pc, "match ") {
								scruts, err := parseScrutList(strings.TrimSpace(pc[len("match "):]))
								if err != nil {
									return nil, at(pline, err)
								}
								node, next, err := parseMatchArms(rows, i+1, rows[i].indent, pline, scruts)
								if err != nil {
									return nil, err
								}
								node.Line = pline
								arm.Matches = append(arm.Matches, node)
								i = next
								continue
							}
							sm, err := parseSmall(pc)
							if err != nil {
								return nil, at(pline, err)
							}
							arm.Preds = append(arm.Preds, sm)
							i++
						}
						if len(arm.Preds) == 0 && len(arm.Matches) == 0 {
							return nil, at(aline, fmt.Errorf("ensures arm with no predicates: %s", ac))
						}
						fn.Ensures = append(fn.Ensures, arm)
					}
					if len(fn.Ensures) == 0 {
						return nil, at(metaLine, fmt.Errorf("ensures with no arms"))
					}
				case c == "tests":
					i++
					for i < len(rows) && rows[i].indent > ind {
						tline := rows[i].line
						tm := reTest.FindStringSubmatch(rows[i].code)
						if tm == nil {
							return nil, at(tline, fmt.Errorf("bad test case: %s", rows[i].code))
						}
						binds, err := parseTypeBinds(tm[2])
						if err != nil {
							return nil, at(tline, err)
						}
						targs, err := parseArgs(tm[3])
						if err != nil {
							return nil, at(tline, err)
						}
						// a87: a trailing `pinned` word marks the row as
						// trusted acceptance. Expectations always end in
						// `)`, so only a suffix past the closing paren
						// can be the marker; `pinned` inside strings or
						// names is untouched.
						expSrc := strings.TrimSpace(tm[4])
						pinned := false
						if strings.HasSuffix(expSrc, ") pinned") {
							expSrc = strings.TrimSpace(strings.TrimSuffix(expSrc, "pinned"))
							pinned = true
						}
						exp, err := parseSmall(expSrc)
						if err != nil {
							return nil, at(tline, err)
						}
						fn.Tests = append(fn.Tests, Test{Name: tm[1], TypeBinds: binds, Args: targs, Expected: exp, Line: tline, Pinned: pinned})
						i++
					}
				default:
					return nil, at(metaLine, fmt.Errorf("unexpected in fn %s: %s", fn.Name, c))
				}
			}
			if i < len(rows) && rows[i].code == "=" {
				return nil, at(rows[i].line, fmt.Errorf("fn %s: lone = separator removed; delete this line", fn.Name))
			}
			if i >= len(rows) || rows[i].indent == 0 {
				return nil, at(declLine, fmt.Errorf("fn %s missing body", fn.Name))
			}
			body, next, err := parseExprBlock(rows, i, -1)
			if err != nil {
				return nil, err
			}
			i = next
			// a69: ensures outcomes resolve against the complete
			// emits set, so block order is free.
			allowed := map[string]bool{"Ok": true}
			for _, e := range fn.Emits {
				allowed[e] = true
			}
			for _, a := range fn.Ensures {
				if !allowed[a.Outcome] {
					return nil, at(a.Line, fmt.Errorf("unknown contract outcome %s: want Ok or a declared emits kind", a.Outcome))
				}
			}
			// Test args stay as parsed here — named or positional.
			// resolveTestArgs (checkStatic) gives positionals their
			// parameter names before any shape, type, run, or emit
			// phase sees them.
			fn.Body = body
			mod.Decls = append(mod.Decls, fn)
		default:
			return nil, at(declLine, fmt.Errorf("unknown top-level decl: %s", code))
		}
	}
	mod.ID = filepath.Clean(path)
	mod.File = filepath.Base(mod.ID)
	return mod, nil
}

// isMetaHead reports whether a row opens fn metadata: emits,
// decreases, effects, requires, ensures, or tests — mirroring
// the metadata switch exactly. Anything else at body depth starts
// the body: with no `=` separator, the first non-metadata row is
// the body by construction, and malformed metadata still errors
// inside its own case (a leading keyword always parses as metadata,
// never as body).
func isMetaHead(c string) bool {
	return strings.HasPrefix(c, "emits ") ||
		strings.HasPrefix(c, "decreases") ||
		strings.HasPrefix(c, "effects ") ||
		c == "requires" || c == "ensures" || c == "tests"
}

// isDigits reports whether s is a non-negative integer literal:
// one or more ASCII digits, no sign, no separators.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseValuePattern parses one value-match slot: top-level `|`
// separates alternatives (quote- and bracket-aware, so string
// pipes never split), each parsed like a lone slot pattern.
// Call-match `on` branches keep parsePattern: `|` stays a parse
// error there, since V1 alternatives are value patterns only.
func parseValuePattern(s string) (Pattern, error) {
	parts := splitTopInner(s, '|', true, false, true)
	if len(parts) == 1 {
		return parsePattern(s)
	}
	alts := make([]Pattern, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return Pattern{}, fmt.Errorf("empty alternative in match pattern: %s", s)
		}
		alt, err := parsePattern(part)
		if err != nil {
			return Pattern{}, err
		}
		alts = append(alts, alt)
	}
	return Pattern{Kind: "or", Alts: alts}, nil
}

func parsePattern(s string) (Pattern, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "_":
		return Pattern{Kind: "wild"}, nil
	case s == "true":
		return Pattern{Kind: "bool", B: true}, nil
	case s == "false":
		return Pattern{Kind: "bool"}, nil
	case strings.HasPrefix(s, `"`):
		// Raw patterns validate like raw literals: a lone quote
		// panics the slice below and an unterminated literal
		// silently loses its last byte, so both fail here.
		if len(s) < 2 || !strings.HasSuffix(s, `"`) {
			return Pattern{}, fmt.Errorf("bad match pattern: %s", s)
		}
		return Pattern{Kind: "str", Str: s[1 : len(s)-1], Raw: s}, nil
	case strings.HasPrefix(s, `e"`):
		j := escClose(s[1:])
		if j < 0 {
			return Pattern{}, fmt.Errorf("unterminated interpreted string: %s", s)
		}
		j++ // account for the e prefix
		if strings.TrimSpace(s[j+1:]) != "" {
			return Pattern{}, fmt.Errorf("bad match pattern: %s", s)
		}
		decoded, err := decodeEscapes(s[2:j])
		if err != nil {
			return Pattern{}, err
		}
		return Pattern{Kind: "str", Str: decoded, Raw: s}, nil
	}
	if isDigits(s) {
		// Slice 3: a non-negative integer literal is a
		// singleton pattern over int (arbitrary precision).
		// Negative bounds wait for unary minus (slice 6).
		n, _ := new(big.Int).SetString(s, 10)
		return Pattern{Kind: "int", Num: n}, nil
	}
	if strings.Contains(s, "..") {
		// Slice 3: a closed range `LO..HI`. Bounds resolve
		// here when both sides are integer literals; anything
		// else (const names, garbage) resolves in buildWorld,
		// so diagnostics stay check-level (CAN4110/CAN2104)
		// instead of parse errors.
		parts := strings.Split(s, "..")
		if len(parts) != 2 {
			return Pattern{}, fmt.Errorf("bad match pattern: %s", s)
		}
		lo, hi := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if isDigits(lo) && isDigits(hi) {
			ln, _ := new(big.Int).SetString(lo, 10)
			hn, _ := new(big.Int).SetString(hi, 10)
			return Pattern{Kind: "range", Num: ln, Hi: hn}, nil
		}
		return Pattern{Kind: "range", LoS: lo, HiS: hi}, nil
	}
	if constNameRe.MatchString(s) {
		// Slice 1: a bare const-shaped name parses as a const
		// pattern; checkSem resolves it against declared
		// constants (falling back to variant treatment when
		// undeclared). Binder forms (with a trailing name)
		// still parse as variant patterns below.
		return Pattern{Kind: "const", Name: s}, nil
	}
	// Generic case patterns use the same explicit arguments as
	// constructions. Ok<T> binds the whole declared success value;
	// dotted errors stay monomorphic.
	if i := strings.IndexByte(s, '<'); i > 0 {
		if end := angleEnd(s, i); end > i && end+1 < len(s) && (s[end+1] == ' ' || s[end+1] == '\t') {
			base, binder := s[:i], strings.TrimSpace(s[end+1:])
			if reName.MatchString(base) && (base == "Ok" || strings.Contains(base, "__")) && reName.MatchString(binder) {
				args, err := splitTypeArgs(s[i+1 : end])
				if err != nil {
					return Pattern{}, err
				}
				if len(args) == 0 || (base == "Ok" && len(args) != 1) {
					return Pattern{}, fmt.Errorf("pattern needs exactly one success type for Ok, or case type arguments: %s", s)
				}
				return Pattern{Kind: "variant", Name: base, Var: binder, TypeArgs: args}, nil
			}
		}
	}
	if m := rePatVar.FindStringSubmatch(s); m != nil && (m[1] == "Ok" || strings.Contains(m[1], ".") || strings.Contains(m[1], "__")) {
		// a75: qualified case names parse as patterns; the
		// checker proves membership, so any __ shape parses.
		return Pattern{Kind: "variant", Name: m[1], Var: m[2]}, nil
	}
	if m := rePatWild.FindStringSubmatch(s); m != nil && (strings.Contains(m[1], ".") || strings.Contains(m[1], "__")) {
		return Pattern{Kind: "variantWild", Name: m[1]}, nil
	}
	return Pattern{}, fmt.Errorf("bad match pattern: %s", s)
}

func parseExprBlock(rows []row, i, parentIndent int) (*Node, int, error) {
	if i >= len(rows) {
		return nil, i, fmt.Errorf("unexpected end of block")
	}
	indent, code := rows[i].indent, rows[i].code
	mline := rows[i].line
	if indent <= parentIndent {
		return nil, i, at(mline, fmt.Errorf("expected expression, found dedent: %s", code))
	}
	if strings.HasPrefix(code, "match ") {
		if strings.TrimSpace(code) == "match chain" && isChainBlock(rows, i+1, indent) {
			return parseChainBlock(rows, i+1, indent, mline)
		}
		head := strings.TrimSpace(code[len("match "):])
		if ref, arg, err, matched := parseInvokeHead(head); matched {
			if err != nil {
				return nil, i, at(mline, err)
			}
			node, next, err := parseMatchArmsKind(rows, i+1, indent, mline, []*Small{ref}, MatchInvoke)
			if err != nil {
				return nil, next, err
			}
			node.InvokeArg = arg
			node.Line = mline
			return node, next, nil
		}
		scruts, err := parseScrutList(head)
		if err != nil {
			return nil, i, at(mline, err)
		}
		node, next, err := parseMatchArms(rows, i+1, indent, mline, scruts)
		if err != nil {
			return nil, next, err
		}
		node.Line = mline
		return node, next, nil
	}
	sm, err := parseSmall(code)
	if err != nil {
		return nil, i, at(mline, err)
	}
	return &Node{Small: sm, Line: mline}, i + 1, nil
}

// parseInvokeHead parses `invoke <ref> with <arg>` (B00): the
// reference is a bare name, the argument one value expression
// (singleton by construction: a comma fails Small parsing).
// ok=false falls through to ordinary scrutinee parsing, so a
// variable named invoke keeps working exactly as before; only a
// head-shaped line takes the invoke path, and then malformed
// references report here. The `with` form and bare-name target
// are a JEV-settled deviation from the b00 design sketch
// (`invoke path(input = v)`): field paths flow through Fn
// parameters instead (see the fn-callback sketch).
func parseInvokeHead(s string) (ref, arg *Small, err error, matched bool) {
	m := regexp.MustCompile(`^invoke\s+(\w+)\s+with\s+(.+)$`).FindStringSubmatch(s)
	if m == nil {
		return nil, nil, nil, false
	}
	ref, err = parseSmall(m[1])
	if err != nil {
		return nil, nil, err, true
	}
	if ref.Kind != "ref" || len(ref.Ref) != 1 {
		return nil, nil, fmt.Errorf("bad invoke target: want a bare name, got %s", m[1]), true
	}
	arg, err = parseSmall(m[2])
	if err != nil {
		return nil, nil, err, true
	}
	return ref, arg, nil, true
}

// parseScrutList parses a match scrutinee list: one value/call expression,
// or several comma-separated value expressions (docs/a28). Arity and
// call-in-multi rules belong to the checker (proper diagnostic codes);
// only malformed slots fail here.
func parseScrutList(s string) ([]*Small, error) {
	parts, err := splitMatchList(s)
	if err != nil {
		return nil, err
	}
	out := make([]*Small, 0, len(parts))
	for _, p := range parts {
		sm, err := parseSmall(p)
		if err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, nil
}

func parseMatchArms(rows []row, i, indent, mline int, scruts []*Small) (*Node, int, error) {
	kind := MatchValue
	if len(scruts) == 1 && scruts[0].Kind == "call" {
		kind = MatchCall
	}
	return parseMatchArmsKind(rows, i, indent, mline, scruts, kind)
}

// parseMatchArmsKind parses arms for a pre-decided kind. Invoke
// heads decide MatchInvoke before scrutinee shape could (a bare
// reference is not a call); every other site derives the kind
// from shape exactly as before.
func parseMatchArmsKind(rows []row, i, indent, mline int, scruts []*Small, kind MatchKind) (*Node, int, error) {
	node := &Node{IsMatch: true, Kind: kind, Scruts: scruts, Line: mline}
	for i < len(rows) && rows[i].indent > indent {
		ind, c := rows[i].indent, rows[i].code
		aline := rows[i].line
		if c == "given" {
			if node.Given != nil {
				return nil, i, at(aline, fmt.Errorf("duplicate given table"))
			}
			var err error
			node.Given, i, err = parseGivenBlock(rows, i, ind)
			if err != nil {
				return nil, i, err
			}
			continue
		}
		m := reArm.FindStringSubmatch(c)
		if m == nil {
			return nil, i, at(aline, fmt.Errorf("bad match arm: %s", c))
		}
		patParts, err := splitMatchParts(m[1], true)
		if err != nil {
			return nil, i, at(aline, err)
		}
		pats := make([]Pattern, 0, len(patParts))
		for _, p := range patParts {
			var pat Pattern
			var err error
			if kind == MatchCall || kind == MatchInvoke {
				pat, err = parsePattern(p)
			} else {
				pat, err = parseValuePattern(p)
			}
			if err != nil {
				return nil, i, at(aline, err)
			}
			pats = append(pats, pat)
		}
		rest := strings.TrimSpace(m[2])
		i++
		var rhs *Node
		rhs, i, err = parseRhs(rows, i, ind, aline, rest)
		if err != nil {
			return nil, i, err
		}
		node.Arms = append(node.Arms, Arm{Pats: pats, Rhs: rhs, Line: aline})
	}
	if len(node.Arms) == 0 {
		return nil, i, at(mline, fmt.Errorf("match with no arms"))
	}
	// Single non-call matches take no given table. Multi matches with a
	// given table parse and fail in checkGiven with CodeGivenOnLocal, so
	// the diagnostic names the rule instead of a coarse parse error.
	// Invoke matches keep their table for the same reason: checkGiven
	// rejects it with CodeGivenOnInvoke, since invocation scripts
	// nothing.
	if node.Given != nil && node.Kind == MatchValue && len(node.Scruts) == 1 {
		return nil, i, at(node.Line, fmt.Errorf("given table on a non-call match"))
	}
	return node, i, nil
}

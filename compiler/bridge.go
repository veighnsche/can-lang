// Historical compiler only: P9 constructors supersede these grants in current
// build/run/assert. Remove this legacy pass with I44; it grants no authority to
// the current compiler under compiler/internal.
// bridge.go: Schema-to-HTML asset bridge (S2 slice plan).
//
// An asset_bridge grant authorizes one exact-shape sink function to
// consume one schema-owned approval witness through the restricted
// schema__asset__fields kernel. certifyAssetBridge is a whole-program
// pass that runs after world build and before any linkage evaluation,
// in both the CLI and the editor pipelines: certificates must exist
// before contradictScriptOk can evaluate anything, or an uncertified
// refusal would launder into trusted script evidence (same barrier
// as certifyExports).
//
// A certificate is the ExportBrand annotation (reused: it names the
// granted brand on a certified call node) on the exact permitted
// kernel call, set only here after full validation. Nothing is cached
// across runs: every checkProgram/diagnose builds a fresh world, so
// grant removal, signature, or body changes invalidate by
// reconstruction even when fn@rev is unchanged.
package main

import (
	"fmt"
)

// assetFieldsKernel is the restricted witness-projection intrinsic. It
// is not a user-visible generic: the permitted nominal inputs come
// from the certificate for the current call site, never a global
// signature. It takes the witness and the policy handle positionally
// and returns the approved url, digest, and role — the minimal
// disclosure the fixed element builders need. Anything else bound
// into the witness stays hidden.
const assetFieldsKernel = "schema__asset__fields"

// assetFieldsRecord is the compiler-owned projection result: the
// approved element fields, nothing else.
const assetFieldsRecord = "Asset__Fields"

// assetBridgeRoles is the closed role set a grant may pin. It mirrors
// the schema role gate; a sink for any other mode is unsupported.
var assetBridgeRoles = map[string]bool{"stylesheet": true, "script": true}

// isAssetFields reports the restricted asset projection intrinsic.
func isAssetFields(fname string) bool {
	return fname == assetFieldsKernel
}

// bridgeGrantSite retains a grant with its owning module: ownership is
// validated identities, not matching strings.
type bridgeGrantSite struct {
	g *AssetBridgeDecl
	m *Module
}

// certifyAssetBridge validates every asset_bridge grant in the program
// and annotates each exact permitted call site. Authority failures are
// CAN6014, sink-shape failures CAN6015. A grant that names a sink
// function (even invalidly) owns that function's diagnostics: the
// uncertified-call rule fires only where no grant names the sink, so
// one problem reports once.
func certifyAssetBridge(mods []*Module, prog *Program, texts map[string]string) []Diag {
	var out []Diag
	emit := func(m *Module, d Diag) {
		d.File = qualifiedFile(mods, m)
		out = append(out, d)
	}
	var grants []bridgeGrantSite
	for _, m := range mods {
		for _, d := range m.Decls {
			if g, ok := d.(*AssetBridgeDecl); ok {
				grants = append(grants, bridgeGrantSite{g, m})
			}
		}
	}
	// Candidates are scanned directly from declarations: duplicate
	// brands are first-wins silent elsewhere, so ambiguity must be
	// detected here before that information is lost.
	brandMods := map[string][]*Module{}
	fnDecls := map[string][]*FnDecl{}
	fnMods := map[string][]*Module{}
	for _, m := range mods {
		for _, d := range m.Decls {
			switch d := d.(type) {
			case *BrandDecl:
				brandMods[d.Name] = append(brandMods[d.Name], m)
			case *FnDecl:
				fnDecls[d.Name] = append(fnDecls[d.Name], d)
				fnMods[d.Name] = append(fnMods[d.Name], m)
			}
		}
	}
	named := map[string]bool{}
	for _, gs := range grants {
		g := gs.g
		named[g.Function] = true
		text := texts[gs.m.ID]
		if gs.m.ID == "" {
			emit(gs.m, spanDiag(text, g.Line, "error",
				"asset_bridge grant has no owning module", g.Asset, CodeAssetBridgeAuthority))
			continue
		}
		if !assetBridgeRoles[g.Role] {
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge grant names unsupported role %s: want stylesheet or script", g.Role), g.Role, CodeAssetBridgeAuthority))
			continue
		}
		if g.Owner == gs.m.Mod {
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge %s via %s@%d is not two-owner: the brand owner and the sink module must differ", g.Asset, g.Function, g.Revision), g.Asset, CodeAssetBridgeAuthority))
			continue
		}
		// Each named brand must resolve to exactly one declaring
		// module, and that module must carry the claimed owner mod
		// name — unless the owner module is not loaded, in which
		// case the sink still compiles standalone on the grant's
		// claim (S4 acceptance verifies provenance where loaded).
		branded := true
		for _, name := range []string{g.Asset, g.Policy} {
			owners, ok := brandMods[name]
			if !ok {
				continue
			}
			if len(owners) != 1 || owners[0].Mod != g.Owner {
				emit(gs.m, spanDiag(text, g.Line, "error",
					fmt.Sprintf("asset_bridge grant names brand %s outside owner module %s", name, g.Owner), name, CodeAssetBridgeAuthority))
				branded = false
			}
		}
		if !branded {
			continue
		}
		fns, ok := fnDecls[g.Function]
		if !ok {
			if prog.Externs[g.Function] != nil {
				emit(gs.m, spanDiag(text, g.Line, "error",
					fmt.Sprintf("asset_bridge grant names extern %s: sinks are can functions", g.Function), g.Function, CodeAssetBridgeAuthority))
				continue
			}
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge grant names unknown function %s", g.Function), g.Function, CodeAssetBridgeAuthority))
			continue
		}
		if len(fns) != 1 {
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge grant names ambiguous function %s", g.Function), g.Function, CodeAssetBridgeAuthority))
			continue
		}
		fn := fns[0]
		fnOwner := fnMods[g.Function][0]
		if g.Revision != fn.Rev {
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge grant names %s@%d but %s declares rev %d", g.Function, g.Revision, g.Function, fn.Rev), g.Function, CodeAssetBridgeAuthority))
			continue
		}
		if gs.m.ID != fnOwner.ID {
			emit(gs.m, spanDiag(text, g.Line, "error",
				fmt.Sprintf("asset_bridge %s via %s@%d keeps its sink outside the granting module", g.Asset, g.Function, g.Revision), g.Asset, CodeAssetBridgeAuthority))
			continue
		}
		scrut, reason := bridgeShape(fn, fnOwner, g)
		if reason != "" {
			emit(fnOwner, spanDiag(texts[fnOwner.ID], fn.Line, "error",
				fmt.Sprintf("%s is not a valid asset sink: %s", fn.Name, reason), fn.Name, CodeAssetBridgeShape))
			continue
		}
		scrut.ExportBrand = g.Asset
	}
	// Uncertified calls: every schema__asset__fields scrutinee must
	// carry a certificate, unless a grant already owns this sink's
	// diagnostics. Calls outside scrutinees belong to the position
	// rule, not this pass.
	for _, m := range mods {
		text := texts[m.ID]
		for _, d := range m.Decls {
			fn, ok := d.(*FnDecl)
			if !ok || named[fn.Name] {
				continue
			}
			for _, n := range matchNodes(fn.Body) {
				if n.Kind != MatchCall || len(n.Scruts) == 0 {
					continue
				}
				s := n.Scruts[0]
				if s.Kind != "call" || !isAssetFields(s.Fname) || s.ExportBrand != "" {
					continue
				}
				emit(m, spanDiag(text, n.Line, "error",
					fmt.Sprintf("call to %s in %s is not authorized by a valid asset_bridge grant", s.Fname, fn.Name), s.Fname, CodeAssetBridgeAuthority))
			}
		}
	}
	return out
}

// bridgeShape checks the exact sink predicate: asset-then-policy params
// of exactly the granted brands, a single-Html__Safe record return from
// the sink's own module, exactly one emits kind, an empty contract, one
// call match over the kernel with both params passed bare and
// positionally, one Ok arm, and the granted role literal gating the
// projected role inside. It returns the permitted call node, or a
// reason.
func bridgeShape(fn *FnDecl, owner *Module, g *AssetBridgeDecl) (*Small, string) {
	if len(fn.Params) != 2 {
		return nil, "want exactly two parameters (asset, policy)"
	}
	if fn.Params[0][1] != g.Asset {
		return nil, fmt.Sprintf("first parameter %s must have type %s", fn.Params[0][0], g.Asset)
	}
	if fn.Params[1][1] != g.Policy {
		return nil, fmt.Sprintf("second parameter %s must have type %s", fn.Params[1][0], g.Policy)
	}
	if !singleSafeReturn(owner, fn.Ret) {
		return nil, fmt.Sprintf("must return a single-Html__Safe record from module %s", owner.Mod)
	}
	if len(fn.Emits) != 1 {
		return nil, "must declare exactly one emits kind (the builder rejection)"
	}
	if len(fn.Effects) != 0 {
		return nil, "must declare no effects"
	}
	if len(fn.DecNames) != 0 || fn.DecSchema != "" {
		return nil, "must declare no decreases clause"
	}
	n := fn.Body
	if n == nil || !n.IsMatch || n.Kind != MatchCall || len(n.Scruts) != 1 {
		return nil, fmt.Sprintf("body must be a single call match over %s", assetFieldsKernel)
	}
	scrut := n.Scruts[0]
	if scrut.Kind != "call" || scrut.Fname != assetFieldsKernel {
		return nil, fmt.Sprintf("body must be a single call match over %s", assetFieldsKernel)
	}
	if len(scrut.Args) != 2 {
		return nil, fmt.Sprintf("projection call takes the bare parameters %s, %s", fn.Params[0][0], fn.Params[1][0])
	}
	for i, p := range fn.Params {
		a := scrut.Args[i]
		if a.HasName || !isBareRef(a.V, p[0]) {
			return nil, fmt.Sprintf("projection call takes the bare parameters %s, %s positionally", fn.Params[0][0], fn.Params[1][0])
		}
	}
	if n.Given != nil {
		return nil, "projection call takes no given table"
	}
	if len(n.Arms) != 1 {
		return nil, "want exactly one Ok arm"
	}
	arm := n.Arms[0]
	if len(arm.Pats) != 1 {
		return nil, "want exactly one Ok arm"
	}
	p := arm.Pats[0]
	if p.Kind != "variant" || p.Name != "Ok" || p.Var == "" {
		return nil, "want exactly one Ok arm"
	}
	if !roleLiteralPresent(arm.Rhs, p.Var, g.Role) {
		return nil, fmt.Sprintf("must gate the projected role on %q", g.Role)
	}
	return scrut, ""
}

// singleSafeReturn reports whether ret names a record declared in the
// sink's own module with exactly one field of that module's Html__Safe
// brand: the HTML side of the two-owner authorization (Schema releases
// the witness, HTML alone mints the fragment).
func singleSafeReturn(owner *Module, ret string) bool {
	for _, d := range owner.Decls {
		t, ok := d.(*TypeDecl)
		if !ok || t.Name != ret || len(t.Fields) != 1 {
			continue
		}
		if t.Fields[0][1] != "Html__Safe" {
			return false
		}
		for _, b := range owner.Decls {
			if bd, ok := b.(*BrandDecl); ok && bd.Name == "Html__Safe" {
				return true
			}
		}
		return false
	}
	return false
}

// roleLiteralPresent searches a body for a value match over ref role of
// the projected record carrying a literal arm for the granted role: the
// structural half of role binding (the rows prove the rejection runs).
func roleLiteralPresent(n *Node, rec, role string) bool {
	found := false
	for _, m := range matchNodes(n) {
		if m.Kind != MatchValue || len(m.Scruts) != 1 {
			continue
		}
		s := m.Scruts[0]
		if s == nil || s.Kind != "ref" || len(s.Ref) != 2 || s.Ref[0] != rec || s.Ref[1] != "role" {
			continue
		}
		for _, a := range m.Arms {
			if len(a.Pats) != 1 {
				continue
			}
			p := a.Pats[0]
			if p.Kind == "str" && p.Str == role {
				found = true
			}
		}
	}
	return found
}

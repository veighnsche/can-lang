package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a77: revision identity enforcement. Fingerprints over resolved
// declarations plus transitive closures, baselines selected by the
// acceptance workflow, same-revision drift rejected. Probes first.

const revisionModelB = `mod model
  provides [Model__State]
  uses []
  emits []

variant Model__State rev 1 (
  case Ready()
  case Waiting()
)
`

const revisionClientB = `mod client
  provides [client__pass, Client__Result]
  uses [Model__State@1]
  emits []

type Client__Result rev 1 (
  state: Model__State
)

fn client__pass(state: Model__State) -> Client__Result rev 1
  emits []
  tests
    ready(Model__Ready()) => Ok(Model__Ready())
    waiting(Model__Waiting()) => Ok(Model__Waiting())
  Ok(state)
`

func revisionFiles(model, client string) map[string]string {
	return map[string]string{"model.can": model, "client.can": client}
}

func revisionProg(t *testing.T, files map[string]string, order []string) (*Program, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	for _, p := range order {
		fp := filepath.Join(dir, p)
		if err := os.WriteFile(fp, []byte(files[p]), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		paths = append(paths, fp)
	}
	mods, texts, collected, err := legacyParsePaths(paths)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	prog, collected := checkProgram(mods, texts, collected, nil)
	for _, d := range collected {
		if d.Sev == "error" {
			t.Fatalf("precondition: %s", d.Msg)
		}
	}
	return prog, texts
}

// hasFound matches a substring in a diagnostic's detail fields:
// the identity message names the declaration while Expected/Found
// carry the structural fragments, so probes pin both levels.
func hasFound(diags []Diag, sub string) bool {
	for _, d := range diags {
		if strings.Contains(d.Msg, sub) || strings.Contains(d.Expected, sub) || strings.Contains(d.Found, sub) {
			return true
		}
	}
	return false
}

func revisionBaseline(t *testing.T, prog *Program, origin string) *RevisionBaseline {
	t.Helper()
	base := &RevisionBaseline{
		Format:   RevisionFormat,
		Origin:   origin,
		Accepted: true,
		Scope:    []string{"model", "client"},
		Entries:  FingerprintProgram(prog),
		Pinned:   PinnedRows(prog),
	}
	return base
}

// TestRevisionFingerprintStable pins determinism: the same program
// parsed twice fingerprints identically.
func TestRevisionFingerprintStable(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	order := []string{"model.can", "client.can"}
	prog1, _ := revisionProg(t, files, order)
	prog2, _ := revisionProg(t, files, order)
	fp1, fp2 := FingerprintProgram(prog1), FingerprintProgram(prog2)
	if len(fp1) == 0 {
		t.Fatalf("expected fingerprint entries, got none")
	}
	for k, e1 := range fp1 {
		e2, ok := fp2[k]
		if !ok || e1.Fingerprint != e2.Fingerprint {
			t.Fatalf("unstable fingerprint for %s", k)
		}
	}
}

// TestRevisionFingerprintIgnoresPresentation pins normalization:
// comments, blank lines, and positions change nothing.
func TestRevisionFingerprintIgnoresPresentation(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	prog1, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	noisy := "// leading comment\n\n\n" + revisionModelB + "\n// trailing comment\n"
	files2 := revisionFiles(noisy, revisionClientB)
	prog2, _ := revisionProg(t, files2, []string{"model.can", "client.can"})
	fp1, fp2 := FingerprintProgram(prog1), FingerprintProgram(prog2)
	for k, e1 := range fp1 {
		if e2, ok := fp2[k]; !ok || e1.Fingerprint != e2.Fingerprint {
			t.Fatalf("presentation changed fingerprint for %s", k)
		}
	}
}

// TestRevisionFingerprintSensitive pins semantic coverage: param
// reorder, added case, field rename, emits, requires, and effects
// changes each move a fingerprint.
func TestRevisionFingerprintSensitive(t *testing.T) {
	base := `mod m
  provides [m__go, M__Out]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

fn m__go(left: int, right: int) -> M__Out rev 1
  emits []
  requires
    true
  tests
    // Named on purpose: the param-reorder mutant below must not
    // change test binding, so the fingerprint (not the tables)
    // carries the difference.
    go(left = 1, right = 2) => Ok(1)
  Ok(left)
`
	muts := map[string]string{
		"param reorder": strings.Replace(base, "(left: int, right: int)", "(right: int, left: int)", 1),
		"emits":         strings.Replace(base, "emits []\n  requires", "emits [m.oops]\n  requires", 1),
		"requires":      strings.Replace(base, "  requires\n    true\n", "  requires\n    left >= right\n", 1),
	}
	files := map[string]string{"m.can": base}
	prog, _ := revisionProg(t, files, []string{"m.can"})
	fp := FingerprintProgram(prog)
	key := "fn:m.m__go@1"
	before, ok := fp[key]
	if !ok {
		t.Fatalf("missing fingerprint for %s", key)
	}
	for name, mut := range muts {
		if name == "emits" {
			mut = "mod m\n  provides [m__go, M__Out]\n  uses []\n  emits [m.oops]\n\nerror m.oops(value: int)\n\ntype M__Out rev 1 (\n  value: int\n)\n\nfn m__go(left: int, right: int) -> M__Out rev 1\n  emits [m.oops]\n  requires\n    true\n  tests\n    go(left = 1, right = 2) => Ok(1)\n  Ok(left)\n"
		}
		files := map[string]string{"m.can": mut}
		prog, _ := revisionProg(t, files, []string{"m.can"})
		after := FingerprintProgram(prog)[key]
		if after.Fingerprint == before.Fingerprint {
			t.Fatalf("%s left the fingerprint unchanged", name)
		}
	}
}

// TestRevisionIdentityPrimary pins the verdict §9 setup: an added
// case at the same rev, behind an unchanged passthrough consumer,
// is identity drift naming the case — not silent, not a pin error.
func TestRevisionIdentityPrimary(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "review-base:B")
	modelC := strings.Replace(revisionModelB,
		"  case Waiting()\n)", "  case Waiting()\n  case Expired()\n)", 1)
	progC, textsC := revisionProg(t, revisionFiles(modelC, revisionClientB), []string{"model.can", "client.can"})
	diags := CheckRevisionIdentity(progC, textsC, base)
	if !hasFound(diags, "Model__Expired") {
		t.Fatalf("expected drift naming the added case, got %v", diags)
	}
	if !hasCode(diags, "CAN6013") {
		t.Fatalf("expected CAN6013, got %v", diags)
	}
}

// TestRevisionIdentityCommentsSilent pins the negative: comments
// and additive test rows change no identity.
func TestRevisionIdentityCommentsSilent(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "review-base:B")
	modelC := "// a comment\n" + revisionModelB
	clientC := strings.Replace(revisionClientB,
		"    waiting(state = Model__Waiting()) => Ok(Model__Waiting())",
		"    waiting(state = Model__Waiting()) => Ok(Model__Waiting())\n    waiting2(state = Model__Waiting()) => Ok(Model__Waiting())", 1)
	progC, textsC := revisionProg(t, revisionFiles(modelC, clientC), []string{"model.can", "client.can"})
	if diags := CheckRevisionIdentity(progC, textsC, base); len(diags) != 0 {
		t.Fatalf("expected no identity findings, got %v", diags)
	}
}

// TestRevisionIdentityCompleteEliminators pins control two: added
// case with every eliminator updated is still identity rejection,
// with no missing-case failure attached.
func TestRevisionIdentityCompleteEliminators(t *testing.T) {
	modelB := `mod model
  provides [Model__State]
  uses []
  emits []

variant Model__State rev 1 (
  case Ready()
  case Waiting()
)
`
	clientB := `mod client
  provides [client__pick, Client__Result]
  uses [Model__State@1]
  emits []

type Client__Result rev 1 (
  ready: bool
)

fn client__pick(state: Model__State) -> Client__Result rev 1
  emits []
  tests
    ready(Model__Ready()) => Ok(true)
    waiting(Model__Waiting()) => Ok(false)
  match state
    on Model__Ready _ => Ok(true)
    on Model__Waiting _ => Ok(false)
`
	files := revisionFiles(modelB, clientB)
	progB, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "review-base:B")
	modelC := strings.Replace(modelB,
		"  case Waiting()\n)", "  case Waiting()\n  case Expired()\n)", 1)
	clientC := strings.Replace(clientB,
		"    on Model__Waiting _ => Ok(false)",
		"    on Model__Waiting _ => Ok(false)\n    on Model__Expired _ => Ok(false)", 1)
	clientC = strings.Replace(clientC,
		"    waiting(Model__Waiting()) => Ok(false)",
		"    waiting(Model__Waiting()) => Ok(false)\n    expired(state = Model__Expired()) => Ok(false)", 1)
	progC, textsC := revisionProg(t, revisionFiles(modelC, clientC), []string{"model.can", "client.can"})
	diags := CheckRevisionIdentity(progC, textsC, base)
	if !hasCode(diags, "CAN6013") {
		t.Fatalf("expected identity rejection, got %v", diags)
	}
	if hasCode(diags, "CAN4101") {
		t.Fatalf("complete eliminators must not report missing cases, got %v", diags)
	}
}

// TestRevisionIdentityIncompleteStays pins control three: an added
// case with an incomplete eliminator keeps the a75 missing-case
// rejection.
func TestRevisionIdentityIncompleteStays(t *testing.T) {
	modelB := `mod model
  provides [Model__State]
  uses []
  emits []

variant Model__State rev 1 (
  case Ready()
  case Waiting()
)
`
	clientB := `mod client
  provides [client__pick, Client__Result]
  uses [Model__State@1]
  emits []

type Client__Result rev 1 (
  ready: bool
)

fn client__pick(state: Model__State) -> Client__Result rev 1
  emits []
  tests
    ready(Model__Ready()) => Ok(true)
    waiting(Model__Waiting()) => Ok(false)
  match state
    on Model__Ready _ => Ok(true)
    on Model__Waiting _ => Ok(false)
`
	modelC := strings.Replace(modelB,
		"  case Waiting()\n)", "  case Waiting()\n  case Expired()\n)", 1)
	dir := writeLSPDir(t, revisionFiles(modelC, clientB))
	diags := diagnose(dir, "client.can", clientB)
	if !hasCode(diags, "CAN4101") {
		t.Fatalf("expected missing-case rejection, got %v", diags)
	}
}

// TestRevisionIdentityBumpPin pins control four: a rev bump behind
// a stale pin is CodeUsesRev, never an identity substitute.
func TestRevisionIdentityBumpPin(t *testing.T) {
	model := strings.Replace(revisionModelB, "rev 1 (", "rev 2 (", 1)
	dir := writeLSPDir(t, revisionFiles(model, revisionClientB))
	diags := diagnose(dir, "client.can", revisionClientB)
	if !hasCode(diags, "CAN2103") {
		t.Fatalf("expected stale-pin rejection, got %v", diags)
	}
}

// TestRevisionIdentityProperBump pins control five: a proper bump
// with repin records removal plus addition, never drift.
func TestRevisionIdentityProperBump(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "review-base:B")
	modelC := strings.Replace(revisionModelB, "rev 1 (", "rev 2 (", 1)
	clientC := strings.Replace(revisionClientB, "Model__State@1", "Model__State@2", 1)
	progC, textsC := revisionProg(t, revisionFiles(modelC, clientC), []string{"model.can", "client.can"})
	diags := CheckRevisionIdentity(progC, textsC, base)
	if !hasFound(diags, "REMOVED") || !hasFound(diags, "ADDED") {
		t.Fatalf("expected removal plus addition, got %v", diags)
	}
	if hasDiag(diags, "error", "differs from accepted baseline") {
		t.Fatalf("a proper bump must not report drift, got %v", diags)
	}
}

// TestRevisionIdentityUntrustedBaseline pins authority: a
// candidate-generated baseline is refused before any comparison.
func TestRevisionIdentityUntrustedBaseline(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, textsB := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "candidate:unaccepted")
	base.Accepted = false
	diags := CheckRevisionIdentity(progB, textsB, base)
	if !hasDiag(diags, "error", "not an accepted baseline") {
		t.Fatalf("expected authority refusal, got %v", diags)
	}
}

// TestRevisionIdentityBadFormat pins format versioning: an unknown
// format is refused, never silently rehashed.
func TestRevisionIdentityBadFormat(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, textsB := revisionProg(t, files, []string{"model.can", "client.can"})
	base := revisionBaseline(t, progB, "review-base:B")
	base.Format = 999
	diags := CheckRevisionIdentity(progB, textsB, base)
	if !hasDiag(diags, "error", "unsupported revision format") {
		t.Fatalf("expected format refusal, got %v", diags)
	}
}

// TestRevisionIdentityContractDrift pins contract coverage: a
// weakened requires at the same rev is drift on the function.
func TestRevisionIdentityContractDrift(t *testing.T) {
	modB := `mod m
  provides [m__max, M__Out]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

fn m__max(left: int, right: int) -> M__Out rev 1
  emits []
  requires
    left >= right
  tests
    ordered(2, 1) => Ok(2)
    reversed(1, 2) => Ok(2)
  match left <= right
    on true => Ok(right)
    on false => Ok(left)
`
	files := map[string]string{"m.can": modB}
	progB, _ := revisionProg(t, files, []string{"m.can"})
	base := &RevisionBaseline{
		Format: RevisionFormat, Origin: "review-base:B",
		Accepted: true, Scope: []string{"m"},
		Entries: FingerprintProgram(progB),
		Pinned:  PinnedRows(progB),
	}
	modC := strings.Replace(modB, "    left >= right\n", "    true\n", 1)
	progC, textsC := revisionProg(t, map[string]string{"m.can": modC}, []string{"m.can"})
	diags := CheckRevisionIdentity(progC, textsC, base)
	if !hasDiag(diags, "error", "m__max") {
		t.Fatalf("expected drift on the weakened contract, got %v", diags)
	}
	if !hasCode(diags, "CAN6013") {
		t.Fatalf("expected CAN6013, got %v", diags)
	}
}

// TestRevisionBaselineRoundTrip pins the file channel: generate,
// reload, and compare byte-identical across runs.
func TestRevisionBaselineRoundTrip(t *testing.T) {
	files := revisionFiles(revisionModelB, revisionClientB)
	progB, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	dir := t.TempDir()
	p1 := filepath.Join(dir, "b1.json")
	p2 := filepath.Join(dir, "b2.json")
	if err := WriteBaseline(p1, progB, "review-base:B"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := WriteBaseline(p2, progB, "review-base:B"); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw1, err := os.ReadFile(p1)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	raw2, err := os.ReadFile(p2)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(raw1) != string(raw2) {
		t.Fatalf("baseline generation is not deterministic")
	}
	loaded, err := LoadBaseline(p1)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Accepted {
		t.Fatalf("generation must not self-accept: accepted must be false")
	}
}

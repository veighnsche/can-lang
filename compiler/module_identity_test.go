package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Module identity is the cleaned input path: two inputs with different
// identities are different modules even when their basenames match.
// Output stems stay bare while unique, then disambiguate injectively;
// the same identity twice is an CAN5007 collision.

const identProvider = `mod alpha
  provides [alpha__get, Alpha__Data, Alpha__Seal]
  uses []
  emits []

brand Alpha__Seal is str rev 1

type Alpha__Data rev 1 (
  id: str
)

fn alpha__get(id: str) -> Alpha__Data rev 1
  emits []
  tests
    r("u") => Ok("u")
  Ok(id)
`

const identConsumer = `mod beta
  provides [beta__go, Beta__Out]
  uses [alpha__get@1]
  emits []

type Beta__Out rev 1 (
  id: str
)

fn beta__go(id: str) -> Beta__Out rev 1
  emits []
  tests
    g("u") => Ok("u")
  match call alpha__get(id)
    given
      g => [exchange args (id = "u") outcome Ok("u")]
    on Ok v => Ok(v.id)
`

func writeSameBasename(t *testing.T, consumer string) (dir, a, b string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	a = filepath.Join(dir, "a", "same.can")
	b = filepath.Join(dir, "b", "same.can")
	if err := os.WriteFile(a, []byte(identProvider), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(consumer), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, a, b
}

func TestSameBasenameKeepsIdentities(t *testing.T) {
	_, a, b := writeSameBasename(t, identConsumer)
	out := t.TempDir()
	// Compiling together must succeed: the cross-module call stays
	// foreign (scripted through given) instead of collapsing to local.
	if err := compile(out, []string{a, b}); err != nil {
		t.Fatalf("compile: %v", err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	var ts []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ts") {
			ts = append(ts, e.Name())
		}
	}
	// Two modules, two artifacts: neither overwrote the other.
	if len(ts) != 2 {
		t.Fatalf("want 2 emitted artifacts, got %v", ts)
	}
	var alphaTS, betaTS string
	for _, name := range ts {
		raw, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "export function alpha__get") {
			alphaTS = name
		}
		if strings.Contains(string(raw), "export function beta__go") {
			betaTS = name
		}
	}
	if alphaTS == "" || betaTS == "" {
		t.Fatalf("artifacts lost an owner: %v", ts)
	}
	if alphaTS == betaTS {
		t.Fatalf("both modules emitted one artifact: %s", alphaTS)
	}
	beta, err := os.ReadFile(filepath.Join(out, betaTS))
	if err != nil {
		t.Fatal(err)
	}
	// The foreign import resolves to the provider's own artifact.
	if !strings.Contains(string(beta), "./"+strings.TrimSuffix(alphaTS, ".ts")) {
		t.Errorf("consumer does not import the provider artifact:\n%s", beta)
	}
}

func TestSameBasenameSealStaysForeign(t *testing.T) {
	forge := `mod beta
  provides [beta__forge, Beta__Out]
  uses [Alpha__Seal@1]
  emits []

type Beta__Out rev 1 (
  id: str
)

fn beta__forge() -> Beta__Out rev 1
  emits []
  tests
    f() => Ok("s")
  Ok(seal Alpha__Seal("s"))
`
	_, a, b := writeSameBasename(t, forge)
	mods, texts, collected, err := legacyParsePaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	_, collected = checkProgram(mods, texts, collected, nil)
	// Same basename must not launder a foreign brand into a local one:
	// beta mints a seal alpha declares.
	found := false
	for _, d := range collected {
		if d.Sev == "error" && strings.Contains(d.Msg, "mints a brand declared in") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected foreign-seal refusal, got %v", collected)
	}
}

func TestDuplicateIdentityIsCAN5007(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "m.can")
	if err := os.WriteFile(f, []byte(identProvider), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, collected, err := legacyParsePaths([]string{f, f})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(collected, "CAN5007") {
		t.Fatalf("expected CAN5007, got %v", collected)
	}
	out := t.TempDir()
	if err := compile(out, []string{f, f}); err == nil {
		t.Fatal("expected duplicate identity to fail the build")
	} else if !strings.Contains(err.Error(), "inputs twice") {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ts") {
			t.Fatalf("collision wrote an artifact: %s", e.Name())
		}
	}
}

func TestAssignStems(t *testing.T) {
	mk := func(id string) *Module {
		m, err := parseModuleText(id, "mod m\n  provides []\n  uses []\n  emits []\n")
		if err != nil {
			t.Fatalf("parse %s: %v", id, err)
		}
		return m
	}
	t.Run("distinct keeps bare stems", func(t *testing.T) {
		mods := []*Module{mk("a/x.can"), mk("b/y.can")}
		legacyAssignStems(mods)
		if mods[0].Stem != "x" || mods[1].Stem != "y" {
			t.Fatalf("stems = %q, %q", mods[0].Stem, mods[1].Stem)
		}
	})
	t.Run("shared basename disambiguates", func(t *testing.T) {
		mods := []*Module{mk("b/same.can"), mk("a/same.can")}
		legacyAssignStems(mods)
		got := map[string]bool{mods[0].Stem: true, mods[1].Stem: true}
		if len(got) != 2 {
			t.Fatalf("stems collide: %q, %q", mods[0].Stem, mods[1].Stem)
		}
	})
	t.Run("sanitize tie breaks with suffix", func(t *testing.T) {
		mods := []*Module{mk("a_b.can"), mk("p/a_b.can"), mk("p_a_b.can")}
		legacyAssignStems(mods)
		got := map[string]bool{}
		for _, m := range mods {
			got[m.Stem] = true
		}
		if len(got) != 3 {
			t.Fatalf("stems collide: %q, %q, %q", mods[0].Stem, mods[1].Stem, mods[2].Stem)
		}
	})
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/internal/scan"
)

const goodAlpha = `package alpha
    provides [item]
    uses [codec]

record item
    int id
`

const goodBeta = `package beta
    provides [describe]
    uses [alpha, codec]

fn str describe
    emits []
    given
        int seed
    asserts
        sample: 1 => ok "one"
    ok "one"
`

const goodAliased = `package gamma
    provides []
    uses [alpha as other]
`

func writeFixtures(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func testCatalogue() map[string]bool {
	return map[string]bool{"codec": true, "http": true}
}

func TestRepoPasses(t *testing.T) {
	root, err := scan.RepoRoot()
	if err != nil {
		t.Skip("not in repo checkout")
	}
	catalogue, err := cataloguePackages(root)
	if err != nil {
		t.Fatal(err)
	}
	scanned, errs := check(maintainedRoots(root), catalogue)
	if len(errs) > 0 {
		t.Fatalf("repo check failed: %v", errs)
	}
	if scanned == 0 {
		t.Fatal("no fixtures scanned")
	}
}

func TestGoodPair(t *testing.T) {
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": goodBeta})
	scanned, errs := check([]string{dir}, testCatalogue())
	if len(errs) > 0 {
		t.Fatalf("good pair failed: %v", errs)
	}
	if scanned != 2 {
		t.Fatalf("scanned=%d", scanned)
	}
}

func TestAliasUses(t *testing.T) {
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "gamma.can": goodAliased})
	if _, errs := check([]string{dir}, testCatalogue()); len(errs) > 0 {
		t.Fatalf("aliased uses failed: %v", errs)
	}
}

func TestQualifiedUses(t *testing.T) {
	good := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [vendor::alpha, codec]", 1)
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": good})
	if _, errs := check([]string{dir}, testCatalogue()); len(errs) > 0 {
		t.Fatalf("qualified uses failed: %v", errs)
	}
	bad := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [vendor::ghost]", 1)
	dir = writeFixtures(t, map[string]string{"beta.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "uses vendor::ghost resolves nowhere") {
		t.Fatalf("expected resolution violation, got %v", errs)
	}
	twin := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [one::alpha, two::alpha]", 1)
	dir = writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": twin})
	_, errs = check([]string{dir}, testCatalogue())
	if !contains(errs, `import alias "alpha" collides`) {
		t.Fatalf("expected twin-alias collision, got %v", errs)
	}
}

func TestUnresolvableUses(t *testing.T) {
	bad := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [ghost]", 1)
	dir := writeFixtures(t, map[string]string{"beta.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "uses ghost resolves nowhere") {
		t.Fatalf("expected resolution violation, got %v", errs)
	}
}

func TestDuplicateProvides(t *testing.T) {
	bad := strings.Replace(goodAlpha, "provides [item]", "provides [item, item]", 1)
	dir := writeFixtures(t, map[string]string{"alpha.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, `duplicate provides entry "item"`) {
		t.Fatalf("expected dup-provides violation, got %v", errs)
	}
}

func TestDuplicateUses(t *testing.T) {
	bad := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [alpha, alpha]", 1)
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, `duplicate uses entry "alpha"`) {
		t.Fatalf("expected dup-uses violation, got %v", errs)
	}
}

func TestAliasCollision(t *testing.T) {
	bad := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [alpha as codec, codec]", 1)
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, `import alias "codec" collides`) {
		t.Fatalf("expected alias-collision violation, got %v", errs)
	}
}

func TestMissingHeader(t *testing.T) {
	dir := writeFixtures(t, map[string]string{"nope.can": "fn int main\n    ok 1\n"})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "no parseable package header") {
		t.Fatalf("expected header violation, got %v", errs)
	}
}

func TestPinBanned(t *testing.T) {
	bad := strings.Replace(goodBeta, "uses [alpha, codec]", "uses [alpha@2]", 1)
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "@N revision pin is gone") {
		t.Fatalf("expected pin violation, got %v", errs)
	}
}

func TestPinInStringAllowed(t *testing.T) {
	withPin := goodBeta + "\nstr tag = \"can.std.option@1::some\"\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": withPin})
	if _, errs := check([]string{dir}, testCatalogue()); len(errs) > 0 {
		t.Fatalf("string pin failed: %v", errs)
	}
}

func TestRetiredModHeader(t *testing.T) {
	bad := "mod alpha\n  provides [item]\n  uses []\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "old.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "no parseable package header") {
		t.Fatalf("expected header violation, got %v", errs)
	}
}

func TestExternBanned(t *testing.T) {
	bad := goodAlpha + "\nextern fn helper()\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "inline extern fn is gone") {
		t.Fatalf("expected extern violation, got %v", errs)
	}
}

func TestDecLiteralBanned(t *testing.T) {
	bad := goodAlpha + "\ndec ratio = d\"1.5\"\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "dec literal is gone") {
		t.Fatalf("expected dec violation, got %v", errs)
	}
}

func TestBracesBanned(t *testing.T) {
	bad := goodAlpha + "\nrecord item { int id }\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": bad})
	_, errs := check([]string{dir}, testCatalogue())
	if !contains(errs, "curly braces are banned") {
		t.Fatalf("expected brace violation, got %v", errs)
	}
}

func TestBracesInStringsAllowed(t *testing.T) {
	withBraces := goodBeta + "\nstr shape = \"{\\\"case\\\": 1}\"\n"
	dir := writeFixtures(t, map[string]string{"alpha.can": goodAlpha, "beta.can": withBraces})
	if _, errs := check([]string{dir}, testCatalogue()); len(errs) > 0 {
		t.Fatalf("string braces failed: %v", errs)
	}
}

func contains(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

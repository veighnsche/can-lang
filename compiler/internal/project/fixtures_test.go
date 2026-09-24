package project

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureDigestExactFramingAndOrder(t *testing.T) {
	files := []Fixture{{Relative: "b.json", Bytes: []byte("two\r\n")}, {Relative: "a.json", Bytes: []byte("one\n")}}
	got, err := FixtureDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	// Independently constructed framing bytes: the P15.1 prefix plus zero,
	// then sorted path/content lengths with exact bytes (including CRLF).
	framing, err := hex.DecodeString("63616e2d666978747572652d747265652d763100" +
		"0000000000000006" + "612e6a736f6e" + "0000000000000004" + "6f6e650a" +
		"0000000000000006" + "622e6a736f6e" + "0000000000000005" + "74776f0d0a")
	if err != nil {
		t.Fatal(err)
	}
	if got != Digest(framing) {
		t.Fatalf("wrong fixture framing: %s / %s", got, Digest(framing))
	}
	files[0], files[1] = files[1], files[0]
	again, err := FixtureDigest(files)
	if err != nil || again != got {
		t.Fatal("input order changed digest")
	}
	empty, err := FixtureDigest(nil)
	if err != nil || empty != Digest([]byte("can-fixture-tree-v1\x00")) {
		t.Fatal("empty set must hash the prefix plus zero alone")
	}
	equalBytes, err := FixtureDigest([]Fixture{{Relative: "a.json", Bytes: []byte("same")}, {Relative: "b.json", Bytes: []byte("same")}})
	if err != nil {
		t.Fatal(err)
	}
	single, err := FixtureDigest([]Fixture{{Relative: "a.json", Bytes: []byte("same")}})
	if err != nil || equalBytes == single || equalBytes == empty {
		t.Fatal("distinct paths with equal bytes must be retained")
	}
	changed, err := FixtureDigest([]Fixture{{Relative: "a.json", Bytes: []byte("one\n")}, {Relative: "b.json", Bytes: []byte("two\n")}})
	if err != nil || changed == got {
		t.Fatal("newline change not detected")
	}
	if _, err := FixtureDigest([]Fixture{{Relative: "a.json"}, {Relative: "a.json"}}); err == nil {
		t.Fatal("duplicate fixture path accepted")
	}
	if _, err := FixtureDigest([]Fixture{{Relative: "../escape.json"}}); err == nil {
		t.Fatal("unnormalized fixture path accepted")
	}
}

func writeRawProject(t *testing.T, root string) {
	t.Helper()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, root, "src/a/one.can", "package alpha\n    provides [one]\n    uses []\nrecord item\n    int value\nfn int one\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n            using raw \"fixtures/one.json\"\n        shared: 1 => ok 2\n            using raw \"../shared/case.json\"\n    ok value + 1\n")
	writeFixture(t, root, "src/a/fixtures/one.json", `{"case":"one"}`)
	writeFixture(t, root, "src/shared/case.json", `{"case":"shared"}`)
	writeFixture(t, root, "src/b/two.can", "package beta\n    provides [two]\n    uses []\nrecord item\n    int value\nfn int two\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n    match call one(value)\n        when\n            sample: 1 => ok 2\n                using raw \"../shared/case.json\"\n        ok int got => match call one(got)\n            when\n                nested: 1 => ok 1\n                    using raw \"fixtures/nested.json\"\n            ok int deep => ok deep\nfixture pair for two\n    given\n        int value\n    cases\n        1 => ok 2\n            using raw \"fixtures/two.json\"\n")
	writeFixture(t, root, "src/b/fixtures/two.json", `{"case":"two"}`)
	writeFixture(t, root, "src/b/fixtures/nested.json", `{"case":"nested"}`)
}

func TestCaptureFixturesDedupesSharedReferences(t *testing.T) {
	root := t.TempDir()
	writeRawProject(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, fixture := range graph.Root.CheckedFixtures {
		got[fixture.Relative] = string(fixture.Bytes)
	}
	want := map[string]string{
		"src/a/fixtures/one.json":    `{"case":"one"}`,
		"src/shared/case.json":       `{"case":"shared"}`,
		"src/b/fixtures/two.json":    `{"case":"two"}`,
		"src/b/fixtures/nested.json": `{"case":"nested"}`,
	}
	if len(got) != len(want) {
		t.Fatalf("captured %d fixtures, want %d: %v", len(got), len(want), got)
	}
	for relative, bytes := range want {
		if got[relative] != bytes {
			t.Fatalf("fixture %s captured %q", relative, got[relative])
		}
	}
	var ordered []Fixture
	for _, fixture := range graph.Root.CheckedFixtures {
		ordered = append(ordered, fixture)
	}
	digest, err := FixtureDigest(ordered)
	if err != nil || digest != graph.Root.FixturesSHA256 {
		t.Fatal("recorded digest does not match captured bytes")
	}
	// Relocation keeps the digest: only manifest-relative paths enter it.
	moved := t.TempDir()
	writeRawProject(t, moved)
	second, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	if second.Root.FixturesSHA256 != graph.Root.FixturesSHA256 {
		t.Fatal("relocation changed the fixture digest")
	}
}

func TestCaptureFixturesRejects(t *testing.T) {
	row := func(path string) string {
		return "package alpha\n    provides [one]\n    uses []\nrecord item\n    int value\nfn int one\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n            using raw \"" + path + "\"\n    ok value + 1\n"
	}
	base := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		writeFixture(t, root, "can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
		return root
	}
	t.Run("missing", func(t *testing.T) {
		root := base(t)
		writeFixture(t, root, "src/main.can", row("fixtures/gone.json"))
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), `raw fixture "fixtures/gone.json" is not readable`) {
			t.Fatalf("missing fixture loaded: %v", err)
		}
	})
	t.Run("escape", func(t *testing.T) {
		root := base(t)
		// From src/, two levels up leaves the project root entirely.
		name := "lf14-outside-" + filepath.Base(root) + ".json"
		if err := os.WriteFile(filepath.Join(filepath.Dir(root), name), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(filepath.Join(filepath.Dir(root), name)) })
		writeFixture(t, root, "src/main.can", row("../../"+name))
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "escapes its project") {
			t.Fatalf("escaping fixture loaded: %v", err)
		}
	})
	t.Run("absolute", func(t *testing.T) {
		root := base(t)
		writeFixture(t, root, "src/main.can", row("/tmp/case.json"))
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "must be a source-relative path") {
			t.Fatalf("absolute fixture loaded: %v", err)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		root := base(t)
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "secret.json"), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
		// The link lives outside the source walk but inside the project,
		// so fixture resolution (not source loading) must refuse it.
		if err := os.MkdirAll(filepath.Join(root, "links"), 0700); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, "src/main.can", row("../links/evil.json"))
		if err := os.Symlink(filepath.Join(outside, "secret.json"), filepath.Join(root, "links", "evil.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "escapes its project") {
			t.Fatalf("symlink escape loaded: %v", err)
		}
	})
	t.Run("size", func(t *testing.T) {
		root := base(t)
		writeFixture(t, root, "src/main.can", row("fixtures/big.json"))
		big := make([]byte, MaxFixtureBytes+1)
		for i := range big {
			big[i] = ' '
		}
		file := filepath.Join(root, "src", "fixtures", "big.json")
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, big, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "exceeds its size bound") {
			t.Fatalf("oversize fixture loaded: %v", err)
		}
	})
}

func TestFixtureBytesServesCapturedSnapshot(t *testing.T) {
	root := t.TempDir()
	writeRawProject(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	// A later working-tree edit cannot change captured bytes: check-time
	// loading and staged workers observe the snapshot, not the new file.
	writeFixture(t, root, "src/shared/case.json", `{"case":"edited"}`)
	data, err := graph.Root.FixtureBytes(filepath.Join(graph.Root.Root, "src", "a"), "../shared/case.json")
	if err != nil || string(data) != `{"case":"shared"}` {
		t.Fatalf("capture not served: %q %v", data, err)
	}
	if _, err := graph.Root.FixtureBytes(filepath.Join(graph.Root.Root, "src", "a"), "fixtures/uncaptured.json"); err == nil || !strings.Contains(err.Error(), "is not readable") {
		t.Fatalf("missing file served: %v", err)
	}
	writeFixture(t, root, "src/a/fixtures/uncaptured.json", `{}`)
	if _, err := graph.Root.FixtureBytes(filepath.Join(graph.Root.Root, "src", "a"), "fixtures/uncaptured.json"); err == nil || !strings.Contains(err.Error(), "was not captured") {
		t.Fatalf("unreferenced file served: %v", err)
	}
}

func writeVendorRawProject(t *testing.T, root, fixtureBytes string) {
	t.Helper()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, root, "src/main.can", sourceText("alpha", ""))
	writeFixture(t, root, "vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	writeFixture(t, root, "vendor/can.errors.json", `{"retired":[],"active":[]}`)
	writeFixture(t, root, "vendor/src/lib.can", "package gamma\n    provides [load]\n    uses []\nrecord item\n    int value\nfn int load\n    emits []\n    given\n        int value\n    asserts\n        decoded: 1 => ok 2\n            using raw \"fixtures/load.json\"\n    ok value + 1\n")
	writeFixture(t, root, "vendor/src/fixtures/load.json", fixtureBytes)
	writeFixtureLockWith(t, root, map[string]string{"vendor": "vendor"}, map[string][]Fixture{
		"vendor": {{Relative: "src/fixtures/load.json", Bytes: []byte(fixtureBytes)}},
	})
}

func TestVendorFixtureChangeStalesLock(t *testing.T) {
	root := t.TempDir()
	writeVendorRawProject(t, root, `{"version":1}`)
	before, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Changing only the dependency's raw fixture bytes stales the lock.
	writeFixture(t, root, "vendor/src/fixtures/load.json", `{"version":2}`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "stale dependency digest") {
		t.Fatalf("fixture-only edit kept a fresh lock: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
	if err != nil || string(after) != string(locked) {
		t.Fatal("failed verification rewrote the lock")
	}
	// A relocked tree carries a different verification identity while the
	// manifest and source digests stay identical.
	relocked := t.TempDir()
	writeVendorRawProject(t, relocked, `{"version":2}`)
	second, err := Load(relocked)
	if err != nil {
		t.Fatal(err)
	}
	vendor, other := before.Root.Dependencies["vendor"], second.Root.Dependencies["vendor"]
	if vendor.ManifestSHA256 != other.ManifestSHA256 || vendor.SourceSHA256 != other.SourceSHA256 {
		t.Fatal("manifest/source identity moved on a fixture-only change")
	}
	if vendor.FixturesSHA256 == other.FixturesSHA256 {
		t.Fatal("fixture-only change kept the fixture digest")
	}
	if before.LockSHA256 == second.LockSHA256 {
		t.Fatal("relock kept the lock identity")
	}
	// Tampering with the locked digest alone also rejects.
	tampered := string(locked)
	tampered = strings.Replace(tampered, before.Root.Dependencies["vendor"].FixturesSHA256, strings.Repeat("0", 64), 1)
	writeFixture(t, root, "vendor/src/fixtures/load.json", `{"version":1}`)
	writeFixture(t, root, "can.lock.json", tampered)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "stale dependency digest") {
		t.Fatalf("tampered fixture digest loaded: %v", err)
	}
}

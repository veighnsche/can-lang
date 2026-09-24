package project

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestExactShapeAndInertSQL(t *testing.T) {
	raw := `{"source_root":"src","dependencies":{"vendor":"vendor"},"assets":{"site-css":"assets/site.css"},"sql":{"find":{"dialect":"postgresql","statement":"SELECT id FROM things WHERE name=$1 LIMIT $2","parameters":["name"],"parameter_type":"app::parameters","row_type":"app::row<int>","cardinality":"many","row_limit_parameter":2}},"error_registry":"can.errors.json"}`
	manifest, err := ParseManifest([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SourceRoot != "src" || manifest.Dependencies["vendor"] != "vendor" || manifest.SQL["find"].RowLimitParameter != 2 {
		t.Fatal(manifest)
	}
	// No database or SQL parser runs here: P12 statement/row shape validation is
	// a later compiler-owned native parser pass, not a second SQL implementation.
	if _, err := ParseManifest([]byte(strings.Replace(raw, "SELECT id FROM things WHERE name=$1 LIMIT $2", "still inert SQL data", 1))); err != nil {
		t.Fatal(err)
	}
}

func TestManifestRejectsUnknownDuplicateAndWrongTypedFields(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{}`, `{"source_root":"src"}`,
		`{"source_root":"src","error_registry":"e","scripts":{}}`,
		`{"source_root":"src","source_root":"else","error_registry":"e"}`,
		`{"source_root":"src","source\u005froot":"else","error_registry":"e"}`,
		`{"source_root":null,"error_registry":"e"}`, `{"source_root":1,"error_registry":"e"}`,
		`{"source_root":"src","error_registry":"e","dependencies":null}`,
		`{"source_root":"src","error_registry":"e","dependencies":{"bad-key":"vendor"}}`,
		`{"source_root":"src","error_registry":"e","dependencies":{"good":"vendor","good":"else"}}`,
		`{"source_root":"src","error_registry":"e","assets":{"a":false}}`,
		`{"source_root":"src","error_registry":"e","sql":[]}`,
		`{"source_root":"src","error_registry":"e","sql":{"q":{"script":"run"}}}`,
		`{"source_root":"\ud800","error_registry":"e"}`,
		`{"source_root":"\udc00","error_registry":"e"}`,
		`{"source_root":"\ud800\u0061","error_registry":"e"}`,
		`{"source_root":"src","error_registry":"e"} {}`,
	} {
		if _, err := ParseManifest([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := ParseManifest([]byte("{\"source_root\":\"\xff\",\"error_registry\":\"e\"}")); err == nil {
		t.Fatal("invalid UTF8 admitted")
	}
	for _, path := range []string{"", "../src", "src/../else", "./src", "src//sub", "src/", "/src", "src/..", "a\u0000b"} {
		if err := NormalizePath(path); err == nil {
			t.Errorf("accepted path %q", path)
		}
	}
	for _, raw := range []string{`{"source_root":"\ud83d\ude00","error_registry":"e"}`, `{"source_root":"a\\ud800","error_registry":"e"}`, `{"source_root":".","error_registry":"e"}`} {
		if _, err := ParseManifest([]byte(raw)); err != nil {
			t.Errorf("rejected scalar/relative path: %v", err)
		}
	}
}

func TestSQLDescriptorShapeRefusals(t *testing.T) {
	base := `{"source_root":"src","error_registry":"e","sql":{"q":{"dialect":"postgresql","statement":"SELECT $1 LIMIT $2","parameters":["name"],"parameter_type":"app::parameters","row_type":"app::row","cardinality":"many","row_limit_parameter":2}}}`
	for _, pair := range [][2]string{
		{`"postgresql"`, `"oracle"`}, {`"many"`, `"execute"`}, {`"many"`, `"unknown"`},
		{`"row_limit_parameter":2`, `"row_limit_parameter":2.0`}, {`"row_limit_parameter":2`, `"row_limit_parameter":1`},
		{`,"row_limit_parameter":2`, ``}, {`"parameters":["name"]`, `"parameters":["name","name"]`},
		{`"app::row"`, `"row"`}, {`"app::row"`, `"app::row[]"`}, {`"statement":`, `"hook":"command","statement":`},
	} {
		if _, err := ParseManifest([]byte(strings.Replace(base, pair[0], pair[1], 1))); err == nil {
			t.Fatal(pair)
		}
	}
	execute := strings.Replace(strings.Replace(base, `"many"`, `"execute"`, 1), `,"row_limit_parameter":2`, "", 1)
	if _, err := ParseManifest([]byte(execute)); err != nil {
		t.Fatal(err)
	}
}

func TestRealpathConfinement(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inside"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, link := range []struct{ name, target string }{{"alias", "src"}, {"inside_alias", "inside"}, {"escape", outside}} {
		if err := os.Symlink(link.target, filepath.Join(root, link.name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ConfinedPath(root, "alias", true); err != nil {
		t.Fatal(err)
	}
	if _, err := ConfinedPath(root, "inside_alias", false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"escape/secret", "../secret", "src"} {
		if _, err := ConfinedPath(root, name, false); err == nil {
			t.Fatalf("unconfined/nonfile path accepted %q", name)
		}
	}
}

func TestRegistryAndLockExactContracts(t *testing.T) {
	registry := `{"active":[{"id":1000000,"kind":"app::failed"}],"retired":[1000001]}`
	r, err := ParseRegistry([]byte(registry))
	if err != nil || len(r.Active) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	for _, pair := range [][2]string{
		{`1000000`, `999999`}, {`1000000`, `2147483648`}, {`1000000`, `1e6`}, {`1000000`, `1000000.0`},
		{`"app::failed"`, `"app::failed<int>"`}, {`1000001`, `1000000`},
		{`"retired":[1000001]`, `"retired":[1000002,1000001]`},
		{`"kind":`, `"extra":true,"kind":`},
	} {
		if _, err := ParseRegistry([]byte(strings.Replace(registry, pair[0], pair[1], 1))); err == nil {
			t.Fatal(pair)
		}
	}
	digest := strings.Repeat("a", 64)
	id := "can.project.dependency/vendor"
	lock := `{"edges":{"vendor":{"target":"` + id + `","path":"vendor"}},"projects":{"` + id + `":{"lineage":"","manifest_sha256":"` + digest + `","source_sha256":"` + digest + `","fixtures_sha256":"` + digest + `","error_registry":` + registry + `,"edges":{}}}}`
	if _, err := ParseLock([]byte(lock)); err != nil {
		t.Fatal(err)
	}
	lineageLock := `{"edges":{"lib":{"target":"can.project.lineage/acme_lib","path":"vendor/lib"}},"projects":{"can.project.lineage/acme_lib":{"lineage":"acme_lib","manifest_sha256":"` + digest + `","source_sha256":"` + digest + `","fixtures_sha256":"` + digest + `","error_registry":{"active":[],"retired":[]},"edges":{}}}}`
	if _, err := ParseLock([]byte(lineageLock)); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{`"path":"vendor"`, `"path":"../vendor"`}, {digest, strings.Repeat("A", 64)}, {digest, "00"},
		{`"path":`, `"hook":"run","path":`}, {`"edges":`, `"edges":{},"edges":`},
		{`,"fixtures_sha256":"` + digest + `"`, ``}, {`"fixtures_sha256":"` + digest + `"`, `"fixtures_sha256":"00"`},
		{`"lineage":""`, `"lineage":"acme_lib"`}, {`"target":"` + id + `"`, `"target":"can.project.root"`},
	} {
		if _, err := ParseLock([]byte(strings.Replace(lock, pair[0], pair[1], 1))); err == nil {
			t.Fatal(pair)
		}
	}
	for _, raw := range []string{
		`{"edges":{},"projects":{}}`,
		`{"edges":{"vendor":{"target":"can.project.dependency/vendor","path":"vendor"}}}`,
		strings.Replace(lock, `"projects":`, `"dependencies":`, 1),
		strings.Replace(lock, id, "can.project.root", -1),
		strings.Replace(lineageLock, `"lineage":"acme_lib"`, `"lineage":"other_lib"`, 1),
		strings.Replace(lineageLock, `"lineage":"acme_lib"`, `"lineage":"Bad"`, 1),
	} {
		parsed, err := ParseLock([]byte(raw))
		if raw == `{"edges":{},"projects":{}}` {
			if err != nil || len(parsed.Edges) != 0 || len(parsed.Projects) != 0 {
				t.Fatalf("empty lock rejected: %v", err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestManifestLineageDeclaresOptionalIdentity(t *testing.T) {
	manifest, err := ParseManifest([]byte(`{"source_root":"src","project":"acme_lib","error_registry":"can.errors.json"}`))
	if err != nil || manifest.Project != "acme_lib" {
		t.Fatalf("%+v %v", manifest, err)
	}
	manifest, err = ParseManifest([]byte(`{"source_root":"src","error_registry":"can.errors.json"}`))
	if err != nil || manifest.Project != "" {
		t.Fatalf("%+v %v", manifest, err)
	}
	for _, raw := range []string{
		`{"source_root":"src","project":"","error_registry":"can.errors.json"}`,
		`{"source_root":"src","project":"Bad","error_registry":"can.errors.json"}`,
		`{"source_root":"src","project":"has-dash","error_registry":"can.errors.json"}`,
		`{"source_root":"src","project":"fn","error_registry":"can.errors.json"}`,
		`{"source_root":"src","project":1,"error_registry":"can.errors.json"}`,
		`{"source_root":"src","project":null,"error_registry":"can.errors.json"}`,
	} {
		if _, err := ParseManifest([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestSourceDigestExactFramingAndOrder(t *testing.T) {
	files := []SourceBytes{{Path: "b.can", Bytes: []byte("two\r\n")}, {Path: "a.can", Bytes: []byte("one\n")}}
	got, err := SourceDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	// Independently constructed framing bytes: prefix, then sorted path and
	// content lengths, then exact source (including CRLF).
	framing, err := hex.DecodeString("63616e2d736f757263652d747265652d7631000000000000000005612e63616e00000000000000046f6e650a0000000000000005622e63616e000000000000000574776f0d0a")
	if err != nil {
		t.Fatal(err)
	}
	if got != Digest(framing) {
		t.Fatalf("wrong source framing: %s / %s", got, Digest(framing))
	}
	files[0], files[1] = files[1], files[0]
	again, err := SourceDigest(files)
	if err != nil || again != got {
		t.Fatal("input order changed digest")
	}
	files[1].Bytes = []byte("two\n")
	changed, _ := SourceDigest(files)
	if changed == got {
		t.Fatal("newline change not detected")
	}
	if _, err := SourceDigest([]SourceBytes{{Path: "a.can"}, {Path: "a.can"}}); err == nil {
		t.Fatal("duplicate source path")
	}
}

func FuzzInertJSONContracts(f *testing.F) {
	for _, text := range []string{`{"source_root":"src","error_registry":"errors.json"}`, `{"active":[],"retired":[]}`, `{"dependencies":{}}`, `{"source_root":"\ud83d\ude00","error_registry":"e"}`} {
		f.Add([]byte(text))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// Every decoder must terminate and refuse malformed JSON without panics;
		// successful manifests must retain normalized path invariants.
		manifest, err := ParseManifest(data)
		if err == nil {
			if NormalizePath(manifest.SourceRoot) != nil || NormalizePath(manifest.ErrorRegistry) != nil {
				t.Fatal("accepted invalid manifest paths")
			}
		}
		ParseRegistry(data)
		ParseLock(data)
	})
}

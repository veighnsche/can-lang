package driver

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func artifactInputs() BuildInputs {
	id := strings.Repeat("a", 64)
	return BuildInputs{id, id, id, id, id, id}
}
func artifactFixture() []ir.Artifact {
	return []ir.Artifact{{Path: "entry.ts", Bytes: []byte("import './packages/p-a/a.ts';\n"), Imports: []string{"./packages/p-a/a.ts"}}, {Path: "packages/p-a/a.ts", Bytes: []byte("export const value=1n;\n")}}
}
func TestArtifactIdentityAndClosedImports(t *testing.T) {
	original := artifactFixture()
	a, err := PrepareOutput(artifactInputs(), "entry.ts", original)
	if err != nil {
		t.Fatal(err)
	}
	reversed := []ir.Artifact{original[1], original[0]}
	b, err := PrepareOutput(artifactInputs(), "entry.ts", reversed)
	if err != nil || !bytes.Equal(a.ManifestJSON(), b.ManifestJSON()) {
		t.Fatal("input enumeration changed content identity", err)
	}
	original[1].Bytes[0] = 'X'
	if a.files[original[1].Path][0] == 'X' {
		t.Fatal("mutable caller storage")
	}
	changed := artifactInputs()
	changed.Options = strings.Repeat("b", 64)
	c, err := PrepareOutput(changed, "entry.ts", artifactFixture())
	if err != nil || c.BuildID() == a.BuildID() {
		t.Fatal("options not bound to content identity")
	}
	var manifest OutputManifest
	if err = json.Unmarshal(a.ManifestJSON(), &manifest); err != nil {
		t.Fatal(err)
	}
	if err = validateOutputManifest(manifest, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"manifest.json/extra.ts", "MANIFEST.JSON/extra.ts", "Manifest.Json", "../entry.ts", "/entry.ts", "packages/../entry.ts", "packages/A.ts", "packages/CON.ts", "packages/a.ts.", "packages/a\\b.ts"} {
		t.Run(name, func(t *testing.T) {
			items := artifactFixture()
			items = append(items, ir.Artifact{Path: name, Bytes: []byte("x")}, ir.Artifact{Path: "packages/a.ts", Bytes: []byte("x")})
			if _, err := PrepareOutput(artifactInputs(), "entry.ts", items); err == nil {
				t.Fatal("unsafe output admitted")
			}
		})
	}
	for _, specifier := range []string{"ambient", "/absolute.ts", "./missing.ts", "../../../outside.ts", "./packages/p-a/a.ts?x", "./packages/p-a/a"} {
		t.Run(specifier, func(t *testing.T) {
			items := artifactFixture()
			items[0].Imports = []string{specifier}
			if _, err := PrepareOutput(artifactInputs(), "entry.ts", items); err == nil {
				t.Fatal("unsafe import admitted")
			}
		})
	}
	items := artifactFixture()
	items[0].NativeImports = []string{"node:fs"}
	if _, err := PrepareOutput(artifactInputs(), "entry.ts", items); err == nil {
		t.Fatal("authored native import admitted")
	}
	items = artifactFixture()
	items = append(items, ir.Artifact{Path: "packages/p-a", Bytes: []byte("x")})
	if _, err := PrepareOutput(artifactInputs(), "entry.ts", items); err == nil {
		t.Fatal("directory/file conflict")
	}
}

func TestArtifactMetadataAndPayloadRefusals(t *testing.T) {
	for _, raw := range []string{`{"schemaVersion":1,"schemaVersion":2}`, `{"files":{"a":1,"\u0061":2}}`} {
		var out any
		if err := decodeOutput([]byte(raw), &out); err == nil {
			t.Fatal("duplicate metadata admitted")
		}
	}
	for _, artifact := range []ir.Artifact{
		{Path: "assets/incorrect/logo.svg", Bytes: []byte("logo")},
		{Path: "entry.ts.map", Bytes: []byte(`{"version":2,"file":"entry.ts","sources":[],"sourcesContent":[]}`)},
		{Path: "entry.ts.map", Bytes: []byte(`{"version":3,"file":"other.ts","sources":[],"sourcesContent":[]}`)},
		{Path: "missing.ts.map", Bytes: []byte(`{"version":3,"file":"missing.ts","sources":[],"sourcesContent":[]}`)},
	} {
		items := append(artifactFixture(), artifact)
		if _, err := PrepareOutput(artifactInputs(), "entry.ts", items); err == nil {
			t.Fatal("malformed asset/map admitted", artifact.Path)
		}
	}
	p, err := PrepareOutput(artifactInputs(), "entry.ts", artifactFixture())
	if err != nil {
		t.Fatal(err)
	}
	var manifest OutputManifest
	json.Unmarshal(p.ManifestJSON(), &manifest)
	manifest.Files["packages/P-A/other.ts"] = strings.Repeat("a", 64)
	manifest.Imports["packages/P-A/other.ts"] = []string{}
	if err = validateOutputManifest(manifest, false); err == nil {
		t.Fatal("decoded case alias admitted")
	}
}

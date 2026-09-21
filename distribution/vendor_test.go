package distribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedOutputParser(t *testing.T) {
	var lock struct {
		SchemaVersion int               `json:"schemaVersion"`
		Name          string            `json:"name"`
		Version       string            `json:"version"`
		Files         map[string]string `json:"files"`
	}
	raw, err := os.ReadFile("../tools/runtime/vendor/acorn.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.SchemaVersion != 1 || lock.Name != "acorn" || lock.Version != "8.18.0" || len(lock.Files) != 2 {
		t.Fatal("unexpected parser lock")
	}
	for _, name := range []string{"tools/runtime/vendor/acorn-8.18.0.mjs", "distribution/notices/acorn-LICENSE.txt"} {
		data, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			t.Fatal(err)
		}
		if Hash(data) != lock.Files[name] {
			t.Fatalf("vendored parser asset changed: %s", name)
		}
	}
}

func TestPinnedSourceMapTools(t *testing.T) {
	var lock struct {
		SchemaVersion int               `json:"schemaVersion"`
		Kind          string            `json:"kind"`
		Bundles       map[string]string `json:"bundles"`
		Packages      []struct {
			Package string `json:"package"`
			Version string `json:"version"`
			Archive struct {
				URL, Integrity, SHA256 string
				Size                   int
			}
			License struct{ Path, SHA256 string }
		} `json:"packages"`
	}
	raw, err := os.ReadFile("notices/source-maps.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.SchemaVersion != 1 || lock.Kind != "can.source-map-tools" || len(lock.Packages) != 4 || len(lock.Bundles) != 2 {
		t.Fatal("incomplete source-map lock")
	}
	versions := map[string]string{"@jridgewell/gen-mapping": "0.3.13", "@jridgewell/trace-mapping": "0.3.31", "@jridgewell/sourcemap-codec": "1.6.0", "@jridgewell/resolve-uri": "3.1.2"}
	for _, p := range lock.Packages {
		if versions[p.Package] != p.Version || p.Archive.URL == "" || p.Archive.Integrity == "" || len(p.Archive.SHA256) != 64 || p.Archive.Size < 1 {
			t.Fatal("incomplete upstream pin", p.Package)
		}
		delete(versions, p.Package)
		data, err := os.ReadFile(filepath.Join("..", p.License.Path))
		if err != nil || Hash(data) != p.License.SHA256 {
			t.Fatal("changed upstream license", p.Package, err)
		}
	}
	if len(versions) != 0 {
		t.Fatal("missing pinned package")
	}
	for name, digest := range lock.Bundles {
		data, err := os.ReadFile(filepath.Join("..", name))
		if err != nil || Hash(data) != digest {
			t.Fatal("changed source-map bundle", name, err)
		}
	}
}

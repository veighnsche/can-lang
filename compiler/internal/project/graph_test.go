package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, name, text string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func sourceText(pkg, decl string) string {
	return "package " + pkg + "\n    provides [item]\n    uses []\nrecord item\n" + decl
}

func projectFixture(t *testing.T, root string) {
	t.Helper()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, root, "src/a/shared.can", sourceText("alpha", ""))
	writeFixture(t, root, "src/a/second.can", "package alpha\n    provides [other]\n    uses []\nrecord other\n")
	writeFixture(t, root, "src/b/shared.can", sourceText("beta", ""))
	writeFixture(t, root, "vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	writeFixture(t, root, "vendor/can.errors.json", `{"retired":[],"active":[]}`)
	writeFixture(t, root, "vendor/src/shared.can", sourceText("gamma", ""))
	writeFixtureLock(t, root, map[string]string{"vendor": "vendor"})
}

func writeFixtureLock(t *testing.T, root string, dependencies map[string]string) {
	t.Helper()
	writeFixtureLockWith(t, root, dependencies, nil)
}

// writeFixtureLockWith pins the root's direct edges plus every transitively
// reachable instance by canonical node identity. Fixture captures are keyed
// by project directory relative to the root.
func writeFixtureLockWith(t *testing.T, root string, dependencies map[string]string, fixtures map[string][]Fixture) {
	t.Helper()
	ids := map[string]string{}
	manifests := map[string]Manifest{}
	var assign func(dir string, edgePath []string)
	assign = func(dir string, edgePath []string) {
		if _, done := ids[dir]; done {
			return
		}
		manifestData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "can.project.json"))
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := ParseManifest(manifestData)
		if err != nil {
			t.Fatal(err)
		}
		manifests[dir] = manifest
		id := "can.project.dependency/" + strings.Join(edgePath, "/")
		if manifest.Project != "" {
			id = "can.project.lineage/" + manifest.Project
		}
		ids[dir] = id
		for _, name := range sortedKeys(manifest.Dependencies) {
			child := dir + "/" + manifest.Dependencies[name]
			if dir == "" {
				child = manifest.Dependencies[name]
			}
			assign(child, append(append([]string{}, edgePath...), name))
		}
	}
	for _, edge := range sortedKeys(dependencies) {
		assign(dependencies[edge], []string{edge})
	}
	rootManifestData, err := os.ReadFile(filepath.Join(root, "can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	rootManifest, err := ParseManifest(rootManifestData)
	if err != nil {
		t.Fatal(err)
	}
	edges := map[string]any{}
	for _, name := range sortedKeys(rootManifest.Dependencies) {
		dir := rootManifest.Dependencies[name]
		edges[name] = map[string]any{"target": ids[dir], "path": dir}
	}
	entries := map[string]any{}
	for _, dir := range sortedKeys(ids) {
		manifest := manifests[dir]
		manifestData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "can.project.json"))
		if err != nil {
			t.Fatal(err)
		}
		registryData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), filepath.FromSlash(manifest.ErrorRegistry)))
		if err != nil {
			t.Fatal(err)
		}
		registry, err := ParseRegistry(registryData)
		if err != nil {
			t.Fatal(err)
		}
		var files []SourceBytes
		sourceRoot := filepath.Join(root, filepath.FromSlash(dir), filepath.FromSlash(manifest.SourceRoot))
		err = filepath.WalkDir(sourceRoot, func(name string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(name, ".can") {
				return nil
			}
			rel, err := filepath.Rel(sourceRoot, name)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			files = append(files, SourceBytes{Path: filepath.ToSlash(rel), Bytes: data})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		digest, err := SourceDigest(files)
		if err != nil {
			t.Fatal(err)
		}
		fixtureDigest, err := FixtureDigest(fixtures[dir])
		if err != nil {
			t.Fatal(err)
		}
		childEdges := map[string]any{}
		for _, name := range sortedKeys(manifest.Dependencies) {
			child := dir + "/" + manifest.Dependencies[name]
			childEdges[name] = map[string]any{"target": ids[child], "path": manifest.Dependencies[name]}
		}
		entries[ids[dir]] = map[string]any{"lineage": manifest.Project, "manifest_sha256": Digest(manifestData), "source_sha256": digest, "fixtures_sha256": fixtureDigest, "error_registry": registry, "edges": childEdges}
	}
	data, err := json.Marshal(map[string]any{"edges": edges, "projects": entries})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "can.lock.json", string(data))
}

func TestGraphIdentityAndOutputStability(t *testing.T) {
	root := t.TempDir()
	projectFixture(t, root)
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Packages) != 3 || len(graph.Packages["can.project.root/alpha"].Sources) != 2 || len(graph.Projects) != 2 {
		t.Fatal("missing package/source graph")
	}
	paths := func(g *Graph) map[string]string {
		out := map[string]string{}
		for _, p := range g.Projects {
			for _, s := range p.Sources {
				out[s.ID] = s.OutputPath
			}
		}
		return out
	}
	first := paths(graph)
	if len(first) != 4 {
		t.Fatal(first)
	}
	seen := map[string]bool{}
	for id, path := range first {
		if strings.Contains(id, root) || seen[strings.ToLower(path)] {
			t.Fatal("machine path or collision", id, path)
		}
		seen[strings.ToLower(path)] = true
	}
	moved := t.TempDir()
	projectFixture(t, moved)
	second, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, paths(second)) {
		t.Fatal("relocation changed identities or output paths")
	}
	writeFixture(t, moved, "src/c/shared.can", sourceText("extra", ""))
	third, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	for id, path := range first {
		if paths(third)[id] != path {
			t.Fatal("adding a same-basename source changed an existing output")
		}
	}
	claims := map[string]string{}
	if err := claimOutput(claims, "packages/P-DIGEST", "first"); err != nil {
		t.Fatal(err)
	}
	if err := claimOutput(claims, "packages/p-digest", "second"); err == nil {
		t.Fatal("case-fold/digest collision did not refuse")
	}
}

func TestGraphRejectsStaleOrIncompleteLocks(t *testing.T) {
	for _, name := range []string{"source", "manifest", "registry", "missing", "unused", "path"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			projectFixture(t, root)
			switch name {
			case "source":
				writeFixture(t, root, "vendor/src/shared.can", sourceText("gamma", "\n// changed\n"))
			case "manifest":
				writeFixture(t, root, "vendor/can.project.json", `{ "source_root":"src","error_registry":"can.errors.json" }`)
			case "registry":
				writeFixture(t, root, "vendor/can.errors.json", `{"active":[],"retired":["gamma::failed"]}`)
			case "missing":
				if err := os.Remove(filepath.Join(root, "can.lock.json")); err != nil {
					t.Fatal(err)
				}
			case "unused":
				data, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, "can.lock.json", strings.Replace(string(data), `"vendor":{`, `"unused":{`, 1))
			case "path":
				data, err := os.ReadFile(filepath.Join(root, "can.lock.json"))
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, "can.lock.json", strings.Replace(string(data), `"path":"vendor"`, `"path":"src"`, 1))
			}
			if _, err := Load(root); err == nil {
				t.Fatal("invalid dependency lock accepted")
			}
		})
	}
}

func TestGraphRejectsPackageAndSourceAliases(t *testing.T) {
	for _, name := range []string{"package-collision", "folder-mismatch", "reserved", "source-escape", "source-alias", "directory-cycle", "dependency-escape", "dependency-cycle"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			projectFixture(t, root)
			switch name {
			case "package-collision":
				writeFixture(t, root, "src/c/new.can", sourceText("alpha", ""))
			case "folder-mismatch":
				writeFixture(t, root, "src/a/new.can", sourceText("different", ""))
			case "reserved":
				writeFixture(t, root, "src/a/shared.can", sourceText("http", ""))
			case "source-escape":
				outside := t.TempDir()
				writeFixture(t, outside, "secret.can", sourceText("secret", ""))
				if err := os.Symlink(filepath.Join(outside, "secret.can"), filepath.Join(root, "src/escape.can")); err != nil {
					t.Fatal(err)
				}
			case "source-alias":
				if err := os.Symlink("a/shared.can", filepath.Join(root, "src/alias.can")); err != nil {
					t.Fatal(err)
				}
			case "directory-cycle":
				if err := os.Symlink(".", filepath.Join(root, "src/cycle")); err != nil {
					t.Fatal(err)
				}
			case "dependency-escape":
				if err := os.Symlink(t.TempDir(), filepath.Join(root, "outside")); err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, "can.project.json", `{"source_root":"src","dependencies":{"vendor":"outside"},"error_registry":"can.errors.json"}`)
			case "dependency-cycle":
				writeFixture(t, root, "vendor/can.project.json", `{"source_root":"src","dependencies":{"self":"."},"error_registry":"can.errors.json"}`)
			}
			if _, err := Load(root); err == nil {
				t.Fatal("ambiguous/unconfined graph accepted")
			}
		})
	}
}

func TestGraphRegistrySourceAgreement(t *testing.T) {
	root := t.TempDir()
	projectFixture(t, root)
	writeFixture(t, root, "src/a/shared.can", sourceText("alpha", "error failed()\n"))
	writeFixture(t, root, "can.errors.json", `{"active":["alpha::failed"],"retired":["alpha::legacy"]}`)
	if _, err := Load(root); err != nil {
		t.Fatal(err)
	}
	// Retired names stay withdrawn per owner: redeclaring one fails even
	// though no numeric allocation collides.
	writeFixture(t, root, "src/a/shared.can", sourceText("alpha", "error failed()\nerror legacy()\n"))
	writeFixture(t, root, "can.errors.json", `{"active":["alpha::failed","alpha::legacy"],"retired":["alpha::legacy"]}`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "still active") {
		t.Fatalf("retired/active overlap admitted: %v", err)
	}
	writeFixture(t, root, "can.errors.json", `{"active":["alpha::failed"],"retired":[]}`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "differs from source") {
		t.Fatalf("source registry mismatch: %v", err)
	}
}

func TestGraphRepeatedEdgeNamesStayParentLocal(t *testing.T) {
	root := t.TempDir()
	projectFixture(t, root)
	writeFixture(t, root, "can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor","shared":"other"},"error_registry":"can.errors.json"}`)
	writeFixture(t, root, "vendor/can.project.json", `{"source_root":"src","dependencies":{"shared":"child"},"error_registry":"can.errors.json"}`)
	for _, dir := range []string{"other", "vendor/child"} {
		writeFixture(t, root, dir+"/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		writeFixture(t, root, dir+"/can.errors.json", `{"active":[],"retired":[]}`)
		pkg := "outer_child"
		if dir == "vendor/child" {
			pkg = "inner_child"
		}
		writeFixture(t, root, dir+"/src/main.can", sourceText(pkg, ""))
	}
	writeFixtureLock(t, root, map[string]string{"vendor": "vendor", "shared": "other"})
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	outer := graph.Root.Dependencies["shared"]
	inner := graph.Root.Dependencies["vendor"].Dependencies["shared"]
	if outer == inner || outer.ID != "can.project.dependency/shared" || inner.ID != "can.project.dependency/vendor/shared" {
		t.Fatalf("parent-local edges merged: %v %v", outer, inner)
	}
	if graph.Packages["can.project.dependency/shared/outer_child"] == nil || graph.Packages["can.project.dependency/vendor/shared/inner_child"] == nil {
		t.Fatal("per-instance packages missing")
	}
}

func TestContainedSourceSymlinkPreservesCanonicalPackageOwner(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	writeFixture(t, root, "can.errors.json", `{"active":[],"retired":[]}`)
	writeFixture(t, root, "owned/internal/pkg/original.can", sourceText("internal_pkg", ""))
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../owned/internal/pkg/original.can", filepath.Join(root, "src/alias.can")); err != nil {
		t.Fatal(err)
	}
	graph, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	pkg := graph.Packages["can.project.root/internal_pkg"]
	if !strings.Contains(filepath.ToSlash(pkg.Directory), "/owned/internal/pkg") || pkg.Sources[0].Name != "alias.can" || pkg.Sources[0].RelativePath != "alias.can" {
		t.Fatal("canonical ownership or logical digest path lost")
	}
	beforeID, beforeOutput, beforeDigest := pkg.Sources[0].ID, pkg.Sources[0].OutputPath, graph.Root.SourceSHA256
	if err := os.Rename(filepath.Join(root, "owned/internal/pkg/original.can"), filepath.Join(root, "owned/internal/pkg/renamed.can")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "src/alias.can")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../owned/internal/pkg/renamed.can", filepath.Join(root, "src/alias.can")); err != nil {
		t.Fatal(err)
	}
	again, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	src := again.Root.Sources[0]
	if src.ID != beforeID || src.OutputPath != beforeOutput || again.Root.SourceSHA256 != beforeDigest {
		t.Fatal("canonical symlink basename changed a logical source identity")
	}
}

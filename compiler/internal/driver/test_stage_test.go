package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/distribution"
)

const stageSuiteApp = "package app\n    provides []\n    uses []\nfn int one\n    emits {}\n    asserts\n        is_one: => ok 1\n    ok 1\nfn void main\n    emits {}\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n        also_empty: [] => ok\n    ok\n"

func stageProject(t *testing.T, app, helper string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/app/main.can": app,
	}
	if helper != "" {
		files["src/helper/helper.can"] = helper
	}
	for name, data := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type stageCall struct {
	args []string
}

type fakeStageRunner struct {
	mu     sync.Mutex
	calls  []stageCall
	list   []byte
	runErr error
	listE  error
}

func (f *fakeStageRunner) execute(_ context.Context, _, _ string, args []string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, stageCall{args: append([]string(nil), args...)})
	if len(args) == 1 && args[0] == "--list" {
		return f.list, f.listE
	}
	return nil, f.runErr
}

func (f *fakeStageRunner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func stageRuntimeDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "runtime"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func stageRuntimeDeps(t *testing.T, runtimeDir string) []ir.Artifact {
	t.Helper()
	var dependencies []ir.Artifact
	err := filepath.WalkDir(runtimeDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// Mirror the closed inventory: ship the runtime, not its tests.
			// vendor/ stays: shipped modules import it (diagnostics-core).
			if entry.Name() == "test" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ts") {
			return nil
		}
		rel, err := filepath.Rel(runtimeDir, path)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		dependencies = append(dependencies, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", rel)), Bytes: content, Runtime: true})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return dependencies
}

func stageOpts(t *testing.T, runner StageRunner) StageOptions {
	t.Helper()
	runtimeDir := stageRuntimeDir(t)
	return StageOptions{RuntimeDir: runtimeDir, EmitPrefix: "runtime", Dependencies: stageRuntimeDeps(t, runtimeDir), BunPath: "bun", Runner: runner, Validate: func(_ context.Context, prepared *PreparedOutput) error {
		// Filesystem tests inject the already-validated compiler boundary.
		prepared.validated = true
		return nil
	}, EncodeSourceMaps: stubSourceMaps, TimeoutMs: 60000}
}

// stubSourceMaps seals a structurally valid but unmapped diagnostics bundle
// for fake-runner tests, which never execute bun: every generated module
// carries its map reference, map, and index entry, mirroring the encoder
// shape that PrepareOutput requires. Map contents are empty; the real
// bootstrap test uses Runtime.encodeSourceMaps, and only that path proves
// mapped diagnostics.
func stubSourceMaps(_ context.Context, _ *check.Program, artifacts []ir.Artifact) ([]ir.Artifact, error) {
	index := sourceIndex{SchemaVersion: 1, Kind: "can.source-index", Sources: []diagnosticSource{}, Modules: []diagnosticModule{}}
	out := make([]ir.Artifact, 0, len(artifacts)*2+1)
	for _, artifact := range artifacts {
		if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
			out = append(out, artifact)
			continue
		}
		artifact.Bytes = append(append([]byte{}, artifact.Bytes...), []byte("//# sourceMappingURL="+path.Base(artifact.Path)+".map\n")...)
		out = append(out, artifact)
		out = append(out, ir.Artifact{Path: artifact.Path + ".map", Bytes: []byte(`{"version":3,"file":"` + path.Base(artifact.Path) + `","sources":[],"sourcesContent":[],"names":[],"mappings":""}`)})
		index.Modules = append(index.Modules, diagnosticModule{Path: artifact.Path})
	}
	data, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	return append(out, ir.Artifact{Path: "diagnostics/source-index.json", Bytes: data}), nil
}

func stageListJSON(t *testing.T, n int) []byte {
	t.Helper()
	roots := make([]map[string]string, 0, n)
	for i := 0; i < n; i++ {
		roots = append(roots, map[string]string{"package": "app", "declaration": "d", "name": fmt.Sprintf("root-%d", i)})
	}
	data, err := json.Marshal(roots)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestStageSuiteValidation(t *testing.T) {
	_, store, cache := reuseSetup(t, &fakeReuseAlloc{})
	defer cache.Close()
	ctx := context.Background()
	runner := &fakeStageRunner{}
	if _, err := StageSuite(ctx, nil, cache, "k", stageOpts(t, runner.execute)); err == nil {
		t.Fatal("nil store staged")
	}
	if _, err := StageSuite(ctx, store, nil, "k", stageOpts(t, runner.execute)); err == nil {
		t.Fatal("nil cache staged")
	}
	bad := stageOpts(t, runner.execute)
	bad.RuntimeDir = ""
	if _, err := StageSuite(ctx, store, cache, "k", bad); err == nil {
		t.Fatal("empty runtime directory staged")
	}
	bad = stageOpts(t, runner.execute)
	bad.EmitPrefix = ""
	if _, err := StageSuite(ctx, store, cache, "k", bad); err == nil {
		t.Fatal("empty emit prefix staged")
	}
	bad = stageOpts(t, runner.execute)
	bad.Validate = nil
	if _, err := StageSuite(ctx, store, cache, "k", bad); err == nil {
		t.Fatal("nil validator staged")
	}
	bad = stageOpts(t, runner.execute)
	bad.EncodeSourceMaps = nil
	if _, err := StageSuite(ctx, store, cache, "k", bad); err == nil {
		t.Fatal("nil source map encoder staged")
	}
	bad = stageOpts(t, runner.execute)
	bad.TimeoutMs = 0
	if _, err := StageSuite(ctx, store, cache, "k", bad); err == nil {
		t.Fatal("zero timeout staged")
	}
}

func TestStageSuiteNeedsRoots(t *testing.T) {
	root := outputProject(t)
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runner := &fakeStageRunner{}
	if _, err := StageSuite(context.Background(), store, cache, "k", stageOpts(t, runner.execute)); err == nil {
		t.Fatal("rootless suite staged")
	}
}

func TestStageSuiteCheckFailureBlocksLease(t *testing.T) {
	root := stageProject(t, "package app\n    provides []\n    uses []\nfn void main\n    emits {}\n    given\n        str[] arguments\n    asserts\n        broken: [] => ok 42\n    ok\n", "")
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runner := &fakeStageRunner{}
	if _, err := StageSuite(context.Background(), store, cache, "k", stageOpts(t, runner.execute)); err == nil {
		t.Fatal("unchecked suite staged")
	}
	if runner.count() != 0 {
		t.Fatalf("bootstrap ran %d times for an unchecked suite", runner.count())
	}
}

func TestStageSuiteHappyPathLeasesOnce(t *testing.T) {
	root := stageProject(t, stageSuiteApp, "")
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runner := &fakeStageRunner{}
	// The fixture carries three assertion roots (unit, customer, empty).
	runner.list = stageListJSON(t, 3)
	lease, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, runner.execute))
	if err != nil {
		t.Fatal(err)
	}
	if lease.BuildID() == "" {
		t.Fatal("staging leased an empty build")
	}
	lease.Close()
	if runner.count() != 4 {
		t.Fatalf("bootstrap ran %d times, want 3 roots + list", runner.count())
	}
	seen := map[string]bool{}
	runner.mu.Lock()
	for _, call := range runner.calls {
		seen[strings.Join(call.args, " ")] = true
	}
	runner.mu.Unlock()
	for _, want := range []string{"root=0", "root=1", "root=2", "--list"} {
		if !seen[want] {
			t.Fatalf("bootstrap never ran %s", want)
		}
	}
	again, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, runner.execute))
	if err != nil {
		t.Fatal(err)
	}
	if again.BuildID() != lease.BuildID() {
		t.Fatalf("same key leased %s then %s", lease.BuildID(), again.BuildID())
	}
	again.Close()
	if runner.count() != 4 {
		t.Fatalf("reused key re-ran bootstrap (%d calls)", runner.count())
	}
	// A changed key (changed source, runtime or entry) misses and rebuilds.
	runner.list = stageListJSON(t, 3)
	changed, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite-changed", stageOpts(t, runner.execute))
	if err != nil {
		t.Fatal(err)
	}
	changed.Close()
	if runner.count() != 8 {
		t.Fatalf("changed key did not rebuild (%d calls)", runner.count())
	}
}

func TestStageSuitePublicationUntouched(t *testing.T) {
	root := stageProject(t, stageSuiteApp, "")
	store := outputBegin(t, root)
	prepared := outputPrepared(t, store, "production")
	if _, err := store.Publish(prepared); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "dist", "current.json"))
	if err != nil {
		t.Fatal(err)
	}
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runner := &fakeStageRunner{list: stageListJSON(t, 3)}
	lease, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, runner.execute))
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	after, err := os.ReadFile(filepath.Join(root, "dist", "current.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("staging moved production current")
	}
	still, err := store.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	still.Close()
}

func TestStageSuiteBootstrapFailureBlocksPublish(t *testing.T) {
	root := stageProject(t, stageSuiteApp, "")
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runner := &fakeStageRunner{runErr: errors.New("root failed")}
	if _, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, runner.execute)); err == nil {
		t.Fatal("failed bootstrap leased")
	} else if !strings.Contains(err.Error(), "bootstrap run") {
		t.Fatalf("wrong failure: %v", err)
	}
	if builds := reuseBuildsDir(t, root); len(builds) != 0 {
		t.Fatalf("failed bootstrap staged %v", builds)
	}
	badList := &fakeStageRunner{list: []byte("not-json")}
	if _, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, badList.execute)); err == nil {
		t.Fatal("bad list leased")
	}
	shortList := &fakeStageRunner{list: stageListJSON(t, 1)}
	if _, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", stageOpts(t, shortList.execute)); err == nil {
		t.Fatal("short list leased")
	}
	if builds := reuseBuildsDir(t, root); len(builds) != 0 {
		t.Fatalf("failed bootstrap staged %v", builds)
	}
}

// stageManifest captures the staged generation manifest through the Validate
// hook, which receives prepared output before bootstrap runs.
func stageManifest(t *testing.T, store *OutputStore, cache *ReuseCache, key string, opts StageOptions) (buildID, source, runtime, options string) {
	t.Helper()
	opts.Validate = func(_ context.Context, prepared *PreparedOutput) error {
		prepared.validated = true
		var manifest struct {
			BuildID string
			Inputs  struct {
				Source  string
				Runtime string
				Options string
			}
		}
		if err := json.Unmarshal(prepared.ManifestJSON(), &manifest); err != nil {
			t.Fatalf("manifest decode: %v", err)
		}
		buildID, source, runtime, options = manifest.BuildID, manifest.Inputs.Source, manifest.Inputs.Runtime, manifest.Inputs.Options
		return nil
	}
	lease, err := StageSuite(context.Background(), store, cache, key, opts)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	return buildID, source, runtime, options
}

func TestStageSuiteInvalidationBindsInputs(t *testing.T) {
	newCache := func(t *testing.T, store *OutputStore) *ReuseCache {
		t.Helper()
		cache, err := NewReuseCache(store, "run-1", &fakeReuseAlloc{}, reuseOwner())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cache.Close() })
		return cache
	}
	base := stageOpts(t, (&fakeStageRunner{list: stageListJSON(t, 3)}).execute)
	root := stageProject(t, stageSuiteApp, "")
	store := outputBegin(t, root)
	baseID, baseSource, baseRuntime, baseOptions := stageManifest(t, store, newCache(t, store), "can-build-v1:base", base)

	// Changed source invalidates: a new assertion changes the source input,
	// the emitted entry digest inside options, and the generation identity.
	changedApp := stageSuiteApp + "fn int two\n    emits {}\n    asserts\n        is_two: => ok 2\n    ok 2\n"
	changedRoot := stageProject(t, changedApp, "")
	changedStore := outputBegin(t, changedRoot)
	changedOpts := stageOpts(t, (&fakeStageRunner{list: stageListJSON(t, 4)}).execute)
	srcID, srcSource, _, srcOptions := stageManifest(t, changedStore, newCache(t, changedStore), "can-build-v1:src", changedOpts)
	if srcSource == baseSource {
		t.Fatal("changed source kept its input identity")
	}
	if srcOptions == baseOptions {
		t.Fatal("changed entry kept its options identity")
	}
	if srcID == baseID {
		t.Fatal("changed source kept its generation identity")
	}

	// Changed runtime invalidates: different runtime tree content changes
	// the runtime input and options even when source and entry match.
	mutatedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mutatedDir, "marker.ts"), []byte("export const marker = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mutatedOpts := stageOpts(t, (&fakeStageRunner{list: stageListJSON(t, 3)}).execute)
	mutatedOpts.RuntimeDir = mutatedDir
	rtID, _, rtRuntime, rtOptions := stageManifest(t, store, newCache(t, store), "can-build-v1:rt", mutatedOpts)
	if rtRuntime == baseRuntime {
		t.Fatal("changed runtime kept its input identity")
	}
	if rtOptions == baseOptions {
		t.Fatal("changed runtime kept its options identity")
	}
	if rtID == baseID {
		t.Fatal("changed runtime kept its generation identity")
	}

	// Changed dependency bytes invalidate the generation even when every
	// semantic input matches: BuildID binds file content, not just inputs.
	depOpts := stageOpts(t, (&fakeStageRunner{list: stageListJSON(t, 3)}).execute)
	depOpts.Dependencies = append([]ir.Artifact(nil), depOpts.Dependencies...)
	depOpts.Dependencies[0].Bytes = append(append([]byte{}, depOpts.Dependencies[0].Bytes...), []byte("\n")...)
	depID, depSource, depRuntime, depOptions := stageManifest(t, store, newCache(t, store), "can-build-v1:dep", depOpts)
	if depSource != baseSource || depRuntime != baseRuntime || depOptions != baseOptions {
		t.Fatal("dependency bytes changed a semantic input")
	}
	if depID == baseID {
		t.Fatal("changed dependency bytes kept the generation identity")
	}
}

func TestStageSuiteRealBootstrap(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("CAN_BUN unset; real bootstrap needs bun")
	}
	root := stageProject(t, stageSuiteApp, "")
	store := outputBegin(t, root)
	alloc := &fakeReuseAlloc{}
	cache, err := NewReuseCache(store, "run-1", alloc, reuseOwner())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	runtimeDir := stageRuntimeDir(t)
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	toolRuntime := &Runtime{Root: repoRoot, Executable: bun, manifest: distribution.Manifest{Files: map[string]string{"tools/runtime/source-maps.ts": "test"}}}
	opts := StageOptions{RuntimeDir: runtimeDir, EmitPrefix: "runtime", Dependencies: stageRuntimeDeps(t, runtimeDir), BunPath: bun, Validate: func(_ context.Context, prepared *PreparedOutput) error {
		prepared.validated = true
		return nil
	}, EncodeSourceMaps: toolRuntime.encodeSourceMaps, TimeoutMs: 60000}
	lease, err := StageSuite(context.Background(), store, cache, "can-build-v1:suite", opts)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
}

func TestDefaultStageRunnerInvokesBunDirectly(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "argv.txt")
	script := filepath.Join(dir, "bun")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + record + "\ncat 0<&3 > " + record + ".fd3\nprintf '[]'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	// The staging bootstrap shells to bun with the staged entry only: no
	// candidate canlc binary, no judge build. The script stands in for bun
	// and records exactly what the runner passes.
	out, err := DefaultStageRunner(context.Background(), script, filepath.Join(dir, "entry.ts"), []string{"root=0"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "[]" {
		t.Fatalf("stdout = %q", out)
	}
	argv, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(argv)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "entry.ts") || lines[1] != "root=0" {
		t.Fatalf("argv = %q", lines)
	}
	for _, line := range lines {
		if strings.Contains(line, "canlc") {
			t.Fatalf("bootstrap invokes a candidate toolchain: %q", lines)
		}
	}
	snapshot, err := os.ReadFile(record + ".fd3")
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot) != "{}" {
		t.Fatalf("fd 3 snapshot = %q", snapshot)
	}
	if _, err := DefaultStageRunner(context.Background(), "", "entry.ts", []string{"--list"}); err == nil {
		t.Fatal("empty bun path ran")
	}
}

func TestHashStageRuntime(t *testing.T) {
	write := func(t *testing.T, dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkTree := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		write(t, dir, "a.ts", "export const a = 1;\n")
		write(t, dir, "b.ts", "export const b = 2;\n")
		return dir
	}
	first, err := hashStageRuntime(mkTree(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := hashStageRuntime(mkTree(t))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("identical trees hash differently")
	}
	mutated := mkTree(t)
	write(t, mutated, "b.ts", "export const b = 3;\n")
	third, err := hashStageRuntime(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("changed content kept its hash")
	}
	linked := mkTree(t)
	if err := os.Symlink(filepath.Join(linked, "a.ts"), filepath.Join(linked, "link.ts")); err != nil {
		t.Fatal(err)
	}
	if _, err := hashStageRuntime(linked); err == nil {
		t.Fatal("symlink tree hashed")
	}
	if _, err := hashStageRuntime(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing tree hashed")
	}
}

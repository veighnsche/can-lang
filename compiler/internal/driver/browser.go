package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// BuildTarget builds one project for the named profile. The default Bun
// profile keeps its exact behavior, pairing one verified browser manifest
// when browserManifest names it; the browser profile verifies the same
// assertion roots under Bun, then ships the distinct browser root with its
// content-addressed asset instead of the Bun entry. Browser builds never
// pair: they publish an empty asset set, retiring any prior paired set.
func (r *Runtime) BuildTarget(ctx context.Context, projectDirectory string, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs int, target browser.Target, browserManifest string) (BuildReport, error) {
	switch target {
	case browser.TargetBun:
		return r.Build(ctx, projectDirectory, environment, stdin, stderr, timeoutMs, browserManifest)
	case browser.TargetBrowser:
		if browserManifest != "" {
			return BuildReport{}, fmt.Errorf("browser builds do not pair a browser manifest")
		}
		return r.buildBrowserTarget(ctx, projectDirectory, environment, stdin, stderr, timeoutMs)
	default:
		return BuildReport{}, fmt.Errorf("unknown build target %q", string(target))
	}
}

func (r *Runtime) buildBrowserTarget(ctx context.Context, projectDirectory string, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs int) (BuildReport, error) {
	if r == nil {
		return BuildReport{}, fmt.Errorf("build requires a bundled runtime")
	}
	store, err := BeginOutput(projectDirectory)
	if err != nil {
		return BuildReport{}, err
	}
	defer store.Close()
	return r.buildBrowser(ctx, store, environment, stdin, stderr, timeoutMs)
}

// buildBrowser mirrors the verified pipeline for the browser profile. The
// checked program passes the transitive capability gate before any staging;
// assertion roots still execute supervised under Bun (they never ship in
// the browser asset), and only the audited browser generation is selected
// as production current.
func (r *Runtime) buildBrowser(ctx context.Context, store *OutputStore, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs int) (BuildReport, error) {
	if _, err := CheckAssertTimeoutMs(timeoutMs); err != nil {
		return BuildReport{}, err
	}
	program, err := check.CheckBrowserProgram(store.Graph)
	if err != nil {
		return BuildReport{}, err
	}
	if err := browser.CheckProgram(program); err != nil {
		return BuildReport{}, err
	}
	stableRootOrder(program.Assertions)
	testID, _, err := r.stageProgram(ctx, store, program, true, timeoutMs, nil)
	if err != nil {
		return BuildReport{}, err
	}
	discardTest := true
	defer func() {
		if discardTest {
			_ = store.DiscardGeneration(testID)
		}
	}()
	lease, err := store.AcquireGeneration(testID)
	if err != nil {
		return BuildReport{}, err
	}
	roots := make([]ir.AssertionRoot, 0, len(program.Assertions))
	for _, test := range program.Assertions {
		roots = append(roots, test.Root)
	}
	entries, err := r.RunSupervised(ctx, lease, roots, environment, stdin, timeoutMs, stderr)
	lease.Close()
	if err != nil {
		return BuildReport{}, fmt.Errorf("build verification failed: %w", err)
	}
	summary, err := summarizeBuildRoots(entries)
	if err != nil {
		return BuildReport{}, err
	}
	prodID, prepared, err := r.stageBrowserProgram(ctx, store, program, timeoutMs)
	if err != nil {
		return BuildReport{}, err
	}
	prior, err := store.currentBuildID()
	if err != nil {
		_ = store.DiscardGeneration(prodID)
		return BuildReport{}, err
	}
	directory, err := store.selectCurrentWithAssets(prior, prodID, nil)
	if err != nil {
		return BuildReport{}, err
	}
	discardTest = false
	return BuildReport{SchemaVersion: 1, Kind: "can.build", Target: string(browser.TargetBrowser), BuildID: prepared.BuildID(), Directory: directory, Entry: browser.BrowserEntry, Asset: browser.AssetPath, Inputs: prepared.manifest.Inputs, Assertions: summary, TimeoutMs: timeoutMs, Validation: verifiedBuild}, nil
}

// stageBrowserProgram emits the browser profile, seals source maps, binds
// the content-addressed asset manifest, audits the pre-bundle graph,
// bundles the audited tree with the native pinned bundler, audits the
// published bundle, and validates the generation. It never selects
// production current.
func (r *Runtime) stageBrowserProgram(ctx context.Context, store *OutputStore, program *check.Program, timeoutMs int) (string, *PreparedOutput, error) {
	assets, err := r.PrivateArtifacts()
	if err != nil {
		return "", nil, err
	}
	artifacts, err := emit.BrowserModules(program, assets.Directory, assets.Files)
	if err != nil {
		return "", nil, err
	}
	artifacts, err = r.encodeSourceMaps(ctx, program, artifacts)
	if err != nil {
		return "", nil, err
	}
	table, err := sealBrowserDiagnosticTable(artifacts)
	if err != nil {
		return "", nil, err
	}
	artifacts, err = sealBrowserEntryTable(artifacts, table)
	if err != nil {
		return "", nil, err
	}
	artifacts, err = appendBrowserAsset(artifacts)
	if err != nil {
		return "", nil, err
	}
	if err := browser.AuditArtifacts(artifacts); err != nil {
		return "", nil, err
	}
	launcher, err := regularFile(r.Root, "bin/canlc", true)
	if err != nil {
		return "", nil, err
	}
	var roots []ir.AssertionRoot
	options, _ := json.Marshal(struct {
		Schema     int
		Assertions bool
		Target     string
		Bundle     struct {
			Target    string
			Sourcemap string
			Minify    bool
		}
		Roots     []ir.AssertionRoot
		TimeoutMs int
	}{2, false, string(browser.TargetBrowser), struct {
		Target    string
		Sourcemap string
		Minify    bool
	}{"browser", "external", false}, roots, timeoutMs})
	inputs := store.BuildInputs(hashBytes(launcher), catalogue.SourceHash(), assets.Identity, hashBytes(options))
	bundled, err := r.buildBrowserBundle(ctx, store.Graph, inputs, browserBundlerToolchain(inputs.Compiler, inputs.Runtime), artifacts, table)
	if err != nil {
		return "", nil, err
	}
	artifacts = append(artifacts, bundled...)
	prepared, err := PrepareOutput(inputs, browser.BrowserEntry, artifacts)
	if err != nil {
		return "", nil, err
	}
	if err = r.ValidateOutput(ctx, prepared); err != nil {
		return "", nil, err
	}
	buildID, _, err := store.Stage(prepared)
	if err != nil {
		return "", nil, err
	}
	return buildID, prepared, nil
}

// appendBrowserAsset binds the emitted non-runtime modules into the
// compiler-produced asset manifest. Source maps are sealed before this
// runs, so the asset covers the exact shipped bytes.
func appendBrowserAsset(artifacts []ir.Artifact) ([]ir.Artifact, error) {
	files := map[string]string{}
	for _, artifact := range artifacts {
		if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		sum := sha256.Sum256(artifact.Bytes)
		files[artifact.Path] = hex.EncodeToString(sum[:])
	}
	encoded, err := browser.AssetBytes(files)
	if err != nil {
		return nil, err
	}
	return append(artifacts, ir.Artifact{Path: browser.AssetPath, Bytes: encoded}), nil
}

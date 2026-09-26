package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

type BuildAssertionSummary struct {
	Roots    int      `json:"roots"`
	Passed   int      `json:"passed"`
	Failed   int      `json:"failed"`
	Evidence []string `json:"evidence"`
}

type BuildReport struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	Target        string                `json:"target"`
	BuildID       string                `json:"buildID"`
	Directory     string                `json:"directory"`
	Entry         string                `json:"entry"`
	Asset         string                `json:"asset,omitempty"`
	Browser       *BrowserReport        `json:"browser,omitempty"`
	Inputs        BuildInputs           `json:"inputs"`
	Assertions    BuildAssertionSummary `json:"assertions"`
	TimeoutMs     int                   `json:"timeoutMs"`
	Validation    string                `json:"validation"`
}

// BrowserReportFile is one paired browser byte the server publication
// serves: its browser-logical path, content digest, served route, media
// type, and server-generation artifact path.
type BrowserReportFile struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Route     string `json:"route"`
	MediaType string `json:"mediaType"`
	File      string `json:"file"`
}

// BrowserLockedInstance is one locked dependency instance the paired
// browser manifest pins.
type BrowserLockedInstance struct {
	Instance       string `json:"instance"`
	Lineage        string `json:"lineage"`
	ManifestSHA256 string `json:"manifestSHA256"`
	SourceSHA256   string `json:"sourceSHA256"`
	FixturesSHA256 string `json:"fixturesSHA256"`
}

// BrowserReport records the exact verified browser build a paired server
// build publishes: the browser build ID, the browser generation it was
// verified from, the served entry and table routes, every served digest,
// and the locked snapshot the pairing was bound against.
type BrowserReport struct {
	BrowserBuildID  string                  `json:"browserBuildId"`
	Generation      string                  `json:"generation"`
	ManifestSHA256  string                  `json:"manifestSHA256"`
	Lock            string                  `json:"lock"`
	LockedInstances []BrowserLockedInstance `json:"lockedInstances"`
	Entry           string                  `json:"entry"`
	Table           string                  `json:"table"`
	Files           []BrowserReportFile     `json:"files"`
}

// verifiedBuild is the validation status carried by a successful build
// report: every required root passed on the captured inputs and the
// published production staging passed output validation.
const verifiedBuild = "verified"

// build runs the P15.1 verified pipeline on one captured snapshot: check,
// paired browser verification when --browser-manifest names one, private
// test staging, supervision of every required root in stable order,
// production staging, validation, input revalidation and atomic
// publication. Any failure reports nonzero, discards its temporary
// staging when possible and leaves production current unchanged. The
// production entry point is never executed.
func (r *Runtime) build(ctx context.Context, store *OutputStore, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs, jobs int, browserManifest string) (BuildReport, error) {
	if _, err := CheckAssertTimeoutMs(timeoutMs); err != nil {
		return BuildReport{}, err
	}
	stopCheck := phaseTimer("build.check")
	program, err := check.CheckProgram(store.Graph)
	if err != nil {
		return BuildReport{}, err
	}
	stopCheck()
	var pairing *browserPairing
	if browserManifest != "" {
		pairing, err = verifyBrowserManifest(browserManifest, store.Graph)
		if err != nil {
			return BuildReport{}, err
		}
		program.Assets = append(program.Assets, pairing.assets...)
	}
	stableRootOrder(program.Assertions)
	stopTestStage := phaseTimer("build.test-stage")
	testID, _, err := r.stageProgram(ctx, store, program, true, timeoutMs, pairing)
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
	stopTestStage()
	roots := make([]ir.AssertionRoot, 0, len(program.Assertions))
	for _, test := range program.Assertions {
		roots = append(roots, test.Root)
	}
	stopSupervised := phaseTimer("build.supervised")
	entries, err := r.RunSupervised(ctx, lease, roots, environment, stdin, timeoutMs, stderr, jobs)
	lease.Close()
	if err != nil {
		return BuildReport{}, fmt.Errorf("build verification failed: %w", err)
	}
	stopSupervised()
	summary, err := summarizeBuildRoots(entries)
	if err != nil {
		return BuildReport{}, err
	}
	stopProdStage := phaseTimer("build.prod-stage")
	prodID, prepared, err := r.stageProgram(ctx, store, program, false, timeoutMs, pairing)
	if err != nil {
		return BuildReport{}, err
	}
	if pairing != nil {
		if err := pairing.reread(); err != nil {
			_ = store.DiscardGeneration(prodID)
			return BuildReport{}, err
		}
	}
	stopProdStage()
	stopPublish := phaseTimer("build.publish")
	prior, err := store.currentBuildID()
	if err != nil {
		_ = store.DiscardGeneration(prodID)
		return BuildReport{}, err
	}
	directory, err := store.selectCurrentWithAssets(prior, prodID, pairingFiles(pairing))
	if err != nil {
		return BuildReport{}, err
	}
	discardTest = false
	stopPublish()
	return BuildReport{SchemaVersion: 1, Kind: "can.build", Target: string(browser.TargetBun), BuildID: prepared.BuildID(), Directory: directory, Entry: "entry.ts", Browser: pairing.pairedReport(), Inputs: prepared.manifest.Inputs, Assertions: summary, TimeoutMs: timeoutMs, Validation: verifiedBuild}, nil
}

// pairingFiles projects the verified pairing to its published file set for
// the pending asset publication. Unpaired builds publish an empty set,
// which retires any previously paired set into retention.
func pairingFiles(pairing *browserPairing) []pairedAssetFile {
	if pairing == nil {
		return nil
	}
	return append([]pairedAssetFile{}, pairing.files...)
}

// summarizeBuildRoots requires every supervised root to pass. A single
// failed root fails the build with a bounded list of failing identities;
// an empty suite verifies vacuously.
func summarizeBuildRoots(entries []map[string]any) (BuildAssertionSummary, error) {
	summary := BuildAssertionSummary{Roots: len(entries), Evidence: []string{}}
	seen := map[string]bool{}
	var failed []string
	for _, entry := range entries {
		if passed, _ := entry["passed"].(bool); passed {
			summary.Passed++
		} else {
			summary.Failed++
			if len(failed) < 8 {
				failed = append(failed, buildRootLabel(entry))
			}
		}
		if evidence, ok := entry["evidence"].([]any); ok {
			for _, label := range evidence {
				if name, ok := label.(string); ok && !seen[name] {
					seen[name] = true
					summary.Evidence = append(summary.Evidence, name)
				}
			}
		}
	}
	sort.Strings(summary.Evidence)
	if summary.Failed != 0 {
		message := fmt.Sprintf("build verification failed: %d of %d roots passed; failed: %s", summary.Passed, summary.Roots, strings.Join(failed, ", "))
		if summary.Failed > len(failed) {
			message += fmt.Sprintf(", and %d more", summary.Failed-len(failed))
		}
		return BuildAssertionSummary{}, errors.New(message)
	}
	return summary, nil
}

func buildRootLabel(entry map[string]any) string {
	root, _ := entry["root"].(map[string]any)
	identity := fmt.Sprintf("%v/%v/%v", root["package"], root["declaration"], root["name"])
	detail := ""
	if reason, ok := entry["reason"].(string); ok && reason != "" {
		detail = reason
	}
	if violations, ok := entry["violations"].([]any); ok {
		var names []string
		for _, violation := range violations {
			if name, ok := violation.(string); ok {
				names = append(names, name)
			}
		}
		if len(names) != 0 {
			if detail != "" {
				detail += ": "
			}
			detail += strings.Join(names, ", ")
		}
	}
	if detail != "" {
		return identity + " (" + detail + ")"
	}
	return identity
}

// stageProgram emits, validates, and privately stages checked modules. It
// never selects production current; callers choose SelectCurrent (build)
// or a generation lease (assert). A verified pairing binds the browser
// build into the staged options and carries the served pairing record.
func (r *Runtime) stageProgram(ctx context.Context, store *OutputStore, program *check.Program, assertions bool, timeoutMs int, pairing *browserPairing) (string, *PreparedOutput, error) {
	assets, err := r.PrivateArtifacts()
	if err != nil {
		return "", nil, err
	}
	var emitted *emit.BrowserPairing
	if pairing != nil {
		emitted = &emit.BrowserPairing{BuildID: pairing.buildID, Entry: pairing.entry, Table: pairing.table}
	}
	stopEmit := phaseTimer("stage.emit")
	var artifacts []ir.Artifact
	if assertions {
		artifacts, err = emit.AssertionModulesPaired(program, assets.Directory, assets.Files, emitted)
	} else {
		artifacts, err = emit.ProgramModulesPaired(program, assets.Directory, assets.Files, emitted)
	}
	if err != nil {
		return "", nil, err
	}
	stopEmit()
	stopMaps := phaseTimer("stage.source-maps")
	artifacts, err = r.encodeSourceMaps(ctx, program, artifacts)
	if err != nil {
		return "", nil, err
	}
	if pairing != nil {
		artifacts = append(artifacts, ir.Artifact{Path: "browser/pairing.json", Bytes: append([]byte(nil), pairing.pairingJSON...)})
	}
	stopMaps()
	launcher, err := regularFile(r.Root, "bin/canlc", true)
	if err != nil {
		return "", nil, err
	}
	var roots []ir.AssertionRoot
	if assertions {
		for _, test := range program.Assertions {
			roots = append(roots, test.Root)
		}
	}
	options, _ := json.Marshal(struct {
		Schema     int
		Assertions bool
		Roots      []ir.AssertionRoot
		TimeoutMs  int
		Browser    *browserOptions
	}{1, assertions, roots, timeoutMs, pairingOptions(pairing)})
	inputs := store.BuildInputs(hashBytes(launcher), catalogue.SourceHash(), assets.Identity, hashBytes(options))
	stopPrepare := phaseTimer("stage.prepare")
	prepared, err := PrepareOutput(inputs, "entry.ts", artifacts)
	if err != nil {
		return "", nil, err
	}
	stopPrepare()
	stopValidate := phaseTimer("stage.validate")
	if err = r.ValidateOutput(ctx, prepared); err != nil {
		return "", nil, err
	}
	stopValidate()
	stopStage := phaseTimer("stage.stage")
	buildID, _, err := store.Stage(prepared)
	if err != nil {
		return "", nil, err
	}
	stopStage()
	return buildID, prepared, nil
}

// browserOptions binds the verified pairing into the staged build inputs:
// the browser build ID plus the digest of the exact manifest bytes the
// verification consumed. The manifest path itself never enters the inputs.
type browserOptions struct {
	BuildID  string `json:"buildId"`
	Manifest string `json:"manifest"`
}

func pairingOptions(pairing *browserPairing) *browserOptions {
	if pairing == nil {
		return nil
	}
	return &browserOptions{BuildID: pairing.buildID, Manifest: pairing.manifestHash}
}
func (r *Runtime) Build(ctx context.Context, projectDirectory string, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs, jobs int, browserManifest string) (BuildReport, error) {
	if r == nil {
		return BuildReport{}, fmt.Errorf("build requires a bundled runtime")
	}
	store, err := BeginOutput(projectDirectory)
	if err != nil {
		return BuildReport{}, err
	}
	defer store.Close()
	return r.build(ctx, store, environment, stdin, stderr, timeoutMs, jobs, browserManifest)
}

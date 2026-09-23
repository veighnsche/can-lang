package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

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
	BuildID       string                `json:"buildID"`
	Directory     string                `json:"directory"`
	Entry         string                `json:"entry"`
	Inputs        BuildInputs           `json:"inputs"`
	Assertions    BuildAssertionSummary `json:"assertions"`
	TimeoutMs     int                   `json:"timeoutMs"`
	Validation    string                `json:"validation"`
}

// verifiedBuild is the validation status carried by a successful build
// report: every required root passed on the captured inputs and the
// published production staging passed output validation.
const verifiedBuild = "verified"

// build runs the P15.1 verified pipeline on one captured snapshot: check,
// private test staging, supervision of every required root in stable
// order, production staging, validation, input revalidation and atomic
// publication. Any failure reports nonzero, discards its temporary
// staging when possible and leaves production current unchanged. The
// production entry point is never executed.
func (r *Runtime) build(ctx context.Context, store *OutputStore, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs int) (BuildReport, error) {
	if _, err := CheckAssertTimeoutMs(timeoutMs); err != nil {
		return BuildReport{}, err
	}
	program, err := check.CheckProgram(store.Graph)
	if err != nil {
		return BuildReport{}, err
	}
	stableRootOrder(program.Assertions)
	testID, _, err := r.stageProgram(ctx, store, program, true, timeoutMs)
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
	prodID, prepared, err := r.stageProgram(ctx, store, program, false, timeoutMs)
	if err != nil {
		return BuildReport{}, err
	}
	directory, err := store.SelectCurrent(prodID)
	if err != nil {
		_ = store.DiscardGeneration(prodID)
		return BuildReport{}, err
	}
	discardTest = false
	return BuildReport{SchemaVersion: 1, Kind: "can.build", BuildID: prepared.BuildID(), Directory: directory, Entry: "entry.ts", Inputs: prepared.manifest.Inputs, Assertions: summary, TimeoutMs: timeoutMs, Validation: verifiedBuild}, nil
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
// or a generation lease (assert).
func (r *Runtime) stageProgram(ctx context.Context, store *OutputStore, program *check.Program, assertions bool, timeoutMs int) (string, *PreparedOutput, error) {
	assets, err := r.PrivateArtifacts()
	if err != nil {
		return "", nil, err
	}
	var artifacts []ir.Artifact
	if assertions {
		artifacts, err = emit.AssertionModules(program, assets.Directory, assets.Files)
	} else {
		artifacts, err = emit.ProgramModules(program, assets.Directory, assets.Files)
	}
	if err != nil {
		return "", nil, err
	}
	artifacts, err = r.encodeSourceMaps(ctx, program, artifacts)
	if err != nil {
		return "", nil, err
	}
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
	}{1, assertions, roots, timeoutMs})
	inputs := store.BuildInputs(hashBytes(launcher), catalogue.SourceHash(), assets.Identity, hashBytes(options))
	prepared, err := PrepareOutput(inputs, "entry.ts", artifacts)
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
func (r *Runtime) Build(ctx context.Context, projectDirectory string, environment []string, stdin io.Reader, stderr io.Writer, timeoutMs int) (BuildReport, error) {
	if r == nil {
		return BuildReport{}, fmt.Errorf("build requires a bundled runtime")
	}
	store, err := BeginOutput(projectDirectory)
	if err != nil {
		return BuildReport{}, err
	}
	defer store.Close()
	return r.build(ctx, store, environment, stdin, stderr, timeoutMs)
}

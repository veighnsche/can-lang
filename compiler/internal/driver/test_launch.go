package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
)

// Launch channel bounds from the P19 handoff contract (worker CAN source
// is authoritative for policy; Go owns only the mechanics).
const (
	// launchSourceCap is the exact fd-3 source-frame cap: bytes past it
	// fail the collect, never truncate-then-parse.
	launchSourceCap = 65536
	// launchOutputCap is the exact fd-1 report-frame cap.
	launchOutputCap = 65536
	// launchRowCap is the report row cap enforced after JSON decode.
	launchRowCap = 64
	// launchMinAttempts..launchMaxAttempts is the fixed planned fan-out.
	// Go never retries: attempts are the planned vector, nothing more.
	launchMinAttempts = 1
	launchMaxAttempts = 8
	// launchStderrCap bounds captured worker logs; stderr is diagnostic
	// only, never parsed as verdict.
	launchStderrCap = 65536
)

// LaunchPlan is one (case, variant) fan-out: the planned attempt vector
// the controller joins. Attempts counts 0..Attempts-1.
type LaunchPlan struct {
	CaseID     string
	VariantID  string
	RunID      string
	Generation string
	Attempts   int
	TimeoutMs  int
}

// LaunchKey renders the identity byte string both sides join on:
// `case <c> variant <v> attempt <n> run <r> generation <g>`.
func LaunchKey(caseID, variantID string, attempt int, runID, generation string) string {
	return "case " + caseID + " variant " + variantID + " attempt " + strconv.Itoa(attempt) + " run " + runID + " generation " + generation
}

// WorkerFacts is one attempt's observed native facts. The Can controller
// reduces these to worker_done/crashed/timed_out/rejected; Go never
// synthesizes verdicts. A clean exit with an echoed key and a bounded
// parseable report sets Report; every other path leaves Report nil with
// the discriminating fields set.
type WorkerFacts struct {
	Attempt      int
	Key          string
	TimedOut     bool
	ExitCode     int
	Signal       string
	HandoffError string
	Report       []byte
	RejectReason string
}

// Crashed reports whether the attempt ended without a verdict: a nonzero
// exit, a termination signal, or a spawn/collect failure. Callers map a
// nonempty HandoffError to the handoff_error cause text.
func (facts WorkerFacts) Crashed() bool {
	return facts.HandoffError != "" || facts.Signal != "" || facts.ExitCode != 0
}

// WorkerSpawn is one fresh-worker invocation. Argv is the complete argv
// with the staged entry pinned by the launcher; callers supply plan
// values only, never paths. SourceFrame is the fd-3 bytes: the launch
// key line plus the generation-lease locator.
type WorkerSpawn struct {
	Argv        []string
	SourceFrame []byte
	Timeout     time.Duration
}

// WorkerResult is the raw fd-1 outcome. Stdout is uncapped here; the
// launcher enforces the exact output cap so overflow fails the collect.
type WorkerResult struct {
	Stdout   []byte
	TimedOut bool
	ExitCode int
	Signal   string
	SpawnErr error
}

// WorkerRunner spawns one fresh worker per call. Production shells to bun
// (BunWorkerRunner); tests inject fakes. Implementations must enforce the
// spawn timeout with SIGKILL and report timeout wins at equality.
type WorkerRunner func(ctx context.Context, spawn WorkerSpawn) WorkerResult

// LaunchConfig wires one launcher: the admission host, the pinned staged
// entry and its judge roots, the spawn runner, and the grant economics.
type LaunchConfig struct {
	Host           *admission.Host
	Root           string
	BunPath        string
	EntryPath      string
	StagingRoot    string
	RuntimeRoot    string
	LeaseID        string
	Runner         WorkerRunner
	Budget         time.Duration
	CleanupReserve time.Duration
	Capability     admission.Capability
}

// validateLaunchPlan enforces the argv contract: opaque string identities
// with no path, flag or control characters, non-negative decimal attempt
// and timeout budgets, and the fixed 1..8 fan-out.
func validateLaunchPlan(plan LaunchPlan) error {
	for name, value := range map[string]string{"case": plan.CaseID, "variant": plan.VariantID, "run": plan.RunID, "generation": plan.Generation} {
		if value == "" {
			return fmt.Errorf("driver: launch plan needs a non-empty %s", name)
		}
		if strings.HasPrefix(value, "-") || strings.ContainsAny(value, "/\\\x00\n\r") {
			return fmt.Errorf("driver: launch %s %q must be an opaque identity, not a path or flag", name, value)
		}
	}
	if plan.Attempts < launchMinAttempts || plan.Attempts > launchMaxAttempts {
		return fmt.Errorf("driver: launch attempts %d outside the fixed %d..%d vector", plan.Attempts, launchMinAttempts, launchMaxAttempts)
	}
	if _, err := CheckAssertTimeoutMs(plan.TimeoutMs); err != nil {
		return err
	}
	return nil
}

// checkJudgeRealm refuses a staged manifest whose import closure escapes
// the judge realm: every import must be relative and resolve inside the
// staging or runtime root. Bare, absolute, loader-scheme and escaping
// specifiers fail closed, as does an entry outside the staging root.
func checkJudgeRealm(manifest OutputManifest, entryPath, stagingRoot, runtimeRoot string) error {
	if stagingRoot == "" || runtimeRoot == "" {
		return fmt.Errorf("driver: launch needs staging and runtime roots")
	}
	stagingAbs, err := filepath.Abs(stagingRoot)
	if err != nil {
		return err
	}
	runtimeAbs, err := filepath.Abs(runtimeRoot)
	if err != nil {
		return err
	}
	entryAbs, err := filepath.Abs(entryPath)
	if err != nil {
		return err
	}
	if entryAbs != stagingAbs && !strings.HasPrefix(entryAbs, stagingAbs+string(filepath.Separator)) {
		return fmt.Errorf("driver: launch entry escapes the staged judge realm")
	}
	within := func(abs string) bool {
		return abs == stagingAbs || strings.HasPrefix(abs, stagingAbs+string(filepath.Separator)) ||
			abs == runtimeAbs || strings.HasPrefix(abs, runtimeAbs+string(filepath.Separator))
	}
	for importer, specs := range manifest.Imports {
		base := filepath.Dir(filepath.Join(stagingAbs, filepath.FromSlash(importer)))
		for _, spec := range specs {
			if spec == "" || (!strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../")) {
				return fmt.Errorf("driver: staged import %q from %s is not a relative judge-module path", spec, importer)
			}
			resolved := filepath.Clean(filepath.Join(base, filepath.FromSlash(spec)))
			if !within(resolved) {
				return fmt.Errorf("driver: staged import %q from %s escapes the judge realm", spec, importer)
			}
		}
	}
	return nil
}

// launchArgv renders the worker argv with the entry pinned: bun, the
// staged entry, then the case/variant/attempt/run/generation/timeout
// flags. No caller-supplied module path is honored.
func launchArgv(bunPath, entryPath string, plan LaunchPlan, attempt int) []string {
	return []string{bunPath, entryPath,
		"--case", plan.CaseID,
		"--variant", plan.VariantID,
		"--attempt", strconv.Itoa(attempt),
		"--run", plan.RunID,
		"--generation", plan.Generation,
		"--timeout-ms", strconv.Itoa(plan.TimeoutMs),
	}
}

// collectWorkerFacts reduces one raw worker result to launch facts. The
// precedence is fail-closed: handoff error, timeout (bytes discarded),
// crash (bytes discarded), then the bounded clean-exit collect with
// whole-string key comparison and the post-decode row cap.
func collectWorkerFacts(key string, result WorkerResult) WorkerFacts {
	facts := WorkerFacts{Key: key, ExitCode: result.ExitCode, Signal: result.Signal}
	if result.SpawnErr != nil {
		facts.HandoffError = result.SpawnErr.Error()
		facts.ExitCode = -1
		return facts
	}
	if result.TimedOut {
		facts.TimedOut = true
		return facts
	}
	if result.Signal != "" || result.ExitCode != 0 {
		return facts
	}
	if len(result.Stdout) > launchOutputCap {
		facts.HandoffError = fmt.Sprintf("output frame exceeds %d bytes", launchOutputCap)
		return facts
	}
	frame := result.Stdout
	if len(bytes.TrimSpace(frame)) == 0 {
		facts.RejectReason = "missing-report"
		return facts
	}
	line := frame
	rest := []byte(nil)
	if at := bytes.IndexByte(frame, '\n'); at >= 0 {
		line, rest = frame[:at], frame[at+1:]
	}
	if string(bytes.TrimSuffix(line, []byte("\r"))) != key {
		facts.RejectReason = "identity-drift"
		return facts
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(rest), &envelope); err != nil {
		facts.RejectReason = "missing-report"
		return facts
	}
	raw, ok := envelope["rows"]
	if !ok {
		facts.RejectReason = "missing-report"
		return facts
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		facts.RejectReason = "missing-report"
		return facts
	}
	if len(rows) > launchRowCap {
		facts.RejectReason = "row-cap"
		return facts
	}
	facts.Report = append([]byte(nil), bytes.TrimSpace(rest)...)
	return facts
}

// LaunchWorkers runs one planned fan-out: admit the live-case grant,
// refuse on judge-realm escape, spawn one fresh worker per attempt with
// its fd-3 source frame, and collect every attempt's facts. Worker
// crash, timeout and rejection facts are observations, never aborts:
// the vector stays complete for the Can join. Grant refusal, realm
// escape, allocation failure and controller-context death abort with an
// error and no aggregate.
func LaunchWorkers(ctx context.Context, cfg LaunchConfig, plan LaunchPlan, manifest OutputManifest) ([]WorkerFacts, error) {
	if cfg.Host == nil {
		return nil, fmt.Errorf("driver: launch needs an admission host")
	}
	if cfg.Runner == nil {
		return nil, fmt.Errorf("driver: launch needs a worker runner")
	}
	if cfg.BunPath == "" || cfg.EntryPath == "" {
		return nil, fmt.Errorf("driver: launch needs a bun path and a pinned entry")
	}
	if err := validateLaunchPlan(plan); err != nil {
		return nil, err
	}
	if err := checkJudgeRealm(manifest, cfg.EntryPath, cfg.StagingRoot, cfg.RuntimeRoot); err != nil {
		return nil, err
	}
	body := time.Duration(plan.TimeoutMs) * time.Millisecond * time.Duration(plan.Attempts)
	grant, err := cfg.Host.Admit(cfg.Root, admission.Request{
		Demand:         admission.Demand{Live: true},
		Capability:     cfg.Capability,
		Budget:         cfg.Budget,
		Body:           body,
		CleanupReserve: cfg.CleanupReserve,
	})
	if err != nil {
		return nil, fmt.Errorf("driver: launch admission refused: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			grant.Release()
		}
	}()
	timeout := time.Duration(plan.TimeoutMs) * time.Millisecond
	facts := make([]WorkerFacts, 0, plan.Attempts)
	for attempt := 0; attempt < plan.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("driver: launch interrupted before attempt %d: %w", attempt, err)
		}
		if err := grant.Allocate(); err != nil {
			return nil, fmt.Errorf("driver: launch allocation refused before attempt %d: %w", attempt, err)
		}
		key := LaunchKey(plan.CaseID, plan.VariantID, attempt, plan.RunID, plan.Generation)
		source := key + "\n" + cfg.LeaseID + "\n"
		if len(source) > launchSourceCap {
			return nil, fmt.Errorf("driver: launch source frame exceeds %d bytes", launchSourceCap)
		}
		result := cfg.Runner(ctx, WorkerSpawn{
			Argv:        launchArgv(cfg.BunPath, cfg.EntryPath, plan, attempt),
			SourceFrame: []byte(source),
			Timeout:     timeout,
		})
		entry := collectWorkerFacts(key, result)
		entry.Attempt = attempt
		facts = append(facts, entry)
	}
	if err := grant.Complete(); err != nil {
		return nil, fmt.Errorf("driver: launch completion refused: %w", err)
	}
	completed = true
	return facts, nil
}

// BunWorkerRunner spawns real workers under bun: argv as pinned by the
// launcher, stdin closed, the source frame on fd 3, stdout read to one
// past the output cap so overflow fails the collect instead of
// truncating. The timeout context SIGKILLs on fire and timeout wins at
// equality: a deadline reached with an exit in hand still reports timed
// out. Stderr is captured bounded as diagnostics only.
func BunWorkerRunner(ctx context.Context, spawn WorkerSpawn) WorkerResult {
	if len(spawn.Argv) < 2 {
		return WorkerResult{ExitCode: -1, SpawnErr: fmt.Errorf("driver: worker spawn needs bun and entry argv")}
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, spawn.Timeout)
	defer cancel()
	reader, writer, err := os.Pipe()
	if err != nil {
		return WorkerResult{ExitCode: -1, SpawnErr: err}
	}
	defer reader.Close()
	if _, err := writer.Write(spawn.SourceFrame); err != nil {
		writer.Close()
		return WorkerResult{ExitCode: -1, SpawnErr: err}
	}
	if err := writer.Close(); err != nil {
		return WorkerResult{ExitCode: -1, SpawnErr: err}
	}
	cmd := exec.CommandContext(timeoutCtx, spawn.Argv[0], spawn.Argv[1:]...)
	cmd.ExtraFiles = []*os.File{reader} // first extra file is fd 3
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &stdout, limit: launchOutputCap + 1}
	cmd.Stderr = &limitedWriter{buf: &stderr, limit: launchStderrCap + 1}
	runErr := cmd.Run()
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return WorkerResult{Stdout: stdout.Bytes(), TimedOut: true, ExitCode: -1}
	}
	if ctx.Err() != nil {
		return WorkerResult{ExitCode: -1, SpawnErr: ctx.Err()}
	}
	if runErr == nil {
		return WorkerResult{Stdout: stdout.Bytes()}
	}
	result := WorkerResult{Stdout: stdout.Bytes(), ExitCode: -1}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			result.Signal = status.Signal().String()
		}
		return result
	}
	result.SpawnErr = runErr
	return result
}

// limitedWriter caps one captured stream: writes past the limit are
// dropped, keeping the byte count exact for overflow detection.
type limitedWriter struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	room := w.limit - w.buf.Len()
	if room <= 0 {
		return len(data), nil
	}
	if len(data) > room {
		data = data[:room]
	}
	return w.buf.Write(data)
}

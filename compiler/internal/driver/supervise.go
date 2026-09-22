package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// DefaultAssertTimeoutMs is the P15.1 per-root wall-time budget. The CLI flag
// --assert-timeout-ms overrides it within MinAssertTimeoutMs..MaxAssertTimeoutMs.
const (
	DefaultAssertTimeoutMs = 5000
	MinAssertTimeoutMs     = 1
	MaxAssertTimeoutMs     = 600000
	// reapTimeoutMs caps process reaping after supervisor termination. An
	// unconfirmed worker stops the suite; nothing is published afterward.
	reapTimeoutMs = 1000
	// progressPrefix frames worker progress records on stderr. The driver
	// consumes these lines; every other stderr byte is worker diagnostic.
	progressPrefix = "CAN-PROGRESS "
	// workerDiagnosticLimit caps forwarded worker diagnostics per root.
	workerDiagnosticLimit = 65536
)

// ErrAssertionsFailed reports a delivered suite whose roots did not all
// pass. The CLI maps it to a silent nonzero exit; the suite report on
// stdout carries the per-root outcomes.
var ErrAssertionsFailed = fmt.Errorf("can.assertion-report: one or more assertion roots failed")

// ParseAssertTimeoutMs validates a CLI timeout budget: a decimal integer in
// 1..600000. No zero, negative, noninteger, excessive, or unlimited value
// is accepted.
func ParseAssertTimeoutMs(text string) (int, error) {
	value, err := strconv.Atoi(text)
	if err != nil || text == "" {
		return 0, fmt.Errorf("invalid --assert-timeout-ms %q: want a decimal integer in %d..%d", text, MinAssertTimeoutMs, MaxAssertTimeoutMs)
	}
	return CheckAssertTimeoutMs(value)
}

// CheckAssertTimeoutMs validates an already parsed budget.
func CheckAssertTimeoutMs(value int) (int, error) {
	if value < MinAssertTimeoutMs || value > MaxAssertTimeoutMs {
		return 0, fmt.Errorf("invalid --assert-timeout-ms %d: want a decimal integer in %d..%d", value, MinAssertTimeoutMs, MaxAssertTimeoutMs)
	}
	return value, nil
}

// supervisedRoot is one worker outcome. Entry is the suite-report element;
// delivered reports reuse the worker's assertion payload verbatim.
type supervisedRoot struct {
	entry      map[string]any
	diagnostic string
}

// RunSupervised executes one staged assertion generation root by root, each
// in its own worker with a fresh harness and an external wall-time budget.
// Roots run sequentially in the given stable order. The supervisor's own
// monotonic clock decides every outcome; timeout wins at equality, so a
// worker cannot pass after its budget elapsed. It returns the ordered suite
// entries; isolation failures stop the suite with an error instead.
func (r *Runtime) RunSupervised(ctx context.Context, lease *OutputLease, roots []ir.AssertionRoot, environment []string, stdin io.Reader, timeoutMs int, stderr io.Writer) ([]map[string]any, error) {
	if _, err := CheckAssertTimeoutMs(timeoutMs); err != nil {
		return nil, err
	}
	entry, err := r.validateLease(lease)
	if err != nil {
		return nil, err
	}
	defer lease.Close()
	budget := time.Duration(timeoutMs) * time.Millisecond
	entries := []map[string]any{}
	for index, root := range roots {
		if err := ctx.Err(); err != nil {
			return entries, fmt.Errorf("assertion supervision interrupted: %w", err)
		}
		outcome, err := r.superviseRoot(ctx, lease, entry, environment, stdin, index, root, budget)
		if err != nil {
			return entries, err
		}
		entries = append(entries, outcome.entry)
		if outcome.diagnostic != "" && stderr != nil {
			fmt.Fprintln(stderr, outcome.diagnostic)
		}
	}
	return entries, nil
}

// validateLease repeats RunOutput's generation checks without executing.
func (r *Runtime) validateLease(lease *OutputLease) (string, error) {
	if lease == nil || lease.file == nil {
		return "", fmt.Errorf("run requires an active generation lease")
	}
	if err := r.validateRuntimeManifest(lease.Manifest); err != nil {
		return "", err
	}
	if err := validateOutputManifest(lease.Manifest, true); err != nil {
		return "", err
	}
	if err := noSymlinkAncestors(lease.Directory); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(lease.Directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	current, err := root.Lstat("manifest.json")
	if err != nil {
		return "", err
	}
	locked, err := lease.file.Stat()
	if err != nil || !os.SameFile(current, locked) {
		return "", fmt.Errorf("leased manifest was replaced")
	}
	if err = validateOutputTree(root, ".", lease.Manifest, false, false); err != nil {
		return "", err
	}
	return lease.Directory + "/" + lease.Manifest.Entry, nil
}

func rootLabel(root ir.AssertionRoot) string {
	return root.Package + "/" + root.Declaration + "/" + root.Name
}

func (r *Runtime) superviseRoot(ctx context.Context, lease *OutputLease, entry string, environment []string, stdin io.Reader, index int, root ir.AssertionRoot, budget time.Duration) (supervisedRoot, error) {
	start := time.Now()
	launch, err := r.prepareEntry(ctx, entry, []string{fmt.Sprintf("root=%d", index)}, environment, stdin, &bytes.Buffer{}, &bytes.Buffer{}, []*os.File{lease.file})
	if err != nil {
		return supervisedRoot{}, err
	}
	defer launch.cleanup()
	stdout := launch.cmd.Stdout.(*bytes.Buffer)
	workerStderr := launch.cmd.Stderr.(*bytes.Buffer)
	wrote, err := launch.begin()
	if err != nil {
		return supervisedRoot{}, fmt.Errorf("assertion worker for %s failed to launch: %w", rootLabel(root), err)
	}
	waited := make(chan error, 1)
	go func() { waited <- launch.cmd.Wait() }()
	remaining := budget - time.Since(start)
	if remaining < 0 {
		remaining = 0
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case waitErr := <-waited:
		elapsed := time.Since(start)
		deliveryErr := finishEntry(launch.write, wrote, waitErr)
		return r.deliveredRoot(root, stdout, workerStderr, deliveryErr, waitErr, elapsed, budget)
	case <-timer.C:
		return r.reapRoot(root, launch, wrote, waited, stdout, workerStderr, start, budget)
	case <-ctx.Done():
		_ = launch.cmd.Process.Kill()
		if !awaitExit(waited, reapTimeoutMs) {
			finishEntry(launch.write, wrote, nil)
			return supervisedRoot{}, fmt.Errorf("assertion supervisor isolation failure: worker for %s did not exit within %dms of termination", rootLabel(root), reapTimeoutMs)
		}
		finishEntry(launch.write, wrote, nil)
		return supervisedRoot{}, fmt.Errorf("assertion supervision interrupted: %w", ctx.Err())
	}
}

// deliveredRoot judges a worker that exited on its own. Timeout still wins
// at equality: a pass observed at or after the budget is a late pass, not
// a pass. Delivered reports keep stderr silent; only undelivered workers
// forward bounded diagnostics.
func (r *Runtime) deliveredRoot(root ir.AssertionRoot, stdout, workerStderr *bytes.Buffer, deliveryErr, waitErr error, elapsed, budget time.Duration) (supervisedRoot, error) {
	elapsedMs := elapsed.Milliseconds()
	progress, _ := lastProgress(workerStderr.String())
	if report, ok := parseRootReport(stdout.Bytes()); ok && sameRoot(report.root, root) {
		if elapsed >= budget {
			return supervisedRoot{entry: map[string]any{
				"root": report.root, "passed": false, "reason": "timeout",
				"timeoutMs": budget.Milliseconds(), "elapsedMs": elapsedMs,
				"progress":   progressOrNull(progress),
				"diagnostic": fmt.Sprintf("late result for %s discarded: observed after %dms budget", rootLabel(root), budget.Milliseconds()),
			}}, nil
		}
		entry := report.entry(elapsedMs)
		return supervisedRoot{entry: entry}, nil
	}
	if deliveryErr == nil {
		return supervisedRoot{entry: map[string]any{
			"root": rootMap(root), "passed": false, "reason": "worker protocol failure",
			"elapsedMs": elapsedMs, "progress": progressOrNull(progress),
			"diagnostic": fmt.Sprintf("unusable report for %s: worker exited %s", rootLabel(root), exitText(waitErr)),
		}, diagnostic: workerDiagnostic(root, workerStderr)}, nil
	}
	return supervisedRoot{entry: map[string]any{
		"root": rootMap(root), "passed": false, "reason": "worker crash",
		"elapsedMs": elapsedMs, "progress": progressOrNull(progress),
		"diagnostic": fmt.Sprintf("worker for %s %s", rootLabel(root), exitText(waitErr)),
	}, diagnostic: workerDiagnostic(root, workerStderr)}, nil
}

// reapRoot terminates an over-budget worker outside the worker and confirms
// the reap within reapTimeoutMs. An unconfirmed worker is a supervisor
// isolation failure: it stops the suite instead of risking another root.
func (r *Runtime) reapRoot(root ir.AssertionRoot, launch *entryLaunch, wrote chan error, waited chan error, stdout, workerStderr *bytes.Buffer, start time.Time, budget time.Duration) (supervisedRoot, error) {
	// The deadline fired, so elapsed already meets the budget: any result
	// found afterward is late by construction. Terminate outside the worker,
	// then read the stabilized buffers only after the reap confirms.
	_ = launch.cmd.Process.Kill()
	reaped := awaitExit(waited, reapTimeoutMs)
	finishEntry(launch.write, wrote, nil)
	elapsed := time.Since(start)
	if !reaped {
		return supervisedRoot{}, fmt.Errorf("assertion supervisor isolation failure: worker for %s did not exit within %dms of termination", rootLabel(root), reapTimeoutMs)
	}
	progress, _ := lastProgress(workerStderr.String())
	return supervisedRoot{entry: map[string]any{
		"root": rootMap(root), "passed": false, "reason": "timeout",
		"timeoutMs": budget.Milliseconds(), "elapsedMs": elapsed.Milliseconds(),
		"progress":   progressOrNull(progress),
		"diagnostic": fmt.Sprintf("worker for %s exceeded %dms budget", rootLabel(root), budget.Milliseconds()),
	}, diagnostic: workerDiagnostic(root, workerStderr)}, nil
}

// awaitExit reports whether the child exited within timeoutMs. It never
// kills; the caller owns termination policy.
func awaitExit(waited chan error, timeoutMs int) bool {
	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-waited:
		return true
	case <-timer.C:
		return false
	}
}

func exitText(waitErr error) string {
	if waitErr == nil {
		return "exited 0"
	}
	if exit, ok := waitErr.(*exec.ExitError); ok {
		return fmt.Sprintf("exited %d", exit.ExitCode())
	}
	return waitErr.Error()
}

func rootMap(root ir.AssertionRoot) map[string]any {
	return map[string]any{"package": root.Package, "declaration": root.Declaration, "name": root.Name}
}

func sameRoot(report map[string]any, root ir.AssertionRoot) bool {
	pkg, _ := report["package"].(string)
	declaration, _ := report["declaration"].(string)
	name, _ := report["name"].(string)
	return pkg == root.Package && declaration == root.Declaration && name == root.Name
}

type rootReport struct {
	root   map[string]any
	passed bool
	raw    map[string]any
}

// parseRootReport accepts exactly one can.assertion-root-report object.
func parseRootReport(stdout []byte) (rootReport, bool) {
	var decoded map[string]any
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(stdout)))
	if err := decoder.Decode(&decoded); err != nil {
		return rootReport{}, false
	}
	if decoder.More() {
		return rootReport{}, false
	}
	version, _ := decoded["schemaVersion"].(float64)
	kind, _ := decoded["kind"].(string)
	root, _ := decoded["root"].(map[string]any)
	passed, _ := decoded["passed"].(bool)
	if version != 1 || kind != "can.assertion-root-report" || root == nil || decoded["passed"] == nil {
		return rootReport{}, false
	}
	return rootReport{root: root, passed: passed, raw: decoded}, true
}

// entry reuses the worker's delivered payload verbatim and records the
// supervisor's observed duration alongside it.
func (report rootReport) entry(elapsedMs int64) map[string]any {
	if assertion, ok := report.raw["assertion"].(map[string]any); ok {
		out := map[string]any{}
		for key, value := range assertion {
			out[key] = value
		}
		out["elapsedMs"] = elapsedMs
		return out
	}
	out := map[string]any{}
	for key, value := range report.raw {
		out[key] = value
	}
	delete(out, "kind")
	delete(out, "schemaVersion")
	out["elapsedMs"] = elapsedMs
	return out
}

// lastProgress returns the worker's most recent CAN-PROGRESS record and how
// many arrived. Anything else on stderr is worker diagnostic, not progress.
func lastProgress(workerStderr string) (map[string]any, int) {
	var last map[string]any
	count := 0
	for _, line := range strings.Split(workerStderr, "\n") {
		trimmed := strings.TrimSuffix(line, "\r")
		if !strings.HasPrefix(trimmed, progressPrefix) {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(trimmed, progressPrefix)), &decoded); err != nil {
			continue
		}
		last, count = decoded, count+1
	}
	return last, count
}

func progressOrNull(progress map[string]any) any {
	if progress == nil {
		return nil
	}
	return progress
}

// workerDiagnostic forwards bounded non-progress stderr for undelivered
// roots only. Delivered reports keep stderr silent by contract.
func workerDiagnostic(root ir.AssertionRoot, workerStderr *bytes.Buffer) string {
	var kept []string
	for _, line := range strings.Split(workerStderr.String(), "\n") {
		trimmed := strings.TrimSuffix(line, "\r")
		if trimmed == "" || strings.HasPrefix(trimmed, progressPrefix) {
			continue
		}
		kept = append(kept, trimmed)
	}
	if len(kept) == 0 {
		return ""
	}
	text := strings.Join(kept, "\n")
	if len(text) > workerDiagnosticLimit {
		text = text[:workerDiagnosticLimit] + "\n[truncated]"
	}
	return fmt.Sprintf("worker diagnostic for %s:\n%s", rootLabel(root), text)
}

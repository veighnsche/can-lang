package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// stableRootOrder sorts assertion roots into the P15.1 execution order:
// stable (package, declaration, assertion) identity order. Assert and
// verified build share it so both run the same sequence.
func stableRootOrder(tests []*ir.Assertion) {
	sort.SliceStable(tests, func(i, j int) bool {
		a, b := tests[i].Root, tests[j].Root
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Declaration != b.Declaration {
			return a.Declaration < b.Declaration
		}
		return a.Name < b.Name
	})
}

// Assert checks all declarations before applying the requested root selector.
// Selectors are empty (all), package/name, or package/declaration/name.
// Roots run in stable identity order across at most jobs workers, each in
// its own worker under the timeoutMs wall-time budget. A selected run
// reports its partial scope; assertion execution never publishes
// production output.
func (r *Runtime) Assert(ctx context.Context, directory string, selector, environment []string, stdin io.Reader, stdout, stderr io.Writer, timeoutMs, jobs int) error {
	if r == nil {
		return fmt.Errorf("assertions require a bundled runtime")
	}
	if _, err := CheckAssertTimeoutMs(timeoutMs); err != nil {
		return err
	}
	store, err := BeginOutput(directory)
	if err != nil {
		return err
	}
	defer store.Close()
	program, err := check.CheckAssertionProgram(store.Graph)
	if err != nil {
		return err
	}
	reportWarnings(stderr, program.Warnings)
	scope := "full"
	if len(selector) != 0 {
		if len(selector) != 2 && len(selector) != 3 {
			return fmt.Errorf("assertion selector requires package/name or package/declaration/name")
		}
		var selected []*ir.Assertion
		for _, test := range program.Assertions {
			if test.Root.Package != selector[0] || test.Root.Name != selector[len(selector)-1] {
				continue
			}
			if len(selector) == 3 && test.Root.Declaration != selector[1] {
				continue
			}
			selected = append(selected, test)
		}
		if len(selected) != 1 {
			return fmt.Errorf("assertion selector resolves to %d roots; supply full package/declaration/name identity", len(selected))
		}
		program.Assertions = selected
		scope = "partial"
	}
	stableRootOrder(program.Assertions)
	buildID, _, err := r.stageProgram(ctx, store, program, true, timeoutMs, nil)
	if err != nil {
		return err
	}
	lease, err := store.AcquireGeneration(buildID)
	if err != nil {
		return err
	}
	defer lease.Close()
	if err = store.Close(); err != nil {
		return err
	}
	roots := make([]ir.AssertionRoot, 0, len(program.Assertions))
	for _, test := range program.Assertions {
		roots = append(roots, test.Root)
	}
	entries, err := r.RunSupervised(ctx, lease, roots, environment, stdin, timeoutMs, stderr, jobs)
	complete := err == nil
	passed := complete && len(entries) == len(roots)
	for _, entry := range entries {
		if ok, _ := entry["passed"].(bool); !ok {
			passed = false
		}
	}
	suite := map[string]any{
		"schemaVersion": 1,
		"kind":          "can.assertion-report",
		"passed":        passed,
		"scope":         scope,
		"timeoutMs":     timeoutMs,
		"complete":      complete,
		"assertions":    entries,
	}
	// Initialization is deterministic per worker: when every root fails in
	// setup, the suite also carries the legacy suite-level shape so
	// initialization diagnostics keep their historical location.
	if complete && len(entries) > 0 {
		initialization := true
		for _, entry := range entries {
			if entry["reason"] != "initialization failed" {
				initialization = false
			}
		}
		if initialization {
			suite["reason"] = "initialization failed"
			suite["frames"] = entries[0]["frames"]
		}
	}
	// Survive a closed report pipe: without this the runtime dies on SIGPIPE
	// while delivering the suite, but a broken consumer must stay a plain
	// assertion failure with silent diagnostics, as worker delivery was.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGPIPE)
	encodeErr := json.NewEncoder(stdout).Encode(suite)
	signal.Stop(signals)
	if encodeErr != nil {
		if errors.Is(encodeErr, syscall.EPIPE) {
			return ErrAssertionsFailed
		}
		return encodeErr
	}
	if err != nil {
		return err
	}
	if !passed {
		return ErrAssertionsFailed
	}
	return nil
}

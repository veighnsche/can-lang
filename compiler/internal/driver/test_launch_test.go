package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
)

func launchTestHost(t *testing.T) *admission.Host {
	t.Helper()
	clock := &fakeReuseClock{now: time.Now()}
	disk := &fakeReuseDisk{avail: 20 << 30}
	host, err := admission.OpenHostWithClockAndFreeDisk(t.TempDir(), clock.at, disk.probe)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func launchTestRoots(t *testing.T) (staging, runtime, entry string) {
	t.Helper()
	staging = t.TempDir()
	runtime = t.TempDir()
	entry = filepath.Join(staging, "entry.ts")
	if err := os.WriteFile(entry, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return staging, runtime, entry
}

func launchTestConfig(host *admission.Host, staging, runtime, entry string, runner WorkerRunner) LaunchConfig {
	return LaunchConfig{
		Host:           host,
		Root:           "/repo",
		BunPath:        "/fake/bun",
		EntryPath:      entry,
		StagingRoot:    staging,
		RuntimeRoot:    runtime,
		LeaseID:        "lease-1",
		Runner:         runner,
		Budget:         time.Minute,
		CleanupReserve: time.Second,
		Capability:     admission.Capability{MaxHandles: 64, MaxPending: 16, MaxBytes: 1 << 30},
	}
}

func launchTestPlan() LaunchPlan {
	return LaunchPlan{CaseID: "c1", VariantID: "v1", RunID: "r1", Generation: "g1", Attempts: 2, TimeoutMs: 1000}
}

func launchJudgeManifest() OutputManifest {
	return OutputManifest{Entry: "entry.ts", Imports: map[string][]string{
		"entry.ts":         {"./program/state.ts"},
		"program/state.ts": {"../runtime/r-abc/domain.ts"},
	}}
}

func TestLaunchKeyRendering(t *testing.T) {
	got := LaunchKey("c1", "v1", 3, "r7", "g2")
	want := "case c1 variant v1 attempt 3 run r7 generation g2"
	if got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
}

func TestLaunchWorkersHappyPath(t *testing.T) {
	host := launchTestHost(t)
	staging, runtime, entry := launchTestRoots(t)
	var spawns []WorkerSpawn
	runner := func(_ context.Context, spawn WorkerSpawn) WorkerResult {
		spawns = append(spawns, spawn)
		attempt := len(spawns) - 1
		key := LaunchKey("c1", "v1", attempt, "r1", "g1")
		frame := key + "\n" + `{"case_id":"c1","rows":[{"ok":true}]}` + "\n"
		return WorkerResult{Stdout: []byte(frame)}
	}
	facts, err := LaunchWorkers(context.Background(), launchTestConfig(host, staging, runtime, entry, runner), launchTestPlan(), launchJudgeManifest())
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 2 {
		t.Fatalf("facts = %d, want 2", len(facts))
	}
	for attempt, entry := range facts {
		if entry.Attempt != attempt || entry.TimedOut || entry.Crashed() || entry.RejectReason != "" || entry.Report == nil {
			t.Fatalf("attempt %d facts = %+v", attempt, entry)
		}
		if entry.Key != LaunchKey("c1", "v1", attempt, "r1", "g1") {
			t.Fatalf("attempt %d key = %q", attempt, entry.Key)
		}
	}
	if len(spawns) != 2 {
		t.Fatalf("spawns = %d, want exactly the planned vector (no retry)", len(spawns))
	}
	wantArgv := []string{"/fake/bun", entry, "--case", "c1", "--variant", "v1", "--attempt", "0", "--run", "r1", "--generation", "g1", "--timeout-ms", "1000"}
	got := spawns[0].Argv
	if len(got) != len(wantArgv) {
		t.Fatalf("argv = %q", got)
	}
	for i := range wantArgv {
		if got[i] != wantArgv[i] {
			t.Fatalf("argv = %q, want %q", got, wantArgv)
		}
	}
	if !strings.HasPrefix(string(spawns[0].SourceFrame), LaunchKey("c1", "v1", 0, "r1", "g1")+"\n") {
		t.Fatalf("source frame = %q, want key line first", spawns[0].SourceFrame)
	}
	if spawns[0].Timeout != time.Second {
		t.Fatalf("timeout = %v, want 1s", spawns[0].Timeout)
	}
}

func TestLaunchWorkersCollectPrecedence(t *testing.T) {
	key := LaunchKey("c1", "v1", 0, "r1", "g1")
	report := `{"rows":[]}`
	many := `{"rows":[` + strings.TrimSuffix(strings.Repeat(`{},`, 65), ",") + `]}`
	big := key + "\n" + `{"rows":[` + strings.TrimSuffix(strings.Repeat(`1,`, 40000), ",") + `]}`
	for _, tc := range []struct {
		name   string
		result WorkerResult
		check  func(t *testing.T, facts WorkerFacts)
	}{
		{"drift rejects", WorkerResult{Stdout: []byte("case c1 variant v1 attempt 9 run r1 generation g1\n" + report)}, func(t *testing.T, facts WorkerFacts) {
			if facts.RejectReason != "identity-drift" || facts.Report != nil {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"timeout discards bytes", WorkerResult{Stdout: []byte(key + "\n" + report), TimedOut: true, ExitCode: 0}, func(t *testing.T, facts WorkerFacts) {
			if !facts.TimedOut || facts.Report != nil {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"nonzero discards bytes", WorkerResult{Stdout: []byte(key + "\n" + report), ExitCode: 3}, func(t *testing.T, facts WorkerFacts) {
			if !facts.Crashed() || facts.Report != nil {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"signal discards", WorkerResult{Stdout: []byte(key + "\n" + report), ExitCode: -1, Signal: "killed"}, func(t *testing.T, facts WorkerFacts) {
			if !facts.Crashed() || facts.Report != nil || facts.Signal != "killed" {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"empty is missing report", WorkerResult{}, func(t *testing.T, facts WorkerFacts) {
			if facts.RejectReason != "missing-report" {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"bad json is missing report", WorkerResult{Stdout: []byte(key + "\nnot-json")}, func(t *testing.T, facts WorkerFacts) {
			if facts.RejectReason != "missing-report" {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"row cap rejects", WorkerResult{Stdout: []byte(key + "\n" + many)}, func(t *testing.T, facts WorkerFacts) {
			if facts.RejectReason != "row-cap" || facts.Report != nil {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"output cap fails collect", WorkerResult{Stdout: []byte(big)}, func(t *testing.T, facts WorkerFacts) {
			if facts.HandoffError == "" || facts.Report != nil {
				t.Fatalf("facts = %+v", facts)
			}
		}},
		{"spawn error is handoff", WorkerResult{SpawnErr: fmt.Errorf("spawn-failed")}, func(t *testing.T, facts WorkerFacts) {
			if facts.HandoffError != "spawn-failed" || !facts.Crashed() {
				t.Fatalf("facts = %+v", facts)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, collectWorkerFacts(key, tc.result))
		})
	}
}

func TestLaunchWorkersVectorStaysComplete(t *testing.T) {
	host := launchTestHost(t)
	staging, runtime, entry := launchTestRoots(t)
	calls := 0
	runner := func(_ context.Context, spawn WorkerSpawn) WorkerResult {
		calls++
		if calls == 1 {
			return WorkerResult{ExitCode: 3, Stdout: []byte("boom")}
		}
		if calls == 2 {
			return WorkerResult{TimedOut: true, ExitCode: -1, Stdout: []byte("partial")}
		}
		key := LaunchKey("c1", "v1", 2, "r1", "g1")
		return WorkerResult{Stdout: []byte(key + "\n" + `{"rows":[]}`)}
	}
	plan := launchTestPlan()
	plan.Attempts = 3
	facts, err := LaunchWorkers(context.Background(), launchTestConfig(host, staging, runtime, entry, runner), plan, launchJudgeManifest())
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 3 || calls != 3 {
		t.Fatalf("facts = %d, calls = %d; want the complete vector, no retry, no short-circuit", len(facts), calls)
	}
	if !facts[0].Crashed() || facts[0].Report != nil {
		t.Fatalf("attempt 0 = %+v", facts[0])
	}
	if !facts[1].TimedOut || facts[1].Report != nil {
		t.Fatalf("attempt 1 = %+v", facts[1])
	}
	if facts[2].Report == nil || facts[2].RejectReason != "" {
		t.Fatalf("attempt 2 = %+v", facts[2])
	}
}

func TestLaunchWorkersRefusals(t *testing.T) {
	host := launchTestHost(t)
	staging, runtime, entry := launchTestRoots(t)
	runner := func(_ context.Context, spawn WorkerSpawn) WorkerResult {
		key := LaunchKey("c1", "v1", 0, "r1", "g1")
		return WorkerResult{Stdout: []byte(key + "\n" + `{"rows":[]}`)}
	}
	called := false
	spy := func(ctx context.Context, spawn WorkerSpawn) WorkerResult {
		called = true
		return runner(ctx, spawn)
	}
	newConfig := func() LaunchConfig { return launchTestConfig(host, staging, runtime, entry, spy) }

	// Judge-realm escapes refuse before any spawn.
	for _, tc := range []struct {
		name     string
		manifest OutputManifest
	}{
		{"bare specifier", OutputManifest{Entry: "entry.ts", Imports: map[string][]string{"entry.ts": {"candidate/impl"}}}},
		{"absolute path", OutputManifest{Entry: "entry.ts", Imports: map[string][]string{"entry.ts": {"/tmp/evil.ts"}}}},
		{"escape", OutputManifest{Entry: "entry.ts", Imports: map[string][]string{"entry.ts": {"../../evil.ts"}}}},
	} {
		t.Run("realm/"+tc.name, func(t *testing.T) {
			called = false
			if _, err := LaunchWorkers(context.Background(), newConfig(), launchTestPlan(), tc.manifest); err == nil {
				t.Fatal("accepted judge-realm escape")
			}
			if called {
				t.Fatal("spawned despite realm refusal")
			}
		})
	}

	// Plan violations refuse before any spawn.
	bad := launchTestPlan()
	bad.Attempts = 9
	for _, tc := range []struct {
		name string
		plan LaunchPlan
	}{
		{"zero attempts", LaunchPlan{CaseID: "c", VariantID: "v", RunID: "r", Generation: "g", Attempts: 0, TimeoutMs: 100}},
		{"nine attempts", bad},
		{"empty case", LaunchPlan{VariantID: "v", RunID: "r", Generation: "g", Attempts: 1, TimeoutMs: 100}},
		{"path identity", LaunchPlan{CaseID: "../x", VariantID: "v", RunID: "r", Generation: "g", Attempts: 1, TimeoutMs: 100}},
		{"flag identity", LaunchPlan{CaseID: "c", VariantID: "--evil", RunID: "r", Generation: "g", Attempts: 1, TimeoutMs: 100}},
		{"zero timeout", LaunchPlan{CaseID: "c", VariantID: "v", RunID: "r", Generation: "g", Attempts: 1}},
	} {
		t.Run("plan/"+tc.name, func(t *testing.T) {
			called = false
			if _, err := LaunchWorkers(context.Background(), newConfig(), tc.plan, launchJudgeManifest()); err == nil {
				t.Fatalf("accepted invalid plan: %+v", tc.plan)
			}
			if called {
				t.Fatal("spawned despite plan refusal")
			}
		})
	}

	// A held live lane refuses the grant before any spawn.
	holder, err := host.Admit("/repo", admission.Request{
		Demand:         admission.Demand{Live: true},
		Capability:     admission.Capability{MaxHandles: 1, MaxPending: 1, MaxBytes: 1},
		Budget:         time.Minute,
		Body:           time.Second,
		CleanupReserve: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Release()
	called = false
	if _, err := LaunchWorkers(context.Background(), newConfig(), launchTestPlan(), launchJudgeManifest()); err == nil {
		t.Fatal("launched despite held live lane")
	}
	if called {
		t.Fatal("spawned despite grant refusal")
	}

	// Controller-context death aborts with no aggregate.
	dead, cancel := context.WithCancel(context.Background())
	cancel()
	called = false
	if _, err := LaunchWorkers(dead, newConfig(), launchTestPlan(), launchJudgeManifest()); err == nil {
		t.Fatal("launched on a dead controller context")
	}
	if called {
		t.Fatal("spawned despite controller death")
	}
}

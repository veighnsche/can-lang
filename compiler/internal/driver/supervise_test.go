package driver

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

var supervisorRoot = ir.AssertionRoot{Package: "can.project.root/app", Declaration: "can.project.root/app::answer", Name: "wrong"}

func TestParseAssertTimeoutMs(t *testing.T) {
	for _, valid := range []string{"1", "5000", "600000"} {
		if _, err := ParseAssertTimeoutMs(valid); err != nil {
			t.Fatalf("valid budget %q rejected: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "0", "-1", "600001", "1.5", "5ms", "unlimited", " 5", "5 ", "0x10", "1e3"} {
		if _, err := ParseAssertTimeoutMs(invalid); err == nil {
			t.Fatalf("invalid budget %q accepted", invalid)
		}
	}
	if value, err := ParseAssertTimeoutMs("5000"); err != nil || value != 5000 {
		t.Fatalf("budget misparsed: %d %v", value, err)
	}
}

func TestParseRootReport(t *testing.T) {
	valid := `{"schemaVersion":1,"kind":"can.assertion-root-report","root":{"package":"p","declaration":"d","name":"n"},"passed":true,"assertion":{"root":{},"passed":true}}` + "\n"
	report, ok := parseRootReport([]byte(valid))
	if !ok || !report.passed || report.root["name"] != "n" {
		t.Fatalf("valid root report rejected: %+v", report)
	}
	for _, bad := range []string{
		"",
		"not json",
		`{"schemaVersion":1,"kind":"can.assertion-report","passed":true}`,
		`{"schemaVersion":1,"kind":"can.assertion-root-report","passed":true}`,
		`{"schemaVersion":2,"kind":"can.assertion-root-report","root":{},"passed":true}`,
		`{"schemaVersion":1,"kind":"can.assertion-root-report","root":{}}`,
		valid + valid,
		"log line\n" + valid,
	} {
		if _, ok := parseRootReport([]byte(bad)); ok {
			t.Fatalf("malformed report accepted: %q", bad)
		}
	}
}

func TestRootReportEntryReusesPayload(t *testing.T) {
	raw := []byte(`{"schemaVersion":1,"kind":"can.assertion-root-report","root":{"package":"p","declaration":"d","name":"n"},"passed":false,"assertion":{"root":{"package":"p"},"passed":false,"reason":"outcome mismatch","violations":[]}}`)
	report, ok := parseRootReport(raw)
	if !ok {
		t.Fatal("valid report rejected")
	}
	entry := report.entry(12)
	if entry["reason"] != "outcome mismatch" || entry["passed"] != false || entry["elapsedMs"] != int64(12) {
		t.Fatalf("payload not reused verbatim: %+v", entry)
	}
	if _, exists := entry["kind"]; exists {
		t.Fatalf("protocol envelope leaked: %+v", entry)
	}
}

func TestLastProgress(t *testing.T) {
	progress, count := lastProgress("Bun warning\nCAN-PROGRESS {\"version\":1,\"phase\":\"started\"}\nCAN-PROGRESS not-json\nCAN-PROGRESS {\"version\":1,\"phase\":\"running\",\"pending\":2}\n")
	if count != 2 || progress["phase"] != "running" {
		t.Fatalf("progress misparsed: %+v %d", progress, count)
	}
	if progress, count := lastProgress("no progress here\n"); count != 0 || progress != nil {
		t.Fatalf("absent progress misreported: %+v %d", progress, count)
	}
}

func TestAwaitExit(t *testing.T) {
	exited := make(chan error, 1)
	exited <- nil
	if !awaitExit(exited, 10) {
		t.Fatal("exited child unconfirmed")
	}
	hanging := make(chan error, 1)
	if awaitExit(hanging, 5) {
		t.Fatal("live child confirmed")
	}
}

func TestExternalKillReapsNoncooperatingSpinner(t *testing.T) {
	// Host-level control: a shell busy loop cannot service any in-process
	// timer. Supervisor termination happens outside the worker by signal.
	cmd := exec.Command("/bin/sh", "-c", "while true; do :; done")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	time.Sleep(50 * time.Millisecond)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if !awaitExit(waited, reapTimeoutMs) {
		t.Fatal("killed spinner unreaped")
	}
}

func TestDeliveredRootLatePassIsTimeout(t *testing.T) {
	r := &Runtime{}
	stdout := bytes.NewBufferString(`{"schemaVersion":1,"kind":"can.assertion-root-report","root":{"package":"can.project.root/app","declaration":"can.project.root/app::answer","name":"wrong"},"passed":true,"assertion":{"passed":true}}`)
	outcome, err := r.deliveredRoot(supervisorRoot, stdout, &bytes.Buffer{}, nil, nil, 5000*time.Millisecond, 5000*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	// Timeout wins at equality: a pass observed exactly at the budget is late.
	if outcome.entry["passed"] != false || outcome.entry["reason"] != "timeout" {
		t.Fatalf("late pass admitted: %+v", outcome.entry)
	}
	early, err := r.deliveredRoot(supervisorRoot, bytes.NewBufferString(stdout.String()), &bytes.Buffer{}, nil, nil, 4999*time.Millisecond, 5000*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if early.entry["passed"] != true {
		t.Fatalf("timely pass rejected: %+v", early.entry)
	}
}

func TestDeliveredRootCrashAndProtocol(t *testing.T) {
	r := &Runtime{}
	crashErr := &exec.ExitError{}
	crashed, err := r.deliveredRoot(supervisorRoot, &bytes.Buffer{}, bytes.NewBufferString("boom\n"), crashErr, crashErr, time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if crashed.entry["passed"] != false || crashed.entry["reason"] != "worker crash" {
		t.Fatalf("crash misjudged: %+v", crashed.entry)
	}
	if !strings.Contains(crashed.diagnostic, "boom") {
		t.Fatalf("crash diagnostic lost: %q", crashed.diagnostic)
	}
	protocol, err := r.deliveredRoot(supervisorRoot, bytes.NewBufferString("garbage\n"), &bytes.Buffer{}, nil, nil, time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if protocol.entry["passed"] != false || protocol.entry["reason"] != "worker protocol failure" {
		t.Fatalf("protocol failure misjudged: %+v", protocol.entry)
	}
	if protocol.diagnostic != "" {
		t.Fatalf("silent worker produced diagnostics: %q", protocol.diagnostic)
	}
}

func TestWorkerDiagnosticFiltersProgress(t *testing.T) {
	r := &Runtime{}
	_ = r
	diagnostic := workerDiagnostic(supervisorRoot, bytes.NewBufferString("CAN-PROGRESS {\"a\":1}\n\nreal problem\n"))
	if !strings.Contains(diagnostic, "real problem") || strings.Contains(diagnostic, "CAN-PROGRESS") {
		t.Fatalf("diagnostic misfiltered: %q", diagnostic)
	}
	if workerDiagnostic(supervisorRoot, bytes.NewBufferString("CAN-PROGRESS {\"a\":1}\n")) != "" {
		t.Fatal("progress-only stderr forwarded")
	}
	long := workerDiagnostic(supervisorRoot, bytes.NewBufferString(strings.Repeat("x", workerDiagnosticLimit+10)))
	if !strings.Contains(long, "[truncated]") {
		t.Fatal("diagnostic unbounded")
	}
}

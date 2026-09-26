package failureconventions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOwnerSetupGreen is the B02 extraction leg: the owner-accepting
// tier_for helper is tested through a private fallible-constructor
// factory, with the trap fault specified by kind/message probes and all
// pre-extraction roots retained.
func TestOwnerSetupGreen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "owner-setup")
	report := requireGreen(t, runAssert(t, ctx, bundle, project))
	requireRealCan(t, report)
	const wantRoots = 14
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	t.Logf("owner-setup: %d roots green", len(report.Assertions))
}

// trapLine finds the 1-based line of the factory trap expression.
func trapLine(t *testing.T, project string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(project, "src/handler/handler.can"))
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "1 / 0") {
			return i + 1
		}
	}
	t.Fatal("trap expression not found")
	return 0
}

// TestOwnerNegativeFailsClosed proves unexpected factory rejection fails
// closed with a located standard fault: only the tainted row fails, and
// its frames point at the trap span plus the tainted input site.
func TestOwnerNegativeFailsClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "owner-negative")
	outcome := runAssert(t, ctx, bundle, project)
	if outcome.Status == 0 || outcome.Report == nil || outcome.Report.Passed {
		t.Fatalf("tainted suite unexpectedly passed: status=%d %s", outcome.Status, outcome.Stdout)
	}
	const wantRoots = 15
	if len(outcome.Report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d", wantRoots, len(outcome.Report.Assertions))
	}
	for _, entry := range outcome.Report.Assertions {
		id := outcome.rootID(entry)
		if id != "can.project.root/handler::tier_for:tainted" {
			if !entry.Passed {
				t.Fatalf("unrelated root %s failed: %s", id, entry.Reason)
			}
			continue
		}
		if entry.Passed {
			t.Fatal("tainted row passed with a rejecting factory input")
		}
		if len(entry.Frames) != 2 {
			t.Fatalf("want 2 located frames, got %+v", entry.Frames)
		}
		first, second := entry.Frames[0], entry.Frames[1]
		if first.Line != trapLine(t, project) || first.Operation != "binary" || first.File != "handler/handler.can" {
			t.Fatalf("first frame does not locate the trap: %+v", first)
		}
		if second.File != "handler/handler.can" || second.Operation != "call" {
			t.Fatalf("second frame does not locate the tainted input: %+v", second)
		}
	}
	t.Logf("tainted: fails closed with trap + input frames")
}

// TestOwnerForgeRejected proves owner values cannot be forged in test
// rows: foreign construction fails at check time.
func TestOwnerForgeRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "owner-forge")
	outcome := runAssert(t, ctx, bundle, project)
	if outcome.Status == 0 || outcome.Report != nil {
		t.Fatalf("forged suite unexpectedly passed: status=%d %s", outcome.Status, outcome.Stdout)
	}
	const want = "owner record ids::user_id can only be constructed in its declaring package"
	if !strings.Contains(outcome.Stdout+outcome.Stderr, want) {
		t.Fatalf("want confinement error %q, got: %s%s", want, outcome.Stdout, outcome.Stderr)
	}
	t.Logf("forge: rejected at check")
}

// TestFactoryPrivateToPackage proves the factory is package-private: a
// foreign package cannot reach it.
func TestFactoryPrivateToPackage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "owner-setup")
	sneak := "package sneak\n    provides []\n    uses [handler]\nfn int grab\n    emits []\n    asserts\n        sample: => ok 1\n    ok call handler::fixture_id(1)\n"
	dir := filepath.Join(project, "src/sneak")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sneak.can"), []byte(sneak), 0600); err != nil {
		t.Fatal(err)
	}
	outcome := runAssert(t, ctx, bundle, project)
	if outcome.Status == 0 || outcome.Report != nil {
		t.Fatalf("foreign factory call unexpectedly passed: status=%d %s", outcome.Status, outcome.Stdout)
	}
	if !strings.Contains(outcome.Stdout+outcome.Stderr, "handler::fixture_id is private") {
		t.Fatalf("want factory-visibility error, got: %s%s", outcome.Stdout, outcome.Stderr)
	}
	t.Logf("privacy: foreign factory call rejected: %s", strings.TrimSpace(outcome.Stdout+outcome.Stderr))
}

// retireConstructor applies the constructor-evolution repair to a staged
// copy of owner-setup without touching the extracted helper.
func retireConstructor(t *testing.T, project string, withFactoryArm bool) {
	t.Helper()
	replaceOnce(t, project, "can.errors.json",
		`"active": ["ids::invalid"]`,
		`"active": ["ids::invalid", "ids::retired"]`)
	replaceOnce(t, project, "src/ids/ids.can",
		"provides [user_id, invalid, parse, value]",
		"provides [user_id, invalid, retired, parse, value]")
	replaceOnce(t, project, "src/ids/ids.can",
		`/// The presented raw score is not a member score.
error invalid(int raw)`,
		`/// The presented raw score is not a member score.
error invalid(int raw)
/// Scores above the member bound are retired, not invalid.
error retired(int raw)`)
	replaceOnce(t, project, "src/ids/ids.can",
		`fn user_id parse
    emits [invalid]`,
		`fn user_id parse
    emits [invalid, retired]`)
	replaceOnce(t, project, "src/ids/ids.can",
		`        good: 1 => ok user_id(1)
        bad: 0 => invalid(0)
    match raw > 0
        false => invalid(raw)
        true => ok user_id(raw)`,
		`        good: 1 => ok user_id(1)
        bad: 0 => invalid(0)
        big: 150 => retired(150)
    match raw > 0
        false => invalid(raw)
        true => match raw > 100
            false => ok user_id(raw)
            true => retired(raw)`)
	if withFactoryArm {
		replaceOnce(t, project, "src/handler/handler.can",
			`        ids::invalid => relay call fixture_id(1 / 0)`,
			`        ids::invalid => relay call fixture_id(1 / 0)
        ids::retired => relay call fixture_id(1 / 0)`)
	}
	replaceOnce(t, project, "src/handler/handler.can",
		`fn tier_view handle_tier
    emits [ids::invalid]`,
		`fn tier_view handle_tier
    emits [ids::invalid, ids::retired]`)
	replaceOnce(t, project, "src/handler/handler.can",
		`        bronze: 10 => ok tier_view("bronze", false)
        bad: 0 => ids::invalid(0)`,
		`        bronze: 10 => ok tier_view("bronze", false)
        bad: 0 => ids::invalid(0)
        retired: 150 => ids::retired(150)`)
	replaceOnce(t, project, "src/handler/handler.can",
		`    match call ids::parse(raw)
        ids::invalid
        ok ids::user_id id => relay call tier_for(id)`,
		`    match call ids::parse(raw)
        ids::invalid
        ids::retired
        ok ids::user_id id => relay call tier_for(id)`)
}

// TestFactoryRepairGuided proves constructor evolution is compiler-guided
// and helper-local: without the factory arm the check names the new
// error; with it the suite is green and the extracted helper is
// byte-identical.
func TestFactoryRepairGuided(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)

	partial := stageProject(t, "owner-setup")
	retireConstructor(t, partial, false)
	miss := runAssert(t, ctx, bundle, partial)
	if miss.Status == 0 || miss.Report != nil {
		t.Fatalf("armless repair unexpectedly passed: status=%d %s", miss.Status, miss.Stdout)
	}
	if !strings.Contains(miss.Stdout+miss.Stderr, "missing completion arm for ids::retired") {
		t.Fatalf("want guided error naming retired, got: %s%s", miss.Stdout, miss.Stderr)
	}
	t.Logf("guided: %s", strings.TrimSpace(miss.Stdout+miss.Stderr))

	full := stageProject(t, "owner-setup")
	before := snapshotFiles(t, full)
	retireConstructor(t, full, true)
	dumpSurgery(t, full, "owner-repair")
	report := requireGreen(t, runAssert(t, ctx, bundle, full))
	const wantRoots = 16
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	after := snapshotFiles(t, full)
	helper := "fn tier_view tier_for"
	beforeBody := helperBody(t, before["src/handler/handler.can"], helper)
	afterBody := helperBody(t, after["src/handler/handler.can"], helper)
	if beforeBody != afterBody {
		t.Fatal("extracted helper changed during constructor repair")
	}
	t.Logf("repair: 16 roots green, extracted helper identical")
}

// helperBody extracts one fn block (through the next blank-line-separated
// top-level declaration) for stability comparison.
func helperBody(t *testing.T, source, header string) string {
	t.Helper()
	start := strings.Index(source, header)
	if start < 0 {
		t.Fatalf("missing %q", header)
	}
	rest := source[start:]
	end := strings.Index(rest, "\n/// ")
	if end < 0 {
		end = len(rest)
	}
	return rest[:end]
}

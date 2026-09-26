package failureconventions

import (
	"context"
	"testing"
	"time"
)

// TestRetryConventionGreen is the B01 result-data leg: one shared retry
// helper serves two unrelated failure sets (profiles: two authored
// errors; billing: authored plus catalogue) with oracle-verified
// attempt/sequence contracts, faithful capture, re-raise round-trips,
// wrapper-around-wrapper traces, and standard-fault transparency.
func TestRetryConventionGreen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry")
	report := requireGreen(t, runAssert(t, ctx, bundle, project))
	requireRealCan(t, report)
	const wantRoots = 35
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	t.Logf("retry: %d roots green", len(report.Assertions))
}

// TestRetryFixedGreen is the B01 fixed-bound comparison leg: per-domain
// retry logic with explicit emits bounds over the same two loaders,
// including one layered bound-repetition consumer.
func TestRetryFixedGreen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry-fixed")
	report := requireGreen(t, runAssert(t, ctx, bundle, project))
	requireRealCan(t, report)
	const wantRoots = 15
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	t.Logf("retry-fixed: %d roots green", len(report.Assertions))
}

// TestRetryNoRetryMutantFails proves the FIFO oracle is sensitive
// downward: a helper that returns the first outcome fails exactly the
// two-attempt roots with unused-fixture violations while every other
// root still passes.
func TestRetryNoRetryMutantFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry-negative")
	outcome := runAssert(t, ctx, bundle, project)
	if outcome.Status == 0 || outcome.Report == nil || outcome.Report.Passed {
		t.Fatalf("mutant unexpectedly passed: status=%d %s", outcome.Status, outcome.Stdout)
	}
	wantFail := map[string]bool{
		"can.project.root/billing::fetch_quote:flaky":             true,
		"can.project.root/billing::fetch_quote:down":              true,
		"can.project.root/billing::read_quote:refused":            true,
		"can.project.root/profiles::fetch_profile:flaky":          true,
		"can.project.root/profiles::fetch_profile:down":           true,
		"can.project.root/profiles::fetch_traced:traced_ok":       true,
		"can.project.root/profiles::fetch_traced:traced_down":     true,
		"can.project.root/profiles::read_profile:locked":          true,
		"can.project.root/profiles::read_profile:recover":         true,
		"can.project.root/profiles::read_traced:roundtrip":        true,
		"can.project.root/profiles::read_traced:roundtrip_denied": true,
	}
	for _, entry := range outcome.Report.Assertions {
		id := outcome.rootID(entry)
		if wantFail[id] {
			if entry.Passed {
				t.Fatalf("oracle root %s passed against the no-retry mutant", id)
			}
			if len(entry.Violations) != 1 || entry.Violations[0] != "unused fixture" {
				t.Fatalf("root %s failed without unused-fixture: %s %v", id, entry.Reason, entry.Violations)
			}
			continue
		}
		if !entry.Passed {
			t.Fatalf("non-oracle root %s failed: %s %v", id, entry.Reason, entry.Violations)
		}
	}
	t.Logf("mutant: %d oracle roots fail unused-fixture, rest green", len(wantFail))
}

// TestRetryOverAttemptFails proves the oracle is sensitive upward: three
// attempts against two queued rows fail with missing fixture.
func TestRetryOverAttemptFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry")
	replaceOnce(t, project, "src/profiles/profiles.can",
		"relay call retry::retry(callable load_outcome, key, 2)",
		"relay call retry::retry(callable load_outcome, key, 3)")
	outcome := runAssert(t, ctx, bundle, project)
	if outcome.Status == 0 || outcome.Report == nil || outcome.Report.Passed {
		t.Fatalf("over-attempt unexpectedly passed: status=%d %s", outcome.Status, outcome.Stdout)
	}
	seen := map[string]bool{}
	for _, entry := range outcome.Report.Assertions {
		if entry.Passed {
			continue
		}
		seen[outcome.rootID(entry)] = true
		if len(entry.Violations) != 1 || entry.Violations[0] != "missing fixture" {
			t.Fatalf("root %s failed without missing-fixture: %s %v", outcome.rootID(entry), entry.Reason, entry.Violations)
		}
	}
	// Every profiles consumer root that exhausts retries must observe the
	// third attempt; single-attempt and helper roots stay green.
	for _, id := range []string{
		"can.project.root/profiles::fetch_profile:down",
		"can.project.root/profiles::read_profile:locked",
		"can.project.root/profiles::fetch_traced:traced_down",
		"can.project.root/profiles::read_traced:roundtrip_denied",
	} {
		if !seen[id] {
			t.Fatalf("exhausted root %s did not fail missing-fixture", id)
		}
	}
	t.Logf("over-attempt: %d roots fail missing-fixture", len(seen))
}

// addSuspendedToResultData applies the add-error edit to a staged copy of
// the result-data project: profiles gains a third error plus its adapter
// arm, loader behavior, and roots. It returns the resulting report.
func addSuspendedToResultData(t *testing.T, ctx context.Context, bundle, project string) *assertReport {
	t.Helper()
	replaceOnce(t, project, "can.errors.json",
		`"profiles::forbidden", "profiles::unavailable"`,
		`"profiles::forbidden", "profiles::suspended", "profiles::unavailable"`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		"provides [profile, unavailable, forbidden, load_failure, load, load_outcome, fetch_profile, read_profile, fetch_traced, read_traced, blast_kind]",
		"provides [profile, unavailable, forbidden, suspended, load_failure, load, load_outcome, fetch_profile, read_profile, fetch_traced, read_traced, blast_kind]")
	replaceOnce(t, project, "src/profiles/profiles.can",
		`/// The directory refused the lookup.
error forbidden(str key)`,
		`/// The directory refused the lookup.
error forbidden(str key)
/// The directory holds the key but it is suspended.
error suspended(str key)`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`variant load_failure
    unavailable
    forbidden`,
		`variant load_failure
    unavailable
    forbidden
    suspended`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`fn profile load
    emits [unavailable, forbidden]`,
		`fn profile load
    emits [unavailable, forbidden, suspended]`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        hit: "ann" => ok profile(7, "Ann")
        gone: "bob" => unavailable("bob")
        shut: "root" => forbidden("root")
    match key is "ann"
        false => match key is "root"
            false => unavailable(key)
            true => forbidden(key)
        true => ok profile(7, "Ann")`,
		`        hit: "ann" => ok profile(7, "Ann")
        gone: "bob" => unavailable("bob")
        shut: "root" => forbidden("root")
        paused: "zed" => suspended("zed")
    match key is "ann"
        false => match key is "root"
            false => match key is "zed"
                false => unavailable(key)
                true => suspended(key)
            true => forbidden(key)
        true => ok profile(7, "Ann")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        shut: "root" => ok retry::rejected<load_failure>(forbidden("root"))`,
		`        shut: "root" => ok retry::rejected<load_failure>(forbidden("root"))
        paused: "zed" => ok retry::rejected<load_failure>(suspended("zed"))`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            shut: "root" => forbidden("root")`,
		`            shut: "root" => forbidden("root")
            paused: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        forbidden as failure => do
            load_failure leaf = forbidden(failure.key)
            ok retry::rejected(leaf)`,
		`        forbidden as failure => do
            load_failure leaf = forbidden(failure.key)
            ok retry::rejected(leaf)
        suspended as failure => do
            load_failure leaf = suspended(failure.key)
            ok retry::rejected(leaf)`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        flaky: "ann" => ok retry::completed(profile(7, "Ann"))
        down: "bob" => ok retry::rejected<load_failure>(unavailable("bob"))
        first_try: "ann" => ok retry::completed(profile(7, "Ann"))`,
		`        flaky: "ann" => ok retry::completed(profile(7, "Ann"))
        down: "bob" => ok retry::rejected<load_failure>(unavailable("bob"))
        first_try: "ann" => ok retry::completed(profile(7, "Ann"))
        on_hold: "zed" => ok retry::rejected<load_failure>(suspended("zed"))`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            first_try: "ann" => ok profile(7, "Ann")`,
		`            first_try: "ann" => ok profile(7, "Ann")
            on_hold: "zed" => suspended("zed")
            on_hold: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`    emits [unavailable, forbidden]
    given
        str key
    asserts
        steady: "ann" => ok profile(7, "Ann")
        locked: "root" => forbidden("root")
        recover: "ann" => ok profile(7, "Ann")`,
		`    emits [unavailable, forbidden, suspended]
    given
        str key
    asserts
        steady: "ann" => ok profile(7, "Ann")
        locked: "root" => forbidden("root")
        recover: "ann" => ok profile(7, "Ann")
        shelved: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            locked: "root" => forbidden("root")
            locked: "root" => forbidden("root")`,
		`            locked: "root" => forbidden("root")
            locked: "root" => forbidden("root")
            shelved: "zed" => suspended("zed")
            shelved: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            retry::rejected<load_failure> => do
                load_failure cause = out.reason
                match cause
                    unavailable => unavailable(cause.key)
                    forbidden => forbidden(cause.key)`,
		`            retry::rejected<load_failure> => do
                load_failure cause = out.reason
                match cause
                    unavailable => unavailable(cause.key)
                    forbidden => forbidden(cause.key)
                    suspended => suspended(cause.key)`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`    emits [unavailable, forbidden]
    given
        str key
    asserts
        roundtrip: "ann" => ok profile(7, "Ann")
        roundtrip_denied: "bob" => unavailable("bob")`,
		`    emits [unavailable, forbidden, suspended]
    given
        str key
    asserts
        roundtrip: "ann" => ok profile(7, "Ann")
        roundtrip_denied: "bob" => unavailable("bob")
        shelved_deep: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            roundtrip_denied: "bob" => unavailable("bob")
            roundtrip_denied: "bob" => unavailable("bob")`,
		`            roundtrip_denied: "bob" => unavailable("bob")
            roundtrip_denied: "bob" => unavailable("bob")
            shelved_deep: "zed" => suspended("zed")
            shelved_deep: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            retry::rejected<retry::traced<retry::traced<load_failure>>> => do
                load_failure cause = out.reason.cause.cause
                match cause
                    unavailable => unavailable(cause.key)
                    forbidden => forbidden(cause.key)`,
		`            retry::rejected<retry::traced<retry::traced<load_failure>>> => do
                load_failure cause = out.reason.cause.cause
                match cause
                    unavailable => unavailable(cause.key)
                    forbidden => forbidden(cause.key)
                    suspended => suspended(cause.key)`)
	return requireGreen(t, runAssert(t, ctx, bundle, project))
}

// TestAddErrorIsolationResultData is the W3 edit-and-recheck leg for the
// result-data style: adding profiles::suspended touches only profiles
// sources plus the registry, and unrelated consumers stay byte-identical
// and green.
func TestAddErrorIsolationResultData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry")
	before := snapshotFiles(t, project)
	report := addSuspendedToResultData(t, ctx, bundle, project)
	dumpSurgery(t, project, "result-data")
	after := snapshotFiles(t, project)
	const wantRoots = 40
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	requireUnchanged(t, before, after,
		"can.project.json",
		"src/app/main.can",
		"src/retry/retry.can",
		"src/billing/billing.can",
	)
	t.Logf("add-error result-data: %d roots green, unrelated files identical", len(report.Assertions))
}

// addSuspendedToFixed applies the same add-error edit to a staged copy of
// the fixed-bound project.
func addSuspendedToFixed(t *testing.T, ctx context.Context, bundle, project string) *assertReport {
	t.Helper()
	replaceOnce(t, project, "can.errors.json",
		`"profiles::forbidden", "profiles::unavailable"`,
		`"profiles::forbidden", "profiles::suspended", "profiles::unavailable"`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		"provides [profile, unavailable, forbidden, load, retry_load, read_via_service]",
		"provides [profile, unavailable, forbidden, suspended, load, retry_load, read_via_service]")
	replaceOnce(t, project, "src/profiles/profiles.can",
		`/// The directory refused the lookup.
error forbidden(str key)`,
		`/// The directory refused the lookup.
error forbidden(str key)
/// The directory holds the key but it is suspended.
error suspended(str key)`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`fn profile load
    emits [unavailable, forbidden]`,
		`fn profile load
    emits [unavailable, forbidden, suspended]`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        hit: "ann" => ok profile(7, "Ann")
        gone: "bob" => unavailable("bob")
        shut: "root" => forbidden("root")
    match key is "ann"
        false => match key is "root"
            false => unavailable(key)
            true => forbidden(key)
        true => ok profile(7, "Ann")`,
		`        hit: "ann" => ok profile(7, "Ann")
        gone: "bob" => unavailable("bob")
        shut: "root" => forbidden("root")
        paused: "zed" => suspended("zed")
    match key is "ann"
        false => match key is "root"
            false => match key is "zed"
                false => unavailable(key)
                true => suspended(key)
            true => forbidden(key)
        true => ok profile(7, "Ann")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`fn profile retry_load
    emits [unavailable, forbidden]`,
		`fn profile retry_load
    emits [unavailable, forbidden, suspended]`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        flaky: "ann", 2 => ok profile(7, "Ann")
        down: "bob", 2 => unavailable("bob")
        first_try: "ann", 2 => ok profile(7, "Ann")`,
		`        flaky: "ann", 2 => ok profile(7, "Ann")
        down: "bob", 2 => unavailable("bob")
        first_try: "ann", 2 => ok profile(7, "Ann")
        paused: "zed", 2 => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            first_try: "ann" => ok profile(7, "Ann")`,
		`            first_try: "ann" => ok profile(7, "Ann")
            paused: "zed" => suspended("zed")
            paused: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        forbidden as failure => match remaining > 1
            false => forbidden(failure.key)
            true => relay call retry_load(key, remaining - 1)`,
		`        forbidden as failure => match remaining > 1
            false => forbidden(failure.key)
            true => relay call retry_load(key, remaining - 1)
        suspended as failure => match remaining > 1
            false => suspended(failure.key)
            true => relay call retry_load(key, remaining - 1)`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`fn profile read_via_service
    emits [unavailable, forbidden]`,
		`fn profile read_via_service
    emits [unavailable, forbidden, suspended]`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`        steady: "ann" => ok profile(7, "Ann")
        locked: "root" => forbidden("root")`,
		`        steady: "ann" => ok profile(7, "Ann")
        locked: "root" => forbidden("root")
        shelved: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`            locked: "root" => forbidden("root")
            locked: "root" => forbidden("root")`,
		`            locked: "root" => forbidden("root")
            locked: "root" => forbidden("root")
            shelved: "zed" => suspended("zed")
            shelved: "zed" => suspended("zed")`)
	replaceOnce(t, project, "src/profiles/profiles.can",
		`    match call retry_load(key, 2)
        unavailable
        forbidden
        ok profile found => ok found`,
		`    match call retry_load(key, 2)
        unavailable
        forbidden
        suspended
        ok profile found => ok found`)
	return requireGreen(t, runAssert(t, ctx, bundle, project))
}

// TestAddErrorIsolationFixed is the W3 edit-and-recheck leg for the
// fixed-bound style: the same edit size comparison input as the
// result-data leg.
func TestAddErrorIsolationFixed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry-fixed")
	before := snapshotFiles(t, project)
	report := addSuspendedToFixed(t, ctx, bundle, project)
	dumpSurgery(t, project, "fixed")
	after := snapshotFiles(t, project)
	const wantRoots = 18
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	requireUnchanged(t, before, after,
		"can.project.json",
		"src/app/main.can",
		"src/billing/billing.can",
	)
	t.Logf("add-error fixed: %d roots green, unrelated files identical", len(report.Assertions))
}

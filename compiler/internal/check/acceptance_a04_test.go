package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const tailCountdownDecl = `fn int countdown
    emits []
    given
        int n
    asserts
        sample: 3 => ok 0
    match n
        0 => ok 0
        _ => relay call countdown(n - 1)
`

func tailRegion(t *testing.T, program *Program, name string) *ir.Region {
	t.Helper()
	for _, fn := range program.Functions {
		if fn.Symbol.Name == name {
			return fn.Region
		}
	}
	t.Fatalf("missing concrete %s region", name)
	return nil
}

func tailRelays(region *ir.Region) []*ir.Completion {
	var out []*ir.Completion
	for _, relay := range scanTailFrame(region).relays {
		out = append(out, relay.completion)
	}
	return out
}

// A04 positive: a direct self relay in a clean frame lowers with no note.
func TestSelfTailPositive(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/main.can": programHeader + tailCountdownDecl + programMain + "    ok\n"})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "countdown")
	if region.TailExclusion != "" {
		t.Fatalf("clean frame excluded: %s", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || !relays[0].SelfTail {
		t.Fatalf("self relay not marked: %+v", relays)
	}
	if len(program.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", program.Warnings)
	}
}

// A04 positive: relays in several match arms lower together.
func TestSelfTailSeveralArms(t *testing.T) {
	text := programHeader + `fn int parity
    emits []
    given
        int n
    asserts
        sample: 4 => ok 0
    match n
        0 => ok 0
        1 => relay call parity(0)
        _ => relay call parity(n - 2)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	relays := tailRelays(tailRegion(t, program, "parity"))
	if len(relays) != 2 || !relays[0].SelfTail || !relays[1].SelfTail {
		t.Fatalf("arm relays not marked: %+v", relays)
	}
	if len(program.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", program.Warnings)
	}
}

// A04 negative: mutual recursion never lowers; each relay is noted.
func TestSelfTailMutual(t *testing.T) {
	text := programHeader + `fn int ping
    emits []
    given
        int n
    asserts
        sample: 0 => ok 0
    match n
        0 => ok 0
        _ => relay call pong(n)
fn int pong
    emits []
    given
        int n
    asserts
        sample: 0 => ok 0
    match n
        0 => ok 0
        _ => relay call ping(n)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ping", "pong"} {
		for _, relay := range tailRelays(tailRegion(t, program, name)) {
			if relay.SelfTail {
				t.Fatalf("mutual relay lowered in %s", name)
			}
		}
	}
	if len(program.Warnings) != 2 {
		t.Fatalf("expected two mutual notes, got %+v", program.Warnings)
	}
	for _, warning := range program.Warnings {
		if warning.Code != "CAN-CHECK-NOT-LOWERED" || warning.Severity != SeverityNote || !strings.Contains(warning.Message, "not the enclosing function") {
			t.Fatalf("wrong mutual note: %+v", warning)
		}
	}
}

// A04 negative: an acyclic relay to another function is not recursion:
// it keeps nested calls and stays silent.
func TestSelfTailNonSelfSilent(t *testing.T) {
	text := programHeader + `fn int helper
    emits []
    given
        int n
    asserts
        sample: 1 => ok 1
    ok n
fn int top
    emits []
    given
        int n
    asserts
        sample: 1 => ok 1
    relay call helper(n)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	for _, relay := range tailRelays(tailRegion(t, program, "top")) {
		if relay.SelfTail {
			t.Fatal("non-self relay lowered")
		}
	}
	if len(program.Warnings) != 0 {
		t.Fatalf("acyclic relay noted: %+v", program.Warnings)
	}
}

// A04 negative: value-producing post-processing is not a relay and never
// lowers; behavior is unchanged and no note fires.
func TestSelfTailPostProcessing(t *testing.T) {
	text := programHeader + `fn int triangle
    emits []
    given
        int n
    asserts
        sample: 3 => ok 6
    match n
        0 => ok 0
        _ => ok call triangle(n - 1) + n
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if relays := tailRelays(tailRegion(t, program, "triangle")); len(relays) != 0 {
		t.Fatalf("nested call mistaken for relay: %+v", relays)
	}
	if len(program.Warnings) != 0 {
		t.Fatalf("nested recursion noted: %+v", program.Warnings)
	}
}

// A04 negative: a frame-created callable defers completion past the relay.
func TestSelfTailCallableExcluded(t *testing.T) {
	text := programHeader + `fn int identity
    emits []
    given
        int value
    asserts
        sample: 4 => ok 4
    ok value
fn int loop
    emits []
    given
        int n
    asserts
        sample: 3 => ok 0
    callable int (int) emits [] action = callable identity
    match n
        0 => ok call action(0)
        _ => relay call loop(n - 1)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "loop")
	if region.TailExclusion != "deferred completion (callable value)" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("callable frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "deferred completion (callable value)")
}

// A04 negative: fixture tables pin invocation frames, so regions holding
// them keep nested calls.
func TestSelfTailFixtureExcluded(t *testing.T) {
	text := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + `fn int loop
    emits [codec::invalid_data]
    given
        int n
    asserts
        sample: 3 => ok 0
    match call number()
        when
            sample: => ok 1
        codec::invalid_data => ok 0
        ok int value => relay call loop(value)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "loop")
	if region.TailExclusion != "fixture table" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("fixture frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "fixture table")
}

// A04 negative: a scheduled timer callback outlives the relay.
func TestSelfTailTimerExcluded(t *testing.T) {
	text := `package app
    provides []
    uses [browser]
fn void on_tick
    emits []
    asserts
        sample: => ok
    ok
fn int ticked
    emits [browser::missing_root, browser::disposed, browser::rejected]
    given
        int n
    asserts
        sample: 0 => ok 0
    match call browser::mount("app")
        browser::missing_root => ok 0
        ok browser::app app => match call browser::open_view(app)
            browser::disposed => ok 0
            ok browser::view view => match call browser::set_timeout(view, 250, callable on_tick)
                browser::disposed => ok 0
                browser::rejected => ok 0
                ok => relay call ticked(n)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "ticked")
	if region.TailExclusion != "pending timer" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("timer frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "pending timer")
}

// A04 negative: transaction work stays open across the relay.
func TestSelfTailLeaseExcluded(t *testing.T) {
	text := `package app
    provides []
    uses [sql, http]
fn sql::decision<int> decide
    emits []
    given
        sql::transaction tx
    asserts
        sample: => ok sql::commit<int>(1)
    ok sql::commit<int>(1)
fn int run
    emits [http::credentials_missing, sql::connection_failed, sql::transaction_failed, sql::commit_unknown]
    asserts
        sample: => ok 1
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        http::credentials_missing => ok 0
        sql::connection_failed => ok 0
        ok sql::pool pool => match call sql::with_transaction<int>(pool, callable decide)
            sql::connection_failed => ok 0
            sql::transaction_failed => ok 0
            sql::commit_unknown => ok 0
            ok int total => relay call run()
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "run")
	if region.TailExclusion != "live lease (transaction)" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("lease frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "live lease (transaction)")
}

// A04 negative: an s3 stream reader needs its drain past the relay.
func TestSelfTailDrainExcluded(t *testing.T) {
	text := `package app
    provides []
    uses [s3, stream, bytes]
fn int streamed
    emits [s3::invalid_config, s3::missing_key, s3::access_denied, s3::service_error, s3::over_limit]
    given
        int n
    asserts
        sample: 0 => ok 0
    match call s3::client_open("http://127.0.0.1:9", "us-east-1", "b", "a", "s")
        s3::invalid_config => ok 0
        ok s3::client handle => match call s3::read_stream(handle, "doc.txt", 65536)
            s3::invalid_config => ok 0
            s3::missing_key => ok 0
            s3::access_denied => ok 0
            s3::service_error => ok 0
            s3::over_limit => ok 0
            ok stream::reader<bytes::buffer> reader => relay call streamed(n)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "streamed")
	if region.TailExclusion != "drain-owned value (s3 stream)" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("drain frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "drain-owned value (s3 stream)")
}

// A04 negative: coordination defers arm completions past participant launch.
func TestSelfTailCoordinationExcluded(t *testing.T) {
	text := strings.Replace(programHeader, "uses []", "uses [codec]", 1) + coordinationDeclarations + `variant failure
    codec::invalid_data
    standard_failure
fn int loop
    emits [all_failed<failure>]
    given
        int n
    asserts
        sample: 3 => ok 0
    int result = match call race
        number()
        number()
        all_failed
        ok int value => ok value
    relay call loop(result)
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	region := tailRegion(t, program, "loop")
	if region.TailExclusion != "deferred completion (coordination)" {
		t.Fatalf("wrong exclusion: %q", region.TailExclusion)
	}
	relays := tailRelays(region)
	if len(relays) != 1 || relays[0].SelfTail {
		t.Fatalf("coordination frame relay lowered: %+v", relays)
	}
	expectTailNote(t, program, "deferred completion (coordination)")
}

func expectTailNote(t *testing.T, program *Program, reason string) {
	t.Helper()
	if len(program.Warnings) != 1 {
		t.Fatalf("expected one tail note, got %+v", program.Warnings)
	}
	warning := program.Warnings[0]
	if warning.Code != "CAN-CHECK-NOT-LOWERED" || warning.Severity != SeverityNote {
		t.Fatalf("wrong note identity: %+v", warning)
	}
	if !strings.Contains(warning.Message, "recursive relay not lowered") || !strings.Contains(warning.Message, reason) {
		t.Fatalf("wrong note message: %+v", warning)
	}
	if !strings.HasSuffix(warning.File, "src/main.can") || warning.Line < 1 || warning.Column < 1 {
		t.Fatalf("note lost its position: %+v", warning)
	}
	_ = source.Span{}
}

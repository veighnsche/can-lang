package check

import (
	"strings"
	"testing"
)

// late fixtures bind one unused str probe so single-value asserts have
// arity; call-site literals mirror into when arms per the c_test
// precedent, and dynamic refusals mirror the probe reference.

func lateSelectFixture(identity string) string {
	return "package app\n    provides []\n    uses [late, test]\nfn late::select_facts choose\n    emits {test::stale_handle, late::late_fault}\n    given\n        test::owner o\n        str probe\n    asserts\n        sample: \"late.alpha\" => ok late::select_facts(\"late.alpha\", false)\n    match call late::select(o, " + identity + ")\n        when\n            sample: o, " + identity + " => ok late::select_facts(\"late.alpha\", false)\n        test::stale_handle\n        late::late_fault\n        ok late::select_facts got => ok got\n" + programMain + "    ok\n"
}

func TestLateSelectIdentityLiteral(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": lateSelectFixture("\"late.alpha\"")}); err != nil {
		t.Fatalf("rejected literal late.alpha identity: %v", err)
	}
}

func TestLateSelectIdentityRefusals(t *testing.T) {
	for _, tc := range []struct{ name, identity, want string }{
		{"unknown identity", "\"late.gamma\"", "is not admitted"},
		{"empty identity", "\"\"", "is not admitted"},
		{"dynamic identity", "probe", "must be a static literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": lateSelectFixture(tc.identity)})
			if err == nil {
				t.Fatalf("accepted invalid identity: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func TestLateParticipantAndKindLiterals(t *testing.T) {
	enroll := "package app\n    provides []\n    uses [late, test]\nfn late::enroll_facts join\n    emits {test::stale_handle, late::late_fault}\n    given\n        test::owner o\n        str probe\n    asserts\n        sample: \"observer\" => ok late::enroll_facts(\"observer\", \"late.beta\")\n    match call late::enroll(o, \"observer\")\n        when\n            sample: o, \"observer\" => ok late::enroll_facts(\"observer\", \"late.beta\")\n        test::stale_handle\n        late::late_fault\n        ok late::enroll_facts got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": enroll}); err != nil {
		t.Fatalf("rejected literal observer participant: %v", err)
	}
	badEnroll := strings.Replace(enroll, "match call late::enroll(o, \"observer\")", "match call late::enroll(o, \"stranger\")", 1)
	badEnroll = strings.Replace(badEnroll, "sample: o, \"observer\" =>", "sample: o, \"stranger\" =>", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": badEnroll}); err == nil {
		t.Fatal("accepted unknown participant")
	} else if !strings.Contains(err.Error(), "is not admitted") {
		t.Fatalf("wrong participant refusal: %v", err)
	}
	emit := "package app\n    provides []\n    uses [late, test]\nfn late::event_facts admit\n    emits {test::stale_handle, late::late_fault}\n    given\n        test::owner o\n        int seq\n    asserts\n        sample: 1 => ok late::event_facts(\"worker\", \"late.alpha\", \"use\", 1, false, [])\n    match call late::emit(o, \"worker\", \"use\", seq)\n        when\n            sample: o, \"worker\", \"use\", 1 => ok late::event_facts(\"worker\", \"late.alpha\", \"use\", 1, false, [])\n        test::stale_handle\n        late::late_fault\n        ok late::event_facts got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": emit}); err != nil {
		t.Fatalf("rejected literal participant+kind: %v", err)
	}
	badEmit := strings.Replace(emit, "\"worker\", \"use\", seq)", "\"worker\", \"using\", seq)", 1)
	badEmit = strings.Replace(badEmit, "o, \"worker\", \"use\", 1", "o, \"worker\", \"using\", 1", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": badEmit}); err == nil {
		t.Fatal("accepted unknown event kind")
	} else if !strings.Contains(err.Error(), "is not admitted") {
		t.Fatalf("wrong kind refusal: %v", err)
	}
}

func TestLateTerminalLiteralAndDynamicLease(t *testing.T) {
	witness := "package app\n    provides []\n    uses [late, test]\nfn late::terminal_facts seal\n    emits {test::stale_handle, late::late_fault}\n    given\n        test::owner o\n        str probe\n    asserts\n        sample: \"failed\" => ok late::terminal_facts(\"observer\", \"late.beta\", \"failed\", \"sha256:abc\")\n    match call late::witness_terminal(o, \"observer\", \"failed\")\n        when\n            sample: o, \"observer\", \"failed\" => ok late::terminal_facts(\"observer\", \"late.beta\", \"failed\", \"sha256:abc\")\n        test::stale_handle\n        late::late_fault\n        ok late::terminal_facts got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": witness}); err != nil {
		t.Fatalf("rejected literal participant+terminal: %v", err)
	}
	lease := "package app\n    provides []\n    uses [late, test]\nfn late::lease_facts read\n    emits {test::stale_handle, late::late_fault}\n    given\n        test::owner o\n        str lease\n    asserts\n        sample: \"llease1\" => ok late::lease_facts(\"llease1\", \"worker\", \"late.alpha\", 1, false)\n    match call late::observe_lease(o, \"worker\", lease)\n        when\n            sample: o, \"worker\", \"llease1\" => ok late::lease_facts(\"llease1\", \"worker\", \"late.alpha\", 1, false)\n        test::stale_handle\n        late::late_fault\n        ok late::lease_facts got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": lease}); err != nil {
		t.Fatalf("rejected dynamic lease id: %v", err)
	}
}

func TestLateStaticTablesComplete(t *testing.T) {
	if len(lateStaticArity) != 10 {
		t.Fatalf("late arity pins %d operations, want 10", len(lateStaticArity))
	}
	for identity, arity := range lateStaticArity {
		words, ok := lateStaticWords[identity]
		if !ok {
			t.Fatalf("%s lacks a static-word entry", identity)
		}
		for index, vocab := range words {
			if index < 1 || index >= arity {
				t.Fatalf("%s pins out-of-range arg %d (arity %d)", identity, index, arity)
			}
			if lateVocabularies[vocab] == nil {
				t.Fatalf("%s pins unknown vocabulary %q", identity, vocab)
			}
		}
	}
	for identity := range lateStaticWords {
		if lateStaticArity[identity] == 0 {
			t.Fatalf("%s lacks an arity entry", identity)
		}
	}
	for _, identity := range []string{lateKillWorker, lateReadOutcome, lateReadCounters} {
		if lateStaticOperation(identity) {
			t.Fatalf("%s is gated but carries no vocabulary", identity)
		}
	}
}

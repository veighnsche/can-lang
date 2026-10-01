package check

import (
	"strings"
	"testing"
)

// dispatch, record_driver_settlement, record_server_ack,
// acquire_cancel_grant and quiesce_engine extend the static db
// operations: the engine must be a K24 word (reused set), the
// driver outcome must be completed or timed-out (no "unknown":
// the driver must report what it saw), and the server effect must
// be applied, absent or unknown. Works, statements, tokens and
// grants stay dynamic.
func dbDispatchFixture(engine string) string {
	stub := "db::dispatched_work(db::dispatch_record(\"work-0\", \"postgres\", \"ns-alpha\", \"sha256:stub\", \"sha256:stub\"), \"tok-stub\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::dispatched_work send\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str work\n        str statement\n    asserts\n        first: \"work-0\", \"SELECT 1\" => ok " + stub + "\n    match call db::dispatch(o, work, statement, " + engine + ")\n        when\n            first: o, work, statement, " + engine + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::dispatched_work got => ok got\n" + programMain + "    ok\n"
}

func dbDriverSettlementFixture(outcome string) string {
	stub := "db::driver_settlement_facts(\"work-0\", \"postgres\", \"completed\", \"unknown\", \"sha256:stub\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::driver_settlement_facts settle\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str work\n        str token\n    asserts\n        first: \"work-0\", \"tok-stub\" => ok " + stub + "\n    match call db::record_driver_settlement(o, work, token, " + outcome + ")\n        when\n            first: o, work, token, " + outcome + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::driver_settlement_facts got => ok got\n" + programMain + "    ok\n"
}

func dbServerAckFixture(effect string) string {
	stub := "db::server_ack_facts(\"work-0\", \"postgres\", \"applied\", \"sha256:stub\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::server_ack_facts ack\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str work\n        str token\n    asserts\n        first: \"work-0\", \"tok-stub\" => ok " + stub + "\n    match call db::record_server_ack(o, work, token, " + effect + ")\n        when\n            first: o, work, token, " + effect + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::server_ack_facts got => ok got\n" + programMain + "    ok\n"
}

func dbCancelGrantFixture(engine string) string {
	stub := "db::cancel_grant(db::cancel_grant_facts(\"postgres\", \"sha256:stub\"), \"grant-stub\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::cancel_grant acquire\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n    asserts\n        first: => ok " + stub + "\n    match call db::acquire_cancel_grant(o, " + engine + ")\n        when\n            first: o, " + engine + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::cancel_grant got => ok got\n" + programMain + "    ok\n"
}

func dbQuiesceEngineFixture(engine string) string {
	return "package app\n    provides []\n    uses [db, test]\nfn str[] rest\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n    asserts\n        first: => ok [\"work-0\"]\n    match call db::quiesce_engine(o, " + engine + ")\n        when\n            first: o, " + engine + " => ok [\"work-0\"]\n        test::stale_handle\n        db::db_fault\n        ok str[] got => ok got\n" + programMain + "    ok\n"
}

func TestDbDeadlineWordLiterals(t *testing.T) {
	for _, engine := range []string{"\"postgres\"", "\"sqlite\"", "\"mysql\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbDispatchFixture(engine)}); err != nil {
			t.Fatalf("rejected literal dispatch engine %s: %v", engine, err)
		}
		if _, err := programFixture(t, map[string]string{"src/main.can": dbCancelGrantFixture(engine)}); err != nil {
			t.Fatalf("rejected literal grant engine %s: %v", engine, err)
		}
		if _, err := programFixture(t, map[string]string{"src/main.can": dbQuiesceEngineFixture(engine)}); err != nil {
			t.Fatalf("rejected literal quiesce engine %s: %v", engine, err)
		}
	}
	for _, outcome := range []string{"\"completed\"", "\"timed-out\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbDriverSettlementFixture(outcome)}); err != nil {
			t.Fatalf("rejected literal driver outcome %s: %v", outcome, err)
		}
	}
	for _, effect := range []string{"\"applied\"", "\"absent\"", "\"unknown\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbServerAckFixture(effect)}); err != nil {
			t.Fatalf("rejected literal server effect %s: %v", effect, err)
		}
	}
}

func TestDbDeadlineRefusals(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture func(string) string
		word    string
		want    string
	}{
		{"misspelled dispatch engine", dbDispatchFixture, "\"oracle\"", "is not admitted"},
		{"dynamic dispatch engine", dbDispatchFixture, "work", "must be a static literal"},
		{"unknown is not a driver outcome", dbDriverSettlementFixture, "\"unknown\"", "is not admitted"},
		{"misspelled driver outcome", dbDriverSettlementFixture, "\"complete\"", "is not admitted"},
		{"dynamic driver outcome", dbDriverSettlementFixture, "token", "must be a static literal"},
		{"misspelled server effect", dbServerAckFixture, "\"denied\"", "is not admitted"},
		{"dynamic server effect", dbServerAckFixture, "token", "must be a static literal"},
		{"misspelled grant engine", dbCancelGrantFixture, "\"oracle\"", "is not admitted"},
		{"misspelled quiesce engine", dbQuiesceEngineFixture, "\"oracle\"", "is not admitted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": tc.fixture(tc.word)})
			if err == nil {
				t.Fatalf("accepted invalid deadline word: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func TestDbObserveDeadlineDynamicInputsAdmit(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [db, test]\nfn db::deadline_record watch\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str work\n        str token\n        int deadline\n    asserts\n        first: \"work-0\", \"tok-stub\", 50 => ok db::deadline_record(\"work-0\", \"postgres\", 50, \"sha256:stub\")\n    match call db::observe_deadline(o, work, token, deadline)\n        when\n            first: o, work, token, deadline => ok db::deadline_record(\"work-0\", \"postgres\", 50, \"sha256:stub\")\n        test::stale_handle\n        db::db_fault\n        ok db::deadline_record got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected dynamic deadline inputs: %v", err)
	}
}

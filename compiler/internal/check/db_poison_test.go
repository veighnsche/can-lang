package check

import (
	"strings"
	"testing"
)

// record_poison_settlement and record_callback_report extend the
// static db operations: the poison outcome must be rolled-back
// or committed (no "unknown": a poison attempt settles
// terminally), and the callback report must be success or threw.
// Attempts, codes, details, rows and statements stay dynamic.

func dbPoisonSettlementFixture(outcome string) string {
	stub := "db::poison_settlement_record(\"attempt-0\", \"rolled-back\", true)"
	return "package app\n    provides []\n    uses [db, test]\nfn db::poison_settlement_record settle\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str attempt\n    asserts\n        first: \"attempt-0\" => ok " + stub + "\n    match call db::record_poison_settlement(o, attempt, " + outcome + ")\n        when\n            first: o, attempt, " + outcome + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::poison_settlement_record got => ok got\n" + programMain + "    ok\n"
}

func dbCallbackReportFixture(reported string) string {
	stub := "db::poison_callback_facts(\"attempt-0\", \"success\", \"nothing\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::poison_callback_facts report\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str attempt\n    asserts\n        first: \"attempt-0\" => ok " + stub + "\n    match call db::record_callback_report(o, attempt, " + reported + ")\n        when\n            first: o, attempt, " + reported + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::poison_callback_facts got => ok got\n" + programMain + "    ok\n"
}

func TestDbPoisonWordLiterals(t *testing.T) {
	for _, outcome := range []string{"\"rolled-back\"", "\"committed\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbPoisonSettlementFixture(outcome)}); err != nil {
			t.Fatalf("rejected literal poison outcome %s: %v", outcome, err)
		}
	}
	for _, reported := range []string{"\"success\"", "\"threw\""} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbCallbackReportFixture(reported)}); err != nil {
			t.Fatalf("rejected literal callback report %s: %v", reported, err)
		}
	}
}

func TestDbPoisonRefusals(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture func(string) string
		word    string
		want    string
	}{
		{"unknown is not a poison outcome", dbPoisonSettlementFixture, "\"unknown\"", "is not admitted"},
		{"misspelled outcome", dbPoisonSettlementFixture, "\"rolledback\"", "is not admitted"},
		{"dynamic outcome", dbPoisonSettlementFixture, "attempt", "must be a static literal"},
		{"misspelled report", dbCallbackReportFixture, "\"ok\"", "is not admitted"},
		{"dynamic report", dbCallbackReportFixture, "attempt", "must be a static literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": tc.fixture(tc.word)})
			if err == nil {
				t.Fatalf("accepted invalid poison word: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func TestDbBeginPoisonAttemptDynamicInputsAdmit(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [db, option, test]\nfn db::poison_attempt_record begin\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str[] schema\n        db::seed_row row\n        str statement\n    asserts\n        fresh: [\"number\"], db::seed_row([db::number_cell(\"7\")]), \"DELETE FROM items\" => ok db::poison_attempt_record(\"attempt-0\", \"poison\", [\"number\"], \"sha256:stub\", option::some(\"sha256:stub\"))\n    match call db::begin_poison_attempt(o, schema, row, statement)\n        when\n            fresh: o, schema, row, statement => ok db::poison_attempt_record(\"attempt-0\", \"poison\", [\"number\"], \"sha256:stub\", option::some(\"sha256:stub\"))\n        test::stale_handle\n        db::db_fault\n        ok db::poison_attempt_record got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected dynamic poison inputs: %v", err)
	}
}

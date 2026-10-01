package check

import (
	"strings"
	"testing"
)

// record_settlement is the only static db operation: outcome and
// engine must be K24 words. All other db inputs (actors, tokens,
// namespaces, tables, compiles, ids, statements) stay dynamic and
// are validated at runtime with precise db_fault failures.

func dbSettlementFixture(outcome, engine string) string {
	stub := "db::settlement_record(\"a1\", \"conn:a1\", \"committed\", \"postgres\", \"sha256:stub\")"
	return "package app\n    provides []\n    uses [db, test]\nfn db::settlement_record settle\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str actor\n        str token\n    asserts\n        first: \"a1\", \"tok\" => ok " + stub + "\n    match call db::record_settlement(o, actor, token, " + outcome + ", " + engine + ")\n        when\n            first: o, actor, token, " + outcome + ", " + engine + " => ok " + stub + "\n        test::stale_handle\n        db::db_fault\n        ok db::settlement_record got => ok got\n" + programMain + "    ok\n"
}

func TestDbSettlementOutcomeEngineLiterals(t *testing.T) {
	for _, tc := range [][2]string{
		{"\"committed\"", "\"postgres\""},
		{"\"rolled-back\"", "\"sqlite\""},
		{"\"unknown\"", "\"mysql\""},
	} {
		if _, err := programFixture(t, map[string]string{"src/main.can": dbSettlementFixture(tc[0], tc[1])}); err != nil {
			t.Fatalf("rejected literal outcome/engine %s/%s: %v", tc[0], tc[1], err)
		}
	}
}

func TestDbSettlementRefusals(t *testing.T) {
	for _, tc := range []struct{ name, outcome, engine, want string }{
		{"unknown outcome", "\"commited\"", "\"postgres\"", "is not admitted"},
		{"empty outcome", "\"\"", "\"postgres\"", "is not admitted"},
		{"dynamic outcome", "actor", "\"postgres\"", "must be a static literal"},
		{"unknown engine", "\"committed\"", "\"oracle\"", "is not admitted"},
		{"empty engine", "\"committed\"", "\"\"", "is not admitted"},
		{"dynamic engine", "\"committed\"", "token", "must be a static literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": dbSettlementFixture(tc.outcome, tc.engine)})
			if err == nil {
				t.Fatalf("accepted invalid outcome/engine: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func TestDbEnterCallbackDynamicActorAdmits(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [db, test]\nfn db::callback_entry enter\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str actor\n    asserts\n        first: \"a1\" => ok db::callback_entry(db::callback_entry_facts(\"a1\", \"conn:a1\", 0, \"sha256:stub\"), \"tok\")\n    match call db::enter_callback(o, actor)\n        when\n            first: o, actor => ok db::callback_entry(db::callback_entry_facts(\"a1\", \"conn:a1\", 0, \"sha256:stub\"), \"tok\")\n        test::stale_handle\n        db::db_fault\n        ok db::callback_entry got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected dynamic db actor: %v", err)
	}
}

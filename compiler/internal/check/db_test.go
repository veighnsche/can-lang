package check

import (
	"strings"
	"testing"
)

// NT-I13 admits the db:: operations through the catalogue task-tag
// allowlist only: every db string input is a dynamic name
// (namespace, table, connection, token, fixture, statement), and
// the closed cell-tag vocabulary (number/text/bytes/null) travels
// inside constructed db::cell values and schema arrays, which the
// adapter validates at runtime with precise db_fault failures.
// There are no closed-vocabulary scalar positions, so db needs no
// checkDbCall word rules; the fixtures below pin admission,
// generic emits enforcement, input typing, and the stale-only
// emits of the never-throwing compares.

func dbOpenFixture(missingFault bool, namespace string) string {
	fault := "\n        db::db_fault"
	if missingFault {
		fault = ""
	}
	return "package app\n    provides []\n    uses [db, test]\nfn db::namespace_receipt choose\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str probe\n    asserts\n        sample: \"shop\" => ok db::namespace_receipt(\"shop\", \"owner-1\", \"h\", \"sha256:abc\", 0)\n    match call db::open_namespace(o, " + namespace + ")\n        when\n            sample: o, \"shop\" => ok db::namespace_receipt(\"shop\", \"owner-1\", \"h\", \"sha256:abc\", 0)\n        test::stale_handle" + fault + "\n        ok db::namespace_receipt got => ok got\n" + programMain + "    ok\n"
}

func TestDbOpenNamespaceAdmits(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": dbOpenFixture(false, "\"shop\"")}); err != nil {
		t.Fatalf("rejected db::open_namespace call: %v", err)
	}
}

func TestDbOpenNamespaceDynamicNameAdmits(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [db, test]\nfn db::namespace_receipt choose\n    emits {test::stale_handle, db::db_fault}\n    given\n        test::owner o\n        str probe\n    asserts\n        sample: \"dyn\" => ok db::namespace_receipt(\"shop\", \"owner-1\", \"h\", \"sha256:abc\", 0)\n    match call db::open_namespace(o, probe)\n        when\n            sample: o, probe => ok db::namespace_receipt(\"shop\", \"owner-1\", \"h\", \"sha256:abc\", 0)\n        test::stale_handle\n        db::db_fault\n        ok db::namespace_receipt got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected dynamic db namespace: %v", err)
	}
}

func TestDbMissingFaultArmRefused(t *testing.T) {
	_, err := programFixture(t, map[string]string{"src/main.can": dbOpenFixture(true, "\"shop\"")})
	if err == nil {
		t.Fatal("accepted db call missing its db_fault arm")
	}
	if !strings.Contains(err.Error(), "db::db_fault") {
		t.Fatalf("wrong refusal for missing db_fault arm: %v", err)
	}
}

func TestDbWrongInputTypeRefused(t *testing.T) {
	_, err := programFixture(t, map[string]string{"src/main.can": dbOpenFixture(false, "7")})
	if err == nil {
		t.Fatal("accepted int db namespace")
	}
	if !strings.Contains(err.Error(), "does not fit expected type") {
		t.Fatalf("wrong refusal for int db namespace: %v", err)
	}
}

func TestDbCompareRowStaleOnlyEmits(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [db, test]\nfn db::row_comparison compare\n    emits {test::stale_handle}\n    given\n        test::owner o\n        db::cell[] stored\n        db::cell[] claimed\n    asserts\n        sample: [db::number_cell(\"1.10\")], [db::number_cell(\"1.1\")] => ok db::row_comparison(false, [0])\n    match call db::compare_row(o, stored, claimed)\n        when\n            sample: o, stored, claimed => ok db::row_comparison(false, [0])\n        test::stale_handle\n        ok db::row_comparison got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected stale-only db::compare_row call: %v", err)
	}
}

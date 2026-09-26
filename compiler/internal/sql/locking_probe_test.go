package sql

import (
	"strconv"
	"strings"
	"testing"
)

// TestF02LockingReadProbe records the X-R10-1 descriptor outcome: worker-
// claim locking reads check as ordinary row-returning SELECTs with no
// syntax change. The live execution half of the probe runs in
// tests/integration/sql_f02_test.go; this file pins only the checker
// admission the live legs depend on.
func TestF02LockingReadProbe(t *testing.T) {
	cases := []struct {
		name      string
		statement string
		params    []string
		cardinal  string
		limit     uint64
	}{
		{"skip", "SELECT id, payload FROM f02_claims WHERE worker = $1 ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED", []string{"worker"}, "many", 2},
		{"block", "SELECT id, payload FROM f02_claims WHERE worker = $1 ORDER BY id LIMIT $2 FOR UPDATE", []string{"worker"}, "many", 2},
		{"nowait", "SELECT id, payload FROM f02_claims WHERE worker = $1 ORDER BY id LIMIT $2 FOR UPDATE NOWAIT", []string{"worker"}, "many", 2},
		{"nokey", "SELECT id, payload FROM f02_claims WHERE worker = $1 ORDER BY id LIMIT $2 FOR NO KEY UPDATE SKIP LOCKED", []string{"worker"}, "many", 2},
		{"share", "SELECT id, payload FROM f02_claims WHERE worker = $1 ORDER BY id LIMIT $2 FOR SHARE", []string{"worker"}, "many", 2},
		{"keyed refetch", "SELECT id, payload FROM f02_claims WHERE worker = $1 AND payload = $2 LIMIT $3", []string{"worker", "payload"}, "one", 3},
		{"gen refetch", "SELECT id, payload FROM f02_gen WHERE payload = $1 ORDER BY id LIMIT $2", []string{"payload"}, "many", 2},
	}
	for _, c := range cases {
		got, err := CheckDescriptor(c.name, c.statement, c.params, c.cardinal, c.limit)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Kind != "SelectStmt" || got.Limit != int(c.limit) || got.Total != int(c.limit) {
			t.Fatalf("%s: %+v", c.name, got)
		}
		var rebuilt strings.Builder
		sites := 0
		for _, segment := range got.Segments {
			if segment.Param > 0 {
				sites++
				rebuilt.WriteString("$" + strconv.Itoa(segment.Param))
			} else {
				rebuilt.WriteString(segment.Text)
			}
		}
		if rebuilt.String() != c.statement {
			t.Fatalf("%s: segments do not tile the statement", c.name)
		}
		if sites != len(c.params)+1 {
			t.Fatalf("%s: %d parameter sites, want %d", c.name, sites, len(c.params)+1)
		}
	}
}

// TestF02ReturningStillRejected records the F02 precondition for the
// RETURNING-need question: the need demonstration runs against a checker
// that still rejects RETURNING, so any admission is F03's gated decision,
// not something this experiment smuggles in.
func TestF02ReturningStillRejected(t *testing.T) {
	for _, statement := range []string{
		"INSERT INTO f02_gen (payload) VALUES ($1) RETURNING id",
		"UPDATE f02_gen SET payload = $1 RETURNING id",
		"DELETE FROM f02_gen RETURNING id",
	} {
		params := []string{"payload"}
		if strings.HasPrefix(statement, "DELETE") {
			params = nil
		}
		_, err := CheckDescriptor("f02", statement, params, "execute", 0)
		if err == nil || !strings.Contains(err.Error(), "RETURNING is not admitted") {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

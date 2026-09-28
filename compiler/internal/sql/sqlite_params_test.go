package sql

import (
	"strings"
	"testing"
)

// siteNumbers runs the SQLite backend and returns the site numbers, refs
// and spans in reported order, failing on backend errors or failures.
func siteNumbers(t *testing.T, statement string) []ParamSite {
	t.Helper()
	analysis, err := Analyze(DialectSQLite, "t", statement)
	if err != nil {
		t.Fatalf("Analyze(%q): %v", statement, err)
	}
	if analysis.Failure != nil {
		t.Fatalf("Analyze(%q): failure %+v", statement, analysis.Failure)
	}
	return analysis.Sites
}

func checkSites(t *testing.T, statement string, wantNumbers []int, wantRefs []string) {
	t.Helper()
	sites := siteNumbers(t, statement)
	if len(sites) != len(wantNumbers) {
		t.Fatalf("%q: %d sites %+v, want numbers %v", statement, len(sites), sites, wantNumbers)
	}
	for i, site := range sites {
		if site.Number != wantNumbers[i] || site.Ref != wantRefs[i] {
			t.Fatalf("%q: site %d = %+v, want number %d ref %q", statement, i, site, wantNumbers[i], wantRefs[i])
		}
		if slice := statement[site.Start:site.End]; i < len(wantRefs) && wantRefs[i] != "" && !strings.HasPrefix(wantRefs[i], "?") && slice != wantRefs[i] {
			t.Fatalf("%q: site %d span %q does not slice its ref", statement, i, slice)
		}
	}
}

func TestSQLiteBareNumbering(t *testing.T) {
	checkSites(t, "SELECT * FROM t WHERE a = ? AND b = ? LIMIT ?",
		[]int{1, 2, 3}, []string{"?1", "?2", "?3"})
}

func TestSQLiteExplicitNumbering(t *testing.T) {
	checkSites(t, "SELECT * FROM t WHERE a = ?2 AND b = ?1 LIMIT ?3",
		[]int{2, 1, 3}, []string{"?2", "?1", "?3"})
}

func TestSQLiteBareAfterExplicitTakesMaxPlusOne(t *testing.T) {
	checkSites(t, "SELECT * FROM t WHERE a = ?5 LIMIT ?",
		[]int{5, 6}, []string{"?5", "?6"})
}

func TestSQLiteRepeatedNamedSharesNumber(t *testing.T) {
	checkSites(t, "SELECT * FROM t WHERE a = :a AND b = :a LIMIT :a",
		[]int{1, 1, 1}, []string{":a", ":a", ":a"})
}

func TestSQLiteSigilsAreDistinct(t *testing.T) {
	sites := siteNumbers(t, "SELECT * FROM t WHERE a = :a AND b = @a AND c = $a LIMIT ?")
	want := []ParamSite{
		{Number: 1, Start: 26, End: 28, Ref: ":a"},
		{Number: 2, Start: 37, End: 39, Ref: "@a"},
		{Number: 3, Start: 48, End: 50, Ref: "$a"},
		{Number: 4, Start: 57, End: 58, Ref: "?4"},
	}
	if len(sites) != len(want) {
		t.Fatalf("sites %+v, want %+v", sites, want)
	}
	for i := range want {
		if sites[i] != want[i] {
			t.Fatalf("site %d = %+v, want %+v", i, sites[i], want[i])
		}
	}
}

func TestSQLiteDigitNamesParse(t *testing.T) {
	checkSites(t, "SELECT * FROM t WHERE a = $1 LIMIT ?",
		[]int{1, 2}, []string{"$1", "?2"})
	checkSites(t, "SELECT * FROM t WHERE a = :1 LIMIT ?",
		[]int{1, 2}, []string{":1", "?2"})
	checkSites(t, "SELECT * FROM t WHERE a = @1 LIMIT ?",
		[]int{1, 2}, []string{"@1", "?2"})
}

func TestSQLiteDigitNamesFailFieldPolicy(t *testing.T) {
	_, err := CheckDescriptorDialect(DialectSQLite, "t", "SELECT * FROM t WHERE a = $1 LIMIT ?2", []string{"id"}, "one", 2)
	if err == nil || !strings.Contains(err.Error(), `site $1 does not match declared parameter "id"`) {
		t.Fatalf("digit name: err=%v", err)
	}
}

func TestSQLiteTclNameParsesThenFailsFieldPolicy(t *testing.T) {
	sites := siteNumbers(t, "SELECT * FROM t WHERE x = $name(arg)")
	if len(sites) != 1 || sites[0].Number != 1 || sites[0].Ref != "$name(arg)" {
		t.Fatalf("sites %+v", sites)
	}
	if slice := "SELECT * FROM t WHERE x = $name(arg)"[sites[0].Start:sites[0].End]; slice != "$name(arg)" {
		t.Fatalf("span %q", slice)
	}
	_, err := CheckDescriptorDialect(DialectSQLite, "t", "SELECT * FROM t WHERE x = $name(arg) LIMIT ?2", []string{"id"}, "one", 2)
	if err == nil || !strings.Contains(err.Error(), "does not match declared parameter") {
		t.Fatalf("tcl name: err=%v", err)
	}
}

func TestSQLiteHashSpellingRejected(t *testing.T) {
	analysis, err := Analyze(DialectSQLite, "t", "SELECT #name")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if analysis.Failure == nil {
		t.Fatalf("want failure, got %+v", analysis)
	}
	if !strings.Contains(analysis.Failure.Message, `unsupported parameter spelling "#name"`) || analysis.Failure.Cursor != 7 {
		t.Fatalf("failure %+v", analysis.Failure)
	}
	if len(analysis.Statements) != 0 || analysis.Version != 0 {
		t.Fatalf("failure must clear statements/version: %+v", analysis)
	}
}

func TestSQLiteZeroAndOverflowRejectedByCoverage(t *testing.T) {
	sites := siteNumbers(t, "SELECT * FROM t WHERE a = ?0 LIMIT ?1")
	if len(sites) != 2 || sites[0].Number != 0 {
		t.Fatalf("sites %+v", sites)
	}
	_, err := CheckDescriptorDialect(DialectSQLite, "t", "SELECT * FROM t WHERE a = ?0 LIMIT ?1", []string{"a"}, "one", 2)
	if err == nil || !strings.Contains(err.Error(), "has no declared parameter") {
		t.Fatalf("?0: err=%v", err)
	}
	_, err = CheckDescriptorDialect(DialectSQLite, "t", "SELECT * FROM t WHERE a = ?99999999999999999999999", []string{"a"}, "execute", 0)
	if err == nil || !strings.Contains(err.Error(), "has no declared parameter") {
		t.Fatalf("overflow: err=%v", err)
	}
}

func TestSQLiteLimitNameIsFree(t *testing.T) {
	got, err := CheckDescriptorDialect(DialectSQLite, "t", "SELECT name FROM users WHERE id = :id LIMIT :lim", []string{"id"}, "one", 2)
	if err != nil {
		t.Fatalf("CheckDescriptor: %v", err)
	}
	if got.Total != 2 || got.Limit != 2 {
		t.Fatalf("total/limit = %d/%d", got.Total, got.Limit)
	}
}

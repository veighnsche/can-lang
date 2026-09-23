package sql

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The G-SQL executable corpus pins every dialect backend to identical
// acceptance rows. It lives with the gate evidence; this runner is the
// in-repo implementation proof and fails loudly if the corpus moves.
const corpusPath = "../../../docs/bun-integration/asap/evidence/sql-corpus.json"

type corpusCase struct {
	ID          string   `json:"id"`
	Dialect     string   `json:"dialect"`
	Statement   string   `json:"statement"`
	Params      []string `json:"params"`
	Cardinality string   `json:"cardinality"`
	RowLimit    *int     `json:"row_limit"`
	Expect      any      `json:"expect"`
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Fatalf("read G-SQL corpus at %s: %v", corpusPath, err)
	}
	var corpus struct {
		Cases []corpusCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parse G-SQL corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("G-SQL corpus is empty")
	}
	return corpus.Cases
}

// invalidCategory extracts the required rejection category, or "" when
// the case must validate.
func invalidCategory(c corpusCase) string {
	want, ok := c.Expect.(map[string]any)
	if !ok {
		return ""
	}
	invalid, _ := want["invalid"].(string)
	return invalid
}

func categoryMatches(category, message string) bool {
	switch category {
	case "syntax":
		return strings.Contains(message, "syntax error")
	case "multi_statement":
		return strings.Contains(message, "statements, want 1")
	case "bad_kind":
		return strings.Contains(message, "requires SELECT") || strings.Contains(message, "requires INSERT, UPDATE, or DELETE")
	case "bad_limit":
		return strings.Contains(message, "unbounded SELECT") || strings.Contains(message, "top-level LIMIT must be") || strings.Contains(message, "appears") && strings.Contains(message, "want once in LIMIT")
	case "returning":
		return strings.Contains(message, "RETURNING is not admitted")
	case "param_unknown":
		return strings.Contains(message, "has no declared parameter")
	case "param_unused":
		return strings.Contains(message, "is not used")
	case "param_name":
		return strings.Contains(message, "does not match declared parameter")
	}
	return false
}

// checkTiling proves the checked segments tile the exact input bytes
// against the backend's own sites: every Text span matches literally,
// every Param consumes the next site in order, and the cursor ends at
// the input end with no gaps or overlaps.
func checkTiling(t *testing.T, c corpusCase, got Descriptor) {
	t.Helper()
	dialect, err := ParseDialect(c.Dialect)
	if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	analysis, err := Analyze(dialect, c.ID, c.Statement)
	if err != nil {
		t.Fatalf("%s: re-analyze: %v", c.ID, err)
	}
	cursor, site := 0, 0
	for _, segment := range got.Segments {
		if segment.Text != "" {
			if !strings.HasPrefix(c.Statement[cursor:], segment.Text) {
				t.Fatalf("%s: text segment %q does not tile at %d", c.ID, segment.Text, cursor)
			}
			cursor += len(segment.Text)
			continue
		}
		if site >= len(analysis.Sites) {
			t.Fatalf("%s: param segment %d exceeds %d sites", c.ID, segment.Param, len(analysis.Sites))
		}
		next := analysis.Sites[site]
		site++
		if next.Number != segment.Param || next.Start != cursor {
			t.Fatalf("%s: param segment %d breaks tiling at %d (site %+v)", c.ID, segment.Param, cursor, next)
		}
		cursor = next.End
	}
	if site != len(analysis.Sites) || cursor != len(c.Statement) {
		t.Fatalf("%s: tiling covers %d/%d sites and %d/%d bytes", c.ID, site, len(analysis.Sites), cursor, len(c.Statement))
	}
}

func TestCorpus(t *testing.T) {
	for _, c := range loadCorpus(t) {
		t.Run(c.ID, func(t *testing.T) {
			var limit uint64
			if c.RowLimit != nil {
				limit = uint64(*c.RowLimit)
			}
			dialect, err := ParseDialect(c.Dialect)
			if err != nil {
				t.Fatalf("dialect: %v", err)
			}
			got, err := CheckDescriptorDialect(dialect, c.ID, c.Statement, c.Params, c.Cardinality, limit)
			if category := invalidCategory(c); category != "" {
				if err == nil {
					t.Fatalf("want %s rejection, descriptor validated: %+v", category, got)
				}
				if !categoryMatches(category, err.Error()) {
					t.Fatalf("want %s rejection, got: %v", category, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want valid descriptor: %v", err)
			}
			if got.Name != c.ID || string(got.Dialect) != c.Dialect || got.Cardinality != c.Cardinality || got.Kind == "" || len(got.Segments) == 0 {
				t.Fatalf("descriptor fields: %+v", got)
			}
			total := len(c.Params)
			if c.Cardinality != "execute" {
				total++
			}
			if got.Total != total {
				t.Fatalf("total=%d, want %d", got.Total, total)
			}
			wantVersion := 170007
			if c.Dialect == "sqlite" {
				wantVersion = 15
			}
			if c.Dialect == "mysql" {
				wantVersion = MySQLVersion
			}
			if got.Version != wantVersion {
				t.Fatalf("version=%d, want %d", got.Version, wantVersion)
			}
			checkTiling(t, c, got)
		})
	}
}

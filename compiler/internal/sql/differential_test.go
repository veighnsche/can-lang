package sql

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The live MySQL differential replays every mysql corpus row through a
// real server-side EXPLAIN and requires the tidb backend to agree on
// every syntax verdict. The probe binds exactly the backend's own site
// count, so a miscounted `?` fails here instead of shifting the
// comparison. my-06 is the one reconciled row: tidb parses RETURNING
// structurally while MySQL 8.4 has no such clause, so the syntax
// verdicts differ by design and the Can policy rejection (already
// pinned by TestCorpus) is what agrees with the server.
//
// Gated on CAN_TEST_MYSQL_URL plus a bun binary; a skip names the
// missing piece, never a passing differential.
func TestMySQLDifferential(t *testing.T) {
	if os.Getenv("CAN_TEST_MYSQL_URL") == "" {
		t.Skip("set CAN_TEST_MYSQL_URL for the live MySQL differential")
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("bun binary not on PATH for the live MySQL differential")
	}
	dialect, err := ParseDialect("mysql")
	if err != nil {
		t.Fatal(err)
	}
	type probeRow struct {
		ID        string `json:"id"`
		Statement string `json:"statement"`
		Sites     int    `json:"sites"`
	}
	var rows []probeRow
	syntaxOK := map[string]bool{}
	for _, c := range loadCorpus(t) {
		if c.Dialect != "mysql" {
			continue
		}
		analysis, err := Analyze(dialect, c.ID, c.Statement)
		if err != nil {
			t.Fatalf("%s: analyze: %v", c.ID, err)
		}
		syntaxOK[c.ID] = analysis.Failure == nil && len(analysis.Statements) == 1
		rows = append(rows, probeRow{ID: c.ID, Statement: c.Statement, Sites: len(analysis.Sites)})
	}
	if len(rows) != 13 {
		t.Fatalf("mysql corpus moved: %d rows, want 13", len(rows))
	}
	dir := t.TempDir()
	rowsPath := filepath.Join(dir, "rows.json")
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rowsPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join("..", "..", "..", "docs", "bun-integration", "asap", "evidence", "mysql-differential-probe.ts")
	outPath := filepath.Join(dir, "report.json")
	cmd := exec.Command(bun, probe, rowsPath, outPath)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("differential probe: %v\n%s", err, out)
	}
	reportRaw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Bun   string `json:"bun"`
		MySQL struct {
			Version    string `json:"version"`
			SQLMode    string `json:"sql_mode"`
			SSLVersion string `json:"Ssl_version"`
			SSLCipher  string `json:"Ssl_cipher"`
		} `json:"mysql"`
		Verdicts map[string]string `json:"verdicts"`
	}
	if err := json.Unmarshal(reportRaw, &report); err != nil {
		t.Fatalf("parse differential report: %v", err)
	}
	if report.Bun != "1.4.2" {
		t.Fatalf("differential ran under bun %q, want pinned 1.4.2", report.Bun)
	}
	if !strings.HasPrefix(report.MySQL.Version, "8.4.") {
		t.Fatalf("differential ran against MySQL %q, want qualified 8.4.x", report.MySQL.Version)
	}
	for _, row := range rows {
		verdict, ok := report.Verdicts[row.ID]
		if !ok {
			t.Fatalf("%s: no server verdict", row.ID)
		}
		serverOK := verdict == "ok"
		if row.ID == "my-06" {
			if !syntaxOK[row.ID] || serverOK {
				t.Fatalf("my-06: want tidb-accept/server-reject, got tidb=%v server=%q", syntaxOK[row.ID], verdict)
			}
			continue
		}
		if syntaxOK[row.ID] != serverOK {
			t.Fatalf("%s: tidb=%v server=%q", row.ID, syntaxOK[row.ID], verdict)
		}
	}
	if save := os.Getenv("CAN_DIFF_SAVE"); save != "" {
		if !filepath.IsAbs(save) {
			save = filepath.Join("..", "..", "..", save)
		}
		if err := os.WriteFile(save, reportRaw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

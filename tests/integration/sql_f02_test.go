package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestF02LiveLockingAndReturning runs the F02 experiment driver: the live PG
// locking-read descriptor probe (X-R10-1) and the post-write-read/race-cost
// RETURNING demonstration. It needs a disposable PostgreSQL named by
// CAN_TEST_POSTGRES_URL (concrete per-run URL, or the provision template
// with a <db> placeholder selecting can_f02) plus bun on PATH; otherwise it
// skips. CAN_TEST_MYSQL_URL enables the MySQL RETURNING-admissibility leg.
// The URL (including any password) is never logged.
func TestF02LiveLockingAndReturning(t *testing.T) {
	url := os.Getenv("CAN_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for the F02 live probe")
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("bun is not on PATH for the F02 live probe")
	}
	redact := func(text string) string {
		sanitized := strings.ReplaceAll(text, url, "<redacted-url>")
		if at := strings.LastIndex(url, "@"); at >= 0 {
			if colon := strings.LastIndex(url[:at], ":"); colon >= 0 {
				if password := url[colon+1 : at]; password != "" {
					sanitized = strings.ReplaceAll(sanitized, password, "<redacted-password>")
				}
			}
		}
		return sanitized
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/f02-live-driver.ts")
	cmd := exec.CommandContext(ctx, bun, driver)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + cmd.Dir, "CAN_TEST_POSTGRES_URL=" + url}
	if mysql := os.Getenv("CAN_TEST_MYSQL_URL"); mysql != "" {
		cmd.Env = append(cmd.Env, "CAN_TEST_MYSQL_URL="+mysql)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("f02 driver: %v %s", err, redact(string(out)))
	}
	var report struct {
		Error   string `json:"error"`
		Version string `json:"serverVersion"`
		Locking struct {
			Tx1Claimed float64 `json:"tx1_claimed"`
			Tx2Skipped float64 `json:"tx2_skipped_count"`
			Nowait     struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
				Payload struct {
					Operation string `json:"operation"`
					Code      string `json:"code"`
				} `json:"payload"`
			} `json:"tx2_nowait"`
			NowaitInTxn struct {
				Query string `json:"query"`
				Txn   string `json:"txn"`
			} `json:"tx2_nowait_in_txn"`
			Tx1Commit    string  `json:"tx1_commit"`
			AfterRelease float64 `json:"tx2_after_release"`
			Blocking     float64 `json:"blocking_for_update_count"`
			NoKey        float64 `json:"nokey_count"`
		} `json:"locking"`
		Cost struct {
			Iterations  float64 `json:"iterations"`
			CanMS       float64 `json:"can_txn_insert_plus_refetch_ms"`
			ReturningMS float64 `json:"raw_single_returning_ms"`
		} `json:"cost"`
		Ambiguity struct {
			Inserts      float64 `json:"concurrent_identical_inserts"`
			Candidates   float64 `json:"refetch_candidate_rows"`
			Identifiable bool    `json:"writer_identifiable"`
		} `json:"ambiguity"`
		ReturningOracle struct {
			Rows float64 `json:"rows"`
		} `json:"returning_oracle"`
		Keyed struct {
			Writers float64 `json:"writers"`
			AllOwn  bool    `json:"all_refetched_own_row"`
		} `json:"keyed_control"`
		MySQL any `json:"mysql"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("invalid f02 report %v %s", err, redact(string(out)))
	}
	if report.Error != "" {
		t.Fatalf("f02 driver: %s", redact(report.Error))
	}
	if !strings.HasPrefix(report.Version, "PostgreSQL 17.") {
		t.Fatalf("unexpected live database %q", redact(report.Version))
	}
	// Locking probe: SKIP LOCKED claims flow through checked descriptors
	// and the runtime path with real skip semantics; NOWAIT classifies to
	// sql::query_failed/55P03 both outside and inside a Can transaction.
	locking := report.Locking
	if locking.Tx1Claimed != 4 {
		t.Fatalf("tx1 claimed %v rows, want 4", locking.Tx1Claimed)
	}
	if locking.Tx2Skipped != 0 {
		t.Fatalf("tx2 skip-locked saw %v rows under contention, want 0", locking.Tx2Skipped)
	}
	if locking.Nowait.Outcome != "domain" || locking.Nowait.Name != "sql::query_failed" ||
		locking.Nowait.Payload.Operation != "query_rows" || locking.Nowait.Payload.Code != "55P03" {
		t.Fatalf("nowait classification %+v, want sql::query_failed/query_rows/55P03", locking.Nowait)
	}
	if !strings.Contains(locking.NowaitInTxn.Query, "55P03") {
		t.Fatalf("in-txn nowait query %q, want a 55P03 classification", locking.NowaitInTxn.Query)
	}
	if locking.Tx1Commit != "ok" {
		t.Fatalf("tx1 commit %q, want ok", locking.Tx1Commit)
	}
	if locking.AfterRelease != 4 || locking.Blocking != 4 || locking.NoKey != 4 {
		t.Fatalf("after=%v blocking=%v nokey=%v, want 4 each",
			locking.AfterRelease, locking.Blocking, locking.NoKey)
	}
	// Cost leg: the endorsed shape is reported, not thresholded; loopback
	// timing is evidence, not a gate.
	if report.Cost.Iterations == 0 || report.Cost.CanMS <= 0 || report.Cost.ReturningMS <= 0 {
		t.Fatalf("cost leg missing measurements %+v", report.Cost)
	}
	// Ambiguity leg: concurrent identical inserts without a natural unique
	// key refetch as an indistinguishable candidate set.
	if report.Ambiguity.Inserts != 8 || report.Ambiguity.Candidates != 8 || report.Ambiguity.Identifiable {
		t.Fatalf("ambiguity leg %+v, want 8 inserts/8 candidates/unidentifiable", report.Ambiguity)
	}
	if report.ReturningOracle.Rows != 1 {
		t.Fatalf("returning oracle rows %v, want 1", report.ReturningOracle.Rows)
	}
	// Keyed control: same-transaction refetch by app-supplied unique key is
	// exact under concurrency.
	if report.Keyed.Writers != 16 || !report.Keyed.AllOwn {
		t.Fatalf("keyed control %+v, want 16 writers all exact", report.Keyed)
	}
	// MySQL leg: present when CAN_TEST_MYSQL_URL is set; MySQL 8.4 rejects
	// INSERT...RETURNING, so any admission needs a per-dialect contract.
	if mysql, ok := report.MySQL.(map[string]any); ok {
		ret, _ := mysql["insert_returning"].(map[string]any)
		if ret["rejected"] != true {
			t.Fatalf("mysql insert_returning %+v, want a rejection record", ret)
		}
		t.Logf("f02 live: PG %s, mysql %v", strings.SplitN(report.Version, " on ", 2)[0], mysql["serverVersion"])
	} else {
		t.Logf("f02 live: PG %s, mysql skipped (%v)", strings.SplitN(report.Version, " on ", 2)[0], report.MySQL)
	}
}

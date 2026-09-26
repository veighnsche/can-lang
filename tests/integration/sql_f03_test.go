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

// TestF03LiveRelationalSlice runs the F03 qualification driver: native
// RETURNING row behavior per dialect, the runtime cardinality/error
// vocabulary the RETURNING path reuses, PG aborted-transaction and
// replay behavior, the lookup-first conflict recipe, blessed-encoding
// roundtrips, and the MySQL RETURNING rejection plus LAST_INSERT_ID
// mapping. It needs a disposable PostgreSQL named by
// CAN_TEST_POSTGRES_URL (concrete per-run URL, or the provision template
// with a <db> placeholder selecting can_f03) plus bun on PATH; otherwise
// it skips. CAN_TEST_MYSQL_URL enables the MySQL legs. The URL (including
// any password) is never logged.
func TestF03LiveRelationalSlice(t *testing.T) {
	url := os.Getenv("CAN_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for the F03 live probe")
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("bun is not on PATH for the F03 live probe")
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
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/f03-live-driver.ts")
	cmd := exec.CommandContext(ctx, bun, driver)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + cmd.Dir, "CAN_TEST_POSTGRES_URL=" + url}
	if mysql := os.Getenv("CAN_TEST_MYSQL_URL"); mysql != "" {
		cmd.Env = append(cmd.Env, "CAN_TEST_MYSQL_URL="+mysql)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("f03 driver: %v %s", err, redact(string(out)))
	}
	var report struct {
		Error   string `json:"error"`
		Version string `json:"serverVersion"`
		Native  struct {
			Single     float64 `json:"single_rows"`
			SingleType string  `json:"single_id_type"`
			Multi      float64 `json:"multi_rows"`
			First      float64 `json:"conflict_first_rows"`
			Repeat     float64 `json:"conflict_repeat_rows"`
			Conflict   struct {
				Rejected   bool   `json:"rejected"`
				Errno      string `json:"errno"`
				Constraint string `json:"constraint"`
			} `json:"conflict"`
		} `json:"returning_native"`
		Vocab struct {
			Duplicate struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
				Payload struct {
					Constraint string `json:"constraint"`
				} `json:"payload"`
			} `json:"duplicate_insert"`
			Absent struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
			} `json:"absent_one"`
			TwoRow struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
				Payload struct {
					Actual float64 `json:"actual"`
				} `json:"payload"`
			} `json:"two_row_one"`
			Wide struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
				Payload struct {
					Reason string `json:"reason"`
				} `json:"payload"`
			} `json:"wide_decode"`
		} `json:"error_vocab"`
		Aborted struct {
			First struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
			} `json:"first_outcome"`
			Followup struct {
				Outcome string `json:"outcome"`
				Name    string `json:"name"`
				Payload struct {
					Code string `json:"code"`
				} `json:"payload"`
			} `json:"followup_outcome"`
			Txn struct {
				Outcome string `json:"outcome"`
			} `json:"txn_outcome"`
			Replay bool `json:"replay_ok"`
		} `json:"aborted_txn"`
		Lookup struct {
			Writers  float64 `json:"writers"`
			Settled  float64 `json:"settled"`
			Accepted float64 `json:"accepted"`
			Winner   string  `json:"winner_digest"`
			Poisoned bool    `json:"saw_25P02"`
		} `json:"lookup_first"`
		Encodings struct {
			NullNone bool `json:"null_audit_is_none"`
			Minor    bool `json:"minor_unit"`
			Epoch    bool `json:"ms_epoch"`
			SetAudit bool `json:"set_audit"`
			JSON     bool `json:"json_codec"`
			Max      bool `json:"int64_max"`
			Min      bool `json:"int64_min"`
		} `json:"encodings"`
		MySQL  any `json:"mysql"`
		SQLite struct {
			Version string  `json:"version"`
			Single  float64 `json:"single_rows"`
			Multi   float64 `json:"multi_rows"`
		} `json:"sqlite"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("invalid f03 report %v %s", err, redact(string(out)))
	}
	if report.Error != "" {
		t.Fatalf("f03 driver: %s", redact(report.Error))
	}
	if !strings.HasPrefix(report.Version, "PostgreSQL 17.") {
		t.Fatalf("unexpected live database %q", redact(report.Version))
	}
	// Native RETURNING rows: single-row insert yields exactly one row
	// with a bigint identity; multi-row yields N; ON CONFLICT DO
	// NOTHING yields 1 then 0; a plain conflicting RETURNING insert
	// surfaces the native 23505 with its constraint.
	native := report.Native
	if native.Single != 1 || native.SingleType != "bigint" {
		t.Fatalf("single RETURNING rows=%v id=%s, want 1/bigint", native.Single, native.SingleType)
	}
	if native.Multi != 3 {
		t.Fatalf("multi RETURNING rows=%v, want 3", native.Multi)
	}
	if native.First != 1 || native.Repeat != 0 {
		t.Fatalf("do-nothing RETURNING first=%v repeat=%v, want 1/0", native.First, native.Repeat)
	}
	if !native.Conflict.Rejected || native.Conflict.Errno != "23505" || native.Conflict.Constraint != "f03_deliveries_pkey" {
		t.Fatalf("conflict RETURNING %+v, want rejected 23505/f03_deliveries_pkey", native.Conflict)
	}
	// Runtime vocabulary: duplicate insert classifies to
	// constraint_failed; absent/ambiguous query_one to row_missing and
	// row_count; a projection wider than the row record to
	// schema_mismatch/extra_column.
	vocab := report.Vocab
	if vocab.Duplicate.Outcome != "domain" || vocab.Duplicate.Name != "sql::constraint_failed" ||
		vocab.Duplicate.Payload.Constraint != "f03_deliveries_pkey" {
		t.Fatalf("duplicate insert %+v, want sql::constraint_failed/f03_deliveries_pkey", vocab.Duplicate)
	}
	if vocab.Absent.Outcome != "domain" || vocab.Absent.Name != "sql::row_missing" {
		t.Fatalf("absent one %+v, want sql::row_missing", vocab.Absent)
	}
	if vocab.TwoRow.Outcome != "domain" || vocab.TwoRow.Name != "sql::row_count" || vocab.TwoRow.Payload.Actual != 2 {
		t.Fatalf("two-row one %+v, want sql::row_count actual 2", vocab.TwoRow)
	}
	if vocab.Wide.Outcome != "domain" || vocab.Wide.Name != "sql::schema_mismatch" || vocab.Wide.Payload.Reason != "extra_column" {
		t.Fatalf("wide decode %+v, want sql::schema_mismatch/extra_column", vocab.Wide)
	}
	// Aborted transaction: the followup inside the poisoned transaction
	// fails 25P02 at the query level while the transaction itself still
	// resolves ok with nothing to commit (F02 NOWAIT observation,
	// extended); a fresh replay then succeeds on the same pool.
	aborted := report.Aborted
	if aborted.First.Outcome != "domain" || aborted.First.Name != "sql::constraint_failed" {
		t.Fatalf("poisoning statement %+v, want sql::constraint_failed", aborted.First)
	}
	if aborted.Followup.Outcome != "domain" || aborted.Followup.Name != "sql::query_failed" ||
		aborted.Followup.Payload.Code != "25P02" {
		t.Fatalf("poisoned followup %+v, want sql::query_failed/25P02", aborted.Followup)
	}
	if aborted.Txn.Outcome != "ok" {
		t.Fatalf("poisoned txn %+v, want ok-with-nothing-committed", aborted.Txn)
	}
	if !aborted.Replay {
		t.Fatalf("replay after abort failed")
	}
	// Lookup-first recipe: exactly one of eight concurrent writers wins;
	// all settle without ever seeing 25P02.
	lookup := report.Lookup
	if lookup.Writers != 8 || lookup.Settled != 8 || lookup.Accepted != 1 {
		t.Fatalf("lookup-first %+v, want 8 writers/8 settled/1 accepted", lookup)
	}
	if lookup.Winner != "race-digest" || lookup.Poisoned {
		t.Fatalf("lookup-first winner=%q poisoned=%v, want race-digest/false", lookup.Winner, lookup.Poisoned)
	}
	// Blessed encodings: minor-unit money, ms-epoch times with a
	// nullable audit instant, TEXT JSON codec, int64 edges.
	enc := report.Encodings
	if !enc.NullNone || !enc.Minor || !enc.Epoch || !enc.SetAudit || !enc.JSON || !enc.Max || !enc.Min {
		t.Fatalf("encodings %+v, want all roundtrips exact", enc)
	}
	// SQLite native RETURNING: single and multi-row shapes.
	if report.SQLite.Single != 1 || report.SQLite.Multi != 3 {
		t.Fatalf("sqlite %+v, want 1/3 rows", report.SQLite)
	}
	// MySQL legs: present when CAN_TEST_MYSQL_URL is set; RETURNING is
	// rejected (1064) and the LAST_INSERT_ID mapping returns every
	// concurrent writer's own row.
	if mysql, ok := report.MySQL.(map[string]any); ok {
		ret, _ := mysql["insert_returning"].(map[string]any)
		if ret["rejected"] != true {
			t.Fatalf("mysql insert_returning %+v, want a rejection record", ret)
		}
		if errno, _ := ret["errno"].(float64); errno != 1064 {
			t.Fatalf("mysql insert_returning errno %v, want 1064", ret["errno"])
		}
		mapping, _ := mysql["mapping"].(map[string]any)
		if mapping["writers"] != float64(8) || mapping["all_own_row"] != true || mapping["distinct_ids"] != float64(8) {
			t.Fatalf("mysql mapping %+v, want 8 writers all exact", mapping)
		}
		t.Logf("f03 live: PG %s, mysql %v, sqlite %s",
			strings.SplitN(report.Version, " on ", 2)[0], mysql["serverVersion"], report.SQLite.Version)
	} else {
		t.Logf("f03 live: PG %s, mysql skipped (%v), sqlite %s",
			strings.SplitN(report.Version, " on ", 2)[0], report.MySQL, report.SQLite.Version)
	}
}

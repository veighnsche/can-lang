package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const sqlQueriesManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{` +
	`"account_by_id":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2},` +
	`"account_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"optional","row_limit_parameter":2},` +
	`"accounts_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},` +
	`"add_account":{"dialect":"postgresql","statement":"INSERT INTO can_i35_accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"},` +
	`"cover_by_id":{"dialect":"postgresql","statement":"SELECT id, payload, note FROM can_i35_cover WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::cover_row","cardinality":"one","row_limit_parameter":2}}}`

func writeSQLQueriesProject(t *testing.T, root, manifest string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/sql/queries.can"))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", manifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(fixture))
}

func TestCurrentSQLQueries(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL query execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-queries-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLQueriesProject(t, root, sqlQueriesManifest)
	runAt := func(root, command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	status, out, diag := runAt(root, "assert")
	if status != 0 || diag != "" {
		t.Fatalf("query assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 9 {
		t.Fatalf("invalid query report %v %s", err, out)
	}
	buildAt := func(root string) (string, string) {
		t.Helper()
		status, out, diag := runAt(root, "build")
		if status != 0 {
			t.Fatalf("build %s: %d %s %s", root, status, out, diag)
		}
		var build struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &build); err != nil {
			t.Fatal(err)
		}
		return build.BuildID, build.Directory
	}
	firstID, firstDir := buildAt(root)
	secondID, _ := buildAt(root)
	if firstID != secondID {
		t.Fatalf("rebuild changed build identity %s %s", firstID, secondID)
	}
	moved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLQueriesProject(t, moved, sqlQueriesManifest)
	if movedID, _ := buildAt(moved); movedID != firstID {
		t.Fatalf("relocated build changed identity %s %s", movedID, firstID)
	}
	state, err := os.ReadFile(filepath.Join(firstDir, "program/state.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canSQLPools=$canCreateSQLPools(",
		`"account_by_term":{"dialect":"postgresql","cardinality":"optional"`,
		"$canSQLPools.queryOne", "$canSQLPools.queryOptional",
		"$canSQLPools.queryRows", "$canSQLPools.execute",
		`{"name":"id","kind":"int"}`, `{"name":"note","kind":"option"`,
	} {
		if !strings.Contains(string(state), needle) {
			t.Fatalf("missing emitted query wiring %s", needle)
		}
	}
	program, err := filepath.Glob(filepath.Join(firstDir, "packages/*/*.ts"))
	if err != nil || len(program) != 1 {
		t.Fatalf("expected one emitted program unit, saw %v %v", program, err)
	}
	unit, err := os.ReadFile(program[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canSQLPools.open", "$canSQLPools.close",
	} {
		if !strings.Contains(string(unit), needle) {
			t.Fatalf("missing emitted query call %s", needle)
		}
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(firstDir, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	negatives := []struct {
		name     string
		manifest string
		want     string
	}{
		{"cardinality", strings.Replace(sqlQueriesManifest, `"account_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"optional"`, `"account_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"one"`, 1), "needs cardinality optional"},
		{"parameter type", strings.Replace(sqlQueriesManifest, `"parameter_type":"app::id_parameters"`, `"parameter_type":"app::missing"`, 1), "not declared"},
		{"row type", strings.Replace(sqlQueriesManifest, `"row_type":"app::cover_row"`, `"row_type":"app::account_row"`, 1), "binds rows"},
		{"undeclared", strings.Replace(sqlQueriesManifest, `,"cover_by_id":{"dialect":"postgresql","statement":"SELECT id, payload, note FROM can_i35_cover WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::cover_row","cardinality":"one","row_limit_parameter":2}`, ``, 1), "undeclared descriptor"},
	}
	for _, negative := range negatives {
		if negative.manifest == sqlQueriesManifest {
			t.Fatalf("%s replacement missed", negative.name)
		}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeSQLQueriesProject(t, dir, negative.manifest)
		status, out, diag := runAt(dir, "build")
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
}

// TestCurrentSQLQueriesLive exercises the compiled query program against a
// disposable PostgreSQL named by CAN_TEST_POSTGRES_URL. Every check runs the
// built entry.ts CLI with a fd-3 environment snapshot holding the credential;
// the process environment never carries the URL, proving the read crosses
// the captured boundary. Tables are seeded and dropped by the fixed
// testdata seed; it runs outside sandbox-exec because real SQL needs
// loopback TCP. The URL (including any password) is never logged.
func TestCurrentSQLQueriesLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL query execution")
	}
	url := os.Getenv("CAN_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for live query checks")
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
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-queries-live")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLQueriesProject(t, root, sqlQueriesManifest)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = home
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live build: %v %s", err, redact(string(buildOut)))
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil || build.Directory == "" {
		t.Fatalf("invalid live build report %v %s", err, redact(string(buildOut)))
	}
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/sql/queries.sql")
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/queries-driver.ts")
	drive := func(mode string) map[string]any {
		t.Helper()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, mode, seed)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_POSTGRES_URL=" + url}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("live %s: %v %s", mode, err, redact(string(out)))
		}
		var report map[string]any
		if err := json.Unmarshal(out, &report); err != nil {
			t.Fatalf("invalid live %s report %v %s", mode, err, redact(string(out)))
		}
		if failure, ok := report["error"]; ok {
			t.Fatalf("live %s: %v", mode, redact(failure.(string)))
		}
		return report
	}
	setup := drive("setup")
	version, _ := setup["serverVersion"].(string)
	if !strings.HasPrefix(version, "PostgreSQL 17.") {
		t.Fatalf("unexpected live database %q", redact(version))
	}
	if setup["accounts"] != 6.0 || setup["cover"] != 2.0 {
		t.Fatalf("unexpected live seed accounts=%v cover=%v", setup["accounts"], setup["cover"])
	}
	defer drive("teardown")
	snapshots := 0
	snapshotFor := func(credential string) string {
		t.Helper()
		data, err := json.Marshal(map[string]string{"CAN_TEST_POSTGRES": credential})
		if err != nil {
			t.Fatal(err)
		}
		snapshots++
		path := filepath.Join(home, "snapshot-"+strconv.Itoa(snapshots)+".json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	live := snapshotFor(url)
	refused := snapshotFor("postgres://127.0.0.1:1/nope")
	run := func(snapshot string, args ...string) (int, string, string) {
		t.Helper()
		// A fresh handle per run: the child consumes fd 3 to EOF.
		file, err := os.Open(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(build.Directory, "entry.ts")}, args...)...)
		cmd.Dir = home
		// The URL travels only in the fd-3 snapshot, never the environment.
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		cmd.ExtraFiles = []*os.File{file}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(live, args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, redact(out), redact(diag))
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(live, args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, redact(out), redact(diag))
		}
	}
	// Connection establishment proves itself: open awaits connect, and the
	// close round trip succeeds against the live server.
	ok([]string{"close"}, "closed")
	ok([]string{"by_id", "4"}, `{"id":4,"display_name":"Bob"}`)
	fault([]string{"by_id", "999"}, "sql::row_missing")
	ok([]string{"by_term", strconv.Quote("Bob")}, `{"case":"option::some<app::account_row>","value":{"value":{"id":4,"display_name":"Bob"}}}`)
	ok([]string{"by_term", strconv.Quote("Nobody")}, `{"case":"option::none","value":{}}`)
	fault([]string{"by_term", strconv.Quote("%o%")}, "sql::row_count")
	ok([]string{"rows", strconv.Quote("%son%"), "10"}, `[{"id":2,"display_name":"Jason"},{"id":3,"display_name":"Mason"}]`)
	fault([]string{"rows", strconv.Quote("%o%"), "1"}, "sql::row_limit")
	ok([]string{"add", strconv.Quote("Zed")}, "1")
	fault([]string{"add", strconv.Quote("Zed")}, "sql::constraint_failed")
	ok([]string{"cover", "1"}, `1[65,66]{"case":"option::some<str>","value":{"value":"n"}}`)
	ok([]string{"cover", "2"}, `2[]{"case":"option::none","value":{}}`)
	fault([]string{"cover", "999"}, "sql::row_missing")
	// Hostile input binds as a value: no rows match, and the table survives.
	ok([]string{"by_term", strconv.Quote("%' OR '1'='1")}, `{"case":"option::none","value":{}}`)
	ok([]string{"rows", strconv.Quote("%"), "100"}, `[{"id":1,"display_name":"Ann"},{"id":2,"display_name":"Jason"},{"id":3,"display_name":"Mason"},{"id":4,"display_name":"Bob"},{"id":5,"display_name":"O'Brien"},{"id":6,"display_name":"Zoë"},{"id":7,"display_name":"Zed"}]`)
	// A refused port surfaces the connect phase instead of a lazy handle.
	checks++
	status, out, diag := run(refused, "close")
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"sql::connection_failed"`) {
		t.Fatalf("live refusal: %d stdout=%q stderr=%s", status, redact(out), redact(diag))
	}
	t.Logf("live queries: %d checks against %s", checks, strings.SplitN(version, " on ", 2)[0])
}

const sqlTransactionsManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{` +
	`"tx_add":{"dialect":"postgresql","statement":"INSERT INTO can_i38_accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"},` +
	`"tx_get":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i38_accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2}}}`

func writeSQLTransactionsProject(t *testing.T, root, manifest string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/sql/transactions.can"))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", manifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(fixture))
}

func TestCurrentSQLTransactions(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL transaction execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-transactions-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLTransactionsProject(t, root, sqlTransactionsManifest)
	runAt := func(root, command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	status, out, diag := runAt(root, "assert")
	if status != 0 || diag != "" {
		t.Fatalf("transaction assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 12 {
		t.Fatalf("invalid transaction report %v %s", err, out)
	}
	buildAt := func(root string) (string, string) {
		t.Helper()
		status, out, diag := runAt(root, "build")
		if status != 0 {
			t.Fatalf("build %s: %d %s %s", root, status, out, diag)
		}
		var build struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &build); err != nil {
			t.Fatal(err)
		}
		return build.BuildID, build.Directory
	}
	firstID, firstDir := buildAt(root)
	secondID, _ := buildAt(root)
	if firstID != secondID {
		t.Fatalf("rebuild changed build identity %s %s", firstID, secondID)
	}
	moved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLTransactionsProject(t, moved, sqlTransactionsManifest)
	if movedID, _ := buildAt(moved); movedID != firstID {
		t.Fatalf("relocated build changed identity %s %s", movedID, firstID)
	}
	state, err := os.ReadFile(filepath.Join(firstDir, "program/state.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canTransactions=$canCreateSQLTransactions(",
		`$canSQLTransaction0 = Object.freeze({run:(pool:unknown,callback:unknown`,
		`$canTransactions.withTransaction(pool,callback,{commit:`,
		`"tx_get":{"dialect":"postgresql","cardinality":"one"`,
		"$canTransactions.queryOne", "$canTransactions.execute",
		`{"name":"id","kind":"int"}`,
	} {
		if !strings.Contains(string(state), needle) {
			t.Fatalf("missing emitted transaction wiring %s", needle)
		}
	}
	program, err := filepath.Glob(filepath.Join(firstDir, "packages/*/*.ts"))
	if err != nil || len(program) != 1 {
		t.Fatalf("expected one emitted program unit, saw %v %v", program, err)
	}
	unit, err := os.ReadFile(program[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canSQLTransaction0.run",
		"$canSQLQuery0.run", "$canSQLQuery1.run",
		"$canSQLPools.open", "$canSQLPools.close",
	} {
		if !strings.Contains(string(unit), needle) {
			t.Fatalf("missing emitted transaction call %s", needle)
		}
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(firstDir, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	negatives := []struct {
		name     string
		manifest string
		want     string
	}{
		{"cardinality", strings.Replace(sqlTransactionsManifest, `"cardinality":"one"`, `"cardinality":"optional"`, 1), "needs cardinality one"},
		{"parameter type", strings.Replace(sqlTransactionsManifest, `"parameter_type":"app::id_parameters"`, `"parameter_type":"app::missing"`, 1), "not declared"},
		{"row type", strings.Replace(sqlTransactionsManifest, `"row_type":"app::account_row","cardinality":"one"`, `"row_type":"app::search_parameters","cardinality":"one"`, 1), "binds rows"},
		{"undeclared", strings.Replace(sqlTransactionsManifest, `"tx_add":{"dialect":"postgresql","statement":"INSERT INTO can_i38_accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"},`, ``, 1), "undeclared descriptor"},
	}
	for _, negative := range negatives {
		if negative.manifest == sqlTransactionsManifest {
			t.Fatalf("%s replacement missed", negative.name)
		}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeSQLTransactionsProject(t, dir, negative.manifest)
		status, out, diag := runAt(dir, "build")
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
}

// TestCurrentSQLTransactionsLive exercises the compiled transaction program
// against a disposable PostgreSQL named by CAN_TEST_POSTGRES_URL. Every check
// runs the built entry.ts CLI with a fd-3 environment snapshot holding the
// credential; the process environment never carries the URL. Tables are
// seeded and dropped by the fixed testdata seed; it runs outside
// sandbox-exec because real SQL needs loopback TCP. The URL (including any
// password) is never logged.
func TestCurrentSQLTransactionsLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL transaction execution")
	}
	url := os.Getenv("CAN_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for live transaction checks")
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
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-transactions-live")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLTransactionsProject(t, root, sqlTransactionsManifest)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = home
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live build: %v %s", err, redact(string(buildOut)))
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil || build.Directory == "" {
		t.Fatalf("invalid live build report %v %s", err, redact(string(buildOut)))
	}
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/sql/transactions.sql")
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/transactions-driver.ts")
	drive := func(mode string) map[string]any {
		t.Helper()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, mode, seed)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_POSTGRES_URL=" + url}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("live %s: %v %s", mode, err, redact(string(out)))
		}
		var report map[string]any
		if err := json.Unmarshal(out, &report); err != nil {
			t.Fatalf("invalid live %s report %v %s", mode, err, redact(string(out)))
		}
		if failure, ok := report["error"]; ok {
			t.Fatalf("live %s: %v", mode, redact(failure.(string)))
		}
		return report
	}
	setup := drive("setup")
	version, _ := setup["serverVersion"].(string)
	if !strings.HasPrefix(version, "PostgreSQL 17.") {
		t.Fatalf("unexpected live database %q", redact(version))
	}
	if setup["accounts"] != 6.0 {
		t.Fatalf("unexpected live seed accounts=%v", setup["accounts"])
	}
	defer drive("teardown")
	snapshots := 0
	snapshotFor := func(credential string) string {
		t.Helper()
		data, err := json.Marshal(map[string]string{"CAN_TEST_POSTGRES": credential})
		if err != nil {
			t.Fatal(err)
		}
		snapshots++
		path := filepath.Join(home, "snapshot-"+strconv.Itoa(snapshots)+".json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	live := snapshotFor(url)
	refused := snapshotFor("postgres://127.0.0.1:1/nope")
	run := func(snapshot string, args ...string) (int, string, string) {
		t.Helper()
		file, err := os.Open(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(build.Directory, "entry.ts")}, args...)...)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		cmd.ExtraFiles = []*os.File{file}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(live, args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, redact(out), redact(diag))
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(live, args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, redact(out), redact(diag))
		}
	}
	// Commits land, rollbacks vanish, and the identity sequence proves both:
	// Zed commits at 7, the conflicting duplicate consumes 8, the rolled-back
	// insert consumes 9, and Yara lands at 10.
	ok([]string{"fetch", "4"}, "4")
	ok([]string{"fetch", "999"}, "0")
	ok([]string{"commit", strconv.Quote("Zed")}, "1")
	ok([]string{"fetch", "7"}, "7")
	ok([]string{"commit", strconv.Quote("Zed")}, "0")
	ok([]string{"rollback", strconv.Quote("TxNo")}, "1")
	ok([]string{"commit", strconv.Quote("Yara")}, "1")
	ok([]string{"fetch", "10"}, "10")
	ok([]string{"fetch", "8"}, "0")
	ok([]string{"fetch", "9"}, "0")
	fault([]string{"fetch", "abc"}, "codec::invalid_data")
	// A nested transaction attempt fails closed: the scope failure lands in
	// the decision's fallback arm, which rolls back with -1 instead of the
	// inner 7 the nested attempt would have committed.
	ok([]string{"nested"}, "-1")
	// A refused port surfaces the connect phase instead of a lazy handle.
	checks++
	status, out, diag := run(refused, "commit", strconv.Quote("Zed"))
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"sql::connection_failed"`) {
		t.Fatalf("live refusal: %d stdout=%q stderr=%s", status, redact(out), redact(diag))
	}
	t.Logf("live transactions: %d checks against %s", checks, strings.SplitN(version, " on ", 2)[0])
}

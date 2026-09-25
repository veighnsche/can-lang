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
)

const sqlMySQLManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{` +
	`"note_add":{"dialect":"mysql","statement":"INSERT INTO notes (id, body) VALUES (?, ?)","parameters":["id","body"],"parameter_type":"app::note_insert","row_type":"app::note_row","cardinality":"execute"},` +
	`"note_by_id":{"dialect":"mysql","statement":"SELECT id, body FROM notes WHERE id = ? LIMIT ?","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::note_row","cardinality":"one","row_limit_parameter":2}}}`

func writeMySQLProject(t *testing.T, root, manifest string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/sql/mysql.can"))
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

func TestCurrentMySQLPersistence(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged MySQL execution")
	}
	mysqlURL := os.Getenv("CAN_TEST_MYSQL_URL")
	if mysqlURL == "" {
		t.Skip("set CAN_TEST_MYSQL_URL for staged MySQL execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeMySQLProject(t, root, sqlMySQLManifest)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("mysql assert: %d %s %s", status, out, diag)
	}
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = home
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live build: %v %s", err, string(buildOut))
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil || build.Directory == "" {
		t.Fatalf("invalid live build report %v %s", err, string(buildOut))
	}
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/sql/mysql-seed.sql")
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/mysql-driver.ts")
	setupCmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, "setup", seed)
	setupCmd.Dir = home
	setupCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_MYSQL_URL=" + mysqlURL}
	setupOut, err := setupCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live setup: %v %s", err, string(setupOut))
	}
	var setup struct {
		ServerVersion string `json:"serverVersion"`
		Notes         string `json:"notes"`
	}
	if err := json.Unmarshal(setupOut, &setup); err != nil || setup.ServerVersion == "" {
		t.Fatalf("invalid live setup report %v %s", err, string(setupOut))
	}
	if setup.Notes != "0" {
		t.Fatalf("unexpected live seed notes=%s", setup.Notes)
	}
	// The compiled CLI reads its credential from the fd-3 environment
	// snapshot, exactly like production entries; the URL never appears
	// in argv or the process environment.
	snapshot := filepath.Join(home, "snapshot.json")
	snap, err := json.Marshal(map[string]string{"CAN_TEST_MYSQL": mysqlURL})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshot, snap, 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
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
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), stdout.String(), stderr.String()
		}
		return 0, stdout.String(), stderr.String()
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	// Every command opens its own pool: a row written by one process
	// is read back by the next, which proves server persistence across
	// separately opened connections rather than shared handles.
	variable := "CAN_TEST_MYSQL"
	ok([]string{"ping", variable}, "mysql-ok")
	ok([]string{"add", variable, "7", strconv.Quote("remember")}, "1")
	ok([]string{"get", variable, "7"}, `{"id":7,"body":"remember"}`)
	fault([]string{"get", variable, "999"}, "sql::row_missing")
	fault([]string{"add", variable, "8", strconv.Quote("remember")}, "sql::constraint_failed")
	ok([]string{"txcommit", variable, "8", strconv.Quote("kept")}, "1")
	ok([]string{"get", variable, "8"}, `{"id":8,"body":"kept"}`)
	ok([]string{"txabort", variable, "9", strconv.Quote("gone")}, "1")
	fault([]string{"get", variable, "9"}, "sql::row_missing")
	fault([]string{"get", variable, "abc"}, "codec::invalid_data")
	t.Logf("live mysql: %d checks against server %s", checks, setup.ServerVersion)
}

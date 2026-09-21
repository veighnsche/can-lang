package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const sqlDescriptorsManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{` +
	`"search_accounts":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i37_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},` +
	`"repeat_search":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i37_accounts WHERE display_name ILIKE $1 OR display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},` +
	`"account_by_id":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i37_accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2},` +
	`"add_account":{"dialect":"postgresql","statement":"INSERT INTO can_i37_accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"},` +
	`"commented_search":{"dialect":"postgresql","statement":"SELECT id FROM can_i37_accounts WHERE display_name ILIKE $1 /* hidden $9 */ LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2}}}`

func writeSQLDescriptorsProject(t *testing.T, root, manifest string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/sql/descriptors.can"))
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

func TestCurrentSQLDescriptors(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL descriptor execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-descriptors-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLDescriptorsProject(t, root, sqlDescriptorsManifest)
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
		t.Fatalf("descriptor assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 1 {
		t.Fatalf("invalid descriptor report %v %s", err, out)
	}
	status, _, diag = runAt(root, "run")
	if status != 0 || diag != "" {
		t.Fatalf("descriptor execution: %d %s", status, diag)
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
	writeSQLDescriptorsProject(t, moved, sqlDescriptorsManifest)
	if movedID, _ := buildAt(moved); movedID != firstID {
		t.Fatalf("relocated build changed identity %s %s", movedID, firstID)
	}
	state, err := os.ReadFile(filepath.Join(firstDir, "program/state.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"$canSQL=$canCreateSQLDescriptors(", `"search_accounts":{"cardinality":"many"`, `{"param":1}`, `"limit":2`, `"version":170007`} {
		if !strings.Contains(string(state), needle) {
			t.Fatalf("missing emitted descriptor %s", needle)
		}
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(firstDir, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	withDescriptor := func(statement, params, paramType, rowType, cardinal, limit string) string {
		limitField := `,"row_limit_parameter":` + limit
		if limit == "" {
			limitField = ""
		}
		return `{"source_root":"src","error_registry":"can.errors.json","sql":{"q":{"dialect":"postgresql","statement":` + quoteJSON(statement) + `,"parameters":` + params + `,"parameter_type":` + quoteJSON(paramType) + `,"row_type":` + quoteJSON(rowType) + `,"cardinality":` + quoteJSON(cardinal) + limitField + `}}}`
	}
	negatives := []struct {
		name     string
		manifest string
		want     string
	}{
		{"unbounded", withDescriptor("SELECT id FROM can_i37_accounts WHERE display_name ILIKE $1", `["term"]`, "app::search_parameters", "app::account_row", "many", "2"), "unbounded SELECT"},
		{"returning", withDescriptor("DELETE FROM can_i37_accounts RETURNING id", `[]`, "app::no_params", "app::account_row", "execute", ""), "RETURNING"},
		{"multi", withDescriptor("SELECT 1; SELECT 2", `[]`, "app::no_params", "app::account_row", "many", "1"), "2 statements"},
		{"gap", withDescriptor("SELECT $1 LIMIT $3", `["a","b"]`, "app::pair", "app::account_row", "many", "3"), "is not used"},
		{"extra", withDescriptor("SELECT $1 LIMIT $3", `["term"]`, "app::search_parameters", "app::account_row", "many", "2"), "no declared parameter"},
		{"shared limit", withDescriptor("SELECT id FROM can_i37_accounts WHERE id = $2 LIMIT $2", `["id"]`, "app::id_parameters", "app::account_row", "many", "2"), "appears 2 times"},
		{"literal limit", withDescriptor("SELECT id FROM can_i37_accounts LIMIT 10", `[]`, "app::no_params", "app::account_row", "one", "1"), "must be $1"},
		{"field order", withDescriptor("SELECT $1, $2 LIMIT $3", `["b","a"]`, "app::pair", "app::account_row", "many", "3"), "manifest lists"},
		{"non scalar", withDescriptor("SELECT $1 LIMIT $2", `["scores"]`, "app::bad_parameters", "app::account_row", "many", "2"), "non-SQL type"},
		{"unknown type", withDescriptor("SELECT $1 LIMIT $2", `["term"]`, "app::missing", "app::account_row", "many", "2"), "not declared"},
		{"execute limit", withDescriptor("DELETE FROM can_i37_accounts", `[]`, "app::no_params", "app::account_row", "execute", "1"), "cannot declare"},
		{"syntax", withDescriptor("SELECT FROM WHERE", `["term"]`, "app::search_parameters", "app::account_row", "many", "2"), "syntax error"},
	}
	for _, negative := range negatives {
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeSQLDescriptorsProject(t, dir, negative.manifest)
		status, out, diag := runAt(dir, "build")
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
}

func quoteJSON(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

// TestCurrentSQLDescriptorsLive exercises the compiled descriptors against a
// disposable PostgreSQL named by CAN_TEST_POSTGRES_URL. The test-only Bun
// driver imports the built program state, binds every query through the
// compiler-owned template path, and drops its table before exiting. It runs
// outside sandbox-exec because real parameterization needs loopback TCP; the
// URL (including any password) is never logged.
func TestCurrentSQLDescriptorsLive(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged SQL descriptor execution")
	}
	url := os.Getenv("CAN_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set CAN_TEST_POSTGRES_URL to a disposable PostgreSQL for live descriptor checks")
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
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "sql-descriptors-live")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLDescriptorsProject(t, root, sqlDescriptorsManifest)
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
	// The built state imports the launcher environment snapshot from fd 3;
	// an empty object satisfies the import without granting environment.
	snapshot := filepath.Join(home, "snapshot.json")
	if err := os.WriteFile(snapshot, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshotFile.Close()
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/sql/descriptors-driver.ts")
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/sql/descriptors.sql")
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, build.Directory, seed)
	cmd.Dir = home
	cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_POSTGRES_URL=" + url}
	cmd.ExtraFiles = []*os.File{snapshotFile}
	var out, diag bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diag
	if err := cmd.Run(); err != nil {
		t.Fatalf("live descriptors: %v stdout=%s stderr=%s", err, redact(out.String()), redact(diag.String()))
	}
	if diag.Len() != 0 {
		t.Fatalf("live descriptor noise: %s", redact(diag.String()))
	}
	var report struct {
		ServerVersion string `json:"serverVersion"`
		Checks        []struct {
			Name   string `json:"name"`
			OK     bool   `json:"ok"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("invalid live report %v %s", err, redact(out.String()))
	}
	if !strings.HasPrefix(report.ServerVersion, "PostgreSQL 17.") {
		t.Fatalf("unexpected live database %q", redact(report.ServerVersion))
	}
	if len(report.Checks) == 0 {
		t.Fatal("live driver reported no checks")
	}
	for _, check := range report.Checks {
		if !check.OK {
			t.Fatalf("live %s: %s", check.Name, redact(check.Detail))
		}
	}
	t.Logf("live descriptors: %d checks against %s", len(report.Checks), strings.SplitN(report.ServerVersion, " on ", 2)[0])
}

package catalogue

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestPackagedCatalogueBoundary(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE to run the offline packaged catalogue boundary")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Fatal("qualification requires the admitted macOS arm64 target")
	}
	sourceRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "catalogue-test")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := Builtin().ErrorIdentity("codec::invalid_data", nil)
	if err != nil {
		t.Fatal(err)
	}
	type occurrence struct {
		Kind         string            `json:"kind"`
		OccurrenceID string            `json:"occurrenceId"`
		Error        ErrorIdentity     `json:"error"`
		Payload      map[string]string `json:"payload"`
	}
	input := occurrence{Kind: "domain", OccurrenceID: "catalogue-fixture-17", Error: identity, Payload: map[string]string{"path": "/amount", "reason": "integer_token"}}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "bin/canlc"), "catalogue-check")
	command.Dir = t.TempDir()
	command.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
	command.Stdin = bytes.NewReader(data)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("packaged catalogue: %v\n%s", err, output)
	}
	var report struct {
		SchemaVersion int        `json:"schemaVersion"`
		Kind          string     `json:"kind"`
		Hash          string     `json:"catalogueSHA256"`
		Checks        []string   `json:"checks"`
		Occurrence    occurrence `json:"occurrence"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("invalid report: %v\n%s", err, output)
	}
	if report.SchemaVersion != 1 || report.Kind != "can.catalogue-conformance" || report.Hash != SourceHash() || len(report.Checks) != 9 || !reflect.DeepEqual(report.Occurrence, input) {
		t.Fatalf("Go/TS boundary mismatch: %s", output)
	}
	t.Logf("packaged Bun %s; native network denied; %d runtime checks; error %s, nominal identity, payload and occurrence retained; source %s", distribution.PinnedTarget().Runtime.Version, len(report.Checks), identity.Identity, report.Hash)
}

// TestCatalogueInclusionInventory is the I43 machine-checked inclusion
// gate: every catalogue operation names an owning implementation task
// with a closed assertion kind, every package carries at least one
// operation, type, or error unless reserved, and every owning task
// holds implementation evidence outside the Jev consultation
// directories. An included
// operation without evidence fails here instead of passing silently.
func TestCatalogueInclusionInventory(t *testing.T) {
	sourceRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler", "internal", "catalogue", "catalogue.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Packages []struct {
			Name string `json:"name"`
		} `json:"packages"`
		Types []struct {
			Identity string `json:"identity"`
		} `json:"types"`
		Errors []struct {
			Identity string `json:"identity"`
		} `json:"errors"`
		Operations []struct {
			Name      string `json:"name"`
			Identity  string `json:"identity"`
			Assertion string `json:"assertion"`
			Lowering  struct {
				Task string `json:"task"`
			} `json:"lowering"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) == 0 {
		t.Fatal("catalogue holds no operations")
	}
	covered := map[string]bool{}
	tasks := map[string][]string{}
	for _, op := range inventory.Operations {
		if op.Lowering.Task == "" {
			t.Errorf("%s names no owning task", op.Name)
		}
		switch op.Assertion {
		case "real", "supplied", "scoped":
		default:
			t.Errorf("%s carries assertion %q", op.Name, op.Assertion)
		}
		switch {
		case strings.Contains(op.Name, "::"):
			if pkg, _, ok := strings.Cut(op.Name, "::"); ok {
				covered[pkg] = true
			}
		case strings.HasPrefix(op.Name, "array.") || strings.HasPrefix(op.Name, "str.") || strings.HasPrefix(op.Name, "bytes."):
			if !strings.HasPrefix(op.Identity, "can.intrinsic.") {
				t.Errorf("%s is not an intrinsic method", op.Name)
			}
		case op.Name == "append":
			if op.Identity != "can.prelude@1::append" {
				t.Errorf("append carries identity %q", op.Identity)
			}
		default:
			t.Errorf("%s has no recognized name shape", op.Name)
		}
		tasks[op.Lowering.Task] = append(tasks[op.Lowering.Task], op.Name)
	}
	for _, typ := range inventory.Types {
		if id, _, ok := strings.Cut(typ.Identity, "::"); ok {
			if name, _, ok := strings.Cut(strings.TrimPrefix(id, "can.std."), "@"); ok {
				covered[name] = true
			}
		}
	}
	for _, failure := range inventory.Errors {
		if id, _, ok := strings.Cut(failure.Identity, "::"); ok {
			if name, _, ok := strings.Cut(strings.TrimPrefix(id, "can.std."), "@"); ok {
				covered[name] = true
			}
		}
	}
	// cli and json are intentionally reserved namespaces: the CLI
	// surface lives in io/env and the JSON surface in codec/bytes,
	// so neither package carries operations or types of its own.
	reserved := map[string]bool{"cli": true, "json": true}
	for _, pkg := range inventory.Packages {
		if !covered[pkg.Name] && !reserved[pkg.Name] {
			t.Errorf("package %s carries no operations, types, or errors", pkg.Name)
		}
	}
	evidence, err := filepath.Glob(filepath.Join(sourceRoot, "docs", "implementation", "evidence", "*", "i*"))
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]bool{}
	for _, path := range evidence {
		base := strings.ToLower(filepath.Base(path))
		if strings.Contains(base, "-jev") {
			continue
		}
		name := strings.TrimPrefix(base, "i")
		digits := ""
		for _, r := range name {
			if r < '0' || r > '9' {
				break
			}
			digits += string(r)
		}
		if digits == "" {
			continue
		}
		rest := name[len(digits):]
		if rest != "" && !strings.HasPrefix(rest, "-") && !strings.HasPrefix(rest, ".") {
			continue
		}
		if info, err := os.Stat(path); err != nil || (!info.IsDir() && filepath.Ext(path) != ".md" && filepath.Ext(path) != ".txt" && filepath.Ext(path) != ".json") {
			continue
		}
		held["I"+digits] = true
	}
	bunEvidence, err := filepath.Glob(filepath.Join(sourceRoot, "docs", "implementation", "evidence", "*", "b1-*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range bunEvidence {
		base := filepath.Base(path)
		if len(base) < 5 || base[0:3] != "b1-" || base[3] < '0' || base[3] > '9' || base[4] < '0' || base[4] > '9' {
			continue
		}
		rest := base[5:]
		if rest != "" && !strings.HasPrefix(rest, "-") && !strings.HasPrefix(rest, ".") {
			continue
		}
		if info, err := os.Stat(path); err != nil || (!info.IsDir() && filepath.Ext(path) != ".md" && filepath.Ext(path) != ".txt" && filepath.Ext(path) != ".json") {
			continue
		}
		held["B1-"+base[3:5]] = true
	}
	languageFixes, err := filepath.Glob(filepath.Join(sourceRoot, "docs", "implementation", "evidence", "*", "language-fixes", "LF??-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range languageFixes {
		base := filepath.Base(path)
		if len(base) >= 4 && base[0:2] == "LF" && base[2] >= '0' && base[2] <= '9' && base[3] >= '0' && base[3] <= '9' {
			held[base[0:4]] = true
		}
	}
	for task, ops := range tasks {
		if task == "" {
			continue
		}
		if !held[task] {
			t.Errorf("task %s holds no implementation evidence for %d operations (e.g. %s)", task, len(ops), ops[0])
		}
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Logf("%d operations across %d packages resolve to %d evidenced tasks", len(inventory.Operations), len(inventory.Packages), len(tasks))
}

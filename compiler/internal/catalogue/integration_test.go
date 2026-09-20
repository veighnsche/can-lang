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
	t.Logf("packaged Bun %s; native network denied; %d runtime checks; error ID %d, nominal identity, payload and occurrence retained; source %s", distribution.PinnedTarget().Runtime.Version, len(report.Checks), identity.ID, report.Hash)
}

package emit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONValueCodecFixture(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/codec/value.can")
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
	}
	for _, want := range []string{"$canJSONValue.decode", "$canJSONValue.encode", "$canCreateJSONValueCodec"} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		return
	}
	root := t.TempDir() // Go owns cleanup on every test exit.
	runtimeRoot, err := filepath.Abs("../../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(runtimeRoot, filepath.Join(root, "runtime")); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "runtime/") {
			continue
		}
		target := filepath.Join(root, artifact.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, artifact.Bytes, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "diagnostics"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "diagnostics/source-index.json"), []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[],"modules":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	environmentPath := filepath.Join(root, "environment.json")
	if err := os.WriteFile(environmentPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	environment, err := os.Open(environmentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer environment.Close()
	for i := range program.Assertions {
		if _, err := environment.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(ctx, bun, "--no-install", filepath.Join(root, "entry.ts"), fmt.Sprintf("root=%d", i))
		cmd.ExtraFiles = []*os.File{environment}
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("JSON value assertion %d: %v\n%s", i, err, output)
		}
	}
}

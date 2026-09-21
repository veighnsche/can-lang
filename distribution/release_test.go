package distribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSHA256Record(t *testing.T) {
	dir := t.TempDir()
	good := strings.Repeat("a", 64) + "  release.zip\n"
	path := filepath.Join(dir, "release.zip.sha256")
	if err := os.WriteFile(path, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	digest, err := ParseSHA256Record(path, "release.zip")
	if err != nil || digest != strings.Repeat("a", 64) {
		t.Fatalf("valid record rejected: %v", err)
	}
	for name, content := range map[string]string{
		"single space":  strings.Repeat("a", 64) + " release.zip\n",
		"wrong file":    strings.Repeat("a", 64) + "  other.zip\n",
		"short digest":  "abc  release.zip\n",
		"long digest":   strings.Repeat("a", 65) + "  release.zip\n",
		"uppercase hex": strings.Repeat("A", 64) + "  release.zip\n",
		"non-hex":       strings.Repeat("z", 64) + "  release.zip\n",
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			bad := filepath.Join(dir, "bad.sha256")
			if err := os.WriteFile(bad, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := ParseSHA256Record(bad, "release.zip"); err == nil {
				t.Fatal("accepted malformed record")
			}
		})
	}
	if _, err := ParseSHA256Record(filepath.Join(dir, "absent"), "release.zip"); err == nil {
		t.Fatal("accepted absent record")
	}
}

func TestReleaseRoundTrip(t *testing.T) {
	ctx := context.Background()
	bundle := syntheticBundle(t, t.TempDir(), "rel-1", archiveRuntime(t))
	out := t.TempDir()
	artifacts, err := Release(ctx, bundle, out)
	if err != nil {
		t.Fatal(err)
	}
	base := "can-rel-1-" + PinnedTarget().TargetID
	if filepath.Base(artifacts.Archive) != base+".zip" || filepath.Base(artifacts.SHA256) != base+".zip.sha256" || filepath.Base(artifacts.Inspection) != base+".inspection.json" {
		t.Fatalf("wrong artifact names: %+v", artifacts)
	}
	record, err := os.ReadFile(artifacts.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := ParseSHA256Record(artifacts.SHA256, base+".zip")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := hashFile(artifacts.Archive)
	if err != nil || actual != digest {
		t.Fatalf("sha record does not match archive: %v", err)
	}
	if !strings.HasSuffix(string(record), "\n") {
		t.Fatal("sha record missing trailing newline")
	}
	raw, err := os.ReadFile(artifacts.Inspection)
	if err != nil {
		t.Fatal(err)
	}
	var inspection Inspection
	if err := json.Unmarshal(raw, &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "can.release-inspection" || inspection.TargetID != PinnedTarget().TargetID || inspection.RuntimeSHA256 != PinnedTarget().Runtime.SHA256 {
		t.Fatalf("wrong inspection identity: %+v", inspection)
	}
	if inspection.Display == "" {
		t.Fatal("empty codesign display transcript")
	}
	// Identical bundles release identical bytes: entry order and timestamps
	// are fixed, so the digest is a pure function of content.
	again, err := Release(ctx, bundle, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	redigest, err := hashFile(again.Archive)
	if err != nil || redigest != digest {
		t.Fatal("release bytes not deterministic")
	}
	if _, err := Release(ctx, bundle, out); err == nil {
		t.Fatal("overwrote existing release artifacts")
	}
}

func TestReleaseRefusals(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	bundle := syntheticBundle(t, t.TempDir(), "rel-bad", runtimeBytes)
	renamed := filepath.Join(t.TempDir(), "renamed-dir")
	if err := os.Rename(bundle, renamed); err != nil {
		t.Fatal(err)
	}
	if _, err := Release(ctx, renamed, t.TempDir()); err == nil || !strings.Contains(err.Error(), "does not match manifest identity") {
		t.Fatalf("released a renamed bundle: %v", err)
	}
	tampered := syntheticBundle(t, t.TempDir(), "rel-tampered", runtimeBytes)
	if err := os.WriteFile(filepath.Join(tampered, "runtime/extra.ts"), []byte("tampered\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Release(ctx, tampered, t.TempDir()); err == nil || !strings.Contains(err.Error(), "modified asset") {
		t.Fatalf("released a tampered bundle: %v", err)
	}
	if _, err := Release(ctx, filepath.Join(t.TempDir(), "absent"), t.TempDir()); err == nil {
		t.Fatal("released an absent bundle")
	}
}

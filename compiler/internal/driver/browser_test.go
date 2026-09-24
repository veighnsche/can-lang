package driver

import (
	"context"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func TestBuildTargetRejectsUnknownTarget(t *testing.T) {
	runtime := &Runtime{}
	if _, err := runtime.BuildTarget(context.Background(), t.TempDir(), nil, nil, nil, DefaultAssertTimeoutMs, browser.Target("worker")); err == nil {
		t.Fatal("worker target admitted")
	}
}

func TestAppendBrowserAssetBindsModules(t *testing.T) {
	artifacts := []ir.Artifact{
		{Path: "runtime/r-0/domain.ts", Bytes: []byte("x\n"), Runtime: true},
		{Path: "browser.ts", Bytes: []byte("export const BROWSER_PROFILE = \"browser-main\";\n")},
		{Path: "program/state.ts", Bytes: []byte("export function $canInitialize(): void {}\n")},
	}
	sealed, err := appendBrowserAsset(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(sealed) != len(artifacts)+1 || sealed[len(sealed)-1].Path != browser.AssetPath {
		t.Fatalf("asset not appended: %v", sealed)
	}
	manifest, err := browser.ParseAsset(sealed[len(sealed)-1].Bytes)
	if err != nil {
		t.Fatalf("asset invalid: %v", err)
	}
	if len(manifest.Files) != 2 || manifest.Files["browser.ts"] == "" || manifest.Files["program/state.ts"] == "" {
		t.Fatalf("asset binds %+v", manifest.Files)
	}
	if err := browser.AuditArtifacts(sealed); err != nil {
		t.Fatalf("sealed graph rejected: %v", err)
	}
	if !strings.Contains(string(sealed[1].Bytes), "browser-main") {
		t.Fatal("entry lost the profile marker")
	}
}

package distribution

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRejectsBeforePublishing(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "invalid.zip")
	if err := os.WriteFile(archive, []byte("not the pinned archive"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"../escape", "ok"} {
		_, err := Build(context.Background(), ".", filepath.Join(root, "output"), archive, version)
		if err == nil {
			t.Fatal("accepted bad input")
		}
		if version == "ok" && !strings.Contains(err.Error(), "archive size or SHA-256 mismatch") {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "output")); !os.IsNotExist(err) {
		t.Fatal("published output for invalid input")
	}
}

func TestBuildRejectsSidecarDestinationCollision(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for archive-backed collision regression")
	}
	for _, name := range []string{"bun", "BUN"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source")
			output := filepath.Join(root, "output")
			for _, dir := range []string{"runtime", "tools/runtime"} {
				if err := os.MkdirAll(filepath.Join(source, dir), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(source, "runtime", name), []byte("must never replace the verified sidecar"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "tools/runtime/tsconfig.json"), []byte("{}"), 0644); err != nil {
				t.Fatal(err)
			}
			path, err := Build(context.Background(), source, output, archive, "collision-test")
			if err == nil || path != "" || !strings.Contains(err.Error(), "bundle destination collision") {
				t.Fatalf("expected collision refusal, got %q: %v", path, err)
			}
			entries, err := os.ReadDir(output)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("failed build left published/staged data: %v", entries)
			}
		})
	}
}

func TestBuildRejectsTamperedHTMX(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for archive-backed tamper regression")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	output := filepath.Join(root, "output")
	for _, dir := range []string{"runtime", "tools/runtime", "distribution/notices", "distribution/assets"} {
		if err := os.MkdirAll(filepath.Join(source, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "tools/runtime/tsconfig.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("assets/htmx.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "distribution/assets/htmx.lock.json"), lock, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "distribution/assets/htmx-4.0.0.min.js"), []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	guard, err := os.ReadFile("assets/htmx-guard.js")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "distribution/assets/htmx-guard.js"), guard, 0644); err != nil {
		t.Fatal(err)
	}
	notice, err := os.ReadFile("notices/htmx-LICENSE.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "distribution/notices/htmx-LICENSE.txt"), notice, 0644); err != nil {
		t.Fatal(err)
	}
	path, err := Build(context.Background(), source, output, archive, "tamper-test")
	if err == nil || path != "" || !strings.Contains(err.Error(), "htmx script") {
		t.Fatalf("expected htmx refusal, got %q: %v", path, err)
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed build left published/staged data: %v", entries)
	}
}

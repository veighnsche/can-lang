package distribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInspectRuntime(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bun")
	if err := os.WriteFile(path, archiveRuntime(t), 0755); err != nil {
		t.Fatal(err)
	}
	target := PinnedTarget()
	if target.Runtime.Platform == "linux" {
		if _, err := exec.LookPath("readelf"); err != nil {
			t.Skip("readelf unavailable for Linux inspection transcripts")
		}
	}
	inspection, err := InspectRuntime(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.SchemaVersion != 1 || inspection.Kind != "can.release-inspection" || inspection.TargetID != target.TargetID {
		t.Fatalf("wrong inspection identity: %+v", inspection)
	}
	if inspection.Runtime != target.Runtime.Executable || inspection.RuntimeSHA256 != target.Runtime.SHA256 {
		t.Fatalf("inspection does not bind the pin: %+v", inspection)
	}
	if inspection.HostOS != runtime.GOOS || inspection.HostArch != runtime.GOARCH {
		t.Fatalf("wrong host record: %+v", inspection)
	}
	if target.Runtime.Platform == "linux" {
		if !strings.Contains(inspection.Display, "ELF64") || !strings.Contains(inspection.Display, "X86-64") {
			t.Fatalf("display transcript missing ELF facts: %q", inspection.Display)
		}
		if !strings.Contains(inspection.Entitlements, "INTERP") {
			t.Fatalf("program-header transcript missing loader facts: %q", inspection.Entitlements)
		}
		return
	}
	if !strings.Contains(inspection.Display, "Identifier=") || !strings.Contains(inspection.Display, "Mach-O") {
		t.Fatalf("display transcript missing signature facts: %q", inspection.Display)
	}
}

func TestInspectRuntimeRefusals(t *testing.T) {
	ctx := context.Background()
	foreign := filepath.Join(t.TempDir(), "other")
	if err := os.WriteFile(foreign, []byte("not the runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectRuntime(ctx, foreign); err == nil || !strings.Contains(err.Error(), "not the pinned executable") {
		t.Fatalf("inspected foreign bytes: %v", err)
	}
	if _, err := InspectRuntime(ctx, filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("inspected an absent file")
	}
}

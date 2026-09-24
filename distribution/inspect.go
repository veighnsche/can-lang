package distribution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Inspection records read-only upstream observations about one release's
// pinned runtime. It is evidence, never a gate: verification binds bytes
// through hashes, and this transcript only shows what the local platform
// tools report about those same bytes.
type Inspection struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	TargetID      string `json:"targetId"`
	Runtime       string `json:"runtime"`
	RuntimeSHA256 string `json:"runtimeSHA256"`
	HostOS        string `json:"hostOS"`
	HostArch      string `json:"hostArch"`
	Display       string `json:"display"`
	Entitlements  string `json:"entitlements"`
}

// InspectRuntime observes the pinned runtime executable with the platform's
// read-only display tools: codesign on macOS, readelf on Linux. It runs no
// network fetch, trusts no publisher, and refuses nothing about the
// content: callers file the transcript.
func InspectRuntime(ctx context.Context, runtimePath string) (Inspection, error) {
	target := PinnedTarget()
	abs, err := filepath.Abs(runtimePath)
	if err != nil {
		return Inspection{}, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect runtime: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Inspection{}, fmt.Errorf("inspect runtime: not a regular file")
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect runtime: %w", err)
	}
	if Hash(data) != target.Runtime.SHA256 {
		return Inspection{}, fmt.Errorf("inspect runtime: not the pinned executable")
	}
	display, entitlements := "", ""
	if target.Runtime.Platform == "linux" {
		// The file header binds class/machine; the program headers show
		// the glibc loader INTERP the T19 target requires.
		if display, err = toolText(ctx, "readelf", "-W", "-h", abs); err != nil {
			return Inspection{}, err
		}
		if entitlements, err = toolText(ctx, "readelf", "-W", "-l", abs); err != nil {
			return Inspection{}, err
		}
	} else {
		if display, err = toolText(ctx, "codesign", "-dvvv", abs); err != nil {
			return Inspection{}, err
		}
		if entitlements, err = toolText(ctx, "codesign", "-d", "--entitlements", ":-", abs); err != nil {
			return Inspection{}, err
		}
	}
	return Inspection{
		SchemaVersion: 1,
		Kind:          "can.release-inspection",
		TargetID:      target.TargetID,
		Runtime:       target.Runtime.Executable,
		RuntimeSHA256: target.Runtime.SHA256,
		HostOS:        runtime.GOOS,
		HostArch:      runtime.GOARCH,
		Display:       display,
		Entitlements:  entitlements,
	}, nil
}

func toolText(ctx context.Context, name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("inspect runtime: %s unavailable: %w", name, err)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent"}
	out, err := cmd.CombinedOutput()
	text := strings.TrimRight(string(out), "\n")
	if err != nil {
		return "", fmt.Errorf("inspect runtime: %s: %v\n%s", name, err, text)
	}
	return text, nil
}

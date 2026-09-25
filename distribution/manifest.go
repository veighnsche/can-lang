// Package distribution builds the local development sidecar bundle.
package distribution

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed assets/htmx-4.0.0.min.js
var HTMXScript []byte

// GuardScript embeds the generated guard bytes. The file is derived from
// runtime/platform/htmx-guard.ts by tools/runtime/guardgen.ts; do not edit
// it by hand.
//
//go:embed assets/htmx-guard.js
var GuardScript []byte

//go:embed target.json
var TargetJSON []byte

//go:embed target-linux-amd64.json
var TargetLinuxJSON []byte

type Target struct {
	SchemaVersion int    `json:"schemaVersion"`
	TargetID      string `json:"targetId"`
	Runtime       struct {
		Name             string `json:"name"`
		Version          string `json:"version"`
		Revision         string `json:"revision"`
		Platform         string `json:"platform"`
		Architecture     string `json:"architecture"`
		MinimumOSVersion string `json:"minimumOSVersion"`
		Executable       string `json:"executable"`
		SHA256           string `json:"sha256"`
	} `json:"runtime"`
	Upstream struct {
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
		Member string `json:"member"`
	} `json:"upstream"`
}

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	Version       string            `json:"version"`
	TargetID      string            `json:"targetId"`
	Files         map[string]string `json:"files"`
}

func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HTMXLock pins the one reviewed upstream browser script. The bundle build
// verifies every field before publication; no later stage downloads code.
type HTMXLock struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Package       string   `json:"package"`
	Version       string   `json:"version"`
	Origins       []string `json:"origins"`
	File          string   `json:"file"`
	Integrity     string   `json:"integrity"`
	SHA256        string   `json:"sha256"`
	Size          int      `json:"size"`
	Route         string   `json:"route"`
	MediaType     string   `json:"mediaType"`
	License       struct {
		SPDX   string `json:"spdx"`
		Source string `json:"source"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"license"`
}

// VerifyHTMX checks the vendored script, its license, and its lock against
// the P11 contract. It reports the first mismatch without publishing.
func VerifyHTMX(source string) error {
	entries, err := os.ReadDir(filepath.Join(source, "distribution", "assets"))
	if err != nil {
		return fmt.Errorf("htmx assets: %w", err)
	}
	if len(entries) != 3 {
		return fmt.Errorf("htmx assets: closed inventory holds the script, its lock, and the generated guard")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
	}
	if !seen["htmx-4.0.0.min.js"] || !seen["htmx.lock.json"] || !seen["htmx-guard.js"] {
		return fmt.Errorf("htmx assets: closed inventory holds the script, its lock, and the generated guard")
	}
	raw, err := os.ReadFile(filepath.Join(source, "distribution", "assets", "htmx.lock.json"))
	if err != nil {
		return fmt.Errorf("htmx lock: %w", err)
	}
	var lock HTMXLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return fmt.Errorf("htmx lock: %w", err)
	}
	if lock.SchemaVersion != 1 || lock.Kind != "can.browser-runtime" || lock.Package != "htmx.org" || lock.Version != "4.0.0" {
		return fmt.Errorf("htmx lock: unexpected pinned package")
	}
	if lock.File != "htmx-4.0.0.min.js" || lock.Route != "/__can/assets/htmx-4.0.0.min.js" || lock.MediaType != "text/javascript" {
		return fmt.Errorf("htmx lock: unexpected route contract")
	}
	if len(lock.Origins) < 2 {
		return fmt.Errorf("htmx lock: cross-checked origins required")
	}
	script, err := os.ReadFile(filepath.Join(source, "distribution", "assets", lock.File))
	if err != nil {
		return fmt.Errorf("htmx script: %w", err)
	}
	if len(script) != lock.Size || Hash(script) != lock.SHA256 || !bytes.Equal(script, HTMXScript) {
		return fmt.Errorf("htmx script: size or SHA-256 mismatch")
	}
	sum := sha512.Sum384(script)
	if "sha384-"+base64.StdEncoding.EncodeToString(sum[:]) != lock.Integrity {
		return fmt.Errorf("htmx script: integrity mismatch")
	}
	if lock.Integrity != "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc" {
		return fmt.Errorf("htmx script: P11 digest mismatch")
	}
	notice, err := os.ReadFile(filepath.Join(source, lock.License.Path))
	if err != nil || Hash(notice) != lock.License.SHA256 {
		return fmt.Errorf("htmx license: changed or missing notice")
	}
	if lock.License.SPDX == "" || lock.License.Source == "" {
		return fmt.Errorf("htmx license: provenance required")
	}
	guard, err := os.ReadFile(filepath.Join(source, "distribution", "assets", "htmx-guard.js"))
	if err != nil {
		return fmt.Errorf("guard script: %w", err)
	}
	if !bytes.Equal(guard, GuardScript) {
		return fmt.Errorf("guard script: embeds stale bytes; run bun tools/runtime/guardgen.ts")
	}
	if _, err := GuardAsset(); err != nil {
		return err
	}
	return nil
}

// HTMXAsset returns the embedded 4.0.0 script after checking the P11 digest.
// Callers serve these bytes; they do not download a replacement.
func HTMXAsset() ([]byte, error) {
	script := append([]byte(nil), HTMXScript...)
	if Hash(script) != "e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f" {
		return nil, fmt.Errorf("htmx script: SHA-256 mismatch")
	}
	sum := sha512.Sum384(script)
	if "sha384-"+base64.StdEncoding.EncodeToString(sum[:]) != "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc" {
		return nil, fmt.Errorf("htmx script: P11 digest mismatch")
	}
	return script, nil
}

// GuardAsset returns the embedded generated guard script after checking its
// digests. The sha384 literal is the P11-style integrity pin shared with
// runtimeHead in runtime/platform/html.ts; re-pin together after any guard
// edit and regeneration. Callers serve these bytes; they do not
// retranspile a replacement.
func GuardAsset() ([]byte, error) {
	script := append([]byte(nil), GuardScript...)
	if Hash(script) != "4d7098c2660af8affaf4693c38f7b91ede6632193f7a1b2bc45731ed7d603da0" {
		return nil, fmt.Errorf("guard script: SHA-256 mismatch")
	}
	sum := sha512.Sum384(script)
	if "sha384-"+base64.StdEncoding.EncodeToString(sum[:]) != "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG" {
		return nil, fmt.Errorf("guard script: P11 digest mismatch")
	}
	return script, nil
}

func mustTarget(raw []byte) Target {
	var target Target
	if err := json.Unmarshal(raw, &target); err != nil {
		panic(err)
	}
	return target
}

// LinuxTarget returns the pinned Debian 13 amd64/glibc runtime record. It
// is selected only on that host; builds never cross targets.
func LinuxTarget() Target {
	return mustTarget(TargetLinuxJSON)
}

// SupportedTargets lists every admitted host target ID for diagnostics.
func SupportedTargets() []string {
	return []string{mustTarget(TargetJSON).TargetID, LinuxTarget().TargetID}
}

// HostSupported reports whether the current host is an admitted target.
func HostSupported() bool {
	return (runtime.GOOS == "darwin" && runtime.GOARCH == "arm64") ||
		(runtime.GOOS == "linux" && runtime.GOARCH == "amd64")
}

func PinnedTarget() Target {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		return LinuxTarget()
	}
	return mustTarget(TargetJSON)
}

// PinnedTargetJSON returns the pinned manifest bytes for this host. Bundles
// embed and verify these exact bytes; only the admitted host target is
// ever published.
func PinnedTargetJSON() []byte {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		return TargetLinuxJSON
	}
	return TargetJSON
}

// Shared build-once bundle cache for the integration suite.
//
// Every integration test needs a staged toolchain bundle, but the bundle
// is a pure function of its inputs: the pinned upstream archive, the
// source trees distribution.Build consumes, and the Go toolchain that
// compiles the launcher. Building one bundle per test re-proves the same
// toolchain ~90 times per suite run; this harness builds it once per
// unique input key and shares the verified result.
//
// Sharing is safe because tests already keep their mutable state (staged
// projects, homes, databases, ports, dist output) in per-test temp dirs;
// the bundle itself is only executed from. Each cache hit re-verifies the
// bundle manifest's per-file hashes, so a test that writes into the
// shared bundle is detected and the entry is rebuilt instead of trusted.
// Tests whose subject IS the build (release determinism, distribution
// refusals) keep calling distribution.Build directly.
//
// Cache entries live under $CAN_TEST_CACHE (default os.TempDir +
// "/can-test-cache") and survive across `go test` processes. Fills are
// serialized by an inter-process lock directory; stale locks expire.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// harnessSharedVersion is the one bundle version every cached test shares.
// No converted test asserts on the embedded version string; the
// build-subject tests that do keep their own direct builds.
const harnessSharedVersion = "harness-shared"

// harnessKeyDirs covers every source tree distribution.Build consumes:
// staged assets (runtime, tools, distribution), the launcher sources
// (compiler), and the module files pinning the launcher build.
var harnessKeyDirs = []string{"compiler", "distribution", "runtime", "tools"}

var (
	harnessMu    sync.Mutex
	harnessPaths = map[string]string{}
	harnessKeys  = map[string]string{}

	// harnessBuilds counts real cache fills in this process. Tests use it
	// to prove sharing; it is not a limit.
	harnessBuilds atomic.Int64

	// heavySlots bounds concurrently running heavyweight tests (browser
	// + servers + builds, ~2GB each). Four of them coincide on a 16GB
	// host and the suite collapses into swap with multi-10s stalls;
	// light tests run free within the go -parallel budget.
	heavySlots = make(chan struct{}, heavySlotCap())
)

// heavySlotCap sizes the heavyweight-test semaphore: 3 by default,
// CAN_TEST_HEAVY_SLOTS to tune for the host. Minimum 1.
func heavySlotCap() int {
	if raw := os.Getenv("CAN_TEST_HEAVY_SLOTS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
	}
	return 3
}

// acquireHeavy takes a heavyweight slot, released at test end. Call it
// right after t.Parallel: acquiring before the pause would deadlock the
// sequential phase, and Cleanup releases on pass, fail, or skip.
func acquireHeavy(t *testing.T) {
	t.Helper()
	acquireFrom(t, heavySlots)
}

func acquireFrom(t *testing.T, slots chan struct{}) {
	t.Helper()
	slots <- struct{}{}
	t.Cleanup(func() { <-slots })
}

// harnessBundle returns the shared staged toolchain bundle for the given
// upstream archive, building it once on cache miss. Callers keep using
// per-test temp dirs for everything mutable.
func harnessBundle(t *testing.T, ctx context.Context, archive string) (string, error) {
	t.Helper()
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		return "", err
	}
	key := harnessKey(t, sourceRoot, archive)
	root := harnessRoot()
	entry := filepath.Join(root, "bundle-"+key)
	// Memos are per cache root: the same content key in two roots
	// names two different entries.
	memo := root + "\x00" + key

	harnessMu.Lock()
	if path, ok := harnessPaths[memo]; ok {
		harnessMu.Unlock()
		// A contaminated memo falls through to fill, which rebuilds.
		if harnessEntryComplete(path, key) && harnessVerify(path) == nil {
			return path, nil
		}
		harnessMu.Lock()
		delete(harnessPaths, memo)
		harnessMu.Unlock()
	} else {
		harnessMu.Unlock()
	}

	path, err := harnessFill(ctx, sourceRoot, archive, root, entry, key)
	if err != nil {
		return "", err
	}
	harnessMu.Lock()
	harnessPaths[memo] = path
	harnessMu.Unlock()
	return path, nil
}

// harnessRoot is the persistent cross-process cache directory.
func harnessRoot() string {
	if root := os.Getenv("CAN_TEST_CACHE"); root != "" {
		return root
	}
	return filepath.Join(os.TempDir(), "can-test-cache")
}

// harnessKey hashes every input distribution.Build consumes, so any change
// to the toolchain inputs (or the Go toolchain compiling the launcher)
// yields a fresh entry instead of a stale bundle.
func harnessKey(t *testing.T, sourceRoot, archive string) string {
	t.Helper()
	// Toolchain inputs cannot change mid-process, so hash once per
	// (source, archive) pair instead of once per test.
	memo := sourceRoot + "\x00" + archive
	harnessMu.Lock()
	if key, ok := harnessKeys[memo]; ok {
		harnessMu.Unlock()
		return key
	}
	harnessMu.Unlock()
	hashes := treeHashes(t, sourceRoot, harnessKeyDirs)
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		hashes[name] = distribution.Hash(data)
	}
	archiveData, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	hashes["@archive"] = distribution.Hash(archiveData)
	hashes["@go"] = runtime.Version()
	hashes["@version"] = harnessSharedVersion
	names := make([]string, 0, len(hashes))
	for name := range hashes {
		names = append(names, name)
	}
	sort.Strings(names)
	var sb strings.Builder
	for _, name := range names {
		sb.WriteString(name)
		sb.WriteString("=")
		sb.WriteString(hashes[name])
		sb.WriteString("\n")
	}
	key := distribution.Hash([]byte(sb.String()))
	harnessMu.Lock()
	harnessKeys[memo] = key
	harnessMu.Unlock()
	return key
}

// harnessFill returns the verified entry, building it under an
// inter-process lock on miss or waiting for a concurrent filler.
func harnessFill(ctx context.Context, sourceRoot, archive, root, entry, key string) (string, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	if harnessEntryComplete(entry, key) && harnessVerify(entry) == nil {
		return entry, nil
	}
	lock := entry + ".lock"
	deadline := time.Now().Add(30 * time.Minute)
	for {
		if err := os.Mkdir(lock, 0755); err == nil {
			defer os.RemoveAll(lock)
			// A verified entry may have appeared while waiting.
			if harnessEntryComplete(entry, key) && harnessVerify(entry) == nil {
				return entry, nil
			}
			return harnessBuild(ctx, sourceRoot, archive, root, entry, key)
		}
		// Another process is filling; wait for its marker.
		if harnessWaitForEntry(ctx, entry, key, deadline) {
			return entry, nil
		}
		harnessReapStaleLock(lock)
		if ctx.Err() != nil {
			return "", fmt.Errorf("harness cache fill wait: %w", ctx.Err())
		}
	}
}

// harnessWaitForEntry polls for a concurrent filler's marker. It reports
// whether a verified entry is now available.
func harnessWaitForEntry(ctx context.Context, entry, key string, deadline time.Time) bool {
	for time.Now().Before(deadline) {
		if harnessEntryComplete(entry, key) && harnessVerify(entry) == nil {
			return true
		}
		select {
		case <-ctx.Done():
			return harnessEntryComplete(entry, key) && harnessVerify(entry) == nil
		case <-time.After(500 * time.Millisecond):
		}
	}
	return false
}

// harnessReapStaleLock removes a lock directory older than the fill
// deadline so a killed filler cannot wedge the cache forever.
func harnessReapStaleLock(lock string) {
	info, err := os.Stat(lock)
	if err != nil {
		return
	}
	if time.Since(info.ModTime()) > 30*time.Minute {
		os.RemoveAll(lock)
	}
}

// harnessBuild constructs the bundle in a staging directory and publishes
// it atomically, then verifies the published entry before returning it.
func harnessBuild(ctx context.Context, sourceRoot, archive, root, entry, key string) (string, error) {
	stage, err := os.MkdirTemp(root, ".fill-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	built, err := distribution.Build(ctx, sourceRoot, stage, archive, harnessSharedVersion)
	if err != nil {
		return "", err
	}
	os.RemoveAll(entry)
	if err := os.Rename(built, entry); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(entry, "cache-complete"), []byte(key+"\n"), 0644); err != nil {
		return "", err
	}
	if err := harnessVerify(entry); err != nil {
		return "", err
	}
	harnessBuilds.Add(1)
	return entry, nil
}

// harnessEntryComplete reports whether the entry carries this key's marker.
func harnessEntryComplete(entry, key string) bool {
	raw, err := os.ReadFile(filepath.Join(entry, "cache-complete"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == key
}

// harnessVerify re-hashes every manifest-listed file in a cached entry. A
// test that contaminates the shared bundle is detected here and the entry
// is rebuilt on the next fill instead of trusted.
func harnessVerify(entry string) error {
	raw, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest distribution.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.development-distribution" || len(manifest.Files) == 0 {
		return fmt.Errorf("unexpected bundle manifest in %s", entry)
	}
	for name, want := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(entry, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if distribution.Hash(data) != want {
			return fmt.Errorf("cached bundle file %s failed verification", name)
		}
	}
	return nil
}

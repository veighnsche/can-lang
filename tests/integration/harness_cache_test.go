package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

// The cache key must be deterministic for identical inputs: the whole
// suite shares one bundle per key, so instability would rebuild every test.
func TestHarnessKeyStable(t *testing.T) {
	t.Parallel()
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(archive, []byte("fake-archive-bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	first := harnessKey(t, sourceRoot, archive)
	second := harnessKey(t, sourceRoot, archive)
	if first == "" || first != second {
		t.Fatalf("unstable harness key %q vs %q", first, second)
	}
}

// Any toolchain input change must rotate the key so no test ever runs
// against a stale shared bundle.
func TestHarnessKeySensitive(t *testing.T) {
	t.Parallel()
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	before := filepath.Join(dir, "before.zip")
	after := filepath.Join(dir, "after.zip")
	if err := os.WriteFile(before, []byte("archive-a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(after, []byte("archive-b"), 0600); err != nil {
		t.Fatal(err)
	}
	if harnessKey(t, sourceRoot, before) == harnessKey(t, sourceRoot, after) {
		t.Fatal("harness key ignores archive bytes")
	}
}

func writeFakeBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	entry := t.TempDir()
	manifest := distribution.Manifest{
		SchemaVersion: 1,
		Kind:          "can.development-distribution",
		Version:       harnessSharedVersion,
		TargetID:      "test",
		Files:         map[string]string{},
	}
	for name, body := range files {
		p := filepath.Join(entry, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		manifest.Files[name] = distribution.Hash([]byte(body))
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "manifest.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return entry
}

// A pristine entry verifies; this is the per-hit fast path.
func TestHarnessVerifyClean(t *testing.T) {
	t.Parallel()
	entry := writeFakeBundle(t, map[string]string{"bin/canlc": "launcher", "runtime/bun": "runtime"})
	if err := harnessVerify(entry); err != nil {
		t.Fatalf("clean entry failed verification: %v", err)
	}
}

// A test that writes into the shared bundle must be detected, never
// trusted: contamination fails verification so the entry is rebuilt.
func TestHarnessVerifyContaminated(t *testing.T) {
	t.Parallel()
	entry := writeFakeBundle(t, map[string]string{"bin/canlc": "launcher"})
	if err := os.WriteFile(filepath.Join(entry, "bin", "canlc"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := harnessVerify(entry); err == nil {
		t.Fatal("contaminated entry passed verification")
	}
}

func TestHarnessEntryComplete(t *testing.T) {
	t.Parallel()
	entry := t.TempDir()
	if harnessEntryComplete(entry, "k") {
		t.Fatal("missing marker reports complete")
	}
	if err := os.WriteFile(filepath.Join(entry, "cache-complete"), []byte("other\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if harnessEntryComplete(entry, "k") {
		t.Fatal("foreign marker reports complete")
	}
	if err := os.WriteFile(filepath.Join(entry, "cache-complete"), []byte("k\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !harnessEntryComplete(entry, "k") {
		t.Fatal("matching marker reports incomplete")
	}
}

// The heavyweight semaphore admits up to its cap and parks the rest;
// a release unparks exactly one waiter. Uses an isolated channel so the
// test never interferes with the suite's own semaphore.
func TestHeavySlotsBounded(t *testing.T) {
	slots := make(chan struct{}, 2)
	acquireFrom(t, slots)
	acquireFrom(t, slots)
	proceeded := make(chan struct{})
	waiting := make(chan struct{})
	go func() {
		close(waiting)
		// No Cleanup here: the waiter must not consume a suite slot,
		// and this goroutine is not a test.
		slots <- struct{}{}
		close(proceeded)
	}()
	<-waiting
	select {
	case <-proceeded:
		t.Fatal("third acquisition proceeded past a cap-2 semaphore")
	case <-time.After(100 * time.Millisecond):
	}
	<-slots
	select {
	case <-proceeded:
	case <-time.After(5 * time.Second):
		t.Fatal("release did not unpark the waiter")
	}
	// The two acquireFrom Cleanups drain the remaining sends at test end.
}

func TestHeavySlotCap(t *testing.T) {
	if got := heavySlotCap(); got != 3 {
		t.Fatalf("default heavy cap %d, want 3", got)
	}
	t.Setenv("CAN_TEST_HEAVY_SLOTS", "7")
	if got := heavySlotCap(); got != 7 {
		t.Fatalf("override heavy cap %d, want 7", got)
	}
	t.Setenv("CAN_TEST_HEAVY_SLOTS", "0")
	if got := heavySlotCap(); got != 3 {
		t.Fatalf("invalid heavy cap %d, want default 3", got)
	}
}

// End-to-end sharing proof: two calls, at most one real fill, same path.
// Runs only with the pinned archive, like every other staged test.
func TestHarnessSharedEndToEnd(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for harness sharing proof")
	}
	t.Setenv("CAN_TEST_CACHE", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	before := harnessBuilds.Load()
	first, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	second, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("shared bundle paths differ: %q vs %q", first, second)
	}
	if got := harnessBuilds.Load() - before; got != 1 {
		t.Fatalf("two shared-bundle calls performed %d fills, want 1", got)
	}
}

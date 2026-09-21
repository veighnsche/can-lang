package distribution

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestUpdatePreservesRunningVersion(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	out := t.TempDir()
	first, err := Release(ctx, syntheticBundle(t, t.TempDir(), "upd-1", runtimeBytes), out)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Release(ctx, syntheticBundle(t, t.TempDir(), "upd-2", runtimeBytes), out)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	previous, current, err := Update(ctx, first.Archive, first.SHA256, root)
	if err != nil {
		t.Fatal(err)
	}
	if previous != "" || !strings.HasSuffix(current, "can-upd-1-"+PinnedTarget().TargetID) {
		t.Fatalf("first update selected wrong: %q -> %q", previous, current)
	}
	// A running process holds the old runtime open across the update.
	held, err := os.Open(filepath.Join(current, "runtime/bun"))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	oldBytes, err := os.ReadFile(filepath.Join(current, "runtime/bun"))
	if err != nil {
		t.Fatal(err)
	}
	previous, current, err = Update(ctx, second.Archive, second.SHA256, root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(previous, "can-upd-1-"+PinnedTarget().TargetID) || !strings.HasSuffix(current, "can-upd-2-"+PinnedTarget().TargetID) {
		t.Fatalf("update moved wrong: %q -> %q", previous, current)
	}
	// The old version is byte-identical and still verifies; the held
	// descriptor kept working throughout.
	if _, err := held.Stat(); err != nil {
		t.Fatal(err)
	}
	kept, err := os.ReadFile(filepath.Join(previous, "runtime/bun"))
	if err != nil || Hash(kept) != Hash(oldBytes) {
		t.Fatal("old version changed under update")
	}
	if _, err := VerifyBundle(previous); err != nil {
		t.Fatalf("old version no longer verifies: %v", err)
	}
	if _, err := VerifyBundle(current); err != nil {
		t.Fatalf("new version does not verify: %v", err)
	}
}

func TestUpdateFailurePreservesSelection(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	out := t.TempDir()
	good, err := Release(ctx, syntheticBundle(t, t.TempDir(), "upd-good", runtimeBytes), out)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, _, err := Update(ctx, good.Archive, good.SHA256, root); err != nil {
		t.Fatal(err)
	}
	before, err := Selection(root)
	if err != nil {
		t.Fatal(err)
	}
	bad := syntheticBundle(t, t.TempDir(), "upd-bad", runtimeBytes)
	if err := os.WriteFile(filepath.Join(bad, "runtime/extra.ts"), []byte("tampered\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Release refuses the tampered tree, so install it raw: craft a release
	// of the good tree, then swap in tampered bytes under a forged record.
	data, err := os.ReadFile(good.Archive)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)/2] ^= 0xff
	forged := filepath.Join(out, "forged.zip")
	if err := os.WriteFile(forged, data, 0644); err != nil {
		t.Fatal(err)
	}
	forgedSHA := filepath.Join(out, "forged.zip.sha256")
	if err := os.WriteFile(forgedSHA, []byte(Hash(data)+"  forged.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Update(ctx, forged, forgedSHA, root); err == nil {
		t.Fatal("tampered update succeeded")
	}
	after, err := Selection(root)
	if err != nil || after != before {
		t.Fatalf("failed update moved selection %q -> %q: %v", before, after, err)
	}
	if _, err := VerifyBundle(after); err != nil {
		t.Fatalf("previous version damaged: %v", err)
	}
}

func TestConcurrentInstallUpdate(t *testing.T) {
	ctx := context.Background()
	runtimeBytes := archiveRuntime(t)
	out := t.TempDir()
	const versions = 6
	type artifact struct{ archive, sha string }
	artifacts := make([]artifact, versions)
	for i := 0; i < versions; i++ {
		release, err := Release(ctx, syntheticBundle(t, t.TempDir(), "conc-"+string(rune('a'+i)), runtimeBytes), out)
		if err != nil {
			t.Fatal(err)
		}
		artifacts[i] = artifact{release.Archive, release.SHA256}
	}
	root := t.TempDir()
	var wg sync.WaitGroup
	errs := make(chan error, versions*2)
	// Installs race updates of the same versions: exactly one lands each
	// version, and already-exists is an accepted outcome on either side.
	for i := 0; i < versions; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := Install(ctx, artifacts[i].archive, artifacts[i].sha, root); err != nil && !strings.Contains(err.Error(), "already exists") {
				errs <- err
			}
		}(i)
	}
	for i := 0; i < versions; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, err := Update(ctx, artifacts[i].archive, artifacts[i].sha, root)
			// Updates race installs of the same version: already-exists is
			// an accepted outcome, anything else is a defect.
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent install/update failed: %v", err)
	}
	for i := 0; i < versions; i++ {
		version := filepath.Join(root, "versions", "can-conc-"+string(rune('a'+i))+"-"+PinnedTarget().TargetID)
		if _, err := VerifyBundle(version); err != nil {
			t.Fatalf("version %d damaged: %v", i, err)
		}
	}
	selected, err := Selection(root)
	if err != nil || selected == "" {
		t.Fatalf("no selection after concurrent updates: %v", err)
	}
	if _, err := VerifyBundle(selected); err != nil {
		t.Fatalf("selected version does not verify: %v", err)
	}
}

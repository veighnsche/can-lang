package tempcache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func candidate(t *testing.T, parent string, pid int, old bool) string {
	t.Helper()
	path, err := os.MkdirTemp(parent, prefix)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(owner{Schema: schema, Name: filepath.Base(path), PID: pid})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, markerName), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, leaseName), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if old {
		before := time.Now().Add(-2 * recoveryAge)
		if err := os.Chtimes(path, before, before); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

const deadPID = 999999999

func TestClosePreservesParentAndConcurrentCache(t *testing.T) {
	parent := t.TempDir()
	first, err := Open(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Open(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	firstPath := first.Path
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("owned cache retained: %v", err)
	}
	if _, err := os.Stat(second.Path); err != nil {
		t.Fatalf("concurrent cache removed: %v", err)
	}
	if _, err := os.Stat(parent); err != nil {
		t.Fatalf("parent removed: %v", err)
	}
}

func TestRecoveryRequiresInactivityAndPreservesForeignPaths(t *testing.T) {
	if !errors.Is(syscall.Kill(deadPID, 0), syscall.ESRCH) {
		t.Fatal("test dead PID is not absent")
	}
	parent := t.TempDir()
	abandoned := candidate(t, parent, deadPID, true)
	fresh := candidate(t, parent, deadPID, false)
	live := candidate(t, parent, os.Getpid(), true)
	locked := candidate(t, parent, deadPID, true)
	file, err := os.OpenFile(filepath.Join(locked, leaseName), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	foreign := candidate(t, parent, deadPID, true)
	if err := os.WriteFile(filepath.Join(foreign, markerName), []byte(`{"schema":"foreign"}`), 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(parent, prefix+"symlink")
	if err := os.Symlink(abandoned, linked); err != nil {
		t.Fatal(err)
	}
	check := func(paths []string) error {
		if len(paths) != 1 || paths[0] != abandoned {
			t.Fatalf("unsafe recovery candidates: %v", paths)
		}
		return nil
	}
	if err := recoverAbandoned(parent, time.Now(), check); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abandoned); !os.IsNotExist(err) {
		t.Fatalf("abandoned cache retained: %v", err)
	}
	for _, path := range []string{fresh, live, locked, foreign, linked} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("protected path removed %s: %v", path, err)
		}
	}
}

func TestUnavailableLivenessRetainsOrphan(t *testing.T) {
	parent := t.TempDir()
	path := candidate(t, parent, deadPID, true)
	if err := recoverAbandoned(parent, time.Now(), func([]string) error { return errors.New("lsof unavailable") }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("inconclusive liveness deleted cache: %v", err)
	}
}

func TestRecoveryRejectsSymlinkMarker(t *testing.T) {
	parent := t.TempDir()
	path := candidate(t, parent, deadPID, true)
	marker := filepath.Join(path, markerName)
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "foreign-marker")
	if err := os.WriteFile(outside, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, marker); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * recoveryAge)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if err := recoverAbandoned(parent, time.Now(), func([]string) error { t.Fatal("symlink reached liveness check"); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryWorkIsBounded(t *testing.T) {
	parent := t.TempDir()
	for i := 0; i < recoveryLimit+1; i++ {
		candidate(t, parent, deadPID, true)
	}
	if err := recoverAbandoned(parent, time.Now(), func(paths []string) error {
		if len(paths) != recoveryLimit {
			t.Fatalf("recovery count %d, want %d", len(paths), recoveryLimit)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	remaining, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 {
		t.Fatalf("unbounded recovery: %d remain", len(remaining))
	}
}

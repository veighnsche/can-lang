package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageSharesIdenticalBytes(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	_, firstDir := mustStage(t, s, "export const value=1n;")
	secondID, secondDir := mustStage(t, s, "export const value=2n;")
	_ = secondID
	sharedA := filepath.Join(firstDir, "entry.ts")
	sharedB := filepath.Join(secondDir, "entry.ts")
	if same, err := sameStagedFile(sharedA, sharedB); err != nil || !same {
		t.Fatal("identical staged bytes do not share an inode", err)
	}
	uniqueA := filepath.Join(firstDir, "packages/p-a/a.ts")
	uniqueB := filepath.Join(secondDir, "packages/p-a/a.ts")
	if same, err := sameStagedFile(uniqueA, uniqueB); err != nil || same {
		t.Fatal("distinct staged bytes share an inode", err)
	}
	manifestA := filepath.Join(firstDir, "manifest.json")
	manifestB := filepath.Join(secondDir, "manifest.json")
	if same, err := sameStagedFile(manifestA, manifestB); err != nil || same {
		t.Fatal("per-build manifests share an inode", err)
	}
	for _, p := range []string{sharedA, uniqueA} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0400 {
			t.Fatalf("staged content %s is writable: %v", p, info.Mode())
		}
		if file, err := os.OpenFile(p, os.O_WRONLY, 0); err == nil {
			file.Close()
			t.Fatalf("in-place staged mutation succeeds on %s", p)
		}
	}
}

func TestBreakLinkTamperIsolatesGenerations(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	firstID, _ := mustStage(t, s, "export const value=1n;")
	secondID, secondDir := mustStage(t, s, "export const value=2n;")
	victim := filepath.Join(secondDir, "entry.ts")
	data, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	data[0] ^= 0x01
	if err := os.Remove(victim); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(victim, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.generation(secondID, false); err == nil {
		t.Fatal("tampered generation still validates")
	}
	if _, _, err := s.generation(firstID, false); err != nil {
		t.Fatal("honest tamper simulation corrupted the linked generation", err)
	}
}

func TestPruneCollectsUnreferencedStore(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := outputPrepared(t, s, "export const value=1n;")
	if _, err := s.Publish(first); err != nil {
		t.Fatal(err)
	}
	second := outputPrepared(t, s, "export const value=2n;")
	if _, err := s.Publish(second); err != nil {
		t.Fatal(err)
	}
	sharedDigest := hashBytes([]byte("import './packages/p-a/a.ts';\n"))
	uniqueDigest := hashBytes([]byte("export const value=1n;"))
	cas := filepath.Join(root, "dist", "cas", sharedDigest)
	if _, err := os.Stat(cas); err != nil {
		t.Fatal("shared bytes missing from the content store", err)
	}
	if err := s.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "cas", uniqueDigest)); !os.IsNotExist(err) {
		t.Fatal("prune kept bytes of a removed generation", err)
	}
	if _, err := os.Stat(cas); err != nil {
		t.Fatal("prune collected bytes the current generation references", err)
	}
}

func mustStage(t *testing.T, s *OutputStore, text string) (string, string) {
	t.Helper()
	prepared := outputPrepared(t, s, text)
	id, directory, err := s.Stage(prepared)
	if err != nil {
		t.Fatal(err)
	}
	return id, directory
}

func sameStagedFile(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(infoA, infoB), nil
}

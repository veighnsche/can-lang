package reference

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func shaOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// fixtureProposal writes a minimal proposal plus matching source and pins
// trees. runtimeRoots maps runtime root -> file contents.
func fixtureProposal(t *testing.T, runtimeRoots map[string]map[string]string) (source, pins, proposal string) {
	t.Helper()
	base := t.TempDir()
	source = filepath.Join(base, "source")
	pins = filepath.Join(base, "pins")
	writeFile(t, filepath.Join(source, "catalogue.json"), "catalogue")
	writeFile(t, filepath.Join(source, "target.json"), "target")
	writeFile(t, filepath.Join(pins, "bun.zip"), "archive")
	writeFile(t, filepath.Join(pins, "bun"), "binary")
	roots := []string{}
	files := 0
	var bytes int64
	for root, members := range runtimeRoots {
		roots = append(roots, root)
		for name, content := range members {
			writeFile(t, filepath.Join(source, root, name), content)
			files++
			bytes += int64(len(content))
		}
	}
	proposal = filepath.Join(base, "proposal.json")
	doc := map[string]any{
		"status": "proposed",
		"head":   "abc123",
		"inputs": map[string]any{
			"catalogue":       map[string]any{"path": "catalogue.json", "sha256": shaOf("catalogue")},
			"target_manifest": map[string]any{"path": "target.json", "sha256": shaOf("target")},
			"bun_archive":     map[string]any{"path": "bun.zip", "sha256": shaOf("archive")},
			"bun_executable":  map[string]any{"path": "bun", "sha256": shaOf("binary")},
			"runtime_tree":    map[string]any{"roots": roots, "files": files + ignoredStrays, "bytes": bytes + 30},
		},
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, proposal, string(encoded))
	return source, pins, proposal
}

func TestSelectFilesAcceptsPristineFixtures(t *testing.T) {
	source, pins, proposal := fixtureProposal(t, map[string]map[string]string{
		"runtime": {"a.ts": "aaa", "b.ts": "bb"},
	})
	sel, err := SelectFiles(source, pins, proposal)
	if err != nil {
		t.Fatalf("SelectFiles: %v", err)
	}
	if sel.Head != "abc123" || sel.RuntimeCount != 2 || len(sel.RuntimeFiles) != 2 {
		t.Fatalf("selection = %+v", sel)
	}
	if sel.RuntimeFiles[0].SHA256 != shaOf("aaa") && sel.RuntimeFiles[1].SHA256 != shaOf("aaa") {
		t.Fatalf("runtime hashes = %+v", sel.RuntimeFiles)
	}
}

func TestSelectFilesRejectsTamperedCatalogue(t *testing.T) {
	source, pins, proposal := fixtureProposal(t, map[string]map[string]string{
		"runtime": {"a.ts": "aaa"},
	})
	writeFile(t, filepath.Join(source, "catalogue.json"), "tampered")
	if _, err := SelectFiles(source, pins, proposal); err == nil {
		t.Fatal("tampered catalogue accepted")
	} else if !strings.Contains(err.Error(), "catalogue") {
		t.Fatalf("err = %v, want catalogue identity", err)
	}
}

func TestSelectFilesRejectsStrayRuntimeFile(t *testing.T) {
	source, pins, proposal := fixtureProposal(t, map[string]map[string]string{
		"runtime": {"a.ts": "aaa"},
	})
	writeFile(t, filepath.Join(source, "runtime", "stray.ts"), "stray")
	if _, err := SelectFiles(source, pins, proposal); err == nil {
		t.Fatal("stray runtime file accepted")
	} else if !strings.Contains(err.Error(), "file count") {
		t.Fatalf("err = %v, want file count", err)
	}
}

func TestSelectFilesRejectsMissingPin(t *testing.T) {
	source, pins, proposal := fixtureProposal(t, map[string]map[string]string{
		"runtime": {"a.ts": "aaa"},
	})
	if err := os.Remove(filepath.Join(pins, "bun.zip")); err != nil {
		t.Fatal(err)
	}
	if _, err := SelectFiles(source, pins, proposal); err == nil {
		t.Fatal("missing archive accepted")
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "x"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, out, err)
		}
	}
}

func gitHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestVerifyPristine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"), "x")
	gitInit(t, dir)
	head := gitHead(t, dir)
	if err := VerifyPristine(dir, head); err != nil {
		t.Fatalf("pristine rejected: %v", err)
	}
	if err := VerifyPristine(dir, "deadbeef"); err == nil {
		t.Fatal("wrong head accepted")
	}
	writeFile(t, filepath.Join(dir, "f.txt"), "dirty")
	if err := VerifyPristine(dir, head); err == nil {
		t.Fatal("dirty tree accepted")
	}
}

func TestVerifyPristineRejectsUntracked(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"), "x")
	gitInit(t, dir)
	head := gitHead(t, dir)
	writeFile(t, filepath.Join(dir, "stray.txt"), "stray")
	if err := VerifyPristine(dir, head); err == nil {
		t.Fatal("untracked file accepted")
	}
}

func TestDistbuildDriftFree(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tools/distbuild/main.go"), "package main")
	writeFile(t, filepath.Join(dir, "other.txt"), "x")
	gitInit(t, dir)
	head := gitHead(t, dir)
	// Unrelated drift plus committed change under tools/distbuild.
	writeFile(t, filepath.Join(dir, "other.txt"), "y")
	cmd := exec.Command("git", "-C", dir, "commit", "-qam", "y")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", out, err)
	}
	if err := DistbuildDriftFree(dir, head); err != nil {
		t.Fatalf("unrelated drift rejected: %v", err)
	}
	writeFile(t, filepath.Join(dir, "tools/distbuild/main.go"), "package main // drift")
	cmd = exec.Command("git", "-C", dir, "commit", "-qam", "drift")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", out, err)
	}
	if err := DistbuildDriftFree(dir, head); err == nil {
		t.Fatal("bootstrap drift accepted")
	}
}

func TestSealWritesManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.json"), `{"Files":{"runtime/a.ts":"abc"}}`)
	writeFile(t, filepath.Join(root, "bin/canlc"), "binary")
	sel := Selection{Head: "abc123"}
	path, err := Seal(root, sel, SealInput{Version: "r-seed-1", GoVersion: "go1", Host: "h", OwnerUID: 1, OwnerGID: 2, StagedUTC: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.BundleFiles["bin/canlc"] != shaOf("binary") || manifest.BundleFiles["runtime/a.ts"] != "abc" {
		t.Fatalf("bundle files = %+v", manifest.BundleFiles)
	}
	if manifest.StagedUTC != "1970-01-01T00:00:00Z" || manifest.OwnerUID != 1 {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestSealRejectsMissingBinary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.json"), `{"Files":{"runtime/a.ts":"abc"}}`)
	if _, err := Seal(root, Selection{}, SealInput{Version: "v"}); err == nil {
		t.Fatal("missing canlc sealed")
	}
}

func TestDisposeRemovesAndReceipts(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "stage")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(target, "f"), "x")
	receipt, err := DisposeWorktree(target, 1, 2)
	if err != nil {
		t.Fatalf("DisposeWorktree: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("target survives disposal")
	}
	receiptPath := filepath.Join(dir, "receipt.json")
	if err := WriteReceipt(receiptPath, receipt); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Receipt
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Path != target || decoded.OwnerUID != 1 {
		t.Fatalf("receipt = %+v", decoded)
	}
	if _, err := DisposeSeed("", 1, 2); err == nil {
		t.Fatal("empty path disposal accepted")
	}
}

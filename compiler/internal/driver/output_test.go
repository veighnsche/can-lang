package driver

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func outputProject(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[],"retired":[]}`, "src/main.can": "package app\n    provides []\n    uses []\nint value = 1\n"} {
		p := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func outputPrepared(t *testing.T, store *OutputStore, text string) *PreparedOutput {
	t.Helper()
	items := artifactFixture()
	items[1].Bytes = []byte(text)
	p, err := PrepareOutput(store.BuildInputs(strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64)), "entry.ts", items)
	if err != nil {
		t.Fatal(err)
	}
	p.validated = true // Filesystem tests inject the already-validated compiler boundary.
	return p
}
func outputBegin(t *testing.T, root string) *OutputStore {
	t.Helper()
	s, err := BeginOutput(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOwnedOutputPublicationAndLeases(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := outputPrepared(t, s, "export const value=1n;")
	directory, err := s.Publish(first)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Publish(first)
	if err != nil || again != directory {
		t.Fatal("identical build was not reused", err)
	}
	if _, err = BeginOutput(root); err == nil {
		t.Fatal("concurrent writer admitted")
	}
	lease, err := s.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	second := outputPrepared(t, s, "export const value=2n;")
	if _, err = s.Publish(second); err != nil {
		t.Fatal(err)
	}
	if err = s.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); err != nil {
		t.Fatal("active generation pruned", err)
	}
	lease.Close()
	if err = s.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("inactive generation retained", err)
	}
	current, err := s.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	if current.Manifest.BuildID != second.BuildID() {
		t.Fatal("wrong current build")
	}
}
func TestOutputRefusesUnownedSymlinkedAndChangedInputs(t *testing.T) {
	for _, mode := range []string{"unowned", "symlink", "wrong owner", "source overlap"} {
		t.Run(mode, func(t *testing.T) {
			root := outputProject(t)
			switch mode {
			case "unowned":
				os.Mkdir(filepath.Join(root, "dist"), 0700)
				os.WriteFile(filepath.Join(root, "dist", "keep.txt"), []byte("keep"), 0600)
			case "symlink":
				if err := os.Symlink(filepath.Join(root, "src"), filepath.Join(root, "dist")); err != nil {
					t.Fatal(err)
				}
			case "wrong owner":
				os.Mkdir(filepath.Join(root, "dist"), 0700)
				os.WriteFile(filepath.Join(root, "dist", ".can-owner.json"), []byte(`{"schemaVersion":1,"kind":"can.output-owner","project":"other"}`), 0600)
			case "source overlap":
				os.Rename(filepath.Join(root, "src"), filepath.Join(root, "dist"))
				os.WriteFile(filepath.Join(root, "can.project.json"), []byte(`{"source_root":"dist","error_registry":"can.errors.json"}`), 0600)
			}
			if s, err := BeginOutput(root); err == nil {
				s.Close()
				t.Fatal("unsafe output admitted")
			}
			if mode == "unowned" {
				data, err := os.ReadFile(filepath.Join(root, "dist", "keep.txt"))
				if err != nil || string(data) != "keep" {
					t.Fatal("unknown data damaged")
				}
			}
		})
	}
	root := outputProject(t)
	s := outputBegin(t, root)
	os.WriteFile(filepath.Join(root, "src", "main.can"), []byte("package app\n    provides []\n    uses []\nint value = 2\n"), 0600)
	if _, err := s.Publish(outputPrepared(t, s, "x")); err == nil || !strings.Contains(err.Error(), "inputs changed") {
		t.Fatal("stale snapshot published", err)
	}
}
func TestInterruptedPublicationRecovery(t *testing.T) {
	for _, point := range []string{"reserved", "file:entry.ts", "staged", "generation", "published"} {
		t.Run(point, func(t *testing.T) {
			root := outputProject(t)
			s := outputBegin(t, root)
			first := outputPrepared(t, s, "one")
			if _, err := s.Publish(first); err != nil {
				t.Fatal(err)
			}
			second := outputPrepared(t, s, "two")
			s.testHook = func(at string) error {
				if at == point {
					return errors.New("simulated interruption")
				}
				return nil
			}
			if _, err := s.Publish(second); err == nil {
				t.Fatal("injection did not fire")
			}
			lease, err := s.AcquireCurrent()
			if err != nil {
				t.Fatal(err)
			}
			expected := first.BuildID()
			if point == "published" {
				expected = second.BuildID()
			}
			if lease.Manifest.BuildID != expected {
				t.Fatal("partial generation became current")
			}
			lease.Close()
			s.Close()
			reopened := outputBegin(t, root)
			if err = reopened.Recover(); err != nil {
				t.Fatal(err)
			}
			if _, err = reopened.Publish(second); err != nil {
				t.Fatal(err)
			}
			if err = reopened.Prune(); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(filepath.Join(root, "dist", "builds"))
			if err != nil || len(entries) != 1 || entries[0].Name() != second.BuildID() {
				t.Fatal("orphan recovery failed", entries, err)
			}
		})
	}
}
func TestOutputPreservesUnexpectedGenerationContents(t *testing.T) {
	for _, mode := range []string{"unknown file", "unknown directory", "modified file", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			root := outputProject(t)
			s := outputBegin(t, root)
			first := outputPrepared(t, s, "one")
			dir, err := s.Publish(first)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Publish(outputPrepared(t, s, "two")); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, "keep.txt")
			switch mode {
			case "unknown file":
				os.WriteFile(target, []byte("keep"), 0600)
			case "unknown directory":
				os.Mkdir(target, 0700)
			case "modified file":
				os.WriteFile(filepath.Join(dir, "entry.ts"), []byte("keep"), 0600)
			case "symlink":
				os.Symlink(filepath.Join(root, "src"), target)
			}
			if err = s.Prune(); err == nil {
				t.Fatal("unsafe tree cleanup admitted")
			}
			if _, err = os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
				t.Fatal("refusal partially deleted generation")
			}
			if mode != "modified file" {
				if _, err = os.Lstat(target); err != nil {
					t.Fatal("unknown entry removed")
				}
			}
		})
	}
}
func TestCurrentManifestIsAuthoritative(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	if _, err := s.Publish(outputPrepared(t, s, "one")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "dist", "current.json")
	raw, _ := os.ReadFile(file)
	var current outputCurrent
	json.Unmarshal(raw, &current)
	current.ManifestSHA256 = strings.Repeat("b", 64)
	changed, _ := json.Marshal(current)
	os.WriteFile(file, changed, 0600)
	if _, err := s.AcquireCurrent(); err == nil {
		t.Fatal("tampered current metadata admitted")
	}
}

func TestCleanRebuildAndSnapshotBinding(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := outputPrepared(t, s, "one")
	if _, err := s.Publish(first); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Clean(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcquireCurrent(); !os.IsNotExist(err) {
		t.Fatal("clean left a current pointer", err)
	}
	if _, err = os.Stat(lease.Directory); err != nil {
		t.Fatal("clean destroyed live generation", err)
	}
	lease.Close()
	if err = s.Prune(); err != nil {
		t.Fatal(err)
	}
	rebuilt := outputPrepared(t, s, "one")
	if rebuilt.BuildID() != first.BuildID() {
		t.Fatal("clean changed semantic identity")
	}
	if _, err = s.Publish(rebuilt); err != nil {
		t.Fatal(err)
	}
	wrong, err := PrepareOutput(artifactInputs(), "entry.ts", artifactFixture())
	if err != nil {
		t.Fatal(err)
	}
	wrong.validated = true
	if _, err = s.Publish(wrong); err == nil {
		t.Fatal("generation from unrelated source snapshot published")
	}
	unvalidated := outputPrepared(t, s, "invalid")
	unvalidated.validated = false
	if _, err = s.Publish(unvalidated); err == nil {
		t.Fatal("native syntax validation bypassed")
	}
}

func TestInterruptedStageUnknownFilesArePreserved(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	s.testHook = func(point string) error {
		if point == "file:entry.ts" {
			return errors.New("interrupt")
		}
		return nil
	}
	if _, err := s.Publish(outputPrepared(t, s, "one")); err == nil {
		t.Fatal("missing interruption")
	}
	raw, err := os.ReadFile(filepath.Join(root, "dist", "pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pending outputPending
	if err = json.Unmarshal(raw, &pending); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(root, "dist", filepath.FromSlash(pending.Stage), "notes.txt")
	if err = os.WriteFile(unknown, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.Recover(); err == nil {
		t.Fatal("unknown staged file removed")
	}
	if data, err := os.ReadFile(unknown); err != nil || string(data) != "keep" {
		t.Fatal("unknown stage data lost")
	}
}

func TestOwnerRevalidationAndInputAssets(t *testing.T) {
	t.Run("owner changed after opening", func(t *testing.T) {
		root := outputProject(t)
		s := outputBegin(t, root)
		p := outputPrepared(t, s, "one")
		dir, err := s.Publish(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, "dist", ".can-owner.json"), []byte(`{"schemaVersion":1,"kind":"can.output-owner","project":"other"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err = s.Clean(); err == nil {
			t.Fatal("changed owner admitted")
		}
		if _, err = os.Stat(filepath.Join(dir, "entry.ts")); err != nil {
			t.Fatal("changed-owner output damaged")
		}
	})
	t.Run("asset mutation invalidates snapshot", func(t *testing.T) {
		root := outputProject(t)
		os.WriteFile(filepath.Join(root, "can.project.json"), []byte(`{"source_root":"src","error_registry":"can.errors.json","assets":{"logo":"logo.svg"}}`), 0600)
		os.WriteFile(filepath.Join(root, "logo.svg"), []byte("before"), 0600)
		s := outputBegin(t, root)
		p := outputPrepared(t, s, "one")
		os.WriteFile(filepath.Join(root, "logo.svg"), []byte("after"), 0600)
		if _, err := s.Publish(p); err == nil || !strings.Contains(err.Error(), "inputs changed") {
			t.Fatal("changed asset published", err)
		}
	})
	for _, name := range []string{"..", "../project"} {
		if store, err := BeginOutput(name); err == nil {
			store.Close()
			t.Fatal("parent component admitted")
		}
	}
}

func TestSourceRemovalPublishesOnlyNewGraph(t *testing.T) {
	root := outputProject(t)
	os.WriteFile(filepath.Join(root, "src", "extra.can"), []byte("package app\n    provides []\n    uses []\nint extra = 2\n"), 0600)
	s := outputBegin(t, root)
	old := outputPrepared(t, s, "export const extra=2n;")
	oldDir, err := s.Publish(old)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	os.Remove(filepath.Join(root, "src", "extra.can"))
	s = outputBegin(t, root)
	next := outputPrepared(t, s, "export const value=1n;")
	if next.BuildID() == old.BuildID() {
		t.Fatal("source removal omitted from build identity")
	}
	if _, err = s.Publish(next); err != nil {
		t.Fatal(err)
	}
	if err = s.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatal("stale generation retained without a lease")
	}
}

func TestProjectLockAcrossProcesses(t *testing.T) {
	if root := os.Getenv("CAN_TEST_OUTPUT_LOCK_ROOT"); root != "" {
		store, err := BeginOutput(root)
		if err == nil {
			store.Close()
			t.Fatal("second process acquired project lock")
		}
		if !strings.Contains(err.Error(), "locked") {
			t.Fatal(err)
		}
		return
	}
	root := outputProject(t)
	s := outputBegin(t, root)
	command := exec.Command(os.Args[0], "-test.run", "^TestProjectLockAcrossProcesses$", "-test.v")
	command.Env = append(os.Environ(), "CAN_TEST_OUTPUT_LOCK_ROOT="+root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	s.Close()
	reopened := outputBegin(t, root)
	if reopened.Graph.Root.Root != root {
		t.Fatal("lock did not release")
	}
}

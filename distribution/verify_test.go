package distribution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtLeastVersion(t *testing.T) {
	cases := []struct {
		actual, minimum string
		want            bool
	}{
		{"14.2.1", "13.0", true},
		{"13.0", "13.0", true},
		{"13.0.1", "13.0", true},
		{"12.9", "13.0", false},
		{"13", "13.0.1", false},
		{"27.0", "13.0", true},
		{"bogus", "13.0", false},
		{"13.0", "bogus", false},
	}
	for _, tc := range cases {
		if got := atLeastVersion(tc.actual, tc.minimum); got != tc.want {
			t.Fatalf("atLeastVersion(%q, %q) = %v, want %v", tc.actual, tc.minimum, got, tc.want)
		}
	}
}

func TestVerifyBundleAcceptsSynthetic(t *testing.T) {
	root := syntheticBundle(t, t.TempDir(), "verify-ok", archiveRuntime(t))
	manifest, err := VerifyBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "verify-ok" || manifest.Kind != "can.development-distribution" {
		t.Fatalf("wrong manifest identity: %+v", manifest)
	}
}

func TestVerifyBundleRefusals(t *testing.T) {
	runtimeBytes := archiveRuntime(t)
	mutate := func(t *testing.T, change func(root string)) error {
		t.Helper()
		root := syntheticBundle(t, t.TempDir(), "verify-bad", runtimeBytes)
		change(root)
		_, err := VerifyBundle(root)
		return err
	}
	cases := []struct {
		name   string
		change func(root string)
		want   string
	}{
		{"missing manifest", func(root string) { os.Remove(filepath.Join(root, "manifest.json")) }, "missing asset manifest.json"},
		{"malformed manifest", func(root string) { os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{oops"), 0644) }, "invalid manifest"},
		{"wrong kind", func(root string) { rewrite(t, root, "can.other-kind") }, "unsupported distribution identity"},
		{"modified asset", func(root string) { os.WriteFile(filepath.Join(root, "runtime/extra.ts"), []byte("tampered\n"), 0644) }, "modified asset runtime/extra.ts"},
		{"unknown file", func(root string) { os.WriteFile(filepath.Join(root, "stowaway.txt"), []byte("x"), 0644) }, "unknown file stowaway.txt"},
		{"symlinked asset", func(root string) {
			os.Remove(filepath.Join(root, "runtime/extra.ts"))
			os.Symlink("extra.ts", filepath.Join(root, "runtime", "extra.ts"))
		}, "symlinked asset runtime/extra.ts"},
		{"tampered runtime", func(root string) {
			data, _ := os.ReadFile(filepath.Join(root, "runtime/bun"))
			data[100] ^= 0xff
			os.WriteFile(filepath.Join(root, "runtime/bun"), data, 0755)
		}, "pinned Bun hash mismatch"},
		{"non-executable runtime", func(root string) { os.Chmod(filepath.Join(root, "runtime/bun"), 0644) }, "not executable"},
		{"wrong target", func(root string) { os.WriteFile(filepath.Join(root, "distribution/target.json"), []byte("{}"), 0644) }, "target differs from compiled pin"},
		{"unstamped launcher", func(root string) { os.WriteFile(filepath.Join(root, "bin/canlc"), []byte("no stamp here"), 0755) }, "launcher stamp does not match"},
		{"missing launcher", func(root string) { os.Remove(filepath.Join(root, "bin/canlc")) }, "missing asset bin/canlc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutate(t, tc.change)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := VerifyBundle(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("verified an absent directory")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBundle(file); err == nil {
		t.Fatal("verified a regular file")
	}
}

func rewrite(t *testing.T, root, kind string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixed := strings.Replace(string(raw), `"kind": "can.development-distribution"`, `"kind": "`+kind+`"`, 1)
	if fixed == string(raw) {
		t.Fatal("kind replacement missed")
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(fixed), 0644); err != nil {
		t.Fatal(err)
	}
}

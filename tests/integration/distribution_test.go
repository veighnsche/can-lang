package integration

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestDevelopmentSidecar(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE to the pinned local archive to run offline distribution integration")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Fatal("admitted integration target is macOS arm64")
	}
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sourceBefore := treeHashes(t, source, []string{"compiler", "distribution", "runtime", "tools"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root, err := distribution.Build(ctx, source, t.TempDir(), archive, "integration")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := distribution.Build(ctx, source, filepath.Dir(root), archive, "integration"); err == nil {
		t.Fatal("overwrote existing version")
	}
	parent := t.TempDir()
	home := filepath.Join(parent, "home")
	cwd := filepath.Join(parent, "unrelated")
	for _, p := range []string{home, cwd} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	poison := filepath.Join(parent, "poison.ts")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(poison, `throw new Error("AMBIENT_PRELOAD_EXECUTED")`)
	for _, p := range []string{filepath.Join(home, ".bunfig.toml"), filepath.Join(cwd, "bunfig.toml")} {
		write(p, "preload = [\""+poison+"\"]\n")
	}
	write(filepath.Join(cwd, ".env"), "CAN_DISTRIBUTION_PROBE=poisoned\n")
	write(filepath.Join(cwd, "tsconfig.json"), "invalid ambient config")
	link := filepath.Join(parent, "linked-canlc")
	if err := os.Symlink(filepath.Join(root, "bin/canlc"), link); err != nil {
		t.Fatal(err)
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + home, "PATH=/nonexistent", "CAN_DISTRIBUTION_PROBE=preserved", "NODE_OPTIONS=--import=" + poison, "BUN_OPTIONS=--preload=" + poison, "BUN_CONFIG_MAX_HTTP_REQUESTS=invalid", "CAN_PRIVATE_SECRET=never-print-this"}
	run := func() ([]byte, error) {
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", link, "runtime-check", "space argument", "--not-a-bun-flag")
		command.Dir = cwd
		command.Env = env
		return command.CombinedOutput()
	}
	bundleBefore := treeHashes(t, root, []string{"."})
	output, err := run()
	if err != nil {
		t.Fatalf("offline bundle run: %v\n%s", err, output)
	}
	var result struct {
		Bun                 string   `json:"bun"`
		Executable          string   `json:"executable"`
		EnvironmentProbe    string   `json:"environmentProbe"`
		Arguments           []string `json:"arguments"`
		OriginalHomePresent bool     `json:"originalHomePresent"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	expectedExecutable, _ := filepath.EvalSymlinks(filepath.Join(root, "runtime/bun"))
	if result.Bun != "1.4.2" || result.Executable != expectedExecutable || result.EnvironmentProbe != "preserved" || !result.OriginalHomePresent || !reflect.DeepEqual(result.Arguments, []string{"space argument", "--not-a-bun-flag"}) {
		t.Fatalf("wrong runtime result: %s", output)
	}
	if strings.Contains(string(output), "never-print-this") {
		t.Fatal("environment secret leaked")
	}
	if !reflect.DeepEqual(bundleBefore, treeHashes(t, root, []string{"."})) {
		t.Fatal("execution wrote to bundle")
	}
	if !reflect.DeepEqual(sourceBefore, treeHashes(t, source, []string{"compiler", "distribution", "runtime", "tools"})) {
		t.Fatal("build/run wrote to source tree")
	}
	bunPath := filepath.Join(root, "runtime/bun")
	original, err := os.ReadFile(bunPath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, code string
		change     func()
	}{
		{"missing", "CAN-DIST-MISSING", func() {
			if err := os.Remove(bunPath); err != nil {
				t.Fatal(err)
			}
		}},
		{"non-executable", "CAN-DIST-PERMISSION", func() {
			if err := os.Chmod(bunPath, 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"tampered", "CAN-DIST-INTEGRITY", func() {
			data := append([]byte(nil), original...)
			data[len(data)-1] ^= 1
			if err := os.WriteFile(bunPath, data, 0755); err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong-architecture", "CAN-DIST-ARCH", func() {
			data := append([]byte(nil), original...)
			binary.LittleEndian.PutUint32(data[4:8], 0x01000007)
			if err := os.WriteFile(bunPath, data, 0755); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlinked-sidecar", "CAN-DIST-PATH", func() {
			if err := os.Remove(bunPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(archive, bunPath); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.change()
			output, err := run()
			if err == nil || !strings.Contains(string(output), tc.code) {
				t.Fatalf("expected %s, got %v: %s", tc.code, err, output)
			}
			if err := os.Remove(bunPath); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if err := os.WriteFile(bunPath, original, 0755); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, asset := range []string{"manifest.json", "tools/runtime/check.ts"} {
		t.Run("tampered-"+asset, func(t *testing.T) {
			path := filepath.Join(root, asset)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(append([]byte(nil), original...), '\n'), 0644); err != nil {
				t.Fatal(err)
			}
			output, err := run()
			code := "CAN-DIST-INTEGRITY"
			if asset == "manifest.json" {
				code = "CAN-DIST-MANIFEST"
			}
			if err == nil || !strings.Contains(string(output), code) {
				t.Fatalf("expected %s, got %v: %s", code, err, output)
			}
			if err := os.WriteFile(path, original, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Logf("offline absolute sidecar and symlink execution passed; hostile config ignored; identity %s", distribution.PinnedTarget().TargetID)
}

func treeHashes(t *testing.T, root string, dirs []string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			hashes[rel] = distribution.Hash(data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return hashes
}

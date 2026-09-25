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
	t.Run("current-parser-is-inert", func(t *testing.T) {
		parse := func(path string) ([]byte, error) {
			command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", link, "parse", "--render", path)
			command.Dir = cwd
			command.Env = env
			return command.CombinedOutput()
		}
		fixture := filepath.Join(source, "compiler/testdata/current/parser/offline.can")
		rendered, err := parse(fixture)
		if err != nil {
			t.Fatalf("offline parse: %v\n%s", err, rendered)
		}
		if !strings.Contains(string(rendered), "http::must_not_run") || !strings.Contains(string(rendered), "1 / 0") || strings.Contains(string(rendered), "never-print-this") {
			t.Fatalf("unexpected parse output: %s", rendered)
		}
		renderedPath := filepath.Join(cwd, "rendered.can")
		write(renderedPath, string(rendered))
		again, err := parse(renderedPath)
		if err != nil || string(again) != string(rendered) {
			t.Fatalf("offline render round-trip: %v\n%s", err, again)
		}
		write(renderedPath, "package invalid\n    provides []\n    uses []\nextern fn obsolete\n")
		rejected, err := parse(renderedPath)
		if err == nil || !strings.Contains(string(rejected), ":4:") || !strings.Contains(string(rejected), "syntax:") {
			t.Fatalf("obsolete grammar accepted: %v\n%s", err, rejected)
		}
	})
	t.Run("current-project-identities-offline", func(t *testing.T) {
		fixture := filepath.Join(source, "compiler/testdata/current/project")
		inspect := func(directory string) ([]byte, error) {
			command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", link, "inspect-project", directory)
			command.Dir = cwd
			command.Env = env
			return command.CombinedOutput()
		}
		var first []byte
		for i := 0; i < 2; i++ {
			directory := t.TempDir()
			err := filepath.WalkDir(fixture, func(name string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(fixture, name)
				if err != nil {
					return err
				}
				target := filepath.Join(directory, rel)
				if entry.IsDir() {
					return os.MkdirAll(target, 0700)
				}
				data, err := os.ReadFile(name)
				if err != nil {
					return err
				}
				return os.WriteFile(target, data, 0600)
			})
			if err != nil {
				t.Fatal(err)
			}
			before := treeHashes(t, directory, []string{"."})
			report, err := inspect(directory)
			if err != nil {
				t.Fatalf("offline project inspection: %v\n%s", err, report)
			}
			var decoded struct {
				Kind     string `json:"kind"`
				Packages []struct {
					ID      string `json:"id"`
					Sources []struct {
						ID         string `json:"id"`
						OutputPath string `json:"outputPath"`
					} `json:"sources"`
				} `json:"packages"`
			}
			if err := json.Unmarshal(report, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Kind != "can.package-resolution" || len(decoded.Packages) != 4 {
				t.Fatalf("wrong resolution report: %s", report)
			}
			paths := map[string]bool{}
			for _, pkg := range decoded.Packages {
				for _, file := range pkg.Sources {
					if paths[strings.ToLower(file.OutputPath)] {
						t.Fatal("independent source identities collided")
					}
					paths[strings.ToLower(file.OutputPath)] = true
				}
			}
			if strings.Contains(string(report), directory) || strings.Contains(string(report), "never-print-this") {
				t.Fatal("machine/environment state entered report")
			}
			if i == 0 {
				first = report
			} else if string(first) != string(report) {
				t.Fatal("relocating project changed resolution/output identities")
			}
			if !reflect.DeepEqual(before, treeHashes(t, directory, []string{"."})) {
				t.Fatal("project inspection wrote files")
			}
			changed := filepath.Join(directory, "vendor/sample/src/shared.can")
			data, err := os.ReadFile(changed)
			if err != nil {
				t.Fatal(err)
			}
			write(changed, string(data)+"\n// stale lock\n")
			rejected, err := inspect(directory)
			if err == nil || !strings.Contains(string(rejected), "stale dependency digest") {
				t.Fatalf("stale lock admitted: %v\n%s", err, rejected)
			}
		}
	})
	t.Run("current-declaration-types-offline", func(t *testing.T) {
		inspect := func(directory string) ([]byte, error) {
			command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", link, "inspect-types", directory)
			command.Dir = cwd
			command.Env = env
			return command.CombinedOutput()
		}
		report, err := inspect(filepath.Join(source, "compiler/testdata/current/project"))
		if err != nil || !strings.Contains(string(report), "can.declaration-types") || strings.Contains(string(report), "never-print-this") {
			t.Fatalf("offline type inspection: %v\n%s", err, report)
		}
		directory := t.TempDir()
		write(filepath.Join(directory, "can.project.json"), `{"source_root":".","error_registry":"can.errors.json"}`)
		write(filepath.Join(directory, "can.errors.json"), `{"active":[],"retired":[]}`)
		write(filepath.Join(directory, "main.can"), "package app\n    provides []\n    uses []\nrecord infinite\n    infinite next\n")
		rejected, err := inspect(directory)
		if err == nil || !strings.Contains(string(rejected), "finite inhabitant") {
			t.Fatalf("uninhabited record admitted: %v\n%s", err, rejected)
		}
	})
	t.Run("native-nominal-data-offline", func(t *testing.T) {
		// Execute the packaged data conformance with the absolute staged Bun.
		// The compiler's authored source cannot select an arbitrary tool entry.
		clean := t.TempDir()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(root, "tools/runtime/bunfig.toml"), "test", filepath.Join(root, "runtime/data.test.ts"))
		command.Dir = clean
		command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
		output, err := command.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "2 pass") {
			t.Fatalf("staged native data conformance: %v\n%s", err, output)
		}
	})
	t.Run("native-primitives-offline", func(t *testing.T) {
		clean := t.TempDir()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(root, "tools/runtime/bunfig.toml"), "test", filepath.Join(root, "runtime/primitive.test.ts"))
		command.Dir = clean
		command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
		output, err := command.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "4 pass") {
			t.Fatalf("staged native primitive conformance: %v\n%s", err, output)
		}
	})
	t.Run("native-failures-offline", func(t *testing.T) {
		clean := t.TempDir()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(root, "tools/runtime/bunfig.toml"), "test", filepath.Join(root, "runtime/failure.test.ts"), filepath.Join(root, "runtime/domain.test.ts"))
		command.Dir = clean
		command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
		output, err := command.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "11 pass") {
			t.Fatalf("staged native failure conformance: %v\n%s", err, output)
		}
	})
	t.Run("native-completions-offline", func(t *testing.T) {
		clean := t.TempDir()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(root, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(root, "tools/runtime/bunfig.toml"), "test", filepath.Join(root, "runtime/completion.test.ts"))
		command.Dir = clean
		command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
		output, err := command.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "5 pass") {
			t.Fatalf("staged completion conformance: %v\n%s", err, output)
		}
	})
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

func TestDistributionShipsBrowserBundleTool(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root, err := distribution.Build(ctx, source, t.TempDir(), archive, "integration")
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	shipped := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("bundle omits %s: %v", name, err)
		}
		if digest, ok := manifest.Files[name]; !ok || digest != distribution.Hash(data) {
			t.Fatalf("bundle manifest does not bind %s", name)
		}
		return data
	}
	tool := shipped("tools/runtime/browser-bundle.ts")
	for _, required := range []string{"Bun.build", `target: "browser"`, "can.browser-bundle-request", "can.browser-bundle-report"} {
		if !strings.Contains(string(tool), required) {
			t.Fatalf("bundle tool lacks %q", required)
		}
	}
	for _, banned := range []string{"onResolve", "onLoad", "plugins:", "plugin(", "alias:", "alias(", "external:"} {
		if strings.Contains(string(tool), banned) {
			t.Fatalf("bundle tool carries %q machinery", banned)
		}
	}
	for _, name := range []string{
		"runtime/browser/formats.ts",
		"runtime/browser/cookies.ts",
		"runtime/browser/csrf.ts",
		"runtime/browser/clock.ts",
		"runtime/browser/log.ts",
		"runtime/browser/markdown.ts",
		"runtime/browser/assets.ts",
		"runtime/assert/lineage.ts",
	} {
		shipped(name)
	}
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

// Package driver resolves and executes only the hash-locked private sidecar.
package driver

import (
	"bytes"
	"context"
	"debug/macho"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/distribution"
)

type Runtime struct {
	Root, Executable string
	manifest         distribution.Manifest
}

func Resolve(expectedManifestSHA256 string) (*Runtime, error) {
	launcher, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return resolve(launcher, expectedManifestSHA256)
}

func resolve(launcher, expectedManifestSHA256 string) (*Runtime, error) {
	if expectedManifestSHA256 == "" {
		return nil, fmt.Errorf("CAN-DIST-UNBUNDLED: build a development distribution first")
	}
	target := distribution.PinnedTarget()
	if runtime.GOOS != target.Runtime.Platform || runtime.GOARCH != target.Runtime.Architecture {
		return nil, fmt.Errorf("CAN-DIST-PLATFORM: requires macOS arm64")
	}
	osVersion, err := hostOSVersion()
	if err != nil {
		return nil, err
	}
	if !atLeast(osVersion, target.Runtime.MinimumOSVersion) {
		return nil, fmt.Errorf("CAN-DIST-OS: macOS %s or newer required (found %s)", target.Runtime.MinimumOSVersion, osVersion)
	}
	launcher, err = filepath.EvalSymlinks(launcher)
	if err != nil {
		return nil, err
	}
	launcher, err = filepath.Abs(launcher)
	if err != nil {
		return nil, err
	}
	if filepath.Base(filepath.Dir(launcher)) != "bin" {
		return nil, fmt.Errorf("CAN-DIST-LAYOUT: launcher must be in bin")
	}
	root := filepath.Dir(filepath.Dir(launcher))
	raw, err := regularFile(root, "manifest.json", false)
	if err != nil {
		return nil, err
	}
	if distribution.Hash(raw) != expectedManifestSHA256 {
		return nil, fmt.Errorf("CAN-DIST-MANIFEST: manifest does not match this launcher")
	}
	var manifest distribution.Manifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("CAN-DIST-MANIFEST: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.development-distribution" || manifest.TargetID != target.TargetID {
		return nil, fmt.Errorf("CAN-DIST-MANIFEST: unsupported distribution identity")
	}
	for _, name := range []string{target.Runtime.Executable, "distribution/target.json", "runtime/environment.ts", "tools/runtime/check.ts", "tools/runtime/bunfig.toml", "tsconfig.json", "distribution/notices/BUN-LICENSE.md", "distribution/notices/README.md"} {
		if manifest.Files[name] == "" {
			return nil, fmt.Errorf("CAN-DIST-MANIFEST: required asset missing: %s", name)
		}
	}
	names := make([]string, 0, len(manifest.Files))
	for name := range manifest.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		isRuntime := name == target.Runtime.Executable
		data, err := regularFile(root, name, isRuntime)
		if err != nil {
			return nil, err
		}
		if isRuntime {
			file, err := macho.NewFile(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("CAN-DIST-ARCH: runtime is not a Mach-O executable")
			}
			if file.Cpu != macho.CpuArm64 {
				return nil, fmt.Errorf("CAN-DIST-ARCH: runtime must be arm64")
			}
			if distribution.Hash(data) != target.Runtime.SHA256 {
				return nil, fmt.Errorf("CAN-DIST-INTEGRITY: pinned Bun hash mismatch")
			}
		}
		if name == "distribution/target.json" && !bytes.Equal(data, distribution.TargetJSON) {
			return nil, fmt.Errorf("CAN-DIST-TARGET: target differs from compiled pin")
		}
		if distribution.Hash(data) != manifest.Files[name] {
			return nil, fmt.Errorf("CAN-DIST-INTEGRITY: modified asset %s", name)
		}
	}
	return &Runtime{Root: root, Executable: filepath.Join(root, target.Runtime.Executable), manifest: manifest}, nil
}

func regularFile(root, name string, executable bool) ([]byte, error) {
	if name == "." || filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || strings.HasPrefix(name, "../") {
		return nil, fmt.Errorf("CAN-DIST-PATH: invalid asset path %q", name)
	}
	path := root
	parts := strings.Split(name, "/")
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("CAN-DIST-MISSING: %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("CAN-DIST-PATH: symlinked asset %s", name)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return nil, fmt.Errorf("CAN-DIST-PATH: non-directory ancestor of %s", name)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("CAN-DIST-PATH: non-regular asset %s", name)
		}
		if executable && info.Mode().Perm()&0111 == 0 {
			return nil, fmt.Errorf("CAN-DIST-PERMISSION: runtime is not executable")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("CAN-DIST-READ: %s: %w", name, err)
	}
	return data, nil
}

func atLeast(actual, minimum string) bool {
	a, m := strings.Split(actual, "."), strings.Split(minimum, ".")
	for i := 0; i < len(a) || i < len(m); i++ {
		av, mv := 0, 0
		var err error
		if i < len(a) {
			av, err = strconv.Atoi(a[i])
			if err != nil {
				return false
			}
		}
		if i < len(m) {
			mv, err = strconv.Atoi(m[i])
			if err != nil {
				return false
			}
		}
		if av != mv {
			return av > mv
		}
	}
	return true
}

// RunTool accepts a manifest-owned entry only. It is not a user JS execution API.
// Future generated-program execution must validate the build manifest first.
func (r *Runtime) RunTool(ctx context.Context, entry string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if r.manifest.Files[entry] == "" || !strings.HasPrefix(entry, "tools/runtime/") || !strings.HasSuffix(entry, ".ts") {
		return fmt.Errorf("CAN-DIST-ENTRY: tool is not in distribution manifest")
	}
	work, err := os.MkdirTemp("", "can-runtime-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	for _, dir := range []string{"home", "config", "tmp", "cwd"} {
		if err := os.Mkdir(filepath.Join(work, dir), 0700); err != nil {
			return err
		}
	}
	snapshot := map[string]string{}
	for _, item := range environment {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			snapshot[key] = value
		}
	}
	input, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	read, write, err := os.Pipe()
	if err != nil {
		return err
	}
	defer read.Close()
	defer write.Close()
	arguments := []string{"--no-install", "--no-env-file", "--no-macros", "--config=" + filepath.Join(r.Root, "tools/runtime/bunfig.toml"), filepath.Join(r.Root, entry)}
	arguments = append(arguments, args...)
	cmd := exec.CommandContext(ctx, r.Executable, arguments...)
	cmd.Dir = filepath.Join(work, "cwd")
	// Application state travels on fd 3, never on argv, disk, or Bun's startup
	// environment. This allowlist avoids an incomplete denylist of launch knobs.
	cmd.Env = []string{"HOME=" + filepath.Join(work, "home"), "XDG_CONFIG_HOME=" + filepath.Join(work, "config"), "TMPDIR=" + filepath.Join(work, "tmp"), "PATH=/nonexistent"}
	cmd.ExtraFiles = []*os.File{read}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("CAN-DIST-EXEC: %w", err)
	}
	read.Close()
	wrote := make(chan error, 1)
	go func() { _, err := write.Write(input); write.Close(); wrote <- err }()
	err = cmd.Wait()
	write.Close() // Unblock the writer if the child exited before reading fd 3.
	writeErr := <-wrote
	if err != nil {
		return err
	}
	if writeErr != nil {
		return fmt.Errorf("CAN-DIST-ENV: %w", writeErr)
	}
	return nil
}

package distribution

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// VerifyBundle checks a version directory against the pinned target and its
// own manifest without executing anything inside it. Release and install
// both verify before trusting a single byte: strict manifest shape,
// required assets, exact per-file hashes with no unknown files, no
// symlinks anywhere, a host-format runtime (arm64 Mach-O or x86-64 ELF)
// matching the pin, a target record identical to the compiled pin, and a
// launcher stamped with the manifest digest.
func VerifyBundle(dir string) (Manifest, error) {
	var empty Manifest
	abs, err := filepath.Abs(dir)
	if err != nil {
		return empty, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return empty, fmt.Errorf("verify bundle: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return empty, fmt.Errorf("verify bundle: not a directory")
	}
	target := PinnedTarget()
	if !HostSupported() || runtime.GOOS != target.Runtime.Platform || runtime.GOARCH != target.Runtime.Architecture {
		return empty, fmt.Errorf("verify bundle: requires one of %s", strings.Join(SupportedTargets(), ", "))
	}
	osVersion, err := hostOSVersion()
	if err != nil {
		return empty, err
	}
	if !atLeastVersion(osVersion, target.Runtime.MinimumOSVersion) {
		return empty, fmt.Errorf("verify bundle: %s %s or newer required (found %s)", target.Runtime.Platform, target.Runtime.MinimumOSVersion, osVersion)
	}
	raw, err := bundleFile(abs, "manifest.json")
	if err != nil {
		return empty, err
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return empty, fmt.Errorf("verify bundle: invalid manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "can.development-distribution" || manifest.TargetID != target.TargetID {
		return empty, fmt.Errorf("verify bundle: unsupported distribution identity")
	}
	for _, name := range []string{target.Runtime.Executable, "distribution/target.json", "runtime/environment.ts", "tools/runtime/check.ts", "tools/runtime/bunfig.toml", "tsconfig.json", "distribution/notices/BUN-LICENSE.md", "distribution/notices/README.md"} {
		if manifest.Files[name] == "" {
			return empty, fmt.Errorf("verify bundle: required asset missing: %s", name)
		}
	}
	names := make([]string, 0, len(manifest.Files))
	for name := range manifest.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := bundleFile(abs, name)
		if err != nil {
			return empty, err
		}
		if name == target.Runtime.Executable {
			if err := checkRuntimeFormat(data, target); err != nil {
				return empty, err
			}
			if Hash(data) != target.Runtime.SHA256 {
				return empty, fmt.Errorf("verify bundle: pinned Bun hash mismatch")
			}
			full := filepath.Join(abs, filepath.FromSlash(name))
			stat, err := os.Stat(full)
			if err != nil || stat.Mode().Perm()&0111 == 0 {
				return empty, fmt.Errorf("verify bundle: runtime is not executable")
			}
		}
		if name == "distribution/target.json" && !bytes.Equal(data, PinnedTargetJSON()) {
			return empty, fmt.Errorf("verify bundle: target differs from compiled pin")
		}
		if Hash(data) != manifest.Files[name] {
			return empty, fmt.Errorf("verify bundle: modified asset %s", name)
		}
	}
	launcher, err := bundleFile(abs, "bin/canlc")
	if err != nil {
		return empty, err
	}
	if !bytes.Contains(launcher, []byte(Hash(raw))) {
		return empty, fmt.Errorf("verify bundle: launcher stamp does not match manifest")
	}
	if err := noUnknownFiles(abs, manifest); err != nil {
		return empty, err
	}
	return manifest, nil
}

// checkRuntimeFormat binds the sidecar to the host executable format:
// arm64 Mach-O on macOS, 64-bit x86-64 ELF on Linux.
func checkRuntimeFormat(data []byte, target Target) error {
	if target.Runtime.Platform == "linux" {
		file, err := elf.NewFile(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("verify bundle: runtime is not an ELF executable")
		}
		file.Close()
		if file.Class != elf.ELFCLASS64 || file.Machine != elf.EM_X86_64 {
			return fmt.Errorf("verify bundle: runtime must be x86-64 ELF")
		}
		return nil
	}
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("verify bundle: runtime is not a Mach-O executable")
	}
	file.Close()
	if file.Cpu != macho.CpuArm64 {
		return fmt.Errorf("verify bundle: runtime must be arm64")
	}
	return nil
}

func bundleFile(root, name string) ([]byte, error) {
	if name == "." || filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || strings.HasPrefix(name, "../") {
		return nil, fmt.Errorf("verify bundle: invalid asset path %q", name)
	}
	path := root
	parts := strings.Split(name, "/")
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("verify bundle: missing asset %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("verify bundle: symlinked asset %s", name)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return nil, fmt.Errorf("verify bundle: non-directory ancestor of %s", name)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("verify bundle: non-regular asset %s", name)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("verify bundle: cannot read asset %s: %w", name, err)
	}
	return data, nil
}

func noUnknownFiles(root string, manifest Manifest) error {
	known := map[string]bool{"manifest.json": true}
	for name := range manifest.Files {
		known[name] = true
	}
	// bin/canlc is launcher-built and stamped rather than manifest-hashed;
	// its identity is the embedded manifest digest checked above.
	known["bin/canlc"] = true
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !known[filepath.ToSlash(rel)] {
			return fmt.Errorf("verify bundle: unknown file %s", rel)
		}
		return nil
	})
}

func atLeastVersion(actual, minimum string) bool {
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

package distribution

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Install lays a release archive out as one immutable version directory
// under root/versions/ and atomically selects it through the current
// symlink. Every step verifies before mutating: the detached SHA-256
// record, each archive entry, the staged tree against its manifest, and
// the selection target for containment. Existing versions are never
// modified, and failures leave the previous selection untouched. Only
// staging directories the installer owns are ever removed.
func Install(ctx context.Context, archive, shaPath, root string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	archiveAbs, err := filepath.Abs(archive)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := prepareRoot(rootAbs); err != nil {
		return "", err
	}
	if err := checkOwnership(rootAbs); err != nil {
		return "", err
	}
	base := filepath.Base(archiveAbs)
	if !strings.HasSuffix(base, ".zip") {
		return "", fmt.Errorf("install release: archive must end in .zip")
	}
	want, err := ParseSHA256Record(shaPath, base)
	if err != nil {
		return "", err
	}
	actual, err := hashFile(archiveAbs)
	if err != nil {
		return "", fmt.Errorf("install release: %w", err)
	}
	if actual != want {
		return "", fmt.Errorf("install release: archive SHA-256 mismatch")
	}
	stage, err := os.MkdirTemp(rootAbs, ".can-stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	name, err := extractRelease(archiveAbs, stage)
	if err != nil {
		return "", err
	}
	version := filepath.Join(stage, name)
	if _, err := VerifyBundle(version); err != nil {
		return "", err
	}
	final := filepath.Join(rootAbs, "versions", name)
	unlock, err := lockRoot(rootAbs)
	if err != nil {
		return "", err
	}
	defer unlock()
	if _, err := os.Lstat(final); !os.IsNotExist(err) {
		return "", fmt.Errorf("install release: version already exists or cannot be inspected: %s", name)
	}
	if err := os.MkdirAll(filepath.Join(rootAbs, "versions"), 0755); err != nil {
		return "", err
	}
	if err := os.Rename(version, final); err != nil {
		return "", err
	}
	if err := swapSelection(rootAbs, filepath.Join("versions", name)); err != nil {
		return "", err
	}
	return final, nil
}

// Selection reports the version directory the install root currently
// selects, after containment validation. An empty root reports "".
func Selection(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	link := filepath.Join(abs, "current")
	target, err := os.Readlink(link)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read selection: %w", err)
	}
	return checkSelectionTarget(abs, target)
}

func prepareRoot(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return os.MkdirAll(root, 0755)
	}
	if err != nil {
		return fmt.Errorf("install release: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("install release: root must not be a symlink")
	}
	if !info.IsDir() {
		return fmt.Errorf("install release: root is not a directory")
	}
	return nil
}

func checkOwnership(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("install release: %w", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("install release: ownership unavailable")
	}
	if int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("install release: root owned by another user")
	}
	return nil
}

// extractRelease unpacks exactly one top-level version directory. Absolute
// paths, escapes, symlinks, non-regular entries, and case-aliasing names
// refuse; the returned name is the single top-level directory.
func extractRelease(archive, stage string) (string, error) {
	data, err := os.ReadFile(archive)
	if err != nil {
		return "", fmt.Errorf("install release: %w", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("install release: invalid archive: %w", err)
	}
	seen := map[string]bool{}
	name := ""
	for _, file := range reader.File {
		entry := filepath.ToSlash(file.Name)
		if entry == "" || filepath.IsAbs(entry) || filepath.ToSlash(filepath.Clean(entry)) != entry || strings.HasPrefix(entry, "../") {
			return "", fmt.Errorf("install release: invalid archive path %q", file.Name)
		}
		top, _, _ := strings.Cut(entry, "/")
		if top == "" || strings.HasPrefix(top, ".") {
			return "", fmt.Errorf("install release: invalid version directory %q", top)
		}
		if name == "" {
			name = top
		} else if name != top {
			return "", fmt.Errorf("install release: archive holds more than one version")
		}
		if strings.HasSuffix(entry, "/") {
			continue
		}
		mode := file.Mode()
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() {
			return "", fmt.Errorf("install release: non-regular archive entry %q", file.Name)
		}
		key := strings.ToLower(entry)
		if seen[key] {
			return "", fmt.Errorf("install release: case-aliasing archive entry %q", file.Name)
		}
		seen[key] = true
		dest := filepath.Join(stage, filepath.FromSlash(entry))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", err
		}
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", fmt.Errorf("install release: %w", err)
		}
		rc, err := file.Open()
		if err != nil {
			out.Close()
			return "", fmt.Errorf("install release: %w", err)
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return "", fmt.Errorf("install release: %w", copyErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("install release: %w", closeErr)
		}
		// Modes come from the archive allowlist, never from untrusted bits:
		// executables stay executable, everything else is owner-read/write.
		perm := os.FileMode(0600)
		if mode.Perm()&0111 != 0 {
			perm = 0755
		}
		if err := os.Chmod(dest, perm); err != nil {
			return "", err
		}
	}
	if name == "" {
		return "", fmt.Errorf("install release: archive holds no version")
	}
	return name, nil
}

func swapSelection(root, target string) error {
	if _, err := checkSelectionTarget(root, target); err != nil {
		return err
	}
	link := filepath.Join(root, "current")
	temp, err := os.CreateTemp(root, ".can-current-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	temp.Close()
	os.Remove(tempPath)
	if err := os.Symlink(target, tempPath); err != nil {
		return err
	}
	return os.Rename(tempPath, link)
}

func checkSelectionTarget(root, target string) (string, error) {
	if target == "" || filepath.IsAbs(target) || filepath.ToSlash(filepath.Clean(target)) != target || strings.HasPrefix(target, "../") {
		return "", fmt.Errorf("install release: selection escapes the install root")
	}
	parts := strings.Split(target, "/")
	if len(parts) != 2 || parts[0] != "versions" || parts[1] == "" || strings.HasPrefix(parts[1], ".") {
		return "", fmt.Errorf("install release: selection is not an installed version")
	}
	final := filepath.Join(root, filepath.FromSlash(target))
	info, err := os.Lstat(final)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("install release: selected version is not installed")
	}
	return final, nil
}

func lockRoot(root string) (func(), error) {
	file, err := os.OpenFile(filepath.Join(root, ".can-lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, fmt.Errorf("install release: %w", err)
	}
	return func() {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}, nil
}

// PruneStaging removes interrupted installer-owned staging directories and
// selection temporaries. Anything else in the root is left alone, and
// unowned patterns refuse rather than delete.
func PruneStaging(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		owned := strings.HasPrefix(name, ".can-stage-") || strings.HasPrefix(name, ".can-current-") || strings.HasPrefix(name, ".can-release-")
		if !owned {
			continue
		}
		if err := checkOwnership(filepath.Join(abs, name)); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(abs, name)); err != nil {
			return err
		}
	}
	return nil
}

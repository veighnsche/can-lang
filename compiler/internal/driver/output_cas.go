package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Per-dist content store: dist/cas/<sha256hex> holds each distinct staged
// byte string once, and every generation links to it instead of copying.
// Layouts, bytes, hashes, and manifests are unchanged; only inode sharing
// differs, so validation, pairing, serving, and execution cannot tell.
//
// Staged content files are read-only (0400). In-place mutation would corrupt
// every linked generation, so it fails loudly with EACCES instead; honest
// tampering replaces the path (remove + recreate), which breaks the link.
// Manifests and metadata stay plain 0600 writes: they are unique per build
// and gain nothing from sharing.
//
// os.Root exposes no Link or Chmod, so linking uses absolute paths built
// internally here from validated digests and validated stage names only;
// no caller-supplied path ever reaches the filesystem calls below.
const casDirName = "cas"

func (s *OutputStore) casAbsPath(digest string) (string, error) {
	if !digestPattern.MatchString(digest) {
		return "", fmt.Errorf("content digest is not well-formed")
	}
	return filepath.Join(s.Graph.Root.Root, "dist", casDirName, digest), nil
}

var stageDirPattern = regexp.MustCompile(`^builds/\.stage-[0-9a-f]{32}$`)

func (s *OutputStore) stageAbsPath(stage, name string) (string, error) {
	if !stageDirPattern.MatchString(stage) {
		return "", fmt.Errorf("staging directory is not a reserved generation")
	}
	if err := outputPath(name); err != nil {
		return "", err
	}
	return filepath.Join(s.Graph.Root.Root, "dist", filepath.FromSlash(stage), filepath.FromSlash(name)), nil
}

func ensureCASDir(dist *os.Root) error {
	if info, err := dist.Lstat(casDirName); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("content store is not a real directory")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return dist.Mkdir(casDirName, 0700)
}

// ensureCAS installs the byte string under its digest and returns its
// absolute path. A present entry is re-verified; a tampered one is replaced
// from the just-hashed bytes, mirroring storeDurableBytes.
func (s *OutputStore) ensureCAS(digest string, data []byte) (string, error) {
	if hashBytes(data) != digest {
		return "", fmt.Errorf("content bytes do not match their digest")
	}
	if err := ensureCASDir(s.dist); err != nil {
		return "", err
	}
	entry, err := s.casAbsPath(digest)
	if err != nil {
		return "", err
	}
	if present, err := os.ReadFile(entry); err == nil {
		if hashBytes(present) == digest {
			return entry, nil
		}
		_ = os.Remove(entry)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	temp := filepath.Join(s.Graph.Root.Root, "dist", casDirName, ".tmp-"+digest)
	_ = os.Remove(temp)
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return "", err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(temp)
		return "", err
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return "", closeErr
	}
	if err := os.Rename(temp, entry); err != nil {
		_ = os.Remove(temp)
		return "", err
	}
	return entry, nil
}

// stageContentFile stages one content file as a link into the content store,
// falling back to a private read-only copy when linking is unavailable
// (cross-device, permissions, link limits). Either branch leaves identical
// bytes; only disk sharing differs.
func (s *OutputStore) stageContentFile(stage, name, digest string, data []byte) error {
	entry, err := s.ensureCAS(digest, data)
	if err != nil {
		return err
	}
	dest, err := s.stageAbsPath(stage, name)
	if err != nil {
		return err
	}
	if err := os.Link(entry, dest); err == nil {
		return nil
	}
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return closeErr
}

// collectCASRoots gathers every digest still referenced by a staged
// generation, a pending publication, or the retention ledger.
func (s *OutputStore) collectCASRoots() (map[string]bool, error) {
	roots := map[string]bool{}
	entries, err := outputEntries(s.dist, "builds")
	if err != nil {
		if os.IsNotExist(err) {
			return roots, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !digestPattern.MatchString(entry.Name()) {
			continue
		}
		raw, err := readOutputRegular(s.dist, "builds/"+entry.Name()+"/manifest.json")
		if err != nil {
			continue
		}
		var manifest OutputManifest
		if err := decodeOutput(raw, &manifest); err != nil {
			continue
		}
		for _, digest := range manifest.Files {
			if digestPattern.MatchString(digest) {
				roots[digest] = true
			}
		}
	}
	if raw, err := readOutputRegular(s.dist, "pending.json"); err == nil {
		var pending outputPending
		if err := decodeOutput(raw, &pending); err == nil {
			for _, digest := range pending.Manifest.Files {
				if digestPattern.MatchString(digest) {
					roots[digest] = true
				}
			}
		}
	}
	if ledger, _, err := s.readAssetLedger(); err == nil {
		for _, retained := range ledger.Retained {
			if digestPattern.MatchString(retained.Digest) {
				roots[retained.Digest] = true
			}
		}
	}
	return roots, nil
}

// sweepCAS deletes every store entry no generation, pending publication, or
// ledger row references, plus crash-debris temporaries. Unknown files fail
// closed for operator inspection, matching sweepAssetStore.
func (s *OutputStore) sweepCAS(roots map[string]bool) error {
	if _, err := s.dist.Lstat(casDirName); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	entries, err := outputEntries(s.dist, casDirName)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := casDirName + "/" + entry.Name()
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			if err := s.dist.Remove(name); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if !digestPattern.MatchString(entry.Name()) {
			return fmt.Errorf("unknown content store entry %s; preserve it before cleanup", entry.Name())
		}
		if roots[entry.Name()] {
			continue
		}
		if err := s.dist.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return syncOutputDir(s.dist, casDirName)
}

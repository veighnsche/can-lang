package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

var errAnalysisChanged = errors.New("analysis inputs changed")

type lspCachedAnalysis struct {
	fingerprint driver.Fingerprint
	snapshot    *driver.Snapshot
}

func scratchPath(uri string) string {
	sum := sha256.Sum256([]byte(uri))
	// Identity only: no directory or file is ever created here.
	return canonicalDocumentPath(filepath.Join(os.TempDir(), "can-lsp-memory", hex.EncodeToString(sum[:16])+".can"))
}

func (s *lspServer) snapshot(uri string) (*driver.Snapshot, error) {
	doc := s.docs[uri]
	if doc == nil || doc.path == "" {
		return nil, fmt.Errorf("document has no supported URI")
	}
	return s.snapshotInput(s.root(uri), doc.path, doc.text, strings.HasPrefix(uri, "untitled:"))
}

func (s *lspServer) snapshotRoot(root string) (*driver.Snapshot, error) {
	root = canonicalDocumentPath(root)
	return s.snapshotInput(root, filepath.Join(root, "can.project.json"), "", false)
}

func (s *lspServer) snapshotInput(root, path, text string, scratch bool) (result *driver.Snapshot, resultErr error) {
	defer func() {
		if resultErr != nil {
			s.analysisErr = resultErr
		}
	}()
	if err := s.bufferConflict(root); err != nil {
		return nil, err
	}
	cached := s.cache[root]
	var previous *driver.Snapshot
	if cached != nil {
		previous = cached.snapshot
	}
	before, err := s.fingerprint(root, previous)
	if err != nil {
		return nil, err
	}
	if cached != nil && before.Equal(cached.fingerprint) {
		s.used[root] = cached
		return cached.snapshot, nil
	}
	var snapshot *driver.Snapshot
	if s.analyze != nil {
		snapshot, err = s.analyze(s.ctx, root, path, s.overlay)
	} else if scratch {
		snapshot, err = driver.CheckScratchSnapshot(s.ctx, path, text)
	} else {
		snapshot, err = driver.CheckSnapshotContext(s.ctx, root, path, s.overlay)
	}
	if err != nil {
		return nil, err
	}
	after, err := s.fingerprint(root, snapshot)
	if err != nil {
		return nil, err
	}
	// Newly discovered inputs are checked against the exact bytes loaded by
	// analysis; pre-existing inputs must also survive a before/after comparison.
	if !fingerprintCompatible(before, after) || !s.loadedInputsMatch(snapshot) {
		return nil, errAnalysisChanged
	}
	if len(s.cache) >= 8 { // Replace whole snapshots; never retain histories.
		for key, entry := range s.cache {
			delete(s.features, entry.snapshot)
			delete(s.cache, key)
		}
	}
	if cached != nil {
		delete(s.features, cached.snapshot)
	}
	s.cache[root] = &lspCachedAnalysis{fingerprint: after, snapshot: snapshot}
	s.used[root] = s.cache[root]
	return snapshot, nil
}

// Additional paths (e.g. a raw fixture discovered from syntax) may only be
// appended after the first analysis. Existing identities cannot change.
func fingerprintCompatible(before, after driver.Fingerprint) bool {
	docs := map[string]driver.DocInput{}
	for _, doc := range after.Docs() {
		docs[doc.Path] = doc
	}
	disk := map[string]string{}
	for _, input := range after.Disk() {
		disk[input.Label] = input.Value
	}
	for _, doc := range before.Docs() {
		if docs[doc.Path] != doc {
			return false
		}
	}
	for _, input := range before.Disk() {
		if value, ok := disk[input.Label]; !ok || value != input.Value {
			return false
		}
	}
	return true
}

func (s *lspServer) loadedInputsMatch(snapshot *driver.Snapshot) bool {
	if snapshot == nil || snapshot.Graph == nil {
		return true
	}
	for path, data := range snapshot.Graph.Inputs {
		if entry, ok := s.overlay.Get(path); ok {
			if entry.Text != string(data) {
				return false
			}
			continue
		}
		// nil marks a failed read, not a loaded empty file.
		if data == nil {
			continue
		}
		hash, err := hashDiskFile(path)
		digest := sha256.Sum256(data)
		if err != nil || hash != hex.EncodeToString(digest[:]) {
			return false
		}
	}
	return true
}

func (s *lspServer) validateInputs() error {
	for root, expected := range s.used {
		snapshot := expected.snapshot
		actual, err := s.fingerprint(root, snapshot)
		if err != nil {
			return err
		}
		if !expected.fingerprint.Equal(actual) {
			return errAnalysisChanged
		}
	}
	return s.ctx.Err()
}

func hashDiskFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// The cache identity includes exact buffers, all source inventories, manifest
// and dependency configuration, declared assets, and every extra compiler read.
// Hashes are streamed; no source trees or private build caches are copied.
func (s *lspServer) fingerprint(root string, snapshot *driver.Snapshot) (driver.Fingerprint, error) {
	overlay := s.overlay.Snapshot()
	docs := map[string]driver.DocInput{}
	disk := map[string]string{}
	roots := map[string]bool{}
	addFile := func(path string) {
		logical := filepath.Clean(path)
		path = canonicalDocumentPath(logical)
		disk["input-path:"+logical] = path
		if entry, ok := overlay[path]; ok {
			docs[path] = driver.DocInput{Path: path, Version: entry.Version, Text: entry.Text}
			return
		}
		value, err := hashDiskFile(path)
		if err != nil {
			value = "unavailable:" + err.Error()
		}
		disk[path] = value
	}
	var visit func(string, int) error
	visit = func(directory string, depth int) error {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if roots[directory] || depth > 256 {
			return nil
		}
		roots[directory] = true
		manifestPath := canonicalDocumentPath(filepath.Join(directory, "can.project.json"))
		addFile(manifestPath)
		addFile(filepath.Join(directory, "can.lock.json"))
		var data []byte
		if entry, ok := overlay[manifestPath]; ok {
			data = []byte(entry.Text)
		} else {
			data, _ = os.ReadFile(manifestPath)
		}
		manifest, _ := project.ParseManifest(data)
		// Recoverable manifest fields remain analysis inputs even when another
		// field is invalid. The manifest bytes always invalidate the result.
		addConfined := func(relative string) {
			logical := filepath.Join(directory, relative)
			real := canonicalDocumentPath(logical)
			if !project.Contains(directory, real) {
				disk["escaped-path:"+logical] = real
				return
			}
			addFile(logical)
		}
		if project.NormalizePath(manifest.ErrorRegistry) == nil {
			addConfined(manifest.ErrorRegistry)
		}
		for _, asset := range manifest.Assets {
			if project.NormalizePath(asset) == nil {
				addConfined(asset)
			}
		}
		for _, dep := range manifest.Dependencies {
			if project.NormalizePath(dep) != nil {
				continue
			}
			target := filepath.Join(directory, dep)
			if real, err := filepath.EvalSymlinks(target); err == nil {
				target = real
			}
			if !project.Contains(directory, target) {
				continue
			}
			if err := visit(target, depth+1); err != nil {
				return err
			}
		}
		if project.NormalizePath(manifest.SourceRoot) != nil {
			return nil
		}
		sourceRoot := filepath.Join(directory, manifest.SourceRoot)
		names := []string{}
		seenDirs := map[string]bool{}
		var walk func(string) error
		walk = func(logical string) error {
			if err := s.ctx.Err(); err != nil {
				return err
			}
			real, err := filepath.EvalSymlinks(logical)
			if err != nil {
				disk["walk:"+logical] = err.Error()
				return nil
			}
			disk["directory-path:"+logical] = real
			if !project.Contains(directory, real) || seenDirs[real] {
				return nil
			}
			seenDirs[real] = true
			entries, err := os.ReadDir(real)
			if err != nil {
				disk["walk:"+logical] = err.Error()
				return nil
			}
			for _, entry := range entries {
				path := filepath.Join(logical, entry.Name())
				target, err := filepath.EvalSymlinks(path)
				if err != nil {
					disk["walk:"+path] = err.Error()
					continue
				}
				if !project.Contains(directory, target) {
					disk["escaped-path:"+path] = target
					continue
				}
				info, err := os.Stat(target)
				if err != nil {
					disk["walk:"+path] = err.Error()
					continue
				}
				if info.IsDir() {
					if err := walk(path); err != nil {
						return err
					}
					continue
				}
				if !strings.HasSuffix(path, ".can") {
					continue
				}
				names = append(names, path)
				addFile(target)
				disk["source-path:"+path] = target
			}
			return nil
		}
		if err := walk(sourceRoot); err != nil {
			return err
		}

		sort.Strings(names)
		disk["source-inventory:"+sourceRoot] = strings.Join(names, "\x00")
		return nil
	}
	scratch := false
	for uri, doc := range s.docs {
		if strings.HasPrefix(uri, "untitled:") && doc.path == root {
			docs[doc.path] = driver.DocInput{Path: doc.path, Version: doc.version, Text: doc.text}
			scratch = true
			break
		}
	}
	if !scratch {
		if err := visit(root, 0); err != nil {
			return driver.Fingerprint{}, err
		}
	}
	for path, entry := range overlay {
		for directory := range roots {
			if project.Contains(directory, path) {
				docs[path] = driver.DocInput{Path: path, Version: entry.Version, Text: entry.Text}
				break
			}
		}
	}
	// Identical alias text is represented once above; preserve every URI and
	// version so closing/changing either alias invalidates versioned answers.
	for uri, doc := range s.docs {
		if doc.path == root || project.Contains(root, doc.path) {
			disk["document-uri:"+uri] = doc.path + "\x00" + strconv.FormatInt(doc.version, 10)
		}
	}
	if snapshot != nil && snapshot.Graph != nil && !scratch {
		for path := range snapshot.Graph.Inputs {
			addFile(path)
		}
		for _, p := range snapshot.Graph.Projects {
			for _, asset := range p.CheckedAssets {
				addFile(filepath.Join(p.Root, asset.Relative))
			}
			for _, fixture := range p.CheckedFixtures {
				addFile(filepath.Join(p.Root, fixture.Relative))
			}
		}
	}
	docInputs := make([]driver.DocInput, 0, len(docs))
	for _, doc := range docs {
		docInputs = append(docInputs, doc)
	}
	diskInputs := make([]driver.DiskInput, 0, len(disk))
	for label, value := range disk {
		diskInputs = append(diskInputs, driver.DiskInput{Label: label, Value: value})
	}
	return driver.NewFingerprint(docInputs, diskInputs), nil
}

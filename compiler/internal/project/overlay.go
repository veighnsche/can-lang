package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Overlay holds unsaved editor buffers keyed by canonical absolute path plus
// the document version that produced them. It covers source text only:
// manifests, registries, locks, and assets always load from disk, and
// entries for paths the project walk never visits are ignored until the
// file is saved into the tree. Versions let publishers prove a diagnosis
// belongs to the buffer the editor holds.
type Overlay struct {
	mu      sync.Mutex
	entries map[string]OverlayEntry
}

// OverlayEntry is one unsaved buffer.
type OverlayEntry struct {
	Version int64
	Text    string
}

// NewOverlay returns an empty overlay.
func NewOverlay() *Overlay { return &Overlay{entries: map[string]OverlayEntry{}} }

// Set records an unsaved buffer. The path must be absolute and name a Can
// source file; anything else is refused rather than silently dropped.
func (o *Overlay) Set(path string, version int64, text string) error {
	canonical, err := canonicalPath(path)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(canonical, ".can") {
		return fmt.Errorf("overlay covers Can sources, not %s", path)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.entries[canonical] = OverlayEntry{Version: version, Text: text}
	return nil
}

// Clear forgets an unsaved buffer, for example on save or close.
func (o *Overlay) Clear(path string) error {
	canonical, err := canonicalPath(path)
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.entries, canonical)
	return nil
}

// Get returns the unsaved buffer for a canonical path, if any.
func (o *Overlay) Get(path string) (OverlayEntry, bool) {
	canonical, err := canonicalPath(path)
	if err != nil {
		return OverlayEntry{}, false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	entry, ok := o.entries[canonical]
	return entry, ok
}

func canonicalPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("overlay paths must be absolute: %s", path)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Unsaved files may not exist on disk yet; canonicalize the
		// existing ancestors instead of refusing the buffer.
		parent := path
		for {
			parent = filepath.Dir(parent)
			if parent == filepath.Dir(parent) {
				return "", fmt.Errorf("overlay path has no existing ancestor: %s", path)
			}
			if resolved, err := filepath.EvalSymlinks(parent); err == nil {
				rel, err := filepath.Rel(parent, path)
				if err != nil {
					return "", err
				}
				return filepath.Join(resolved, rel), nil
			}
		}
	}
	return real, nil
}

// LoadWithOverlay reads exactly the named local project like Load, except
// source bytes for walked files come from the overlay when the editor holds
// an unsaved buffer. Substitution happens before parsing, so a diagnosis
// over the snapshot reports precisely what the CLI would report had the
// buffers been saved. A nil overlay behaves exactly like Load.
func LoadWithOverlay(directory string, overlay *Overlay) (*Graph, error) {
	if overlay == nil {
		return Load(directory)
	}
	return load(directory, overlay.bytesFor)
}

func (o *Overlay) bytesFor(real string) ([]byte, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	entry, ok := o.entries[real]
	if !ok {
		return nil, false
	}
	return []byte(entry.Text), true
}

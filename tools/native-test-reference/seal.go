package reference

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SealInput carries the observations the seal binds to the selection.
type SealInput struct {
	// Version is the staged seed version root suffix.
	Version string
	// GoVersion and Host are observed at build time, not pinned.
	GoVersion string
	Host      string
	// OwnerUID and OwnerGID own the staged seed (run/acceptance ownership).
	OwnerUID int
	OwnerGID int
	// StagedUTC is the staging completion time in UTC.
	StagedUTC time.Time
}

// Manifest is the sealed acceptance record: exact selection inputs plus the
// exact staged bundle contents. Any later mismatch against this manifest
// invalidates the seed.
type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	Version       string    `json:"version"`
	SeedRoot      string    `json:"seed_root"`
	Selection     Selection `json:"selection"`
	// BundleFiles maps bundle-relative paths to SHA-256: the distribution
	// manifest entries plus the compiled canlc binary the distribution
	// manifest claims but does not hash.
	BundleFiles map[string]string `json:"bundle_files"`
	GoVersion   string            `json:"go_version"`
	Host        string            `json:"host"`
	OwnerUID    int               `json:"owner_uid"`
	OwnerGID    int               `json:"owner_gid"`
	StagedUTC   string            `json:"staged_utc"`
}

// Seal verifies the staged seed root and writes the acceptance manifest
// inside it (seed-manifest.json), returning the manifest path. The seed root
// must hold the distribution manifest.json plus bin/canlc.
func Seal(seedRoot string, sel Selection, input SealInput) (string, error) {
	if input.Version == "" {
		return "", fmt.Errorf("reference: seal needs a seed version")
	}
	distRaw, err := os.ReadFile(filepath.Join(seedRoot, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("reference: read distribution manifest: %w", err)
	}
	var dist struct {
		Files map[string]string `json:"Files"`
	}
	if err := json.Unmarshal(distRaw, &dist); err != nil {
		return "", fmt.Errorf("reference: decode distribution manifest: %w", err)
	}
	if len(dist.Files) == 0 {
		return "", fmt.Errorf("reference: distribution manifest holds no files")
	}
	binary, err := hashFile(filepath.Join(seedRoot, "bin", "canlc"))
	if err != nil {
		return "", fmt.Errorf("reference: hash staged canlc: %w", err)
	}
	files := make(map[string]string, len(dist.Files)+1)
	for name, sum := range dist.Files {
		files[name] = sum
	}
	files["bin/canlc"] = binary.SHA256
	manifest := Manifest{
		SchemaVersion: 1,
		Kind:          "can.native-test.seed",
		Version:       input.Version,
		SeedRoot:      seedRoot,
		Selection:     sel,
		BundleFiles:   files,
		GoVersion:     input.GoVersion,
		Host:          input.Host,
		OwnerUID:      input.OwnerUID,
		OwnerGID:      input.OwnerGID,
		StagedUTC:     input.StagedUTC.UTC().Format(time.RFC3339),
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')
	path := filepath.Join(seedRoot, "seed-manifest.json")
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		return "", fmt.Errorf("reference: write seed manifest: %w", err)
	}
	return path, nil
}

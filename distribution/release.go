package distribution

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReleaseArtifacts names the three files one release produces beside each
// other: the shippable zip, its detached SHA-256 record in shasum form,
// and the read-only inspection transcript.
type ReleaseArtifacts struct {
	Archive    string
	SHA256     string
	Inspection string
}

// Release verifies a built version directory and packages it into one
// shippable zip plus integrity and inspection records. The bundle is never
// modified; entry order and timestamps are fixed so identical bundles
// produce identical archives. Existing artifacts are never overwritten.
func Release(ctx context.Context, bundle, output string) (ReleaseArtifacts, error) {
	var none ReleaseArtifacts
	abs, err := filepath.Abs(bundle)
	if err != nil {
		return none, err
	}
	manifest, err := VerifyBundle(abs)
	if err != nil {
		return none, err
	}
	name := "can-" + manifest.Version + "-" + manifest.TargetID
	if filepath.Base(abs) != name {
		return none, fmt.Errorf("release bundle: directory name %q does not match manifest identity %q", filepath.Base(abs), name)
	}
	out, err := filepath.Abs(output)
	if err != nil {
		return none, err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return none, err
	}
	artifacts := ReleaseArtifacts{
		Archive:    filepath.Join(out, name+".zip"),
		SHA256:     filepath.Join(out, name+".zip.sha256"),
		Inspection: filepath.Join(out, name+".inspection.json"),
	}
	for _, path := range []string{artifacts.Archive, artifacts.SHA256, artifacts.Inspection} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return none, fmt.Errorf("release bundle: artifact already exists or cannot be inspected: %s", path)
		}
	}
	inspection, err := InspectRuntime(ctx, filepath.Join(abs, PinnedTarget().Runtime.Executable))
	if err != nil {
		return none, err
	}
	files := []string{"manifest.json", "bin/canlc"}
	for file := range manifest.Files {
		files = append(files, file)
	}
	sort.Strings(files)
	stage, err := os.CreateTemp(out, ".can-release-*.zip")
	if err != nil {
		return none, err
	}
	stagePath := stage.Name()
	defer os.Remove(stagePath)
	writer := zip.NewWriter(stage)
	// Fixed timestamp plus sorted entries: the archive bytes are a pure
	// function of the verified bundle contents.
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, file := range files {
		data, err := bundleFile(abs, file)
		if err != nil {
			writer.Close()
			stage.Close()
			return none, err
		}
		header := &zip.FileHeader{Name: name + "/" + file, Method: zip.Deflate, Modified: stamp}
		header.SetMode(0644)
		if file == PinnedTarget().Runtime.Executable || file == "bin/canlc" {
			header.SetMode(0755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			writer.Close()
			stage.Close()
			return none, err
		}
		if _, err := entry.Write(data); err != nil {
			writer.Close()
			stage.Close()
			return none, err
		}
	}
	if err := writer.Close(); err != nil {
		stage.Close()
		return none, err
	}
	if err := stage.Close(); err != nil {
		return none, err
	}
	if err := os.Rename(stagePath, artifacts.Archive); err != nil {
		return none, err
	}
	digest, err := hashFile(artifacts.Archive)
	if err != nil {
		return none, err
	}
	record := digest + "  " + name + ".zip\n"
	if err := os.WriteFile(artifacts.SHA256, []byte(record), 0644); err != nil {
		return none, err
	}
	transcript, err := json.MarshalIndent(inspection, "", "  ")
	if err != nil {
		return none, err
	}
	transcript = append(transcript, '\n')
	if err := os.WriteFile(artifacts.Inspection, transcript, 0644); err != nil {
		return none, err
	}
	return artifacts, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return Hash(data), nil
}

// ParseSHA256Record reads a detached "<hex>  <filename>" record and returns
// the digest after checking the shape. It binds the record to one expected
// file name so records cannot be swapped between artifacts.
func ParseSHA256Record(path, filename string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("sha256 record: %w", err)
	}
	text := strings.TrimRight(string(raw), "\n")
	digest, name, ok := strings.Cut(text, "  ")
	if !ok || name != filename || len(digest) != 64 {
		return "", fmt.Errorf("sha256 record: malformed record for %s", filename)
	}
	for _, c := range digest {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return "", fmt.Errorf("sha256 record: malformed digest for %s", filename)
		}
	}
	return digest, nil
}

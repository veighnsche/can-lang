package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// TestPairedHandshakeIdentity is the C-H build contract: the exact identity
// chain a paired server build publishes for the generation handshake. The
// server generation names itself (manifest buildID equals the directory
// name), the pairing record is hash-bound inside that generation, and the
// selection points at it — everything the E/C/A slices consume.
func TestPairedHandshakeIdentity(t *testing.T) {
	browserDir := t.TempDir()
	manifestPath := writeBrowserGeneration(t, browserDir, pairingBundle(), nil, "")
	pairing, err := verifyBrowserManifest(manifestPath, pairingServerGraph(nil))
	if err != nil {
		t.Fatal(err)
	}
	root := outputProject(t)
	s := outputBegin(t, root)
	artifacts := []ir.Artifact{{Path: "entry.ts", Bytes: []byte("export const value=1n;\n")}}
	for _, asset := range pairing.assets {
		artifacts = append(artifacts, ir.Artifact{Path: asset.Artifact, Bytes: asset.Bytes})
	}
	artifacts = append(artifacts, ir.Artifact{Path: pairingRecordPath, Bytes: pairing.pairingJSON})
	prepared, err := PrepareOutput(s.BuildInputs(strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64)), "entry.ts", artifacts)
	if err != nil {
		t.Fatal(err)
	}
	prepared.validated = true
	id, directory, err := s.Stage(prepared)
	if err != nil {
		t.Fatal(err)
	}
	// The server generation is self-naming: the startup read in the
	// handshake contract resolves the manifest beside the running code.
	if id != filepath.Base(directory) {
		t.Fatalf("staged %s under %s", id, directory)
	}
	encoded, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tree OutputManifest
	if err := json.Unmarshal(encoded, &tree); err != nil {
		t.Fatal(err)
	}
	if tree.BuildID != id || tree.Kind != "can.output-generation" {
		t.Fatalf("server manifest names %q", tree.BuildID)
	}
	if tree.Files[pairingRecordPath] != hashBytes(pairing.pairingJSON) {
		t.Fatal("pairing record is not bound to the server generation")
	}
	var record struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		BrowserBuild  string `json:"browserBuildId"`
		Generation    string `json:"generation"`
	}
	if err := json.Unmarshal(pairing.pairingJSON, &record); err != nil {
		t.Fatal(err)
	}
	if record.SchemaVersion != 1 || record.Kind != "can.browser-pairing" ||
		record.BrowserBuild != pairing.buildID || record.Generation != pairing.generationID {
		t.Fatalf("pairing record names %q/%q", record.BrowserBuild, record.Generation)
	}
	if !digestPattern.MatchString(id) || !digestPattern.MatchString(pairing.buildID) {
		t.Fatal("handshake identities are not content digests")
	}
	report := pairing.pairedReport()
	if report == nil || report.BrowserBuildID != pairing.buildID || report.Generation != pairing.generationID {
		t.Fatalf("build report diverges from the verified pairing: %+v", report)
	}
	if _, err := s.selectCurrentWithAssets("", id, pairing.files); err != nil {
		t.Fatal(err)
	}
	if current, err := s.currentBuildID(); err != nil || current != id {
		t.Fatalf("current = %q %v", current, err)
	}
	// A replaced pairing record breaks the generation binding and fails
	// the retention re-read closed: no silent unpairing.
	staged := filepath.Join(directory, filepath.FromSlash(pairingRecordPath))
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	data[0] ^= 0x01
	if err := os.Remove(staged); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.generation(id, false); err == nil {
		t.Fatal("generation with a tampered pairing record still validates")
	}
	if _, _, err := s.priorPairedSet(id); err == nil {
		t.Fatal("tampered pairing record still reads as a prior set")
	}
}

// TestMetadataWritesAreUnique pins the metadata half of CAS-safe staging:
// manifests and dist metadata are private 0600 writes, never shared
// across builds. Only content bytes share inodes, and only read-only.
func TestMetadataWritesAreUnique(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := outputPrepared(t, s, "export const value=1n;")
	second := outputPrepared(t, s, "export const value=2n;")
	firstDir, err := s.Publish(first)
	if err != nil {
		t.Fatal(err)
	}
	secondDir, err := s.Publish(second)
	if err != nil {
		t.Fatal(err)
	}
	firstManifest := filepath.Join(firstDir, "manifest.json")
	secondManifest := filepath.Join(secondDir, "manifest.json")
	for _, name := range []string{firstManifest, secondManifest, filepath.Join(root, "dist", "current.json")} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("metadata %s has mode %v", name, info.Mode())
		}
	}
	if same, err := sameStagedFile(firstManifest, secondManifest); err != nil || same {
		t.Fatal("per-build manifests share an inode", err)
	}
	// The retention ledger is dist metadata too: private to the dist.
	tagged := pairedTestSet("9")
	taggedID := stageAssetGeneration(t, s, "9", tagged)
	retired := pairedTestSet("retired")
	retiredID := stageAssetGeneration(t, s, "retired", retired)
	if _, err := s.selectCurrentWithAssets(taggedID, retiredID, retired); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "dist", assetLedgerName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("asset ledger has mode %v", info.Mode())
	}
}

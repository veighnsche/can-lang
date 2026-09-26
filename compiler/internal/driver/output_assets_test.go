package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// pairedTestSet builds one fake paired file set with stable digests derived
// from its tag. Sets with different tags share no digest.
func pairedTestSet(tag string) []pairedAssetFile {
	file := func(logical, ext, media string) pairedAssetFile {
		digest := hashBytes([]byte(tag + "\x00" + logical))
		base := map[string]string{".js": "browser.js", ".js.map": "browser.js.map", ".json": "table.json"}[ext]
		return pairedAssetFile{
			Logical:   logical,
			Route:     "/__can/assets/" + digest + ext,
			Digest:    digest,
			MediaType: media,
			File:      "assets/" + digest + "/" + base,
		}
	}
	return []pairedAssetFile{
		file("browser/browser.js", ".js", "text/javascript"),
		file("browser/browser.js.map", ".js.map", "application/json"),
		file("diagnostics/table.json", ".json", "application/json"),
	}
}

func stageAssetGeneration(t *testing.T, s *OutputStore, tag string, files []pairedAssetFile) string {
	t.Helper()
	artifacts := []ir.Artifact{{Path: "entry.ts", Bytes: []byte("export const tag = " + tag + ";\n")}}
	for _, file := range files {
		body := tag + "\x00" + file.Logical
		if hashBytes([]byte(body)) != file.Digest {
			t.Fatalf("test set %s digest mismatch for %s", tag, file.Route)
		}
		artifacts = append(artifacts, ir.Artifact{Path: file.File, Bytes: []byte(body)})
	}
	if len(files) != 0 {
		record, err := json.Marshal(struct {
			SchemaVersion int               `json:"schemaVersion"`
			Kind          string            `json:"kind"`
			BrowserBuild  string            `json:"browserBuildId"`
			Generation    string            `json:"generation"`
			Lock          string            `json:"lock"`
			Entry         string            `json:"entry"`
			Table         string            `json:"table"`
			Files         []pairedAssetFile `json:"files"`
		}{1, "can.browser-pairing", strings.Repeat("b", 64), strings.Repeat("c", 64), "", files[0].Route, files[2].Route, files})
		if err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, ir.Artifact{Path: pairingRecordPath, Bytes: append(record, '\n')})
	}
	prepared, err := PrepareOutput(s.BuildInputs(strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64), strings.Repeat("a", 64)), "entry.ts", artifacts)
	if err != nil {
		t.Fatal(err)
	}
	prepared.validated = true
	id, _, err := s.Stage(prepared)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func readLedger(t *testing.T, s *OutputStore) assetLedger {
	t.Helper()
	ledger, _, err := s.readAssetLedger()
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func TestAssetRetentionLifecycle(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	second := pairedTestSet("2")
	firstID := stageAssetGeneration(t, s, "1", first)
	prior, err := s.currentBuildID()
	if err != nil || prior != "" {
		t.Fatalf("prior = %q %v", prior, err)
	}
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.dist.Lstat(assetLedgerName); !os.IsNotExist(err) {
		t.Fatal("first paired build wrote a ledger with nothing to retain")
	}
	secondID := stageAssetGeneration(t, s, "2", second)
	if _, err := s.selectCurrentWithAssets(firstID, secondID, second); err != nil {
		t.Fatal(err)
	}
	ledger := readLedger(t, s)
	if len(ledger.Retained) != 3 {
		t.Fatalf("retained = %d", len(ledger.Retained))
	}
	before := time.Now().UnixMilli()
	for _, entry := range ledger.Retained {
		if entry.ReplacedAt > before || before-entry.ReplacedAt > 60_000 {
			t.Fatalf("replacedAt = %d, now = %d", entry.ReplacedAt, before)
		}
		data, err := readOutputRegular(s.dist, entry.File)
		if err != nil || hashBytes(data) != entry.Digest {
			t.Fatalf("durable copy of %s failed verification: %v", entry.Route, err)
		}
	}
	// The replaced generation stays prunable: retention no longer depends
	// on it once the durable copies land.
	if err := s.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "builds", firstID)); !os.IsNotExist(err) {
		t.Fatal("superseded generation survived prune")
	}
	ledger = readLedger(t, s)
	if len(ledger.Retained) != 3 {
		t.Fatal("prune dropped retention")
	}
	for _, entry := range ledger.Retained {
		if _, err := readOutputRegular(s.dist, entry.File); err != nil {
			t.Fatalf("prune deleted durable bytes for %s", entry.Route)
		}
	}
}

func TestAssetRetentionExpiryBound(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	second := pairedTestSet("2")
	third := pairedTestSet("3")
	firstID := stageAssetGeneration(t, s, "1", first)
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	secondID := stageAssetGeneration(t, s, "2", second)
	if _, err := s.selectCurrentWithAssets(firstID, secondID, second); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	ledger := readLedger(t, s)
	// One row comfortably before the bound and one comfortably past it;
	// the exact bound itself is covered deterministically by
	// TestAssetRetainedBound, since staging time would blur a 1ms margin.
	// Rows are digest-ordered, so address each by its file digest.
	moment := map[string]int64{
		first[0].Digest: now - assetRetentionMs + 60_000,
		first[1].Digest: now - assetRetentionMs + 60_000,
		first[2].Digest: now - assetRetentionMs - 60_000,
	}
	for i := range ledger.Retained {
		ledger.Retained[i].ReplacedAt = moment[ledger.Retained[i].Digest]
	}
	if err := s.atomicMetadata(assetLedgerName, assetLedgerTemp, ledger); err != nil {
		t.Fatal(err)
	}
	thirdID := stageAssetGeneration(t, s, "3", third)
	if _, err := s.selectCurrentWithAssets(secondID, thirdID, third); err != nil {
		t.Fatal(err)
	}
	kept := map[string]bool{}
	for _, entry := range readLedger(t, s).Retained {
		kept[entry.Digest] = true
	}
	v1 := map[string]bool{}
	for _, file := range first {
		v1[file.Digest] = true
	}
	if !kept[first[0].Digest] || !kept[first[1].Digest] || kept[first[2].Digest] {
		t.Fatalf("expiry bound misapplied: %v", kept)
	}
	if _, err := readOutputRegular(s.dist, assetStoreDir+"/"+first[2].Digest+".json"); !os.IsNotExist(err) {
		t.Fatal("expired durable bytes survived collection")
	}
	for _, file := range second {
		if !kept[file.Digest] {
			t.Fatalf("newly replaced route %s missing from retention", file.Route)
		}
	}
}

func TestAssetRetainedBound(t *testing.T) {
	now := int64(1790314240936)
	if !assetRetained(now, now-assetRetentionMs+1) {
		t.Fatal("before the bound does not serve")
	}
	if !assetRetained(now, now-assetRetentionMs) {
		t.Fatal("at the bound does not serve")
	}
	if assetRetained(now, now-assetRetentionMs-1) {
		t.Fatal("past the bound serves")
	}
}

func TestAssetPublicationRestoresPriorOnFailure(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	second := pairedTestSet("2")
	firstID := stageAssetGeneration(t, s, "1", first)
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	// Tamper with the prior generation after publication: retention must
	// fail closed and the prior set must stay selected.
	full := filepath.Join(root, "dist", "builds", firstID, filepath.FromSlash(first[0].File))
	// Staged content is read-only and inode-shared: honest tampering
	// replaces the path instead of mutating shared bytes in place.
	if err := os.Remove(full); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	secondID := stageAssetGeneration(t, s, "2", second)
	if _, err := s.selectCurrentWithAssets(firstID, secondID, second); err == nil {
		t.Fatal("tampered prior generation admitted")
	}
	current, err := s.currentBuildID()
	if err != nil || current != firstID {
		t.Fatalf("current = %q, want prior %q", current, firstID)
	}
	if _, err := s.dist.Lstat(pendingAssetsName); !os.IsNotExist(err) {
		t.Fatal("failed publication left a pending record")
	}
}

func TestAssetPublicationRecoversAfterCrash(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	second := pairedTestSet("2")
	firstID := stageAssetGeneration(t, s, "1", first)
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	secondID := stageAssetGeneration(t, s, "2", second)
	// Simulate a crash between selection and retention: the pending record
	// exists and the new generation is already current.
	if err := s.atomicMetadata(pendingAssetsName, ".pending-assets.tmp", outputPendingAssets{SchemaVersion: 1, Kind: "can.pending-assets", Prior: firstID, Current: secondID, Files: second}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SelectCurrent(secondID); err != nil {
		t.Fatal(err)
	}
	if err := s.Recover(); err != nil {
		t.Fatal(err)
	}
	if len(readLedger(t, s).Retained) != 3 {
		t.Fatal("crash recovery skipped retention")
	}
	if _, err := s.dist.Lstat(pendingAssetsName); !os.IsNotExist(err) {
		t.Fatal("recovery left a pending record")
	}
	// A pending record for a selection that never landed is abandoned.
	thirdID := stageAssetGeneration(t, s, "3", pairedTestSet("3"))
	if err := s.atomicMetadata(pendingAssetsName, ".pending-assets.tmp", outputPendingAssets{SchemaVersion: 1, Kind: "can.pending-assets", Prior: secondID, Current: thirdID, Files: pairedTestSet("3")}); err != nil {
		t.Fatal(err)
	}
	if err := s.Recover(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.dist.Lstat(pendingAssetsName); !os.IsNotExist(err) {
		t.Fatal("abandoned pending record survived recovery")
	}
	if current, _ := s.currentBuildID(); current != secondID {
		t.Fatal("abandoned publication moved current")
	}
}

func TestAssetPublicationRetiresOnUnpairedBuild(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	firstID := stageAssetGeneration(t, s, "1", first)
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	plainID := stageAssetGeneration(t, s, "2", nil)
	if _, err := s.selectCurrentWithAssets(firstID, plainID, nil); err != nil {
		t.Fatal(err)
	}
	if len(readLedger(t, s).Retained) != 3 {
		t.Fatal("unpaired build dropped the paired set without retention")
	}
}

func TestAssetPublicationSkipsQuietBuilds(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	firstID := stageAssetGeneration(t, s, "1", nil)
	if _, err := s.selectCurrentWithAssets("", firstID, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := outputEntries(s.dist, ".")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	for _, unexpected := range []string{assetLedgerName, pendingAssetsName, assetStoreDir} {
		if names[unexpected] {
			t.Fatalf("quiet build left %s", unexpected)
		}
	}
}

func TestAssetCleanResetsRetention(t *testing.T) {
	root := outputProject(t)
	s := outputBegin(t, root)
	first := pairedTestSet("1")
	firstID := stageAssetGeneration(t, s, "1", first)
	if _, err := s.selectCurrentWithAssets("", firstID, first); err != nil {
		t.Fatal(err)
	}
	secondID := stageAssetGeneration(t, s, "2", pairedTestSet("2"))
	if _, err := s.selectCurrentWithAssets(firstID, secondID, pairedTestSet("2")); err != nil {
		t.Fatal(err)
	}
	if err := s.Clean(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{assetLedgerName, pendingAssetsName, assetStoreDir, "current.json"} {
		if _, err := s.dist.Lstat(name); !os.IsNotExist(err) {
			t.Fatalf("clean left %s", name)
		}
	}
	// The store rebuilds cleanly afterward.
	thirdID := stageAssetGeneration(t, s, "3", nil)
	if _, err := s.selectCurrentWithAssets("", thirdID, nil); err != nil {
		t.Fatal(err)
	}
}

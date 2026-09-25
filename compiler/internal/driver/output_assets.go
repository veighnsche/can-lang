package driver

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// The durable asset store keeps replaced digest URLs servable for at least
// seven days after replacement. dist/assets/<digest><ext> holds
// content-addressed bytes outside any generation, and dist/assets.json is
// the retention ledger binding each route to its replacement moment.
// Generations stay timestamp-free so identical inputs rebuild identically;
// only this mutable dist metadata carries wall-clock time.
//
// Publication is atomic across crashes: selectCurrentWithAssets records the
// pending set, selects the generation, then applies retention. Recover
// completes a pending application when the selection landed and abandons it
// otherwise. Prune calls Recover first, so a prior generation is never
// deleted before its replaced bytes reach the durable store.

const (
	assetLedgerName   = "assets.json"
	assetLedgerTemp   = ".assets.tmp"
	assetStoreDir     = "assets"
	pendingAssetsName = "pending-assets.json"
	pairingRecordPath = "browser/pairing.json"
)

// assetLedgerEntry is one retained route: the digest-addressed file in the
// durable store plus the unix-millisecond moment its generation stopped
// being current.
type assetLedgerEntry struct {
	Digest     string `json:"digest"`
	Route      string `json:"route"`
	MediaType  string `json:"mediaType"`
	File       string `json:"file"`
	ReplacedAt int64  `json:"replacedAt"`
}

type assetLedger struct {
	SchemaVersion int                `json:"schemaVersion"`
	Kind          string             `json:"kind"`
	Retained      []assetLedgerEntry `json:"retained"`
}

// outputPendingAssets is the crash-safe publication intent: the newly
// selected generation plus the paired set it serves. Retention applies
// only after the selection lands.
type outputPendingAssets struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	Prior         string            `json:"prior"`
	Current       string            `json:"current"`
	Files         []pairedAssetFile `json:"files"`
}

// currentBuildID reports the selected production build, or "" when no
// generation is selected yet.
func (s *OutputStore) currentBuildID() (string, error) {
	raw, err := readOutputRegular(s.dist, "current.json")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var current outputCurrent
	if err := decodeOutput(raw, &current); err != nil || current.SchemaVersion != 1 || !digestPattern.MatchString(current.BuildID) {
		return "", fmt.Errorf("invalid current manifest")
	}
	return current.BuildID, nil
}

// needsAssetPublication reports whether the selection carries any
// retention duty: a new paired set, an existing ledger to collect, or a
// prior paired set to retire. Builds with no duty keep the historical
// select-only path byte for byte.
func (s *OutputStore) needsAssetPublication(prior string, files []pairedAssetFile) bool {
	if len(files) != 0 {
		return true
	}
	if _, err := s.dist.Lstat(assetLedgerName); err == nil {
		return true
	}
	if prior == "" {
		return false
	}
	raw, err := readOutputRegular(s.dist, "builds/"+prior+"/manifest.json")
	if err != nil {
		return true
	}
	var manifest OutputManifest
	if err := decodeOutput(raw, &manifest); err != nil {
		return true
	}
	_, paired := manifest.Files[pairingRecordPath]
	return paired
}

// selectCurrentWithAssets publishes the report, the generation, and the
// entire asset set as one atomic step. Any failure preserves the prior
// set: selection failures discard the staged generation, and a retention
// failure after selection restores the prior current before reporting.
func (s *OutputStore) selectCurrentWithAssets(prior, id string, files []pairedAssetFile) (string, error) {
	if !s.needsAssetPublication(prior, files) {
		directory, err := s.SelectCurrent(id)
		if err != nil {
			_ = s.DiscardGeneration(id)
		}
		return directory, err
	}
	ordered := append([]pairedAssetFile{}, files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Route < ordered[j].Route })
	seen := map[string]bool{}
	for _, file := range ordered {
		if seen[file.Digest] || seen[file.Route] {
			_ = s.DiscardGeneration(id)
			return "", fmt.Errorf("paired asset set publishes %s twice", file.Route)
		}
		seen[file.Digest] = true
		seen[file.Route] = true
	}
	if err := s.atomicMetadata(pendingAssetsName, ".pending-assets.tmp", outputPendingAssets{SchemaVersion: 1, Kind: "can.pending-assets", Prior: prior, Current: id, Files: ordered}); err != nil {
		_ = s.DiscardGeneration(id)
		return "", err
	}
	directory, err := s.SelectCurrent(id)
	if err != nil {
		s.abandonPendingAssets()
		_ = s.DiscardGeneration(id)
		return "", err
	}
	if err := s.applyPendingAssets(); err != nil {
		applyErr := err
		if prior != "" {
			if _, rerr := s.SelectCurrent(prior); rerr == nil {
				s.abandonPendingAssets()
				_ = s.DiscardGeneration(id)
				return "", applyErr
			} else {
				return "", fmt.Errorf("%v; prior restore failed: %w", applyErr, rerr)
			}
		}
		_ = s.DiscardGeneration(id)
		_ = s.dist.Remove("current.json")
		_ = syncOutputDir(s.dist, ".")
		return "", applyErr
	}
	return directory, nil
}

func (s *OutputStore) abandonPendingAssets() {
	_ = s.dist.Remove(pendingAssetsName)
	_ = syncOutputDir(s.dist, ".")
}

// applyPendingAssets completes the retention half of publication. It is
// idempotent under the project lock: durable copies are verified before
// reuse, ledger rows keep their first replacement moment, and the ledger
// write itself is atomic, so a repeated apply after a crash converges.
func (s *OutputStore) applyPendingAssets() error {
	raw, err := readOutputRegular(s.dist, pendingAssetsName)
	if err != nil {
		return fmt.Errorf("pending asset publication is missing")
	}
	var pending outputPendingAssets
	if err := decodeOutput(raw, &pending); err != nil || pending.SchemaVersion != 1 || pending.Kind != "can.pending-assets" || !digestPattern.MatchString(pending.Current) || (pending.Prior != "" && !digestPattern.MatchString(pending.Prior)) {
		return fmt.Errorf("invalid pending asset publication")
	}
	current, err := s.currentBuildID()
	if err != nil {
		return err
	}
	if current != pending.Current {
		return fmt.Errorf("pending asset publication does not match production current")
	}
	now := time.Now().UnixMilli()
	priorFiles, priorBytes, err := s.priorPairedSet(pending.Prior)
	if err != nil {
		return err
	}
	ledger, existed, err := s.readAssetLedger()
	if err != nil {
		return err
	}
	currentDigests := map[string]bool{}
	for _, file := range pending.Files {
		currentDigests[file.Digest] = true
	}
	retained := make([]assetLedgerEntry, 0, len(ledger.Retained)+len(priorFiles))
	kept := map[string]bool{}
	for _, entry := range ledger.Retained {
		if err := validateLedgerEntry(entry); err != nil {
			return err
		}
		if currentDigests[entry.Digest] || now-entry.ReplacedAt > assetRetentionMs {
			continue
		}
		retained = append(retained, entry)
		kept[entry.Digest] = true
	}
	var added []assetLedgerEntry
	for _, file := range priorFiles {
		if currentDigests[file.Digest] || kept[file.Digest] {
			continue
		}
		digest, ext, err := pairedRouteExtension(file.Route)
		if err != nil || digest != file.Digest {
			return fmt.Errorf("prior paired route %q does not bind its digest", file.Route)
		}
		added = append(added, assetLedgerEntry{Digest: file.Digest, Route: file.Route, MediaType: file.MediaType, File: assetStoreDir + "/" + file.Digest + ext, ReplacedAt: now})
		kept[file.Digest] = true
	}
	retained = append(retained, added...)
	sort.Slice(retained, func(i, j int) bool {
		if retained[i].ReplacedAt != retained[j].ReplacedAt {
			return retained[i].ReplacedAt < retained[j].ReplacedAt
		}
		return retained[i].Digest < retained[j].Digest
	})
	if len(added) != 0 {
		if err := s.ensureAssetStore(); err != nil {
			return err
		}
		for _, entry := range added {
			data, ok := priorBytes[entry.Digest]
			if !ok {
				return fmt.Errorf("prior paired bytes for %s are missing", entry.Route)
			}
			if err := s.storeDurableBytes(entry.File, data, entry.Digest); err != nil {
				return err
			}
		}
		if err := syncOutputDir(s.dist, assetStoreDir); err != nil {
			return err
		}
	}
	if existed || len(retained) != 0 {
		if err := s.atomicMetadata(assetLedgerName, assetLedgerTemp, assetLedger{SchemaVersion: 1, Kind: "can.asset-ledger", Retained: retained}); err != nil {
			return err
		}
		if err := s.sweepAssetStore(retained); err != nil {
			return err
		}
	}
	if err := s.dist.Remove(pendingAssetsName); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncOutputDir(s.dist, ".")
}

// recoverPendingAssets completes a publication interrupted by a crash when
// its selection landed, and abandons it when the selection never did.
// Orphaned durable copies from an abandoned publication carry no ledger
// row, are never served, and are swept by the next successful apply.
func (s *OutputStore) recoverPendingAssets() error {
	raw, err := readOutputRegular(s.dist, pendingAssetsName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var pending outputPendingAssets
	if err := decodeOutput(raw, &pending); err != nil || pending.SchemaVersion != 1 || pending.Kind != "can.pending-assets" {
		return fmt.Errorf("invalid pending asset publication")
	}
	current, err := s.currentBuildID()
	if err != nil {
		return err
	}
	if current == pending.Current {
		return s.applyPendingAssets()
	}
	s.abandonPendingAssets()
	return nil
}

// priorPairedSet reads the paired set of the generation being replaced,
// bound to its own manifest and rehashed from its own bytes. A generation
// without a pairing record is unpaired and contributes an empty set.
func (s *OutputStore) priorPairedSet(prior string) ([]pairedAssetFile, map[string][]byte, error) {
	if prior == "" {
		return nil, nil, nil
	}
	treeRaw, err := readOutputRegular(s.dist, "builds/"+prior+"/manifest.json")
	if err != nil {
		return nil, nil, fmt.Errorf("prior generation is unreadable: %w", err)
	}
	var tree OutputManifest
	if err := decodeOutput(treeRaw, &tree); err != nil {
		return nil, nil, fmt.Errorf("prior generation manifest is invalid")
	}
	recorded, paired := tree.Files[pairingRecordPath]
	if !paired {
		return nil, nil, nil
	}
	encoded, err := readOutputRegular(s.dist, "builds/"+prior+"/"+pairingRecordPath)
	if err != nil {
		return nil, nil, fmt.Errorf("prior pairing record is unreadable: %w", err)
	}
	if hashBytes(encoded) != recorded {
		return nil, nil, fmt.Errorf("prior pairing record failed hash verification")
	}
	var pairing struct {
		SchemaVersion int               `json:"schemaVersion"`
		Kind          string            `json:"kind"`
		BrowserBuild  string            `json:"browserBuildId"`
		Generation    string            `json:"generation"`
		Lock          string            `json:"lock"`
		Entry         string            `json:"entry"`
		Table         string            `json:"table"`
		Files         []pairedAssetFile `json:"files"`
	}
	if err := decodeOutput(encoded, &pairing); err != nil || pairing.SchemaVersion != 1 || pairing.Kind != "can.browser-pairing" {
		return nil, nil, fmt.Errorf("prior pairing record is invalid")
	}
	if pairing.Generation != prior || len(pairing.Files) == 0 {
		return nil, nil, fmt.Errorf("prior pairing record does not match its generation")
	}
	routes := map[string]bool{}
	bytes := map[string][]byte{}
	for _, file := range pairing.Files {
		digest, ext, err := pairedRouteExtension(file.Route)
		if err != nil || digest != file.Digest {
			return nil, nil, fmt.Errorf("prior paired route %q does not bind its digest", file.Route)
		}
		if want, ok := pairedMedia[ext]; !ok || want != file.MediaType {
			return nil, nil, fmt.Errorf("prior paired route %q carries a wrong media type", file.Route)
		}
		if !strings.HasPrefix(file.File, "assets/"+file.Digest+"/") {
			return nil, nil, fmt.Errorf("prior paired file %q is outside the asset namespace", file.File)
		}
		if err := outputPath(file.File); err != nil {
			return nil, nil, fmt.Errorf("prior paired file is unsafe: %w", err)
		}
		if routes[file.Route] {
			return nil, nil, fmt.Errorf("prior pairing record publishes %s twice", file.Route)
		}
		routes[file.Route] = true
		data, err := readOutputRegular(s.dist, "builds/"+prior+"/"+file.File)
		if err != nil || hashBytes(data) != file.Digest {
			return nil, nil, fmt.Errorf("prior paired file %s failed hash verification", file.File)
		}
		bytes[file.Digest] = data
	}
	if !routes[pairing.Entry] || !routes[pairing.Table] {
		return nil, nil, fmt.Errorf("prior pairing record lacks its entry or table route")
	}
	return pairing.Files, bytes, nil
}

func (s *OutputStore) readAssetLedger() (assetLedger, bool, error) {
	raw, err := readOutputRegular(s.dist, assetLedgerName)
	if err != nil {
		if os.IsNotExist(err) {
			return assetLedger{SchemaVersion: 1, Kind: "can.asset-ledger", Retained: []assetLedgerEntry{}}, false, nil
		}
		return assetLedger{}, false, err
	}
	var ledger assetLedger
	if err := decodeOutput(raw, &ledger); err != nil || ledger.SchemaVersion != 1 || ledger.Kind != "can.asset-ledger" {
		return assetLedger{}, false, fmt.Errorf("asset ledger is invalid")
	}
	if ledger.Retained == nil {
		ledger.Retained = []assetLedgerEntry{}
	}
	return ledger, true, nil
}

func validateLedgerEntry(entry assetLedgerEntry) error {
	digest, ext, err := pairedRouteExtension(entry.Route)
	if err != nil || digest != entry.Digest {
		return fmt.Errorf("asset ledger route %q does not bind its digest", entry.Route)
	}
	if want, ok := pairedMedia[ext]; !ok || want != entry.MediaType {
		return fmt.Errorf("asset ledger route %q carries a wrong media type", entry.Route)
	}
	if entry.File != assetStoreDir+"/"+entry.Digest+ext {
		return fmt.Errorf("asset ledger file %q is outside the durable store", entry.File)
	}
	return nil
}

func (s *OutputStore) ensureAssetStore() error {
	if info, err := s.dist.Lstat(assetStoreDir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("asset store is not a real directory")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return s.dist.Mkdir(assetStoreDir, 0700)
}

// storeDurableBytes installs one content-addressed durable copy. An
// existing copy is re-verified and only rewritten when tampered, so a
// repeated apply after a crash converges on identical bytes.
func (s *OutputStore) storeDurableBytes(name string, data []byte, digest string) error {
	if hashBytes(data) != digest {
		return fmt.Errorf("durable asset bytes failed hash verification")
	}
	if _, err := s.dist.Lstat(name); err == nil {
		present, err := readOutputRegular(s.dist, name)
		if err == nil && hashBytes(present) == digest {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	temp := assetStoreDir + "/.tmp-" + digest
	_ = s.dist.Remove(temp)
	if err := writeOutputNew(s.dist, temp, data); err != nil {
		return err
	}
	return s.dist.Rename(temp, name)
}

// sweepAssetStore deletes every durable copy the ledger no longer binds,
// plus crash-debris temporaries. Any other unknown file fails closed for
// operator inspection instead of being silently collected.
func (s *OutputStore) sweepAssetStore(retained []assetLedgerEntry) error {
	if _, err := s.dist.Lstat(assetStoreDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	live := map[string]bool{}
	for _, entry := range retained {
		live[entry.File] = true
	}
	entries, err := outputEntries(s.dist, assetStoreDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := assetStoreDir + "/" + entry.Name()
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			if err := s.dist.Remove(name); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if live[name] {
			continue
		}
		rest, ok := strings.CutPrefix(name, assetStoreDir+"/")
		matched := false
		if ok {
			for _, ext := range []string{".js.map", ".js", ".json"} {
				if digest, cut := strings.CutSuffix(rest, ext); cut && digestPattern.MatchString(digest) {
					matched = true
				}
			}
		}
		if !matched {
			return fmt.Errorf("unknown durable asset %s; preserve it before cleanup", entry.Name())
		}
		if err := s.dist.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return syncOutputDir(s.dist, assetStoreDir)
}

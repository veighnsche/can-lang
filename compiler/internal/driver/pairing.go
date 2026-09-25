package driver

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// assetRetentionMs is the minimum servable lifetime of a replaced digest URL:
// seven days. Serve-time expiry and build-time collection both honor it, and
// a route exactly at the bound still serves.
const assetRetentionMs = int64(7 * 24 * 60 * 60 * 1000)

// pairedAssetFile is one verified browser byte bound to its served route. The
// File path is generation-relative (assets/<digest>/<base>); the durable
// retention store rekeys the same digest as assets/<digest><ext>.
type pairedAssetFile struct {
	Logical   string `json:"logical"`
	Route     string `json:"route"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	File      string `json:"file"`
}

// browserPairing is one verified --browser-manifest input: the exact browser
// build identity plus every published byte carried in memory. Publication
// uses these bytes; nothing is re-read from the browser dist for staging,
// and a byte-compare re-read guards the verify-to-publish window.
type browserPairing struct {
	buildID      string
	manifestHash string
	generationID string
	lock         string
	entry        string
	table        string
	files        []pairedAssetFile
	assets       []project.Asset
	pairingJSON  []byte
	manifestPath string
	generation   string
	manifest     []byte
	treeManifest []byte
	inputs       BuildInputs
	instances    []BrowserLockedInstance
	byLogical    map[string][]byte
}

var pairedMedia = map[string]string{
	".js":     "text/javascript",
	".js.map": "application/json",
	".json":   "application/json",
}

// pairedRouteExtension splits a served browser route into its content digest
// and published extension. The .js.map suffix wins over .js so maps never
// alias their script route.
func pairedRouteExtension(route string) (string, string, error) {
	rest, ok := strings.CutPrefix(route, "/__can/assets/")
	if !ok {
		return "", "", fmt.Errorf("browser route %q is outside the asset namespace", route)
	}
	for _, ext := range []string{".js.map", ".js", ".json"} {
		if digest, ok := strings.CutSuffix(rest, ext); ok && digestPattern.MatchString(digest) {
			return digest, ext, nil
		}
	}
	return "", "", fmt.Errorf("browser route %q is not a digest asset route", route)
}

// verifyBrowserManifest reads and verifies the browser manifest named by the
// explicit --browser-manifest path. The path must name the browser/manifest.json
// of one staged browser generation; every file hash, the generation binding,
// the reference graph, the post-bundle audit, and the shared locked snapshot
// are verified before anything stages or publishes.
func verifyBrowserManifest(manifestPath string, serverGraph *project.Graph) (*browserPairing, error) {
	if serverGraph == nil {
		return nil, fmt.Errorf("browser pairing requires the server project graph")
	}
	absolute, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, err
	}
	if filepath.Base(absolute) != "manifest.json" || filepath.Base(filepath.Dir(absolute)) != "browser" {
		return nil, fmt.Errorf("browser manifest path must name the browser/manifest.json of one staged browser generation")
	}
	generation := filepath.Dir(filepath.Dir(absolute))
	manifest, err := readBoundedFile(absolute, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("browser manifest is unreadable: %w", err)
	}
	var decoded browserManifest
	if err := decodeOutput(manifest, &decoded); err != nil {
		return nil, fmt.Errorf("invalid browser manifest: %w", err)
	}
	if decoded.SchemaVersion != 1 || decoded.Kind != "can.browser-manifest" {
		return nil, fmt.Errorf("invalid browser manifest identity")
	}
	identity, err := json.Marshal(decoded.Files)
	if err != nil {
		return nil, err
	}
	if hashBytes(append([]byte("can-browser-bundle-v1\x00"), identity...)) != decoded.BrowserBuildID || !digestPattern.MatchString(decoded.BrowserBuildID) {
		return nil, fmt.Errorf("browser manifest build identity mismatch")
	}
	if len(decoded.Files) == 0 {
		return nil, fmt.Errorf("browser manifest publishes no files")
	}
	seenLogical, seenRoute := map[string]bool{}, map[string]bool{}
	scripts, maps := map[string]bool{}, map[string]bool{}
	for _, file := range decoded.Files {
		if err := outputPath(file.Path); err != nil {
			return nil, fmt.Errorf("browser manifest names an unsafe path: %w", err)
		}
		if !digestPattern.MatchString(file.SHA256) {
			return nil, fmt.Errorf("browser manifest carries an invalid digest for %s", file.Path)
		}
		if file.Route != browserAssetRoute(file.Path, file.SHA256) {
			return nil, fmt.Errorf("browser manifest route %q does not bind %s", file.Route, file.Path)
		}
		if seenLogical[file.Path] || seenRoute[file.Route] {
			return nil, fmt.Errorf("browser manifest publishes %s twice", file.Path)
		}
		seenLogical[file.Path] = true
		seenRoute[file.Route] = true
		switch {
		case file.Path == decoded.Table:
			if !strings.HasSuffix(file.Path, ".json") {
				return nil, fmt.Errorf("browser diagnostic table %s is not a published .json file", file.Path)
			}
		case strings.HasSuffix(file.Path, ".js.map"):
			maps[strings.TrimSuffix(file.Path, ".map")] = true
		case strings.HasSuffix(file.Path, ".js"):
			scripts[file.Path] = true
		default:
			return nil, fmt.Errorf("browser manifest publishes unaccounted file %s", file.Path)
		}
	}
	if !seenLogical[decoded.Entry] || !strings.HasSuffix(decoded.Entry, ".js") {
		return nil, fmt.Errorf("browser manifest entry %s is not published", decoded.Entry)
	}
	if !seenLogical[decoded.Table] {
		return nil, fmt.Errorf("browser manifest table %s is not published", decoded.Table)
	}
	for script := range scripts {
		if !maps[script] {
			return nil, fmt.Errorf("paired script %s lacks its source map", script)
		}
	}
	for script := range maps {
		if !scripts[script] {
			return nil, fmt.Errorf("paired source map %s.map has no published script", script)
		}
	}
	treeRaw, err := readBoundedFile(filepath.Join(generation, "manifest.json"), 8<<20)
	if err != nil {
		return nil, fmt.Errorf("browser generation manifest is unreadable: %w", err)
	}
	var tree OutputManifest
	if err := decodeOutput(treeRaw, &tree); err != nil {
		return nil, fmt.Errorf("invalid browser generation manifest: %w", err)
	}
	if err := validateOutputManifest(tree, true); err != nil {
		return nil, fmt.Errorf("invalid browser generation manifest: %w", err)
	}
	if tree.Entry != browser.BrowserEntry {
		return nil, fmt.Errorf("paired generation entry %q is not a browser build", tree.Entry)
	}
	if tree.Files[browserBundleManifest] != hashBytes(manifest) {
		return nil, fmt.Errorf("browser manifest is not bound to its generation")
	}
	root, err := os.OpenRoot(generation)
	if err != nil {
		return nil, fmt.Errorf("browser generation is unreadable: %w", err)
	}
	validateErr := validateOutputTree(root, ".", tree, false, false)
	root.Close()
	if validateErr != nil {
		return nil, fmt.Errorf("browser generation failed verification: %w", validateErr)
	}
	byLogical := map[string][]byte{}
	assembled := browser.Bundle{Entry: decoded.Entry, Table: decoded.Table, Digests: map[string]string{}}
	for _, file := range decoded.Files {
		data, err := readBoundedFile(filepath.Join(generation, filepath.FromSlash(file.Path)), 32<<20)
		if err != nil {
			return nil, fmt.Errorf("paired file %s is unreadable: %w", file.Path, err)
		}
		if hashBytes(data) != file.SHA256 {
			return nil, fmt.Errorf("paired file %s failed hash verification", file.Path)
		}
		byLogical[file.Path] = data
		assembled.Files = append(assembled.Files, browser.BundleFile{Path: file.Path, Bytes: data})
		assembled.Digests[file.Path] = file.SHA256
	}
	if err := browser.AuditBundle(assembled); err != nil {
		return nil, fmt.Errorf("paired browser bundle failed audit: %w", err)
	}
	if err := bindSharedLock(decoded.LockedInstances, serverGraph); err != nil {
		return nil, err
	}
	pairing := &browserPairing{
		buildID:      decoded.BrowserBuildID,
		manifestHash: hashBytes(manifest),
		generationID: tree.BuildID,
		lock:         decoded.Lock,
		manifestPath: absolute,
		generation:   generation,
		manifest:     manifest,
		treeManifest: treeRaw,
		byLogical:    byLogical,
		inputs:       BuildInputs{Source: decoded.Inputs.Source, Dependencies: decoded.Inputs.Dependencies, Catalogue: decoded.Inputs.Catalogue, Compiler: decoded.Inputs.Compiler, Runtime: decoded.Inputs.Runtime, Options: decoded.Inputs.Options},
	}
	for _, file := range decoded.Files {
		digest, ext, err := pairedRouteExtension(file.Route)
		if err != nil || digest != file.SHA256 {
			return nil, fmt.Errorf("browser manifest route %q does not bind its digest", file.Route)
		}
		media, ok := pairedMedia[ext]
		if !ok {
			return nil, fmt.Errorf("browser manifest route %q has no served media type", file.Route)
		}
		base := path.Base(file.Path)
		artifact := "assets/" + file.SHA256 + "/" + base
		if err := outputPath(artifact); err != nil {
			return nil, fmt.Errorf("paired artifact path is unsafe: %w", err)
		}
		pairing.files = append(pairing.files, pairedAssetFile{Logical: file.Path, Route: file.Route, Digest: file.SHA256, MediaType: media, File: artifact})
		pairing.assets = append(pairing.assets, project.Asset{SafeName: base, Digest: file.SHA256, MediaType: media, URL: file.Route, Artifact: artifact, Bytes: append([]byte(nil), byLogical[file.Path]...)})
		if file.Path == decoded.Entry {
			pairing.entry = file.Route
		}
		if file.Path == decoded.Table {
			pairing.table = file.Route
		}
	}
	sort.Slice(pairing.files, func(i, j int) bool { return pairing.files[i].Route < pairing.files[j].Route })
	sort.Slice(pairing.assets, func(i, j int) bool { return pairing.assets[i].URL < pairing.assets[j].URL })
	for _, entry := range decoded.LockedInstances {
		pairing.instances = append(pairing.instances, BrowserLockedInstance{Instance: entry.Instance, Lineage: entry.Lineage, ManifestSHA256: entry.ManifestSHA256, SourceSHA256: entry.SourceSHA256, FixturesSHA256: entry.FixturesSHA256})
	}
	sort.Slice(pairing.instances, func(i, j int) bool { return pairing.instances[i].Instance < pairing.instances[j].Instance })
	encoded, err := json.Marshal(struct {
		SchemaVersion int               `json:"schemaVersion"`
		Kind          string            `json:"kind"`
		BrowserBuild  string            `json:"browserBuildId"`
		Generation    string            `json:"generation"`
		Lock          string            `json:"lock"`
		Entry         string            `json:"entry"`
		Table         string            `json:"table"`
		Files         []pairedAssetFile `json:"files"`
	}{1, "can.browser-pairing", pairing.buildID, pairing.generationID, pairing.lock, pairing.entry, pairing.table, pairing.files})
	if err != nil {
		return nil, err
	}
	pairing.pairingJSON = append(encoded, '\n')
	return pairing, nil
}

// bindSharedLock requires every locked instance shared by the browser and
// server graphs to pin identical lineage and content. The two roots may
// differ (the grid and its server are separate projects); only their shared
// snapshot must agree. An empty intersection passes so a dependency-free
// browser fixture can pair while the grid migrates.
func bindSharedLock(browser []browserLockedInstance, serverGraph *project.Graph) error {
	seen := map[string]bool{}
	for _, entry := range browser {
		if entry.Instance == "" || seen[entry.Instance] {
			return fmt.Errorf("browser manifest carries a duplicate locked instance")
		}
		seen[entry.Instance] = true
		pinned, shared := serverGraph.Lock.Projects[entry.Instance]
		if !shared {
			continue
		}
		if pinned.Lineage != entry.Lineage || pinned.ManifestSHA256 != entry.ManifestSHA256 || pinned.SourceSHA256 != entry.SourceSHA256 || pinned.FixturesSHA256 != entry.FixturesSHA256 {
			return fmt.Errorf("browser manifest locks shared instance %q differently than the server project", entry.Instance)
		}
	}
	return nil
}

// reread guards the verify-to-publish window: every browser input is read
// again immediately before atomic selection and must match the verified
// bytes exactly. Verification passed on the carried bytes, so identical
// bytes keep the audit valid without re-running it.
func (p *browserPairing) reread() error {
	if p == nil {
		return fmt.Errorf("browser pairing is missing")
	}
	manifest, err := readBoundedFile(p.manifestPath, 1<<20)
	if err != nil || hashBytes(manifest) != p.manifestHash {
		return fmt.Errorf("browser manifest changed during the server build; retry from a fresh snapshot")
	}
	tree, err := readBoundedFile(filepath.Join(p.generation, "manifest.json"), 8<<20)
	if err != nil || hashBytes(tree) != hashBytes(p.treeManifest) {
		return fmt.Errorf("browser generation changed during the server build; retry from a fresh snapshot")
	}
	for logical, verified := range p.byLogical {
		data, err := readBoundedFile(filepath.Join(p.generation, filepath.FromSlash(logical)), 32<<20)
		if err != nil {
			return fmt.Errorf("paired file %s changed during the server build; retry from a fresh snapshot", logical)
		}
		sum := sha256.Sum256(data)
		want := sha256.Sum256(verified)
		if sum != want {
			return fmt.Errorf("paired file %s changed during the server build; retry from a fresh snapshot", logical)
		}
	}
	return nil
}

func readBoundedFile(name string, limit int64) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", name)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds its size bound", name)
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, fmt.Errorf("%s changed while opening", name)
	}
	data := make([]byte, 0, info.Size()+1)
	buffer := make([]byte, 32768)
	for {
		n, err := file.Read(buffer)
		data = append(data, buffer[:n]...)
		if int64(len(data)) > limit {
			return nil, fmt.Errorf("%s exceeds its size bound", name)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return data, nil
			}
			return nil, err
		}
	}
}

// pairedReport renders the verified pairing for the server build report: the
// exact browser build ID plus every served digest, route, and locked
// instance the publication carries.
func (p *browserPairing) pairedReport() *BrowserReport {
	if p == nil {
		return nil
	}
	report := &BrowserReport{BrowserBuildID: p.buildID, Generation: p.generationID, ManifestSHA256: p.manifestHash, Lock: p.lock, Entry: p.entry, Table: p.table, LockedInstances: append([]BrowserLockedInstance{}, p.instances...)}
	for _, file := range p.files {
		report.Files = append(report.Files, BrowserReportFile{Path: file.Logical, SHA256: file.Digest, Route: file.Route, MediaType: file.MediaType, File: file.File})
	}
	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Route < report.Files[j].Route })
	return report
}

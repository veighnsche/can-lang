package emit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/distribution"
)

type servedAsset struct {
	Route     string `json:"route"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	File      string `json:"file"`
	Integrity string `json:"integrity"`
}

type browserAssetTable struct {
	BuildID string        `json:"buildId"`
	Entry   string        `json:"entry"`
	Table   string        `json:"table"`
	Files   []servedAsset `json:"files"`
}

type assetTable struct {
	HTMX    servedAsset        `json:"htmx"`
	Guard   servedAsset        `json:"guard"`
	Project []servedAsset      `json:"project"`
	Browser *browserAssetTable `json:"browser,omitempty"`
}

// BrowserPairing binds one verified browser build into server emission: the
// exact browser build ID plus the served entry and table routes. The driver
// supplies it only with the verified bytes it describes.
type BrowserPairing struct {
	BuildID string
	Entry   string
	Table   string
}

func assetBundle(program *check.Program, pairing *BrowserPairing) (assetTable, []string, []ir.Artifact, error) {
	script, err := distribution.HTMXAsset()
	if err != nil {
		return assetTable{}, nil, nil, err
	}
	sum := sha256.Sum256(script)
	digest := hex.EncodeToString(sum[:])
	guard, err := distribution.GuardAsset()
	if err != nil {
		return assetTable{}, nil, nil, err
	}
	guardSum := sha256.Sum256(guard)
	guardDigest := hex.EncodeToString(guardSum[:])
	table := assetTable{Project: []servedAsset{}, HTMX: servedAsset{
		Route: "/__can/assets/htmx-4.0.0.min.js", Digest: digest, MediaType: "text/javascript",
		File: "assets/" + digest + "/htmx-4.0.0.min.js", Integrity: "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc",
	}, Guard: servedAsset{
		// The guard integrity pins the Bun-transpiled bytes of
		// runtime/platform/htmx-guard.ts; re-pin together with
		// runtimeHead after any guard edit and regeneration.
		Route: "/__can/assets/htmx-guard.js", Digest: guardDigest, MediaType: "text/javascript",
		File: "assets/" + guardDigest + "/htmx-guard.js", Integrity: "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG",
	}}
	files := map[string][]byte{table.HTMX.File: script, table.Guard.File: guard}
	urls := []string{}
	var browser []servedAsset
	for _, asset := range program.Assets {
		if asset.URL == "" || asset.Artifact == "" || asset.Digest == "" || asset.MediaType == "" {
			return assetTable{}, nil, nil, fmt.Errorf("checked asset is incomplete")
		}
		if previous, ok := files[asset.Artifact]; ok && !bytes.Equal(previous, asset.Bytes) {
			return assetTable{}, nil, nil, fmt.Errorf("asset artifact collision")
		}
		files[asset.Artifact] = append([]byte(nil), asset.Bytes...)
		entry := servedAsset{Route: asset.URL, Digest: asset.Digest, MediaType: asset.MediaType, File: asset.Artifact}
		// Checked project assets always route under /__can/project/; only
		// driver-paired browser bytes route under /__can/assets/. The
		// paired entry, maps, and table stay out of the declared-URL set:
		// pages receive the selected script from the server report, never
		// through asset::url.
		if strings.HasPrefix(asset.URL, "/__can/assets/") {
			actual := sha256.Sum256(asset.Bytes)
			if hex.EncodeToString(actual[:]) != asset.Digest {
				return assetTable{}, nil, nil, fmt.Errorf("paired browser asset failed hash verification")
			}
			browser = append(browser, entry)
			continue
		}
		table.Project = append(table.Project, entry)
		urls = append(urls, asset.URL)
	}
	if len(browser) != 0 || pairing != nil {
		section, err := browserAssetSection(browser, pairing)
		if err != nil {
			return assetTable{}, nil, nil, err
		}
		table.Browser = section
	}
	var paths []string
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var artifacts []ir.Artifact
	for _, path := range paths {
		artifacts = append(artifacts, ir.Artifact{Path: path, Bytes: files[path]})
	}
	return table, urls, artifacts, nil
}

// browserAssetSection binds the partitioned paired bytes to the verified
// pairing identity. Browser-routed bytes without a pairing, or a pairing
// whose entry and table routes are not both served, fail closed.
func browserAssetSection(files []servedAsset, pairing *BrowserPairing) (*browserAssetTable, error) {
	if pairing == nil {
		return nil, fmt.Errorf("paired browser assets lack their verified pairing")
	}
	if len(files) == 0 || pairing.BuildID == "" || pairing.Entry == "" || pairing.Table == "" {
		return nil, fmt.Errorf("browser pairing is incomplete")
	}
	routes := map[string]bool{}
	ordered := append([]servedAsset{}, files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Route < ordered[j].Route })
	for _, file := range ordered {
		if routes[file.Route] {
			return nil, fmt.Errorf("paired browser route %s is published twice", file.Route)
		}
		routes[file.Route] = true
	}
	if !routes[pairing.Entry] || !routes[pairing.Table] {
		return nil, fmt.Errorf("browser pairing entry or table is not served")
	}
	return &browserAssetTable{BuildID: pairing.BuildID, Entry: pairing.Entry, Table: pairing.Table, Files: ordered}, nil
}

func assetInvocation(resolution *ir.AssetResolution) (string, error) {
	if resolution == nil {
		return "", fmt.Errorf("missing asset resolution")
	}
	if resolution.URL != "" && resolution.Reason == "" {
		return "$canHTML.declareAsset(" + quote(resolution.URL) + ",$canContext)", nil
	}
	if resolution.URL == "" && (resolution.Reason == "missing" || resolution.Reason == "unowned") {
		return "$canHTML.rejectAsset(" + quote(resolution.Reason) + ",$canContext)", nil
	}
	return "", fmt.Errorf("invalid asset resolution")
}

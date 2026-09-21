package emit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

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

type assetTable struct {
	HTMX    servedAsset   `json:"htmx"`
	Project []servedAsset `json:"project"`
}

func assetBundle(program *check.Program) (assetTable, []string, []ir.Artifact, error) {
	script, err := distribution.HTMXAsset()
	if err != nil {
		return assetTable{}, nil, nil, err
	}
	sum := sha256.Sum256(script)
	digest := hex.EncodeToString(sum[:])
	table := assetTable{Project: []servedAsset{}, HTMX: servedAsset{
		Route: "/__can/assets/htmx-4.0.0.min.js", Digest: digest, MediaType: "text/javascript",
		File: "assets/" + digest + "/htmx-4.0.0.min.js", Integrity: "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc",
	}}
	files := map[string][]byte{table.HTMX.File: script}
	urls := []string{}
	for _, asset := range program.Assets {
		if asset.URL == "" || asset.Artifact == "" || asset.Digest == "" || asset.MediaType == "" {
			return assetTable{}, nil, nil, fmt.Errorf("checked asset is incomplete")
		}
		if previous, ok := files[asset.Artifact]; ok && !bytes.Equal(previous, asset.Bytes) {
			return assetTable{}, nil, nil, fmt.Errorf("asset artifact collision")
		}
		files[asset.Artifact] = append([]byte(nil), asset.Bytes...)
		table.Project = append(table.Project, servedAsset{Route: asset.URL, Digest: asset.Digest, MediaType: asset.MediaType, File: asset.Artifact})
		urls = append(urls, asset.URL)
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

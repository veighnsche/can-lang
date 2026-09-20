// Package distribution builds the local development sidecar bundle.
package distribution

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
)

//go:embed target.json
var TargetJSON []byte

type Target struct {
	SchemaVersion int    `json:"schemaVersion"`
	TargetID      string `json:"targetId"`
	Runtime       struct {
		Name             string `json:"name"`
		Version          string `json:"version"`
		Revision         string `json:"revision"`
		Platform         string `json:"platform"`
		Architecture     string `json:"architecture"`
		MinimumOSVersion string `json:"minimumOSVersion"`
		Executable       string `json:"executable"`
		SHA256           string `json:"sha256"`
	} `json:"runtime"`
	Upstream struct {
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
		Member string `json:"member"`
	} `json:"upstream"`
}

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	Version       string            `json:"version"`
	TargetID      string            `json:"targetId"`
	Files         map[string]string `json:"files"`
}

func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func PinnedTarget() Target {
	var target Target
	if err := json.Unmarshal(TargetJSON, &target); err != nil {
		panic(err)
	}
	return target
}

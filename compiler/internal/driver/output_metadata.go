package driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func outputAssetHashes(graph *project.Graph) (map[string]map[string]string, error) {
	result := map[string]map[string]string{}
	for key, p := range graph.Projects {
		result[key] = map[string]string{}
		captured := map[string]string{}
		for _, asset := range p.CheckedAssets {
			captured[asset.Name] = asset.Digest
		}
		for name := range p.Manifest.Assets {
			digest, ok := captured[name]
			if !ok {
				return nil, fmt.Errorf("asset %q was not captured with its project inputs", name)
			}
			result[key][name] = digest
		}
	}
	return result, nil
}
func outputSnapshot(graph *project.Graph, assets map[string]map[string]string) string {
	raw, _ := json.Marshal(assets)
	return hashBytes(append([]byte(graphSnapshot(graph)+"\x00"+graph.LockSHA256), raw...))
}

// Native JSON tokenization owns grammar and key decoding; this finite walk only
// rejects ambiguous duplicate metadata keys before decoding the expected schema.
func uniqueOutputJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 output metadata")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 256 {
			return fmt.Errorf("output metadata nesting limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate or invalid output metadata key")
				}
				seen[name] = true
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected output metadata delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing output metadata")
	}
	return nil
}

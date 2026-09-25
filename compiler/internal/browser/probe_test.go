package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestProbeModulesJSONExactness(t *testing.T) {
	raw, err := os.ReadFile("../../../runtime/modules.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Modules map[string][]string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	var drift []string
	for _, name := range sortedKeysProbe(inventory.Modules) {
		body, err := os.ReadFile(filepath.Join("../../../runtime", name))
		if err != nil {
			t.Fatalf("missing module file %s: %v", name, err)
		}
		scan, err := ScanModule(body)
		if err != nil {
			t.Fatalf("lex error in %s: %v", name, err)
		}
		lexed := map[string]bool{}
		for _, edge := range scan.Edges {
			lexed[edge.Specifier] = true
		}
		declared := map[string]bool{}
		for _, spec := range inventory.Modules[name] {
			declared[spec] = true
		}
		for spec := range lexed {
			if !declared[spec] {
				drift = append(drift, name+": lexed-but-undeclared "+spec)
			}
		}
		for spec := range declared {
			if !lexed[spec] {
				drift = append(drift, name+": declared-but-unlexed "+spec)
			}
		}
		// Every inventory key must have a file (covered) and every .ts file an entry.
	}
	var files []string
	filepath.Walk("../../../runtime", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && len(path) > 4 && path[len(path)-3:] == ".ts" {
			rel, _ := filepath.Rel("../../../runtime", path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	for _, file := range files {
		if _, ok := inventory.Modules[file]; !ok {
			drift = append(drift, file+": file-without-inventory-entry")
		}
	}
	sort.Strings(drift)
	for _, line := range drift {
		t.Log(line)
	}
	t.Logf("modules=%d files=%d drift=%d", len(inventory.Modules), len(files), len(drift))
}

func sortedKeysProbe(m map[string][]string) []string {
	var out []string
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

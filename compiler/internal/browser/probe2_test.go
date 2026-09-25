package browser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestProbeRuntimeCensus(t *testing.T) {
	var files []string
	filepath.Walk("../../../runtime", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".ts") &&
			!strings.HasSuffix(path, ".test.ts") && !strings.Contains(path, "/test/") {
			rel, _ := filepath.Rel("../../../runtime", path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	selfUses := 0
	for _, file := range files {
		body, err := os.ReadFile(filepath.Join("../../../runtime", file))
		if err != nil {
			t.Fatal(err)
		}
		scan, err := ScanModule(body)
		if err != nil {
			t.Fatalf("lex error in %s: %v", file, err)
		}
		if len(scan.Findings) > 0 {
			ops := map[string]int{}
			for _, finding := range scan.Findings {
				ops[finding.Operation]++
			}
			t.Logf("%s: %v", file, ops)
		}
		for _, edge := range scan.Edges {
			if !strings.HasPrefix(edge.Specifier, "./") && !strings.HasPrefix(edge.Specifier, "../") {
				t.Logf("%s: native edge %s%s", file, edge.Specifier, map[bool]string{true: " (type-only)", false: ""}[edge.TypeOnly])
			}
		}
		if strings.Contains(string(body), "self.") {
			selfUses++
			t.Logf("%s: contains self.", file)
		}
	}
	t.Logf("shipped files=%d self-uses=%d", len(files), selfUses)
}

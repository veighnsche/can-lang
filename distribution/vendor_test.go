package distribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedOutputParser(t *testing.T) {
	var lock struct {
		SchemaVersion int               `json:"schemaVersion"`
		Name          string            `json:"name"`
		Version       string            `json:"version"`
		Files         map[string]string `json:"files"`
	}
	raw, err := os.ReadFile("../tools/runtime/vendor/acorn.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.SchemaVersion != 1 || lock.Name != "acorn" || lock.Version != "8.18.0" || len(lock.Files) != 2 {
		t.Fatal("unexpected parser lock")
	}
	for _, name := range []string{"tools/runtime/vendor/acorn-8.18.0.mjs", "distribution/notices/acorn-LICENSE.txt"} {
		data, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			t.Fatal(err)
		}
		if Hash(data) != lock.Files[name] {
			t.Fatalf("vendored parser asset changed: %s", name)
		}
	}
}

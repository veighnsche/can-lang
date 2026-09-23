package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stageRawFixtures copies the committed can.native-fixture.v1 files for area
// into the staged project next to the staged sources, so using-raw rows
// resolve exactly as in real projects. Replacements apply to the staged
// copies only (for example endpoint rewrites to the test server), keeping
// exact request comparison meaningful after source substitution.
func stageRawFixtures(t *testing.T, write func(name, text string), sourceRoot, area string, replacements ...[2]string) {
	t.Helper()
	dir := filepath.Join(sourceRoot, "compiler/testdata/current", area, "fixtures")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, replacement := range replacements {
			text = strings.ReplaceAll(text, replacement[0], replacement[1])
		}
		write(filepath.Join("src/fixtures", entry.Name()), text)
	}
}

// stripStagedAuthorization removes the authorization pair from every staged
// raw fixture request. Tests that strip connection auth before a build must
// restage matching fixtures: the authless program sends no authorization
// header, so exact request comparison would otherwise report an argument
// mismatch against the recorded credential header.
func stripStagedAuthorization(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "src/fixtures")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture map[string]any
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		exchange, ok := fixture["exchange"].(map[string]any)
		if !ok {
			continue
		}
		request, ok := exchange["request"].(map[string]any)
		if !ok {
			continue
		}
		staged, ok := request["headers"].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(staged))
		for _, pair := range staged {
			cells, ok := pair.([]any)
			if ok && len(cells) == 2 && cells[0] == "authorization" {
				continue
			}
			kept = append(kept, pair)
		}
		if len(kept) == len(staged) {
			continue
		}
		request["headers"] = kept
		stripped, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, stripped, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// clearStagedFixtureEnvironments empties the environment object of every
// staged raw fixture. Tests that strip connection auth before a build must
// restage matching fixtures: a named credential the inherited connection
// no longer reads is a check error, not silent absence.
func clearStagedFixtureEnvironments(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "src/fixtures")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture map[string]any
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		fixture["environment"] = map[string]any{}
		cleared, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, cleared, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

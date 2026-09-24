package distribution

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// archiveRuntime extracts the pinned executable from the local archive. Tests
// that need real runtime bytes skip without CAN_BUN_ARCHIVE; refusal tests
// that never reach the runtime check run unconditionally.
func archiveRuntime(t *testing.T) []byte {
	t.Helper()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for runtime-backed distribution tests")
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	target := PinnedTarget()
	if int64(len(data)) != target.Upstream.Size || Hash(data) != target.Upstream.SHA256 {
		t.Fatal("pinned archive mismatch")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name != target.Upstream.Member {
			continue
		}
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		out, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if Hash(out) != target.Runtime.SHA256 {
			t.Fatal("pinned runtime mismatch")
		}
		return out
	}
	t.Fatal("runtime member missing")
	return nil
}

// syntheticBundle assembles a minimal verifiable version directory. The
// runtime bytes must be the pinned executable; every other asset carries
// small unique contents hashed into the manifest, and the stub launcher
// embeds the manifest digest exactly like the stamped production binary.
func syntheticBundle(t *testing.T, parent, version string, runtimeBytes []byte) string {
	t.Helper()
	target := PinnedTarget()
	name := "can-" + version + "-" + target.TargetID
	root := filepath.Join(parent, name)
	files := map[string][]byte{
		target.Runtime.Executable:                runtimeBytes,
		"distribution/target.json":               PinnedTargetJSON(),
		"runtime/environment.ts":                 []byte("export const test = 1;\n"),
		"tools/runtime/check.ts":                 []byte("check\n"),
		"tools/runtime/bunfig.toml":              []byte("bunfig\n"),
		"tsconfig.json":                          []byte("{}\n"),
		"distribution/notices/BUN-LICENSE.md":    []byte("bun license\n"),
		"distribution/notices/README.md":         []byte("notices\n"),
		"distribution/assets/htmx-4.0.0.min.js":  HTMXScript,
		"distribution/notices/htmx-LICENSE.txt":  []byte("htmx license\n"),
		"distribution/notices/acorn-LICENSE.txt": []byte("acorn\n"),
		"runtime/extra.ts":                       []byte("extra\n"),
	}
	manifest := Manifest{SchemaVersion: 1, Kind: "can.development-distribution", Version: version, TargetID: target.TargetID, Files: map[string]string{}}
	for file, data := range files {
		manifest.Files[file] = Hash(data)
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	write := func(name string, data []byte, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	for file, data := range files {
		mode := os.FileMode(0644)
		if file == target.Runtime.Executable {
			mode = 0755
		}
		write(file, data, mode)
	}
	write("manifest.json", encoded, 0644)
	write("bin/canlc", []byte("stub-launcher:"+Hash(encoded)+"\n"), 0755)
	return root
}

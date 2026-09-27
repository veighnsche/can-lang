// W1 check/build qualification (C06): both apps assert pinned counts
// over unchanged shared libraries, a breaking control-API edit fails
// the non-edited app's check with a located diagnostic, and
// route/capture/wire/body/result-leaf contract edits rebuild both
// targets or diagnose. Live-browser legs live in the matrix tests
// below; gate5 stays the served regression.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// c06Stage copies one example into a fresh resolved temp dir, skipping
// built generations so staged checks never see another root's dist.
func c06Stage(t *testing.T, sourceRoot, name string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(sourceRoot, "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "dist" {
			continue
		}
		src := filepath.Join(sourceRoot, "examples", name, entry.Name())
		dst := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			copyDir(t, src, dst)
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// c06Relock recomputes one staged vendored dependency's source digest
// after editing vendor/, through the same SourceDigest the toolchain
// verifies. Formatting stays compact like the committed locks.
func c06Relock(t *testing.T, appRoot, dep string) {
	t.Helper()
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	rel, ok := manifest.Dependencies[dep]
	if !ok {
		t.Fatalf("missing dependency %q", dep)
	}
	var depManifest struct {
		SourceRoot string `json:"source_root"`
	}
	raw, err = os.ReadFile(filepath.Join(appRoot, rel, "can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &depManifest); err != nil {
		t.Fatal(err)
	}
	type sourceBytes struct {
		path string
		data []byte
	}
	var files []sourceBytes
	err = filepath.WalkDir(filepath.Join(appRoot, rel, depManifest.SourceRoot), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".can") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(filepath.Join(appRoot, rel, depManifest.SourceRoot), path)
		if err != nil {
			return err
		}
		files = append(files, sourceBytes{path: relPath, data: data})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Byte-exact port of the toolchain source-tree digest: sorted
	// paths, length-prefixed path and content frames.
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	hash := sha256.New()
	hash.Write([]byte("can-source-tree-v1\x00"))
	var length [8]byte
	for _, file := range files {
		binary.BigEndian.PutUint64(length[:], uint64(len(file.path)))
		hash.Write(length[:])
		hash.Write([]byte(file.path))
		binary.BigEndian.PutUint64(length[:], uint64(len(file.data)))
		hash.Write(length[:])
		hash.Write(file.data)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	lockPath := filepath.Join(appRoot, "can.lock.json")
	raw, err = os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Edges map[string]struct {
			Target string `json:"target"`
		} `json:"edges"`
		Projects map[string]struct {
			SourceSHA256 string `json:"source_sha256"`
		} `json:"projects"`
	}
	// Preserve every other lock field byte-exactly: decode, patch the
	// one digest, re-encode compact.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	target := lock.Edges[dep].Target
	generic["projects"].(map[string]any)[target].(map[string]any)["source_sha256"] = digest
	out, err := json.Marshal(generic)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, out, 0600); err != nil {
		t.Fatal(err)
	}
}

// c06EditVendor replaces one substring in a staged vendored file,
// failing unless it occurs exactly once.
func c06EditVendor(t *testing.T, appRoot, vendorPath, old, new string) {
	t.Helper()
	path := filepath.Join(appRoot, vendorPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), old) != 1 {
		t.Fatalf("%s: want exactly one %q", vendorPath, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0600); err != nil {
		t.Fatal(err)
	}
}

// c06Assert runs one offline assert and parses its verdict report.
func c06Assert(t *testing.T, ctx context.Context, canlc, home, root string) (int, string, string) {
	t.Helper()
	return gate5Canlc(t, ctx, canlc, home, "assert", root)
}

func c06AssertCounts(t *testing.T, out string) (total, real int) {
	t.Helper()
	var report struct {
		Passed     bool `json:"passed"`
		Assertions []struct {
			Passed   bool     `json:"passed"`
			Evidence []string `json:"evidence"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed {
		t.Fatalf("invalid assert report %v %.300s", err, out)
	}
	for _, assertion := range report.Assertions {
		if !assertion.Passed {
			t.Fatalf("assert leg failed: %.300s", out)
		}
		for _, evidence := range assertion.Evidence {
			if evidence == "real-can" {
				real++
				break
			}
		}
	}
	return len(report.Assertions), real
}

func TestC06Positive(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged C06 execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	_, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	home := t.TempDir()

	// Both apps assert pinned counts over the unchanged shared
	// libraries: every leg runs real Can.
	grid := c06Stage(t, sourceRoot, "invoice-grid")
	status, out, diag := c06Assert(t, ctx, canlc, home, grid)
	if status != 0 || diag != "" {
		t.Fatalf("grid assert: %d %.300s %s", status, out, diag)
	}
	total, real := c06AssertCounts(t, out)
	if total != 394 || real != 394 {
		t.Fatalf("grid asserts %d (%d real), want 394 (394 real)", total, real)
	}
	compare := c06Stage(t, sourceRoot, "invoice-compare")
	status, out, diag = c06Assert(t, ctx, canlc, home, compare)
	if status != 0 || diag != "" {
		t.Fatalf("compare assert: %d %.300s %s", status, out, diag)
	}
	total, real = c06AssertCounts(t, out)
	if total != 221 || real != 221 {
		t.Fatalf("compare asserts %d (%d real), want 221 (221 real)", total, real)
	}

	// The second app pairs with the invoice server like the grid:
	// browser build, paired server build, report verification.
	browser := gate5Build(t, ctx, canlc, home, compare, "--target", "browser")
	again := gate5Build(t, ctx, canlc, home, compare, "--target", "browser")
	if browser.BuildID != again.BuildID {
		t.Fatalf("compare rebuild drifted: %s vs %s", browser.BuildID, again.BuildID)
	}
	assertNoStrayEmit(t, compare, again.Directory)
	gate5Asset(t, again.Directory)
	gate5ImportAudit(t, again.Directory)
	server := c06Stage(t, sourceRoot, "invoice")
	manifest := filepath.Join(again.Directory, "browser", "manifest.json")
	pairing := gate5PairBuild(t, ctx, canlc, home, server, manifest)
	repeat := gate5PairBuild(t, ctx, canlc, home, server, manifest)
	if pairing.BuildID != repeat.BuildID {
		t.Fatalf("compare pairing drifted: %s vs %s", pairing.BuildID, repeat.BuildID)
	}
	assertNoStrayEmit(t, server, pairing.Directory)
	t.Logf("C06 positive: grid 394, compare 221, compare pairing %s", pairing.BuildID[:12])
}

func TestC06NegativeAPIBreak(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged C06 execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	_, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	home := t.TempDir()

	// A breaking control-API edit (parse_qty leaves the shared
	// fields surface) fails the non-edited app's check with a
	// located diagnostic: no silent drift. Both consumers stage the
	// same broken library; each check names the member and the
	// consumer file that needs it.
	for _, name := range []string{"invoice-compare", "invoice-grid"} {
		staged := c06Stage(t, sourceRoot, name)
		c06EditVendor(t, staged, "vendor/controls/src/fields/fields.can", "parse_qty, ", "")
		c06Relock(t, staged, "controls")
		status, out, diag := c06Assert(t, ctx, canlc, home, staged)
		if status == 0 {
			t.Fatalf("%s check passed over the broken library", name)
		}
		combined := out + "\n" + diag
		for _, want := range []string{"parse_qty", ".can", "private"} {
			if !strings.Contains(combined, want) {
				t.Fatalf("%s diagnostic lacks %q: %.500s", name, want, combined)
			}
		}
		t.Logf("C06 negative %s: %.200s", name, strings.TrimSpace(combined))
	}
}

func TestC06CaptureEdits(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged C06 execution")
	}
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	_, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	home := t.TempDir()

	const contract = "vendor/billing/src/invoice_contract/invoice_contract.can"
	edits := []struct {
		name    string
		old     string
		new     string
		rebuild bool
		want    []string
	}{
		{
			name:    "route",
			old:     `get "/api/tenants/:tenant_id/invoices/:invoice_id"`,
			new:     `get "/api/v2/tenants/:tenant_id/invoices/:invoice_id"`,
			rebuild: true,
		},
		{
			name: "capture",
			old:  `get "/api/tenants/:tenant_id/invoices/:invoice_id"`,
			new:  `get "/api/tenants/:tenant/invoices/:invoice_id"`,
			want: []string{"load_invoice_grid", ":tenant", "captures"},
		},
		{
			name: "wire",
			old:  "record grid_line_wire\n    str key\n",
			new:  "record grid_line_wire\n    str key\n    str memo\n",
			want: []string{".can", "arity"},
		},
		{
			name: "body",
			old:  "    input none\n    returns grid_load_outcome\n    body json\n",
			new:  "    input none\n    returns grid_load_outcome\n    body html\n",
			want: []string{"load_invoice_grid", "document"},
		},
		{
			name: "leaf",
			old:  "        grid_loaded status 200\n",
			new:  "        grid_ready status 200\n",
			want: []string{"grid_ready", ".can"},
		},
	}
	for _, edit := range edits {
		t.Run(edit.name, func(t *testing.T) {
			acquireHeavy(t)
			server := c06Stage(t, sourceRoot, "invoice")
			grid := c06Stage(t, sourceRoot, "invoice-grid")
			for _, staged := range []string{server, grid} {
				c06EditVendor(t, staged, contract, edit.old, edit.new)
				c06Relock(t, staged, "billing")
			}
			if edit.rebuild {
				// A compatible route edit moves both targets
				// together: the server and the browser builds
				// both exit 0 with verified assertions.
				gate5Build(t, ctx, canlc, home, server)
				gate5Build(t, ctx, canlc, home, grid, "--target", "browser")
				t.Logf("C06 capture %s: both targets rebuilt", edit.name)
				return
			}
			// A breaking edit diagnoses in each consuming
			// target with a located message naming the edit.
			for _, staged := range []struct {
				name, root string
			}{{"server", server}, {"grid", grid}} {
				status, out, diag := c06Assert(t, ctx, canlc, home, staged.root)
				if status == 0 {
					t.Fatalf("%s %s check passed over the breaking edit", edit.name, staged.name)
				}
				combined := out + "\n" + diag
				for _, want := range edit.want {
					if !strings.Contains(combined, want) {
						t.Fatalf("%s %s diagnostic lacks %q: %.500s", edit.name, staged.name, want, combined)
					}
				}
				t.Logf("C06 capture %s %s: %.200s", edit.name, staged.name, strings.TrimSpace(combined))
			}
		})
	}
}

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
	"os/exec"
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

const (
	c06PortCompareChromium = 18781
	c06PortCompareWebkit   = 18782
	c06PortGridChromium    = 18784
	c06PortGridWebkit      = 18787
	c06PortDriftChromium   = 18785
	c06PortDriftWebkit     = 18788
	// Container Firefox reaches staged leg servers through the
	// provisioned loopback forwarders, like the gate5 legs.
	c06PortFirefox = 18651
)

type c06Matrix struct {
	ctx        context.Context
	sourceRoot string
	browserDir string
	toolchain  string
	canlc      string
	nodePath   string
	driver     string
	serverRoot string
	serverV2   string
	compare    gate5Pairing
	grid       gate5Pairing
	driftA     gate5Pairing
	driftB     gate5Pairing
}

func (m *c06Matrix) serve(t *testing.T, pairing gate5Pairing, serverRoot string, port int) (base string, db string, stop func()) {
	t.Helper()
	home := t.TempDir()
	db = gate5SeedDB(t, m.ctx, m.toolchain, home, m.driver, serverRoot)
	base, stop = serveInvoice(t, m.ctx, m.toolchain, home, filepath.Join(pairing.Directory, "entry.ts"), db, port, "")
	return base, db, stop
}

func (m *c06Matrix) compareLeg(t *testing.T, engine string, port int) {
	t.Helper()
	base, _, stop := m.serve(t, m.compare, m.serverRoot, port)
	gate5ServedPairing(t, "compare", base, "/invoice-compare", "", m.compare)
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "compare", engine,
		"compare.mjs", engine, base, outdir, m.compare.Entry)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "compare", engine, raw, 13, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("compare %s limitations %+v, want none", engine, report.Limitations)
	}
	for _, entry := range report.Requests {
		if strings.Contains(entry.URL, "/api/") {
			t.Fatalf("compare %s called an API: %s", engine, entry.URL)
		}
	}
	gate5Screenshot(t, "compare", engine, outdir)
	gate5Evidence(t, outdir, "compare-"+engine)
	t.Logf("C06 compare %s %s: 13 checks, %d loopback requests", engine, report.Version, len(report.Requests))
}

func (m *c06Matrix) gridLeg(t *testing.T, engine string, port int) {
	t.Helper()
	base, _, stop := m.serve(t, m.grid, m.serverRoot, port)
	gate5ServedPairing(t, "w1-grid", base, "/invoice-grid?tenant=1&invoice=7", "", m.grid)
	outdir := t.TempDir()
	gate5RunHarness(t, m.ctx, m.nodePath, m.browserDir, "w1-grid", engine,
		"w1-grid.mjs", engine, base, outdir, m.grid.Entry, m.grid.BuildID)
	stop()
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "w1-grid", engine, raw, 12, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("w1-grid %s limitations %+v, want none", engine, report.Limitations)
	}
	gate5Screenshot(t, "w1-grid", engine, outdir)
	gate5Evidence(t, outdir, "w1-grid-"+engine)
	t.Logf("C06 w1-grid %s %s: 12 checks, %d loopback requests, %d invoice calls",
		engine, report.Version, len(report.Requests), len(report.Ledger))
}

// driftLeg proves a real old-page/new-server refusal: serve from,
// boot the page, wait for the harness READY rendezvous, restart the
// same port on to, release the harness with GO, and read its report.
func (m *c06Matrix) driftLeg(t *testing.T, engine string, port int, from, to gate5Pairing, fromRoot, toRoot, name string) {
	t.Helper()
	home := t.TempDir()
	db := gate5SeedDB(t, m.ctx, m.toolchain, home, m.driver, fromRoot)
	base, stop := serveInvoice(t, m.ctx, m.toolchain, home, filepath.Join(from.Directory, "entry.ts"), db, port, "")
	outdir := t.TempDir()
	env := []string{"PATH=" + filepath.Dir(m.nodePath) + ":/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	if value, ok := os.LookupEnv("CAN_FIREFOX_WS"); ok {
		env = append(env, "CAN_FIREFOX_WS="+value)
	}
	cmd := exec.CommandContext(m.ctx, m.nodePath, "drift.mjs", engine, base, outdir, from.Entry, from.BuildID, to.BuildID)
	cmd.Dir = m.browserDir
	cmd.Env = env
	result := make(chan []byte, 1)
	runErr := make(chan error, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		result <- out
		runErr <- err
	}()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if _, err := os.Stat(filepath.Join(outdir, "READY")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("drift %s %s never readied", name, engine)
		}
		time.Sleep(250 * time.Millisecond)
	}
	stop()
	_, restart := serveInvoice(t, m.ctx, m.toolchain, home, filepath.Join(to.Directory, "entry.ts"), db, port, "")
	if err := os.WriteFile(filepath.Join(outdir, "GO"), []byte("go\n"), 0600); err != nil {
		restart()
		t.Fatal(err)
	}
	out, err := <-result, <-runErr
	restart()
	if err != nil {
		t.Fatalf("drift %s %s harness: %v %s", name, engine, err, out)
	}
	raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := gate5ReadReport(t, "drift-"+name, engine, raw, 4, true)
	if len(report.Limitations) != 0 {
		t.Fatalf("drift %s %s limitations %+v, want none", name, engine, report.Limitations)
	}
	gate5Screenshot(t, "drift-"+name, engine, outdir)
	gate5Evidence(t, outdir, "drift-"+name+"-"+engine)
	t.Logf("C06 drift %s %s %s: 4 checks, %s -> %s", name, engine, report.Version, from.BuildID[:12], to.BuildID[:12])
}

func TestC06ServedMatrix(t *testing.T) {
	t.Parallel()
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged C06 execution")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	sourceRoot := mustSourceRoot(t)
	browserDir := filepath.Join(sourceRoot, "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	matrix := &c06Matrix{
		ctx:        ctx,
		sourceRoot: sourceRoot,
		browserDir: browserDir,
	}
	toolchain, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	matrix.toolchain = toolchain
	matrix.canlc = canlc
	matrix.nodePath = nodePath
	matrix.driver = filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")

	// All three named engines are required: probe before building.
	for _, engine := range []string{"chromium", "webkit", "firefox"} {
		gate5BrowserProbe(t, ctx, nodePath, browserDir, engine)
	}
	home := t.TempDir()
	serverRoot := c06Stage(t, sourceRoot, "invoice")
	matrix.serverRoot = serverRoot
	gridRoot := c06Stage(t, sourceRoot, "invoice-grid")
	compareRoot := c06Stage(t, sourceRoot, "invoice-compare")
	status, out, diag := c06Assert(t, ctx, canlc, home, serverRoot)
	if status != 0 || diag != "" {
		t.Fatalf("server assert: %d %.300s %s", status, out, diag)
	}
	total, real := c06AssertCounts(t, out)
	if total != 347 || real != 347 {
		t.Fatalf("server asserts %d (%d real), want 347 (347 real)", total, real)
	}
	compare := gate5Build(t, ctx, canlc, home, compareRoot, "--target", "browser")
	compareAgain := gate5Build(t, ctx, canlc, home, compareRoot, "--target", "browser")
	if compare.BuildID != compareAgain.BuildID {
		t.Fatalf("compare browser rebuild drifted: %s vs %s", compare.BuildID, compareAgain.BuildID)
	}
	assertNoStrayEmit(t, compareRoot, compareAgain.Directory)
	gate5Asset(t, compareAgain.Directory)
	gate5ImportAudit(t, compareAgain.Directory)
	grid := gate5Build(t, ctx, canlc, home, gridRoot, "--target", "browser")
	gridAgain := gate5Build(t, ctx, canlc, home, gridRoot, "--target", "browser")
	if grid.BuildID != gridAgain.BuildID {
		t.Fatalf("grid browser rebuild drifted: %s vs %s", grid.BuildID, gridAgain.BuildID)
	}
	assertNoStrayEmit(t, gridRoot, gridAgain.Directory)
	gate5Asset(t, gridAgain.Directory)
	gate5ImportAudit(t, gridAgain.Directory)
	matrix.compare = gate5PairBuild(t, ctx, canlc, home, serverRoot, filepath.Join(compareAgain.Directory, "browser", "manifest.json"))
	matrix.grid = gate5PairBuild(t, ctx, canlc, home, serverRoot, filepath.Join(gridAgain.Directory, "browser", "manifest.json"))
	assertNoStrayEmit(t, serverRoot, matrix.grid.Directory)

	// The drift pair needs two server generations over one browser
	// build: a comment-only second stage keeps behavior identical
	// while the content address moves.
	secondRoot := c06Stage(t, sourceRoot, "invoice")
	webPath := filepath.Join(secondRoot, "src/web/web.can")
	web, err := os.ReadFile(webPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(webPath, append([]byte("/// C06 drift generation two.\n"), web...), 0600); err != nil {
		t.Fatal(err)
	}
	matrix.serverV2 = secondRoot
	manifest := filepath.Join(gridAgain.Directory, "browser", "manifest.json")
	matrix.driftA = gate5PairBuild(t, ctx, canlc, home, serverRoot, manifest)
	matrix.driftB = gate5PairBuild(t, ctx, canlc, home, secondRoot, manifest)
	if matrix.driftA.BuildID == matrix.driftB.BuildID {
		t.Fatalf("drift generations collide: %s", matrix.driftA.BuildID)
	}
	t.Logf("C06 builds: server %d assertions; compare %s paired %s; grid %s paired %s; drift %s vs %s",
		total, compare.BuildID[:12], matrix.compare.BuildID[:12], grid.BuildID[:12], matrix.grid.BuildID[:12],
		matrix.driftA.BuildID[:12], matrix.driftB.BuildID[:12])

	t.Run("compare", func(t *testing.T) {
		matrix.compareLeg(t, "chromium", c06PortCompareChromium)
		matrix.compareLeg(t, "webkit", c06PortCompareWebkit)
		matrix.compareLeg(t, "firefox", c06PortFirefox)
	})
	t.Run("grid", func(t *testing.T) {
		matrix.gridLeg(t, "chromium", c06PortGridChromium)
		matrix.gridLeg(t, "webkit", c06PortGridWebkit)
		matrix.gridLeg(t, "firefox", c06PortFirefox)
	})
	t.Run("drift-rollout", func(t *testing.T) {
		matrix.driftLeg(t, "chromium", c06PortDriftChromium, matrix.driftA, matrix.driftB, matrix.serverRoot, matrix.serverV2, "rollout")
		matrix.driftLeg(t, "webkit", c06PortDriftWebkit, matrix.driftA, matrix.driftB, matrix.serverRoot, matrix.serverV2, "rollout")
		matrix.driftLeg(t, "firefox", c06PortFirefox, matrix.driftA, matrix.driftB, matrix.serverRoot, matrix.serverV2, "rollout")
	})
	t.Run("drift-rollback", func(t *testing.T) {
		matrix.driftLeg(t, "chromium", c06PortDriftChromium, matrix.driftB, matrix.driftA, matrix.serverV2, matrix.serverRoot, "rollback")
		matrix.driftLeg(t, "webkit", c06PortDriftWebkit, matrix.driftB, matrix.driftA, matrix.serverV2, matrix.serverRoot, "rollback")
		matrix.driftLeg(t, "firefox", c06PortFirefox, matrix.driftB, matrix.driftA, matrix.serverV2, matrix.serverRoot, "rollback")
	})
}

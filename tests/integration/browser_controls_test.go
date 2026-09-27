// C02 native browser controls matrix. Two suites qualify the control
// surface in the three required named browsers, Chromium, WebKit and
// Firefox: controls-native pins the DOM facts (checkbox IDL, dirty
// live value vs attribute, form reset, multiselect/single selection,
// file metadata, modifiers, composition, caret) on a static page, and
// controls drives the same behaviors through the emitted Can fixture
// (tests/integration/testdata/browser-controls) paired with the real
// invoice server. Firefox runs through the pinned container runner
// when CAN_FIREFOX_WS is set, so its legs keep the exact loopback
// base via the provisioned forwarders; the firefox port below is one
// of the four forwarded ports.
package integration

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	c02PortNativeChromium   = 18721
	c02PortNativeWebkit     = 18722
	c02PortNativeFirefox    = 18651
	c02PortControlsChromium = 18723
	c02PortControlsWebkit   = 18724
	c02PortControlsFirefox  = 18651
)

func c02ServeStatic(t *testing.T, dir string, port int) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	// Raw bytes, not FileServer/ServeFile: both canonicalize
	// /index.html to / with a 301, which would pollute the ledger.
	page, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})
	return "http://127.0.0.1:" + strconv.Itoa(port)
}

func c02StageControls(t *testing.T, sourceRoot string) (root, home string) {
	t.Helper()
	var err error
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	from := filepath.Join(sourceRoot, "tests/integration/testdata/browser-controls")
	for _, name := range []string{"can.project.json", "can.errors.json"} {
		data, err := os.ReadFile(filepath.Join(from, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyDir(t, filepath.Join(from, "src"), filepath.Join(root, "src"))
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root, home
}

func c02BrowserDeps(t *testing.T) (nodePath, browserDir string) {
	t.Helper()
	var err error
	nodePath, err = exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the browser harness")
	}
	browserDir = filepath.Join(mustSourceRoot(t), "tests/integration/browser")
	if _, err := os.Stat(filepath.Join(browserDir, "node_modules/playwright/package.json")); err != nil {
		t.Skip("run bun ci in tests/integration/browser for the pinned harness")
	}
	return nodePath, browserDir
}

func TestC02NativeMatrix(t *testing.T) {
	acquireHeavy(t)
	nodePath, browserDir := c02BrowserDeps(t)
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	for _, engine := range []string{"chromium", "webkit", "firefox"} {
		t.Logf("c02 native required browser %s at %s", engine, gate5BrowserProbe(t, ctx, nodePath, browserDir, engine))
	}
	static := filepath.Join(sourceRoot, "tests/integration/testdata/browser-controls-native")
	for _, leg := range []struct {
		engine string
		port   int
	}{
		{"chromium", c02PortNativeChromium},
		{"webkit", c02PortNativeWebkit},
		{"firefox", c02PortNativeFirefox},
	} {
		base := c02ServeStatic(t, static, leg.port)
		outdir := t.TempDir()
		gate5RunHarness(t, ctx, nodePath, browserDir, "controls-native", leg.engine,
			"controls-native.mjs", leg.engine, base, outdir)
		raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
		if err != nil {
			t.Fatal(err)
		}
		report := gate5ReadReport(t, "controls-native", leg.engine, raw, 11, true)
		if len(report.Limitations) != 0 {
			t.Fatalf("controls-native %s limitations %+v, want none", leg.engine, report.Limitations)
		}
		for _, entry := range report.Requests {
			if !strings.HasSuffix(entry.URL, "/index.html") {
				t.Fatalf("controls-native %s unexpected request %s", leg.engine, entry.URL)
			}
		}
		gate5Screenshot(t, "controls-native", leg.engine, outdir)
		gate5Evidence(t, outdir, "controls-native-"+leg.engine)
		t.Logf("c02 native %s %s: 11 checks, %d loopback requests", leg.engine, report.Version, len(report.Requests))
	}
}

func TestC02ControlsMatrix(t *testing.T) {
	acquireHeavy(t)
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged C02 execution")
	}
	nodePath, browserDir := c02BrowserDeps(t)
	sourceRoot := mustSourceRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	for _, engine := range []string{"chromium", "webkit", "firefox"} {
		t.Logf("c02 controls required browser %s at %s", engine, gate5BrowserProbe(t, ctx, nodePath, browserDir, engine))
	}
	toolchain, canlc, _, _ := gate5Toolchain(t, ctx, sourceRoot, archive)
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/invoice/driver.ts")
	serverRoot, serverHome := stageApplication(t, ctx, toolchain, sourceRoot, "invoice")
	gate5Assert(t, ctx, canlc, serverHome, serverRoot, "invoice")
	controlsRoot, controlsHome := c02StageControls(t, sourceRoot)
	first := gate5Build(t, ctx, canlc, controlsHome, controlsRoot, "--target", "browser")
	second := gate5Build(t, ctx, canlc, controlsHome, controlsRoot, "--target", "browser")
	if first.BuildID != second.BuildID {
		t.Fatalf("controls browser rebuild drifted: %s vs %s", first.BuildID, second.BuildID)
	}
	assertNoStrayEmit(t, controlsRoot, second.Directory)
	gate5Asset(t, second.Directory)
	gate5ImportAudit(t, second.Directory)
	pairing := gate5PairBuild(t, ctx, canlc, serverHome, serverRoot, filepath.Join(second.Directory, "browser", "manifest.json"))
	repeat := gate5PairBuild(t, ctx, canlc, serverHome, serverRoot, filepath.Join(second.Directory, "browser", "manifest.json"))
	if repeat.BuildID != pairing.BuildID || repeat.Entry != pairing.Entry {
		t.Fatalf("controls paired rebuild drifted: %s vs %s", repeat.BuildID, pairing.BuildID)
	}
	t.Logf("c02 controls builds: browser %s paired %s entry %s", first.BuildID[:12], pairing.BuildID[:12], pairing.Entry)
	for _, leg := range []struct {
		engine string
		port   int
	}{
		{"chromium", c02PortControlsChromium},
		{"webkit", c02PortControlsWebkit},
		{"firefox", c02PortControlsFirefox},
	} {
		home := t.TempDir()
		db := gate5SeedDB(t, ctx, toolchain, home, driver, serverRoot)
		base, stop := serveInvoice(t, ctx, toolchain, home, filepath.Join(pairing.Directory, "entry.ts"), db, leg.port, "")
		gate5ServedPairing(t, "controls", base, "/invoice-grid?controls=1", "", pairing)
		outdir := t.TempDir()
		gate5RunHarness(t, ctx, nodePath, browserDir, "controls", leg.engine,
			"controls.mjs", leg.engine, base, outdir, pairing.Entry)
		stop()
		raw, err := os.ReadFile(filepath.Join(outdir, "report.json"))
		if err != nil {
			t.Fatal(err)
		}
		report := gate5ReadReport(t, "controls", leg.engine, raw, 17, true)
		if len(report.Limitations) != 0 {
			t.Fatalf("controls %s limitations %+v, want none", leg.engine, report.Limitations)
		}
		for _, entry := range report.Requests {
			if strings.Contains(entry.URL, "/api/") {
				t.Fatalf("controls %s fixture called an API: %s", leg.engine, entry.URL)
			}
		}
		gate5Screenshot(t, "controls", leg.engine, outdir)
		store := inspectInvoice(t, ctx, toolchain, t.TempDir(), driver, db)
		row := requireInvoice(t, store, "7", "1", gate5SeedSeats, gate5SeedDetails)
		gate5RequireSeedLines(t, "controls", leg.engine, row)
		if len(store.Replay) != 0 {
			t.Fatalf("controls %s replay rows %+v, want none", leg.engine, store.Replay)
		}
		gate5Evidence(t, outdir, "controls-"+leg.engine)
		t.Logf("c02 controls %s %s: 17 checks, %d loopback requests, database untouched",
			leg.engine, report.Version, len(report.Requests))
	}
}

// W2 second-vendor integration legs (D03): vendor-specific capabilities
// stay unadmitted with located rejection, the deliverables introduce no
// catalogue identities, Vendor B reuses the delivered D02 client and
// protocol, and the C-owned browser closure suite stays green read-only.
// D03 requests no closure patch (C closure patches go through C; none is
// needed): the witness below runs C's suite unmodified.
package conformance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// vendorCapabilities are the vendor-specific package spellings W2 would
// have introduced had it taken a vendor-patch path instead of the assigned
// T3 path. Each must reject exactly like the D02 capabilities: the real
// compiler fails with file-attributed location evidence naming the
// capability. The D02 names ride along to prove W2 admitted nothing.
var vendorCapabilities = []string{
	"chart_vendor_b",
	"vendor_b",
	"storage",
	"clipboard",
	"chart",
}

// TestVendorCapabilitiesRejectWithLocation proves no vendor-specific
// admission exists: every vendor-flavored capability name fails the real
// compiler black-box with a located unknown-package diagnostic.
func TestVendorCapabilitiesRejectWithLocation(t *testing.T) {
	binary := canlcBinary(t)
	for _, capability := range vendorCapabilities {
		t.Run(capability, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
				"can.errors.json":  `{"active":[],"retired":[]}`,
				"src/main.can": "package app\n" +
					"    provides []\n" +
					"    uses [" + capability + "]\n" +
					"fn void main\n" +
					"    emits []\n" +
					"    given\n" +
					"        str[] arguments\n" +
					"    asserts\n" +
					"        empty: [] => ok\n" +
					"    ok\n",
			}
			for name, text := range files {
				p := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(binary, "inspect-project", root)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("capability %q admitted; inspect-project passed:\n%s", capability, combined)
			}
			output := string(combined)
			if !strings.Contains(output, "src/main.can") {
				t.Fatalf("capability %q rejection lacks file location:\n%s", capability, output)
			}
			if !strings.Contains(output, `unknown package "`+capability+`"`) {
				t.Fatalf("capability %q rejection names no capability:\n%s", capability, output)
			}
		})
	}
}

// TestDeliverablesIntroduceNoCatalogueIdentities pins the W2.1 no-patch
// bar from the deliverable side: no host module may name a catalogue
// operation identity or reach for the catalogue merge source, so the
// second vendor adds nothing the browser closure would need to rule on.
func TestDeliverablesIntroduceNoCatalogueIdentities(t *testing.T) {
	dir := hostDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "REVIEW-MANIFEST.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest reviewManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, entry := range manifest.Deliverables {
		body, err := os.ReadFile(filepath.Join(dir, entry.Path))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Path, err)
		}
		for _, forbidden := range []string{"can.std.", "catalogue.json"} {
			if strings.Contains(string(body), forbidden) {
				t.Fatalf("deliverable %q names %q; W2 adds no catalogue surface", entry.Path, forbidden)
			}
		}
	}
}

// TestVendorBReusesDeliveredClient pins the reusable-path structure of
// the second vendor statically: Vendor B imports the shared `d02.chart/1`
// contract from the D02 module, defines no client factory, and forks no
// protocol constant. The bun legs prove the behavior; this leg pins the
// shape so a future edit cannot silently fork either.
func TestVendorBReusesDeliveredClient(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(hostDir(t), "companions", "chart-vendor-b.ts"))
	if err != nil {
		t.Fatalf("read vendor-b module: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `from "./chart.ts"`) {
		t.Fatal("vendor-b module does not import the shared d02.chart/1 contract")
	}
	if !strings.Contains(text, "CHART_PROTOCOL,") {
		t.Fatal("vendor-b module does not use the shared protocol constant")
	}
	for _, fork := range []string{"function createChartCompanionClient", "const CHART_PROTOCOL"} {
		if strings.Contains(text, fork) {
			t.Fatalf("vendor-b module contains %q; the client and protocol stay shared", fork)
		}
	}
}

// TestBrowserClosureSuiteGreen runs the C-owned browser capability closure
// suite read-only and requires it green. D03 patches no closure file (all
// D03 writes stay under host/); this witness gates W2 on the real gate:
// server-only operations keep rejecting with span evidence while the W2
// integration — which adds no checked surface — stays out of its way.
func TestBrowserClosureSuiteGreen(t *testing.T) {
	runner, err := exec.LookPath("go")
	if err != nil {
		t.Fatal("go not on PATH; closure witness needs the Go toolchain")
	}
	cmd := exec.Command(runner, "test", "./compiler/internal/browser/", "-count=1")
	cmd.Dir = repoRoot(t)
	combined, err := cmd.CombinedOutput()
	t.Logf("closure suite output:\n%s", combined)
	if err != nil {
		t.Fatalf("C-owned browser closure suite red with W2 present: %v", err)
	}
}

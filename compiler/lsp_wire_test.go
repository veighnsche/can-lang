package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
)

// I41: diagnostic transport. Published frames must carry the bridge
// code and the editor coordinates; codeless lines stay byte-identical
// apart from the omitted key. Probes first.

func publishBridgeFrame(t *testing.T, version *int64, diags []driver.Diagnostic) (map[string]any, []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := publishBridgeDiagnostics(w, "file:///m.can", version, diags); err != nil {
		t.Fatalf("publish: %v", err)
	}
	raw := buf.String()
	head, body, ok := strings.Cut(raw, "\r\n\r\n")
	if !ok {
		t.Fatalf("no frame header in %q", raw)
	}
	if !strings.HasPrefix(head, "Content-Length: ") {
		t.Fatalf("bad frame header %q", head)
	}
	var frame struct {
		Method string `json:"method"`
		Params struct {
			URI         string           `json:"uri"`
			Version     *int64           `json:"version"`
			Diagnostics []map[string]any `json:"diagnostics"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(body), &frame); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if frame.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q", frame.Method)
	}
	params := map[string]any{"uri": frame.Params.URI, "version": frame.Params.Version}
	return params, frame.Params.Diagnostics
}

// TestPublishCarriesCode pins the stable code on the wire.
func TestPublishCarriesCode(t *testing.T) {
	version := int64(7)
	_, got := publishBridgeFrame(t, &version, []driver.Diagnostic{{
		File: "m.can", Line: 3, Start: 1, End: 5,
		Code: "CAN4107", Message: "bad arm", Severity: "error",
	}})
	if len(got) != 1 || got[0]["code"] != "CAN4107" {
		t.Fatalf("wire diagnostic carries no code: %v", got)
	}
	if got[0]["message"] != "bad arm" {
		t.Fatalf("wire diagnostic carries no message: %v", got[0])
	}
	rng, _ := got[0]["range"].(map[string]any)
	start, _ := rng["start"].(map[string]any)
	if start["line"] != 3.0 || start["character"] != 1.0 {
		t.Fatalf("wire diagnostic range not anchored: %v", rng)
	}
}

// TestPublishOmitsEmpty pins backward compat: codeless lines carry
// no code key, and a clean file publishes an empty array.
func TestPublishOmitsEmpty(t *testing.T) {
	_, got := publishBridgeFrame(t, nil, []driver.Diagnostic{{
		File: "m.can", Line: 0, Message: "unused", Severity: "error",
	}})
	if len(got) != 1 {
		t.Fatalf("expected one diagnostic, got %v", got)
	}
	if _, present := got[0]["code"]; present {
		t.Fatalf("codeless line carries code: %v", got[0])
	}
	params, cleared := publishBridgeFrame(t, nil, nil)
	if cleared == nil {
		t.Fatalf("clean file published null diagnostics, want []")
	}
	if len(cleared) != 0 {
		t.Fatalf("clean file published %v", cleared)
	}
	if v, _ := params["version"].(*int64); v != nil {
		t.Fatalf("versionless publish carries version: %v", params)
	}
}

// TestPublishCarriesVersion pins the version echo: editors drop
// stale frames by matching it against the open document version.
func TestPublishCarriesVersion(t *testing.T) {
	version := int64(12)
	params, _ := publishBridgeFrame(t, &version, nil)
	if params["version"] == nil || *params["version"].(*int64) != 12 {
		t.Fatalf("version not echoed: %v", params)
	}
}

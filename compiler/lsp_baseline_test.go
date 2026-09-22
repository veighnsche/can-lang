package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a79: LSP revision enforcement. `canlc lsp --baseline` surfaces the
// same CAN6013 drift findings as the CLI on every keystroke. Probes
// first: diagnoseWith threads an accepted baseline through diagnose.

func lspBaselineDir(t *testing.T, model, client string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"model.can": model, "client.can": client} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func lspAcceptedBaseline(t *testing.T) *RevisionBaseline {
	t.Helper()
	files := revisionFiles(revisionModelB, revisionClientB)
	prog, _ := revisionProg(t, files, []string{"model.can", "client.can"})
	return revisionBaseline(t, prog, "review-base:lsp")
}

// TestLSPBaselineClean pins the quiet case: an accepted baseline
// matching the open world reports no identity findings.
func TestLSPBaselineClean(t *testing.T) {
	dir := lspBaselineDir(t, revisionModelB, revisionClientB)
	base := lspAcceptedBaseline(t)
	if diags := diagnoseWith(dir, "client.can", revisionClientB, base); hasCode(diags, CodeRevisionIdentity) {
		t.Fatalf("clean world under accepted baseline reported identity drift: %+v", diags)
	}
}

// TestLSPBaselineDrift pins enforcement: a sibling interface change
// (new variant case, same revision) surfaces CAN6013 in the editor
// even though the program still checks clean.
func TestLSPBaselineDrift(t *testing.T) {
	drifted := `mod model
  provides [Model__State]
  uses []
  emits []

variant Model__State rev 1 (
  case Ready()
  case Waiting()
  case Loading()
)
`
	dir := lspBaselineDir(t, drifted, revisionClientB)
	base := lspAcceptedBaseline(t)
	diags := diagnoseWith(dir, "client.can", revisionClientB, base)
	if !hasCode(diags, CodeRevisionIdentity) {
		t.Fatalf("drifted world under accepted baseline reported no identity finding")
	}
	for _, d := range diags {
		if d.Code == CodeRevisionIdentity && d.Sev != "error" {
			t.Fatalf("identity finding is not an error: %+v", d)
		}
	}
}

// TestLSPBaselineNil pins the additive path: no baseline configured
// means no identity findings, exactly today's behavior.
func TestLSPBaselineNil(t *testing.T) {
	dir := lspBaselineDir(t, revisionModelB, revisionClientB)
	if diags := diagnoseWith(dir, "client.can", revisionClientB, nil); hasCode(diags, CodeRevisionIdentity) {
		t.Fatalf("nil baseline reported identity drift: %+v", diags)
	}
}

// TestLSPBaselineUnaccepted pins generation-is-not-acceptance in the
// editor: a generated-but-unaccepted baseline still squiggles.
func TestLSPBaselineUnaccepted(t *testing.T) {
	dir := lspBaselineDir(t, revisionModelB, revisionClientB)
	base := lspAcceptedBaseline(t)
	base.Accepted = false
	if diags := diagnoseWith(dir, "client.can", revisionClientB, base); !hasCode(diags, CodeRevisionIdentity) {
		t.Fatalf("unaccepted baseline reported no identity finding")
	}
}

// TestLSPArgsBaseline pins flag parsing: --baseline was retired with the
// baseline-veto handshake and is now refused; bare `lsp` or --stdio
// starts the stdio server.
func TestLSPArgsBaseline(t *testing.T) {
	if err := parseLSPArgs([]string{"--baseline", "base.json"}); err == nil {
		t.Fatalf("--baseline accepted")
	}
	if err := parseLSPArgs(nil); err != nil {
		t.Fatalf("parse bare: err=%v", err)
	}
	if err := parseLSPArgs([]string{"--format", "json"}); err == nil {
		t.Fatalf("unknown flag accepted")
	}
}

// TestLSPArgsStdio pins the editor handshake: vscode-languageclient
// over stdio transport always spawns `canlc lsp --stdio`, so the
// marker must be accepted (and ignored).
func TestLSPArgsStdio(t *testing.T) {
	if err := parseLSPArgs([]string{"--stdio"}); err != nil {
		t.Fatalf("parse --stdio: err=%v", err)
	}
	if err := parseLSPArgs([]string{"lsp", "--stdio"}); err == nil {
		t.Fatalf("positional arg accepted")
	}
}

// TestLSPBaselineRunGated pins the refusal: `lsp --baseline` is a
// usage error naming the retired flag, so a server never starts.
func TestLSPBaselineRunGated(t *testing.T) {
	err := parseLSPArgs([]string{"--baseline", "base.json"})
	if err == nil || !strings.Contains(err.Error(), "baseline") {
		t.Fatalf("err=%v", err)
	}
	if code := runLSP([]string{"--baseline", "base.json"}); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

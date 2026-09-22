package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// I41 protocol exchanges: drive serveLSP over buffers with framed
// JSON-RPC and assert the published diagnostics, versions, and jumps.

const serverMain = `package app
    provides [tally, helper]
    uses []

fn int helper
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 1
    ok seed

fn int tally
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 2
    ok call helper(seed)
`

const serverSecond = `package app
    provides [describe]
    uses []

fn str describe
    emits []
    given
        int seed
    asserts
        sample: 1 => ok "one"
    ok "one"
`

const serverPoison = `package offline
    provides [run_poison]
    uses []

fn int run_poison
    emits []
    asserts
        explosive: => ok (1 / 0)
    ok (1 / 0)
`

func writeServerProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	for name, text := range files {
		write(name, text)
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func runExchange(t *testing.T, bodies []string) []map[string]any {
	t.Helper()
	var in bytes.Buffer
	for _, body := range bodies {
		in.WriteString(frame(body))
	}
	var out bytes.Buffer
	serveLSP(bufio.NewReader(&in), bufio.NewWriter(&out))
	raw := out.Bytes()
	var frames []map[string]any
	for len(raw) > 0 {
		head, body, ok := bytes.Cut(raw, []byte("\r\n\r\n"))
		if !ok {
			t.Fatalf("truncated frame header in %q", raw)
		}
		var length int
		if _, err := fmt.Sscanf(string(head), "Content-Length: %d", &length); err != nil {
			t.Fatalf("bad frame header %q", head)
		}
		if len(body) < length {
			t.Fatalf("truncated frame body: want %d, have %d", length, len(body))
		}
		var decoded map[string]any
		if err := json.Unmarshal(body[:length], &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		frames = append(frames, decoded)
		raw = body[length:]
	}
	return frames
}

func didOpen(uri, text string, version int64) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":%s,"languageId":"can","version":%d,"text":%s}}}`,
		jsonQuote(uri), version, jsonQuote(text))
}

func didChange(uri, text string, version int64) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":%s,"version":%d},"contentChanges":[{"text":%s}]}}`,
		jsonQuote(uri), version, jsonQuote(text))
}

func didClose(uri string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":%s}}}`, jsonQuote(uri))
}

func definition(id int, uri string, line, character int) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/definition","params":{"textDocument":{"uri":%s},"position":{"line":%d,"character":%d}}}`,
		id, jsonQuote(uri), line, character)
}

func jsonQuote(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

func publishes(frames []map[string]any) []map[string]any {
	var out []map[string]any
	for _, f := range frames {
		if f["method"] == "textDocument/publishDiagnostics" {
			out = append(out, f["params"].(map[string]any))
		}
	}
	return out
}

func lastPublishFor(frames []map[string]any, uri string) map[string]any {
	var last map[string]any
	for _, p := range publishes(frames) {
		if p["uri"] == uri {
			last = p
		}
	}
	return last
}

func diagnosticsOf(t *testing.T, params map[string]any) []map[string]any {
	t.Helper()
	raw, ok := params["diagnostics"].([]any)
	if !ok {
		t.Fatalf("diagnostics not an array: %v", params)
	}
	var out []map[string]any
	for _, d := range raw {
		out = append(out, d.(map[string]any))
	}
	return out
}

func positionOf(t *testing.T, text, needle string) (int, int) {
	t.Helper()
	plain := strings.Replace(needle, "|", "", 1)
	index := strings.Index(text, plain)
	if index < 0 {
		t.Fatalf("needle %q not in fixture", needle)
	}
	cursor := index + strings.Index(needle, "|")
	line := strings.Count(text[:cursor], "\n")
	character := cursor - (strings.LastIndex(text[:cursor], "\n") + 1)
	return line, character
}

// TestServerCleanPublishesEmpty pins the quiet case: a healthy open
// file publishes an empty array echoing the document version.
func TestServerCleanPublishesEmpty(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, serverMain, 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish for %s in %v", uri, frames)
	}
	if diags := diagnosticsOf(t, last); len(diags) != 0 {
		t.Fatalf("healthy file published %v", diags)
	}
	if last["version"] != 1.0 {
		t.Fatalf("version not echoed: %v", last)
	}
}

// TestServerUnsavedSibling pins overlay coherence: breaking an unsaved
// sibling squiggles the sibling while the opener stays clean, and the
// bytes on disk never change.
func TestServerUnsavedSibling(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain, "src/second.can": serverSecond})
	mainURI := uriFromPath(filepath.Join(root, "src/main.can"))
	secondURI := uriFromPath(filepath.Join(root, "src/second.can"))
	broken := strings.Replace(serverSecond, `ok "one"`, `ok call missing_fn(seed)`, 1)
	frames := runExchange(t, []string{
		didOpen(mainURI, serverMain, 1),
		didOpen(secondURI, serverSecond, 1),
		didChange(secondURI, broken, 2),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	sibling := lastPublishFor(frames, secondURI)
	if sibling == nil {
		t.Fatalf("no publish for sibling in %v", frames)
	}
	diags := diagnosticsOf(t, sibling)
	if len(diags) != 1 || !strings.Contains(diags[0]["message"].(string), "missing_fn") {
		t.Fatalf("sibling breakage not reported: %v", diags)
	}
	if sibling["version"] != 2.0 {
		t.Fatalf("sibling version not echoed: %v", sibling)
	}
	if main := lastPublishFor(frames, mainURI); main == nil {
		t.Fatalf("opener lost its publish")
	} else if diags := diagnosticsOf(t, main); len(diags) != 0 {
		t.Fatalf("opener dirtied by sibling: %v", diags)
	}
	if disk, err := os.ReadFile(filepath.Join(root, "src/second.can")); err != nil || string(disk) != serverSecond {
		t.Fatalf("editor write reached disk: %v", err)
	}
}

// TestServerDuplicateBasenames pins full-path identity: two open files
// sharing a basename diagnose independently under their own URIs.
func TestServerDuplicateBasenames(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/a/dup.can": serverMain, "src/b/dup.can": serverSecond})
	aURI := uriFromPath(filepath.Join(root, "src/a/dup.can"))
	bURI := uriFromPath(filepath.Join(root, "src/b/dup.can"))
	broken := strings.Replace(serverSecond, "fn str describe", "fn str describe(", 1)
	frames := runExchange(t, []string{
		didOpen(aURI, serverMain, 1),
		didOpen(bURI, serverSecond, 1),
		didChange(bURI, broken, 2),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	brokenParams := lastPublishFor(frames, bURI)
	if brokenParams == nil || len(diagnosticsOf(t, brokenParams)) != 1 {
		t.Fatalf("broken twin unpublished: %v", brokenParams)
	}
	if !strings.Contains(brokenParams["uri"].(string), "/b/dup.can") {
		t.Fatalf("diagnostic URI lost its directory: %v", brokenParams)
	}
	clean := lastPublishFor(frames, aURI)
	if clean == nil || len(diagnosticsOf(t, clean)) != 0 {
		t.Fatalf("clean twin misdiagnosed: %v", clean)
	}
}

// TestServerRapidEdits pins last-version-wins: only the newest buffer
// state survives a burst of keystrokes.
func TestServerRapidEdits(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	broken := strings.Replace(serverMain, "fn int helper", "fn int helper(", 1)
	frames := runExchange(t, []string{
		didOpen(uri, serverMain, 1),
		didChange(uri, broken, 2),
		didChange(uri, broken, 3),
		didChange(uri, serverMain, 4),
		didChange(uri, broken, 5),
		didChange(uri, serverMain, 6),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	if last["version"] != 6.0 {
		t.Fatalf("stale version published: %v", last)
	}
	if diags := diagnosticsOf(t, last); len(diags) != 0 {
		t.Fatalf("stale errors survived the fix: %v", diags)
	}
}

// TestServerCloseClears pins stale-error removal: closing a broken
// file publishes an empty versionless clear for its URI.
func TestServerCloseClears(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	broken := strings.Replace(serverMain, "fn int helper", "fn int helper(", 1)
	frames := runExchange(t, []string{
		didOpen(uri, broken, 1),
		didClose(uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	pubs := publishes(frames)
	if len(pubs) < 2 {
		t.Fatalf("expected breakage then clear, got %v", pubs)
	}
	if diags := diagnosticsOf(t, pubs[0]); len(diags) != 1 {
		t.Fatalf("breakage unpublished: %v", pubs[0])
	}
	clear := pubs[len(pubs)-1]
	if diags := diagnosticsOf(t, clear); len(diags) != 0 {
		t.Fatalf("close did not clear: %v", clear)
	}
	if _, versioned := clear["version"]; versioned {
		t.Fatalf("clear carries a version: %v", clear)
	}
}

// TestServerMalformedKeepsServing pins crash-freedom: garbage bytes and
// a corrupt frame never wedge the server; later files still diagnose.
func TestServerMalformedKeepsServing(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain, "src/second.can": serverSecond})
	badURI := uriFromPath(filepath.Join(root, "src/main.can"))
	goodURI := uriFromPath(filepath.Join(root, "src/second.can"))
	frames := runExchange(t, []string{
		didOpen(badURI, "{{{\x00\xff", 1),
		`{"jsonrpc": broken`,
		didOpen(goodURI, serverSecond, 2),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	good := lastPublishFor(frames, goodURI)
	if good == nil || len(diagnosticsOf(t, good)) != 0 {
		t.Fatalf("server wedged after garbage: %v", good)
	}
}

// TestServerUnicodePublish pins UTF-16 fidelity: published columns
// equal the bridge's UTF-16 columns exactly, even beside multibyte text.
func TestServerUnicodePublish(t *testing.T) {
	withPoem := strings.Replace(serverMain, "fn int helper", "str poem = \"é😀 — e\"\n\nfn int helper", 1)
	broken := strings.Replace(withPoem, "fn int helper", "fn int helper(", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": withPoem})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, broken, 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diags)
	}
	open := filepath.Join(root, "src/main.can")
	overlay := project.NewOverlay()
	if err := overlay.Set(open, 1, broken); err != nil {
		t.Fatal(err)
	}
	snapshot, err := driver.CheckSnapshot(root, open, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("bridge disagrees: %+v", snapshot.Diagnostics)
	}
	want := snapshot.Diagnostics[0]
	rng := diags[0]["range"].(map[string]any)
	start := rng["start"].(map[string]any)
	end := rng["end"].(map[string]any)
	if start["line"] != float64(want.Line) || start["character"] != float64(want.Start) || end["character"] != float64(want.End) {
		t.Fatalf("wire range %v != bridge %+v", rng, want)
	}
	if diags[0]["code"] != want.Code || diags[0]["message"] != want.Message {
		t.Fatalf("wire %v != bridge %+v", diags[0], want)
	}
}

// TestServerDefinition pins go-to-definition over the wire: the call
// jumps to the declaring span in the same file.
func TestServerDefinition(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	line, character := positionOf(t, serverMain, "call help|er(seed)")
	frames := runExchange(t, []string{
		didOpen(uri, serverMain, 1),
		definition(1, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	var result map[string]any
	for _, f := range frames {
		if id, ok := f["id"]; ok && id == 1.0 {
			result, _ = f["result"].(map[string]any)
		}
	}
	if result == nil {
		t.Fatalf("no definition result in %v", frames)
	}
	if result["uri"] != uri {
		t.Fatalf("jump left the file: %v", result)
	}
	rng := result["range"].(map[string]any)
	start := rng["start"].(map[string]any)
	declLine := strings.Count(serverMain[:strings.Index(serverMain, "fn int helper")], "\n")
	if start["line"] != float64(declLine) {
		t.Fatalf("jump landed on %v, want declaration line %d", rng, declLine)
	}
}

const serverNative = `package app
    provides []
    uses [codec]

connection classifier
    endpoint "http://127.0.0.1:1/systemone"
    auth bearer env "CAN_I27_TOKEN"
    timeout_ms 1000
    metadata
        protocol "typesafe_systemone_v1"
        model "jev-latest"

fn float report
    emits [codec::invalid_data]
    given
        float probability
        str marker
    asserts
        sample: 0.5, "T" => ok 0.5
    ok probability

record choice_weights choice float weights from classifier
    emits [codec::invalid_data]
    confidence as certainty
    asks "Weights"
        first "First" => relay call report(% + certainty, "C")
        second "Second" => relay call report(%, "D")

choice_weights tally = choice_weights(0.1, 0.2, 0.3)

fn float show
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 0.1
    ok tally.first
`

// TestServerGeneratedFieldDefinition pins navigation into AI-generated
// members: the choice-arm field jumps to its declaring arm over the wire.
func TestServerGeneratedFieldDefinition(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/native.can": serverNative})
	uri := uriFromPath(filepath.Join(root, "src/native.can"))
	line, character := positionOf(t, serverNative, "tally.f|irst")
	frames := runExchange(t, []string{
		didOpen(uri, serverNative, 1),
		definition(1, uri, line, character),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	var result map[string]any
	for _, f := range frames {
		if id, ok := f["id"]; ok && id == 1.0 {
			result, _ = f["result"].(map[string]any)
		}
	}
	if result == nil {
		t.Fatalf("no definition result in %v", frames)
	}
	if result["uri"] != uri {
		t.Fatalf("jump left the file: %v", result)
	}
	rng := result["range"].(map[string]any)
	start := rng["start"].(map[string]any)
	end := rng["end"].(map[string]any)
	armLine := strings.Count(serverNative[:strings.Index(serverNative, `first "First"`)], "\n")
	if start["line"] != float64(armLine) {
		t.Fatalf("jump landed on %v, want arm line %d", rng, armLine)
	}
	landed := serverNative
	for i := 0; i < armLine; i++ {
		landed = landed[strings.IndexByte(landed, '\n')+1:]
	}
	armEnd := strings.IndexByte(landed, '\n')
	if token := landed[int(start["character"].(float64)):int(end["character"].(float64))]; token != "first" {
		t.Fatalf("jump token %q on line %q", token, landed[:armEnd])
	}
}

// TestServerParsesWithoutExecuting pins inertness: assertions that
// would divide by zero and calls that would dial out produce no
// diagnostics, because the server parses but never runs.
func TestServerParsesWithoutExecuting(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverPoison})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, serverPoison, 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	if diags := diagnosticsOf(t, last); len(diags) != 0 {
		t.Fatalf("poison fixture diagnosed (or executed): %v", diags)
	}
}

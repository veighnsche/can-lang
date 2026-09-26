// canlc lsp: minimal Language Server over stdio.
//
// Speaks just enough JSON-RPC to drive editor squiggles: initialize,
// textDocument/didOpen, textDocument/didChange, shutdown/exit. Every
// keystroke diagnoses the in-memory overlay snapshot through the
// current parse/resolve/check bridge and publishes diagnostics.
//
// Run: canlc lsp [--stdio]   (editors connect stdout/stdin with
// Content-Length framing).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// ------------------------------------------------------------- protocol ---

type rpcMsg struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
}

type docID struct {
	URI string `json:"uri"`
}

func pathFromURI(uri string) string {
	parsed, err := url.ParseRequestURI(uri)
	if err != nil || parsed.Scheme != "file" {
		return ""
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return ""
	}
	return path
}

func uriFromPath(path string) string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return "file://" + strings.Join(segments, "/")
}

func writeFrame(w *bufio.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body))
	w.Write(body)
	return w.Flush()
}

// ---------------------------------------------------------------- server ---

// The I41 server keeps the stdio JSON-RPC transport and replaces every
// semantic hook: each keystroke diagnoses an in-memory overlay snapshot
// through the current parse, resolve, and check pipeline, and definition
// requests resolve through file, package, prelude, and import scopes. No
// request builds, runs, asserts, dials out, queries, reads the
// environment, emits files, or mutates registries; the --baseline
// execution hook is gone, and predecessor spellings diagnose as ordinary
// current-syntax errors.
type lspDoc struct {
	path    string
	text    string
	version int64
}

type lspServer struct {
	docs      map[string]*lspDoc
	overlay   *project.Overlay
	published map[string]bool
}

func newLSPServer() *lspServer {
	return &lspServer{docs: map[string]*lspDoc{}, overlay: project.NewOverlay(), published: map[string]bool{}}
}

func parseLSPArgs(argv []string) error {
	for _, arg := range argv {
		if arg == "--baseline" || strings.HasPrefix(arg, "--baseline=") {
			return fmt.Errorf("canlc lsp: --baseline was retired with the baseline-veto handshake; the server now reports live diagnostics")
		}
	}
	fs := flag.NewFlagSet("canlc lsp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("stdio", false, "stdio transport marker from LSP clients (ignored)")
	if err := fs.Parse(argv); err != nil {
		return fmt.Errorf("usage: canlc lsp [--stdio]")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: canlc lsp [--stdio]")
	}
	return nil
}

func runLSP(argv []string) int {
	if err := parseLSPArgs(argv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	serveLSP(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout))
	return 0
}

func serveLSP(in *bufio.Reader, out *bufio.Writer) {
	server := newLSPServer()
	respond := func(id *json.RawMessage, result any) {
		writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	}
	respondErr := func(id *json.RawMessage, code int, message string) {
		writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
	}
	for {
		var length int
		for {
			line, err := in.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			}
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(in, body); err != nil {
			return
		}
		var msg rpcMsg
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}
		switch msg.Method {
		case "initialize":
			respond(msg.ID, map[string]any{"capabilities": map[string]any{"textDocumentSync": 1, "definitionProvider": true, "documentFormattingProvider": true, "hoverProvider": true}})
		case "initialized", "$/cancelRequest":
		case "textDocument/didOpen":
			var p struct {
				TextDocument struct {
					URI     string `json:"uri"`
					Text    string `json:"text"`
					Version int64  `json:"version"`
				} `json:"textDocument"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				continue
			}
			server.open(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
			server.diagnose(out, p.TextDocument.URI)
		case "textDocument/didChange":
			var p struct {
				TextDocument struct {
					URI     string `json:"uri"`
					Version int64  `json:"version"`
				} `json:"textDocument"`
				Changes []struct {
					Text string `json:"text"`
				} `json:"contentChanges"`
			}
			if json.Unmarshal(msg.Params, &p) != nil || len(p.Changes) == 0 {
				continue
			}
			server.change(p.TextDocument.URI, p.Changes[len(p.Changes)-1].Text, p.TextDocument.Version)
			server.diagnose(out, p.TextDocument.URI)
		case "textDocument/didClose":
			var p struct {
				TextDocument docID `json:"textDocument"`
			}
			if json.Unmarshal(msg.Params, &p) == nil {
				server.close(out, p.TextDocument.URI)
			}
		case "textDocument/definition":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid definition params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.definition(p.TextDocument.URI, p.Position.Line, p.Position.Character))
			}
		case "textDocument/hover":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid hover params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.hover(p.TextDocument.URI, p.Position.Line, p.Position.Character))
			}
		case "textDocument/formatting":
			var p struct {
				TextDocument docID `json:"textDocument"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid formatting params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.formatting(p.TextDocument.URI))
			}
		case "shutdown":
			respond(msg.ID, nil)
		case "exit":
			return
		default:
			if msg.ID != nil {
				respondErr(msg.ID, -32601, "unknown method "+msg.Method)
			}
		}
	}
}

func (s *lspServer) open(uri, text string, version int64) {
	path := pathFromURI(uri)
	s.docs[uri] = &lspDoc{path: path, text: text, version: version}
	if path == "" {
		return
	}
	if err := s.overlay.Set(path, version, text); err != nil {
		s.overlay.Clear(path)
	}
}

func (s *lspServer) change(uri, text string, version int64) {
	doc, ok := s.docs[uri]
	if !ok {
		doc = &lspDoc{path: pathFromURI(uri)}
		s.docs[uri] = doc
	}
	doc.text, doc.version = text, version
	if doc.path == "" {
		return
	}
	if err := s.overlay.Set(doc.path, version, text); err != nil {
		s.overlay.Clear(doc.path)
	}
}

func (s *lspServer) close(out *bufio.Writer, uri string) {
	doc, ok := s.docs[uri]
	if !ok {
		return
	}
	if doc.path != "" {
		s.overlay.Clear(doc.path)
	}
	delete(s.docs, uri)
	delete(s.published, uri)
	publishBridgeDiagnostics(out, uri, nil, nil)
}

func (s *lspServer) diagnose(out *bufio.Writer, uri string) {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		publishBridgeDiagnostics(out, uri, nil, nil)
		return
	}
	root := discoverRoot(doc.path)
	snapshot, err := driver.CheckSnapshot(root, doc.path, s.overlay)
	if err != nil {
		publishBridgeDiagnostics(out, uri, &doc.version, nil)
		return
	}
	byURI := map[string][]driver.Diagnostic{}
	for _, diagnostic := range snapshot.Diagnostics {
		uri := uriFromPath(diagnostic.File)
		byURI[uri] = append(byURI[uri], diagnostic)
	}
	for openURI, openDoc := range s.docs {
		version := openDoc.version
		publishBridgeDiagnostics(out, openURI, &version, byURI[openURI])
		if len(byURI[openURI]) > 0 {
			s.published[openURI] = true
		} else {
			delete(s.published, openURI)
		}
	}
	for diagnosedURI, diags := range byURI {
		if _, open := s.docs[diagnosedURI]; open {
			continue
		}
		publishBridgeDiagnostics(out, diagnosedURI, nil, diags)
		s.published[diagnosedURI] = true
	}
	for publishedURI := range s.published {
		if _, still := byURI[publishedURI]; still {
			continue
		}
		if _, open := s.docs[publishedURI]; open {
			continue
		}
		publishBridgeDiagnostics(out, publishedURI, nil, nil)
		delete(s.published, publishedURI)
	}
}

// formatting renders the open buffer through formatSource and validates
// the candidate like --write before returning any edit: the live overlay
// snapshot must be error-free and the formatted text must check clean
// through a copied overlay carrying every other open buffer. Anything
// unformattable or failing validation yields null so the editor applies
// nothing; an already-canonical buffer yields an empty edit list.
// Warnings never block formatting, matching the CLI's warnings-only
// exit 0.
func (s *lspServer) formatting(uri string) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	formatted, err := formatSource(doc.path, doc.text)
	if err != nil {
		return nil
	}
	if formatted == doc.text {
		return []any{}
	}
	root := discoverRoot(doc.path)
	original, err := driver.CheckSnapshot(root, doc.path, s.overlay)
	if err != nil || snapshotHasErrors(original) {
		return nil
	}
	candidate := project.NewOverlay()
	for path, entry := range s.overlay.Snapshot() {
		if err := candidate.Set(path, entry.Version, entry.Text); err != nil {
			return nil
		}
	}
	if err := candidate.Set(doc.path, doc.version, formatted); err != nil {
		return nil
	}
	proposed, err := driver.CheckSnapshot(root, doc.path, candidate)
	if err != nil || snapshotHasErrors(proposed) {
		return nil
	}
	return []any{map[string]any{"range": fullDocumentRange(doc.text), "newText": formatted}}
}

// snapshotHasErrors reports whether a diagnosis carries anything above
// advisory findings. Warning-severity entries never count: they ride
// along with successful checks on both the CLI and the editor streams.
func snapshotHasErrors(snapshot *driver.Snapshot) bool {
	if snapshot == nil {
		return true
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity != "warning" {
			return true
		}
	}
	return false
}

// fullDocumentRange spans the whole buffer in editor coordinates: the
// formatter rewrites the document, so the edit replaces it entirely.
func fullDocumentRange(text string) map[string]any {
	lines := strings.Split(text, "\n")
	last := strings.TrimSuffix(lines[len(lines)-1], "\r")
	width := 0
	for _, r := range last {
		if r > 0xFFFF {
			width += 2
		} else {
			width++
		}
	}
	return map[string]any{
		"start": map[string]any{"line": 0, "character": 0},
		"end":   map[string]any{"line": len(lines) - 1, "character": width},
	}
}

// hover answers type-at-offset over the checked snapshot of the open
// buffer: the resolved contract rendered as Markdown plus the hovered
// token range. Like definition it reads an inert overlay snapshot and
// declines to null wherever the name does not resolve, so unresolved
// source never receives a guessed type.
func (s *lspServer) hover(uri string, line, character int) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	result, ok, err := driver.HoverAt(snapshot, doc.path, line, character)
	if err != nil || !ok {
		return nil
	}
	return map[string]any{
		"contents": map[string]any{"kind": "markdown", "value": result.Contents},
		"range": map[string]any{
			"start": map[string]any{"line": result.Line, "character": result.Start},
			"end":   map[string]any{"line": result.Line, "character": result.End},
		},
	}
}

func (s *lspServer) definition(uri string, line, character int) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	location, ok, err := driver.Definition(snapshot, doc.path, line, character)
	if err != nil || !ok {
		return nil
	}
	return map[string]any{
		"uri": uriFromPath(location.File),
		"range": map[string]any{
			"start": map[string]any{"line": location.Line, "character": location.Start},
			"end":   map[string]any{"line": location.Line, "character": location.End},
		},
	}
}

// discoverRoot walks up from a source file to the enclosing project
// manifest. Without one, the file's own directory becomes the diagnosis
// root so the bridge reports the missing project instead of silence.
func discoverRoot(path string) string {
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		abs = path
	}
	dir := abs
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for i := 0; i < 256; i++ {
		if info, err := os.Lstat(filepath.Join(dir, "can.project.json")); err == nil && info.Mode().IsRegular() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(abs)
}

func publishBridgeDiagnostics(out *bufio.Writer, uri string, version *int64, diags []driver.Diagnostic) error {
	items := []any{}
	for _, d := range diags {
		line := d.Line
		if line < 0 {
			line = 0
		}
		endLine := d.EndLine
		if endLine < line {
			endLine = line
		}
		start, end := d.Start, d.End
		if start < 0 {
			start = 0
		}
		if endLine == line && end < start {
			end = start
		}
		if end < 0 {
			end = 0
		}
		severity := 1
		if d.Severity == "warning" {
			severity = 2
		}
		item := map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": line, "character": start},
				"end":   map[string]any{"line": endLine, "character": end},
			},
			"severity": severity,
			"source":   "canlc",
			"message":  d.Message,
		}
		if d.Code != "" {
			item["code"] = d.Code
		}
		if len(d.Related) > 0 {
			related := []any{}
			for _, r := range d.Related {
				rLine := r.Line
				if rLine < 0 {
					rLine = 0
				}
				rEndLine := r.EndLine
				if rEndLine < rLine {
					rEndLine = rLine
				}
				rStart, rEnd := r.Start, r.End
				if rStart < 0 {
					rStart = 0
				}
				if rEndLine == rLine && rEnd < rStart {
					rEnd = rStart
				}
				if rEnd < 0 {
					rEnd = 0
				}
				related = append(related, map[string]any{
					"location": map[string]any{
						"uri": uriFromPath(r.File),
						"range": map[string]any{
							"start": map[string]any{"line": rLine, "character": rStart},
							"end":   map[string]any{"line": rEndLine, "character": rEnd},
						},
					},
					"message": r.Message,
				})
			}
			item["relatedInformation"] = related
		}
		items = append(items, item)
	}
	params := map[string]any{"uri": uri, "diagnostics": items}
	if version != nil {
		params["version"] = *version
	}
	return writeFrame(out, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params":  params,
	})
}

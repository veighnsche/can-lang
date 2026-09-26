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
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
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
			respond(msg.ID, map[string]any{"capabilities": map[string]any{"textDocumentSync": 1, "definitionProvider": true, "documentFormattingProvider": true, "hoverProvider": true, "referencesProvider": true}})
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
		case "textDocument/references":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
				Context *struct {
					IncludeDeclaration bool `json:"includeDeclaration"`
				} `json:"context"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid references params")
				}
				continue
			}
			if msg.ID != nil {
				include := true
				if p.Context != nil {
					include = p.Context.IncludeDeclaration
				}
				respond(msg.ID, server.references(p.TextDocument.URI, p.Position.Line, p.Position.Character, include))
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

// references answers find-references over the checked snapshot: the token
// at the cursor resolves to its binding identity through the project
// index — top-level symbols, record fields and methods, and
// function-local bindings by scope identity — and every indexed
// occurrence sharing that identity is returned. Like definition it reads
// an inert overlay snapshot and declines to null wherever the token does
// not resolve, so unresolved source never receives a guessed reference.
func (s *lspServer) references(uri string, line, character int, includeDeclaration bool) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	locations := references(snapshot, doc.path, line, character, includeDeclaration)
	if len(locations) == 0 {
		return nil
	}
	out := make([]any, 0, len(locations))
	for _, location := range locations {
		out = append(out, map[string]any{
			"uri": uriFromPath(location.File),
			"range": map[string]any{
				"start": map[string]any{"line": location.Line, "character": location.Start},
				"end":   map[string]any{"line": location.Line, "character": location.End},
			},
		})
	}
	return out
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

// canlc references: project-wide find-references over an inert snapshot.
//
// The index walks every source in the loaded graph and resolves each name
// occurrence to a binding identity: top-level symbols by their resolve
// identity, record fields and methods behind annotation-known receivers,
// and function-local bindings (inputs, steps, chain and pattern bindings)
// by their binding site. A query resolves the token at the cursor to its
// identity and returns every occurrence sharing it, so shadowed or
// same-spelled bindings in other scopes never leak into the results.
// Anything unresolvable declines to null rather than guessing.

// refOccurrence is one indexed name token: the canonical file, the token
// span, the binding identity it resolves to, and whether the token is the
// binding's own declaration site.
type refOccurrence struct {
	file string
	span source.Span
	id   string
	decl bool
}

// referenceIndex is the whole-project occurrence table behind one query.
type referenceIndex struct {
	byFile map[string][]*refOccurrence
	byID   map[string][]*refOccurrence
}

// buildReferenceIndex resolves every referenceable token in the snapshot.
// Files without resolved scope data contribute nothing; unresolvable
// tokens are left out so queries over them decline.
func buildReferenceIndex(snapshot *driver.Snapshot) *referenceIndex {
	index := &referenceIndex{byFile: map[string][]*refOccurrence{}, byID: map[string][]*refOccurrence{}}
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return index
	}
	keys := make([]string, 0, len(snapshot.Graph.Projects))
	for key := range snapshot.Graph.Projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, src := range snapshot.Graph.Projects[key].Sources {
			if src.Syntax == nil {
				continue
			}
			resolved, ok := snapshot.World.Files[src]
			if !ok || resolved == nil {
				continue
			}
			walker := &refWalker{path: src.Path, file: resolved, pkgID: packageID(resolved)}
			walker.walkFile(src.Syntax)
			for _, occurrence := range walker.occs {
				index.byFile[occurrence.file] = append(index.byFile[occurrence.file], occurrence)
				index.byID[occurrence.id] = append(index.byID[occurrence.id], occurrence)
			}
		}
	}
	for file := range index.byFile {
		sort.Slice(index.byFile[file], func(i, j int) bool {
			if index.byFile[file][i].span.Start != index.byFile[file][j].span.Start {
				return index.byFile[file][i].span.Start < index.byFile[file][j].span.Start
			}
			return index.byFile[file][i].span.End < index.byFile[file][j].span.End
		})
	}
	for id := range index.byID {
		sort.Slice(index.byID[id], func(i, j int) bool {
			if index.byID[id][i].file != index.byID[id][j].file {
				return index.byID[id][i].file < index.byID[id][j].file
			}
			return index.byID[id][i].span.Start < index.byID[id][j].span.Start
		})
	}
	return index
}

func packageID(file *compileresolve.File) string {
	if file == nil || file.Package == nil {
		return ""
	}
	return file.Package.ID
}

// occurrenceAt returns the innermost indexed token covering the offset.
func (index *referenceIndex) occurrenceAt(file string, offset int) *refOccurrence {
	var best *refOccurrence
	for _, occurrence := range index.byFile[file] {
		if occurrence.span.Start > offset {
			break
		}
		if offset >= occurrence.span.End {
			continue
		}
		if best == nil || occurrence.span.End-occurrence.span.Start < best.span.End-best.span.Start {
			best = occurrence
		}
	}
	return best
}

// references resolves the token at an editor offset to its binding
// identity and reports every indexed occurrence sharing it, declaration
// site included unless the caller opts out. It declines wherever the
// token is unindexed or no occurrence converts to editor coordinates.
func references(snapshot *driver.Snapshot, file string, line, character int, includeDeclaration bool) []driver.Location {
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return nil
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil
	}
	var src *project.Source
	for _, p := range snapshot.Graph.Projects {
		for _, s := range p.Sources {
			if s.Path == canonical {
				src = s
			}
		}
	}
	if src == nil || src.Syntax == nil {
		return nil
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return nil
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return nil
	}
	index := buildReferenceIndex(snapshot)
	at := index.occurrenceAt(canonical, offset)
	if at == nil {
		return nil
	}
	files := map[string]*source.File{canonical: text}
	converted := func(path string, span source.Span) (driver.Location, bool) {
		converted, ok := files[path]
		if !ok {
			for _, p := range snapshot.Graph.Projects {
				for _, s := range p.Sources {
					if s.Path == path {
						file, err := source.New(path, string(s.Bytes))
						if err != nil {
							return driver.Location{}, false
						}
						files[path] = file
						converted = file
					}
				}
			}
		}
		if converted == nil {
			return driver.Location{}, false
		}
		start, startErr := converted.UTF16Position(span.Start)
		end, endErr := converted.UTF16Position(span.End)
		if startErr != nil || endErr != nil || start.Line != end.Line {
			return driver.Location{}, false
		}
		return driver.Location{File: path, Line: start.Line, Start: start.Character, End: end.Character}, true
	}
	var out []driver.Location
	for _, occurrence := range index.byID[at.id] {
		if occurrence.decl && !includeDeclaration {
			continue
		}
		location, ok := converted(occurrence.file, occurrence.span)
		if !ok {
			continue
		}
		out = append(out, location)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ------------------------------------------------------------- walker ---

type refBinder struct {
	name string
	span source.Span
	id   string
}

type refScope struct {
	binders []*refBinder
}

// refWalker resolves one file's tokens against the checked World plus a
// stack of function-local scopes. Binder visibility is order-sensitive: a
// use only sees binders starting before it, so a later same-spelled
// binding never captures an earlier use.
type refWalker struct {
	path   string
	pkgID  string
	file   *compileresolve.File
	scopes []*refScope
	occs   []*refOccurrence
}

func (w *refWalker) record(span source.Span, id string, decl bool) {
	if id == "" {
		return
	}
	w.occs = append(w.occs, &refOccurrence{file: w.path, span: span, id: id, decl: decl})
}

func (w *refWalker) localID(span source.Span) string {
	return fmt.Sprintf("local:%s:%d:%d", w.path, span.Start, span.End)
}

func localIDFor(path string, span source.Span) string {
	return fmt.Sprintf("local:%s:%d:%d", path, span.Start, span.End)
}

func symbolID(symbol *compileresolve.Symbol) string {
	if symbol == nil || symbol.ID == "" {
		return ""
	}
	return "symbol:" + symbol.ID
}

// selfID names a declaration in the walked file by the identity resolve
// assigned it: package identity plus the declared name.
func (w *refWalker) selfID(name string) string {
	if w.pkgID == "" || name == "" {
		return ""
	}
	return "symbol:" + w.pkgID + "::" + name
}

func (w *refWalker) push() {
	w.scopes = append(w.scopes, &refScope{})
}

func (w *refWalker) pop() {
	w.scopes = w.scopes[:len(w.scopes)-1]
}

func (w *refWalker) bind(name syntax.Token) *refBinder {
	binder := &refBinder{name: name.Text, span: name.Span, id: w.localID(name.Span)}
	w.scopes[len(w.scopes)-1].binders = append(w.scopes[len(w.scopes)-1].binders, binder)
	w.record(name.Span, binder.id, true)
	return binder
}

// lookup finds the innermost visible local binder: frames inside out, and
// within a frame the nearest binder starting before the use.
func (w *refWalker) lookup(name string, offset int) *refBinder {
	for i := len(w.scopes) - 1; i >= 0; i-- {
		var best *refBinder
		for _, binder := range w.scopes[i].binders {
			if binder.name != name || binder.span.Start >= offset {
				continue
			}
			best = binder
		}
		if best != nil {
			return best
		}
	}
	return nil
}

// name resolves one value-position token: the innermost visible local
// binder wins, else the checked World, else the token stays unindexed.
func (w *refWalker) name(qualified syntax.QualifiedName, usage compileresolve.Usage, offset int) {
	if binder := w.lookup(qualified.Name, offset); binder != nil && qualified.Package == "" {
		w.record(qualified.Span, binder.id, false)
		return
	}
	symbol, err := w.file.Lookup(nil, qualified, usage)
	if err != nil {
		return
	}
	w.record(qualified.Span, symbolID(symbol), false)
}

func (w *refWalker) walkFile(file *syntax.File) {
	for _, declaration := range file.Declarations {
		w.walkDecl(declaration)
	}
}

func (w *refWalker) walkDecl(declaration syntax.Declaration) {
	switch node := declaration.(type) {
	case *syntax.FunctionDecl:
		w.record(node.Name.Span, w.selfID(node.Name.Text), true)
		w.walkType(node.Result, compileresolve.TypeUse)
		if node.Receiver != nil {
			w.walkType(node.Receiver.Type, compileresolve.TypeUse)
		}
		w.walkBound(node.Errors)
		for i := range node.Inputs {
			w.walkType(node.Inputs[i].Type, compileresolve.TypeUse)
		}
		w.push()
		if node.Receiver != nil {
			w.bind(node.Receiver.Name)
		}
		for i := range node.Inputs {
			w.bind(node.Inputs[i].Name)
		}
		w.walkBlock(node.Body)
		for i := range node.Assertions {
			w.walkAssertion(&node.Assertions[i])
		}
		w.pop()
	case *syntax.RecordDecl:
		w.record(node.Name.Span, w.selfID(node.Name.Text), true)
		for i := range node.Fields {
			w.record(node.Fields[i].Name.Span, w.fieldID(w.selfID(node.Name.Text), node.Fields[i].Name.Text), true)
			w.walkType(node.Fields[i].Type, compileresolve.TypeUse)
		}
	case *syntax.VariantDecl:
		w.record(node.Name.Span, w.selfID(node.Name.Text), true)
		for _, alternative := range node.Alternatives {
			w.walkType(alternative, compileresolve.TypeUse)
		}
	case *syntax.ErrorDecl:
		w.record(node.Name.Span, w.selfID(node.Name.Text), true)
		for i := range node.Fields {
			w.record(node.Fields[i].Name.Span, w.fieldID(w.selfID(node.Name.Text), node.Fields[i].Name.Text), true)
			w.walkType(node.Fields[i].Type, compileresolve.TypeUse)
		}
	case *syntax.ValueDecl:
		w.record(node.Binding.Name.Span, w.selfID(node.Binding.Name.Text), true)
		w.walkType(node.Binding.Type, compileresolve.TypeUse)
		w.walkExpr(node.Binding.Value)
	case *syntax.QuestionDecl:
		if node.RecordName != nil {
			w.record(node.RecordName.Span, w.selfID(node.RecordName.Text), true)
		}
		for _, option := range node.Options {
			w.walkExpr(option.Description)
			w.walkExpr(option.Spread)
			w.walkBody(option.Body)
		}
		w.walkExpr(node.Asks)
		w.walkExpr(node.Minimum)
		w.walkBody(node.Fallback)
		w.walkBody(node.Shared)
	}
}

func (w *refWalker) fieldID(record, field string) string {
	if record == "" || field == "" {
		return ""
	}
	return "field:" + record + "." + field
}

func (w *refWalker) walkType(node syntax.TypeNode, usage compileresolve.Usage) {
	switch node := node.(type) {
	case *syntax.NamedType:
		for _, argument := range node.Arguments {
			w.walkType(argument, compileresolve.TypeUse)
		}
		w.name(node.Name, usage, node.Name.Span.Start)
	case *syntax.ArrayType:
		w.walkType(node.Element, compileresolve.TypeUse)
	case *syntax.CallableType:
		w.walkType(node.Result, compileresolve.TypeUse)
		for _, input := range node.Inputs {
			w.walkType(input, compileresolve.TypeUse)
		}
		w.walkBound(node.Errors)
	case *syntax.ChoiceArmType:
		w.walkType(node.Result, compileresolve.TypeUse)
		w.walkBound(node.Errors)
	}
}

func (w *refWalker) walkBound(bound syntax.ErrorBound) {
	for _, typ := range bound.Types {
		w.walkType(typ, compileresolve.ErrorUse)
	}
}

func (w *refWalker) walkBlock(block syntax.Block) {
	w.push()
	for _, step := range block.Steps {
		w.walkStep(step)
	}
	w.walkBody(block.Terminal)
	w.pop()
}

func (w *refWalker) walkStep(step syntax.Step) {
	switch node := step.(type) {
	case *syntax.BindingStep:
		w.walkType(node.Binding.Type, compileresolve.TypeUse)
		// The value walks before its own name binds: an initializer
		// never sees the binding it defines.
		w.walkExpr(node.Binding.Value)
		w.bind(node.Binding.Name)
	case *syntax.CallStep:
		w.walkExpr(node.Call)
	case *syntax.CoordinationStep:
		w.walkCoordination(&node.Coordination)
	}
}

func (w *refWalker) walkBody(body syntax.Body) {
	switch node := body.(type) {
	case *syntax.ValueBody:
		w.walkExpr(node.Value)
	case *syntax.SuccessBody:
		w.walkExpr(node.Value)
	case *syntax.FailureBody:
		if node.Error != nil {
			w.walkExpr(node.Error)
		}
	case *syntax.RelayBody:
		if node.Call != nil {
			w.walkExpr(node.Call)
		}
	case *syntax.DoBody:
		w.walkBlock(node.Block)
	case *syntax.MatchBody:
		w.walkMatch(&node.Match)
	}
}

func (w *refWalker) walkMatch(match *syntax.Match) {
	for _, value := range match.Values {
		w.walkExpr(value)
	}
	if match.Call != nil {
		w.walkExpr(match.Call)
	}
	w.push()
	for i := range match.Chain {
		entry := &match.Chain[i]
		if entry.Call != nil {
			w.walkExpr(entry.Call)
		}
		if entry.Binding != nil {
			w.bind(entry.Binding.Name)
		}
	}
	for i := range match.When {
		w.walkAssertion(&match.When[i])
	}
	for i := range match.Arms {
		w.walkArm(&match.Arms[i])
	}
	w.pop()
}

func (w *refWalker) walkArm(arm *syntax.MatchArm) {
	w.push()
	for _, pattern := range arm.Patterns {
		w.walkPattern(pattern)
	}
	if arm.Outcome != nil {
		if arm.Outcome.Error != nil {
			w.walkType(arm.Outcome.Error, compileresolve.ErrorUse)
		}
		if arm.Outcome.Binding != nil {
			w.bind(arm.Outcome.Binding.Name)
		}
		if arm.Outcome.Alias != nil {
			w.bind(*arm.Outcome.Alias)
		}
	}
	w.walkBody(arm.Body)
	w.pop()
}

func (w *refWalker) walkCoordination(coordination *syntax.Coordination) {
	for i := range coordination.Participants {
		participant := &coordination.Participants[i]
		if participant.Call != nil {
			w.walkExpr(participant.Call)
		}
		w.walkExpr(participant.Spread)
		for j := range participant.Arms {
			w.walkArm(&participant.Arms[j])
		}
	}
	for i := range coordination.Arms {
		w.walkArm(&coordination.Arms[i])
	}
}

func (w *refWalker) walkPattern(pattern syntax.PatternNode) {
	switch node := pattern.(type) {
	case *syntax.BindPattern:
		w.bind(node.Name)
	case *syntax.ConstructorPattern:
		for _, field := range node.Fields {
			w.walkPattern(field)
		}
		for _, typ := range node.Types {
			w.walkType(typ, compileresolve.TypeUse)
		}
		w.name(node.Name, compileresolve.ConstructorUse, node.Name.Span.Start)
	case *syntax.NamePattern:
		for _, typ := range node.Types {
			w.walkType(typ, compileresolve.TypeUse)
		}
	case *syntax.ArrayPattern:
		for _, element := range node.Elements {
			w.walkPattern(element)
		}
		if node.Rest != nil {
			w.bind(*node.Rest)
		}
	case *syntax.AlternativePattern:
		for _, alternative := range node.Alternatives {
			w.walkPattern(alternative)
		}
	}
}

func (w *refWalker) walkAssertion(assertion *syntax.Assertion) {
	w.walkExpr(assertion.Receiver)
	for i := range assertion.Arguments {
		w.walkArgument(&assertion.Arguments[i])
	}
	w.walkBody(assertion.Expected)
}

func (w *refWalker) walkArgument(argument *syntax.Argument) {
	w.walkExpr(argument.Value)
	if argument.Group != nil {
		for _, value := range argument.Group.Values {
			w.walkExpr(value)
		}
	}
}

func (w *refWalker) walkExpr(expr syntax.Expr) {
	switch node := expr.(type) {
	case *syntax.NameExpr:
		w.name(node.Name, compileresolve.ValueUse, node.Name.Span.Start)
	case *syntax.ConstructorExpr:
		for _, typ := range node.Types {
			w.walkType(typ, compileresolve.TypeUse)
		}
		for i := range node.Arguments {
			w.walkArgument(&node.Arguments[i])
		}
		w.name(node.Name, compileresolve.ConstructorUse, node.Name.Span.Start)
	case *syntax.CallExpr:
		w.walkCallee(node.Invocation.Callee)
		for _, typ := range node.Invocation.Types {
			w.walkType(typ, compileresolve.TypeUse)
		}
		for i := range node.Invocation.Arguments {
			w.walkArgument(&node.Invocation.Arguments[i])
		}
		for i := range node.Methods {
			method := &node.Methods[i]
			w.methodOn(node.Invocation.Callee, method.Name)
			for _, typ := range method.Types {
				w.walkType(typ, compileresolve.TypeUse)
			}
			for j := range method.Arguments {
				w.walkArgument(&method.Arguments[j])
			}
		}
	case *syntax.ReferenceExpr:
		w.walkReference(node)
	case *syntax.FieldExpr:
		w.field(node)
		w.walkExpr(node.Receiver)
	case *syntax.GroupExpr:
		w.walkExpr(node.Value)
	case *syntax.UnaryExpr:
		w.walkExpr(node.Operand)
	case *syntax.BinaryExpr:
		w.walkExpr(node.Left)
		w.walkExpr(node.Right)
	case *syntax.ComparisonExpr:
		for _, operand := range node.Operands {
			w.walkExpr(operand)
		}
	case *syntax.ArrayExpr:
		for i := range node.Elements {
			w.walkArgument(&node.Elements[i])
		}
	case *syntax.IndexExpr:
		w.walkExpr(node.Receiver)
		w.walkExpr(node.Index)
	case *syntax.SliceExpr:
		w.walkExpr(node.Receiver)
		w.walkExpr(node.Start)
		w.walkExpr(node.End)
	case *syntax.UpdateExpr:
		w.walkExpr(node.Receiver)
		for i := range node.Fields {
			w.walkExpr(node.Fields[i].Value)
		}
	case *syntax.MatchExpr:
		w.walkMatch(&node.Match)
	case *syntax.CoordinationExpr:
		w.walkCoordination(&node.Coordination)
	}
}

func (w *refWalker) walkCallee(callee syntax.Expr) {
	if name, ok := callee.(*syntax.NameExpr); ok {
		w.name(name.Name, compileresolve.CallUse, name.Name.Span.Start)
		return
	}
	w.walkExpr(callee)
}

// walkReference indexes a callable reference: the callee like a call, the
// explicit with pins as references to the callee's near parameters, and
// every pinned value in the creation scope.
func (w *refWalker) walkReference(node *syntax.ReferenceExpr) {
	w.walkCallee(node.Callee)
	for _, typ := range node.Types {
		w.walkType(typ, compileresolve.TypeUse)
	}
	for i := range node.Bindings {
		binding := &node.Bindings[i]
		w.withName(node.Callee, binding.Name)
		w.walkExpr(binding.Value)
	}
}

// withName resolves one explicit near-input pin to the callee parameter
// it names. Only a callee resolving to a checked function declaration
// with a near input of that name records; anything else — a local
// callable, an unknown name, a non-near input — stays unindexed so the
// checker owns the error and queries decline.
func (w *refWalker) withName(callee syntax.Expr, name syntax.Token) {
	named, ok := callee.(*syntax.NameExpr)
	if !ok {
		return
	}
	if w.lookup(named.Name.Name, named.Name.Span.Start) != nil && named.Name.Package == "" {
		return
	}
	symbol, err := w.file.Lookup(nil, named.Name, compileresolve.CallUse)
	if err != nil || symbol.Source == nil {
		return
	}
	declaration, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok {
		return
	}
	for i := range declaration.Inputs {
		input := &declaration.Inputs[i]
		if input.Near && input.Name.Text == name.Text {
			w.record(name.Span, localIDFor(symbol.Source.Path, input.Name.Span), false)
			return
		}
	}
}

// field indexes a receiver-dot-field token against the record behind an
// annotation-known receiver. Local receivers decline: their nominal type
// needs checking, and guessing would risk a wrong-scope reference.
func (w *refWalker) field(node *syntax.FieldExpr) {
	record, err := w.receiverRecord(node.Receiver)
	if err != nil {
		return
	}
	w.record(node.Field.Span, w.fieldID(symbolID(record), node.Field.Text), false)
}

// methodOn indexes a chained method name against the receiver record of
// the enclosing call's callee.
func (w *refWalker) methodOn(callee syntax.Expr, name syntax.Token) {
	record, err := w.receiverRecord(callee)
	if err != nil {
		return
	}
	method, err := w.file.Method(record, name.Text)
	if err != nil {
		return
	}
	w.record(name.Span, symbolID(method), false)
}

// receiverRecord mirrors go-to-definition's known-receiver set: module
// values with a local nominal annotation and direct constructor calls.
// Local receivers decline rather than guess.
func (w *refWalker) receiverRecord(receiver syntax.Expr) (*compileresolve.Symbol, error) {
	switch node := receiver.(type) {
	case *syntax.NameExpr:
		if w.lookup(node.Name.Name, node.Name.Span.Start) != nil && node.Name.Package == "" {
			return nil, fmt.Errorf("receiver is function-local")
		}
		symbol, err := w.file.Lookup(nil, node.Name, compileresolve.ValueUse)
		if err != nil {
			return nil, err
		}
		value, ok := symbol.Declaration.(*syntax.ValueDecl)
		if !ok {
			return nil, fmt.Errorf("not a module value")
		}
		named, ok := value.Binding.Type.(*syntax.NamedType)
		if !ok || named.Name.Package != "" {
			return nil, fmt.Errorf("receiver type is not a local nominal")
		}
		record, err := w.file.Lookup(nil, named.Name, compileresolve.TypeUse)
		if err != nil {
			return nil, err
		}
		if record.Kind != compileresolve.Record {
			return nil, fmt.Errorf("receiver type is not a record")
		}
		return record, nil
	case *syntax.ConstructorExpr:
		symbol, err := w.file.Lookup(nil, node.Name, compileresolve.ConstructorUse)
		if err != nil {
			return nil, err
		}
		if symbol.Kind != compileresolve.Record {
			return nil, fmt.Errorf("constructor is not a record")
		}
		return symbol, nil
	}
	return nil, fmt.Errorf("receiver type needs checking")
}

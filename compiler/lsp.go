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
	"regexp"
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
// environment, emits files, or mutates registries.
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
			respond(msg.ID, map[string]any{"capabilities": map[string]any{"textDocumentSync": 1, "definitionProvider": true, "documentFormattingProvider": true, "hoverProvider": true, "referencesProvider": true, "completionProvider": map[string]any{"triggerCharacters": []string{".", ":"}}, "renameProvider": true}})
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
		case "textDocument/completion":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid completion params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.completion(p.TextDocument.URI, p.Position.Line, p.Position.Character))
			}
		case "textDocument/rename":
			var p struct {
				TextDocument docID `json:"textDocument"`
				Position     struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
				NewName string `json:"newName"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				if msg.ID != nil {
					respondErr(msg.ID, -32602, "invalid rename params")
				}
				continue
			}
			if msg.ID != nil {
				respond(msg.ID, server.rename(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.NewName))
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

// shadowsLocal reports whether a name occurrence is shadowed by a
// function-local binder visible at its offset.
func (w *refWalker) shadowsLocal(name syntax.QualifiedName, offset int) bool {
	return name.Package == "" && w.lookup(name.Name, offset) != nil
}

// calleeFunction resolves a callable callee to its checked function
// declaration: a name that is not function-local, resolving through call
// usage to a project function. Anything else — a local callable, an
// unknown name, a catalogue or non-function symbol — declines so the
// checker owns the error.
func calleeFunction(file *compileresolve.File, isLocal func(syntax.QualifiedName, int) bool, callee syntax.Expr) (*syntax.FunctionDecl, *compileresolve.Symbol, bool) {
	named, ok := callee.(*syntax.NameExpr)
	if !ok {
		return nil, nil, false
	}
	if isLocal(named.Name, named.Name.Span.Start) {
		return nil, nil, false
	}
	symbol, err := file.Lookup(nil, named.Name, compileresolve.CallUse)
	if err != nil || symbol.Source == nil {
		return nil, nil, false
	}
	declaration, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok {
		return nil, nil, false
	}
	return declaration, symbol, true
}

// withName resolves one explicit near-input pin to the callee parameter
// it names. Only a callee resolving to a checked function declaration
// with a near input of that name records; anything else — a local
// callable, an unknown name, a non-near input — stays unindexed so the
// checker owns the error and queries decline.
func (w *refWalker) withName(callee syntax.Expr, name syntax.Token) {
	declaration, symbol, ok := calleeFunction(w.file, w.shadowsLocal, callee)
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
	return knownReceiverRecord(w.file, w.shadowsLocal, receiver)
}

// knownReceiverRecord resolves the record behind an annotation-known
// receiver: a module value carrying a local nominal annotation, or a
// direct record constructor call. Function-local receivers decline —
// their nominal type needs checking, and guessing would risk a
// wrong-scope member. References and completion share this set.
func knownReceiverRecord(file *compileresolve.File, isLocal func(syntax.QualifiedName, int) bool, receiver syntax.Expr) (*compileresolve.Symbol, error) {
	switch node := receiver.(type) {
	case *syntax.NameExpr:
		if isLocal(node.Name, node.Name.Span.Start) {
			return nil, fmt.Errorf("receiver is function-local")
		}
		symbol, err := file.Lookup(nil, node.Name, compileresolve.ValueUse)
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
		record, err := file.Lookup(nil, named.Name, compileresolve.TypeUse)
		if err != nil {
			return nil, err
		}
		if record.Kind != compileresolve.Record {
			return nil, fmt.Errorf("receiver type is not a record")
		}
		return record, nil
	case *syntax.ConstructorExpr:
		symbol, err := file.Lookup(nil, node.Name, compileresolve.ConstructorUse)
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

// canlc completion: scope-aware completion over an inert snapshot.
//
// The service walks the open file once, tracking function-local scopes
// with their spans plus the innermost cursor context (member, with-pin,
// callee, type, qualified package, declaration gaps), then offers the
// exact candidate set for that context: visible locals by binding
// identity and order, captured names from enclosing scopes, checked
// World symbols filtered by usage eligibility, and the selected-surface
// keywords valid there. Callable candidates carry their exact checked
// arity so caller repair shows signatures; near inputs surface both as
// value candidates and as with-pin names behind a resolved callee.
//
// Precision rules mirror the G02/G03 query boundaries: shadowed or
// same-spelled bindings in other scopes are never offered, unselected
// grammars contribute no keywords (A06 is inactive, so C-A adds none),
// member candidates resolve only behind annotation-known receivers,
// with-pins only behind a checked function callee, and anything
// unresolvable — an unloadable snapshot, an unknown receiver or
// callee — yields an empty list or a declined null rather than a
// guess. The walk reads the snapshot and returns freshly built items,
// so no mutable compiler state escapes.

// LSP CompletionItemKind numbers for the candidates below.
const (
	compText          = 1
	compMethod        = 2
	compFunction      = 3
	compField         = 5
	compVariable      = 6
	compClass         = 7
	compModule        = 9
	compProperty      = 10
	compKeyword       = 14
	compConstant      = 21
	compStruct        = 22
	compTypeParameter = 25
)

// compItem is one completion candidate: the insertable label, its LSP
// kind, a signature or type detail, and a provenance note. Details only
// ever state checked facts — exact arity, declared types, near marks —
// and stay empty where the source carries none.
type compItem struct {
	label string
	kind  int
	tail  string
	doc   string
}

func sortCompItems(items []compItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].label != items[j].label {
			return items[i].label < items[j].label
		}
		if items[i].kind != items[j].kind {
			return items[i].kind < items[j].kind
		}
		if items[i].tail != items[j].tail {
			return items[i].tail < items[j].tail
		}
		return items[i].doc < items[j].doc
	})
}

// Keyword sets per cursor context, drawn only from the selected
// surface. Unselected grammars (Q4/Q5/Q6) contribute nothing: A06 is
// inactive, so C-A adds no iteration keywords.
var (
	compGeneralKeywords = []string{"and", "call", "callable", "do", "false", "is", "match", "not", "ok", "or", "relay", "true"}
	compTypeKeywords    = []string{"bool", "float", "int", "str", "void"}
	compTopKeywords     = []string{"bool", "error", "float", "fn", "int", "record", "str", "variant", "void"}
	compHeaderKeywords  = []string{"as", "package", "provides", "uses"}
	compSignKeywords    = []string{"asserts", "emits", "given", "near"}
	compWithKeyword     = []string{"with"}
)

func keywordItems(words []string) []compItem {
	items := make([]compItem, 0, len(words))
	for _, word := range words {
		items = append(items, compItem{label: word, kind: compKeyword, doc: "keyword"})
	}
	return items
}

// ------------------------------------------------------------ walker ---

// compBinder is one function-local binding: inputs, receiver, steps,
// chain and pattern bindings with their declared type and near mark.
// from is the offset the binding becomes visible at: the name start
// for parameters and patterns, the step end for step and chain
// bindings, whose initializers never see the name they define.
type compBinder struct {
	name string
	span source.Span
	from int
	typ  syntax.TypeNode
	near bool
	fn   string
}

// compScope is one lexical frame with the source span it covers and
// its nesting depth.
type compScope struct {
	span    source.Span
	depth   int
	binders []*compBinder
}

type compWithPin struct {
	span   source.Span
	callee syntax.Expr
}

type compCallee struct {
	span      source.Span
	reference bool
}

type compMember struct {
	span     source.Span
	receiver syntax.Expr
	method   bool
}

type compQualified struct {
	span  source.Span
	name  syntax.QualifiedName
	usage compileresolve.Usage
}

type compRefExpr struct {
	span      source.Span
	calleeEnd int
	firstBind int
}

type compFunc struct {
	decl    source.Span
	body    source.Span
	asserts []source.Span
}

// compWalker mirrors the G03 reference walk for one file, recording
// scope spans, binder metadata, and the context markers behind the
// cursor classification. Traversal order matches refWalker so binding
// identity and visibility stay identical.
type compWalker struct {
	text string
	file *compileresolve.File

	scopes []*compScope
	frames []*compScope
	curFn  string

	types     []source.Span
	bounds    []source.Span
	withPins  []compWithPin
	callees   []compCallee
	members   []compMember
	qualified []compQualified
	refExprs  []compRefExpr
	ctors     []source.Span
	decls     []source.Span
	literals  []source.Span
	funcs     []compFunc
	records   []source.Span
	opaque    []source.Span
	declSpans []source.Span
	leads     []source.Span
	header    source.Span
}

func (w *compWalker) push(span source.Span) {
	frame := &compScope{span: span, depth: len(w.scopes)}
	w.scopes = append(w.scopes, frame)
	// Frames outlive the walk: queries resolve visibility from the
	// retained spans, since push and pop mirror AST nesting exactly.
	w.frames = append(w.frames, frame)
}

func (w *compWalker) pop() {
	w.scopes = w.scopes[:len(w.scopes)-1]
}

func (w *compWalker) bind(name syntax.Token, typ syntax.TypeNode, near bool, from int) {
	w.scopes[len(w.scopes)-1].binders = append(w.scopes[len(w.scopes)-1].binders, &compBinder{name: name.Text, span: name.Span, from: from, typ: typ, near: near, fn: w.curFn})
	w.decls = append(w.decls, name.Span)
}

// framesAt returns the frames enclosing the offset, innermost first.
func (w *compWalker) framesAt(offset int) []*compScope {
	var out []*compScope
	for _, frame := range w.frames {
		if coversEnd(frame.span, offset) {
			out = append(out, frame)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].depth > out[j].depth })
	return out
}

func (w *compWalker) lookup(name string, offset int) *compBinder {
	for _, frame := range w.framesAt(offset) {
		var best *compBinder
		for _, binder := range frame.binders {
			if binder.name != name || binder.from >= offset {
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

func (w *compWalker) shadowsLocal(name syntax.QualifiedName, offset int) bool {
	return name.Package == "" && w.lookup(name.Name, offset) != nil
}

// occurrence records one name token with the usage its position
// carries. Only qualified spellings feed the query; unqualified names
// resolve through scope visibility and the World instead.
func (w *compWalker) occurrence(name syntax.QualifiedName, usage compileresolve.Usage) {
	w.qualified = append(w.qualified, compQualified{span: name.Span, name: name, usage: usage})
}

func (w *compWalker) walkFile(file *syntax.File) {
	w.header = file.Header.Span
	for _, declaration := range file.Declarations {
		w.declSpans = append(w.declSpans, declaration.DeclSpan())
		w.walkDecl(declaration)
	}
}

func (w *compWalker) walkDecl(declaration syntax.Declaration) {
	switch node := declaration.(type) {
	case *syntax.FunctionDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.walkType(node.Result)
		if node.Receiver != nil {
			w.walkType(node.Receiver.Type)
		}
		w.walkBound(node.Errors)
		for i := range node.Inputs {
			w.walkType(node.Inputs[i].Type)
		}
		fn := compFunc{decl: node.DeclSpan(), body: node.Body.Span}
		for i := range node.Assertions {
			fn.asserts = append(fn.asserts, node.Assertions[i].Span)
		}
		w.funcs = append(w.funcs, fn)
		outer := w.curFn
		w.curFn = node.Name.Text
		w.push(node.DeclSpan())
		if node.Receiver != nil {
			w.bind(node.Receiver.Name, node.Receiver.Type, false, node.Receiver.Name.Span.Start)
		}
		for i := range node.Inputs {
			w.bind(node.Inputs[i].Name, node.Inputs[i].Type, node.Inputs[i].Near, node.Inputs[i].Name.Span.Start)
		}
		w.walkBlock(node.Body)
		for i := range node.Assertions {
			w.walkAssertion(&node.Assertions[i])
		}
		w.pop()
		w.curFn = outer
	case *syntax.RecordDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for i := range node.Fields {
			w.decls = append(w.decls, node.Fields[i].Name.Span)
			w.walkType(node.Fields[i].Type)
		}
	case *syntax.VariantDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for _, alternative := range node.Alternatives {
			w.walkType(alternative)
		}
	case *syntax.ErrorDecl:
		w.decls = append(w.decls, node.Name.Span)
		w.leads = append(w.leads, source.Span{Start: node.DeclSpan().Start, End: node.Name.Span.End})
		for _, parameter := range node.Parameters {
			w.decls = append(w.decls, parameter.Span)
		}
		w.records = append(w.records, node.DeclSpan())
		for i := range node.Fields {
			w.decls = append(w.decls, node.Fields[i].Name.Span)
			w.walkType(node.Fields[i].Type)
		}
	case *syntax.ValueDecl:
		w.decls = append(w.decls, node.Binding.Name.Span)
		w.walkType(node.Binding.Type)
		w.walkExpr(node.Binding.Value)
	case *syntax.QuestionDecl:
		if node.RecordName != nil {
			w.decls = append(w.decls, node.RecordName.Span)
		}
		w.decls = append(w.decls, node.Name.Span)
		for _, option := range node.Options {
			if option.Name != nil {
				w.decls = append(w.decls, option.Name.Span)
			}
			w.walkExpr(option.Description)
			w.walkExpr(option.Spread)
			w.walkBody(option.Body)
		}
		w.walkExpr(node.Asks)
		w.walkExpr(node.Minimum)
		w.walkBody(node.Fallback)
		w.walkBody(node.Shared)
	default:
		w.opaque = append(w.opaque, node.DeclSpan())
	}
}

func (w *compWalker) walkType(node syntax.TypeNode) {
	w.walkTypeAt(node, compileresolve.TypeUse)
}

// walkTypeAt records one annotation with the usage its name position
// carries: error usage inside emits bounds, type usage elsewhere. Only
// qualified spellings consult the usage; the markers stay positional.
func (w *compWalker) walkTypeAt(node syntax.TypeNode, usage compileresolve.Usage) {
	if node == nil {
		return
	}
	w.types = append(w.types, node.TypeSpan())
	switch node := node.(type) {
	case *syntax.NamedType:
		for _, argument := range node.Arguments {
			w.walkType(argument)
		}
		w.occurrence(node.Name, usage)
	case *syntax.ArrayType:
		w.walkTypeAt(node.Element, usage)
	case *syntax.CallableType:
		w.walkType(node.Result)
		for _, input := range node.Inputs {
			w.walkType(input)
		}
		w.walkBound(node.Errors)
	case *syntax.ChoiceArmType:
		w.walkType(node.Result)
		w.walkBound(node.Errors)
	}
}

func (w *compWalker) walkBound(bound syntax.ErrorBound) {
	w.bounds = append(w.bounds, bound.Span)
	for _, typ := range bound.Types {
		w.walkTypeAt(typ, compileresolve.ErrorUse)
	}
}

func (w *compWalker) walkBlock(block syntax.Block) {
	w.push(block.Span)
	for _, step := range block.Steps {
		w.walkStep(step)
	}
	w.walkBody(block.Terminal)
	w.pop()
}

func (w *compWalker) walkStep(step syntax.Step) {
	switch node := step.(type) {
	case *syntax.BindingStep:
		w.walkType(node.Binding.Type)
		// The value walks before its own name binds: an initializer
		// never sees the binding it defines.
		w.walkExpr(node.Binding.Value)
		w.bind(node.Binding.Name, node.Binding.Type, false, node.Binding.Span.End)
	case *syntax.CallStep:
		w.walkExpr(node.Call)
	case *syntax.CoordinationStep:
		w.walkCoordination(&node.Coordination)
	}
}

func (w *compWalker) walkBody(body syntax.Body) {
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

func (w *compWalker) walkMatch(match *syntax.Match) {
	for _, value := range match.Values {
		w.walkExpr(value)
	}
	if match.Call != nil {
		w.walkExpr(match.Call)
	}
	w.push(match.Span)
	for i := range match.Chain {
		entry := &match.Chain[i]
		if entry.Call != nil {
			w.walkExpr(entry.Call)
		}
		if entry.Binding != nil {
			w.bind(entry.Binding.Name, entry.Binding.Type, false, entry.Span.End)
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

func (w *compWalker) walkArm(arm *syntax.MatchArm) {
	w.push(arm.Span)
	for _, pattern := range arm.Patterns {
		w.walkPattern(pattern)
	}
	if arm.Outcome != nil {
		if arm.Outcome.Error != nil {
			w.walkTypeAt(arm.Outcome.Error, compileresolve.ErrorUse)
		}
		if arm.Outcome.Binding != nil {
			w.bind(arm.Outcome.Binding.Name, arm.Outcome.Binding.Type, false, arm.Outcome.Binding.Name.Span.Start)
		}
		if arm.Outcome.Alias != nil {
			w.bind(*arm.Outcome.Alias, nil, false, arm.Outcome.Alias.Span.Start)
		}
	}
	w.walkBody(arm.Body)
	w.pop()
}

func (w *compWalker) walkCoordination(coordination *syntax.Coordination) {
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

func (w *compWalker) walkPattern(pattern syntax.PatternNode) {
	switch node := pattern.(type) {
	case *syntax.BindPattern:
		w.bind(node.Name, nil, false, node.Name.Span.Start)
	case *syntax.ConstructorPattern:
		for _, field := range node.Fields {
			w.walkPattern(field)
		}
		for _, typ := range node.Types {
			w.walkType(typ)
		}
		w.ctors = append(w.ctors, node.Name.Span)
		w.occurrence(node.Name, compileresolve.ConstructorUse)
	case *syntax.NamePattern:
		// Bare pattern names test nominal leaves, never values:
		// completing one offers type candidates.
		w.types = append(w.types, node.Name.Span)
		w.occurrence(node.Name, compileresolve.TypeUse)
		for _, typ := range node.Types {
			w.walkType(typ)
		}
	case *syntax.ArrayPattern:
		for _, element := range node.Elements {
			w.walkPattern(element)
		}
		if node.Rest != nil {
			w.bind(*node.Rest, nil, false, node.Rest.Span.Start)
		}
	case *syntax.AlternativePattern:
		for _, alternative := range node.Alternatives {
			w.walkPattern(alternative)
		}
	}
}

func (w *compWalker) walkAssertion(assertion *syntax.Assertion) {
	w.decls = append(w.decls, assertion.Name.Span)
	if assertion.Scenario != nil {
		w.decls = append(w.decls, assertion.Scenario.Span)
	}
	if assertion.Use != nil {
		w.decls = append(w.decls, assertion.Use.Template.Span)
		for i := range assertion.Use.Arguments {
			w.walkArgument(&assertion.Use.Arguments[i])
		}
	}
	for _, link := range assertion.Links {
		w.decls = append(w.decls, link.Span)
	}
	w.walkExpr(assertion.Receiver)
	for i := range assertion.Arguments {
		w.walkArgument(&assertion.Arguments[i])
	}
	w.walkBody(assertion.Expected)
}

func (w *compWalker) walkArgument(argument *syntax.Argument) {
	w.walkExpr(argument.Value)
	if argument.Group != nil {
		for _, value := range argument.Group.Values {
			w.walkExpr(value)
		}
	}
}

func (w *compWalker) walkExpr(expr syntax.Expr) {
	switch node := expr.(type) {
	case *syntax.NameExpr:
		w.occurrence(node.Name, compileresolve.ValueUse)
	case *syntax.LiteralExpr:
		// Only data literals decline: keyword literals (true/false)
		// stay completable through the general keyword set.
		switch node.Token.Kind {
		case syntax.String, syntax.Integer, syntax.Float:
			w.literals = append(w.literals, node.Token.Span)
		}
	case *syntax.ConstructorExpr:
		for _, typ := range node.Types {
			w.walkType(typ)
		}
		for i := range node.Arguments {
			w.walkArgument(&node.Arguments[i])
		}
		w.ctors = append(w.ctors, node.Name.Span)
		w.occurrence(node.Name, compileresolve.ConstructorUse)
	case *syntax.CallExpr:
		w.walkCallee(node.Invocation.Callee, false)
		for _, typ := range node.Invocation.Types {
			w.walkType(typ)
		}
		for i := range node.Invocation.Arguments {
			w.walkArgument(&node.Invocation.Arguments[i])
		}
		for i := range node.Methods {
			method := &node.Methods[i]
			w.members = append(w.members, compMember{span: method.Name.Span, receiver: node.Invocation.Callee, method: true})
			for _, typ := range method.Types {
				w.walkType(typ)
			}
			for j := range method.Arguments {
				w.walkArgument(&method.Arguments[j])
			}
		}
	case *syntax.ReferenceExpr:
		w.walkReference(node)
	case *syntax.FieldExpr:
		w.members = append(w.members, compMember{span: node.Field.Span, receiver: node.Receiver})
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
		for i := range node.Fields {
			w.members = append(w.members, compMember{span: node.Fields[i].Name.Span, receiver: node.Receiver})
			w.walkExpr(node.Fields[i].Value)
		}
		w.walkExpr(node.Receiver)
	case *syntax.MatchExpr:
		w.walkMatch(&node.Match)
	case *syntax.CoordinationExpr:
		w.walkCoordination(&node.Coordination)
	}
}

// walkCallee records a bare-name callee with the usage its call form
// carries: reference usage for callable references, call usage for
// calls. Complex callees walk as ordinary expressions.
func (w *compWalker) walkCallee(callee syntax.Expr, reference bool) {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		w.walkExpr(callee)
		return
	}
	usage := compileresolve.CallUse
	if reference {
		usage = compileresolve.ReferenceUse
	}
	w.callees = append(w.callees, compCallee{span: name.Name.Span, reference: reference})
	w.occurrence(name.Name, usage)
}

func (w *compWalker) walkReference(node *syntax.ReferenceExpr) {
	marker := compRefExpr{span: node.ExprSpan(), firstBind: -1}
	if callee, ok := node.Callee.(*syntax.NameExpr); ok {
		marker.calleeEnd = callee.Name.Span.End
	} else {
		marker.calleeEnd = node.Callee.ExprSpan().End
	}
	if len(node.Bindings) > 0 {
		marker.firstBind = node.Bindings[0].Span.Start
	}
	w.refExprs = append(w.refExprs, marker)
	w.walkCallee(node.Callee, true)
	for _, typ := range node.Types {
		w.walkType(typ)
	}
	for i := range node.Bindings {
		binding := &node.Bindings[i]
		w.withPins = append(w.withPins, compWithPin{span: binding.Name.Span, callee: node.Callee})
		w.walkExpr(binding.Value)
	}
}

// ------------------------------------------------------ classification ---

type compCtxKind int

const (
	ctxEmpty compCtxKind = iota
	ctxQualified
	ctxType
	ctxErrorType
	ctxWithPin
	ctxMember
	ctxCallee
	ctxCtor
	ctxWithKw
	ctxHeader
	ctxSignature
	ctxTopLevel
	ctxGeneral
)

// compContext is the classified cursor: the context kind plus the
// payload its candidates need.
type compContext struct {
	kind      compCtxKind
	usage     compileresolve.Usage
	pkg       string
	pkgPos    bool
	reference bool
	callee    syntax.Expr
	receiver  syntax.Expr
	method    bool
}

// coversEnd reports whether the offset sits inside the span or exactly
// at its end, where completion cursors rest after a partial token.
func coversEnd(span source.Span, offset int) bool {
	return span.Start <= offset && offset <= span.End
}

func coversAny(spans []source.Span, offset int) bool {
	for _, span := range spans {
		if coversEnd(span, offset) {
			return true
		}
	}
	return false
}

// sameLineGap reports whether the source between two offsets holds
// only same-line whitespace.
func (w *compWalker) sameLineGap(from, to int) bool {
	if from < 0 || to > len(w.text) || from > to {
		return false
	}
	for i := from; i < to; i++ {
		if w.text[i] != ' ' && w.text[i] != '\t' {
			return false
		}
	}
	return true
}

// classify resolves the cursor to its innermost context. Declaration
// sites, literals, comments, and uncovered native declarations yield
// no candidates; strict name positions (qualified, type, with-pin,
// member, callee, constructor) take precedence over the looser
// declaration gaps and the general body context.
func (w *compWalker) classify(file *syntax.File, offset int) compContext {
	for _, comment := range file.Comments {
		if coversEnd(comment.Span, offset) {
			return compContext{kind: ctxEmpty}
		}
	}
	if coversAny(w.decls, offset) || coversAny(w.literals, offset) || coversAny(w.opaque, offset) {
		return compContext{kind: ctxEmpty}
	}
	if found, marker := smallestQualified(w.qualified, offset); found {
		return w.qualifiedContext(marker, offset)
	}
	// Bounds precede types: positions inside an emits clause —
	// members or gaps — take error candidates, never the general
	// type set.
	if coversAny(w.bounds, offset) {
		return compContext{kind: ctxErrorType}
	}
	if coversAny(w.types, offset) {
		return compContext{kind: ctxType}
	}
	for _, pin := range w.withPins {
		if coversEnd(pin.span, offset) {
			return compContext{kind: ctxWithPin, callee: pin.callee}
		}
	}
	for _, member := range w.members {
		if coversEnd(member.span, offset) {
			return compContext{kind: ctxMember, receiver: member.receiver, method: member.method}
		}
	}
	for _, callee := range w.callees {
		if coversEnd(callee.span, offset) {
			return compContext{kind: ctxCallee, reference: callee.reference}
		}
	}
	if coversAny(w.ctors, offset) {
		return compContext{kind: ctxCtor}
	}
	for _, ref := range w.refExprs {
		if coversEnd(ref.span, offset) && offset > ref.calleeEnd && (ref.firstBind < 0 || offset < ref.firstBind) {
			return compContext{kind: ctxWithKw}
		}
		// A bare callee's span ends at its name, but `with` still
		// follows across same-line whitespace.
		if ref.firstBind < 0 && offset > ref.calleeEnd && w.sameLineGap(ref.span.End, offset) {
			return compContext{kind: ctxWithKw}
		}
	}
	if coversEnd(w.header, offset) {
		return compContext{kind: ctxHeader}
	}
	// Declaration leads (the starter keywords before a declared
	// name) complete like top level: the name itself already declined
	// above, and annotation positions resolved earlier.
	if coversAny(w.leads, offset) {
		return compContext{kind: ctxTopLevel}
	}
	for _, fn := range w.funcs {
		if !coversEnd(fn.decl, offset) {
			continue
		}
		if coversEnd(fn.body, offset) {
			return compContext{kind: ctxGeneral}
		}
		for _, assert := range fn.asserts {
			if coversEnd(assert, offset) {
				return compContext{kind: ctxGeneral}
			}
		}
		return compContext{kind: ctxSignature}
	}
	if coversAny(w.records, offset) {
		return compContext{kind: ctxType}
	}
	for _, span := range w.declSpans {
		if coversEnd(span, offset) {
			return compContext{kind: ctxGeneral}
		}
	}
	return compContext{kind: ctxTopLevel}
}

func smallestQualified(markers []compQualified, offset int) (bool, compQualified) {
	var best compQualified
	found := false
	for _, marker := range markers {
		if marker.name.Package == "" || !coversEnd(marker.span, offset) {
			continue
		}
		if !found || marker.span.End-marker.span.Start < best.span.End-best.span.Start {
			best, found = marker, true
		}
	}
	return found, best
}

// qualifiedContext splits a qualified occurrence at its `::`: the
// package part completes visible import aliases, the name part the
// imported package's members under the position's usage.
func (w *compWalker) qualifiedContext(marker compQualified, offset int) compContext {
	pkgEnd := marker.span.Start + len(marker.name.Package)
	if pkgEnd+1 >= len(w.text) || w.text[pkgEnd:pkgEnd+2] != "::" {
		return compContext{kind: ctxEmpty}
	}
	if offset <= pkgEnd {
		return compContext{kind: ctxQualified, pkgPos: true}
	}
	return compContext{kind: ctxQualified, pkg: marker.name.Package, usage: marker.usage}
}

// -------------------------------------------------------- candidates ---

// localsAt collects the binders visible at the offset: frames inside
// out, nearest binder starting before the use within a frame. Each
// name surfaces once, so shadowed bindings never leak beside the
// visible one and later same-spelled bindings never capture the use.
func (w *compWalker) localsAt(offset int) []compItem {
	seen := map[string]bool{}
	var items []compItem
	for _, frame := range w.framesAt(offset) {
		byName := map[string]*compBinder{}
		for _, binder := range frame.binders {
			if binder.from >= offset {
				continue
			}
			byName[binder.name] = binder
		}
		for name, binder := range byName {
			if seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, localItem(binder))
		}
	}
	return items
}

func localNames(locals []compItem) map[string]bool {
	names := map[string]bool{}
	for _, local := range locals {
		names[local.label] = true
	}
	return names
}

func localItem(binder *compBinder) compItem {
	typ := ""
	if binder.typ != nil {
		typ = syntax.FormatType(binder.typ)
	}
	tail, doc := typ, "local binding"
	if binder.near {
		tail = "near " + typ
		doc = "near input"
		if binder.fn != "" {
			doc = "near input of " + binder.fn
		}
	}
	return compItem{label: binder.name, kind: compVariable, tail: tail, doc: doc}
}

// visibleSymbols walks the file scope chain innermost out, keeping the
// first binding per name and applying the context's eligibility
// filter. Results are freshly collected per request.
func visibleSymbols(file *compileresolve.File, accept func(*compileresolve.Symbol) bool) []*compileresolve.Symbol {
	seen := map[string]bool{}
	var out []*compileresolve.Symbol
	for scope := file.Scope; scope != nil; scope = scope.Parent {
		for name, symbol := range scope.Symbols {
			if seen[name] {
				continue
			}
			seen[name] = true
			if accept == nil || accept(symbol) {
				out = append(out, symbol)
			}
		}
	}
	return out
}

// typeSymbolAccept keeps type-eligible symbols except the primitives
// the type keywords already spell: one candidate per spelling.
func typeSymbolAccept(symbol *compileresolve.Symbol) bool {
	if !symbol.Eligible(compileresolve.TypeUse) {
		return false
	}
	switch symbol.Name {
	case "int", "float", "bool", "str", "void":
		return false
	}
	return true
}

func packageProvenance(symbol *compileresolve.Symbol) string {
	if symbol == nil || symbol.Package == nil || symbol.Package.Source == nil {
		return "catalogue"
	}
	return "package " + symbol.Package.Name
}

// symbolItem renders one checked symbol. Callable declarations carry
// their exact arity; catalogue operations without a declaration state
// only kind and provenance rather than a guessed signature.
func symbolItem(symbol *compileresolve.Symbol) compItem {
	doc := packageProvenance(symbol)
	switch symbol.Kind {
	case compileresolve.Function, compileresolve.Fetch, compileresolve.Judge, compileresolve.LLM, compileresolve.Wrapper:
		kind := compFunction
		if symbol.Receiver != nil {
			kind = compMethod
		}
		return compItem{label: symbol.Name, kind: kind, tail: callableTail(symbol), doc: doc}
	case compileresolve.Value, compileresolve.ChoiceArm:
		return compItem{label: symbol.Name, kind: compConstant, tail: valueTail(symbol), doc: doc}
	case compileresolve.Record, compileresolve.Variant, compileresolve.Error, compileresolve.Opaque, compileresolve.Primitive:
		return compItem{label: symbol.Name, kind: compClass, tail: string(symbol.Kind), doc: doc}
	case compileresolve.TypeParameter:
		return compItem{label: symbol.Name, kind: compTypeParameter, doc: doc}
	default:
		return compItem{label: symbol.Name, kind: compStruct, tail: string(symbol.Kind), doc: doc}
	}
}

// callableTail renders the exact checked arity of a callable
// declaration: named inputs with near and variadic marks plus the
// result type. Anything without a signature-shaped declaration falls
// back to its kind label instead of a guessed arity.
func callableTail(symbol *compileresolve.Symbol) string {
	switch declaration := symbol.Declaration.(type) {
	case *syntax.FunctionDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.FetchDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.LLMDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	case *syntax.JudgeDecl:
		if tail, ok := functionArity(declaration.Result, declaration.Inputs); ok {
			return tail
		}
	}
	return string(symbol.Kind)
}

func functionArity(result syntax.TypeNode, inputs []syntax.Input) (string, bool) {
	if result == nil {
		return "", false
	}
	params := make([]string, len(inputs))
	for i := range inputs {
		if inputs[i].Type == nil {
			return "", false
		}
		param := ""
		if inputs[i].Near {
			param = "near "
		}
		param += inputs[i].Name.Text + ": " + syntax.FormatType(inputs[i].Type)
		if inputs[i].Variadic {
			param += "..."
		}
		params[i] = param
	}
	joined := ""
	for i, param := range params {
		if i > 0 {
			joined += ", "
		}
		joined += param
	}
	return "(" + joined + ") -> " + syntax.FormatType(result), true
}

// valueTail renders a module value's declared type, or the callable
// arity for callable-typed values.
func valueTail(symbol *compileresolve.Symbol) string {
	if symbol.Type == nil {
		return ""
	}
	if callable, ok := symbol.Type.(*syntax.CallableType); ok {
		inputs := make([]string, len(callable.Inputs))
		for i, input := range callable.Inputs {
			if input == nil {
				return syntax.FormatType(symbol.Type)
			}
			inputs[i] = syntax.FormatType(input)
		}
		if callable.Result == nil {
			return syntax.FormatType(symbol.Type)
		}
		joined := ""
		for i, input := range inputs {
			if i > 0 {
				joined += ", "
			}
			joined += input
		}
		return "(" + joined + ") -> " + syntax.FormatType(callable.Result)
	}
	return syntax.FormatType(symbol.Type)
}

// candidates builds the exact candidate set for a classified context.
func (w *compWalker) candidates(context compContext, offset int) []compItem {
	var items []compItem
	switch context.kind {
	case ctxEmpty:
		return []compItem{}
	case ctxQualified:
		if context.pkgPos {
			return packageAliasItems(w.file)
		}
		return w.packageMemberItems(context.pkg, context.usage)
	case ctxType:
		for _, symbol := range visibleSymbols(w.file, typeSymbolAccept) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compTypeKeywords)...)
	case ctxErrorType:
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(compileresolve.ErrorUse) }) {
			items = append(items, symbolItem(symbol))
		}
	case ctxWithPin:
		return w.withPinItems(context.callee)
	case ctxMember:
		return w.memberItems(context.receiver, context.method)
	case ctxCallee:
		usage := compileresolve.CallUse
		if context.reference {
			usage = compileresolve.ReferenceUse
		}
		shadowed := localNames(w.localsAt(offset))
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(usage) && !shadowed[symbol.Name] }) {
			items = append(items, symbolItem(symbol))
		}
		if !context.reference {
			items = append(items, w.callableLocals(offset)...)
		}
	case ctxCtor:
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return symbol.Eligible(compileresolve.ConstructorUse) }) {
			items = append(items, symbolItem(symbol))
		}
	case ctxWithKw:
		return keywordItems(compWithKeyword)
	case ctxHeader:
		items = append(items, keywordItems(compHeaderKeywords)...)
		items = append(items, w.fileDeclItems()...)
		items = append(items, packageAliasItems(w.file)...)
	case ctxSignature:
		return keywordItems(compSignKeywords)
	case ctxTopLevel:
		for _, symbol := range visibleSymbols(w.file, typeSymbolAccept) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compTopKeywords)...)
	case ctxGeneral:
		locals := w.localsAt(offset)
		items = append(items, locals...)
		// A visible local shadows the same-spelled symbol for
		// unqualified uses, so the unreachable symbol stays out.
		shadowed := localNames(locals)
		for _, symbol := range visibleSymbols(w.file, func(symbol *compileresolve.Symbol) bool { return !shadowed[symbol.Name] }) {
			items = append(items, symbolItem(symbol))
		}
		items = append(items, keywordItems(compGeneralKeywords)...)
	}
	sortCompItems(items)
	return items
}

// callableLocals offers function-local bindings with an explicit
// callable annotation as call candidates. The annotation is
// source-evident, never inferred.
func (w *compWalker) callableLocals(offset int) []compItem {
	var items []compItem
	for _, item := range w.localsAt(offset) {
		binder := w.lookup(item.label, offset)
		if binder == nil || binder.typ == nil {
			continue
		}
		callable, ok := binder.typ.(*syntax.CallableType)
		if !ok || callable.Result == nil {
			continue
		}
		inputs := make([]string, len(callable.Inputs))
		complete := true
		for i, input := range callable.Inputs {
			if input == nil {
				complete = false
				break
			}
			inputs[i] = syntax.FormatType(input)
		}
		if !complete {
			continue
		}
		joined := ""
		for i, input := range inputs {
			if i > 0 {
				joined += ", "
			}
			joined += input
		}
		item.tail = "(" + joined + ") -> " + syntax.FormatType(callable.Result)
		items = append(items, item)
	}
	return items
}

// withPinItems offers the resolved callee's near parameter names, one
// identity per parameter. An unresolvable callee yields nothing: the
// checker owns the error. A06 is inactive, so no iteration surface
// expands this set.
func (w *compWalker) withPinItems(callee syntax.Expr) []compItem {
	declaration, symbol, ok := calleeFunction(w.file, w.shadowsLocal, callee)
	if !ok {
		return []compItem{}
	}
	var items []compItem
	for i := range declaration.Inputs {
		input := &declaration.Inputs[i]
		if !input.Near {
			continue
		}
		tail := "near"
		if input.Type != nil {
			tail = "near " + syntax.FormatType(input.Type)
		}
		items = append(items, compItem{label: input.Name.Text, kind: compProperty, tail: tail, doc: "near parameter of " + symbol.Name})
	}
	sortCompItems(items)
	return items
}

// memberItems offers the fields or methods behind an annotation-known
// receiver, sharing the G03 known-receiver set. Unknown receivers —
// locals, unresolvable names, non-nominal annotations — yield nothing
// rather than a guessed member.
func (w *compWalker) memberItems(receiver syntax.Expr, method bool) []compItem {
	record, err := knownReceiverRecord(w.file, w.shadowsLocal, receiver)
	if err != nil {
		return []compItem{}
	}
	decl, ok := record.Declaration.(*syntax.RecordDecl)
	if !ok {
		return []compItem{}
	}
	if method {
		return w.methodItems(record)
	}
	if record.Owner && record.Package != w.file.Package {
		return []compItem{}
	}
	var items []compItem
	for i := range decl.Fields {
		tail := ""
		if decl.Fields[i].Type != nil {
			tail = syntax.FormatType(decl.Fields[i].Type)
		}
		items = append(items, compItem{label: decl.Fields[i].Name.Text, kind: compField, tail: tail, doc: "field of " + record.Name})
	}
	sortCompItems(items)
	return items
}

// methodItems enumerates the package methods owned by a record,
// applying the same export rules as go-to-definition.
func (w *compWalker) methodItems(record *compileresolve.Symbol) []compItem {
	if record.Package == nil || record.Package.Source == nil {
		return []compItem{}
	}
	imported := record.Package == w.file.Package
	if !imported {
		for _, pkg := range w.file.Imports {
			if pkg == record.Package {
				imported = true
			}
		}
	}
	var items []compItem
	for _, symbol := range record.Package.Scope.Symbols {
		if symbol.Receiver != record {
			continue
		}
		if !imported || (!symbol.Public && record.Package != w.file.Package) {
			continue
		}
		items = append(items, compItem{label: symbol.Name, kind: compMethod, tail: callableTail(symbol), doc: "method of " + record.Name})
	}
	sortCompItems(items)
	return items
}

// packageMemberItems offers one imported package's members under the
// position's usage, with the privacy and owner rules of qualified
// lookup. Unimported packages yield nothing.
func (w *compWalker) packageMemberItems(alias string, usage compileresolve.Usage) []compItem {
	pkg := w.file.Imports[alias]
	if pkg == nil {
		return []compItem{}
	}
	var items []compItem
	for _, symbol := range pkg.Scope.Symbols {
		if !symbol.Eligible(usage) {
			continue
		}
		if pkg != w.file.Package && !symbol.Public {
			continue
		}
		if usage == compileresolve.ConstructorUse && symbol.Owner && pkg != w.file.Package {
			continue
		}
		items = append(items, symbolItem(symbol))
	}
	sortCompItems(items)
	return items
}

func packageAliasItems(file *compileresolve.File) []compItem {
	var items []compItem
	for alias, pkg := range file.Imports {
		doc := "package " + pkg.Name
		if pkg.Source == nil {
			doc = "catalogue package " + pkg.Name
		}
		items = append(items, compItem{label: alias, kind: compModule, doc: doc})
	}
	sortCompItems(items)
	return items
}

// fileDeclItems offers the names this file declares, for header
// provides positions.
func (w *compWalker) fileDeclItems() []compItem {
	var items []compItem
	for _, symbol := range w.file.Package.Scope.Symbols {
		if symbol.Source != w.file.Source {
			continue
		}
		items = append(items, symbolItem(symbol))
	}
	sortCompItems(items)
	return items
}

// ------------------------------------------------------------- entry ---

// completion resolves the cursor to its context over the checked
// snapshot and reports the exact candidate set for it. It declines
// wherever the snapshot cannot support a query — unloadable,
// unresolvable, or missing scope data — so unparseable buffers never
// receive guessed candidates. ok distinguishes a decline (null on the
// wire) from an empty candidate list.
func completion(snapshot *driver.Snapshot, file string, line, character int) ([]compItem, bool) {
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return nil, false
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil, false
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
		return nil, false
	}
	resolved, ok := snapshot.World.Files[src]
	if !ok || resolved == nil {
		return nil, false
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return nil, false
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return nil, false
	}
	walker := &compWalker{text: string(src.Bytes), file: resolved}
	walker.walkFile(src.Syntax)
	items := walker.candidates(walker.classify(src.Syntax, offset), offset)
	if items == nil {
		items = []compItem{}
	}
	return items, true
}

// completion answers textDocument/completion over the checked snapshot:
// the exact candidate set for the cursor context, or null where the
// snapshot cannot support the query. Like the other queries it reads
// an inert overlay snapshot and returns freshly built items.
func (s *lspServer) completion(uri string, line, character int) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	snapshot, err := driver.CheckSnapshot(discoverRoot(doc.path), doc.path, s.overlay)
	if err != nil || snapshot.World == nil {
		return nil
	}
	items, ok := completion(snapshot, doc.path, line, character)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"label":         item.label,
			"kind":          item.kind,
			"detail":        item.tail,
			"documentation": item.doc,
		})
	}
	return out
}

// canlc rename: validated safe rename over the reference index.
//
// The token at the cursor resolves to its binding identity through the
// G03 project index — top-level symbols, record fields, explicit with
// pins to the callee's near parameter, and function-local bindings by
// scope identity — and every occurrence sharing that identity,
// declaration site included, becomes one atomic WorkspaceEdit.
// Shadowed or same-spelled bindings in other scopes are different
// identities, so they stay untouched; implicit fallback captures keep
// their own caller-scope identity as well.
//
// Validation follows the --write discipline. The renamed text checks
// through a copied overlay carrying every other open buffer, and the
// candidate must introduce no new error beside the live snapshot's
// own: pre-existing diagnostics ride along, added ones veto. The
// renamed occurrences must also re-resolve to exactly one shared
// identity in the candidate index — no renamed use may resolve
// elsewhere, and no pre-existing occurrence may merge into the
// renamed group. Anything unresolvable, misspelled, colliding,
// capturing, or breaking the check declines to null so the editor
// applies nothing; renaming to the current spelling yields an empty
// edit. Warnings never block a rename, matching the CLI's
// warnings-only exit 0.

// renameNamePattern is the Can identifier spelling (lexer namePattern):
// a lowercase lead, lowercase/digit body, underscore-joined segments.
var renameNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// validRenameName admits only spellings the lexer emits as a Name: the
// identifier pattern minus hard keywords. Contextual words stay
// admissible — the candidate re-check owns positional breakage.
func validRenameName(name string) bool {
	if name == "" || syntax.IsHardKeyword(name) {
		return false
	}
	return renameNamePattern.MatchString(name)
}

// renameTarget is one occurrence replacement: the token span in the
// live snapshot plus the span the renamed token occupies in the
// candidate text.
type renameTarget struct {
	file    string
	span    source.Span
	renamed source.Span
}

// renameTokenRange narrows an indexed span to the token rename
// replaces. Qualified occurrences span the whole `pkg::name` form, so
// only the trailing segment is replaced; every other occurrence is a
// single Name token already.
func renameTokenRange(text string, span source.Span) (source.Span, bool) {
	if span.Start < 0 || span.End > len(text) || span.Start >= span.End {
		return source.Span{}, false
	}
	raw := text[span.Start:span.End]
	if i := strings.LastIndex(raw, "::"); i >= 0 {
		return source.Span{Start: span.Start + i + 2, End: span.End}, true
	}
	return span, true
}

// renamePlan resolves the token at an editor offset to its binding
// identity and lays out the replacement of every occurrence sharing
// it. It declines wherever the token is unindexed or a replacement
// range cannot be laid out exactly.
func renamePlan(snapshot *driver.Snapshot, file string, line, character int, newName string) (string, map[string][]renameTarget, bool) {
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return "", nil, false
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return "", nil, false
	}
	bases := map[string]*project.Source{}
	var src *project.Source
	for _, p := range snapshot.Graph.Projects {
		for _, s := range p.Sources {
			bases[s.Path] = s
			if s.Path == canonical {
				src = s
			}
		}
	}
	if src == nil || src.Syntax == nil {
		return "", nil, false
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return "", nil, false
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return "", nil, false
	}
	index := buildReferenceIndex(snapshot)
	at := index.occurrenceAt(canonical, offset)
	if at == nil {
		return "", nil, false
	}
	anchor, ok := renameTokenRange(string(src.Bytes), at.span)
	if !ok {
		return "", nil, false
	}
	oldName := string(src.Bytes[anchor.Start:anchor.End])
	if oldName == "" || oldName != newName && !validRenameName(newName) {
		return "", nil, false
	}
	group := index.byID[at.id]
	if len(group) == 0 {
		return "", nil, false
	}
	byFile := map[string][]*refOccurrence{}
	for _, occurrence := range group {
		byFile[occurrence.file] = append(byFile[occurrence.file], occurrence)
	}
	targets := map[string][]renameTarget{}
	for path, occurrences := range byFile {
		base, ok := bases[path]
		if !ok || base == nil {
			return "", nil, false
		}
		body := string(base.Bytes)
		sorted := append([]*refOccurrence(nil), occurrences...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].span.Start < sorted[j].span.Start })
		delta := 0
		previous := -1
		for _, occurrence := range sorted {
			token, ok := renameTokenRange(body, occurrence.span)
			if !ok || token.Start < previous || body[token.Start:token.End] != oldName {
				return "", nil, false
			}
			previous = token.End
			shifted := source.Span{Start: token.Start + delta, End: token.Start + delta + len(newName)}
			delta += len(newName) - (token.End - token.Start)
			targets[path] = append(targets[path], renameTarget{file: path, span: token, renamed: shifted})
		}
	}
	return oldName, targets, true
}

// applyRenameTargets splices the new spelling into each edited file.
// Spans ascend within a file, so one left-to-right pass suffices.
func applyRenameTargets(bases map[string]string, targets map[string][]renameTarget, newName string) map[string]string {
	out := map[string]string{}
	for path, list := range targets {
		body := bases[path]
		var rebuilt strings.Builder
		cursor := 0
		for _, target := range list {
			rebuilt.WriteString(body[cursor:target.span.Start])
			rebuilt.WriteString(newName)
			cursor = target.span.End
		}
		rebuilt.WriteString(body[cursor:])
		out[path] = rebuilt.String()
	}
	return out
}

// renameErrorKey fingerprints one error diagnostic for the no-new-error
// comparison: severity, code, file, line span, and the message with
// digit runs blanked, since checker messages embed byte offsets that
// shift under any length-changing rename. Columns and related notes
// shift too, so they stay out of the key.
func renameErrorKey(diagnostic driver.Diagnostic) string {
	var message strings.Builder
	digits := false
	for _, r := range diagnostic.Message {
		if r >= '0' && r <= '9' {
			if !digits {
				message.WriteByte('#')
				digits = true
			}
			continue
		}
		digits = false
		message.WriteRune(r)
	}
	return diagnostic.Severity + "\x00" + diagnostic.Code + "\x00" + diagnostic.File + "\x00" +
		strconv.Itoa(diagnostic.Line) + "\x00" + strconv.Itoa(diagnostic.EndLine) + "\x00" + message.String()
}

// renameIntroducesNoErrors reports whether every candidate error has a
// live counterpart: pre-existing errors ride along, added ones veto.
// Warnings never count on either side.
func renameIntroducesNoErrors(live, candidate []driver.Diagnostic) bool {
	remaining := map[string]int{}
	for _, diagnostic := range live {
		if diagnostic.Severity == "warning" {
			continue
		}
		remaining[renameErrorKey(diagnostic)]++
	}
	for _, diagnostic := range candidate {
		if diagnostic.Severity == "warning" {
			continue
		}
		key := renameErrorKey(diagnostic)
		if remaining[key] == 0 {
			return false
		}
		remaining[key]--
	}
	return true
}

// renameStable re-resolves every renamed token in the candidate index:
// each must carry the new spelling, all must share one identity, and
// that identity must own exactly the renamed set — no use lost to
// another binding, none gained from one.
func renameStable(proposed *driver.Snapshot, targets map[string][]renameTarget, newName string) bool {
	if proposed == nil || proposed.World == nil || proposed.Graph == nil {
		return false
	}
	bodies := map[string]string{}
	for _, p := range proposed.Graph.Projects {
		for _, s := range p.Sources {
			bodies[s.Path] = string(s.Bytes)
		}
	}
	index := buildReferenceIndex(proposed)
	want := ""
	total := 0
	for _, list := range targets {
		for _, target := range list {
			total++
			body, ok := bodies[target.file]
			if !ok || target.renamed.Start < 0 || target.renamed.End > len(body) || body[target.renamed.Start:target.renamed.End] != newName {
				return false
			}
			occurrence := index.occurrenceAt(target.file, target.renamed.Start)
			if occurrence == nil {
				return false
			}
			if want == "" {
				want = occurrence.id
			} else if occurrence.id != want {
				return false
			}
		}
	}
	return want != "" && len(index.byID[want]) == total
}

// rename answers textDocument/rename over the checked snapshot: one
// atomic WorkspaceEdit for the cursor token's binding identity, or
// null where the name, the token, or the validated candidate cannot
// support the rename. Like the other queries it reads an inert
// overlay snapshot and validates through a copied overlay, so live
// editor state never changes.
func (s *lspServer) rename(uri string, line, character int, newName string) any {
	doc, ok := s.docs[uri]
	if !ok || doc.path == "" {
		return nil
	}
	if !validRenameName(newName) {
		return nil
	}
	root := discoverRoot(doc.path)
	snapshot, err := driver.CheckSnapshot(root, doc.path, s.overlay)
	if err != nil || snapshot.World == nil || snapshot.Graph == nil {
		return nil
	}
	oldName, targets, ok := renamePlan(snapshot, doc.path, line, character, newName)
	if !ok {
		return nil
	}
	if oldName == newName {
		return map[string]any{"changes": map[string]any{}}
	}
	live := s.overlay.Snapshot()
	candidate := project.NewOverlay()
	for path, entry := range live {
		if err := candidate.Set(path, entry.Version, entry.Text); err != nil {
			return nil
		}
	}
	bases := map[string]string{}
	versions := map[string]int64{}
	for _, p := range snapshot.Graph.Projects {
		for _, s := range p.Sources {
			bases[s.Path] = string(s.Bytes)
			if entry, ok := live[s.Path]; ok {
				versions[s.Path] = entry.Version
			}
		}
	}
	for path, text := range applyRenameTargets(bases, targets, newName) {
		if err := candidate.Set(path, versions[path], text); err != nil {
			return nil
		}
	}
	proposed, err := driver.CheckSnapshot(root, doc.path, candidate)
	if err != nil || proposed == nil {
		return nil
	}
	if !renameIntroducesNoErrors(snapshot.Diagnostics, proposed.Diagnostics) {
		return nil
	}
	if !renameStable(proposed, targets, newName) {
		return nil
	}
	files := map[string]*source.File{}
	converted := func(path string) (*source.File, bool) {
		if file, ok := files[path]; ok {
			return file, file != nil
		}
		body, ok := bases[path]
		if !ok {
			files[path] = nil
			return nil, false
		}
		file, err := source.New(path, body)
		if err != nil {
			files[path] = nil
			return nil, false
		}
		files[path] = file
		return file, true
	}
	paths := make([]string, 0, len(targets))
	for path := range targets {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	changes := map[string]any{}
	for _, path := range paths {
		file, ok := converted(path)
		if !ok {
			return nil
		}
		edits := []any{}
		for _, target := range targets[path] {
			start, startErr := file.UTF16Position(target.span.Start)
			end, endErr := file.UTF16Position(target.span.End)
			if startErr != nil || endErr != nil || start.Line != end.Line {
				return nil
			}
			edits = append(edits, map[string]any{
				"range": map[string]any{
					"start": map[string]any{"line": start.Line, "character": start.Character},
					"end":   map[string]any{"line": end.Line, "character": end.Character},
				},
				"newText": newName,
			})
		}
		changes[uriFromPath(path)] = edits
	}
	return map[string]any{"changes": changes}
}

// canlc lsp: an inert, snapshot-based Language Server over stdio.
//
// Run: canlc lsp [--stdio]   (editors connect stdout/stdin with
// Content-Length framing).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/editortrace"
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
	if parsed.Host != "" && parsed.Host != "localhost" {
		return ""
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	return parsed.Path
}

func uriFromPath(path string) string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return "file://" + strings.Join(segments, "/")
}

func writeFrame(w *bufio.Writer, v any) error {
	defer editortrace.Stage("serialize-flush")()
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
	docs         map[string]*lspDoc
	overlay      *project.Overlay
	published    map[string]string // URI -> analysis root that owns the publication
	workspaces   []string
	ctx          context.Context
	cache        map[string]*lspCachedAnalysis
	features     map[*driver.Snapshot]*lspFeatureData
	used         map[string]*lspCachedAnalysis
	analyze      func(context.Context, string, string, *project.Overlay) (*driver.Snapshot, error)
	inlayEnabled bool
	analysisErr  error
}

func newLSPServer() *lspServer {
	return &lspServer{inlayEnabled: true, docs: map[string]*lspDoc{}, overlay: project.NewOverlay(), published: map[string]string{}, ctx: context.Background(), cache: map[string]*lspCachedAnalysis{}, features: map[*driver.Snapshot]*lspFeatureData{}, used: map[string]*lspCachedAnalysis{}}
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
	if path := os.Getenv("CAN_LSP_TRACE"); path != "" {
		closeTrace, err := editortrace.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		serveLSP(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout))
		// Write, cap and close failures invalidate the trace; report them
		// in the exit status instead of masquerading as a clean run.
		if err := closeTrace(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return 0
	}
	serveLSP(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout))
	return 0
}

// Canonicalize existing ancestors for new unsaved source files as well as
// ordinary files, retaining the original URI separately for replies.
func canonicalDocumentPath(path string) string {
	if path == "" || !filepath.IsAbs(path) {
		return ""
	}
	ancestor := path
	for {
		if real, err := filepath.EvalSymlinks(ancestor); err == nil {
			rel, err := filepath.Rel(ancestor, path)
			if err == nil {
				return filepath.Join(real, rel)
			}
			return path
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return filepath.Clean(path)
		}
		ancestor = parent
	}
}

func (s *lspServer) open(uri, text string, version int64) {
	path := canonicalDocumentPath(pathFromURI(uri))
	if strings.HasPrefix(uri, "untitled:") {
		path = scratchPath(uri)
	}
	previous := s.docs[uri]
	s.docs[uri] = &lspDoc{path: path, text: text, version: version}
	if previous != nil && previous.path != path {
		s.syncOverlay(previous.path)
	}
	s.syncOverlay(path)
}

// Full-sync changes must move forward. Replayed old versions never replace newer bytes.
func (s *lspServer) change(uri, text string, version int64) {
	doc := s.docs[uri]
	if doc == nil || version <= doc.version {
		return
	}
	doc.text, doc.version = text, version
	s.syncOverlay(doc.path)
}

func (s *lspServer) close(out *bufio.Writer, uri string) {
	if doc := s.docs[uri]; doc != nil {
		delete(s.docs, uri)
		s.syncOverlay(doc.path)
	}
	delete(s.published, uri)
	_ = publishBridgeDiagnostics(out, uri, nil, nil, s.uri)
}

// Multiple URIs can name the same canonical file. Preserve their buffers;
// agreeing text shares analysis, while conflicting text has no unique meaning.
func (s *lspServer) documentURIs(path string) []string {
	var uris []string
	for uri, doc := range s.docs {
		if doc.path == path {
			uris = append(uris, uri)
		}
	}
	sort.Strings(uris)
	return uris
}
func (s *lspServer) syncOverlay(path string) {
	if path == "" {
		return
	}
	uris := s.documentURIs(path)
	_ = s.overlay.Clear(path)
	if len(uris) > 0 {
		doc := s.docs[uris[0]]
		_ = s.overlay.Set(path, doc.version, doc.text)
	}
}
func (s *lspServer) uri(path string) string {
	if uris := s.documentURIs(path); len(uris) > 0 {
		return uris[0]
	}
	return uriFromPath(path)
}
func (s *lspServer) editablePath(path string) error {
	if len(s.documentURIs(path)) > 1 {
		return &lspResponseError{-32803, "close duplicate URI aliases for " + path + " before applying source edits"}
	}
	return nil
}
func (s *lspServer) bufferConflict(root string) error {
	paths := map[string]bool{}
	for _, doc := range s.docs {
		if doc.path == root || project.Contains(root, doc.path) {
			paths[doc.path] = true
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	for _, path := range ordered {
		uris := s.documentURIs(path)
		for _, uri := range uris[1:] {
			if s.docs[uri].text != s.docs[uris[0]].text {
				return &lspResponseError{-32803, "conflicting unsaved URI aliases for " + path + "; reconcile or close a duplicate buffer to resume Can analysis"}
			}
		}
	}
	return nil
}

func (s *lspServer) root(uri string) string {
	doc := s.docs[uri]
	if doc == nil || doc.path == "" {
		return ""
	}
	if strings.HasPrefix(uri, "untitled:") {
		return doc.path
	}
	return discoverRoot(doc.path)
}

func (s *lspServer) diagnose(out *bufio.Writer, uri string) {
	s.diagnoseRoot(out, s.root(uri), uri)
}

func (s *lspServer) diagnoseRoot(out *bufio.Writer, root, uri string) {
	if root == "" {
		return
	}
	var snapshot *driver.Snapshot
	var err error
	if s.docs[uri] != nil {
		snapshot, err = s.snapshot(uri)
	} else {
		snapshot, err = s.snapshotRoot(root)
	}
	if err != nil {
		if s.ctx.Err() == nil {
			_ = writeFrame(out, map[string]any{"jsonrpc": "2.0", "method": "window/logMessage", "params": map[string]any{"type": 1, "message": "Can analysis: " + err.Error()}})
		}
		return
	}
	byURI := map[string][]driver.Diagnostic{}
	projectMessages := []string{}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.File == "" {
			projectMessages = append(projectMessages, diagnostic.Message)
			continue
		}
		targets := s.documentURIs(diagnostic.File)
		if len(targets) == 0 {
			targets = []string{uriFromPath(diagnostic.File)}
		}
		for _, target := range targets {
			byURI[target] = append(byURI[target], diagnostic)
		}
	}
	for openURI, doc := range s.docs {
		// Dependencies also belong to the current root's graph. Never clear a foreign root.
		belongs := s.root(openURI) == root
		if !belongs && snapshot.Graph != nil {
			for _, p := range snapshot.Graph.Projects {
				for _, src := range p.Sources {
					if src.Path == doc.path {
						belongs = true
					}
				}
			}
		}
		if belongs && byURI[openURI] == nil {
			byURI[openURI] = []driver.Diagnostic{}
		}
	}
	for target, diagnostics := range byURI {
		var version *int64
		if doc := s.docs[target]; doc != nil {
			value := doc.version
			version = &value
		}
		_ = publishBridgeDiagnostics(out, target, version, diagnostics, s.uri)
		s.published[target] = root
	}
	for target, owner := range s.published {
		if owner != root {
			continue
		}
		if _, present := byURI[target]; !present {
			_ = publishBridgeDiagnostics(out, target, nil, nil, s.uri)
			delete(s.published, target)
		}
	}
	_ = writeFrame(out, map[string]any{"jsonrpc": "2.0", "method": "can/projectStatus", "params": map[string]any{"rootUri": uriFromPath(root), "messages": projectMessages, "version": version}})
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

func publishBridgeDiagnostics(out *bufio.Writer, uri string, version *int64, diags []driver.Diagnostic, uriForPath func(string) string) error {
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
		switch d.Severity {
		case "warning":
			severity = 2
		case "note", "information":
			severity = 3
		case "hint":
			severity = 4
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
				if r.File == "" {
					continue
				}
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
						"uri": uriForPath(r.File),
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

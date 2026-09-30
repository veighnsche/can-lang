package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/editortrace"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const maxLSPFrame = 16 << 20
const maxPendingRequests = 128

type lspResponseError struct {
	code    int
	message string
}

func (e *lspResponseError) Error() string { return e.message }

type lspInput struct {
	msg rpcMsg
	err error
}
type lspQueuedRequest struct {
	msg       rpcMsg
	revision  uint64
	cancelled bool
}
type lspWork struct {
	server    *lspServer
	msg       rpcMsg
	uri       string
	root      string
	revision  uint64
	cancel    context.CancelFunc
	cancelled bool
}
type lspWorkResult struct {
	work   *lspWork
	frames []byte
	err    error
}

// Framing has explicit resource limits. A bad body can be skipped safely; an
// invalid length cannot be resynchronized without guessing where the next frame starts.
func readLSPMessage(in *bufio.Reader) (rpcMsg, error) {
	var msg rpcMsg
	length := -1
	size := 0
	for {
		var raw []byte
		for {
			fragment, err := in.ReadSlice('\n')
			size += len(fragment)
			if size > 8192 {
				return msg, fmt.Errorf("LSP header exceeds 8192 bytes")
			}
			raw = append(raw, fragment...)
			if errors.Is(err, bufio.ErrBufferFull) {
				continue
			}
			if err != nil {
				return msg, err
			}
			break
		}
		line := strings.TrimSpace(string(raw))
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return msg, fmt.Errorf("invalid LSP header")
		}
		if strings.EqualFold(name, "Content-Length") {
			if length != -1 {
				return msg, fmt.Errorf("duplicate Content-Length")
			}
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 || n > maxLSPFrame {
				return msg, fmt.Errorf("Content-Length must be between 0 and %d", maxLSPFrame)
			}
			length = n
		}
	}
	if length < 0 {
		return msg, fmt.Errorf("missing Content-Length")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(in, data); err != nil {
		return msg, err
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, &lspResponseError{-32700, "invalid JSON-RPC body"}
	}
	if msg.Method == "" {
		return msg, &lspResponseError{-32600, "missing JSON-RPC method"}
	}
	return msg, nil
}

func (s *lspServer) clone(ctx context.Context) *lspServer {
	copy := newLSPServer()
	copy.ctx, copy.cache, copy.features = ctx, s.cache, s.features
	copy.analyze = s.analyze
	copy.inlayEnabled = s.inlayEnabled
	copy.workspaces = append([]string(nil), s.workspaces...)
	for uri, doc := range s.docs {
		value := *doc
		copy.docs[uri] = &value
	}
	for path, entry := range s.overlay.Snapshot() {
		_ = copy.overlay.Set(path, entry.Version, entry.Text)
	}
	for uri, root := range s.published {
		copy.published[uri] = root
	}
	return copy
}

// One worker owns every mutable compiler and semantic-index operation. Intake
// owns document state and can cancel an obsolete analysis without waiting for it.
func serveLSP(in *bufio.Reader, out *bufio.Writer) {
	serveLSPWithServer(newLSPServer(), in, out)
}

func serveLSPWithServer(server *lspServer, in *bufio.Reader, out *bufio.Writer) {
	inputs := make(chan lspInput, 32)
	stopReader := make(chan struct{})
	defer close(stopReader)
	go func() {
		defer close(inputs)
		for {
			msg, err := readLSPMessage(in)
			select {
			case inputs <- lspInput{msg, err}:
			case <-stopReader:
				return
			}
			var protocolErr *lspResponseError
			if (err != nil && !errors.As(err, &protocolErr)) || msg.Method == "exit" {
				return
			}
		}
	}()
	work := make(chan *lspWork)
	completed := make(chan lspWorkResult, 1)
	defer close(work)
	go func() {
		for job := range work {
			var buffer bytes.Buffer
			writer := bufio.NewWriter(&buffer)
			var err error
			end := editortrace.Request(job.msg.Method, job.msg.ID, 0)
			if job.root != "" {
				job.server.diagnoseRoot(writer, job.root, job.uri)
			} else {
				result, requestErr := job.server.request(job.msg)
				if requestErr != nil {
					code := -32602
					var responseErr *lspResponseError
					if errors.As(requestErr, &responseErr) {
						code = responseErr.code
					}
					err = writeFrame(writer, rpcError(job.msg.ID, code, requestErr.Error()))
				} else {
					err = writeFrame(writer, map[string]any{"jsonrpc": "2.0", "id": job.msg.ID, "result": result})
				}
			}
			end()
			if job.server.ctx.Err() != nil {
				err = job.server.ctx.Err()
			}
			if err == nil {
				err = job.server.analysisErr
			}
			if err == nil {
				err = job.server.validateInputs()
			}
			completed <- lspWorkResult{job, buffer.Bytes(), err}
		}
	}()
	var revision uint64
	var active *lspWork
	pending := []lspQueuedRequest{}
	dirty := map[string]string{} // one newest diagnostic job per root
	retries := map[string]int{}
	var input <-chan lspInput = inputs
	shutdown := false
	var writeErr error
	send := func(v any) {
		if writeErr == nil {
			writeErr = writeFrame(out, v)
		}
	}
	invalidate := func() {
		revision++
		clear(retries)
		if active != nil {
			active.cancel()
		}
		// Keep roots with published findings fresh even after their last
		// buffer closes. Workspace manifests also remain watchable inputs.
		for _, root := range server.published {
			if root != "" {
				dirty[root] = ""
			}
		}
		for _, workspace := range server.workspaces {
			root := canonicalDocumentPath(workspace)
			manifest := filepath.Join(root, "can.project.json")
			if info, err := os.Stat(manifest); err == nil && info.Mode().IsRegular() {
				dirty[root] = ""
			}
		}
		for uri := range server.docs {
			if root := server.root(uri); root != "" {
				dirty[root] = uri
			}
		}
	}
	for {
		if writeErr != nil {
			if active != nil {
				active.cancel()
			}
			return
		}
		if active == nil {
			for len(pending) > 0 {
				next := pending[0]
				pending = pending[1:]
				if next.cancelled {
					send(rpcError(next.msg.ID, -32800, "request cancelled"))
					continue
				}
				if next.revision != revision {
					send(rpcError(next.msg.ID, -32801, "document content changed"))
					continue
				}
				ctx, cancel := context.WithCancel(context.Background())
				active = &lspWork{server: server.clone(ctx), msg: next.msg, revision: revision, cancel: cancel}
				break
			}
			if active == nil && len(dirty) > 0 {
				roots := make([]string, 0, len(dirty))
				for root := range dirty {
					roots = append(roots, root)
				}
				sort.Strings(roots)
				uri := dirty[roots[0]]
				delete(dirty, roots[0])
				if roots[0] != "" {
					ctx, cancel := context.WithCancel(context.Background())
					active = &lspWork{server: server.clone(ctx), uri: uri, root: roots[0], revision: revision, cancel: cancel, msg: rpcMsg{Method: "textDocument/publishDiagnostics"}}
				}
			}
			if active != nil {
				work <- active
			}
		}
		if input == nil && active == nil && len(pending) == 0 && len(dirty) == 0 {
			return
		}
		select {
		case read, ok := <-input:
			if !ok {
				input = nil
				continue
			}
			if read.err != nil {
				if errors.Is(read.err, io.EOF) {
					input = nil
					continue
				}
				var responseErr *lspResponseError
				if errors.As(read.err, &responseErr) {
					send(rpcError(nil, responseErr.code, responseErr.message))
					continue
				}
				send(rpcError(nil, -32600, read.err.Error()))
				input = nil
				continue
			}
			msg := read.msg
			if shutdown && msg.Method != "exit" && msg.Method != "shutdown" {
				if msg.ID != nil {
					send(rpcError(msg.ID, -32600, "server is shut down"))
				}
				continue
			}
			switch msg.Method {
			case "exit":
				if active != nil {
					active.cancel()
				}
				// Exit ends the transport immediately. A canceled worker may
				// finish its current stage, but no queued result is published.
				return
			case "initialize":
				var p struct {
					RootURI               string `json:"rootUri"`
					InitializationOptions struct {
						InlayHints *bool `json:"inlayHints"`
					} `json:"initializationOptions"`
					Folders []struct {
						URI string `json:"uri"`
					} `json:"workspaceFolders"`
				}
				if json.Unmarshal(msg.Params, &p) == nil {
					if p.InitializationOptions.InlayHints != nil {
						server.inlayEnabled = *p.InitializationOptions.InlayHints
					}
					if path := pathFromURI(p.RootURI); path != "" {
						server.workspaces = append(server.workspaces, path)
					}
					for _, folder := range p.Folders {
						if path := pathFromURI(folder.URI); path != "" {
							server.workspaces = append(server.workspaces, path)
						}
					}
				}
				caps := map[string]any{"textDocumentSync": map[string]any{"openClose": true, "change": 1, "save": map[string]any{"includeText": false}}, "definitionProvider": true, "documentFormattingProvider": true, "hoverProvider": true, "referencesProvider": true, "completionProvider": map[string]any{"triggerCharacters": []string{".", ":"}}, "renameProvider": true, "workspace": map[string]any{"workspaceFolders": map[string]any{"supported": true, "changeNotifications": true}}, "positionEncoding": "utf-16"}
				for k, v := range featureCapabilities() {
					caps[k] = v
				}
				send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": map[string]any{"capabilities": caps, "serverInfo": map[string]any{"name": "canlc", "version": version}}})
			case "initialized":
			case "shutdown":
				shutdown = true
				if active != nil {
					active.cancelled = true
					active.cancel()
				}
				for _, request := range pending {
					send(rpcError(request.msg.ID, -32800, "server is shutting down"))
				}
				pending = nil
				clear(dirty)
				send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": nil})
			case "$/cancelRequest":
				var p struct {
					ID json.RawMessage `json:"id"`
				}
				if json.Unmarshal(msg.Params, &p) != nil {
					continue
				}
				if active != nil && active.msg.ID != nil && string(*active.msg.ID) == string(p.ID) {
					active.cancelled = true
					active.cancel()
				}
				for i := range pending {
					if pending[i].msg.ID != nil && string(*pending[i].msg.ID) == string(p.ID) {
						pending[i].cancelled = true
					}
				}
			case "textDocument/didOpen":
				var p struct {
					TextDocument struct {
						URI, Text string
						Version   int64
					} `json:"textDocument"`
				}
				if json.Unmarshal(msg.Params, &p) == nil && p.TextDocument.URI != "" {
					server.open(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
					invalidate()
				}
			case "textDocument/didChange":
				var p struct {
					TextDocument struct {
						URI     string
						Version int64
					} `json:"textDocument"`
					Changes []struct {
						Text  string
						Range json.RawMessage
					} `json:"contentChanges"`
				}
				if json.Unmarshal(msg.Params, &p) != nil || len(p.Changes) == 0 {
					continue
				}
				doc := server.docs[p.TextDocument.URI]
				if doc == nil || p.TextDocument.Version <= doc.version {
					continue
				}
				full := true
				for _, change := range p.Changes {
					if len(change.Range) > 0 && string(change.Range) != "null" {
						full = false
					}
				}
				if !full {
					send(map[string]any{"jsonrpc": "2.0", "method": "window/logMessage", "params": map[string]any{"type": 1, "message": "Can requires full text synchronization; incremental change rejected"}})
					continue
				}
				server.change(p.TextDocument.URI, p.Changes[len(p.Changes)-1].Text, p.TextDocument.Version)
				invalidate()
			case "textDocument/didClose":
				var p struct {
					TextDocument docID `json:"textDocument"`
				}
				if json.Unmarshal(msg.Params, &p) == nil {
					root := server.root(p.TextDocument.URI)
					server.close(out, p.TextDocument.URI)
					invalidate()
					if root != "" && !strings.HasPrefix(p.TextDocument.URI, "untitled:") {
						dirty[root] = ""
					}
				}
			case "textDocument/didSave", "workspace/didChangeWatchedFiles":
				invalidate()
			case "workspace/didChangeConfiguration":
				var p struct {
					Settings struct {
						Canlc struct {
							InlayHints *bool `json:"inlayHints"`
						} `json:"canlc"`
					} `json:"settings"`
				}
				if json.Unmarshal(msg.Params, &p) == nil && p.Settings.Canlc.InlayHints != nil {
					server.inlayEnabled = *p.Settings.Canlc.InlayHints
				}
				invalidate()
			case "workspace/didChangeWorkspaceFolders":
				var p struct {
					Event struct {
						Added, Removed []struct {
							URI string `json:"uri"`
						}
					} `json:"event"`
				}
				if json.Unmarshal(msg.Params, &p) == nil {
					for _, removed := range p.Event.Removed {
						path := pathFromURI(removed.URI)
						for i := len(server.workspaces) - 1; i >= 0; i-- {
							if server.workspaces[i] == path {
								server.workspaces = append(server.workspaces[:i], server.workspaces[i+1:]...)
							}
						}
					}
					for _, added := range p.Event.Added {
						if path := pathFromURI(added.URI); path != "" {
							server.workspaces = append(server.workspaces, path)
						}
					}
					invalidate()
				}
			default:
				if msg.ID == nil {
					continue
				}
				if shutdown {
					send(rpcError(msg.ID, -32600, "server is shut down"))
					continue
				}
				if len(pending) >= maxPendingRequests {
					send(rpcError(msg.ID, -32000, "Can request queue is full; retry after pending requests complete"))
					continue
				}
				pending = append(pending, lspQueuedRequest{msg: msg, revision: revision})
			}
		case result := <-completed:
			result.work.cancel()
			if result.work.cancelled {
				if result.work.msg.ID != nil {
					send(rpcError(result.work.msg.ID, -32800, "request cancelled"))
				}
			} else if result.work.revision != revision || result.err != nil {
				stale := result.work.revision != revision || errors.Is(result.err, context.Canceled) || errors.Is(result.err, errAnalysisChanged)
				if result.work.msg.ID != nil {
					if stale {
						send(rpcError(result.work.msg.ID, -32801, "analysis inputs changed; retry the request"))
					} else {
						code := -32603
						var responseErr *lspResponseError
						if errors.As(result.err, &responseErr) {
							code = responseErr.code
						}
						send(rpcError(result.work.msg.ID, code, "Can analysis failed: "+result.err.Error()))
					}
				}
				root := result.work.root
				closedScratch := strings.HasPrefix(result.work.uri, "untitled:") && server.docs[result.work.uri] == nil
				if root != "" && !closedScratch {
					if stale && (result.work.revision != revision || retries[root] < 2) {
						retries[root]++
						dirty[root] = result.work.uri
					} else {
						message := "Can analysis failed"
						if result.err != nil {
							message += ": " + result.err.Error()
						}
						for uri, owner := range server.published {
							if owner != root {
								continue
							}
							var docVersion *int64
							if doc := server.docs[uri]; doc != nil {
								v := doc.version
								docVersion = &v
							}
							if err := publishBridgeDiagnostics(out, uri, docVersion, nil, server.uri); err != nil {
								writeErr = err
							}
							delete(server.published, uri)
						}
						send(map[string]any{"jsonrpc": "2.0", "method": "can/projectStatus", "params": map[string]any{"rootUri": uriFromPath(root), "messages": []string{message}, "version": version}})
					}
				}
			} else {
				if _, err := out.Write(result.frames); err != nil {
					writeErr = err
				} else {
					writeErr = out.Flush()
				}
				if result.work.root != "" {
					delete(retries, result.work.root)
					server.published = result.work.server.published
				}
			}
			active = nil
		}
	}
}

func rpcError(id *json.RawMessage, code int, message string) any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}

// Required coordinates must be present: Go zero values are valid positions,
// so decoding alone cannot distinguish a missing field from an actual 0,0.
func requireLSPFields(params json.RawMessage, paths ...string) error {
	var root map[string]json.RawMessage
	if json.Unmarshal(params, &root) != nil || root == nil {
		return &lspResponseError{-32602, "request parameters must be an object"}
	}
	for _, path := range paths {
		fields := root
		parts := strings.Split(path, ".")
		for i, part := range parts {
			value, exists := fields[part]
			if !exists || string(value) == "null" {
				return &lspResponseError{-32602, "missing request parameter " + path}
			}
			if i < len(parts)-1 {
				var nested map[string]json.RawMessage
				if json.Unmarshal(value, &nested) != nil || nested == nil {
					return &lspResponseError{-32602, "invalid request parameter " + path}
				}
				fields = nested
			}
		}
	}
	return nil
}

func (s *lspServer) request(msg rpcMsg) (any, error) {
	if result, handled, err := s.handleFeature(msg); handled {
		return result, err
	}
	switch msg.Method {
	case "textDocument/definition", "textDocument/hover", "textDocument/references", "textDocument/completion", "textDocument/rename", "textDocument/formatting":
	default:
		return nil, &lspResponseError{-32601, "unknown method " + msg.Method}
	}

	required := []string{"textDocument.uri"}
	if msg.Method != "textDocument/formatting" {
		required = append(required, "position.line", "position.character")
	}
	if msg.Method == "textDocument/rename" {
		required = append(required, "newName")
	}
	if err := requireLSPFields(msg.Params, required...); err != nil {
		return nil, err
	}
	var p struct {
		TextDocument docID                         `json:"textDocument"`
		Position     struct{ Line, Character int } `json:"position"`
		Context      *struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
		NewName string `json:"newName"`
	}
	if json.Unmarshal(msg.Params, &p) != nil {
		return nil, &lspResponseError{-32602, "invalid request parameters"}
	}
	if strings.HasPrefix(msg.Method, "textDocument/") {
		doc := s.docs[p.TextDocument.URI]
		if doc == nil {
			return nil, &lspResponseError{-32602, "document is not open"}
		}
		if msg.Method == "textDocument/formatting" || msg.Method == "textDocument/rename" {
			if err := s.editablePath(doc.path); err != nil {
				return nil, err
			}
		}
		if msg.Method != "textDocument/formatting" {
			file, err := source.New(doc.path, doc.text)
			if err != nil {
				return nil, err
			}
			if _, err := file.Offset(source.UTF16Position{Line: p.Position.Line, Character: p.Position.Character}); err != nil {
				return nil, &lspResponseError{-32602, "position is outside the document or splits a character"}
			}
		}
	}
	switch msg.Method {
	case "textDocument/definition":
		return s.definition(p.TextDocument.URI, p.Position.Line, p.Position.Character), nil
	case "textDocument/hover":
		return s.hover(p.TextDocument.URI, p.Position.Line, p.Position.Character), nil
	case "textDocument/references":
		include := true
		if p.Context != nil {
			include = p.Context.IncludeDeclaration
		}
		return s.references(p.TextDocument.URI, p.Position.Line, p.Position.Character, include), nil
	case "textDocument/completion":
		return s.completion(p.TextDocument.URI, p.Position.Line, p.Position.Character), nil
	case "textDocument/rename":
		return s.renameChecked(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.NewName)
	case "textDocument/formatting":
		return s.formatting(p.TextDocument.URI), nil
	default:
		return nil, &lspResponseError{-32601, "unknown method " + msg.Method}
	}
}

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestLSPURIPathIsDecodedOnce(t *testing.T) {
	path := "/tmp/literal%20 and 😀.can"
	if got := pathFromURI(uriFromPath(path)); got != path {
		t.Fatalf("round trip %q != %q", got, path)
	}
	if pathFromURI("file://foreign/tmp/a.can") != "" {
		t.Fatal("foreign file authority accepted")
	}
}

func TestLSPFrameRejectsUnsafeLengths(t *testing.T) {
	for _, header := range []string{"Content-Length: -1\r\n\r\n", "Content-Length: 9999999999\r\n\r\n", "Content-Length: 2\r\nContent-Length: 2\r\n\r\n{}", "X: yes\r\n\r\n"} {
		if _, err := readLSPMessage(bufio.NewReader(strings.NewReader(header))); err == nil {
			t.Fatalf("accepted %q", header)
		}
	}
}

func TestLSPChangeRejectsOldOrUnknownVersions(t *testing.T) {
	s := newLSPServer()
	uri := "file:///tmp/version.can"
	s.change(uri, "unknown", 1)
	if s.docs[uri] != nil {
		t.Fatal("change implicitly opened unknown document")
	}
	s.open(uri, "new", 5)
	s.change(uri, "old", 4)
	s.change(uri, "same", 5)
	if s.docs[uri].text != "new" {
		t.Fatal("stale version replaced latest bytes")
	}
}

func TestLSPDiagnosticSeveritiesAndMultiline(t *testing.T) {
	_, got := publishBridgeFrame(t, nil, []driver.Diagnostic{
		{Line: 1, Start: 3, EndLine: 2, End: 7, Severity: "error"},
		{Severity: "warning"}, {Severity: "note"}, {Severity: "hint"},
	})
	for i, d := range got {
		if d["severity"] != float64(i+1) {
			t.Fatalf("severity %d: %v", i, d)
		}
	}
	end := got[0]["range"].(map[string]any)["end"].(map[string]any)
	if end["line"] != 2.0 || end["character"] != 7.0 {
		t.Fatalf("multiline range widened: %v", end)
	}
}

func TestLSPSnapshotReusesFactsAndInvalidatesEveryInput(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain, "src/second.can": serverSecond})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	s.open(uri, serverMain, 1)
	count := 0
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		count++
		return &driver.Snapshot{}, nil
	}
	first, err := s.snapshot(uri)
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.snapshot(uri)
	if err != nil || same != first || count != 1 {
		t.Fatalf("unchanged snapshot not reused: %d %v", count, err)
	}
	sibling := uriFromPath(filepath.Join(root, "src/second.can"))
	s.open(sibling, serverSecond+"\n", 1)
	second, err := s.snapshot(uri)
	if err != nil || second == first || count != 2 {
		t.Fatalf("sibling buffer did not invalidate: %d %v", count, err)
	}
	// Identical version with distinct text also belongs to a distinct identity.
	s.open(sibling, serverSecond+"\n\n", 1)
	if _, err = s.snapshot(uri); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatal("same-version text change reused analysis")
	}
	for _, name := range []string{"can.errors.json", "src/new.can", "can.lock.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("changed"), 0644); err != nil {
			t.Fatal(err)
		}
		before := count
		if _, err := s.snapshot(uri); err != nil {
			t.Fatal(err)
		}
		if count != before+1 {
			t.Fatalf("disk mutation %s did not invalidate", name)
		}
	}
}

func TestLSPPublicationsRemainOwnedByTheirProject(t *testing.T) {
	rootA := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	rootB := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	pathA, pathB := filepath.Join(rootA, "src/main.can"), filepath.Join(rootB, "src/main.can")
	a, b := uriFromPath(pathA), uriFromPath(pathB)
	s := newLSPServer()
	s.open(a, serverMain, 1)
	s.open(b, serverMain, 1)
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		return &driver.Snapshot{Diagnostics: []driver.Diagnostic{{File: file, Line: 3, Start: 4, EndLine: 3, End: 10, Message: "bad token", Severity: "error"}}}, nil
	}
	var out bytes.Buffer
	s.diagnose(bufio.NewWriter(&out), a)
	out.Reset()
	s.diagnose(bufio.NewWriter(&out), b)
	if strings.Contains(out.String(), a) {
		t.Fatalf("checking B cleared or republished A: %s", out.String())
	}
	if s.published[a] != rootA || s.published[b] != rootB {
		t.Fatalf("publication ownership: %v", s.published)
	}
}

func readTestFrames(reader io.Reader, frames chan<- map[string]any) {
	defer close(frames)
	in := bufio.NewReader(reader)
	for {
		length := 0
		for {
			line, err := in.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
			}
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(in, data); err != nil {
			return
		}
		var result map[string]any
		if json.Unmarshal(data, &result) == nil {
			frames <- result
		}
	}
}

func TestLSPIntakeCancelsPausedWorkerAndPublishesLatest(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	started := make(chan struct{})
	cancelled := make(chan struct{})
	var count atomic.Int32
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		if count.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return nil, ctx.Err()
		}
		return &driver.Snapshot{}, nil
	}
	input, inputWriter := io.Pipe()
	outputReader, output := io.Pipe()
	frames := make(chan map[string]any, 32)
	go readTestFrames(outputReader, frames)
	done := make(chan struct{})
	go func() {
		defer close(done)
		serveLSPWithServer(s, bufio.NewReader(input), bufio.NewWriter(output))
		output.Close()
	}()
	t.Cleanup(func() {
		inputWriter.Close()
		outputReader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server failed to stop")
		}
	})
	if _, err := io.WriteString(inputWriter, frame(didOpen(uri, serverMain, 1))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not start")
	}
	if _, err := io.WriteString(inputWriter, frame(didChange(uri, serverMain+"\n", 2))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("intake blocked behind compiler analysis")
	}
	if err := inputWriter.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not drain latest analysis")
	}
	found := false
	for result := range frames {
		if result["method"] != "textDocument/publishDiagnostics" {
			continue
		}
		params := result["params"].(map[string]any)
		if params["version"] != 2.0 {
			t.Fatalf("stale publication after cancellation: %v", params)
		}
		found = true
	}
	if !found {
		t.Fatal("latest diagnosis absent")
	}
}

func TestLSPProtocolBuildIdentity(t *testing.T) {
	frames := runExchange(t, []string{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, `{"jsonrpc":"2.0","method":"exit"}`})
	if len(frames) != 1 {
		t.Fatalf("unexpected frames: %v", frames)
	}
	result := frames[0]["result"].(map[string]any)
	if got := result["serverInfo"].(map[string]any)["version"]; got != version {
		t.Fatalf("build identity %v != %s", got, version)
	}
}

func TestLSPRequestCancellationWhileWorkerPaused(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	s.open(uri, serverMain, 1)
	started := make(chan struct{})
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	input, inputWriter := io.Pipe()
	outputReader, output := io.Pipe()
	frames := make(chan map[string]any, 8)
	go readTestFrames(outputReader, frames)
	done := make(chan struct{})
	go func() {
		defer close(done)
		serveLSPWithServer(s, bufio.NewReader(input), bufio.NewWriter(output))
		output.Close()
	}()
	t.Cleanup(func() {
		inputWriter.Close()
		outputReader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server failed to stop")
		}
	})
	line, column := positionOf(t, serverMain, "call help|er(seed)")
	if _, err := io.WriteString(inputWriter, frame(definition(12, uri, line, column))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not reach analyzer")
	}
	io.WriteString(inputWriter, frame(`{"jsonrpc":"2.0","method":"$/cancelRequest","params":{"id":12}}`))
	select {
	case response := <-frames:
		err, ok := response["error"].(map[string]any)
		if !ok || err["code"] != -32800.0 || response["id"] != 12.0 {
			t.Fatalf("cancelled request response: %v", response)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel request blocked behind analysis")
	}
	io.WriteString(inputWriter, frame(`{"jsonrpc":"2.0","method":"exit"}`))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not exit")
	}
}

func TestLSPSnapshotSeesDependencyAssetsAndSymlinkSourceCreates(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	dep := filepath.Join(root, "deps", "shared")
	if err := os.MkdirAll(filepath.Join(dep, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"can.project.json":             `{"source_root":"src","error_registry":"registry.json","dependencies":{"shared":"deps/shared"},"assets":{"payload":"payload.txt"}}`,
		"registry.json":                `{"active":[],"retired":[]}`,
		"payload.txt":                  "original",
		"deps/shared/can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"deps/shared/can.errors.json":  `{"active":[],"retired":[]}`,
		"deps/shared/src/lib.can":      serverSecond,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	linked := filepath.Join(root, "linked")
	if err := os.Mkdir(linked, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(linked, filepath.Join(root, "src", "linked")); err != nil {
		t.Fatal(err)
	}
	s := newLSPServer()
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s.open(uri, serverMain, 1)
	count := 0
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		count++
		return &driver.Snapshot{}, nil
	}
	if _, err := s.snapshot(uri); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"registry.json", "payload.txt", "deps/shared/src/lib.can", "linked/new.can"} {
		before := count
		if err := os.WriteFile(filepath.Join(root, name), []byte("different"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := s.snapshot(uri); err != nil {
			t.Fatal(err)
		}
		if count != before+1 {
			t.Fatalf("input change %s retained stale snapshot", name)
		}
	}
}

func TestLSPCloseRechecksDiskAfterLastBuffer(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	path := filepath.Join(root, "src/main.can")
	uri := uriFromPath(path)
	s := newLSPServer()
	s.open(uri, "unsaved", 1)
	var sawDisk bool
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		_, open := overlay.Get(path)
		sawDisk = !open
		return &driver.Snapshot{}, nil
	}
	var out bytes.Buffer
	serveLSPWithServer(s, bufio.NewReader(strings.NewReader(frame(didClose(uri)))), bufio.NewWriter(&out))
	if !sawDisk {
		t.Fatal("closing the final buffer did not recheck disk-backed project")
	}
}

func TestLSPSnapshotTracksSourcesBehindIndependentManifestError(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	if err := os.WriteFile(filepath.Join(root, "can.project.json"), []byte(`{"source_root":"src","error_registry":3}`), 0644); err != nil {
		t.Fatal(err)
	}
	s := newLSPServer()
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s.open(uri, serverMain, 1)
	count := 0
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		count++
		return &driver.Snapshot{}, nil
	}
	first, err := s.snapshot(uri)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src/new.can"), []byte(serverSecond), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := s.snapshot(uri)
	if err != nil || first == second || count != 2 {
		t.Fatalf("partial manifest hid source creation: %d %v", count, err)
	}
}

func TestLSPFingerprintDoesNotReadEscapedAsset(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("outside-one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "asset.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "can.project.json"), []byte(`{"source_root":"src","error_registry":"can.errors.json","assets":{"image":"asset.txt"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	s := newLSPServer()
	before, err := s.fingerprint(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside-two"), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := s.fingerprint(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !before.Equal(after) {
		t.Fatal("fingerprint hashed data outside the confined project")
	}
}

func TestLSPFrameBoundsHeaderBeforeNewline(t *testing.T) {
	_, err := readLSPMessage(bufio.NewReader(strings.NewReader(strings.Repeat("X", 16384))))
	if err == nil || !strings.Contains(err.Error(), "8192") {
		t.Fatalf("unbounded header read: %v", err)
	}
}

func TestLSPDiagnosticJobRetriesTransientDiskMutation(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	count := 0
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		count++
		if count == 1 {
			if err := os.WriteFile(filepath.Join(root, "src/created.can"), []byte(serverSecond), 0644); err != nil {
				return nil, err
			}
		}
		return &driver.Snapshot{}, nil
	}
	var out bytes.Buffer
	serveLSPWithServer(s, bufio.NewReader(strings.NewReader(frame(didOpen(uri, serverMain, 1)))), bufio.NewWriter(&out))
	if count != 2 {
		t.Fatalf("transient source creation was not retried: %d checks", count)
	}
	if !strings.Contains(out.String(), "textDocument/publishDiagnostics") {
		t.Fatalf("latest diagnostics were dropped: %s", out.String())
	}
}

func TestLSPDiagnosticRetryIsBoundedAndFailureVisible(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	count := 0
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		count++
		return nil, errAnalysisChanged
	}
	var out bytes.Buffer
	serveLSPWithServer(s, bufio.NewReader(strings.NewReader(frame(didOpen(uri, serverMain, 1)))), bufio.NewWriter(&out))
	if count != 3 {
		t.Fatalf("retry should be bounded to two additional checks: %d", count)
	}
	if !strings.Contains(out.String(), "Can analysis failed: analysis inputs changed") {
		t.Fatalf("failure remained invisible: %s", out.String())
	}
}

func TestLSPMissingPositionsAreNotSilentlyZero(t *testing.T) {
	s := newLSPServer()
	uri := "untitled:missing-position"
	s.open(uri, serverMain, 1)
	for _, params := range []string{
		`{"textDocument":{"uri":"untitled:missing-position"}}`,
		`{"textDocument":{"uri":"untitled:missing-position"},"position":null}`,
		`{"textDocument":{"uri":"untitled:missing-position"},"position":{"line":0}}`,
	} {
		_, err := s.request(rpcMsg{Method: "textDocument/hover", Params: json.RawMessage(params)})
		protocol, ok := err.(*lspResponseError)
		if !ok || protocol.code != -32602 {
			t.Fatalf("missing coordinates accepted: %s %v", params, err)
		}
	}
}

func TestLSPRelatedLocationsPreserveUntitledURI(t *testing.T) {
	s := newLSPServer()
	uri := "untitled:related-source"
	s.open(uri, "abc", 1)
	var out bytes.Buffer
	err := publishBridgeDiagnostics(bufio.NewWriter(&out), uri, nil, []driver.Diagnostic{{File: s.docs[uri].path, Message: "duplicate", Related: []driver.RelatedDiagnostic{{File: s.docs[uri].path, Message: "first"}, {Message: "unavailable"}}}}, s.uri)
	if err != nil {
		t.Fatal(err)
	}
	frames := make(chan map[string]any, 1)
	readTestFrames(strings.NewReader(out.String()), frames)
	frame := <-frames
	item := frame["params"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)
	related := item["relatedInformation"].([]any)
	if len(related) != 1 || related[0].(map[string]any)["location"].(map[string]any)["uri"] != uri {
		t.Fatalf("related URI fabricated: %v", related)
	}
}

func TestLSPConfigurationOverlayKeepsIndependentSourceFindings(t *testing.T) {
	broken := strings.Replace(serverMain, "call helper(seed)", "call missing_fn(seed)", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	sourceURI := uriFromPath(filepath.Join(root, "src/main.can"))
	configURI := uriFromPath(filepath.Join(root, "can.project.json"))
	config := "{\r\n\"source_root\":\"src\",\r\n\"error_registry\":3,\"unknown\":\"😀\"}"
	frames := runExchange(t, []string{didOpen(sourceURI, broken, 4), didOpen(configURI, config, 7), `{"jsonrpc":"2.0","method":"exit"}`})
	foundConfig, foundSource := false, false
	for _, frame := range frames {
		if frame["method"] != "textDocument/publishDiagnostics" {
			continue
		}
		params := frame["params"].(map[string]any)
		for _, raw := range params["diagnostics"].([]any) {
			diagnostic := raw.(map[string]any)
			rng := diagnostic["range"].(map[string]any)
			start, end := rng["start"].(map[string]any), rng["end"].(map[string]any)
			if params["uri"] == configURI {
				if params["version"] != 7.0 || start["line"] != 2.0 || end["line"] != 2.0 {
					t.Fatalf("config version/range: %v", params)
				}
				width := end["character"].(float64) - start["character"].(float64)
				if width != 1 && width != 9 {
					t.Fatalf("config finding widened: %v", diagnostic)
				}
				foundConfig = true
			}
			if params["uri"] == sourceURI && strings.Contains(diagnostic["message"].(string), "missing_fn") {
				if params["version"] != 4.0 || end["character"].(float64)-start["character"].(float64) != 10 {
					t.Fatalf("callee finding widened: %v", diagnostic)
				}
				foundSource = true
			}
		}
	}
	if !foundConfig || !foundSource {
		t.Fatalf("independent source/config errors absent: config=%v source=%v %v", foundConfig, foundSource, frames)
	}
}

func TestLSPExitCancelsWithoutDrainingWorker(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	started, cancelled, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		defer close(finished)
		close(started)
		<-ctx.Done()
		close(cancelled)
		<-release
		return nil, ctx.Err()
	}
	input, inputWriter := io.Pipe()
	var out bytes.Buffer
	done := make(chan struct{})
	go func() { defer close(done); serveLSPWithServer(s, bufio.NewReader(input), bufio.NewWriter(&out)) }()
	t.Cleanup(func() {
		inputWriter.Close()
		close(release)
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("worker did not release")
		}
	})
	if _, err := io.WriteString(inputWriter, frame(didOpen(uri, serverMain, 1))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("analysis not started")
	}
	if _, err := io.WriteString(inputWriter, frame(`{"jsonrpc":"2.0","method":"exit"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("exit waited for canceled compiler stage")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("exit did not cancel analysis")
	}
	if out.Len() != 0 {
		t.Fatalf("exit published queued analysis: %s", out.String())
	}
}

func TestLSPShutdownCancelsAndStopsAcceptingChanges(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	s := newLSPServer()
	started, cancelled := make(chan struct{}), make(chan struct{})
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	input, inputWriter := io.Pipe()
	reader, output := io.Pipe()
	frames := make(chan map[string]any, 4)
	go readTestFrames(reader, frames)
	done := make(chan struct{})
	go func() {
		defer close(done)
		serveLSPWithServer(s, bufio.NewReader(input), bufio.NewWriter(output))
		output.Close()
	}()
	t.Cleanup(func() {
		inputWriter.Close()
		reader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server did not release")
		}
	})
	if _, err := io.WriteString(inputWriter, frame(didOpen(uri, serverMain, 1))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker not started")
	}
	if _, err := io.WriteString(inputWriter, frame(`{"jsonrpc":"2.0","id":42,"method":"shutdown"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-frames:
		if response["id"] != 42.0 || response["error"] != nil {
			t.Fatalf("shutdown response: %v", response)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown response blocked")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not cancel worker")
	}
	if _, err := io.WriteString(inputWriter, frame(didChange(uri, serverMain+"\n", 2))+frame(`{"jsonrpc":"2.0","method":"exit"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("exit did not complete")
	}
	if s.docs[uri].version != 1 {
		t.Fatal("change accepted after shutdown")
	}
}

func TestLSPAliasClosePreservesRemainingOverlayAndVersions(t *testing.T) {
	for _, closeFirst := range []bool{false, true} {
		t.Run(strconv.FormatBool(closeFirst), func(t *testing.T) {
			root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
			path := filepath.Join(root, "src/main.can")
			alias := filepath.Join(root, "alias.can")
			if err := os.Symlink(path, alias); err != nil {
				t.Fatal(err)
			}
			uris := []string{uriFromPath(path), uriFromPath(alias)}
			s := newLSPServer()
			s.open(uris[0], serverMain+"\n", 3)
			s.open(uris[1], serverMain+"\n", 8)
			before, err := s.fingerprint(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			s.change(uris[0], serverMain+"\n", 4)
			after, err := s.fingerprint(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if before.Equal(after) {
				t.Fatal("non-selected alias version was absent from fingerprint")
			}
			first := s.uri(path)
			if first != s.uri(path) || first != s.documentURIs(path)[0] {
				t.Fatal("non-deterministic alias selection")
			}
			closed, remaining := 0, 1
			if !closeFirst {
				closed, remaining = 1, 0
			}
			var output bytes.Buffer
			s.close(bufio.NewWriter(&output), uris[closed])
			entry, ok := s.overlay.Get(path)
			if !ok || entry.Text != serverMain+"\n" || entry.Version != s.docs[uris[remaining]].version {
				t.Fatalf("closing alias lost surviving buffer: %+v", entry)
			}
		})
	}
}

func TestLSPAgreeingAliasesReceiveOwnVersions(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	path := filepath.Join(root, "src/main.can")
	alias := filepath.Join(root, "alias.can")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	s := newLSPServer()
	realURI, aliasURI := uriFromPath(path), uriFromPath(alias)
	s.open(realURI, serverMain, 3)
	s.open(aliasURI, serverMain, 9)
	s.analyze = func(ctx context.Context, root, file string, overlay *project.Overlay) (*driver.Snapshot, error) {
		return &driver.Snapshot{Diagnostics: []driver.Diagnostic{{File: path, Message: "token", Line: 0, Start: 0, EndLine: 0, End: 7}}}, nil
	}
	var out bytes.Buffer
	s.diagnose(bufio.NewWriter(&out), realURI)
	frames := make(chan map[string]any, 8)
	readTestFrames(strings.NewReader(out.String()), frames)
	versions := map[string]float64{}
	for frame := range frames {
		if frame["method"] == "textDocument/publishDiagnostics" {
			p := frame["params"].(map[string]any)
			if len(p["diagnostics"].([]any)) != 1 {
				t.Fatalf("alias omitted diagnostic: %v", p)
			}
			versions[p["uri"].(string)] = p["version"].(float64)
		}
	}
	if versions[realURI] != 3 || versions[aliasURI] != 9 {
		t.Fatalf("wrong alias publication: %v", versions)
	}
}

func TestLSPConflictingAliasesClearStaleFindingsWithoutLosingBuffers(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": serverMain})
	path := filepath.Join(root, "src/main.can")
	alias := filepath.Join(root, "alias.can")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	s := newLSPServer()
	realURI, aliasURI := uriFromPath(path), uriFromPath(alias)
	s.open(realURI, serverMain, 1)
	s.published[realURI] = root
	var out bytes.Buffer
	serveLSPWithServer(s, bufio.NewReader(strings.NewReader(frame(didOpen(aliasURI, serverMain+"\n", 2)))), bufio.NewWriter(&out))
	if !strings.Contains(out.String(), "conflicting unsaved URI aliases") || !strings.Contains(out.String(), `"diagnostics":[]`) {
		t.Fatalf("ambiguous buffers left stale findings: %s", out.String())
	}
	if s.docs[realURI].text != serverMain || s.docs[aliasURI].text != serverMain+"\n" {
		t.Fatal("conflict resolution overwrote a buffer")
	}
	if len(s.published) != 0 {
		t.Fatalf("stale publication ownership retained: %v", s.published)
	}
	var clear bytes.Buffer
	s.close(bufio.NewWriter(&clear), aliasURI)
	if _, err := s.snapshot(realURI); err != nil {
		t.Fatalf("closing conflict did not recover analysis: %v", err)
	}
}

func TestLSPWatchedChangeRefreshesClosedProjectDiagnostics(t *testing.T) {
	broken := strings.Replace(serverMain, "call helper(seed)", "call missing_fn(seed)", 1)
	root := writeServerProject(t, map[string]string{"src/main.can": broken})
	path := filepath.Join(root, "src/main.can")
	uri := uriFromPath(path)
	s := newLSPServer()
	var previous bytes.Buffer
	s.diagnoseRoot(bufio.NewWriter(&previous), root, "")
	if s.published[uri] != root {
		t.Fatalf("closed-file fixture did not publish: %s", previous.String())
	}
	if err := os.WriteFile(path, []byte(serverMain), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	serveLSPWithServer(s, bufio.NewReader(strings.NewReader(frame(`{"jsonrpc":"2.0","method":"workspace/didChangeWatchedFiles","params":{"changes":[]}}`))), bufio.NewWriter(&out))
	if !strings.Contains(out.String(), `"diagnostics":[]`) || !strings.Contains(out.String(), uri) {
		t.Fatalf("closed-project findings remained stale: %s", out.String())
	}
}

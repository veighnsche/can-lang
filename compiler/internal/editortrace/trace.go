// Package editortrace provides opt-in, bounded editor pipeline timings. No
// source, paths or response payloads are recorded: rows carry only the
// request sequence, method, request ID, document version, stage name and an
// inclusive wall-clock duration in nanoseconds. The sink is a caller-chosen
// file, never stdout, so traced runs still speak valid Content-Length
// JSON-RPC.
//
// Durations use the process's monotonic clock and are inclusive: a parent
// stage contains its children's intervals, so nested rows must never be
// summed. Row counts per (sequence, stage) show how often that stage ran;
// compare counts with inclusive durations to distinguish removed waste
// from skipped work. The retained attribution counters are the
// catalogue-inventory-clone, collection-operation, snapshot and
// resolve-build stages; request rows delimit each decoded frame.
package editortrace

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const Limit = 32 << 20

type recorder struct {
	mu       sync.Mutex
	file     *os.File
	bytes    int
	sequence int64
	method   string
	id       json.RawMessage
	version  int64
	err      error
}

var active atomic.Pointer[recorder]

// Open refuses to overwrite a sink. Call only at server startup, and always
// close it after serving. Failures and reaching the cap invalidate the trace.
func Open(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	r := &recorder{file: file}
	if !active.CompareAndSwap(nil, r) {
		file.Close()
		return nil, fmt.Errorf("editor trace already active")
	}
	return func() error {
		active.CompareAndSwap(r, nil)
		r.mu.Lock()
		defer r.mu.Unlock()
		closeErr := r.file.Close()
		if r.err != nil {
			return r.err
		}
		return closeErr
	}, nil
}

// Request identifies decoded frames without capturing document text. The
// server dispatches synchronously, so child stages belong to this request.
// Call it immediately after frame decoding so the request interval covers
// all server work; when the document version is decoded separately, pass 0
// here and attribute it with SetRequestVersion before dispatch. Every
// Request must be ended exactly once on all paths, including malformed
// params, so the trace holds one request row per decoded frame.
func Request(method string, id *json.RawMessage, version int64) func() {
	r := active.Load()
	if r == nil {
		return func() {}
	}
	r.mu.Lock()
	r.sequence++
	r.method, r.version, r.id = method, version, nil
	if id != nil {
		r.id = append(json.RawMessage(nil), (*id)...)
	}
	r.mu.Unlock()
	return Stage("request")
}

// SetRequestVersion attributes the document version to the current request
// after its timing has started. It is a no-op when tracing is disabled.
func SetRequestVersion(version int64) {
	r := active.Load()
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.version = version
}

func Stage(name string) func() {
	r := active.Load()
	if r == nil {
		return func() {}
	}
	start := time.Now()
	return func() {
		duration := time.Since(start).Nanoseconds()
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.err != nil {
			return
		}
		row := struct {
			Sequence int64           `json:"sequence"`
			Method   string          `json:"method"`
			ID       json.RawMessage `json:"id,omitempty"`
			Version  int64           `json:"version"`
			Stage    string          `json:"stage"`
			Duration int64           `json:"duration_ns"`
		}{r.sequence, r.method, r.id, r.version, name, duration}
		data, err := json.Marshal(row)
		if err != nil {
			r.err = err
			return
		}
		data = append(data, '\n')
		if r.bytes+len(data) > Limit {
			r.err = fmt.Errorf("editor trace exceeds %d byte cap", Limit)
			return
		}
		_, r.err = r.file.Write(data)
		r.bytes += len(data)
	}
}

func Enabled() bool { return active.Load() != nil }

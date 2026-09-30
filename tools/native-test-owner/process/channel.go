package process

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// statusPayload renders the content-bound acknowledgment a well-behaved
// child returns: it names the operation and the digest of the exact
// environment bytes the owner offered, so acceptance is bound to what
// was offered. A child that never read fd 3 cannot mint it.
func statusPayload(opID string, envDigest []byte) string {
	return "ACK:" + opID + ":" + hex.EncodeToString(envDigest)
}

// snapshotEnv renders the exact environment bytes offered on fd 3: one
// JSON object with keys in ascending order. The rendering is the
// contract; children observe exactly these bytes.
func snapshotEnv(env map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(env))
	for _, k := range keys {
		ordered[k] = env[k]
	}
	raw, err := json.Marshal(ordered)
	if err != nil {
		return nil, fmt.Errorf("process: snapshot environment: %w", err)
	}
	// json.Marshal on a map already sorts keys, but the explicit sort
	// above keeps the contract obvious at the call site.
	return raw, nil
}

// encodeStatusFrame renders one status-channel frame carrying payload.
// Tests embed the result in child scripts via OctalEscape.
func encodeStatusFrame(payload string) ([]byte, error) {
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, MaxStatusPayload)
	if err := enc.Encode(codec.KindEvent, []byte(payload)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// OctalEscape renders raw as a sh printf '%b' argument: every byte
// becomes a three-digit octal escape, so no byte value (NUL, newline,
// percent) can disturb the shell or printf format parsing.
func OctalEscape(raw []byte) string {
	var out bytes.Buffer
	for _, b := range raw {
		fmt.Fprintf(&out, "\\%03o", b)
	}
	return out.String()
}

// envDigest returns the SHA-256 of the offered environment snapshot.
func envDigest(snapshot []byte) []byte {
	sum := sha256.Sum256(snapshot)
	return sum[:]
}

// offerResult is the writer-exit fact: how many snapshot bytes the
// kernel accepted and the delivery outcome.
type offerResult struct {
	wrote int
	err   error
}

// offerEnv streams the snapshot to the child's fd 3 and closes the
// write end, reporting the outcome once. A child that exits without
// reading closes the read end, which surfaces here as EPIPE: a fact,
// never a silent success.
func offerEnv(w io.WriteCloser, snapshot []byte) <-chan offerResult {
	out := make(chan offerResult, 1)
	go func() {
		n, err := w.Write(snapshot)
		cerr := w.Close()
		if err == nil {
			err = cerr
		}
		out <- offerResult{wrote: n, err: err}
	}()
	return out
}

// statusResult carries one decoded status frame or the terminal read
// outcome. Exactly one value is sent per collection stage.
type statusResult struct {
	payload   []byte
	malformed error // non-nil when bytes failed frame validation
	eof       bool  // clean end of stream with no further bytes
}

// collectStatus decodes the child's status stream: exactly one ack
// frame naming the expected payload, then clean EOF. It sends the ack
// outcome first and the EOF outcome second; the caller applies its own
// bounded waits to each stage so delayed EOF is observable.
func collectStatus(r io.Reader, maxPayload int) <-chan statusResult {
	out := make(chan statusResult, 2)
	go func() {
		dec := codec.NewDecoder(r, maxPayload)
		frame, err := dec.Decode()
		if err != nil {
			if err == io.EOF {
				out <- statusResult{eof: true}
				return
			}
			out <- statusResult{malformed: err}
			return
		}
		out <- statusResult{payload: frame.Payload}
		if _, err := dec.Decode(); err != nil {
			if err == io.EOF {
				out <- statusResult{eof: true}
				return
			}
			out <- statusResult{malformed: err}
			return
		}
		// A second frame where EOF belongs is a malformed stream: the
		// child said more than the one permitted acknowledgment.
		out <- statusResult{malformed: fmt.Errorf("%w: trailing frame after ack", codec.ErrBadEnvelope)}
	}()
	return out
}

// awaitStatus waits for one collector stage up to timeout.
func awaitStatus(ch <-chan statusResult, timeout time.Duration) (statusResult, bool) {
	if timeout <= 0 {
		timeout = DefaultStatusTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res, ok := <-ch:
		if !ok {
			return statusResult{}, false
		}
		return res, true
	case <-timer.C:
		return statusResult{}, false
	}
}

// Framed owner<->observer wire protocol.
//
// The actual observation channel between the N/T owner and the observer
// child process: newline-delimited JSON over the child's stdin/stdout
// pipes. The owner sends one request per line and reads exactly one
// response line per request. Decoding is strict (unknown fields reject)
// and bounded (oversize lines reject); anything the child cannot parse
// is a corrupt channel, and the child exits fail-closed so the owner
// treats the interval as observer-lost rather than inventing facts.
package browser_observer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

// Wire operations. Mutations journal before the response; reads never
// journal. interrupt/resume suspend and resume observation; shutdown
// ends the child cleanly after its final response.
const (
	OpSample          = "sample"
	OpEndTick         = "end-tick"
	OpNoteDriverDeath = "note-driver-death"
	OpPublishCorr     = "publish-correction"
	OpMarkUnknown     = "mark-unknown"
	OpSealDisposal    = "seal-disposal"
	OpIntervalFacts   = "interval-facts"
	OpInterrupt       = "interrupt"
	OpResume          = "resume"
	OpShutdown        = "shutdown"
)

// MaxWireLine bounds one request or response line. Facts are small
// digests; anything larger is a corrupt or hostile frame.
const MaxWireLine = 1 << 20

// wireRequest is one owner->observer command. Owner, scope, and launch
// are bound at child boot and never cross per-request: the child owns
// exactly one launch, so a request cannot name a foreign interval.
type wireRequest struct {
	Op      string `json:"op"`
	Token   string `json:"token,omitempty"`
	Channel string `json:"channel,omitempty"`
	State   string `json:"state,omitempty"`
	Tick    int    `json:"tick,omitempty"`
	After   string `json:"after,omitempty"`
}

// wireResponse is one observer->owner reply. On ok=false, Layer and Code
// carry the closed failure vocabulary and Message is diagnostic only.
type wireResponse struct {
	Ok      bool            `json:"ok"`
	Body    json.RawMessage `json:"body,omitempty"`
	Layer   string          `json:"layer,omitempty"`
	Code    string          `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
}

// readyReport is the child's first stdout line: the launch attestation
// plus the raw launch token. The token crosses exactly here, mirroring
// TokenForTest, so later commands authenticate by lookup.
type readyReport struct {
	Ready       bool              `json:"ready"`
	Token       string            `json:"token"`
	Attestation LaunchAttestation `json:"attestation"`
}

func validOp(op string) bool {
	switch op {
	case OpSample, OpEndTick, OpNoteDriverDeath, OpPublishCorr,
		OpMarkUnknown, OpSealDisposal, OpIntervalFacts,
		OpInterrupt, OpResume, OpShutdown:
		return true
	default:
		return false
	}
}

// journaledOp reports whether an accepted op mutates the interval and
// must be durably published before the response.
func journaledOp(op string) bool {
	switch op {
	case OpSample, OpEndTick, OpNoteDriverDeath, OpPublishCorr,
		OpMarkUnknown, OpSealDisposal, OpInterrupt, OpResume:
		return true
	default:
		return false
	}
}

func encodeRequest(req wireRequest) ([]byte, error) {
	if !validOp(req.Op) {
		return nil, fmt.Errorf("browser_observer: unknown wire op %q", req.Op)
	}
	line, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browser_observer: encode request: %w", err)
	}
	if len(line)+1 > MaxWireLine {
		return nil, fmt.Errorf("browser_observer: request exceeds the wire bound")
	}
	return append(line, '\n'), nil
}

func decodeRequest(line []byte) (wireRequest, error) {
	var req wireRequest
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return wireRequest{}, fmt.Errorf("browser_observer: malformed wire request: %w", err)
	}
	if dec.More() {
		return wireRequest{}, fmt.Errorf("browser_observer: malformed wire request: trailing data")
	}
	if !validOp(req.Op) {
		return wireRequest{}, fmt.Errorf("browser_observer: unknown wire op %q", req.Op)
	}
	return req, nil
}

func encodeResponse(resp wireResponse) ([]byte, error) {
	line, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("browser_observer: encode response: %w", err)
	}
	if len(line)+1 > MaxWireLine {
		return nil, fmt.Errorf("browser_observer: response exceeds the wire bound")
	}
	return append(line, '\n'), nil
}

func decodeResponse(line []byte) (wireResponse, error) {
	var resp wireResponse
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resp); err != nil {
		return wireResponse{}, fmt.Errorf("browser_observer: malformed wire response: %w", err)
	}
	if dec.More() {
		return wireResponse{}, fmt.Errorf("browser_observer: malformed wire response: trailing data")
	}
	return resp, nil
}

func okResponse(body any) (wireResponse, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return wireResponse{}, fmt.Errorf("browser_observer: encode facts: %w", err)
	}
	return wireResponse{Ok: true, Body: raw}, nil
}

func errResponse(err error) wireResponse {
	if oerr, ok := err.(*ObserverError); ok {
		msg := "observer refused"
		if oerr.Err != nil {
			msg = oerr.Err.Error()
		}
		return wireResponse{Ok: false, Layer: string(oerr.Layer), Code: oerr.Code, Message: msg}
	}
	return wireResponse{Ok: false, Layer: string(LayerInterval), Code: CodeUnknownEffect, Message: err.Error()}
}

// responseError converts a wire error reply back into the layered
// contract. Unknown layer/code pairs never occur on this channel: the
// child only emits the closed vocabulary, so anything else is channel
// corruption and fails closed as observer-lost.
func responseError(resp wireResponse) error {
	layer := Layer(resp.Layer)
	code := resp.Code
	if want, ok := LayerOfCode(code); !ok || want != layer {
		return obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: corrupt wire error", ErrDenied))
	}
	return &ObserverError{Layer: layer, Code: code, Err: fmt.Errorf("%s", resp.Message)}
}

// readLine reads one bounded line. An overlong line is channel
// corruption; EOF propagates to the caller, which treats a mid-stream
// EOF as observer loss. A final line without its newline still reads
// (bufio semantics) and then fails strict frame decoding fail-closed.
func readLine(r *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, chunk...)
		if len(line)+1 > MaxWireLine {
			return nil, fmt.Errorf("browser_observer: wire line exceeds the bound")
		}
		if !isPrefix {
			return line, nil
		}
	}
}

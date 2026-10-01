// Wire protocol unit controls: framing, strictness, and bounds. No
// processes here; the process controls live in process_test.go.
package browser_observer

import (
	"bufio"
	"strings"
	"testing"
)

func TestWireRoundTrip(t *testing.T) {
	line, err := encodeRequest(wireRequest{Op: OpSample, Token: "tok", Channel: "window", State: StateSeen})
	if err != nil {
		t.Fatalf("encodeRequest: %v", err)
	}
	req, err := decodeRequest(line[:len(line)-1])
	if err != nil {
		t.Fatalf("decodeRequest: %v", err)
	}
	if req.Op != OpSample || req.Token != "tok" || req.Channel != "window" || req.State != StateSeen {
		t.Fatalf("wire request did not survive: %+v", req)
	}
	resp, err := okResponse(TickFacts{LaunchID: "l", Tick: 3, Closed: true})
	if err != nil {
		t.Fatalf("okResponse: %v", err)
	}
	raw, err := encodeResponse(resp)
	if err != nil {
		t.Fatalf("encodeResponse: %v", err)
	}
	back, err := decodeResponse(raw[:len(raw)-1])
	if err != nil {
		t.Fatalf("decodeResponse: %v", err)
	}
	if !back.Ok || len(back.Body) == 0 {
		t.Fatalf("wire response did not survive: %+v", back)
	}
}

func TestWireRejectsUnknownFieldsAndOps(t *testing.T) {
	for _, line := range []string{
		`{"op":"sample","token":"t","channel":"window","state":"seen","extra":1}`,
		`{"op":"reboot-observer"}`,
		`{"op":"sample",}`,
		`not json`,
		`{"op":"sample"} {"op":"sample"}`,
		`{"op":7}`,
		``,
	} {
		if _, err := decodeRequest([]byte(line)); err == nil {
			t.Fatalf("decodeRequest accepted %q", line)
		}
	}
	for _, line := range []string{
		`{"ok":true,"body":{},"extra":1}`,
		`[1,2]`,
		``,
	} {
		if _, err := decodeResponse([]byte(line)); err == nil {
			t.Fatalf("decodeResponse accepted %q", line)
		}
	}
	if _, err := encodeRequest(wireRequest{Op: "reboot-observer"}); err == nil {
		t.Fatal("encodeRequest accepted an unknown op")
	}
}

func TestWireBoundEnforced(t *testing.T) {
	big := strings.Repeat("x", MaxWireLine)
	if _, err := encodeRequest(wireRequest{Op: OpSample, Token: big}); err == nil {
		t.Fatal("oversize request encoded")
	}
	r := bufio.NewReader(strings.NewReader(big + "\n"))
	if _, err := readLine(r); err == nil {
		t.Fatal("oversize line read")
	}
	// A final line without its newline still reads per bufio
	// semantics; strict frame decoding then rejects it fail-closed.
	r = bufio.NewReader(strings.NewReader(`{"op":"sample"`))
	line, err := readLine(r)
	if err != nil {
		t.Fatalf("readLine: %v", err)
	}
	if _, derr := decodeRequest(line); derr == nil {
		t.Fatal("torn frame decoded")
	}
	if _, err := readLine(r); err == nil {
		t.Fatal("expected EOF after the final line")
	}
}

func TestWireErrorRoundTrip(t *testing.T) {
	for _, code := range Codes {
		layer, _ := LayerOfCode(code)
		resp := errResponse(&ObserverError{Layer: layer, Code: code, Err: ErrDenied})
		if resp.Ok || resp.Layer != string(layer) || resp.Code != code {
			t.Fatalf("error not transported: %+v", resp)
		}
		back := responseError(resp)
		requireObsErr(t, back, layer, code)
	}
	// A corrupt wire error fails closed as observer loss, never as a
	// forged verdict layer.
	for _, resp := range []wireResponse{
		{Ok: false, Layer: "observer", Code: "no-such-code"},
		{Ok: false, Layer: "kernel", Code: CodeObserverLost},
		{Ok: false, Layer: "admission", Code: CodeObserverLost},
		{Ok: false},
	} {
		requireObsErr(t, responseError(resp), LayerObserver, CodeObserverLost)
	}
	// Non-layered child failures map into the closed vocabulary.
	resp := errResponse(assertAnError())
	if resp.Ok || resp.Code != CodeUnknownEffect || resp.Layer != string(LayerInterval) {
		t.Fatalf("unlayered error escaped the vocabulary: %+v", resp)
	}
}

func assertAnError() error { return errTestUnlayered }

var errTestUnlayered = errorString("unlayered test failure")

type errorString string

func (e errorString) Error() string { return string(e) }

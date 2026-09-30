package dispatch

import (
	"bytes"
	"errors"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

func TestChannelRoundTrip(t *testing.T) {
	kinds := []struct {
		name string
		kind byte
	}{
		{"request", codec.KindRequest},
		{"reply", codec.KindReply},
		{"event", codec.KindEvent},
	}
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			c, err := NewChannel(k.kind, 0, 0)
			if err != nil {
				t.Fatalf("NewChannel: %v", err)
			}
			if c.Kind() != k.kind {
				t.Fatalf("Kind = %#02x, want %#02x", c.Kind(), k.kind)
			}
			payloads := []string{`{"a":1}`, "", `{"b":[1,2,3]}`}
			for _, p := range payloads {
				if err := c.Send([]byte(p)); err != nil {
					t.Fatalf("Send: %v", err)
				}
			}
			if got := c.Pending(); got != len(payloads) {
				t.Fatalf("Pending = %d, want %d", got, len(payloads))
			}
			for i, want := range payloads {
				f, err := c.Recv()
				if err != nil {
					t.Fatalf("Recv %d: %v", i, err)
				}
				if f.Kind != k.kind || f.Seq != uint32(i) || string(f.Payload) != want {
					t.Fatalf("frame %d = kind %#02x seq %d payload %q", i, f.Kind, f.Seq, f.Payload)
				}
			}
			if _, err := c.Recv(); !errors.Is(err, ErrEmpty) {
				t.Fatalf("empty Recv err = %v, want ErrEmpty", err)
			}
		})
	}
}

func TestChannelBounds(t *testing.T) {
	c, err := NewChannel(codec.KindRequest, 16, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Send(bytes.Repeat([]byte("a"), 16)); err != nil {
		t.Fatalf("exactly-max send: %v", err)
	}
	if err := c.Send(bytes.Repeat([]byte("a"), 17)); !errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("oversize send err = %v, want ErrTooLarge", err)
	}
	if got := c.Pending(); got != 1 {
		t.Fatalf("Pending after rejected send = %d, want 1", got)
	}
	if err := c.Send([]byte("b")); err != nil {
		t.Fatalf("second send: %v", err)
	}
	if err := c.Send([]byte("c")); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("full queue err = %v, want ErrQueueFull", err)
	}
	if _, err := c.Recv(); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if err := c.Send([]byte("c")); err != nil {
		t.Fatalf("send after drain: %v", err)
	}

	invalid := []struct {
		name       string
		kind       byte
		maxPayload int
		maxQueue   int
	}{
		{"bad kind", 0x09, 0, 0},
		{"payload over cap", codec.KindRequest, MaxPayloadCap + 1, 0},
		{"queue over cap", codec.KindRequest, 0, MaxQueueCap + 1},
	}
	for _, c := range invalid {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewChannel(c.kind, c.maxPayload, c.maxQueue); !errors.Is(err, ErrInvalid) {
				t.Fatalf("NewChannel err = %v, want ErrInvalid", err)
			}
		})
	}
	if _, err := NewSet(0, MaxQueueCap+1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewSet err = %v, want ErrInvalid", err)
	}
	set, err := NewSet(0, 0)
	if err != nil {
		t.Fatalf("NewSet: %v", err)
	}
	if set.Request.Kind() != codec.KindRequest || set.Reply.Kind() != codec.KindReply || set.Event.Kind() != codec.KindEvent {
		t.Fatal("channel set kinds wrong")
	}
}

func TestChannelClose(t *testing.T) {
	c, err := NewChannel(codec.KindReply, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Send([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("double Close: %v", err)
	}
	if err := c.Send([]byte(`{}`)); !errors.Is(err, ErrClosed) {
		t.Fatalf("send after close err = %v, want ErrClosed", err)
	}
	if _, err := c.Recv(); err != nil {
		t.Fatalf("drain after close: %v", err)
	}
	if _, err := c.Recv(); !errors.Is(err, ErrClosed) {
		t.Fatalf("empty closed Recv err = %v, want ErrClosed", err)
	}
}

func requestEnv() codec.Envelope {
	return codec.Envelope{
		SchemaVersion: "1", RunID: "run-1", OperationID: "op-1",
		OwnerGrant: "ng1-test", Operation: "read",
		ArgumentsDigest: "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		DeadlineMs:      codec.ClockTime{Clock: "wall-utc", Ms: 0},
	}
}

func TestChannelEnvelopes(t *testing.T) {
	req, err := NewChannel(codec.KindRequest, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.SendEnvelope(requestEnv()); err != nil {
		t.Fatalf("SendEnvelope: %v", err)
	}
	got, err := req.RecvEnvelope()
	if err != nil {
		t.Fatalf("RecvEnvelope: %v", err)
	}
	if got.RunID != "run-1" || got.OperationID != "op-1" || got.Operation != "read" {
		t.Fatalf("envelope = %+v", got)
	}

	rep, err := NewChannel(codec.KindReply, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	res := codec.Envelope{SchemaVersion: "1", RunID: "run-1", OperationID: "op-1",
		Outcome: codec.OutcomeCompleted, Kind: codec.KindOK, Facts: map[string]any{"n": float64(1)}}
	if err := rep.SendEnvelope(res); err != nil {
		t.Fatalf("reply SendEnvelope: %v", err)
	}
	gotRes, err := rep.RecvEnvelope()
	if err != nil {
		t.Fatalf("reply RecvEnvelope: %v", err)
	}
	if gotRes.Outcome != codec.OutcomeCompleted || gotRes.Facts["n"] != float64(1) {
		t.Fatalf("result = %+v", gotRes)
	}

	// A malformed envelope is never emitted.
	bad := requestEnv()
	bad.Operation = ""
	if err := req.SendEnvelope(bad); !errors.Is(err, codec.ErrBadEnvelope) {
		t.Fatalf("malformed send err = %v, want ErrBadEnvelope", err)
	}
	if got := req.Pending(); got != 0 {
		t.Fatalf("Pending after malformed send = %d, want 0", got)
	}
	// Unknown fields smuggled into the payload are rejected on receipt.
	if err := req.Send([]byte(`{"schema_version":"1","run_id":"run-1","operation_id":"op-1","owner_grant":"g","operation":"read","arguments_digest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","deadline_ms":{"clock":"wall-utc","ms":0},"smuggled":1}`)); err != nil {
		t.Fatalf("smuggled send: %v", err)
	}
	if _, err := req.RecvEnvelope(); !errors.Is(err, codec.ErrBadEnvelope) {
		t.Fatalf("smuggled recv err = %v, want ErrBadEnvelope", err)
	}
	// Unmarshalable facts fail before anything is queued.
	unmarshalable := res
	unmarshalable.Facts = map[string]any{"fn": func() {}}
	if err := rep.SendEnvelope(unmarshalable); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unmarshalable send err = %v, want ErrInvalid", err)
	}
	if got := rep.Pending(); got != 0 {
		t.Fatalf("Pending after unmarshalable send = %d, want 0", got)
	}
}

// TestForgeControls proves subject stdout presented as supervisor input is
// rejected: bare JSON carries no frame header, and forged frames fail
// sequence and shape checks.
func TestForgeControls(t *testing.T) {
	payload := []byte(`{"schema_version":"1","run_id":"run-1","operation_id":"op-1","outcome":"completed","kind":"ok"}`)
	cases := []struct {
		name  string
		input func() []byte
		want  error
	}{
		{"bare stdout json", func() []byte { return []byte(`{"outcome":"completed","kind":"ok"}`) }, codec.ErrBadMagic},
		{"bare envelope json", func() []byte { return payload }, codec.ErrBadMagic},
		{"forged future seq", func() []byte { return codec.EncodeFrame(codec.KindReply, 7, payload) }, codec.ErrGap},
		{"truncated header", func() []byte { return []byte{0x4e, 0x54, 0x01} }, codec.ErrTruncated},
		{"bad version", func() []byte { f := codec.EncodeFrame(codec.KindReply, 0, payload); f[2] = 0x09; return f }, codec.ErrBadVersion},
		{"bad kind", func() []byte { f := codec.EncodeFrame(codec.KindReply, 0, payload); f[3] = 0x09; return f }, codec.ErrBadKind},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ch, err := NewChannel(codec.KindReply, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			ch.buf.Write(c.input())
			if _, err := ch.dec.Decode(); !errors.Is(err, c.want) {
				t.Fatalf("decode err = %v, want %v", err, c.want)
			}
			if got := ch.Pending(); got != 0 {
				t.Fatalf("Pending = %d, want 0", got)
			}
			if _, err := ch.Recv(); !errors.Is(err, ErrEmpty) {
				t.Fatalf("Recv err = %v, want ErrEmpty", err)
			}
		})
	}
	t.Run("duplicate seq", func(t *testing.T) {
		ch, err := NewChannel(codec.KindReply, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		ch.buf.Write(codec.EncodeFrame(codec.KindReply, 0, payload))
		ch.buf.Write(codec.EncodeFrame(codec.KindReply, 0, payload))
		if _, err := ch.dec.Decode(); err != nil {
			t.Fatalf("first decode: %v", err)
		}
		if _, err := ch.dec.Decode(); !errors.Is(err, codec.ErrDuplicate) {
			t.Fatalf("duplicate decode err = %v, want ErrDuplicate", err)
		}
	})
}

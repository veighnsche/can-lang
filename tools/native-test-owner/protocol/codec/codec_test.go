package codec

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldenFrameBytes(t *testing.T) {
	got := EncodeFrame(KindRequest, 0, []byte(`{"a":1}`))
	want := "4e54010100000000000000077b2261223a317d"
	if hex.EncodeToString(got) != want {
		t.Fatalf("golden frame mismatch:\n got %x\nwant %s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, 0)
	payloads := []struct {
		kind    byte
		payload string
	}{
		{KindRequest, `{"schemaVersion":"1"}`},
		{KindReply, `{"schemaVersion":"1"}`},
		{KindEvent, `{"schemaVersion":"1"}`},
	}
	for _, p := range payloads {
		if err := enc.Encode(p.kind, []byte(p.payload)); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	dec := NewDecoder(&buf, 0)
	for i, p := range payloads {
		frame, err := dec.Decode()
		if err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
		if frame.Kind != p.kind || frame.Seq != uint32(i) || string(frame.Payload) != p.payload {
			t.Fatalf("frame %d = %+v, want kind %d seq %d", i, frame, p.kind, i)
		}
	}
	if _, err := dec.Decode(); err != io.EOF {
		t.Fatalf("clean stream end = %v, want io.EOF", err)
	}
}

func TestEncoderBounds(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, 8)
	if err := enc.Encode(KindRequest, []byte("12345678")); err != nil {
		t.Fatalf("exactly-max payload rejected: %v", err)
	}
	if err := enc.Encode(KindRequest, []byte("123456789")); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("over-max payload err = %v, want ErrTooLarge", err)
	}
	if err := enc.Encode(0x09, []byte("x")); !errors.Is(err, ErrBadKind) {
		t.Fatalf("bad kind err = %v, want ErrBadKind", err)
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  error
	}{
		{"empty-header", []byte{0x4e, 0x54, 0x01}, ErrTruncated},
		{"bad-kind", func() []byte { f := EncodeFrame(KindRequest, 0, []byte(`{}`)); f[3] = 0x09; return f }(), ErrBadKind},
		{"short-payload", EncodeFrame(KindRequest, 0, []byte(`{"a":1}`))[:len(EncodeFrame(KindRequest, 0, []byte(`{"a":1}`)))-2], ErrTruncated},
	}
	for _, c := range cases {
		dec := NewDecoder(bytes.NewReader(c.input), 0)
		if _, err := dec.Decode(); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestOversizeCheckedBeforeBuffering(t *testing.T) {
	// Declared length exceeds the bound but no body follows: the decoder must
	// report ErrTooLarge without attempting a body read.
	header := EncodeFrame(KindRequest, 0, nil)[:headerSize]
	header[8], header[9], header[10], header[11] = 0x00, 0x20, 0x00, 0x00
	dec := NewDecoder(bytes.NewReader(header), 1024)
	if _, err := dec.Decode(); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversize err = %v, want ErrTooLarge", err)
	}
}

func loadJSON(t *testing.T, name string, target any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatal(err)
	}
}

func TestFrameVectors(t *testing.T) {
	var valid []struct {
		Name   string   `json:"name"`
		Stream []string `json:"stream"`
		Want   []struct {
			Kind    byte   `json:"kind"`
			Seq     uint32 `json:"seq"`
			Payload string `json:"payload"`
		} `json:"want"`
	}
	loadJSON(t, "frames_valid.json", &valid)
	for _, v := range valid {
		var stream []byte
		for _, chunk := range v.Stream {
			raw, err := hex.DecodeString(chunk)
			if err != nil {
				t.Fatalf("%s: bad hex: %v", v.Name, err)
			}
			stream = append(stream, raw...)
		}
		dec := NewDecoder(bytes.NewReader(stream), 0)
		for i, w := range v.Want {
			frame, err := dec.Decode()
			if err != nil {
				t.Fatalf("%s frame %d: %v", v.Name, i, err)
			}
			if frame.Kind != w.Kind || frame.Seq != w.Seq || string(frame.Payload) != w.Payload {
				t.Fatalf("%s frame %d = kind %d seq %d payload %q", v.Name, i, frame.Kind, frame.Seq, frame.Payload)
			}
		}
		if _, err := dec.Decode(); err != io.EOF {
			t.Fatalf("%s: trailing bytes, final decode = %v", v.Name, err)
		}
	}
}

func TestFrameInvalidVectors(t *testing.T) {
	var invalid []struct {
		Name       string   `json:"name"`
		Stream     []string `json:"stream"`
		MaxPayload int      `json:"maxPayload"`
		WantError  string   `json:"wantError"`
	}
	loadJSON(t, "frames_invalid.json", &invalid)
	for _, v := range invalid {
		var stream []byte
		for _, chunk := range v.Stream {
			raw, err := hex.DecodeString(chunk)
			if err != nil {
				t.Fatalf("%s: bad hex: %v", v.Name, err)
			}
			stream = append(stream, raw...)
		}
		dec := NewDecoder(bytes.NewReader(stream), v.MaxPayload)
		var firstErr error
		for {
			_, err := dec.Decode()
			if err != nil {
				firstErr = err
				break
			}
		}
		if firstErr == nil {
			t.Fatalf("%s: invalid stream accepted", v.Name)
		}
		if !strings.Contains(firstErr.Error(), v.WantError) {
			t.Fatalf("%s: err %q lacks %q", v.Name, firstErr, v.WantError)
		}
	}
}

func TestEnvelopeVectors(t *testing.T) {
	var valid []struct {
		Name string         `json:"name"`
		Kind byte           `json:"kind"`
		JSON map[string]any `json:"json"`
	}
	loadJSON(t, "envelopes_valid.json", &valid)
	for _, v := range valid {
		raw, err := json.Marshal(v.JSON)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeEnvelope(raw, v.Kind); err != nil {
			t.Errorf("%s: valid envelope rejected: %v", v.Name, err)
		}
	}
	var invalid []struct {
		Name      string         `json:"name"`
		Kind      byte           `json:"kind"`
		JSON      map[string]any `json:"json"`
		WantError string         `json:"wantError"`
	}
	loadJSON(t, "envelopes_invalid.json", &invalid)
	for _, v := range invalid {
		raw, err := json.Marshal(v.JSON)
		if err != nil {
			t.Fatal(err)
		}
		_, err = DecodeEnvelope(raw, v.Kind)
		if err == nil {
			t.Errorf("%s: invalid envelope accepted", v.Name)
		} else if !strings.Contains(err.Error(), v.WantError) {
			t.Errorf("%s: err %q lacks %q", v.Name, err, v.WantError)
		}
	}
}

func TestSubjectStdoutCannotForgeMessage(t *testing.T) {
	// A bare JSON report of the kind a subject prints to stdout carries no
	// frame header and must be rejected as a supervisor message.
	dec := NewDecoder(strings.NewReader(`{"outcome":"completed","kind":"ok"}`), 0)
	if _, err := dec.Decode(); !errors.Is(err, ErrBadMagic) {
		t.Fatalf("raw stdout err = %v, want ErrBadMagic", err)
	}
}

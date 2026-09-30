package f1

import (
	"bytes"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// The binary vectors are fixed: lengths, anchors and full coverage are
// pinned so QF1 authorship can rely on them byte for byte.
func TestBinaryVectorsFixed(t *testing.T) {
	vecs := BinaryStdinVectors()
	if len(vecs) != 5 {
		t.Fatalf("len(vectors) = %d, want 5", len(vecs))
	}
	byName := map[string][]byte{}
	for _, v := range vecs {
		byName[v.Name] = v.Bytes
	}
	if got := byName["nul"]; !bytes.Equal(got, []byte{'o', 'k', 0, 'f', 0, 0, 'f', 0}) {
		t.Fatalf("nul = %q", got)
	}
	if got := byName["newlines"]; !bytes.Equal(got, []byte{'a', '\r', '\n', 'b', '\r', 'c', '\n', 'd', '\r'}) {
		t.Fatalf("newlines = %q", got)
	}
	if got := byName["percent"]; !strings.Contains(string(got), "%b") || !strings.Contains(string(got), "%n") {
		t.Fatalf("percent = %q, want format-hostile bytes", got)
	}
	if got := byName["invalid-utf8"]; !bytes.Equal(got, []byte{0xff, 0xfe, 0x80, 'o', 'k', 0xfa, 0xfb}) {
		t.Fatalf("invalid-utf8 = %x", got)
	}
	all := byName["all-bytes"]
	if len(all) != 256 {
		t.Fatalf("len(all-bytes) = %d, want 256", len(all))
	}
	var seen [256]int
	for _, b := range all {
		seen[b]++
	}
	for i, n := range seen {
		if n != 1 {
			t.Fatalf("byte %#02x appears %d times, want exactly once", i, n)
		}
	}
}

// Fixture accessors hand out copies: mutating a returned slice must not
// disturb the masters a later control observes.
func TestBinaryVectorsInert(t *testing.T) {
	first := BinaryStdinVectors()
	for _, v := range first {
		for i := range v.Bytes {
			v.Bytes[i] ^= 0xff
		}
	}
	second := BinaryStdinVectors()
	for _, v := range second {
		switch v.Name {
		case "nul":
			if !bytes.Equal(v.Bytes, []byte{'o', 'k', 0, 'f', 0, 0, 'f', 0}) {
				t.Fatalf("nul master mutated: %q", v.Bytes)
			}
		case "all-bytes":
			if len(v.Bytes) != 256 || v.Bytes[0] != 0 || v.Bytes[255] != 255 {
				t.Fatalf("all-bytes master mutated: len=%d", len(v.Bytes))
			}
		}
	}
	malformed := MalformedHeadVectors()
	for _, v := range malformed {
		for i := range v.Bytes {
			v.Bytes[i] ^= 0xff
		}
	}
	if !bytes.Equal(MalformedBadMagic[:2], []byte{0x5a, 0x5a}) {
		t.Fatalf("bad-magic master mutated: %x", MalformedBadMagic[:2])
	}
}

// Snapshot rendering matches the owner contract: sorted keys, no
// trailing newline, empty map renders as {}.
func TestRenderSnapshotShape(t *testing.T) {
	raw, err := RenderSnapshot(map[string]string{"B": "2", "A": "1"})
	if err != nil {
		t.Fatalf("RenderSnapshot: %v", err)
	}
	if string(raw) != `{"A":"1","B":"2"}` {
		t.Fatalf("RenderSnapshot = %q, want sorted compact JSON", raw)
	}
	raw, err = RenderSnapshot(map[string]string{})
	if err != nil {
		t.Fatalf("RenderSnapshot empty: %v", err)
	}
	if string(raw) != `{}` {
		t.Fatalf("RenderSnapshot empty = %q, want {}", raw)
	}
}

// The fixture ack frame decodes as one event frame carrying the
// content-bound payload shape ACK:<op>:<64 hex>.
func TestAckFrameDecodes(t *testing.T) {
	frame := MustAckFrameFor("shape1", map[string]string{"A": "1"})
	dec := codec.NewDecoder(bytes.NewReader(frame), 1<<20)
	got, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Kind != codec.KindEvent || got.Seq != 0 {
		t.Fatalf("frame = kind %#02x seq %d, want event seq 0", got.Kind, got.Seq)
	}
	payload := string(got.Payload)
	if !strings.HasPrefix(payload, "ACK:shape1:") || len(payload) != len("ACK:shape1:")+64 {
		t.Fatalf("payload = %q, want ACK:shape1:<64 hex>", payload)
	}
}

// Every malformed head vector fails frame validation on its own, before
// any child is involved.
func TestMalformedHeadVectorsDecodeAlone(t *testing.T) {
	for _, v := range MalformedHeadVectors() {
		t.Run(v.Name, func(t *testing.T) {
			dec := codec.NewDecoder(bytes.NewReader(v.Bytes), 64<<10)
			if _, err := dec.Decode(); err == nil {
				t.Fatalf("Decode(%s): want error, got nil", v.Name)
			}
		})
	}
	if _, err := codec.NewDecoder(bytes.NewReader(GapHeadFrame()), 64<<10).Decode(); err == nil {
		t.Fatalf("Decode(gap-head): want sequence-gap error, got nil")
	}
}

// Fixed inert fixture bytes for F1 descriptor delivery.
//
// Every vector below is fixed stimulus: binary stdin bytes (kept inert
// here because the P10 owner pins child stdin to /dev/null; QF1 decides
// how they drive a real subject), environment name/value edge cases,
// malformed status-channel frames, and unused-snapshot environments.
// Accessors return fresh copies so no control can mutate the masters.
package f1

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// Binary stdin stimuli. The owner wires child stdin to /dev/null, so
// these bytes are inert fixtures for QF1 scenario authorship; the
// TestBinaryStimuliDeliverExactly control proves the same hostile bytes
// traverse the shell-embedding path (process.OctalEscape + printf %b)
// to a fixture child without mangling.
var (
	// BinaryStdinNUL carries embedded NULs that truncate C-string readers.
	BinaryStdinNUL = []byte{'o', 'k', 0x00, 'f', 0x00, 0x00, 'f', 0x00}
	// BinaryStdinNewlines mixes lone CR, lone LF, CRLF and trailing CR.
	BinaryStdinNewlines = []byte{'a', '\r', '\n', 'b', '\r', 'c', '\n', 'd', '\r'}
	// BinaryStdinPercent carries printf-format-hostile bytes.
	BinaryStdinPercent = []byte{'%', 's', ' ', '%', 'b', ' ', '%', 'n', ' ', '%', 'x', ' ', '\\', ' ', '%', 'd'}
	// BinaryStdinInvalidUTF8 is not decodable as UTF-8 or JSON text.
	BinaryStdinInvalidUTF8 = []byte{0xff, 0xfe, 0x80, 'o', 'k', 0xfa, 0xfb}
	// BinaryStdinAllBytes holds every byte value 0x00..0xff exactly once.
	BinaryStdinAllBytes = allBytes()
)

// BinaryStdinVectors names every binary stdin stimulus in a fixed order.
func BinaryStdinVectors() []struct {
	Name  string
	Bytes []byte
} {
	vecs := []struct {
		Name  string
		Bytes []byte
	}{
		{"nul", BinaryStdinNUL},
		{"newlines", BinaryStdinNewlines},
		{"percent", BinaryStdinPercent},
		{"invalid-utf8", BinaryStdinInvalidUTF8},
		{"all-bytes", BinaryStdinAllBytes},
	}
	out := make([]struct {
		Name  string
		Bytes []byte
	}, len(vecs))
	for i, v := range vecs {
		out[i].Name = v.Name
		out[i].Bytes = append([]byte(nil), v.Bytes...)
	}
	return out
}

func allBytes() []byte {
	out := make([]byte, 256)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

// Environment name edge cases.
var (
	// UnicodeEnvNames must round-trip through the snapshot exactly.
	UnicodeEnvNames = []string{"ÜNICODE_NAME", "名前_KEY", "emoji_🎯_key"}
	// EmptyValueKey carries an empty value; the entry stays present.
	EmptyValueKey = "F1_EMPTY_VALUE"
	// MissingName is absent from every fixture environment; controls
	// assert its absence from the observed snapshot.
	MissingName = "F1_MISSING_NAME"
	// HostileEnvKeys must each be rejected at spawn with
	// process.ErrInvalid: empty, '='-bearing, newline-bearing, NUL-bearing.
	HostileEnvKeys = []string{"", "HAS=EQUALS", "HAS\nNEWLINE", "HAS\x00NUL"}
)

// UnusedSnapshotEnv is a small environment for the unused-snapshot
// control: offered on fd 3 but never consumed by the child.
func UnusedSnapshotEnv() map[string]string {
	return map[string]string{"F1_UNUSED_A": "1", "F1_UNUSED_B": "two"}
}

// RenderSnapshot renders the exact environment bytes the owner offers
// on fd 3: one JSON object with keys in ascending order (encoding/json
// sorts map keys, matching the owner's contract byte for byte).
func RenderSnapshot(env map[string]string) ([]byte, error) {
	raw, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("f1: render snapshot: %w", err)
	}
	return raw, nil
}

// MustRenderSnapshot renders env or panics; envs here are static.
func MustRenderSnapshot(env map[string]string) []byte {
	raw, err := RenderSnapshot(env)
	if err != nil {
		panic(err)
	}
	return raw
}

// SnapshotDigest returns the SHA-256 of rendered snapshot bytes, in the
// hex form carried by the content-bound ack payload.
func SnapshotDigest(snapshot []byte) string {
	sum := sha256.Sum256(snapshot)
	return hex.EncodeToString(sum[:])
}

// AckPayload renders the content-bound acknowledgment naming opID and
// the digest of the exact offered bytes.
func AckPayload(opID string, snapshot []byte) string {
	return "ACK:" + opID + ":" + SnapshotDigest(snapshot)
}

// AckFrameFor renders the exact status-channel frame a well-behaved
// child emits after reading fd 3: the content-bound ack in one event
// frame. TestPositiveDrainAckEOFSeparated checks it byte-for-byte
// against Owner.ExpectedAck, proving the fixture matches the owner.
func AckFrameFor(opID string, env map[string]string) ([]byte, error) {
	snapshot, err := RenderSnapshot(env)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, process.MaxStatusPayload)
	if err := enc.Encode(codec.KindEvent, []byte(AckPayload(opID, snapshot))); err != nil {
		return nil, fmt.Errorf("f1: encode ack frame: %w", err)
	}
	return buf.Bytes(), nil
}

// MustAckFrameFor renders the ack frame or panics; inputs are static.
func MustAckFrameFor(opID string, env map[string]string) []byte {
	frame, err := AckFrameFor(opID, env)
	if err != nil {
		panic(err)
	}
	return frame
}

// Malformed status-channel vectors. Each must set Facts.Malformed and
// never Facts.Accepted when it leads the stream; trailer-position
// vectors (correct ack first, then the vector) must keep Accepted while
// still setting Malformed.
var (
	// MalformedBadMagic flips the NT magic.
	MalformedBadMagic = []byte{0x5a, 0x5a, 0x01, 0x03, 0, 0, 0, 0, 0, 0, 0, 1, 'x'}
	// MalformedBadVersion bumps the frame version.
	MalformedBadVersion = []byte{0x4e, 0x54, 0x02, 0x03, 0, 0, 0, 0, 0, 0, 0, 1, 'x'}
	// MalformedBadKind uses an unknown frame kind.
	MalformedBadKind = []byte{0x4e, 0x54, 0x01, 0x09, 0, 0, 0, 0, 0, 0, 0, 1, 'x'}
	// MalformedTruncatedHeader ends mid-header.
	MalformedTruncatedHeader = []byte{0x4e, 0x54, 0x01, 0x03, 0}
	// MalformedTruncatedPayload carries a 100-byte header with 3 bytes.
	MalformedTruncatedPayload = append(
		[]byte{0x4e, 0x54, 0x01, 0x03, 0, 0, 0, 0, 0, 0, 0, 100},
		'x', 'y', 'z',
	)
	// MalformedOversize claims one byte more than the status bound.
	MalformedOversize = oversizeHeader()
)

// MalformedHeadVectors names every stream-leading malformed vector.
func MalformedHeadVectors() []struct {
	Name  string
	Bytes []byte
} {
	vecs := []struct {
		Name  string
		Bytes []byte
	}{
		{"bad-magic", MalformedBadMagic},
		{"bad-version", MalformedBadVersion},
		{"bad-kind", MalformedBadKind},
		{"truncated-header", MalformedTruncatedHeader},
		{"truncated-payload", MalformedTruncatedPayload},
		{"oversize-length", MalformedOversize},
	}
	out := make([]struct {
		Name  string
		Bytes []byte
	}, len(vecs))
	for i, v := range vecs {
		out[i].Name = v.Name
		out[i].Bytes = append([]byte(nil), v.Bytes...)
	}
	return out
}

func oversizeHeader() []byte {
	header := []byte{0x4e, 0x54, 0x01, 0x03, 0, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[8:12], uint32(process.MaxStatusPayload+1))
	return header
}

// SequenceFrame renders one event frame with an explicit sequence for
// negative controls the owner's encoder would never emit.
func SequenceFrame(seq uint32, payload []byte) []byte {
	return codec.EncodeFrame(codec.KindEvent, seq, append([]byte(nil), payload...))
}

// GapHeadFrame leads a stream with sequence 5 where 0 belongs.
func GapHeadFrame() []byte {
	return SequenceFrame(5, []byte("ACK:gap:late"))
}

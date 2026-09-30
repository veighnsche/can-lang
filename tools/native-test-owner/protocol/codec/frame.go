package codec

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Wire constants for the native-test frame header.
const (
	Magic0      = 0x4E // 'N'
	Magic1      = 0x54 // 'T'
	Version     = 0x01
	headerSize  = 12
	MaxPayload  = 1 << 20 // 1 MiB default payload bound (case evidence stream ceiling).
	MaxSequence = 1 << 30 // Finite sequence space; wrap-around is a protocol error.
)

// Frame kinds.
const (
	KindRequest = 0x01
	KindReply   = 0x02
	KindEvent   = 0x03
)

// Typed decode failures. Callers must treat every one as a broken channel,
// never as an empty successful report.
var (
	ErrBadMagic     = errors.New("codec: bad frame magic")
	ErrBadVersion   = errors.New("codec: unknown frame version")
	ErrBadKind      = errors.New("codec: unknown frame kind")
	ErrTooLarge     = errors.New("codec: frame exceeds payload bound")
	ErrTruncated    = errors.New("codec: truncated frame")
	ErrDuplicate    = errors.New("codec: duplicate sequence")
	ErrGap          = errors.New("codec: sequence gap")
	ErrSeqExhausted = errors.New("codec: sequence space exhausted")
	ErrBadEnvelope  = errors.New("codec: malformed envelope")
)

// Frame is one decoded wire frame. Payload is the raw JSON envelope bytes;
// use DecodeEnvelope for typed validation.
type Frame struct {
	Kind    byte
	Seq     uint32
	Payload []byte
}

// Encoder writes length-prefixed frames with strictly increasing sequences.
type Encoder struct {
	w       io.Writer
	max     uint32
	next    uint32
	started bool
}

// NewEncoder returns an Encoder writing to w with the given payload bound.
// A non-positive max selects MaxPayload.
func NewEncoder(w io.Writer, max int) *Encoder {
	if max <= 0 {
		max = MaxPayload
	}
	return &Encoder{w: w, max: uint32(max)}
}

// Encode writes one frame. kind must be KindRequest, KindReply or KindEvent
// and payload must fit the bound; violations are Encode-time errors so a
// malformed frame can never be emitted by this implementation.
func (e *Encoder) Encode(kind byte, payload []byte) error {
	switch kind {
	case KindRequest, KindReply, KindEvent:
	default:
		return fmt.Errorf("%w: %#02x", ErrBadKind, kind)
	}
	if len(payload) > int(e.max) {
		return fmt.Errorf("%w: %d > %d", ErrTooLarge, len(payload), e.max)
	}
	if e.started && e.next >= MaxSequence {
		return ErrSeqExhausted
	}
	seq := e.next
	if !e.started {
		seq = 0
		e.started = true
	}
	header := []byte{Magic0, Magic1, Version, kind, 0, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[4:8], seq)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(payload)))
	if _, err := e.w.Write(header); err != nil {
		return err
	}
	if _, err := e.w.Write(payload); err != nil {
		return err
	}
	e.next = seq + 1
	return nil
}

// Decoder reads length-prefixed frames, enforcing bounds before buffering
// and strict per-channel sequence order.
type Decoder struct {
	r        io.Reader
	max      uint32
	expected uint32
}

// NewDecoder returns a Decoder reading from r with the given payload bound.
// A non-positive max selects MaxPayload.
func NewDecoder(r io.Reader, max int) *Decoder {
	if max <= 0 {
		max = MaxPayload
	}
	return &Decoder{r: r, max: uint32(max)}
}

// Decode reads exactly one frame. io.EOF is reported only when no header
// byte is available (clean stream end); any short header or payload is
// ErrTruncated, never a partial success.
func (d *Decoder) Decode() (Frame, error) {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(d.r, header); err != nil {
		if err == io.EOF {
			return Frame{}, io.EOF
		}
		return Frame{}, fmt.Errorf("%w: %v", ErrTruncated, err)
	}
	if header[0] != Magic0 || header[1] != Magic1 {
		return Frame{}, fmt.Errorf("%w: %#02x%02x", ErrBadMagic, header[0], header[1])
	}
	if header[2] != Version {
		return Frame{}, fmt.Errorf("%w: %#02x", ErrBadVersion, header[2])
	}
	switch header[3] {
	case KindRequest, KindReply, KindEvent:
	default:
		return Frame{}, fmt.Errorf("%w: %#02x", ErrBadKind, header[3])
	}
	seq := binary.BigEndian.Uint32(header[4:8])
	length := binary.BigEndian.Uint32(header[8:12])
	if length > d.max {
		return Frame{}, fmt.Errorf("%w: %d > %d", ErrTooLarge, length, d.max)
	}
	if seq < d.expected {
		return Frame{}, fmt.Errorf("%w: got %d, want %d", ErrDuplicate, seq, d.expected)
	}
	if seq > d.expected {
		return Frame{}, fmt.Errorf("%w: got %d, want %d", ErrGap, seq, d.expected)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(d.r, payload); err != nil {
		return Frame{}, fmt.Errorf("%w: %v", ErrTruncated, err)
	}
	d.expected = seq + 1
	return Frame{Kind: header[3], Seq: seq, Payload: payload}, nil
}

// EncodeFrame is a convenience helper that encodes one frame to a byte slice
// with an explicit sequence, for fixtures and negative controls that need
// sequences the Encoder would never emit (duplicates, gaps).
func EncodeFrame(kind byte, seq uint32, payload []byte) []byte {
	var buf bytes.Buffer
	header := []byte{Magic0, Magic1, Version, kind, 0, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[4:8], seq)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(payload)))
	buf.Write(header)
	buf.Write(payload)
	return buf.Bytes()
}

// DecodeEnvelope parses and structurally validates a JSON payload envelope
// against the P01 operation envelope shape. Unknown fields are rejected so
// a subject cannot smuggle control meaning through extra keys.
func DecodeEnvelope(payload []byte, wantKind byte) (Envelope, error) {
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&raw); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrBadEnvelope, err)
	}
	var env Envelope
	var err error
	switch wantKind {
	case KindRequest:
		env, err = envelopeFromMap(raw, true)
	case KindReply, KindEvent:
		env, err = envelopeFromMap(raw, false)
	default:
		return Envelope{}, fmt.Errorf("%w: %#02x", ErrBadKind, wantKind)
	}
	if err != nil {
		return Envelope{}, err
	}
	if err := env.Validate(wantKind); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

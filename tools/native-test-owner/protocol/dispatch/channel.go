package dispatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// DefaultMaxPayload is the default per-frame payload bound.
	DefaultMaxPayload = codec.MaxPayload
	// DefaultMaxQueue is the default queued-frame bound.
	DefaultMaxQueue = 64
	// MaxPayloadCap caps a caller-supplied payload bound.
	MaxPayloadCap = 8 << 20
	// MaxQueueCap caps a caller-supplied queue bound.
	MaxQueueCap = 4096
)

// Channel is a kind-pinned bounded FIFO of codec frames with its own
// sequence space starting at 0. Only bytes written through Send are ever
// decoded; there is no path that decodes subject stdout as a supervisor
// message. A Channel is safe for concurrent use.
type Channel struct {
	mu       sync.Mutex
	kind     byte
	buf      bytes.Buffer
	enc      *codec.Encoder
	dec      *codec.Decoder
	maxQueue int
	queued   int
	closed   bool
}

// NewChannel returns a channel pinned to kind (request, reply or event).
// Non-positive maxPayload/maxQueue select the defaults.
func NewChannel(kind byte, maxPayload, maxQueue int) (*Channel, error) {
	switch kind {
	case codec.KindRequest, codec.KindReply, codec.KindEvent:
	default:
		return nil, fmt.Errorf("%w: unknown channel kind %#02x", ErrInvalid, kind)
	}
	if maxPayload <= 0 {
		maxPayload = DefaultMaxPayload
	}
	if maxPayload > MaxPayloadCap {
		return nil, fmt.Errorf("%w: payload bound too large", ErrInvalid)
	}
	if maxQueue <= 0 {
		maxQueue = DefaultMaxQueue
	}
	if maxQueue > MaxQueueCap {
		return nil, fmt.Errorf("%w: queue bound too large", ErrInvalid)
	}
	c := &Channel{kind: kind, maxQueue: maxQueue}
	c.enc = codec.NewEncoder(&c.buf, maxPayload)
	c.dec = codec.NewDecoder(&c.buf, maxPayload)
	return c, nil
}

// Kind reports the pinned frame kind.
func (c *Channel) Kind() byte { return c.kind }

// Send enqueues one payload frame. Queue depth and payload size are
// enforced before anything is written; a rejected send enqueues nothing.
func (c *Channel) Send(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if c.queued >= c.maxQueue {
		return fmt.Errorf("%w: channel queue full", ErrQueueFull)
	}
	if err := c.enc.Encode(c.kind, payload); err != nil {
		return err
	}
	c.queued++
	return nil
}

// marshalEnvelope renders only the kind-appropriate envelope keys. It exists
// because Go's omitempty does not drop the request field block on result
// envelopes, while the wire rejects request fields on results.
func marshalEnvelope(env codec.Envelope, kind byte) ([]byte, error) {
	m := map[string]any{
		"schema_version": env.SchemaVersion,
		"run_id":         env.RunID,
		"operation_id":   env.OperationID,
	}
	switch kind {
	case codec.KindRequest:
		m["owner_grant"] = env.OwnerGrant
		m["operation"] = env.Operation
		m["arguments_digest"] = env.ArgumentsDigest
		m["deadline_ms"] = map[string]any{"clock": env.DeadlineMs.Clock, "ms": env.DeadlineMs.Ms}
	case codec.KindReply, codec.KindEvent:
		m["outcome"] = env.Outcome
		m["kind"] = env.Kind
		if env.Facts != nil {
			m["facts"] = env.Facts
		}
		if env.Partial != nil {
			m["partial"] = env.Partial
		}
	default:
		return nil, fmt.Errorf("%w: unknown envelope kind %#02x", ErrInvalid, kind)
	}
	return json.Marshal(m)
}

// SendEnvelope validates, marshals and sends one envelope. A malformed
// envelope is an Encode-time error: it is never emitted.
func (c *Channel) SendEnvelope(env codec.Envelope) error {
	if err := env.Validate(c.kind); err != nil {
		return err
	}
	raw, err := marshalEnvelope(env, c.kind)
	if err != nil {
		return fmt.Errorf("%w: encode envelope: %v", ErrInvalid, err)
	}
	return c.Send(raw)
}

// Recv dequeues one frame. An empty open channel reports ErrEmpty (no work,
// not a broken channel); a drained closed channel reports ErrClosed.
func (c *Channel) Recv() (codec.Frame, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.queued == 0 {
		if c.closed {
			return codec.Frame{}, ErrClosed
		}
		return codec.Frame{}, ErrEmpty
	}
	f, err := c.dec.Decode()
	if err != nil {
		return codec.Frame{}, err
	}
	c.queued--
	return f, nil
}

// RecvEnvelope dequeues one frame and validates its envelope.
func (c *Channel) RecvEnvelope() (codec.Envelope, error) {
	f, err := c.Recv()
	if err != nil {
		return codec.Envelope{}, err
	}
	if f.Kind != c.kind {
		return codec.Envelope{}, fmt.Errorf("%w: frame kind %#02x on channel %#02x", ErrInvalid, f.Kind, c.kind)
	}
	return codec.DecodeEnvelope(f.Payload, c.kind)
}

// Pending reports the queued frame count.
func (c *Channel) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.queued
}

// Close marks the channel closed; queued frames still drain. It is
// idempotent.
func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// Set bundles the three supervisor channels. Each channel owns its sequence
// space, matching the codec's per-channel ordering.
type Set struct {
	Request *Channel
	Reply   *Channel
	Event   *Channel
}

// NewSet returns request, reply and event channels sharing the given
// bounds. Non-positive values select the defaults.
func NewSet(maxPayload, maxQueue int) (*Set, error) {
	req, err := NewChannel(codec.KindRequest, maxPayload, maxQueue)
	if err != nil {
		return nil, err
	}
	rep, err := NewChannel(codec.KindReply, maxPayload, maxQueue)
	if err != nil {
		return nil, err
	}
	ev, err := NewChannel(codec.KindEvent, maxPayload, maxQueue)
	if err != nil {
		return nil, err
	}
	return &Set{Request: req, Reply: rep, Event: ev}, nil
}

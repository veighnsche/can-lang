package codec

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Outcome and mechanical-kind vocabularies mirror
// schemas/native-test/operation.schema.json.
const (
	OutcomeCompleted     = "completed"
	OutcomeRejected      = "rejected"
	OutcomeFailed        = "failed"
	OutcomeDeadline      = "deadline"
	OutcomeIndeterminate = "indeterminate"
	KindOK               = "ok"
)

var (
	idPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	outcomes      = map[string]bool{
		OutcomeCompleted: true, OutcomeRejected: true, OutcomeFailed: true,
		OutcomeDeadline: true, OutcomeIndeterminate: true,
	}
	kinds = map[string]bool{
		"ok": true, "invalid-request": true, "unsupported-capability": true,
		"wrong-owner": true, "stale-handle": true, "closed-handle": true,
		"not-found": true, "permission": true, "changed-input": true,
		"kind-collision": true, "resource-limit": true, "native-io": true,
		"transport-failure": true, "unresolved-cleanup": true,
	}
	clocks = map[string]bool{"n-monotonic": true, "wall-utc": true}
)

// ClockTime is a clock-domain-tagged millisecond timestamp.
type ClockTime struct {
	Clock string `json:"clock"`
	Ms    int64  `json:"ms"`
}

// Envelope is a validated operation request or result envelope. Request-only
// and result-only fields are mutually exclusive by frame kind.
type Envelope struct {
	SchemaVersion string `json:"schema_version"`
	RunID         string `json:"run_id"`
	OperationID   string `json:"operation_id"`
	// Request fields.
	OwnerGrant      string    `json:"owner_grant,omitempty"`
	Operation       string    `json:"operation,omitempty"`
	ArgumentsDigest string    `json:"arguments_digest,omitempty"`
	DeadlineMs      ClockTime `json:"deadline_ms,omitempty"`
	// Result fields.
	Outcome string         `json:"outcome,omitempty"`
	Kind    string         `json:"kind,omitempty"`
	Facts   map[string]any `json:"facts,omitempty"`
	Partial map[string]any `json:"partial,omitempty"`
}

var requestKeys = map[string]bool{
	"schema_version": true, "run_id": true, "operation_id": true,
	"owner_grant": true, "operation": true, "arguments_digest": true, "deadline_ms": true,
}

var resultKeys = map[string]bool{
	"schema_version": true, "run_id": true, "operation_id": true,
	"outcome": true, "kind": true, "facts": true, "partial": true,
}

// envelopeFromMap decodes raw into an Envelope, rejecting unknown keys for
// the given shape (isRequest selects request keys, else result keys).
func envelopeFromMap(raw map[string]json.RawMessage, isRequest bool) (Envelope, error) {
	allowed := resultKeys
	if isRequest {
		allowed = requestKeys
	}
	for key := range raw {
		if !allowed[key] {
			return Envelope{}, fmt.Errorf("%w: unknown field %q", ErrBadEnvelope, key)
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrBadEnvelope, err)
	}
	var env Envelope
	if err := json.Unmarshal(encoded, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrBadEnvelope, err)
	}
	return env, nil
}

// Validate checks identities, version, digest shapes and outcome/kind
// agreement. wantKind selects the request or result shape.
func (e Envelope) Validate(wantKind byte) error {
	if e.SchemaVersion != "1" {
		return fmt.Errorf("%w: schema_version %q", ErrBadEnvelope, e.SchemaVersion)
	}
	if !idPattern.MatchString(e.RunID) {
		return fmt.Errorf("%w: run_id %q", ErrBadEnvelope, e.RunID)
	}
	if !idPattern.MatchString(e.OperationID) {
		return fmt.Errorf("%w: operation_id %q", ErrBadEnvelope, e.OperationID)
	}
	switch wantKind {
	case KindRequest:
		return e.validateRequest()
	case KindReply, KindEvent:
		return e.validateResult()
	default:
		return fmt.Errorf("%w: %#02x", ErrBadKind, wantKind)
	}
}

func (e Envelope) validateRequest() error {
	if e.OwnerGrant == "" || len(e.OwnerGrant) > 256 {
		return fmt.Errorf("%w: owner_grant length", ErrBadEnvelope)
	}
	if e.Operation == "" || len(e.Operation) > 128 {
		return fmt.Errorf("%w: operation length", ErrBadEnvelope)
	}
	if !digestPattern.MatchString(e.ArgumentsDigest) {
		return fmt.Errorf("%w: arguments_digest", ErrBadEnvelope)
	}
	if !clocks[e.DeadlineMs.Clock] {
		return fmt.Errorf("%w: deadline clock %q", ErrBadEnvelope, e.DeadlineMs.Clock)
	}
	if e.DeadlineMs.Ms < 0 {
		return fmt.Errorf("%w: negative deadline", ErrBadEnvelope)
	}
	if e.Outcome != "" || e.Kind != "" || e.Facts != nil || e.Partial != nil {
		return fmt.Errorf("%w: result fields on request", ErrBadEnvelope)
	}
	return nil
}

func (e Envelope) validateResult() error {
	if !outcomes[e.Outcome] {
		return fmt.Errorf("%w: outcome %q", ErrBadEnvelope, e.Outcome)
	}
	if !kinds[e.Kind] {
		return fmt.Errorf("%w: kind %q", ErrBadEnvelope, e.Kind)
	}
	if e.Outcome == OutcomeCompleted && e.Kind != KindOK {
		return fmt.Errorf("%w: completed requires kind ok", ErrBadEnvelope)
	}
	if e.Outcome != OutcomeCompleted && e.Kind == KindOK {
		return fmt.Errorf("%w: outcome %s requires non-ok kind", ErrBadEnvelope, e.Outcome)
	}
	if len(e.Facts) > 32 || len(e.Partial) > 32 {
		return fmt.Errorf("%w: facts/partial exceed 32 entries", ErrBadEnvelope)
	}
	if e.OwnerGrant != "" || e.Operation != "" || e.ArgumentsDigest != "" || e.DeadlineMs != (ClockTime{}) {
		return fmt.Errorf("%w: request fields on result", ErrBadEnvelope)
	}
	return nil
}

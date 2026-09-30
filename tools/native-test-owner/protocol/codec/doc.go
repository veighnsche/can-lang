// Package codec implements the bounded, typed native-test wire codec (task P02).
//
// Frames carry length-prefixed JSON envelopes between the Go owner (N) and
// Can workers over private channels that subjects can never inherit:
// candidate stdout, CLI JSON and page events are observations, never
// supervisor messages.
//
// Frame layout (all integers big-endian):
//
//	magic   [2]byte  "NT" (0x4E 0x54)
//	version [1]byte  0x01; any other version is rejected
//	kind    [1]byte  0x01 request, 0x02 reply, 0x03 event
//	seq     uint32   per-channel sequence; strictly increasing from 0
//	length  uint32   payload bytes; enforced before buffering
//	payload [length]byte JSON envelope, at most MaxPayload bytes
//
// The decoder enforces the length bound before allocating, rejects duplicate
// or gapped sequences, truncated reads and malformed envelopes, and never
// trusts subject stdout: only bytes arriving on the supervisor channel are
// decoded. Payload envelopes follow schemas/native-test/operation.schema.json
// (schemaVersion "1", run/operation identities, owner grant or outcome kind).
//
// Cross-language vectors live in testdata/: the Can codec (P02 Can side, P23)
// must accept and reject the same frames. This package is transport only;
// it knows no test names, scenarios or verdicts.
package codec

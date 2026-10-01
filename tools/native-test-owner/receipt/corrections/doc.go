// Package corrections implements the owner-only append-only correction
// registry for native-test report authority (task P22).
//
// Each correction record commits to the previous record's digest, forming
// a hash chain over the full history: seq numbers run 0, 1, 2 without
// gaps, every record's PrevDigest is its predecessor's Digest (or the
// genesis word for seq 0), and every Digest recomputes from the record's
// own fields. Records carry no timestamps and no process identities, so a
// chain built from the same entries always carries the same digests.
//
// The registry file is JSON lines, one record per line, written with
// append-only O_APPEND writes. A refused append — an invalid entry or a
// broken existing chain — writes nothing. History is never rewritten in
// either direction: corrections add records, and Resolve derives the
// conservative current standing from the whole chain, so a fresh pass
// appended after a failure still resolves to that failure.
//
// Resolve refuses on any breakage: a digest mismatch, a prev-link gap, a
// resequence, an unknown schema, or a malformed line is never resolved
// to a standing. This package owns only registry mechanics; qualification
// verdicts and policy live in the Can authority package, which re-checks
// linkage structurally over the same words.
//
// The registry assumes a single writer: concurrent appends need external
// coordination, which this package does not provide.
package corrections

package dispatch

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// MaxOperations caps recorded operations in one dispatcher.
const MaxOperations = 8192

// maxFactsEntries mirrors the envelope bound on facts/partial entries.
const maxFactsEntries = 32

// Result is the terminal outcome of one operation, using the
// schemas/native-test/operation.schema.json outcome/kind vocabulary. Facts
// and Partial values must be JSON-marshalable; maps are deep-copied on the
// way in and out so callers cannot mutate recorded results. Numbers
// normalize to float64 through the copy, matching their envelope encoding.
type Result struct {
	Outcome string
	Kind    string
	Facts   map[string]any
	Partial map[string]any
}

// deepCopyMap copies a JSON-shaped map through an encoding round trip so the
// copy shares nothing with the source, nested values included. It reports
// false for values no envelope can carry.
func deepCopyMap(m map[string]any) (map[string]any, bool) {
	if m == nil {
		return nil, true
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false
	}
	return out, true
}

func (r Result) copy() Result {
	// Recorded results pass the marshal check at record time, so these
	// copies cannot fail; a nil map on error keeps the no-aliasing
	// guarantee even then.
	facts, _ := deepCopyMap(r.Facts)
	partial, _ := deepCopyMap(r.Partial)
	return Result{Outcome: r.Outcome, Kind: r.Kind, Facts: facts, Partial: partial}
}

var validOutcomes = map[string]bool{
	codec.OutcomeCompleted: true, codec.OutcomeRejected: true, codec.OutcomeFailed: true,
	codec.OutcomeDeadline: true, codec.OutcomeIndeterminate: true,
}

var validKinds = map[string]bool{
	"ok": true, "invalid-request": true, "unsupported-capability": true,
	"wrong-owner": true, "stale-handle": true, "closed-handle": true,
	"not-found": true, "permission": true, "changed-input": true,
	"kind-collision": true, "resource-limit": true, "native-io": true,
	"transport-failure": true, "unresolved-cleanup": true,
}

// coerceResult validates an effect result. Unknown outcomes, kinds or
// outcome/kind disagreements become failed/native-io; oversized facts or
// partial maps become failed/resource-limit; unmarshalable maps (evidence no
// reply frame could carry) become failed/native-io. Every recorded result is
// a deep copy, so callers can neither mutate recorded results nor observe
// later mutations through returned ones.
func coerceResult(r Result) Result {
	if !validOutcomes[r.Outcome] || !validKinds[r.Kind] {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	if r.Outcome == codec.OutcomeCompleted && r.Kind != codec.KindOK {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	if r.Outcome != codec.OutcomeCompleted && r.Kind == codec.KindOK {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	if len(r.Facts) > maxFactsEntries || len(r.Partial) > maxFactsEntries {
		return Result{Outcome: codec.OutcomeFailed, Kind: "resource-limit"}
	}
	facts, ok := deepCopyMap(r.Facts)
	if !ok {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	partial, ok := deepCopyMap(r.Partial)
	if !ok {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	return Result{Outcome: r.Outcome, Kind: r.Kind, Facts: facts, Partial: partial}
}

type entryState uint8

const (
	statePending entryState = iota
	stateDone
	stateIndeterminate
)

type opEntry struct {
	runID      string
	digest     string
	operation  string
	result     Result
	state      entryState
	effectRuns int
	done       chan struct{} // closed when a pending entry resolves
}

// Dispatcher executes each operation effect at most once per operation ID.
// The idempotency key is (run, operation ID, operation name, argument
// digest): repeats with an identical key join the recorded result, and any
// difference is rejected with ErrChangedInput. A Dispatcher is safe for
// concurrent use; concurrent dispatches of one ID single-flight so the
// effect still runs exactly once.
type Dispatcher struct {
	mu  sync.Mutex
	ops map[string]*opEntry
}

// NewDispatcher returns an empty dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{ops: make(map[string]*opEntry)}
}

func validOperation(name string) bool { return len(name) >= 1 && len(name) <= 128 }

// runEffect invokes the effect outside the dispatcher lock. An effect error
// or panic becomes a terminal failed/native-io result: the effect ran, so
// the ID stays consumed and is never redispatched.
func runEffect(effect func() (Result, error)) (res Result) {
	defer func() {
		if recover() != nil {
			res = Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
		}
	}()
	r, err := effect()
	if err != nil {
		return Result{Outcome: codec.OutcomeFailed, Kind: "native-io"}
	}
	return coerceResult(r)
}

// Dispatch executes effect for a fresh ID and joins recorded results for
// repeats. It returns joined=true when no new effect ran. An indeterminate
// join returns the indeterminate Result together with ErrIndeterminate;
// callers must treat that as unknown, never as success. Dispatch errors are
// dispatch-level problems only: operation failures arrive as a Result with
// outcome failed and a nil error.
func (d *Dispatcher) Dispatch(runID, opID, digest, operation string, effect func() (Result, error)) (Result, bool, error) {
	if !validID(runID) {
		return Result{}, false, fmt.Errorf("%w: malformed run ID", ErrInvalid)
	}
	if !validID(opID) {
		return Result{}, false, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if !validDigest(digest) {
		return Result{}, false, fmt.Errorf("%w: malformed argument digest", ErrInvalid)
	}
	if !validOperation(operation) {
		return Result{}, false, fmt.Errorf("%w: malformed operation name", ErrInvalid)
	}
	if effect == nil {
		return Result{}, false, fmt.Errorf("%w: nil effect", ErrInvalid)
	}
	d.mu.Lock()
	for {
		e, ok := d.ops[opID]
		if !ok {
			break
		}
		if e.digest != digest || e.operation != operation || e.runID != runID {
			d.mu.Unlock()
			return Result{}, false, fmt.Errorf("%w: operation %q", ErrChangedInput, opID)
		}
		if e.state == statePending {
			done := e.done
			d.mu.Unlock()
			<-done
			d.mu.Lock()
			continue
		}
		res := e.result.copy()
		ind := e.state == stateIndeterminate
		d.mu.Unlock()
		if ind {
			return res, true, fmt.Errorf("%w: operation %q", ErrIndeterminate, opID)
		}
		return res, true, nil
	}
	if len(d.ops) >= MaxOperations {
		d.mu.Unlock()
		return Result{}, false, fmt.Errorf("%w: dispatcher full", ErrCapacity)
	}
	e := &opEntry{runID: runID, digest: digest, operation: operation, state: statePending, done: make(chan struct{})}
	d.ops[opID] = e
	d.mu.Unlock()

	res := runEffect(effect)

	d.mu.Lock()
	e.effectRuns = 1
	e.result = res
	e.state = stateDone
	close(e.done)
	d.mu.Unlock()
	return res.copy(), false, nil
}

// MarkAckLost reports that the acknowledgment for a recorded operation was
// lost. The unacknowledged result is discarded and every later join reports
// indeterminate with kind transport-failure; the effect is never
// redispatched. It is idempotent and fails with ErrNotFound for unknown
// IDs.
func (d *Dispatcher) MarkAckLost(opID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.ops[opID]
	if !ok {
		return fmt.Errorf("%w: operation %q", ErrNotFound, opID)
	}
	if e.state == statePending {
		return fmt.Errorf("%w: operation %q still in flight", ErrInvalid, opID)
	}
	if e.state == stateIndeterminate {
		return nil
	}
	e.state = stateIndeterminate
	e.result = Result{Outcome: codec.OutcomeIndeterminate, Kind: "transport-failure"}
	return nil
}

// AdoptIndeterminate records durable intent that has no local terminal fact
// (crash, restart, or an effect owned elsewhere) as indeterminate with kind
// unresolved-cleanup. Later joins report indeterminate without running any
// effect here. Adopting an already recorded ID with the same key is a
// no-op that never clobbers the recorded result.
func (d *Dispatcher) AdoptIndeterminate(runID, opID, digest, operation string) error {
	if !validID(runID) {
		return fmt.Errorf("%w: malformed run ID", ErrInvalid)
	}
	if !validID(opID) {
		return fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if !validDigest(digest) {
		return fmt.Errorf("%w: malformed argument digest", ErrInvalid)
	}
	if !validOperation(operation) {
		return fmt.Errorf("%w: malformed operation name", ErrInvalid)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e, ok := d.ops[opID]; ok {
		if e.digest != digest || e.operation != operation || e.runID != runID {
			return fmt.Errorf("%w: operation %q", ErrChangedInput, opID)
		}
		return nil
	}
	if len(d.ops) >= MaxOperations {
		return fmt.Errorf("%w: dispatcher full", ErrCapacity)
	}
	d.ops[opID] = &opEntry{
		runID: runID, digest: digest, operation: operation,
		state:  stateIndeterminate,
		result: Result{Outcome: codec.OutcomeIndeterminate, Kind: "unresolved-cleanup"},
	}
	return nil
}

// Lookup returns a copy of the recorded result. It reports false for unknown
// IDs and for effects still in flight, which have no terminal fact yet.
func (d *Dispatcher) Lookup(opID string) (Result, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.ops[opID]
	if !ok || e.state == statePending {
		return Result{}, false
	}
	return e.result.copy(), true
}

// EffectRuns reports how many times the effect ran: 0 for adopted
// indeterminate intents, 1 after the single execution. It reports false for
// unknown IDs.
func (d *Dispatcher) EffectRuns(opID string) (int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.ops[opID]
	if !ok {
		return 0, false
	}
	return e.effectRuns, true
}

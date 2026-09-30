package dispatch

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

func td(n int) string { return fmt.Sprintf("sha256:%064x", n) }

func okRes() Result { return Result{Outcome: codec.OutcomeCompleted, Kind: codec.KindOK} }

// TestDispatchSequence drives one dispatcher through join, reject and
// indeterminate steps, asserting the effect runs at most once per ID.
func TestDispatchSequence(t *testing.T) {
	d := NewDispatcher()
	var runs atomic.Int32
	mkEffect := func(res Result, err error) func() (Result, error) {
		return func() (Result, error) {
			runs.Add(1)
			return res, err
		}
	}
	bigFacts := make(map[string]any, 33)
	for i := 0; i < 33; i++ {
		bigFacts[fmt.Sprintf("k%d", i)] = i
	}
	type step struct {
		name        string
		act         string // dispatch|acklost|adopt
		run, op     string
		digest      string
		operation   string
		res         Result
		effErr      error
		wantJoined  bool
		wantErr     error
		wantOutcome string
		wantKind    string
		wantRuns    int32
	}
	steps := []step{
		{name: "fresh executes", act: "dispatch", run: "run-1", op: "op-1", digest: td(1), operation: "read", res: okRes(), wantJoined: false, wantOutcome: codec.OutcomeCompleted, wantKind: codec.KindOK, wantRuns: 1},
		{name: "same joins without new effect", act: "dispatch", run: "run-1", op: "op-1", digest: td(1), operation: "read", res: okRes(), wantJoined: true, wantOutcome: codec.OutcomeCompleted, wantKind: codec.KindOK, wantRuns: 1},
		{name: "changed digest rejects", act: "dispatch", run: "run-1", op: "op-1", digest: td(2), operation: "read", res: okRes(), wantErr: ErrChangedInput, wantRuns: 1},
		{name: "changed operation rejects", act: "dispatch", run: "run-1", op: "op-1", digest: td(1), operation: "write", res: okRes(), wantErr: ErrChangedInput, wantRuns: 1},
		{name: "changed run rejects", act: "dispatch", run: "run-2", op: "op-1", digest: td(1), operation: "read", res: okRes(), wantErr: ErrChangedInput, wantRuns: 1},
		{name: "effect error becomes failed", act: "dispatch", run: "run-1", op: "op-2", digest: td(3), operation: "read", effErr: errors.New("boom"), wantJoined: false, wantOutcome: codec.OutcomeFailed, wantKind: "native-io", wantRuns: 2},
		{name: "failed joins", act: "dispatch", run: "run-1", op: "op-2", digest: td(3), operation: "read", res: okRes(), wantJoined: true, wantOutcome: codec.OutcomeFailed, wantKind: "native-io", wantRuns: 2},
		{name: "oversized facts become resource-limit", act: "dispatch", run: "run-1", op: "op-3", digest: td(4), operation: "read", res: Result{Outcome: codec.OutcomeCompleted, Kind: codec.KindOK, Facts: bigFacts}, wantJoined: false, wantOutcome: codec.OutcomeFailed, wantKind: "resource-limit", wantRuns: 3},
		{name: "bad outcome coerced", act: "dispatch", run: "run-1", op: "op-4", digest: td(5), operation: "read", res: Result{Outcome: "bogus", Kind: codec.KindOK}, wantJoined: false, wantOutcome: codec.OutcomeFailed, wantKind: "native-io", wantRuns: 4},
		{name: "completed with non-ok kind coerced", act: "dispatch", run: "run-1", op: "op-5", digest: td(6), operation: "read", res: Result{Outcome: codec.OutcomeCompleted, Kind: "native-io"}, wantJoined: false, wantOutcome: codec.OutcomeFailed, wantKind: "native-io", wantRuns: 5},
		{name: "ack lost recorded", act: "acklost", op: "op-1", wantRuns: 5},
		{name: "ack lost is idempotent", act: "acklost", op: "op-1", wantRuns: 5},
		{name: "lost ack joins indeterminate without redispatch", act: "dispatch", run: "run-1", op: "op-1", digest: td(1), operation: "read", res: okRes(), wantJoined: true, wantErr: ErrIndeterminate, wantOutcome: codec.OutcomeIndeterminate, wantKind: "transport-failure", wantRuns: 5},
		{name: "ack lost on unknown id", act: "acklost", op: "op-missing", wantErr: ErrNotFound, wantRuns: 5},
		{name: "adopt records indeterminate", act: "adopt", run: "run-1", op: "op-9", digest: td(9), operation: "read", wantRuns: 5},
		{name: "adopt is a no-op for recorded ids", act: "adopt", run: "run-1", op: "op-9", digest: td(9), operation: "read", wantRuns: 5},
		{name: "adopt with changed digest rejects", act: "adopt", run: "run-1", op: "op-9", digest: td(10), operation: "read", wantErr: ErrChangedInput, wantRuns: 5},
		{name: "adopted joins indeterminate without effect", act: "dispatch", run: "run-1", op: "op-9", digest: td(9), operation: "read", res: okRes(), wantJoined: true, wantErr: ErrIndeterminate, wantOutcome: codec.OutcomeIndeterminate, wantKind: "unresolved-cleanup", wantRuns: 5},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			switch st.act {
			case "dispatch":
				res, joined, err := d.Dispatch(st.run, st.op, st.digest, st.operation, mkEffect(st.res, st.effErr))
				if st.wantErr == nil && err != nil {
					t.Fatalf("Dispatch err = %v, want nil", err)
				}
				if st.wantErr != nil && !errors.Is(err, st.wantErr) {
					t.Fatalf("Dispatch err = %v, want %v", err, st.wantErr)
				}
				if joined != st.wantJoined {
					t.Fatalf("joined = %v, want %v", joined, st.wantJoined)
				}
				if st.wantErr == nil || errors.Is(err, ErrIndeterminate) {
					if res.Outcome != st.wantOutcome || res.Kind != st.wantKind {
						t.Fatalf("result = %s/%s, want %s/%s", res.Outcome, res.Kind, st.wantOutcome, st.wantKind)
					}
				}
			case "acklost":
				err := d.MarkAckLost(st.op)
				if st.wantErr == nil && err != nil {
					t.Fatalf("MarkAckLost err = %v, want nil", err)
				}
				if st.wantErr != nil && !errors.Is(err, st.wantErr) {
					t.Fatalf("MarkAckLost err = %v, want %v", err, st.wantErr)
				}
			case "adopt":
				err := d.AdoptIndeterminate(st.run, st.op, st.digest, st.operation)
				if st.wantErr == nil && err != nil {
					t.Fatalf("AdoptIndeterminate err = %v, want nil", err)
				}
				if st.wantErr != nil && !errors.Is(err, st.wantErr) {
					t.Fatalf("AdoptIndeterminate err = %v, want %v", err, st.wantErr)
				}
			}
			if got := runs.Load(); got != st.wantRuns {
				t.Fatalf("effect runs = %d, want %d", got, st.wantRuns)
			}
		})
	}
	if n, ok := d.EffectRuns("op-1"); !ok || n != 1 {
		t.Fatalf("EffectRuns(op-1) = %d,%v, want 1,true", n, ok)
	}
	if n, ok := d.EffectRuns("op-9"); !ok || n != 0 {
		t.Fatalf("EffectRuns(op-9) = %d,%v, want 0,true", n, ok)
	}
	if _, ok := d.EffectRuns("op-missing"); ok {
		t.Fatal("EffectRuns(op-missing) reported an entry")
	}
	if _, ok := d.Lookup("op-missing"); ok {
		t.Fatal("Lookup(op-missing) reported an entry")
	}
}

func TestDispatchInvalid(t *testing.T) {
	d := NewDispatcher()
	longOp := make([]byte, 129)
	for i := range longOp {
		longOp[i] = 'o'
	}
	cases := []struct {
		name      string
		run, op   string
		digest    string
		operation string
		nilEffect bool
	}{
		{"empty run", "", "op-1", td(1), "read", false},
		{"bad op id", "run-1", "../escape", td(1), "read", false},
		{"bad digest", "run-1", "op-1", "not-a-digest", "read", false},
		{"empty operation", "run-1", "op-1", td(1), "", false},
		{"long operation", "run-1", "op-1", td(1), string(longOp), false},
		{"nil effect", "run-1", "op-1", td(1), "read", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var eff func() (Result, error)
			if !c.nilEffect {
				eff = func() (Result, error) { return okRes(), nil }
			}
			if _, _, err := d.Dispatch(c.run, c.op, c.digest, c.operation, eff); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Dispatch err = %v, want ErrInvalid", err)
			}
		})
	}
	if _, ok := d.Lookup("op-1"); ok {
		t.Fatal("invalid dispatch recorded an entry")
	}
	if err := d.AdoptIndeterminate("run-1", "bad id!", td(1), "read"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("AdoptIndeterminate err = %v, want ErrInvalid", err)
	}
}

// TestDispatchConcurrentSingleFlight requires N racing dispatches of one ID
// to run the effect exactly once.
func TestDispatchConcurrentSingleFlight(t *testing.T) {
	d := NewDispatcher()
	var runs atomic.Int32
	const racers = 16
	var wg sync.WaitGroup
	joined := make([]bool, racers)
	errs := make([]error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, j, err := d.Dispatch("run-1", "op-race", td(7), "read", func() (Result, error) {
				runs.Add(1)
				time.Sleep(20 * time.Millisecond)
				return okRes(), nil
			})
			joined[i], errs[i] = j, err
		}(i)
	}
	wg.Wait()
	fresh := 0
	for i := range joined {
		if errs[i] != nil {
			t.Fatalf("racer %d err = %v", i, errs[i])
		}
		if !joined[i] {
			fresh++
		}
	}
	if fresh != 1 {
		t.Fatalf("fresh executions = %d, want 1", fresh)
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("effect runs = %d, want 1", got)
	}
}

func TestEffectPanicBecomesFailed(t *testing.T) {
	d := NewDispatcher()
	mk := func() func() (Result, error) {
		return func() (Result, error) { panic("handler bug") }
	}
	res, joined, err := d.Dispatch("run-1", "op-panic", td(8), "read", mk())
	if err != nil || joined {
		t.Fatalf("panic dispatch = %+v,%v,%v, want failed result without join", res, joined, err)
	}
	if res.Outcome != codec.OutcomeFailed || res.Kind != "native-io" {
		t.Fatalf("result = %s/%s, want failed/native-io", res.Outcome, res.Kind)
	}
	res2, joined2, err2 := d.Dispatch("run-1", "op-panic", td(8), "read", mk())
	if err2 != nil || !joined2 || res2.Outcome != res.Outcome || res2.Kind != res.Kind {
		t.Fatalf("panic join = %+v,%v,%v, want identical joined result", res2, joined2, err2)
	}
	if n, _ := d.EffectRuns("op-panic"); n != 1 {
		t.Fatalf("effect runs = %d, want 1", n)
	}
}

func TestDispatchCapacity(t *testing.T) {
	d := NewDispatcher()
	eff := func() (Result, error) { return okRes(), nil }
	for i := 0; i < MaxOperations; i++ {
		if _, _, err := d.Dispatch("run-1", fmt.Sprintf("op-%d", i), td(i), "read", eff); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}
	if _, _, err := d.Dispatch("run-1", "op-overflow", td(MaxOperations+1), "read", eff); !errors.Is(err, ErrCapacity) {
		t.Fatalf("overflow err = %v, want ErrCapacity", err)
	}
	if err := d.AdoptIndeterminate("run-1", "op-adopt-overflow", td(MaxOperations+2), "read"); !errors.Is(err, ErrCapacity) {
		t.Fatalf("adopt overflow err = %v, want ErrCapacity", err)
	}
}

func TestCoerceUnmarshalableFacts(t *testing.T) {
	d := NewDispatcher()
	res, joined, err := d.Dispatch("run-1", "op-1", td(1), "work.echo", func() (Result, error) {
		return Result{Outcome: codec.OutcomeCompleted, Kind: codec.KindOK, Facts: map[string]any{"fn": func() {}}}, nil
	})
	if err != nil || joined {
		t.Fatalf("Dispatch = %+v %v %v", res, joined, err)
	}
	if res.Outcome != codec.OutcomeFailed || res.Kind != "native-io" {
		t.Fatalf("coerced = %+v, want failed/native-io", res)
	}
	if res.Facts != nil {
		t.Fatalf("coerced facts = %v, want dropped", res.Facts)
	}
}

func TestRecordedResultsDoNotAlias(t *testing.T) {
	d := NewDispatcher()
	nested := map[string]any{"n": 1}
	if _, _, err := d.Dispatch("run-1", "op-1", td(1), "work.echo", func() (Result, error) {
		return Result{Outcome: codec.OutcomeCompleted, Kind: codec.KindOK, Facts: map[string]any{"deep": nested}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	nested["n"] = 999
	first, ok := d.Lookup("op-1")
	if !ok {
		t.Fatal("recorded result missing")
	}
	first.Facts["deep"].(map[string]any)["n"] = 1000
	second, ok := d.Lookup("op-1")
	if !ok {
		t.Fatal("recorded result missing")
	}
	if got := second.Facts["deep"].(map[string]any)["n"]; got != float64(1) {
		t.Fatalf("recorded nested value = %v (%T), want 1 (caller/reader aliasing)", got, got)
	}
}

// Observer child process: the actual independent host-effect observer.
//
// The child runs OUTSIDE the launch/driver service and its killable
// subtree: the N/T owner spawns it as a sibling of the driver, never as
// a driver descendant, so killing the driver (even SIGKILL) cannot take
// the observer down. The child owns exactly one launch interval: it
// opens its granted scope, attests the launch, serves the wire protocol
// on stdin/stdout, and durably publishes every accepted mutation before
// answering. Sensor samples arrive as framed wire bytes carrying
// (channel, state) pairs; the bounded controls feed a pipe stub, and the
// qualified host profile wires the real host-UI sensor to the same
// intake (see the K07 evidence residual).
//
// Fail-closed: an unparsable frame, an unknown op, a journal failure, or
// a broken pipe exits nonzero without another word, so the owner reads
// EOF and marks the interval observer-lost. Only a clean shutdown op or
// stdin EOF exits zero.
package browser_observer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Child environment contract. The owner spawns the child with
// ChildEnvGate set and ChildEnvSpec carrying the JSON ChildSpec.
const (
	ChildEnvGate = "CAN_BROWSER_OBSERVER_CHILD"
	ChildEnvSpec = "CAN_BROWSER_OBSERVER_SPEC"
)

// Child exit codes. Zero is a clean shutdown; anything else is loss.
const (
	ChildExitClean   = 0
	ChildExitUsage   = 2
	ChildExitChannel = 3
)

// ChildSpec binds one observer child to its owned scope, launch, and
// journal file. The owner mints the journal path; the child creates it.
type ChildSpec struct {
	Owner       Owner    `json:"owner"`
	Scope       string   `json:"scope"`
	Channels    []string `json:"channels"`
	Limits      Limits   `json:"limits"`
	LaunchID    string   `json:"launchId"`
	JournalPath string   `json:"journalPath"`
}

// EncodeChildSpec encodes the spec for the child environment.
func EncodeChildSpec(spec ChildSpec) (string, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("browser_observer: encode child spec: %w", err)
	}
	return string(raw), nil
}

// ObserverChildMain runs the observer child. It returns the process exit
// code; the host main or TestMain exits with it.
func ObserverChildMain() int {
	raw := os.Getenv(ChildEnvSpec)
	var spec ChildSpec
	dec := json.NewDecoder(bytes.NewBufferString(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil || dec.More() {
		return ChildExitUsage
	}
	engine, err := New([]Grant{{Owner: spec.Owner, Scope: spec.Scope, Channels: spec.Channels}}, spec.Limits)
	if err != nil {
		return ChildExitUsage
	}
	if _, err := engine.OpenScope(spec.Owner, spec.Scope); err != nil {
		return ChildExitUsage
	}
	attestation, err := engine.AdmitLaunch(spec.Owner, spec.Scope, spec.LaunchID)
	if err != nil {
		return ChildExitUsage
	}
	token, err := engine.TokenForTest(spec.Owner, spec.Scope, spec.LaunchID)
	if err != nil {
		return ChildExitUsage
	}
	journal, err := CreateJournal(spec.JournalPath)
	if err != nil {
		return ChildExitUsage
	}
	defer journal.Close()
	if err := journal.WriteHeader(journalHeader{
		Owner: spec.Owner, Scope: spec.Scope, Channels: spec.Channels,
		LaunchID: spec.LaunchID, Limits: spec.Limits,
	}); err != nil {
		return ChildExitUsage
	}
	out := bufio.NewWriter(os.Stdout)
	ready, err := json.Marshal(readyReport{Ready: true, Token: token, Attestation: attestation})
	if err != nil {
		return ChildExitChannel
	}
	ready = append(ready, '\n')
	if _, err := out.Write(ready); err != nil {
		return ChildExitChannel
	}
	if err := out.Flush(); err != nil {
		return ChildExitChannel
	}
	bound := &childBinding{engine: engine, owner: spec.Owner, scope: spec.Scope, launch: spec.LaunchID, journal: journal}
	in := bufio.NewReader(os.Stdin)
	for {
		line, err := readLine(in)
		if err != nil {
			// EOF is the owner's clean hangup only when stdin
			// closes at a frame boundary; anything else is loss.
			// Either way the child goes quiet: the journal holds
			// every acknowledged mutation.
			return ChildExitClean
		}
		req, err := decodeRequest(line)
		if err != nil {
			return ChildExitChannel
		}
		resp, done, fatal := bound.serve(req)
		if fatal {
			return ChildExitChannel
		}
		raw, err := encodeResponse(resp)
		if err != nil {
			return ChildExitChannel
		}
		if _, err := out.Write(raw); err != nil {
			return ChildExitChannel
		}
		if err := out.Flush(); err != nil {
			return ChildExitChannel
		}
		if done {
			return ChildExitClean
		}
	}
}

type childBinding struct {
	engine  *Observer
	owner   Owner
	scope   string
	launch  string
	journal *Journal
}

// serve executes one request. Mutations journal after acceptance and
// before the response. A journal failure is fatal: the engine accepted a
// mutation the journal cannot publish, so no honest reply exists; the
// child exits and the owner reads loss.
func (b *childBinding) serve(req wireRequest) (resp wireResponse, done bool, fatal bool) {
	var body any
	var err error
	switch req.Op {
	case OpSample:
		var facts TickFacts
		facts, err = b.engine.Observe(b.owner, b.scope, b.launch, req.Token, req.Channel, req.State)
		body = facts
	case OpEndTick:
		var facts TickFacts
		facts, err = b.engine.EndTick(b.owner, b.scope, b.launch, req.Token)
		body = facts
	case OpNoteDriverDeath:
		err = b.engine.NoteDriverDeath(b.owner, b.scope, b.launch, req.Token)
		body = map[string]string{}
	case OpPublishCorr:
		var facts CorrectionFacts
		facts, err = b.engine.PublishCorrection(b.owner, b.scope, b.launch, req.Token, req.Tick, req.Channel, req.After)
		body = facts
	case OpMarkUnknown:
		var facts CorrectionFacts
		facts, err = b.engine.MarkUnknown(b.owner, b.scope, b.launch, req.Token, req.Tick, req.Channel)
		body = facts
	case OpSealDisposal:
		var receipt DisposalReceipt
		receipt, err = b.engine.SealDisposal(b.owner, b.scope, b.launch, req.Token)
		body = receipt
	case OpIntervalFacts:
		// Read-only: no owner check beyond scope binding, no token.
		// The interval is evidence and stays readable.
		var facts IntervalFacts
		facts, err = b.engine.IntervalFacts(b.owner, b.scope, b.launch)
		body = facts
	case OpInterrupt:
		err = b.engine.InterruptScope(b.owner, b.scope)
		body = map[string]string{}
	case OpResume:
		err = b.engine.ResumeScope(b.owner, b.scope)
		body = map[string]string{}
	case OpShutdown:
		resp, rerr := okResponse(map[string]string{})
		if rerr != nil {
			return errResponse(rerr), false, false
		}
		return resp, true, false
	default:
		return errResponse(fmt.Errorf("browser_observer: unknown wire op %q", req.Op)), false, false
	}
	if err != nil {
		return errResponse(err), false, false
	}
	if journaledOp(req.Op) {
		if jerr := b.journal.Append(req.Op, req.Channel, req.State, req.Tick, req.After); jerr != nil {
			return wireResponse{}, false, true
		}
	}
	resp, rerr := okResponse(body)
	if rerr != nil {
		return errResponse(rerr), false, false
	}
	return resp, false, false
}

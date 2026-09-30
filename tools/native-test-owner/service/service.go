package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/dispatch"
)

// ErrInvalid reports malformed service requests.
var ErrInvalid = errors.New("service: invalid request")

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxSubjectCases caps cases with retained stdout observations.
	MaxSubjectCases = 1024
	// MaxSubjectObsPerCase caps retained observations per case.
	MaxSubjectObsPerCase = 1024
	// MaxSubjectBytesPerCase caps retained stdout bytes per case.
	MaxSubjectBytesPerCase = 1 << 20
	// MaxHandlers caps registered operation names: every field is finite.
	MaxHandlers = 256
)

// Handler executes one operation. The argument digest identifies the
// recorded arguments; the handler must be deterministic in them because it
// runs at most once per operation ID. A non-nil error becomes a terminal
// failed/native-io result. Results follow the operation result vocabulary;
// invalid results are coerced to failed outcomes.
type Handler func(argsDigest string) (dispatch.Result, error)

// Request asks the owner to dispatch one operation under a grant and
// handle. The handle must be a run or case handle bound to the grant.
type Request struct {
	GrantToken      string
	Handle          dispatch.Handle
	RunID           string
	OperationID     string
	Operation       string
	ArgumentsDigest string
}

// Reply reports one dispatch. Joined is true when no new effect ran. On
// every error except indeterminate the Reply is zero; on indeterminate the
// Reply carries the indeterminate Result.
type Reply struct {
	RunID       string
	OperationID string
	Result      dispatch.Result
	Joined      bool
	Handle      dispatch.Handle // typed operation handle
}

// Config tunes a Service. The zero value selects the wall clock and default
// channel bounds.
type Config struct {
	Clock      func() int64 // wall ms for grant expiry; nil selects time.Now
	MaxPayload int          // per-frame payload bound; <=0 selects the default
	MaxQueue   int          // queued-frame bound per channel; <=0 selects the default
}

// Service is the native-test Go owner: grants, typed handles, journaled
// at-most-once dispatch and bounded framed transport. The journal is owned
// by the caller and must outlive the service. A Service is safe for
// concurrent use.
type Service struct {
	mu           sync.Mutex
	j            *journal.Journal
	owner        journal.Owner
	clock        func() int64
	grants       *dispatch.GrantTable
	handles      *dispatch.HandleTable
	dispatcher   *dispatch.Dispatcher
	channels     *dispatch.Set
	handlers     map[string]Handler
	grantHandles map[string]dispatch.Handle
	subjectLog   map[string][][]byte
	subjectBytes map[string]int
}

// New returns a Service bound to a journal and an owner identity. The
// spawn-start token disambiguates PID reuse; a bare PID is never authority.
func New(j *journal.Journal, pid int, startToken string, cfg Config) (*Service, error) {
	if j == nil {
		return nil, fmt.Errorf("%w: nil journal", ErrInvalid)
	}
	if pid <= 0 || startToken == "" || len(startToken) > journal.MaxStartTokenLen {
		return nil, fmt.Errorf("%w: owner needs positive PID and spawn-start token", ErrInvalid)
	}
	clock := cfg.Clock
	if clock == nil {
		clock = func() int64 { return time.Now().UnixMilli() }
	}
	ch, err := dispatch.NewSet(cfg.MaxPayload, cfg.MaxQueue)
	if err != nil {
		return nil, fmt.Errorf("service: channels: %w", err)
	}
	return &Service{
		j:            j,
		owner:        journal.Owner{PID: pid, StartToken: startToken},
		clock:        clock,
		grants:       dispatch.NewGrantTableWithClock(clock),
		handles:      dispatch.NewHandleTable(),
		dispatcher:   dispatch.NewDispatcher(),
		channels:     ch,
		handlers:     make(map[string]Handler),
		grantHandles: make(map[string]dispatch.Handle),
		subjectLog:   make(map[string][][]byte),
		subjectBytes: make(map[string]int),
	}, nil
}

func validID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '_' || c == '.' || c == ':' || c == '-'
		if !ok {
			return false
		}
		if i == 0 && !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validDigest(s string) bool {
	if len(s) != 7+64 || len(s) < 7 || s[:7] != "sha256:" {
		return false
	}
	for i := 7; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// Channels exposes the bounded framed request/reply/event channels.
func (s *Service) Channels() *dispatch.Set { return s.channels }

// RegisterOperation installs the handler for one operation name. Names
// follow the envelope operation field (1..128 chars). Re-registering a
// name is rejected.
func (s *Service) RegisterOperation(name string, h Handler) error {
	if len(name) == 0 || len(name) > 128 {
		return fmt.Errorf("%w: malformed operation name", ErrInvalid)
	}
	if h == nil {
		return fmt.Errorf("%w: nil handler", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.handlers[name]; dup {
		return fmt.Errorf("%w: operation %q already registered", ErrInvalid, name)
	}
	if len(s.handlers) >= MaxHandlers {
		return fmt.Errorf("%w: operation table full", dispatch.ErrCapacity)
	}
	s.handlers[name] = h
	return nil
}

// pruneGrantHandlesLocked drops mappings whose grants the table no longer
// holds live, so the mapping stays bounded by the table it mirrors.
func (s *Service) pruneGrantHandlesLocked(now int64) {
	for tok := range s.grantHandles {
		g, ok := s.grants.Lookup(tok)
		if !ok || g.Revoked || now >= g.ExpiresWallMs {
			delete(s.grantHandles, tok)
		}
	}
}

// IssueRunGrant authorizes runID and opens its typed run handle.
func (s *Service) IssueRunGrant(runID string, ttlMs int64) (dispatch.Grant, dispatch.Handle, error) {
	g, err := s.grants.IssueRunGrant(runID, ttlMs)
	if err != nil {
		return dispatch.Grant{}, dispatch.Handle{}, err
	}
	h, herr := s.handles.Open(dispatch.HandleRun, runID)
	if herr != nil {
		_ = s.grants.Revoke(g.Token)
		return dispatch.Grant{}, dispatch.Handle{}, herr
	}
	s.mu.Lock()
	s.pruneGrantHandlesLocked(s.clock())
	s.grantHandles[g.Token] = h
	s.mu.Unlock()
	return g, h, nil
}

// IssueCaseGrant derives a case grant from a live run grant and opens the
// typed case handle.
func (s *Service) IssueCaseGrant(runToken, caseID string, ttlMs int64) (dispatch.Grant, dispatch.Handle, error) {
	g, err := s.grants.IssueCaseGrant(runToken, caseID, ttlMs)
	if err != nil {
		return dispatch.Grant{}, dispatch.Handle{}, err
	}
	h, herr := s.handles.Open(dispatch.HandleCase, caseID)
	if herr != nil {
		_ = s.grants.Revoke(g.Token)
		return dispatch.Grant{}, dispatch.Handle{}, herr
	}
	s.mu.Lock()
	s.pruneGrantHandlesLocked(s.clock())
	s.grantHandles[g.Token] = h
	s.mu.Unlock()
	return g, h, nil
}

// RevokeGrant revokes one grant token.
func (s *Service) RevokeGrant(token string) error {
	err := s.grants.Revoke(token)
	s.mu.Lock()
	delete(s.grantHandles, token)
	s.mu.Unlock()
	return err
}

// CloseHandle closes one handle.
func (s *Service) CloseHandle(h dispatch.Handle) error {
	return s.handles.Close(h)
}

// closeOp walks a successful operation to its terminal journal state.
// Best effort: the dispatcher already holds the terminal fact.
func (s *Service) closeOp(opID string) {
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive, journal.StateClosing, journal.StateClosed} {
		if _, err := s.j.Advance(opID, st); err != nil {
			return
		}
	}
}

// failOp records a terminal failure. Best effort for the same reason.
func (s *Service) failOp(opID, label string) {
	if len(label) > journal.MaxPartialLen {
		label = label[:journal.MaxPartialLen]
	}
	_, _ = s.j.Advance(opID, journal.StateFailed, label)
}

func terminalState(st journal.State) bool {
	return st == journal.StateClosed || st == journal.StateFailed
}

func (s *Service) openOpHandle(runID, opID string, res dispatch.Result, joined bool) (Reply, error) {
	h, err := s.handles.Open(dispatch.HandleOperation, opID)
	rep := Reply{RunID: runID, OperationID: opID, Result: res, Joined: joined, Handle: h}
	if err != nil {
		rep.Handle = dispatch.Handle{}
		return rep, err
	}
	return rep, nil
}

// Dispatch authorizes, journals and executes one operation at most once.
// Operation IDs are owner-global: repeats must carry the identical run,
// operation name and argument digest to join, and any difference is
// rejected with dispatch.ErrChangedInput. Unknown operations consume their
// ID as a terminal rejected/unsupported-capability outcome without running
// any effect. A lost acknowledgment (see ReportAckLost) or durable intent
// without a local terminal fact reports indeterminate without
// redispatching.
func (s *Service) Dispatch(req Request) (Reply, error) {
	if !validID(req.RunID) {
		return Reply{}, fmt.Errorf("%w: malformed run ID", ErrInvalid)
	}
	if !validID(req.OperationID) {
		return Reply{}, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if len(req.Operation) == 0 || len(req.Operation) > 128 {
		return Reply{}, fmt.Errorf("%w: malformed operation name", ErrInvalid)
	}
	if !validDigest(req.ArgumentsDigest) {
		return Reply{}, fmt.Errorf("%w: malformed argument digest", ErrInvalid)
	}
	var wantScope dispatch.Scope
	var caseID string
	switch req.Handle.Kind {
	case dispatch.HandleRun:
		wantScope = dispatch.ScopeRun
	case dispatch.HandleCase:
		wantScope = dispatch.ScopeCase
		caseID = req.Handle.ID
	default:
		return Reply{}, fmt.Errorf("%w: dispatch needs a run or case handle", dispatch.ErrWrongKind)
	}
	if err := s.grants.Verify(req.GrantToken, wantScope, req.RunID, caseID); err != nil {
		return Reply{}, err
	}
	if err := s.handles.Check(req.Handle, req.Handle.Kind); err != nil {
		return Reply{}, err
	}
	if req.Handle.Kind == dispatch.HandleRun && req.Handle.ID != req.RunID {
		return Reply{}, fmt.Errorf("%w: handle bound to another run", ErrInvalid)
	}
	s.mu.Lock()
	handler := s.handlers[req.Operation]
	s.mu.Unlock()

	// Intent before effect: consult the durable journal before running
	// anything. A reservation this owner cannot join is never re-executed.
	if prev, ok := s.j.Lookup(req.OperationID); ok {
		if prev.ArgumentsDigest != req.ArgumentsDigest {
			return Reply{}, fmt.Errorf("%w: operation %q", dispatch.ErrChangedInput, req.OperationID)
		}
		if _, known := s.dispatcher.Lookup(req.OperationID); !known {
			_ = s.dispatcher.AdoptIndeterminate(req.RunID, req.OperationID, req.ArgumentsDigest, req.Operation)
			res, _ := s.dispatcher.Lookup(req.OperationID)
			rep, herr := s.openOpHandle(req.RunID, req.OperationID, res, true)
			if herr != nil {
				return rep, herr
			}
			return rep, fmt.Errorf("%w: operation %q", dispatch.ErrIndeterminate, req.OperationID)
		}
	} else if _, err := s.j.Reserve(req.OperationID, req.ArgumentsDigest, s.owner, journal.PathIdentity{}, "", 0); err != nil {
		if errors.Is(err, journal.ErrConflict) {
			return Reply{}, fmt.Errorf("%w: operation %q", dispatch.ErrChangedInput, req.OperationID)
		}
		return Reply{}, fmt.Errorf("service: journal reserve: %w", err)
	}

	effect := func() (dispatch.Result, error) {
		if handler == nil {
			return dispatch.Result{Outcome: codec.OutcomeRejected, Kind: "unsupported-capability"}, nil
		}
		return handler(req.ArgumentsDigest)
	}
	res, joined, derr := s.dispatcher.Dispatch(req.RunID, req.OperationID, req.ArgumentsDigest, req.Operation, effect)
	if derr != nil {
		if errors.Is(derr, dispatch.ErrIndeterminate) {
			rep, herr := s.openOpHandle(req.RunID, req.OperationID, res, joined)
			if herr != nil {
				return rep, herr
			}
			return rep, derr
		}
		return Reply{}, derr
	}
	if !joined {
		if res.Outcome == codec.OutcomeCompleted {
			s.closeOp(req.OperationID)
		} else {
			s.failOp(req.OperationID, res.Kind)
		}
	}
	return s.openOpHandle(req.RunID, req.OperationID, res, joined)
}

// Status reports the recorded result for a typed operation handle.
func (s *Service) Status(h dispatch.Handle) (dispatch.Result, error) {
	if err := s.handles.Check(h, dispatch.HandleOperation); err != nil {
		return dispatch.Result{}, err
	}
	res, ok := s.dispatcher.Lookup(h.ID)
	if !ok {
		return dispatch.Result{}, fmt.Errorf("%w: operation %q", dispatch.ErrNotFound, h.ID)
	}
	return res, nil
}

// ReportAckLost reports a lost acknowledgment: later joins of the operation
// report indeterminate without redispatching. Non-terminal journal intents
// advance to failed with an ack-lost label; terminal ones are left for
// reconciliation through Pending.
func (s *Service) ReportAckLost(opID string) error {
	if !validID(opID) {
		return fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	if err := s.dispatcher.MarkAckLost(opID); err != nil {
		return err
	}
	if r, ok := s.j.Lookup(opID); ok && !terminalState(r.State) {
		s.failOp(opID, "ack-lost")
	}
	return nil
}

// Pending reports journaled operations left non-terminal, so intents that
// survived a crash stay discoverable for reconciliation.
func (s *Service) Pending() []journal.Reservation {
	return s.j.Pending()
}

// EffectRuns reports how many times an operation effect ran here.
func (s *Service) EffectRuns(opID string) (int, bool) {
	return s.dispatcher.EffectRuns(opID)
}

// ObserveSubjectStdout retains subject output opaquely and returns its
// per-case sequence number. The bytes are never decoded, verified or
// dispatched: even a byte-perfect frame or envelope here has no supervisor
// effect. Retention is bounded per case; overflow drops the new
// observation with dispatch.ErrCapacity.
func (s *Service) ObserveSubjectStdout(caseID string, data []byte) (int, error) {
	if !validID(caseID) {
		return 0, fmt.Errorf("%w: malformed case ID", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	obs := s.subjectLog[caseID]
	if obs == nil && len(s.subjectLog) >= MaxSubjectCases {
		return 0, fmt.Errorf("%w: subject log full", dispatch.ErrCapacity)
	}
	if len(obs) >= MaxSubjectObsPerCase {
		return 0, fmt.Errorf("%w: subject observations for case full", dispatch.ErrCapacity)
	}
	if s.subjectBytes[caseID]+len(data) > MaxSubjectBytesPerCase {
		return 0, fmt.Errorf("%w: subject bytes for case full", dispatch.ErrCapacity)
	}
	s.subjectLog[caseID] = append(obs, append([]byte(nil), data...))
	s.subjectBytes[caseID] += len(data)
	return len(obs), nil
}

// SubjectObservations returns copies of the retained opaque observations
// for one case.
func (s *Service) SubjectObservations(caseID string) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	obs, ok := s.subjectLog[caseID]
	if !ok {
		return nil, fmt.Errorf("%w: case %q", dispatch.ErrNotFound, caseID)
	}
	out := make([][]byte, len(obs))
	for i, b := range obs {
		out[i] = append([]byte(nil), b...)
	}
	return out, nil
}

func resultEnvelope(runID, opID, outcome, kind string, facts, partial map[string]any) codec.Envelope {
	return codec.Envelope{
		SchemaVersion: "1",
		RunID:         runID,
		OperationID:   opID,
		Outcome:       outcome,
		Kind:          kind,
		Facts:         facts,
		Partial:       partial,
	}
}

// mapDispatchError renders dispatch failures as result outcomes. The
// indeterminate case is handled by the caller from the recorded Result;
// the entry here is defensive only.
func mapDispatchError(err error) (string, string) {
	switch {
	case errors.Is(err, dispatch.ErrChangedInput):
		return codec.OutcomeRejected, "changed-input"
	case errors.Is(err, dispatch.ErrIndeterminate):
		return codec.OutcomeIndeterminate, "transport-failure"
	case errors.Is(err, dispatch.ErrNotFound):
		return codec.OutcomeRejected, "not-found"
	case errors.Is(err, dispatch.ErrStaleHandle):
		return codec.OutcomeRejected, "stale-handle"
	case errors.Is(err, dispatch.ErrClosedHandle):
		return codec.OutcomeRejected, "closed-handle"
	case errors.Is(err, dispatch.ErrWrongKind):
		return codec.OutcomeRejected, "invalid-request"
	case errors.Is(err, dispatch.ErrExpired), errors.Is(err, dispatch.ErrRevoked):
		return codec.OutcomeRejected, "permission"
	case errors.Is(err, dispatch.ErrCapacity):
		return codec.OutcomeRejected, "resource-limit"
	case errors.Is(err, ErrInvalid):
		return codec.OutcomeRejected, "invalid-request"
	case errors.Is(err, journal.ErrConflict):
		return codec.OutcomeRejected, "changed-input"
	case errors.Is(err, journal.ErrInvalid):
		return codec.OutcomeRejected, "invalid-request"
	default:
		return codec.OutcomeFailed, "native-io"
	}
}

// HandleRequestFrame serves one framed request with exactly one framed
// reply. Requests without a bound grant are answered
// rejected/permission without journaling or dispatching, so unauthorized
// bytes leave no intent. Envelope deadlines are carried but not enforced
// here; shape validity is already established by the codec. An empty
// request channel reports dispatch.ErrEmpty.
func (s *Service) HandleRequestFrame() error {
	env, err := s.channels.Request.RecvEnvelope()
	if err != nil {
		return err
	}
	s.mu.Lock()
	h, ok := s.grantHandles[env.OwnerGrant]
	s.mu.Unlock()
	var reply codec.Envelope
	if !ok {
		reply = resultEnvelope(env.RunID, env.OperationID, codec.OutcomeRejected, "permission", nil, nil)
	} else {
		rep, derr := s.Dispatch(Request{
			GrantToken:      env.OwnerGrant,
			Handle:          h,
			RunID:           env.RunID,
			OperationID:     env.OperationID,
			Operation:       env.Operation,
			ArgumentsDigest: env.ArgumentsDigest,
		})
		if derr == nil || errors.Is(derr, dispatch.ErrIndeterminate) {
			reply = resultEnvelope(env.RunID, env.OperationID, rep.Result.Outcome, rep.Result.Kind, rep.Result.Facts, rep.Result.Partial)
		} else {
			outcome, kind := mapDispatchError(derr)
			reply = resultEnvelope(env.RunID, env.OperationID, outcome, kind, nil, nil)
		}
	}
	return s.channels.Reply.SendEnvelope(reply)
}

// EmitEvent publishes one supervisor event envelope.
func (s *Service) EmitEvent(runID, operationID, outcome, kind string, facts, partial map[string]any) error {
	return s.channels.Event.SendEnvelope(resultEnvelope(runID, operationID, outcome, kind, facts, partial))
}

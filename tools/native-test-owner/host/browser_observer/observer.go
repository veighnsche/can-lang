// Independent browser host-effect observer: K07's Go mirror.
//
// This package implements the ACTUAL host-UI observer outside the
// launch/driver service and its killable subtree, under independent N/T
// ownership. It is self-contained by design: only the standard library,
// no driver import, no process sampling, no environment reads, no flags.
// The observer binds to a declared host-UI effect scope at open, attests
// launches with opaque minted tokens, records per-tick UI-effect
// observations through confirmed disposal, marks interruptions and gaps
// explicitly, and publishes durable corrections. Observation survives
// driver death: a dead driver is a noted fact, never a cleanup witness
// and never the end of the interval. Observer loss makes the interval
// unknown, and a missing observer blocks launch: admission refuses
// without a live observer binding.
//
// Independence scope from the launch/driver service under test
// (normative, mirrored in
// tools/runtime/test-services/browser-driver/admission.ts): the observer
// shares no driver code, decoded values, receipts, flags, environment, or
// process state; it proves scope receipt, launch attestation, per-tick UI
// effects, gaps, corrections, disposal, and layered error provenance from
// its own seeded doubles alone. Driver claims are compared against these
// facts; no fact is ever derived from a driver claim. Consequently no
// headless flag, HOME value, launch flag set, or process sample can ever
// prove no-UI: only an observer attestation over a fully observed
// interval with zero seen effects proves no-UI.
//
// Split with the TS admission adapter: the TS service owns typed interval
// mechanics, the admission gate, and the bounded self-check. This package
// owns the same contract as the outside-service observer: owned-scope
// admission, minted launch tokens verified by lookup, the interval
// through disposal, and the honest unknown-seal. Both enforce opaque
// tokens, digest-only facts, and layered errors. Neither reads the
// driver.
//
// Engine layer: pure in-memory mechanics with no I/O, timers,
// transports, services, browsers, processes, or live runtimes. The
// actual observation channel runs in channel.go (framed wire protocol),
// child.go (observer child process outside the driver subtree), and
// process.go (owner-side handle); journal.go durably publishes every
// accepted mutation. Local controls only.
package browser_observer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Layer names the failing layer of one observer error. Every observer
// failure names its layer; nothing unclassified is returned.
type Layer string

// Observer layers. Observer owns scope admission, capacity, ownership,
// lifecycle, and loss; admission owns launch binding, tokens, and
// forbidden no-UI proofs; interval owns effect keys, ticks, gaps,
// corrections, and the disposal seal.
const (
	LayerObserver  Layer = "observer"
	LayerAdmission Layer = "admission"
	LayerInterval  Layer = "interval"
)

// Closed failure-code vocabulary. Each code maps to exactly one layer,
// mirroring the TS adapter's vocabulary.
const (
	CodeUnknownScope   = "unknown-scope"        // observer
	CodeScopeClosed    = "scope-closed"         // observer
	CodeCapacity       = "capacity-exhausted"   // observer
	CodeWrongOwner     = "wrong-owner"          // observer
	CodeObserverLost   = "observer-lost"        // observer
	CodeInterrupted    = "observer-interrupted" // observer
	CodeNoObserver     = "no-observer"          // admission
	CodeForgedToken    = "forged-token"         // admission
	CodeUnknownLaunch  = "unknown-launch"       // admission
	CodeLaunchClosed   = "launch-closed"        // admission
	CodeForbiddenProof = "forbidden-proof"      // admission
	CodeUnknownEffect  = "unknown-effect"       // interval
	CodeOutOfScope     = "out-of-scope"         // interval
	CodeGapOpen        = "gap-open"             // interval
	CodeCorrectionMiss = "correction-unknown"   // interval
	CodeCorrectionCold = "correction-stale"     // interval
	CodeDisposalOpen   = "disposal-open"        // interval
	CodeAlreadyGone    = "already-disposed"     // interval
)

// Codes lists the closed vocabulary in layer order.
var Codes = []string{
	CodeUnknownScope, CodeScopeClosed, CodeCapacity, CodeWrongOwner,
	CodeObserverLost, CodeInterrupted, CodeNoObserver, CodeForgedToken,
	CodeUnknownLaunch, CodeLaunchClosed, CodeForbiddenProof,
	CodeUnknownEffect, CodeOutOfScope, CodeGapOpen, CodeCorrectionMiss,
	CodeCorrectionCold, CodeDisposalOpen, CodeAlreadyGone,
}

// LayerOfCode maps one closed code to its single layer.
func LayerOfCode(code string) (Layer, bool) {
	switch code {
	case CodeUnknownScope, CodeScopeClosed, CodeCapacity, CodeWrongOwner,
		CodeObserverLost, CodeInterrupted:
		return LayerObserver, true
	case CodeNoObserver, CodeForgedToken, CodeUnknownLaunch,
		CodeLaunchClosed, CodeForbiddenProof:
		return LayerAdmission, true
	case CodeUnknownEffect, CodeOutOfScope, CodeGapOpen,
		CodeCorrectionMiss, CodeCorrectionCold, CodeDisposalOpen,
		CodeAlreadyGone:
		return LayerInterval, true
	default:
		return "", false
	}
}

// Sentinel causes. Callers distinguish them with errors.Is; the layered
// Code is the contract and the sentinel is the cause.
var (
	ErrInvalid    = errors.New("browser_observer: invalid request")
	ErrDenied     = errors.New("browser_observer: refused")
	ErrNotFound   = errors.New("browser_observer: unknown scope, launch or entry")
	ErrWrongOwner = errors.New("browser_observer: wrong owner identity")
	ErrCapacity   = errors.New("browser_observer: table capacity exhausted")
)

// ObserverError is one layered observer failure. Layer always names
// observer, admission, or interval; Code is the closed failure code; Err
// carries the wrapped sentinel cause for errors.Is matching.
type ObserverError struct {
	Layer Layer
	Code  string
	Err   error
}

func (e *ObserverError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Code, e.Err)
	}
	return fmt.Sprintf("[%s:%s] observer refused", e.Layer, e.Code)
}

// Unwrap exposes the wrapped sentinel cause.
func (e *ObserverError) Unwrap() error { return e.Err }

func obsErr(layer Layer, code string, err error) *ObserverError {
	return &ObserverError{Layer: layer, Code: code, Err: err}
}

// Owner is the independent N/T spawn-start identity that owns one scope.
// Ownership is checked on every call: a foreign identity fails the owner
// check and changes nothing.
type Owner struct {
	PID        int    `json:"pid"`
	StartToken string `json:"startToken"`
}

func (o Owner) same(other Owner) bool {
	return o.PID == other.PID && o.StartToken == other.StartToken
}

// Limits bounds every observer table. Nothing here means unlimited.
type Limits struct {
	MaxScopes               int `json:"maxScopes"`
	MaxLaunches             int `json:"maxLaunches"`
	MaxTicksPerLaunch       int `json:"maxTicksPerLaunch"`
	MaxCorrectionsPerLaunch int `json:"maxCorrectionsPerLaunch"`
}

// DefaultLimits is the roomy bounded double the tests use.
func DefaultLimits() Limits {
	return Limits{MaxScopes: 4, MaxLaunches: 8, MaxTicksPerLaunch: 8, MaxCorrectionsPerLaunch: 16}
}

func checkLimits(l Limits) error {
	if l.MaxScopes <= 0 || l.MaxLaunches <= 0 || l.MaxTicksPerLaunch <= 0 || l.MaxCorrectionsPerLaunch <= 0 {
		return obsErr(LayerInterval, CodeUnknownEffect, fmt.Errorf("%w: limits must all be positive", ErrInvalid))
	}
	return nil
}

// EffectChannels is the finite host-UI effect vocabulary. A scope
// observes a non-empty subset; anything outside this vocabulary is not
// an effect key at all.
var EffectChannels = []string{"window", "icon", "notification", "focus"}

// Directly observed states: the observer either saw a host-UI effect on
// the channel during the tick or it saw none. Gap and unknown are never
// accepted from Observe: they arise only from EndTick, interruption,
// loss, or durable publication.
const (
	StateSeen    = "seen"
	StateAbsent  = "absent"
	StateGap     = "gap"
	StateUnknown = "unknown"
)

// ForbiddenProofKinds are the inadmissible no-UI proof kinds. Only an
// observer attestation judges; every other kind refuses with
// forbidden-proof before any verdict is read.
var ForbiddenProofKinds = []string{"headless-flag", "home-env", "launch-flags", "process-sample"}

const maxNameLen = 128

func validName(s string) bool {
	if len(s) == 0 || len(s) > maxNameLen {
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

func checkOwnerName(s string) error {
	if !validName(s) {
		return obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: malformed owner name", ErrInvalid))
	}
	return nil
}

func checkScopeName(s string) error {
	if !validName(s) {
		return obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: malformed scope name", ErrInvalid))
	}
	return nil
}

func checkLaunchName(s string) error {
	if !validName(s) {
		return obsErr(LayerAdmission, CodeUnknownLaunch, fmt.Errorf("%w: malformed launch name", ErrInvalid))
	}
	return nil
}

func checkChannel(s string) error {
	for _, c := range EffectChannels {
		if c == s {
			return nil
		}
	}
	return obsErr(LayerInterval, CodeUnknownEffect, fmt.Errorf("%w: not a declared host-UI effect key", ErrInvalid))
}

func checkObservedState(s string) error {
	if s != StateSeen && s != StateAbsent {
		return obsErr(LayerInterval, CodeUnknownEffect, fmt.Errorf("%w: observation must be seen or absent", ErrInvalid))
	}
	return nil
}

func mintToken(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("browser_observer: random token: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(b[:]), nil
}

func digestText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// DigestInterval recomputes the interval digest from carried ticks and
// corrections, so AssertNoUiProof re-verifies a presented attestation
// instead of trusting its digest string.
func DigestInterval(launchID string, ticks []TickFacts, corrections []CorrectionFacts, flags string) string {
	h := sha256.New()
	h.Write([]byte(launchID))
	h.Write([]byte{0})
	for _, tick := range ticks {
		h.Write([]byte(fmt.Sprintf("%d", tick.Tick)))
		if tick.Closed {
			h.Write([]byte("C"))
		} else {
			h.Write([]byte("O"))
		}
		for _, entry := range tick.Channels {
			corrected := "0"
			if entry.Corrected {
				corrected = "1"
			}
			h.Write([]byte(entry.Channel + "=" + entry.State + "/" + entry.Origin + "/" + corrected))
			h.Write([]byte(";"))
		}
		h.Write([]byte("\n"))
	}
	h.Write([]byte{0})
	for _, c := range corrections {
		h.Write([]byte(fmt.Sprintf("%s:%d:%s:%s>%s;", c.CorrectionID, c.Tick, c.Channel, c.Before, c.After)))
	}
	h.Write([]byte{0})
	h.Write([]byte(flags))
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------------------
// Facts (digest-only; returned by value)
// ---------------------------------------------------------------------------

// ScopeReceipt binds one open scope to its owner, channels, and handle
// digest. The raw token never crosses.
type ScopeReceipt struct {
	Scope        string   `json:"scope"`
	Owner        Owner    `json:"owner"`
	Channels     []string `json:"channels"`
	HandleDigest string   `json:"handleDigest"`
	Launches     int      `json:"launches"`
}

// LaunchAttestation attests one launch under a live observer binding.
type LaunchAttestation struct {
	LaunchID       string `json:"launchId"`
	Scope          string `json:"scope"`
	Owner          Owner  `json:"owner"`
	ScopeDigest    string `json:"scopeDigest"`
	ObserverDigest string `json:"observerDigest"`
	HandleDigest   string `json:"handleDigest"`
	Tick           int    `json:"tick"`
}

// ChannelFacts names one channel entry: its state, origin, and whether a
// durable correction resolved it.
type ChannelFacts struct {
	Channel   string `json:"channel"`
	State     string `json:"state"`
	Origin    string `json:"origin"`
	Corrected bool   `json:"corrected"`
}

// TickFacts names one tick: its entries, closure, and digest.
type TickFacts struct {
	LaunchID string         `json:"launchId"`
	Tick     int            `json:"tick"`
	Channels []ChannelFacts `json:"channels"`
	Closed   bool           `json:"closed"`
	Digest   string         `json:"digest"`
}

// CorrectionFacts names one durable correction: tick, channel, the
// before/after states, and its digest.
type CorrectionFacts struct {
	CorrectionID string `json:"correctionId"`
	LaunchID     string `json:"launchId"`
	Tick         int    `json:"tick"`
	Channel      string `json:"channel"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Digest       string `json:"digest"`
}

// IntervalFacts names the whole observed interval. Unknown is true when
// any entry carries unknown (interruption, loss, or published unknown);
// an unknown interval can never prove no-UI.
type IntervalFacts struct {
	LaunchID         string            `json:"launchId"`
	Scope            string            `json:"scope"`
	Owner            Owner             `json:"owner"`
	Ticks            []TickFacts       `json:"ticks"`
	Corrections      []CorrectionFacts `json:"corrections"`
	DriverDeathNoted bool              `json:"driverDeathNoted"`
	Disposed         bool              `json:"disposed"`
	Unknown          bool              `json:"unknown"`
	Digest           string            `json:"digest"`
}

// DisposalReceipt seals the interval through confirmed disposal.
type DisposalReceipt struct {
	LaunchID         string `json:"launchId"`
	Scope            string `json:"scope"`
	Owner            Owner  `json:"owner"`
	Ticks            int    `json:"ticks"`
	Corrections      int    `json:"corrections"`
	DriverDeathNoted bool   `json:"driverDeathNoted"`
	Unknown          bool   `json:"unknown"`
	Digest           string `json:"digest"`
}

// NoUiVerdict judges one verified observer attestation.
type NoUiVerdict struct {
	LaunchID string
	UISeen   bool
	Unknown  bool
	Digest   string
}

type tickEntry struct {
	state     string
	origin    string
	corrected bool
}

type tickRecord struct {
	tick    int
	entries map[string]*tickEntry
	closed  bool
}

type launchRecord struct {
	launchID      string
	scope         string
	owner         Owner
	token         string
	ticks         []*tickRecord
	corrections   []CorrectionFacts
	correctionSeq int
	driverDeath   bool
	disposed      bool
	disposal      DisposalReceipt
	// lost records observer loss for a live interval. Once set, the
	// interval is unknown even when every tick was fully observed:
	// the observer died before the seal, so late host effects between
	// the last observation and disposal are unobserved.
	lost bool
}

type scopeState string

const (
	stateOpen        scopeState = "open"
	stateInterrupted scopeState = "interrupted"
	stateLost        scopeState = "lost"
	stateClosed      scopeState = "closed"
)

type scopeRecord struct {
	scope    string
	owner    Owner
	channels []string
	token    string
	state    scopeState
	launches map[string]*launchRecord
}

// Grant declares one owned (owner, scope) binding over a non-empty
// channel subset. Only exact declared grants admit.
type Grant struct {
	Owner    Owner
	Scope    string
	Channels []string
}

// Observer watches declared host-UI scopes outside the launch/driver
// service. An Observer is safe for concurrent use.
type Observer struct {
	mu     sync.Mutex
	limits Limits
	grants map[string]Grant
	scopes map[string]*scopeRecord
}

func grantKey(owner Owner, scope string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", owner.PID, owner.StartToken, scope)
}

// New binds an observer to its finite declared grants. Anything outside
// them is foreign.
func New(grants []Grant, limits Limits) (*Observer, error) {
	if err := checkLimits(limits); err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: declare at least one owned scope", ErrInvalid))
	}
	if len(grants) > limits.MaxScopes {
		return nil, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: declared scopes exceed the cap", ErrCapacity))
	}
	table := make(map[string]Grant, len(grants))
	for _, g := range grants {
		if !validName(g.Owner.StartToken) {
			return nil, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: malformed owner name", ErrInvalid))
		}
		if err := checkScopeName(g.Scope); err != nil {
			return nil, err
		}
		if len(g.Channels) == 0 {
			return nil, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: scope declares no effect channels", ErrInvalid))
		}
		if len(g.Channels) > len(EffectChannels) {
			return nil, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: scope declares too many channels", ErrCapacity))
		}
		seen := make(map[string]bool, len(g.Channels))
		for _, c := range g.Channels {
			if err := checkChannel(c); err != nil {
				return nil, err
			}
			if seen[c] {
				return nil, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: scope repeats channel", ErrInvalid))
			}
			seen[c] = true
		}
		key := grantKey(g.Owner, g.Scope)
		if _, dup := table[key]; dup {
			return nil, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: duplicate owned scope", ErrInvalid))
		}
		cp := make([]string, len(g.Channels))
		copy(cp, g.Channels)
		table[key] = Grant{Owner: g.Owner, Scope: g.Scope, Channels: cp}
	}
	return &Observer{limits: limits, grants: table, scopes: make(map[string]*scopeRecord)}, nil
}

// OpenScope opens one owned scope. Only an exact declared (owner, scope)
// grant admits; scopes bind to exactly one owner.
func (o *Observer) OpenScope(owner Owner, scope string) (ScopeReceipt, error) {
	if err := checkOwnerName(owner.StartToken); err != nil {
		return ScopeReceipt{}, err
	}
	if err := checkScopeName(scope); err != nil {
		return ScopeReceipt{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	grant, ok := o.grants[grantKey(owner, scope)]
	if !ok {
		return ScopeReceipt{}, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no owned grant for scope", ErrNotFound))
	}
	if prior, exists := o.scopes[scope]; exists && prior.state != stateClosed {
		if !prior.owner.same(owner) {
			return ScopeReceipt{}, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
		}
		return receiptOf(prior), nil
	}
	if len(o.scopes) >= o.limits.MaxScopes {
		return ScopeReceipt{}, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: scope table full", ErrCapacity))
	}
	token, err := mintToken("obs")
	if err != nil {
		return ScopeReceipt{}, obsErr(LayerObserver, CodeCapacity, err)
	}
	channels := make([]string, len(grant.Channels))
	copy(channels, grant.Channels)
	record := &scopeRecord{scope: scope, owner: owner, channels: channels, token: token, state: stateOpen, launches: make(map[string]*launchRecord)}
	o.scopes[scope] = record
	return receiptOf(record), nil
}

func receiptOf(record *scopeRecord) ScopeReceipt {
	channels := make([]string, len(record.channels))
	copy(channels, record.channels)
	return ScopeReceipt{Scope: record.scope, Owner: record.owner, Channels: channels, HandleDigest: digestText(record.token), Launches: len(record.launches)}
}

// Receipt reports the scope receipt: scope, owner, channels, handle
// digest, and launch count. The raw token never crosses.
func (o *Observer) Receipt(owner Owner, scope string) (ScopeReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return ScopeReceipt{}, err
	}
	return receiptOf(record), nil
}

func (o *Observer) requireScopeLocked(owner Owner, scope string) (*scopeRecord, error) {
	record, ok := o.scopes[scope]
	if !ok {
		return nil, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no open scope for owner", ErrNotFound))
	}
	if !record.owner.same(owner) {
		return nil, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
	}
	if record.state == stateClosed {
		return nil, obsErr(LayerObserver, CodeScopeClosed, fmt.Errorf("%w: scope is closed", ErrDenied))
	}
	if record.state == stateLost {
		return nil, obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: observer is lost for scope", ErrDenied))
	}
	return record, nil
}

// requireLiveBindingLocked is admission-first: the launch gate reports
// only whether a LIVE binding exists. Never opened, closed, lost,
// interrupted, or foreign all refuse with no-observer.
func (o *Observer) requireLiveBindingLocked(owner Owner, scope string) (*scopeRecord, error) {
	record, ok := o.scopes[scope]
	if !ok || !record.owner.same(owner) || record.state != stateOpen {
		return nil, obsErr(LayerAdmission, CodeNoObserver, fmt.Errorf("%w: no live observer binding for launch", ErrDenied))
	}
	return record, nil
}

// CloseScope closes one scope. Every launch interval must already be
// disposed; a live launch refuses with disposal-open.
func (o *Observer) CloseScope(owner Owner, scope string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return err
	}
	for _, launch := range record.launches {
		if !launch.disposed {
			return obsErr(LayerInterval, CodeDisposalOpen, fmt.Errorf("%w: launch interval still open", ErrDenied))
		}
	}
	record.state = stateClosed
	return nil
}

// InterruptScope interrupts the observer: the open tick's unobserved
// channels become unknown, observation suspends until ResumeScope, and
// the interrupted ticks stay unknown forever.
func (o *Observer) InterruptScope(owner Owner, scope string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return err
	}
	if record.state == stateInterrupted {
		return obsErr(LayerObserver, CodeInterrupted, fmt.Errorf("%w: scope already interrupted", ErrDenied))
	}
	// Refusal precedes effect: every live launch needs room for its
	// post-resume tick before anything is marked.
	for _, launch := range record.launches {
		if launch.disposed {
			continue
		}
		open := launch.ticks[len(launch.ticks)-1]
		if open.closed {
			continue
		}
		if len(launch.ticks) >= o.limits.MaxTicksPerLaunch {
			return obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: tick table full", ErrCapacity))
		}
	}
	record.state = stateInterrupted
	for _, launch := range record.launches {
		if launch.disposed {
			continue
		}
		open := launch.ticks[len(launch.ticks)-1]
		if open.closed {
			continue
		}
		for _, channel := range record.channels {
			if _, observed := open.entries[channel]; !observed {
				open.entries[channel] = &tickEntry{state: StateUnknown, origin: "interruption"}
			}
		}
		open.closed = true
		launch.ticks = append(launch.ticks, &tickRecord{tick: open.tick + 1, entries: make(map[string]*tickEntry)})
	}
	return nil
}

// ResumeScope resumes an interrupted observer. Interrupted ticks stay
// unknown; only new ticks observe again.
func (o *Observer) ResumeScope(owner Owner, scope string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, ok := o.scopes[scope]
	if !ok {
		return obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no open scope for owner", ErrNotFound))
	}
	if !record.owner.same(owner) {
		return obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
	}
	if record.state == stateClosed {
		return obsErr(LayerObserver, CodeScopeClosed, fmt.Errorf("%w: scope is closed", ErrDenied))
	}
	if record.state == stateLost {
		return obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: lost observer cannot resume", ErrDenied))
	}
	if record.state != stateInterrupted {
		return obsErr(LayerObserver, CodeInterrupted, fmt.Errorf("%w: scope is not interrupted", ErrDenied))
	}
	record.state = stateOpen
	return nil
}

// LoseObserver loses the observer permanently: every live interval
// becomes unknown and stays unknown; only an honest unknown-seal
// remains. Lost observers never resume and never admit new launches.
func (o *Observer) LoseObserver(owner Owner, scope string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return err
	}
	record.state = stateLost
	for _, launch := range record.launches {
		if launch.disposed {
			continue
		}
		launch.lost = true
		for _, tick := range launch.ticks {
			for _, channel := range record.channels {
				entry, ok := tick.entries[channel]
				if !ok {
					tick.entries[channel] = &tickEntry{state: StateUnknown, origin: "loss"}
				} else if entry.state == StateGap {
					entry.state = StateUnknown
					entry.origin = "loss"
				}
			}
			tick.closed = true
		}
	}
	return nil
}

// AttestLaunch attests one launch under a live observer binding. The
// launch token is minted here and verified by table lookup on every
// later call, so an invented token is never authority. AdmitLaunch is
// the same gate under its admission name: a missing observer blocks
// launch.
func (o *Observer) AttestLaunch(owner Owner, scope, launchID string) (LaunchAttestation, error) {
	if err := checkOwnerName(owner.StartToken); err != nil {
		return LaunchAttestation{}, err
	}
	if err := checkScopeName(scope); err != nil {
		return LaunchAttestation{}, err
	}
	if err := checkLaunchName(launchID); err != nil {
		return LaunchAttestation{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireLiveBindingLocked(owner, scope)
	if err != nil {
		return LaunchAttestation{}, err
	}
	total := 0
	for _, s := range o.scopes {
		total += len(s.launches)
	}
	if total >= o.limits.MaxLaunches {
		return LaunchAttestation{}, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: launch table full", ErrCapacity))
	}
	if prior, exists := record.launches[launchID]; exists {
		if !prior.disposed {
			return attestationOf(record, prior), nil
		}
		return LaunchAttestation{}, obsErr(LayerAdmission, CodeLaunchClosed, fmt.Errorf("%w: launch id is single-use and already disposed", ErrDenied))
	}
	token, err := mintToken("lnch")
	if err != nil {
		return LaunchAttestation{}, obsErr(LayerObserver, CodeCapacity, err)
	}
	launch := &launchRecord{
		launchID: launchID, scope: scope, owner: owner, token: token,
		ticks: []*tickRecord{{tick: 0, entries: make(map[string]*tickEntry)}},
	}
	record.launches[launchID] = launch
	return attestationOf(record, launch), nil
}

// AdmitLaunch admits one browser launch through the observer binding.
// Admission-first: without a live binding the launch refuses with
// no-observer before anything else is consulted.
func (o *Observer) AdmitLaunch(owner Owner, scope, launchID string) (LaunchAttestation, error) {
	return o.AttestLaunch(owner, scope, launchID)
}

func attestationOf(record *scopeRecord, launch *launchRecord) LaunchAttestation {
	channels := make([]string, len(record.channels))
	copy(channels, record.channels)
	sort.Strings(channels)
	open := launch.ticks[len(launch.ticks)-1]
	scopeDigest := digestText(fmt.Sprintf("scope:%d:%s:%s:%s", record.owner.PID, record.owner.StartToken, record.scope, joinChannels(channels)))
	return LaunchAttestation{
		LaunchID: launch.launchID, Scope: record.scope, Owner: record.owner,
		ScopeDigest: scopeDigest, ObserverDigest: digestText(record.token),
		HandleDigest: digestText(launch.token), Tick: open.tick,
	}
}

func joinChannels(channels []string) string {
	out := ""
	for i, c := range channels {
		if i > 0 {
			out += ","
		}
		out += c
	}
	return out
}

// Attestation reports the launch attestation: ids, digests, and the open
// tick. The raw token never crosses.
func (o *Observer) Attestation(owner Owner, scope, launchID string) (LaunchAttestation, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return LaunchAttestation{}, err
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return LaunchAttestation{}, err
	}
	return attestationOf(record, launch), nil
}

func requireLaunchLocked(record *scopeRecord, launchID string) (*launchRecord, error) {
	launch, ok := record.launches[launchID]
	if !ok {
		return nil, obsErr(LayerAdmission, CodeUnknownLaunch, fmt.Errorf("%w: no attested launch", ErrNotFound))
	}
	return launch, nil
}

// requireLiveLaunchLocked verifies the launch token by table lookup.
// Unknown launches and invented tokens are never authority; disposed
// launches report launch-closed.
func (o *Observer) requireLiveLaunchLocked(owner Owner, scope, launchID, token string) (*scopeRecord, *launchRecord, error) {
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return nil, nil, err
	}
	if record.state == stateInterrupted {
		return nil, nil, obsErr(LayerObserver, CodeInterrupted, fmt.Errorf("%w: observer is interrupted for scope", ErrDenied))
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return nil, nil, err
	}
	if launch.disposed {
		return nil, nil, obsErr(LayerAdmission, CodeLaunchClosed, fmt.Errorf("%w: launch is disposed", ErrDenied))
	}
	if token == "" || token != launch.token {
		return nil, nil, obsErr(LayerAdmission, CodeForgedToken, fmt.Errorf("%w: launch token is not the attested token", ErrDenied))
	}
	return record, launch, nil
}

// TokenForTest exposes the raw launch token to bounded controls. Facts
// carry digests only; the token crosses exactly here.
func (o *Observer) TokenForTest(owner Owner, scope, launchID string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return "", err
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return "", err
	}
	return launch.token, nil
}

func inScope(record *scopeRecord, channel string) bool {
	for _, c := range record.channels {
		if c == channel {
			return true
		}
	}
	return false
}

// Observe records one UI-effect observation at the launch's open tick.
// Only the launch's declared scope channels admit; a channel outside the
// finite vocabulary is an unknown effect key, and a vocabulary channel
// outside the declared scope is out of scope.
func (o *Observer) Observe(owner Owner, scope, launchID, token, channel, state string) (TickFacts, error) {
	if err := checkChannel(channel); err != nil {
		return TickFacts{}, err
	}
	if err := checkObservedState(state); err != nil {
		return TickFacts{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	record, launch, err := o.requireLiveLaunchLocked(owner, scope, launchID, token)
	if err != nil {
		return TickFacts{}, err
	}
	if !inScope(record, channel) {
		return TickFacts{}, obsErr(LayerInterval, CodeOutOfScope, fmt.Errorf("%w: channel is outside the declared scope", ErrDenied))
	}
	open := launch.ticks[len(launch.ticks)-1]
	if open.closed {
		return TickFacts{}, obsErr(LayerInterval, CodeGapOpen, fmt.Errorf("%w: tick is closed; end the tick before observing", ErrDenied))
	}
	open.entries[channel] = &tickEntry{state: state, origin: "observed"}
	return tickFactsOf(launch, open), nil
}

// EndTick closes the open tick and opens the next one. Channels with no
// observation become explicit gap entries: silence is a gap, never proof
// of absence.
func (o *Observer) EndTick(owner Owner, scope, launchID, token string) (TickFacts, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, launch, err := o.requireLiveLaunchLocked(owner, scope, launchID, token)
	if err != nil {
		return TickFacts{}, err
	}
	open := launch.ticks[len(launch.ticks)-1]
	if open.closed {
		return TickFacts{}, obsErr(LayerInterval, CodeGapOpen, fmt.Errorf("%w: tick is already closed", ErrDenied))
	}
	for _, channel := range record.channels {
		if _, observed := open.entries[channel]; !observed {
			open.entries[channel] = &tickEntry{state: StateGap, origin: "gap"}
		}
	}
	open.closed = true
	facts := tickFactsOf(launch, open)
	if len(launch.ticks) >= o.limits.MaxTicksPerLaunch {
		return TickFacts{}, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: tick table full: dispose the launch", ErrCapacity))
	}
	launch.ticks = append(launch.ticks, &tickRecord{tick: open.tick + 1, entries: make(map[string]*tickEntry)})
	return facts, nil
}

// NoteDriverDeath notes the driver's death. The interval continues: late
// host effects after driver death are observed through confirmed
// disposal, and a dead driver never seals anything on its own.
func (o *Observer) NoteDriverDeath(owner Owner, scope, launchID, token string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, launch, err := o.requireLiveLaunchLocked(owner, scope, launchID, token)
	if err != nil {
		return err
	}
	launch.driverDeath = true
	return nil
}

// PublishCorrection publishes one durable correction. Only gap and
// unknown entries admit correction, exactly once each: a directly
// observed seen/absent entry is never rewritten, and a corrected entry
// is never corrected again. Corrections are journaled in order and
// survive disposal.
func (o *Observer) PublishCorrection(owner Owner, scope, launchID, token string, tick int, channel, after string) (CorrectionFacts, error) {
	if err := checkChannel(channel); err != nil {
		return CorrectionFacts{}, err
	}
	if err := checkObservedState(after); err != nil {
		return CorrectionFacts{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	record, launch, err := o.requireLiveLaunchLocked(owner, scope, launchID, token)
	if err != nil {
		return CorrectionFacts{}, err
	}
	if !inScope(record, channel) {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeOutOfScope, fmt.Errorf("%w: channel is outside the declared scope", ErrDenied))
	}
	if tick < 0 || tick >= len(launch.ticks) || !launch.ticks[tick].closed {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionMiss, fmt.Errorf("%w: no closed tick entry", ErrNotFound))
	}
	entry, ok := launch.ticks[tick].entries[channel]
	if !ok {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionMiss, fmt.Errorf("%w: no entry for tick channel", ErrNotFound))
	}
	if entry.state != StateGap && entry.state != StateUnknown {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionCold, fmt.Errorf("%w: entry holds a direct observation", ErrDenied))
	}
	if entry.corrected {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionCold, fmt.Errorf("%w: entry is already corrected", ErrDenied))
	}
	if len(launch.corrections) >= o.limits.MaxCorrectionsPerLaunch {
		return CorrectionFacts{}, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: correction journal full", ErrCapacity))
	}
	launch.correctionSeq++
	facts := CorrectionFacts{
		CorrectionID: fmt.Sprintf("c%d", launch.correctionSeq), LaunchID: launchID,
		Tick: tick, Channel: channel, Before: entry.state, After: after,
		Digest: digestText(fmt.Sprintf("correction:%s:%d:%s:%s>%s", launchID, tick, channel, entry.state, after)),
	}
	entry.state = after
	entry.origin = "correction"
	entry.corrected = true
	launch.corrections = append(launch.corrections, facts)
	return facts, nil
}

// MarkUnknown publishes a durable unknown for one gap entry: the gap is
// explicitly unresolved, and the interval stays honestly unknown.
func (o *Observer) MarkUnknown(owner Owner, scope, launchID, token string, tick int, channel string) (CorrectionFacts, error) {
	if err := checkChannel(channel); err != nil {
		return CorrectionFacts{}, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	record, launch, err := o.requireLiveLaunchLocked(owner, scope, launchID, token)
	if err != nil {
		return CorrectionFacts{}, err
	}
	if !inScope(record, channel) {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeOutOfScope, fmt.Errorf("%w: channel is outside the declared scope", ErrDenied))
	}
	if tick < 0 || tick >= len(launch.ticks) || !launch.ticks[tick].closed {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionMiss, fmt.Errorf("%w: no closed tick entry", ErrNotFound))
	}
	entry, ok := launch.ticks[tick].entries[channel]
	if !ok {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionMiss, fmt.Errorf("%w: no entry for tick channel", ErrNotFound))
	}
	if entry.state != StateGap {
		return CorrectionFacts{}, obsErr(LayerInterval, CodeCorrectionCold, fmt.Errorf("%w: entry is not an open gap", ErrDenied))
	}
	if len(launch.corrections) >= o.limits.MaxCorrectionsPerLaunch {
		return CorrectionFacts{}, obsErr(LayerObserver, CodeCapacity, fmt.Errorf("%w: correction journal full", ErrCapacity))
	}
	launch.correctionSeq++
	facts := CorrectionFacts{
		CorrectionID: fmt.Sprintf("c%d", launch.correctionSeq), LaunchID: launchID,
		Tick: tick, Channel: channel, Before: StateGap, After: StateUnknown,
		Digest: digestText(fmt.Sprintf("correction:%s:%d:%s:gap>unknown", launchID, tick, channel)),
	}
	entry.state = StateUnknown
	entry.origin = "published-unknown"
	entry.corrected = true
	launch.corrections = append(launch.corrections, facts)
	return facts, nil
}

func canonicalTick(launchID string, tick *tickRecord) string {
	channels := make([]string, 0, len(tick.entries))
	for channel := range tick.entries {
		channels = append(channels, channel)
	}
	sort.Strings(channels)
	out := fmt.Sprintf("tick:%d:closed:", tick.tick)
	if tick.closed {
		out += "1"
	} else {
		out += "0"
	}
	for _, channel := range channels {
		entry := tick.entries[channel]
		corrected := "0"
		if entry.corrected {
			corrected = "1"
		}
		out += "|" + channel + "=" + entry.state + "/" + entry.origin + "/" + corrected
	}
	return "tick:" + launchID + ":" + out
}

func tickFactsOf(launch *launchRecord, tick *tickRecord) TickFacts {
	channels := make([]string, 0, len(tick.entries))
	for channel := range tick.entries {
		channels = append(channels, channel)
	}
	sort.Strings(channels)
	entries := make([]ChannelFacts, 0, len(channels))
	for _, channel := range channels {
		entry := tick.entries[channel]
		entries = append(entries, ChannelFacts{Channel: channel, State: entry.state, Origin: entry.origin, Corrected: entry.corrected})
	}
	return TickFacts{LaunchID: launch.launchID, Tick: tick.tick, Channels: entries, Closed: tick.closed, Digest: digestText(canonicalTick(launch.launchID, tick))}
}

// TickFacts reports one tick: its entries, closure, and digest.
func (o *Observer) TickFacts(owner Owner, scope, launchID string, tick int) (TickFacts, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, err := o.requireScopeLocked(owner, scope)
	if err != nil {
		return TickFacts{}, err
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return TickFacts{}, err
	}
	if tick < 0 || tick >= len(launch.ticks) {
		return TickFacts{}, obsErr(LayerInterval, CodeUnknownEffect, fmt.Errorf("%w: no such tick", ErrNotFound))
	}
	return tickFactsOf(launch, launch.ticks[tick]), nil
}

func intervalUnknown(launch *launchRecord) bool {
	if launch.lost {
		return true
	}
	for _, tick := range launch.ticks {
		for _, entry := range tick.entries {
			if entry.state == StateUnknown {
				return true
			}
		}
	}
	return false
}

func intervalFlags(launch *launchRecord) string {
	death, disposed, unknown := "0", "0", "0"
	if launch.driverDeath {
		death = "1"
	}
	if launch.disposed {
		disposed = "1"
	}
	if intervalUnknown(launch) {
		unknown = "1"
	}
	return "driver-death:" + death + "|disposed:" + disposed + "|unknown:" + unknown
}

// IntervalFacts reports the whole observed interval. Facts stay readable
// after disposal and after observer loss: the interval is evidence, and
// loss makes it unknown rather than unreadable.
func (o *Observer) IntervalFacts(owner Owner, scope, launchID string) (IntervalFacts, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, ok := o.scopes[scope]
	if !ok {
		return IntervalFacts{}, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no open scope for owner", ErrNotFound))
	}
	if !record.owner.same(owner) {
		return IntervalFacts{}, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
	}
	if record.state == stateClosed {
		return IntervalFacts{}, obsErr(LayerObserver, CodeScopeClosed, fmt.Errorf("%w: scope is closed", ErrDenied))
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return IntervalFacts{}, err
	}
	ticks := make([]TickFacts, 0, len(launch.ticks))
	for _, tick := range launch.ticks {
		ticks = append(ticks, tickFactsOf(launch, tick))
	}
	corrections := make([]CorrectionFacts, len(launch.corrections))
	copy(corrections, launch.corrections)
	flags := intervalFlags(launch)
	return IntervalFacts{
		LaunchID: launch.launchID, Scope: record.scope, Owner: record.owner,
		Ticks: ticks, Corrections: corrections,
		DriverDeathNoted: launch.driverDeath, Disposed: launch.disposed,
		Unknown: intervalUnknown(launch),
		Digest:  DigestInterval(launch.launchID, ticks, corrections, flags),
	}, nil
}

// CorrectionLog reports the durable correction journal: ordered and
// readable after disposal. Publication order is the journal order.
func (o *Observer) CorrectionLog(owner Owner, scope, launchID string) ([]CorrectionFacts, error) {
	facts, err := o.IntervalFacts(owner, scope, launchID)
	if err != nil {
		return nil, err
	}
	return facts.Corrections, nil
}

// SealDisposal seals the interval through confirmed disposal. Every tick
// must be closed with no open gap: fully observed open ticks close
// implicitly, but any gap entry refuses with gap-open until corrected or
// published unknown. Unknown entries seal honestly: the receipt carries
// unknown, and unknown never proves no-UI. A lost observer seals
// unknown: every remaining gap becomes unknown and the receipt says so.
func (o *Observer) SealDisposal(owner Owner, scope, launchID, token string) (DisposalReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, ok := o.scopes[scope]
	if !ok {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no open scope for owner", ErrNotFound))
	}
	if !record.owner.same(owner) {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
	}
	if record.state == stateClosed {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeScopeClosed, fmt.Errorf("%w: scope is closed", ErrDenied))
	}
	if record.state == stateInterrupted {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeInterrupted, fmt.Errorf("%w: observer is interrupted for scope", ErrDenied))
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return DisposalReceipt{}, err
	}
	if launch.disposed {
		return DisposalReceipt{}, obsErr(LayerInterval, CodeAlreadyGone, fmt.Errorf("%w: launch is already disposed", ErrDenied))
	}
	if token == "" || token != launch.token {
		return DisposalReceipt{}, obsErr(LayerAdmission, CodeForgedToken, fmt.Errorf("%w: launch token is not the attested token", ErrDenied))
	}
	if record.state == stateLost {
		for _, tick := range launch.ticks {
			for _, channel := range record.channels {
				entry, exists := tick.entries[channel]
				if !exists || entry.state == StateGap {
					corrected := false
					if exists {
						corrected = entry.corrected
					}
					tick.entries[channel] = &tickEntry{state: StateUnknown, origin: "loss", corrected: corrected}
				}
			}
			tick.closed = true
		}
	} else {
		// Refusal precedes effect: every gap is checked before any tick
		// closes, so a refused seal mutates nothing.
		open := launch.ticks[len(launch.ticks)-1]
		if !open.closed {
			for _, channel := range record.channels {
				if _, observed := open.entries[channel]; !observed {
					return DisposalReceipt{}, obsErr(LayerInterval, CodeGapOpen, fmt.Errorf("%w: open tick holds unobserved channels", ErrDenied))
				}
			}
		}
		for _, tick := range launch.ticks {
			if tick == open && !open.closed {
				continue
			}
			for _, entry := range tick.entries {
				if entry.state == StateGap {
					return DisposalReceipt{}, obsErr(LayerInterval, CodeGapOpen, fmt.Errorf("%w: tick holds an unresolved gap", ErrDenied))
				}
			}
		}
		open.closed = true
	}
	launch.disposed = true
	receipt := DisposalReceipt{
		LaunchID: launch.launchID, Scope: record.scope, Owner: record.owner,
		Ticks: len(launch.ticks), Corrections: len(launch.corrections),
		DriverDeathNoted: launch.driverDeath, Unknown: intervalUnknown(launch),
		Digest: digestText(fmt.Sprintf("disposal:%s:%d:%d:%s", launch.launchID, len(launch.ticks), len(launch.corrections), intervalFlags(launch))),
	}
	launch.disposal = receipt
	return receipt, nil
}

// DisposalReceipt reports the durable disposal receipt. An open interval
// refuses with disposal-open: a dead driver never seals on its own.
func (o *Observer) DisposalReceipt(owner Owner, scope, launchID string) (DisposalReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	record, ok := o.scopes[scope]
	if !ok {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeUnknownScope, fmt.Errorf("%w: no open scope for owner", ErrNotFound))
	}
	if !record.owner.same(owner) {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeWrongOwner, fmt.Errorf("%w: scope is owned by another identity", ErrWrongOwner))
	}
	if record.state == stateClosed {
		return DisposalReceipt{}, obsErr(LayerObserver, CodeScopeClosed, fmt.Errorf("%w: scope is closed", ErrDenied))
	}
	launch, err := requireLaunchLocked(record, launchID)
	if err != nil {
		return DisposalReceipt{}, err
	}
	if !launch.disposed {
		return DisposalReceipt{}, obsErr(LayerInterval, CodeDisposalOpen, fmt.Errorf("%w: launch interval is still open", ErrDenied))
	}
	return launch.disposal, nil
}

// ---------------------------------------------------------------------------
// Admission adapter: the typed launch gate
// ---------------------------------------------------------------------------

// ProofClaim is one no-UI proof claim. Only Kind "observer-attestation"
// with carried interval facts is admissible; every other kind refuses
// with forbidden-proof before any verdict is read.
type ProofClaim struct {
	Kind     string
	Detail   string
	Interval IntervalFacts
}

// AssertNoUiProof judges one no-UI proof claim. Only a verified observer
// attestation over a fully observed interval with zero seen effects
// proves no-UI: headless flags, HOME values, launch flag sets, process
// samples, and any other non-observer claim refuse with forbidden-proof,
// and a forged interval digest never verifies.
func AssertNoUiProof(claim ProofClaim) (NoUiVerdict, error) {
	if claim.Kind != "observer-attestation" {
		return NoUiVerdict{}, obsErr(LayerAdmission, CodeForbiddenProof, fmt.Errorf("%w: no-UI proof kind is inadmissible", ErrDenied))
	}
	interval := claim.Interval
	death, disposed, unknown := "0", "0", "0"
	if interval.DriverDeathNoted {
		death = "1"
	}
	if interval.Disposed {
		disposed = "1"
	}
	if interval.Unknown {
		unknown = "1"
	}
	flags := "driver-death:" + death + "|disposed:" + disposed + "|unknown:" + unknown
	if DigestInterval(interval.LaunchID, interval.Ticks, interval.Corrections, flags) != interval.Digest {
		return NoUiVerdict{}, obsErr(LayerAdmission, CodeForbiddenProof, fmt.Errorf("%w: observer attestation digest does not verify", ErrDenied))
	}
	uiSeen := false
	for _, tick := range interval.Ticks {
		if !tick.Closed {
			return NoUiVerdict{}, obsErr(LayerAdmission, CodeForbiddenProof, fmt.Errorf("%w: attested interval holds an open tick", ErrDenied))
		}
		for _, entry := range tick.Channels {
			if entry.State == StateSeen {
				uiSeen = true
			}
			if entry.State == StateGap {
				return NoUiVerdict{}, obsErr(LayerAdmission, CodeForbiddenProof, fmt.Errorf("%w: attested interval holds an open gap", ErrDenied))
			}
		}
	}
	seen, unk := "0", "0"
	if uiSeen {
		seen = "1"
	}
	if interval.Unknown {
		unk = "1"
	}
	return NoUiVerdict{
		LaunchID: interval.LaunchID, UISeen: uiSeen, Unknown: interval.Unknown,
		Digest: digestText("verdict:" + interval.LaunchID + ":" + seen + ":" + unk),
	}, nil
}

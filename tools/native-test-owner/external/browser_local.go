// Outside-service N child-discovery/containment/reclaim adapter: K08's Go
// mirror.
//
// This file implements the ACTUAL outside-service adapter under independent
// N ownership, over in-memory doubles only. No browser is launched, no
// process is spawned or sampled, no environment is read, and no flags are
// inspected. The authority admits pinned drivers, binds qualified observer
// scopes, declares launches, records launcher-reported children, observes
// children externally, contains them, and reclaims them with an opaque
// minted witness token verified by lookup. Driver death mid-launch cannot
// orphan children: launcher reports stop at death while external
// observation, containment, and reclaim continue, and every reported or
// observed child must be contained before the seal. A receipt from the
// dead driver is never its own cleanup witness: only a verified external
// witness judges cleanup.
//
// Independence scope from the launch/driver service under test
// (normative, mirrored in
// tools/runtime/test-services/browser-driver/launch.ts): the authority
// shares no driver code, decoded values, receipts, flags, environment, or
// process state; it proves pinned-launcher admission, child discovery,
// containment, externally witnessed reclaim, and layered error provenance
// from its own seeded doubles alone. Driver claims are compared against
// these facts; no cleanup fact is ever derived from a driver claim.
// Consequently a driver receipt can never prove cleanup: only an external
// witness over every known child proves it.
//
// Split with the TS service: the TS service owns the typed launch gate,
// identity capture, context exposure, and the witness-verifier seam plus
// the bounded self-check. This file owns the same contract as the
// outside-service adapter: pinned-driver admission, external observation,
// containment, reclaim, and witness minting. Both enforce pinned
// launchers, opaque tokens verified by lookup, digest-only facts, and
// layered errors. Neither launches a browser.
//
// Pure in-memory mechanics: no I/O, timers, transports, services,
// browsers, processes, or live runtimes. Local controls only. Actual
// launch, host effects, and service-death controls are QB0.

package external

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// BrowserLocalLayer names the failing layer of one adapter error. Every
// adapter failure names its layer; nothing unclassified is returned.
type BrowserLocalLayer string

// Adapter layers. Service owns pinned drivers, the observer gate,
// ownership, and capacity; launch owns launch binding, tokens, death, and
// the reporting window; reclaim owns witness admission, children,
// orphans, and the reclaim seal.
const (
	BrowserLocalLayerService BrowserLocalLayer = "service"
	BrowserLocalLayerLaunch  BrowserLocalLayer = "launch"
	BrowserLocalLayerReclaim BrowserLocalLayer = "reclaim"
)

// Closed failure-code vocabulary. Each code maps to exactly one layer,
// mirroring the TS service's vocabulary for this facet.
const (
	BrowserLocalCodeUnknownDriver    = "unknown-driver"     // service
	BrowserLocalCodeLauncherUnpinned = "launcher-unpinned"  // service
	BrowserLocalCodeNoObserver       = "no-observer"        // service
	BrowserLocalCodeWrongOwner       = "wrong-owner"        // service
	BrowserLocalCodeCapacity         = "capacity-exhausted" // service
	BrowserLocalCodeForgedToken      = "forged-token"       // launch
	BrowserLocalCodeUnknownLaunch    = "unknown-launch"     // launch
	BrowserLocalCodeLaunchClosed     = "launch-closed"      // launch
	BrowserLocalCodeDriverDead       = "driver-dead"        // launch
	BrowserLocalCodeForbiddenWitness = "forbidden-witness"  // reclaim
	BrowserLocalCodeForgedWitness    = "forged-witness"     // reclaim
	BrowserLocalCodeUnknownChild     = "unknown-child"      // reclaim
	BrowserLocalCodeOrphanOpen       = "orphan-open"        // reclaim
	BrowserLocalCodeAlreadyReclaimed = "already-reclaimed"  // reclaim
)

// BrowserLocalCodes lists the closed vocabulary in layer order.
var BrowserLocalCodes = []string{
	BrowserLocalCodeUnknownDriver, BrowserLocalCodeLauncherUnpinned,
	BrowserLocalCodeNoObserver, BrowserLocalCodeWrongOwner,
	BrowserLocalCodeCapacity, BrowserLocalCodeForgedToken,
	BrowserLocalCodeUnknownLaunch, BrowserLocalCodeLaunchClosed,
	BrowserLocalCodeDriverDead, BrowserLocalCodeForbiddenWitness,
	BrowserLocalCodeForgedWitness, BrowserLocalCodeUnknownChild,
	BrowserLocalCodeOrphanOpen, BrowserLocalCodeAlreadyReclaimed,
}

// BrowserLocalLayerOfCode maps one closed code to its single layer.
func BrowserLocalLayerOfCode(code string) (BrowserLocalLayer, bool) {
	switch code {
	case BrowserLocalCodeUnknownDriver, BrowserLocalCodeLauncherUnpinned,
		BrowserLocalCodeNoObserver, BrowserLocalCodeWrongOwner,
		BrowserLocalCodeCapacity:
		return BrowserLocalLayerService, true
	case BrowserLocalCodeForgedToken, BrowserLocalCodeUnknownLaunch,
		BrowserLocalCodeLaunchClosed, BrowserLocalCodeDriverDead:
		return BrowserLocalLayerLaunch, true
	case BrowserLocalCodeForbiddenWitness, BrowserLocalCodeForgedWitness,
		BrowserLocalCodeUnknownChild, BrowserLocalCodeOrphanOpen,
		BrowserLocalCodeAlreadyReclaimed:
		return BrowserLocalLayerReclaim, true
	default:
		return "", false
	}
}

// Adapter bounds. Every adapter table is finite; nothing here means
// unlimited.
const (
	// MaxLocalDrivers caps pinned drivers per authority.
	MaxLocalDrivers = 16
	// MaxLocalObservers caps qualified observer scopes per authority.
	MaxLocalObservers = 16
	// MaxLocalLaunches caps declared launches per authority.
	MaxLocalLaunches = 64
	// MaxLocalChildren caps children known per launch.
	MaxLocalChildren = 64
)

// BrowserLocalError is one layered adapter failure. Layer always names
// service, launch, or reclaim; Code is the closed failure code; Err
// carries the wrapped package cause for errors.Is matching.
type BrowserLocalError struct {
	Layer BrowserLocalLayer
	Code  string
	Err   error
}

func (e *BrowserLocalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %v", e.Layer, e.Code, e.Err)
	}
	return fmt.Sprintf("[%s:%s] adapter refused", e.Layer, e.Code)
}

// Unwrap exposes the wrapped package cause.
func (e *BrowserLocalError) Unwrap() error { return e.Err }

func localErr(layer BrowserLocalLayer, code string, err error) *BrowserLocalError {
	return &BrowserLocalError{Layer: layer, Code: code, Err: err}
}

func checkLocalOwner(self journal.Owner) error {
	if self.PID <= 0 || !validID(self.StartToken) {
		return localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: owner needs a positive PID and a valid spawn-start token", ErrInvalid))
	}
	return nil
}

func checkLocalDriverName(s, what string) error {
	if !validID(s) {
		return localErr(BrowserLocalLayerService, BrowserLocalCodeUnknownDriver, fmt.Errorf("%w: malformed %s name", ErrInvalid, what))
	}
	return nil
}

func checkLocalScope(s string) error {
	if !validID(s) {
		return localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: malformed scope name", ErrInvalid))
	}
	return nil
}

func checkLocalLaunchID(s string) error {
	if !validID(s) {
		return localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: malformed launch ID", ErrInvalid))
	}
	return nil
}

func checkLocalChild(s string) error {
	if !validID(s) {
		return localErr(BrowserLocalLayerReclaim, BrowserLocalCodeUnknownChild, fmt.Errorf("%w: malformed child identity", ErrInvalid))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Facts (digest-only; returned by value)
// ---------------------------------------------------------------------------

// LocalBindingFacts binds one qualified observer scope to its owner and
// handle digest. The raw token never crosses.
type LocalBindingFacts struct {
	Owner        journal.Owner
	Scope        string
	HandleDigest string
}

// LocalLaunchFacts names one declared launch: the pinned driver and
// launcher, the requested profile, the launcher-reported children, the
// externally observed children, the contained children, and the token
// digest. Raw tokens never cross.
type LocalLaunchFacts struct {
	LaunchID    string
	Driver      string
	Launcher    string
	Profile     string
	Reported    []string
	Observed    []string
	Contained   []string
	DriverDead  bool
	Reclaimed   bool
	TokenDigest string
}

// LocalWitnessFacts names one witnessed reclaim: the launch, the
// reclaimed children, containment, and the witness token digest.
type LocalWitnessFacts struct {
	LaunchID    string
	Children    []string
	Contained   bool
	TokenDigest string
}

// LocalReclaimReceipt seals the launch through externally witnessed
// reclaim.
type LocalReclaimReceipt struct {
	LaunchID   string
	Owner      journal.Owner
	Driver     string
	Launcher   string
	Profile    string
	Children   []string
	DriverDead bool
	Digest     string
}

// LocalCleanupClaim is one cleanup proof claim. Only Kind
// "external-witness" with a verifying receipt judges; every other kind
// refuses with forbidden-witness before any verdict is read.
type LocalCleanupClaim struct {
	Kind    string
	Receipt LocalReclaimReceipt
	Detail  string
}

// LocalCleanupVerdict judges one verified external witness.
type LocalCleanupVerdict struct {
	LaunchID  string
	Witnessed bool
	Children  []string
	Digest    string
}

// ForbiddenWitnessKinds are the inadmissible cleanup proof kinds.
var ForbiddenWitnessKinds = []string{"driver-receipt", "launcher-log", "exit-code"}

// DigestLocalReclaim recomputes the reclaim digest from carried fields,
// so AssertLocalCleanup re-verifies a presented receipt instead of
// trusting its digest string.
func DigestLocalReclaim(launchID string, owner journal.Owner, driver, launcher, profile string, children []string, driverDead bool) string {
	h := sha256.New()
	h.Write([]byte(launchID))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d\x00%s", owner.PID, owner.StartToken)))
	h.Write([]byte{0})
	h.Write([]byte(driver))
	h.Write([]byte{0})
	h.Write([]byte(launcher))
	h.Write([]byte{0})
	h.Write([]byte(profile))
	h.Write([]byte{0})
	cp := append([]string(nil), children...)
	sort.Strings(cp)
	for _, c := range cp {
		h.Write([]byte(c))
		h.Write([]byte(";"))
	}
	h.Write([]byte{0})
	if driverDead {
		h.Write([]byte("dead"))
	} else {
		h.Write([]byte("live"))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func digestLocalText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type localBindingState string

const (
	localBindingOpen localBindingState = "open"
	localBindingLost localBindingState = "lost"
)

type localBinding struct {
	scope string
	owner journal.Owner
	token string
	state localBindingState
}

type localLaunch struct {
	launchID   string
	scope      string
	owner      journal.Owner
	driver     string
	launcher   string
	profile    string
	token      string
	reported   map[string]bool
	observed   map[string]bool
	contained  map[string]bool
	driverDead bool
	reclaimed  bool
	witness    string // minted witness token; "" until reclaimed
	receipt    LocalReclaimReceipt
}

// LocalDriverGrant pins one driver's launcher for one owner. Only exact
// declared grants admit.
type LocalDriverGrant struct {
	Owner    journal.Owner
	Driver   string
	Launcher string
}

// LocalObserverGrant declares one qualified observer scope for one owner.
type LocalObserverGrant struct {
	Owner journal.Owner
	Scope string
}

// LocalAuthority is the outside-service N child-discovery, containment,
// and reclaim adapter. It never reads the driver: discovery,
// containment, and reclaim come from its own table alone. A
// LocalAuthority is safe for concurrent use.
type LocalAuthority struct {
	mu        sync.Mutex
	drivers   map[string]LocalDriverGrant
	observers map[string]LocalObserverGrant
	bindings  map[string]*localBinding
	launches  map[string]*localLaunch
}

func localGrantKey(owner journal.Owner, name string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", owner.PID, owner.StartToken, name)
}

// NewLocalAuthority binds an authority to its finite pinned drivers and
// qualified observer scopes. Anything outside them is foreign.
func NewLocalAuthority(drivers []LocalDriverGrant, observers []LocalObserverGrant) (*LocalAuthority, error) {
	if len(drivers) == 0 {
		return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeUnknownDriver, fmt.Errorf("%w: declare at least one pinned driver", ErrInvalid))
	}
	if len(drivers) > MaxLocalDrivers {
		return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: declared drivers exceed the cap", ErrCapacity))
	}
	if len(observers) == 0 {
		return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: declare at least one qualified observer scope", ErrInvalid))
	}
	if len(observers) > MaxLocalObservers {
		return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: declared observers exceed the cap", ErrCapacity))
	}
	dtable := make(map[string]LocalDriverGrant, len(drivers))
	for _, g := range drivers {
		if err := checkLocalOwner(g.Owner); err != nil {
			return nil, err
		}
		if err := checkLocalDriverName(g.Driver, "driver"); err != nil {
			return nil, err
		}
		if err := checkLocalDriverName(g.Launcher, "launcher"); err != nil {
			return nil, err
		}
		key := localGrantKey(g.Owner, g.Driver)
		if _, dup := dtable[key]; dup {
			return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeUnknownDriver, fmt.Errorf("%w: duplicate pinned driver", ErrInvalid))
		}
		dtable[key] = g
	}
	otable := make(map[string]LocalObserverGrant, len(observers))
	for _, g := range observers {
		if err := checkLocalOwner(g.Owner); err != nil {
			return nil, err
		}
		if err := checkLocalScope(g.Scope); err != nil {
			return nil, err
		}
		key := localGrantKey(g.Owner, g.Scope)
		if _, dup := otable[key]; dup {
			return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: duplicate qualified observer scope", ErrInvalid))
		}
		otable[key] = g
	}
	return &LocalAuthority{
		drivers:   dtable,
		observers: otable,
		bindings:  make(map[string]*localBinding),
		launches:  make(map[string]*localLaunch),
	}, nil
}

func sameLocalOwner(a, b journal.Owner) bool {
	return a.PID == b.PID && a.StartToken == b.StartToken
}

// BindObserver binds one qualified observer scope. Only an exact
// declared (owner, scope) grant admits; bindings hold one owner.
func (a *LocalAuthority) BindObserver(self journal.Owner, scope string) (LocalBindingFacts, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalBindingFacts{}, err
	}
	if err := checkLocalScope(scope); err != nil {
		return LocalBindingFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.observers[localGrantKey(self, scope)]; !ok {
		return LocalBindingFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: no qualified grant for observer scope", ErrDenied))
	}
	if prior, ok := a.bindings[scope]; ok {
		if !sameLocalOwner(prior.owner, self) {
			return LocalBindingFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: observer scope is owned by another identity", ErrWrongOwner))
		}
		return LocalBindingFacts{Owner: prior.owner, Scope: prior.scope, HandleDigest: handleDigest(prior.token)}, nil
	}
	if len(a.bindings) >= MaxLocalObservers {
		return LocalBindingFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: observer binding table full", ErrCapacity))
	}
	tok, err := mintToken()
	if err != nil {
		return LocalBindingFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, err)
	}
	a.bindings[scope] = &localBinding{scope: scope, owner: self, token: tok, state: localBindingOpen}
	return LocalBindingFacts{Owner: self, Scope: scope, HandleDigest: handleDigest(tok)}, nil
}

// LoseObserver loses the binding permanently: later declarations refuse
// with no-observer, while declared launch facts stay readable.
func (a *LocalAuthority) LoseObserver(self journal.Owner, scope string) error {
	if err := checkLocalOwner(self); err != nil {
		return err
	}
	if err := checkLocalScope(scope); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	b, ok := a.bindings[scope]
	if !ok || !sameLocalOwner(b.owner, self) {
		return localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: no live observer binding to lose", ErrDenied))
	}
	b.state = localBindingLost
	return nil
}

// liveBindingLocked reports only whether a LIVE qualified binding
// exists. Callers hold the lock.
func (a *LocalAuthority) liveBindingLocked(self journal.Owner, scope string) bool {
	b, ok := a.bindings[scope]
	return ok && sameLocalOwner(b.owner, self) && b.state == localBindingOpen
}

// DeclareLaunch declares one launch through the pinned driver's
// launcher. Admission-first: the observer binding is checked before
// anything else, and the presented launcher must equal the pinned
// launcher exactly. The launch token is minted here and verified by
// table lookup on every later call; the returned facts carry its digest
// only. Re-declaring the identical launch joins; re-declaring the ID
// with different arguments is refused.
func (a *LocalAuthority) DeclareLaunch(self journal.Owner, scope, launchID, driver, launcher, profile string) (LocalLaunchFacts, string, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	if err := checkLocalScope(scope); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.liveBindingLocked(self, scope) {
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeNoObserver, fmt.Errorf("%w: no live qualified observer binding for launch", ErrDenied))
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	if err := checkLocalDriverName(driver, "driver"); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	if err := checkLocalDriverName(launcher, "launcher"); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	if err := checkLocalDriverName(profile, "profile"); err != nil {
		return LocalLaunchFacts{}, "", err
	}
	grant, ok := a.drivers[localGrantKey(self, driver)]
	if !ok {
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeUnknownDriver, fmt.Errorf("%w: no pinned launcher grant for driver", ErrDenied))
	}
	if grant.Launcher != launcher {
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeLauncherUnpinned, fmt.Errorf("%w: launcher is not the pinned launcher for driver", ErrDenied))
	}
	if prior, dup := a.launches[launchID]; dup {
		if sameLocalOwner(prior.owner, self) && prior.scope == scope && prior.driver == driver && prior.launcher == launcher && prior.profile == profile {
			return a.factsLocked(prior), prior.token, nil
		}
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: launch ID is already declared", ErrInvalid))
	}
	if len(a.launches) >= MaxLocalLaunches {
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: launch table full", ErrCapacity))
	}
	tok, err := mintToken()
	if err != nil {
		return LocalLaunchFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, err)
	}
	rec := &localLaunch{
		launchID: launchID, scope: scope, owner: self,
		driver: driver, launcher: launcher, profile: profile, token: tok,
		reported: make(map[string]bool), observed: make(map[string]bool), contained: make(map[string]bool),
	}
	a.launches[launchID] = rec
	return a.factsLocked(rec), tok, nil
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (a *LocalAuthority) factsLocked(rec *localLaunch) LocalLaunchFacts {
	return LocalLaunchFacts{
		LaunchID: rec.launchID, Driver: rec.driver, Launcher: rec.launcher, Profile: rec.profile,
		Reported: sortedKeys(rec.reported), Observed: sortedKeys(rec.observed), Contained: sortedKeys(rec.contained),
		DriverDead: rec.driverDead, Reclaimed: rec.reclaimed, TokenDigest: handleDigest(rec.token),
	}
}

// liveLaunchLocked verifies one launch and its token by table lookup.
// Unknown launches reject before the token is even read; invented tokens
// are never authority. Callers hold the lock.
func (a *LocalAuthority) liveLaunchLocked(self journal.Owner, launchID, token string) (*localLaunch, error) {
	rec, ok := a.launches[launchID]
	if !ok {
		return nil, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: launch %q", ErrNotFound, launchID))
	}
	if !sameLocalOwner(rec.owner, self) {
		return nil, localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: launch is owned by another identity", ErrWrongOwner))
	}
	if rec.reclaimed {
		return nil, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeLaunchClosed, fmt.Errorf("%w: launch is reclaimed", ErrInvalid))
	}
	if token == "" || token != rec.token {
		return nil, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeForgedToken, fmt.Errorf("%w: launch token is not the declared token", ErrDenied))
	}
	return rec, nil
}

func checkLocalChildren(children []string, cap int) error {
	seen := make(map[string]bool, len(children))
	for _, c := range children {
		if err := checkLocalChild(c); err != nil {
			return err
		}
		if seen[c] {
			return localErr(BrowserLocalLayerReclaim, BrowserLocalCodeUnknownChild, fmt.Errorf("%w: child identities repeat a child", ErrInvalid))
		}
		seen[c] = true
	}
	if len(children) > cap {
		return localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: child table full for launch", ErrCapacity))
	}
	return nil
}

// ReportChildren records the launcher-reported child set. The reporting
// window closes at driver death: a dead launcher reports nothing
// further, while external observation continues.
func (a *LocalAuthority) ReportChildren(self journal.Owner, launchID, token string, children []string) (LocalLaunchFacts, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalLaunchFacts{}, err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalLaunchFacts{}, err
	}
	if err := checkLocalChildren(children, MaxLocalChildren); err != nil {
		return LocalLaunchFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveLaunchLocked(self, launchID, token)
	if err != nil {
		return LocalLaunchFacts{}, err
	}
	if rec.driverDead {
		return LocalLaunchFacts{}, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeDriverDead, fmt.Errorf("%w: launcher is dead for launch", ErrDenied))
	}
	rec.reported = make(map[string]bool, len(children))
	for _, c := range children {
		rec.reported[c] = true
	}
	return a.factsLocked(rec), nil
}

// ObserveChildren records externally observed children: the adapter's
// own discovery, never a driver claim. Observation is additive and
// continues after driver death; that is how death mid-launch orphans
// nothing.
func (a *LocalAuthority) ObserveChildren(self journal.Owner, launchID, token string, children []string) (LocalLaunchFacts, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalLaunchFacts{}, err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalLaunchFacts{}, err
	}
	if err := checkLocalChildren(children, MaxLocalChildren); err != nil {
		return LocalLaunchFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveLaunchLocked(self, launchID, token)
	if err != nil {
		return LocalLaunchFacts{}, err
	}
	known := make(map[string]bool, len(rec.reported)+len(rec.observed)+len(children))
	for c := range rec.reported {
		known[c] = true
	}
	for c := range rec.observed {
		known[c] = true
	}
	for _, c := range children {
		known[c] = true
	}
	if len(known) > MaxLocalChildren {
		return LocalLaunchFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, fmt.Errorf("%w: child table full for launch", ErrCapacity))
	}
	for _, c := range children {
		rec.observed[c] = true
	}
	return a.factsLocked(rec), nil
}

// NoteDriverDeath notes the driver's death. Reporting stops; external
// observation, containment, and reclaim proceed.
func (a *LocalAuthority) NoteDriverDeath(self journal.Owner, launchID, token string) error {
	if err := checkLocalOwner(self); err != nil {
		return err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveLaunchLocked(self, launchID, token)
	if err != nil {
		return err
	}
	rec.driverDead = true
	return nil
}

// Discover reads the externally known launch facts. Discovery stays
// readable after death and after reclaim: the launch is evidence.
func (a *LocalAuthority) Discover(self journal.Owner, launchID string) (LocalLaunchFacts, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalLaunchFacts{}, err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalLaunchFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.launches[launchID]
	if !ok {
		return LocalLaunchFacts{}, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: launch %q", ErrNotFound, launchID))
	}
	if !sameLocalOwner(rec.owner, self) {
		return LocalLaunchFacts{}, localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: launch is owned by another identity", ErrWrongOwner))
	}
	return a.factsLocked(rec), nil
}

// Contain contains one known child: reported or externally observed.
// Unknown children refuse; containment continues after driver death.
func (a *LocalAuthority) Contain(self journal.Owner, launchID, token, child string) error {
	if err := checkLocalOwner(self); err != nil {
		return err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return err
	}
	if err := checkLocalChild(child); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, err := a.liveLaunchLocked(self, launchID, token)
	if err != nil {
		return err
	}
	if !rec.reported[child] && !rec.observed[child] {
		return localErr(BrowserLocalLayerReclaim, BrowserLocalCodeUnknownChild, fmt.Errorf("%w: child %q is not a known child", ErrNotFound, child))
	}
	rec.contained[child] = true
	return nil
}

// Reclaim reclaims the launch under external ownership. Every reported
// and every observed child must already be contained or the seal names
// the orphans; nothing uncontained seals. The witness token is minted
// here and verified by table lookup in VerifyWitness; the returned facts
// carry its digest only and the raw token crosses alongside exactly
// once.
func (a *LocalAuthority) Reclaim(self journal.Owner, launchID, token string) (LocalWitnessFacts, string, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalWitnessFacts{}, "", err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalWitnessFacts{}, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.launches[launchID]
	if !ok {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: launch %q", ErrNotFound, launchID))
	}
	if !sameLocalOwner(rec.owner, self) {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: launch is owned by another identity", ErrWrongOwner))
	}
	if rec.reclaimed {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerReclaim, BrowserLocalCodeAlreadyReclaimed, fmt.Errorf("%w: launch is already reclaimed", ErrInvalid))
	}
	if token == "" || token != rec.token {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerLaunch, BrowserLocalCodeForgedToken, fmt.Errorf("%w: launch token is not the declared token", ErrDenied))
	}
	var missing []string
	for c := range rec.reported {
		if !rec.contained[c] {
			missing = append(missing, c)
		}
	}
	for c := range rec.observed {
		if !rec.contained[c] {
			missing = append(missing, c)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerReclaim, BrowserLocalCodeOrphanOpen, fmt.Errorf("%w: children lack containment: %v", ErrDenied, missing))
	}
	wtok, err := mintToken()
	if err != nil {
		return LocalWitnessFacts{}, "", localErr(BrowserLocalLayerService, BrowserLocalCodeCapacity, err)
	}
	rec.witness = wtok
	rec.reclaimed = true
	children := sortedKeys(rec.contained)
	rec.receipt = LocalReclaimReceipt{
		LaunchID: rec.launchID, Owner: rec.owner, Driver: rec.driver,
		Launcher: rec.launcher, Profile: rec.profile, Children: children,
		DriverDead: rec.driverDead,
		Digest:     DigestLocalReclaim(rec.launchID, rec.owner, rec.driver, rec.launcher, rec.profile, children, rec.driverDead),
	}
	return LocalWitnessFacts{
		LaunchID: rec.launchID, Children: children, Contained: true, TokenDigest: handleDigest(wtok),
	}, wtok, nil
}

// VerifyWitness verifies one witness token by table lookup. Unknown
// tokens and cross-launch tokens are never authority.
func (a *LocalAuthority) VerifyWitness(launchID, token string) (LocalWitnessFacts, error) {
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalWitnessFacts{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, rec := range a.launches {
		if rec.witness != "" && rec.witness == token {
			if id != launchID {
				return LocalWitnessFacts{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForgedWitness, fmt.Errorf("%w: witness is for another launch", ErrDenied))
			}
			children := append([]string(nil), rec.receipt.Children...)
			return LocalWitnessFacts{LaunchID: id, Children: children, Contained: true, TokenDigest: handleDigest(token)}, nil
		}
	}
	return LocalWitnessFacts{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForgedWitness, fmt.Errorf("%w: witness token is unknown", ErrNotFound))
}

// ReclaimReceipt reads the sealed receipt. Before the seal the reclaim
// is still open.
func (a *LocalAuthority) ReclaimReceipt(self journal.Owner, launchID string) (LocalReclaimReceipt, error) {
	if err := checkLocalOwner(self); err != nil {
		return LocalReclaimReceipt{}, err
	}
	if err := checkLocalLaunchID(launchID); err != nil {
		return LocalReclaimReceipt{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	rec, ok := a.launches[launchID]
	if !ok {
		return LocalReclaimReceipt{}, localErr(BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch, fmt.Errorf("%w: launch %q", ErrNotFound, launchID))
	}
	if !sameLocalOwner(rec.owner, self) {
		return LocalReclaimReceipt{}, localErr(BrowserLocalLayerService, BrowserLocalCodeWrongOwner, fmt.Errorf("%w: launch is owned by another identity", ErrWrongOwner))
	}
	if !rec.reclaimed {
		return LocalReclaimReceipt{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeOrphanOpen, fmt.Errorf("%w: launch reclaim is still open", ErrDenied))
	}
	out := rec.receipt
	out.Children = append([]string(nil), rec.receipt.Children...)
	return out, nil
}

// AssertLocalCleanup judges one cleanup proof claim. Only a verified
// external-witness receipt proves cleanup: driver receipts, launcher
// logs, exit codes, and any other non-witness claim refuse with
// forbidden-witness before any verdict is read, and a forged receipt
// never verifies.
func AssertLocalCleanup(claim LocalCleanupClaim) (LocalCleanupVerdict, error) {
	if claim.Kind != "external-witness" {
		return LocalCleanupVerdict{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForbiddenWitness, fmt.Errorf("%w: cleanup proof kind is inadmissible: %q", ErrDenied, claim.Kind))
	}
	r := claim.Receipt
	if !validID(r.LaunchID) || !validID(r.Driver) || !validID(r.Launcher) || !validID(r.Profile) {
		return LocalCleanupVerdict{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForbiddenWitness, fmt.Errorf("%w: external witness carries a malformed receipt", ErrInvalid))
	}
	if r.Owner.PID <= 0 || !validID(r.Owner.StartToken) {
		return LocalCleanupVerdict{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForbiddenWitness, fmt.Errorf("%w: external witness carries a malformed receipt", ErrInvalid))
	}
	for _, c := range r.Children {
		if !validID(c) {
			return LocalCleanupVerdict{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForbiddenWitness, fmt.Errorf("%w: external witness carries a malformed receipt", ErrInvalid))
		}
	}
	recomputed := DigestLocalReclaim(r.LaunchID, r.Owner, r.Driver, r.Launcher, r.Profile, r.Children, r.DriverDead)
	if recomputed != r.Digest {
		return LocalCleanupVerdict{}, localErr(BrowserLocalLayerReclaim, BrowserLocalCodeForbiddenWitness, fmt.Errorf("%w: witness receipt digest does not verify", ErrDenied))
	}
	return LocalCleanupVerdict{
		LaunchID: r.LaunchID, Witnessed: true, Children: append([]string(nil), r.Children...),
		Digest: digestLocalText("verdict:" + r.LaunchID + ":" + fmt.Sprintf("%v", append([]string(nil), r.Children...))),
	}, nil
}

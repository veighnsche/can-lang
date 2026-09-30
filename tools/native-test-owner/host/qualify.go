package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
)

// Battery check names. Every Qualify run records every name so records
// compare across adapters and envelopes.
const (
	CheckProbeAvailable   = "probe-available"
	CheckMechanism        = "mechanism-declared"
	CheckRefuseProc       = "sync-refusal-proc"
	CheckRefuseMem        = "sync-refusal-mem"
	CheckRefuseTmp        = "sync-refusal-tmp"
	CheckOvershootFinite  = "overshoot-finite"
	CheckDetachedCharged  = "detached-charged"
	CheckBypassCaught     = "bypass-caught"
	CheckOverflowClosed   = "bypass-overflow-fail-closed"
	CheckDisposalReserve  = "disposal-reserve"
	CheckHostFit          = "host-fit"
	CheckOpenRealEnvelope = "open-real-envelope"
	CheckKernelNproc      = "kernel-nproc"
	CheckKernelFsize      = "kernel-fsize"
	CheckMemBackstopProbe = "mem-backstop-probe"
)

// CheckResult is one battery outcome with its evidence detail.
type CheckResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// Qualification is the battery record for one adapter plus envelope.
// Valid reports whether every check passed; only then may Admit open
// scopes under it.
type Qualification struct {
	AdapterName string        `json:"adapter"`
	Mechanisms  []Mechanism   `json:"mechanisms"`
	Envelope    Envelope      `json:"envelope"`
	Facts       HostFacts     `json:"facts"`
	Overshoot   Overshoot     `json:"overshoot"`
	Checks      []CheckResult `json:"checks"`
	AtUnixMilli int64         `json:"atUnixMilli"`

	adapter Adapter
}

// Valid reports whether the battery ran to completion with every check
// passing. A nil qualification is never valid.
func (q *Qualification) Valid() bool {
	if q == nil || q.adapter == nil || len(q.Checks) == 0 {
		return false
	}
	for _, c := range q.Checks {
		if !c.Pass {
			return false
		}
	}
	return true
}

// Check returns one check outcome by name.
func (q *Qualification) Check(name string) (CheckResult, bool) {
	if q == nil {
		return CheckResult{}, false
	}
	for _, c := range q.Checks {
		if c.Name == name {
			return c, true
		}
	}
	return CheckResult{}, false
}

// Failing lists the names of failed checks for negative-control evidence.
func (q *Qualification) Failing() []string {
	var out []string
	if q == nil {
		return []string{"no-qualification"}
	}
	for _, c := range q.Checks {
		if !c.Pass {
			out = append(out, c.Name)
		}
	}
	return out
}

// QualifyOpts tunes the battery. TmpParent is required: an absolute
// existing directory holding all battery tmp roots (tests pass a
// t.TempDir).
type QualifyOpts struct {
	TmpParent string
}

// Qualify runs the full battery for one adapter plus envelope and
// returns the record. The error reports only that the battery could
// not run (bad inputs, tmp unavailable); pass or fail is read from
// Valid. Behavioral checks run on small control envelopes; the real
// envelope is proven by charged-scope arithmetic against probed host
// facts plus an open/dispose round-trip, never by forking 64 processes
// or allocating GiBs.
func Qualify(a Adapter, env Envelope, opts QualifyOpts) (*Qualification, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: nil adapter", ErrInvalid)
	}
	if err := env.valid(); err != nil {
		return nil, err
	}
	if err := checkTmpParent(opts.TmpParent); err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp(opts.TmpParent, "qual-*")
	if err != nil {
		return nil, fmt.Errorf("host: create battery workdir: %w", err)
	}
	defer os.RemoveAll(work)

	q := &Qualification{
		AdapterName: a.Name(),
		Envelope:    env,
		AtUnixMilli: time.Now().UnixMilli(),
		adapter:     a,
	}
	caps := a.Probe(work)
	q.Mechanisms = append([]Mechanism(nil), caps.Mechanisms...)
	q.Facts = caps.Facts

	q.Checks = append(q.Checks, checkProbeAvailable(caps))
	q.Checks = append(q.Checks, checkMechanism(caps))
	if !caps.Available {
		for _, name := range []string{
			CheckRefuseProc, CheckRefuseMem, CheckRefuseTmp,
			CheckOvershootFinite, CheckDetachedCharged,
			CheckBypassCaught, CheckOverflowClosed, CheckDisposalReserve,
			CheckHostFit, CheckOpenRealEnvelope,
			CheckKernelNproc, CheckKernelFsize, CheckMemBackstopProbe,
		} {
			q.Checks = append(q.Checks, CheckResult{
				Name: name, Detail: "not run: host unavailable",
			})
		}
		// The one positive demand on an unavailable host: OpenScope
		// must refuse with ErrUnavailable, never a scope. It folds
		// into open-real-envelope so the record keeps its shape.
		if _, err := a.OpenScope(env, work); errors.Is(err, ErrUnavailable) {
			q.Checks = append(q.Checks, CheckResult{
				Name: "open-refused", Pass: true,
				Detail: "OpenScope refused: " + err.Error(),
			})
		} else {
			q.Checks = append(q.Checks, CheckResult{
				Name:   "open-refused",
				Detail: fmt.Sprintf("OpenScope returned %v, want ErrUnavailable", err),
			})
		}
		return q, nil
	}

	fresh := func(name string, env Envelope) (Scope, string, error) {
		sub, merr := os.MkdirTemp(work, name+"-*")
		if merr != nil {
			return nil, "", merr
		}
		s, oerr := a.OpenScope(env, sub)
		if oerr != nil {
			return nil, "", oerr
		}
		return s, sub, nil
	}
	q.Checks = append(q.Checks, checkRefuseProc(fresh))
	q.Checks = append(q.Checks, checkRefuseMem(fresh))
	q.Checks = append(q.Checks, checkRefuseTmp(fresh))
	q.Checks = append(q.Checks, checkOvershootFinite(fresh))
	q.Checks = append(q.Checks, checkDetachedCharged(fresh))
	q.Checks = append(q.Checks, checkBypassCaught(fresh))
	q.Checks = append(q.Checks, checkOverflowClosed(fresh))
	q.Checks = append(q.Checks, checkDisposalReserve(fresh))
	q.Checks = append(q.Checks, checkHostFit(caps.Facts, env))
	q.Checks = append(q.Checks, q.checkOpenReal(a, work))
	q.Checks = append(q.Checks, checkKernelDemo(caps, CheckKernelNproc, DemoNprocRefusal))
	q.Checks = append(q.Checks, checkKernelDemo(caps, CheckKernelFsize, func() DemoResult {
		return DemoFsizeRefusal(work)
	}))
	q.Checks = append(q.Checks, checkMemBackstopProbe())
	return q, nil
}

func checkProbeAvailable(caps Capabilities) CheckResult {
	if caps.Available {
		return CheckResult{Name: CheckProbeAvailable, Pass: true,
			Detail: fmt.Sprintf("available: %s", stringsJoinMechanisms(caps.Mechanisms))}
	}
	return CheckResult{Name: CheckProbeAvailable, Detail: "unavailable: " + caps.Detail}
}

func stringsJoinMechanisms(ms []Mechanism) string {
	out := ""
	for i, m := range ms {
		if i > 0 {
			out += "+"
		}
		out += string(m)
	}
	if out == "" {
		return "(none)"
	}
	return out
}

// checkMechanism requires refuse-before-effect and forbids poll/sample
// claims. The label check is secondary: behavioral checks fail lying
// adapters whatever they declare.
func checkMechanism(caps Capabilities) CheckResult {
	has := func(m Mechanism) bool {
		for _, c := range caps.Mechanisms {
			if c == m {
				return true
			}
		}
		return false
	}
	switch {
	case has(MechPollKill) || has(MechSampledPeak):
		return CheckResult{Name: CheckMechanism,
			Detail: "non-qualifying mechanism claimed: " + stringsJoinMechanisms(caps.Mechanisms)}
	case !has(MechRefuseBeforeEffect):
		return CheckResult{Name: CheckMechanism,
			Detail: "missing refuse-before-effect: " + stringsJoinMechanisms(caps.Mechanisms)}
	default:
		return CheckResult{Name: CheckMechanism, Pass: true,
			Detail: stringsJoinMechanisms(caps.Mechanisms)}
	}
}

type scopeFactory func(name string, env Envelope) (Scope, string, error)

func failCheck(name, format string, args ...any) CheckResult {
	return CheckResult{Name: name, Detail: fmt.Sprintf(format, args...)}
}

// checkRefuseProc fills the 1-proc control ceiling, demands synchronous
// ErrOverEnvelope for the second, and proves the refusal had no effect.
func checkRefuseProc(fresh scopeFactory) CheckResult {
	const name = CheckRefuseProc
	s, _, err := fresh("proc", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	t, err := s.ChargeProc(false, 0)
	if err != nil {
		return failCheck(name, "first proc charge refused: %v", err)
	}
	if _, err := s.ChargeProc(false, 0); !errors.Is(err, ErrOverEnvelope) {
		return failCheck(name, "second proc charge returned %v, want ErrOverEnvelope", err)
	}
	if u := s.Usage(); u.Procs != 1 {
		return failCheck(name, "refusal had effect: %d procs charged, want 1", u.Procs)
	}
	if err := s.ReleaseProc(t); err != nil {
		return failCheck(name, "release: %v", err)
	}
	if u := s.Usage(); u.Procs != 0 || u.MemBytes != 0 {
		return failCheck(name, "release left %+v, want zero", u)
	}
	return CheckResult{Name: name, Pass: true, Detail: "1 admitted, 2nd refused ErrOverEnvelope, usage stayed 1"}
}

// checkRefuseMem fills the 96-unit mem ceiling via declaration and
// demands synchronous refusal past it. The mem envelope leaves proc
// headroom, so the over-mem charge is refused by the memory ledger and
// nothing else: a broken mem ledger cannot hide behind the proc ceiling.
func checkRefuseMem(fresh scopeFactory) CheckResult {
	const name = CheckRefuseMem
	s, _, err := fresh("mem", memControlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	t, err := s.ChargeProc(false, 96)
	if err != nil {
		return failCheck(name, "declare 96 refused: %v", err)
	}
	if _, err := s.ChargeProc(false, 1); !errors.Is(err, ErrOverEnvelope) {
		return failCheck(name, "declare 96+1 returned %v, want ErrOverEnvelope", err)
	}
	if u := s.Usage(); u.MemBytes != 96 {
		return failCheck(name, "refusal had effect: %d mem charged, want 96", u.MemBytes)
	}
	if err := s.ReleaseProc(t); err != nil {
		return failCheck(name, "release: %v", err)
	}
	return CheckResult{Name: name, Pass: true, Detail: "96 admitted, +1 refused ErrOverEnvelope, usage stayed 96"}
}

// checkRefuseTmp fills the 96-unit control tmp ceiling and demands
// synchronous refusal past it.
func checkRefuseTmp(fresh scopeFactory) CheckResult {
	const name = CheckRefuseTmp
	s, _, err := fresh("tmp", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	if err := s.ChargeTmp(96); err != nil {
		return failCheck(name, "charge 96 refused: %v", err)
	}
	if err := s.ChargeTmp(1); !errors.Is(err, ErrOverEnvelope) {
		return failCheck(name, "charge 96+1 returned %v, want ErrOverEnvelope", err)
	}
	if u := s.Usage(); u.TmpBytes != 96 {
		return failCheck(name, "refusal had effect: %d tmp charged, want 96", u.TmpBytes)
	}
	if err := s.ReleaseTmp(96); err != nil {
		return failCheck(name, "release: %v", err)
	}
	return CheckResult{Name: name, Pass: true, Detail: "96 admitted, +1 refused ErrOverEnvelope, usage stayed 96"}
}

// checkOvershootFinite demands exactly zero mediated overshoot and one
// finite positive bypass bound.
func checkOvershootFinite(fresh scopeFactory) CheckResult {
	const name = CheckOvershootFinite
	s, _, err := fresh("over", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	o := s.Overshoot()
	if !o.Finite() {
		return failCheck(name, "unbounded or negative overshoot: %+v", o)
	}
	if o.MaxProcsOver != 0 || o.MaxMemOver != 0 || o.MaxTmpMediatedOver != 0 {
		return failCheck(name, "nonzero mediated overshoot: %+v", o)
	}
	return CheckResult{Name: name, Pass: true,
		Detail: fmt.Sprintf("mediated 0/0/0, bypass bound %d bytes", o.TmpBypassBound)}
}

// checkDetachedCharged proves detached spawns fill the same ceiling:
// one detached token charges 1 proc, a second is refused.
func checkDetachedCharged(fresh scopeFactory) CheckResult {
	const name = CheckDetachedCharged
	s, _, err := fresh("detach", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	t, err := s.ChargeProc(true, 0)
	if err != nil {
		return failCheck(name, "detached charge refused: %v", err)
	}
	if !t.Detached() {
		return failCheck(name, "token lost the detached flag")
	}
	if u := s.Usage(); u.Procs != 1 || u.DetachedProcs != 1 {
		return failCheck(name, "detached charge left %+v, want 1/1", u)
	}
	if _, err := s.ChargeProc(true, 0); !errors.Is(err, ErrOverEnvelope) {
		return failCheck(name, "second detached charge returned %v, want ErrOverEnvelope", err)
	}
	if err := s.ReleaseProc(t); err != nil {
		return failCheck(name, "release: %v", err)
	}
	return CheckResult{Name: name, Pass: true, Detail: "detached 1 admitted (1/1), 2nd refused ErrOverEnvelope"}
}

// checkBypassCaught writes 40 bytes past the helpers and demands the
// reconcile absorb them so the next charge counts them.
func checkBypassCaught(fresh scopeFactory) CheckResult {
	const name = CheckBypassCaught
	s, _, err := fresh("bypass", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	if err := os.WriteFile(filepath.Join(s.TmpRoot(), "bypass.bin"), make([]byte, 40), 0o600); err != nil {
		return failCheck(name, "plant bypass bytes: %v", err)
	}
	facts, err := s.ReconcileTmp()
	if err != nil {
		return failCheck(name, "reconcile: %v", err)
	}
	if facts.Observed != 40 || facts.BypassCharged != 40 || s.Usage().TmpBytes != 40 {
		return failCheck(name, "bypass escaped: facts %+v usage %+v", facts, s.Usage())
	}
	// 40 absorbed + 57 would pass the 96 ceiling: must refuse.
	if err := s.ChargeTmp(57); !errors.Is(err, ErrOverEnvelope) {
		return failCheck(name, "charge past absorbed bypass returned %v, want ErrOverEnvelope", err)
	}
	if err := os.Remove(filepath.Join(s.TmpRoot(), "bypass.bin")); err != nil {
		return failCheck(name, "remove bypass file: %v", err)
	}
	if err := s.ReleaseTmp(40); err != nil {
		return failCheck(name, "release absorbed bypass: %v", err)
	}
	return CheckResult{Name: name, Pass: true, Detail: "40 bypass bytes absorbed; 40+57 refused ErrOverEnvelope"}
}

// checkOverflowClosed plants 200 bypass bytes past the 128 max and
// demands the scope latch overflow (all charges ErrOverflow) until
// Dispose clears it.
func checkOverflowClosed(fresh scopeFactory) CheckResult {
	const name = CheckOverflowClosed
	s, _, err := fresh("overflow", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	defer s.Dispose()
	planted := filepath.Join(s.TmpRoot(), "flood.bin")
	if err := os.WriteFile(planted, make([]byte, 200), 0o600); err != nil {
		return failCheck(name, "plant flood bytes: %v", err)
	}
	if _, err := s.ReconcileTmp(); !errors.Is(err, ErrOverflow) {
		return failCheck(name, "reconcile past max returned %v, want ErrOverflow", err)
	}
	if _, err := s.ChargeProc(false, 0); !errors.Is(err, ErrOverflow) {
		return failCheck(name, "proc charge in overflow returned %v, want ErrOverflow", err)
	}
	if err := s.ChargeTmp(1); !errors.Is(err, ErrOverflow) {
		return failCheck(name, "tmp charge in overflow returned %v, want ErrOverflow", err)
	}
	if err := os.Remove(planted); err != nil {
		return failCheck(name, "remove flood file: %v", err)
	}
	if err := s.Dispose(); err != nil {
		return failCheck(name, "dispose from overflow: %v", err)
	}
	if _, serr := os.Stat(s.TmpRoot()); !os.IsNotExist(serr) {
		return failCheck(name, "tmp root survives disposal")
	}
	return CheckResult{Name: name, Pass: true, Detail: "overflow latched ErrOverflow on all charges; Dispose cleared it"}
}

// checkDisposalReserve fills every ceiling, writes the charged tmp
// bytes for real, and demands Dispose succeed with the root gone plus
// ceiling-plus-reserve arithmetic exact on every dimension.
func checkDisposalReserve(fresh scopeFactory) CheckResult {
	const name = CheckDisposalReserve
	s, _, err := fresh("reserve", controlEnvelope())
	if err != nil {
		return failCheck(name, "open control scope: %v", err)
	}
	env := s.Envelope()
	if env.CeilingProcs()+env.DisposalReserve.Procs != env.MaxProcs ||
		env.CeilingMem()+env.DisposalReserve.MemBytes != env.MaxMemBytes ||
		env.CeilingTmp()+env.DisposalReserve.TmpBytes != env.MaxTmpBytes {
		return failCheck(name, "ceiling+reserve != max for %+v", env)
	}
	if _, err := s.ChargeProc(false, env.CeilingMem()); err != nil {
		return failCheck(name, "fill proc/mem ceiling: %v", err)
	}
	if err := s.ChargeTmp(env.CeilingTmp()); err != nil {
		return failCheck(name, "fill tmp ceiling: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.TmpRoot(), "full.bin"),
		make([]byte, env.CeilingTmp()), 0o600); err != nil {
		return failCheck(name, "write charged bytes: %v", err)
	}
	if err := s.Dispose(); err != nil {
		return failCheck(name, "dispose at full charge: %v", err)
	}
	if _, serr := os.Stat(s.TmpRoot()); !os.IsNotExist(serr) {
		return failCheck(name, "tmp root survives disposal")
	}
	return CheckResult{Name: name, Pass: true,
		Detail: "full-ceiling scope disposed; ceiling+reserve==max on all dimensions"}
}

// fitEnvelope proves the real envelope fits probed host facts. Unknown
// (zero) facts fail closed: only demonstrated fit admits.
func fitEnvelope(facts HostFacts, env Envelope) error {
	if facts.NProcSoft == 0 {
		return fmt.Errorf("%w: NPROC limit unknown", ErrNoFit)
	}
	if uint64(env.MaxProcs) > facts.NProcSoft {
		return fmt.Errorf("%w: %d procs over NPROC %d", ErrNoFit, env.MaxProcs, facts.NProcSoft)
	}
	if facts.MemBytes == 0 {
		return fmt.Errorf("%w: host memory unknown", ErrNoFit)
	}
	if uint64(env.MaxMemBytes) > facts.MemBytes {
		return fmt.Errorf("%w: %d mem bytes over host %d", ErrNoFit, env.MaxMemBytes, facts.MemBytes)
	}
	if facts.TmpFreeBytes == 0 {
		return fmt.Errorf("%w: tmp free disk unknown", ErrNoFit)
	}
	if uint64(env.MaxTmpBytes) > facts.TmpFreeBytes {
		return fmt.Errorf("%w: %d tmp bytes over free %d", ErrNoFit, env.MaxTmpBytes, facts.TmpFreeBytes)
	}
	return nil
}

func checkHostFit(facts HostFacts, env Envelope) CheckResult {
	if err := fitEnvelope(facts, env); err != nil {
		return CheckResult{Name: CheckHostFit, Detail: err.Error()}
	}
	return CheckResult{Name: CheckHostFit, Pass: true, Detail: fmt.Sprintf(
		"%s: %d procs<=NPROC %d; %d mem<=host %d; %d tmp<=free %d",
		env.Name, env.MaxProcs, facts.NProcSoft,
		env.MaxMemBytes, facts.MemBytes, env.MaxTmpBytes, facts.TmpFreeBytes)}
}

// checkOpenReal opens and disposes a real-envelope scope (empty ledger
// plus mkdir: no envelope-scale effects) and records its overshoot.
func (q *Qualification) checkOpenReal(a Adapter, work string) CheckResult {
	const name = CheckOpenRealEnvelope
	sub, err := os.MkdirTemp(work, "real-*")
	if err != nil {
		return failCheck(name, "workdir: %v", err)
	}
	s, err := a.OpenScope(q.Envelope, sub)
	if err != nil {
		return failCheck(name, "open %s scope: %v", q.Envelope.Name, err)
	}
	q.Overshoot = s.Overshoot()
	if err := s.Dispose(); err != nil {
		return failCheck(name, "dispose %s scope: %v", q.Envelope.Name, err)
	}
	return CheckResult{Name: name, Pass: true,
		Detail: fmt.Sprintf("%s scope opened and disposed; overshoot %+v", q.Envelope.Name, q.Overshoot)}
}

// checkKernelDemo runs one live kernel-refusal demonstration when the
// adapter claims a resource-limit backstop. Unclaimed means unproven:
// the check fails. A host too quiet to demonstrate skips, never fails.
func checkKernelDemo(caps Capabilities, name string, demo func() DemoResult) CheckResult {
	claimed := false
	for _, m := range caps.Mechanisms {
		if m == MechResourceLimit {
			claimed = true
		}
	}
	if !claimed {
		return CheckResult{Name: name, Detail: "no resource-limit backstop claimed"}
	}
	d := demo()
	if d.Skip {
		return CheckResult{Name: name, Pass: true, Detail: "skipped: " + d.Detail}
	}
	return CheckResult{Name: name, Pass: d.Pass, Detail: d.Detail}
}

// checkMemBackstopProbe records whether an AS cap exists on this host.
// Informational: both outcomes pass; the record states what the memory
// envelope may rest on.
func checkMemBackstopProbe() CheckResult {
	settable, detail := ProbeAddrSpaceSettable()
	if settable {
		return CheckResult{Name: CheckMemBackstopProbe, Pass: true,
			Detail: "memory backstop present: " + detail}
	}
	return CheckResult{Name: CheckMemBackstopProbe, Pass: true,
		Detail: "no memory backstop (" + detail + "); envelope rests on declared-peak ledger plus host-commit fit"}
}

// Admit binds a valid qualification to the P12 admission gate: fresh
// host facts are re-probed (fail closed), the lane is taken, then the
// scope opens. Any failure before the lane holds nothing; a scope-open
// failure releases the lane. tmpParent holds the admitted scope's tmp
// root; root and req follow admission.Host.Admit.
func Admit(h *admission.Host, root string, req admission.Request, q *Qualification, tmpParent string) (*Binding, error) {
	if h == nil || q == nil || q.adapter == nil || !q.Valid() {
		return nil, fmt.Errorf("%w: admission needs a valid qualification", ErrNotQualified)
	}
	if err := checkTmpParent(tmpParent); err != nil {
		return nil, err
	}
	caps := q.adapter.Probe(tmpParent)
	if !caps.Available {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, caps.Detail)
	}
	if err := fitEnvelope(caps.Facts, q.Envelope); err != nil {
		return nil, err
	}
	g, err := h.Admit(root, req)
	if err != nil {
		return nil, err
	}
	s, err := q.adapter.OpenScope(q.Envelope, tmpParent)
	if err != nil {
		g.Release()
		return nil, err
	}
	return &Binding{grant: g, scope: s}, nil
}

// Binding is one admitted envelope: a P12 lane grant plus an open
// charged scope. Every Binding must end in exactly one verdict —
// Complete — or in Release on abort paths.
type Binding struct {
	grant *admission.Grant
	scope Scope
}

// Grant reports the admission lane grant.
func (b *Binding) Grant() *admission.Grant { return b.grant }

// Scope reports the open charged scope.
func (b *Binding) Scope() Scope { return b.scope }

// Complete disposes the scope, then completes the lane grant: the
// cleanup-equivalent exists before the success verdict. A disposal
// failure releases the lane instead and no success is recorded.
// Second calls report the grant's released verdict.
func (b *Binding) Complete() error {
	if err := b.scope.Dispose(); err != nil {
		b.grant.Release()
		return err
	}
	return b.grant.Complete()
}

// Release abandons the binding: best-effort scope disposal, then the
// lane is released. It is idempotent.
func (b *Binding) Release() {
	_ = b.scope.Dispose()
	b.grant.Release()
}

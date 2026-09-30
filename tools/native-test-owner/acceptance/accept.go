package acceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/admission"
	"github.com/veighnsche/can-lang/tools/native-test-owner/host"
	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
	"github.com/veighnsche/can-lang/tools/native-test-owner/receipt/cleanup"
	"github.com/veighnsche/can-lang/tools/native-test-owner/recovery"
)

// ManifestSchemaVersion versions the N acceptance manifest.
const ManifestSchemaVersion = 1

// Behavior check names. Every Accept records every name so manifests
// compare across runs and hosts.
const (
	CheckWitness     = "witness"
	CheckHostScope   = "host-scope"
	CheckProtocol    = "protocol"
	CheckEnforcement = "enforcement"
	CheckJournal     = "journal"
	CheckRecovery    = "recovery"
	CheckReceipt     = "receipt"
)

var checkOrder = []string{
	CheckWitness, CheckHostScope, CheckProtocol,
	CheckEnforcement, CheckJournal, CheckRecovery, CheckReceipt,
}

// Version-fact keys carried in N's offered environment snapshot. N must
// observe exactly these bytes for its content-bound acknowledgment, so
// the ack binds the version facts to the run.
const (
	EnvOp      = "N_OP"
	EnvVersion = "N_VERSION"
	EnvDigest  = "N_DIGEST"
	EnvMode    = "N_MODE"
	// ModeDie asks the test subject helper to die mid-run without
	// acknowledging. It is a negative control, never a pass path.
	ModeDie = "die"
)

// DefaultDeclaredMem is the declared memory peak charged for N's spawn
// when the caller declares none: a trivially fitting 1 MiB.
const DefaultDeclaredMem = 1 << 20

// Default observation bounds.
const (
	DefaultAckTimeout  = 10 * time.Second
	DefaultEofTimeout  = 10 * time.Second
	DefaultWaitTimeout = 30 * time.Second
)

// ErrConfig reports caller misuse: a nil witness, a missing host
// selection, or a malformed field. Misuse returns no manifest.
var ErrConfig = errors.New("acceptance: invalid config")

// NSubject is N under test: an attested executable plus its argv and
// extra environment entries. Accept merges the version facts on top.
type NSubject struct {
	Identity NIdentity
	Args     []string
	Env      map[string]string
}

// Config is one acceptance run.
type Config struct {
	// OpID identifies the run; the cleanup operation derives from it.
	OpID string
	// N is the subject under test.
	N NSubject
	// T is the independent witness; it must be alive.
	T *Witness
	// Host is the explicitly selected host profile.
	Host *SelectedHost
	// Root is the admitted repository root (absolute).
	Root string
	// Request is the admission demand with its charged costs.
	Request admission.Request
	// TmpParent holds the admitted scope's tmp root.
	TmpParent string
	// ReceiptPath is the absolute acceptance receipt path.
	ReceiptPath string
	// DeclaredMem is N's declared memory peak; <=0 selects the default.
	DeclaredMem int64
	// AckTimeout, EofTimeout and WaitTimeout bound N observation;
	// non-positive values select the defaults.
	AckTimeout  time.Duration
	EofTimeout  time.Duration
	WaitTimeout time.Duration
	// OmitVersionFacts strips the version facts from N's environment.
	// It is a negative control: N must refuse to run versionless.
	OmitVersionFacts bool
	// CleanupTarget is the receipt-cleanup deletion target; "" deletes
	// nothing. Remove deletes it and must be non-nil then.
	CleanupTarget string
	Remove        func(path string) error
}

// ChildEnv renders the exact environment offered to N: the subject
// extras plus the run operation and (unless stripped for the negative
// control) the bound version facts.
func (c Config) ChildEnv() map[string]string {
	env := make(map[string]string, len(c.N.Env)+4)
	for k, v := range c.N.Env {
		env[k] = v
	}
	env[EnvOp] = c.OpID
	if !c.OmitVersionFacts {
		env[EnvVersion] = c.N.Identity.Version
		env[EnvDigest] = c.N.Identity.Digest
	}
	return env
}

// Check is one behavior binding outcome with its evidence detail.
type Check struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// ReleaseFacts are T's independently witnessed release facts for N:
// separate observations that all hold at once only on a clean release.
type ReleaseFacts struct {
	Offered      bool   `json:"offered"`
	WriterDone   bool   `json:"writerDone"`
	WriterErr    string `json:"writerErr,omitempty"`
	Accepted     bool   `json:"accepted"`
	Malformed    bool   `json:"malformed"`
	EOF          bool   `json:"eof"`
	Reaped       bool   `json:"reaped"`
	ExitCode     int    `json:"exitCode"`
	Signaled     bool   `json:"signaled"`
	Signal       string `json:"signal,omitempty"`
	LeaseHeld    bool   `json:"leaseHeld"`
	LeaseRelease bool   `json:"leaseReleased"`
	Orphan       bool   `json:"orphan"`
	Clean        bool   `json:"clean"`
	Reason       string `json:"reason,omitempty"`
}

// ReceiptFacts are the written acceptance receipt facts.
type ReceiptFacts struct {
	OperationID string `json:"operationId"`
	Outcome     string `json:"outcome"`
	Reason      string `json:"reason"`
	Removed     bool   `json:"removed"`
	Complete    bool   `json:"complete"`
}

// Manifest is the N acceptance record: N and T identities, the exact
// host profile scope, the behavior bindings, the independent release
// facts and the receipt facts. Complete is true only when every check
// passed; Reason names the first missing binding otherwise.
type Manifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	Complete      bool          `json:"complete"`
	Reason        string        `json:"reason,omitempty"`
	N             NIdentity     `json:"n"`
	T             journal.Owner `json:"t"`
	Host          HostRecord    `json:"host"`
	Checks        []Check       `json:"checks"`
	Release       ReleaseFacts  `json:"release"`
	Receipt       ReceiptFacts  `json:"receipt"`
}

// HostRecord scopes acceptance to the exact qualified host profile.
type HostRecord struct {
	Adapter  string         `json:"adapter"`
	Envelope string         `json:"envelope"`
	Facts    host.HostFacts `json:"facts"`
}

// JSON renders the manifest for the test-emitted artifact.
func (m *Manifest) JSON() string {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion":%d,"error":"encode failed"}`, ManifestSchemaVersion)
	}
	return string(raw)
}

// Verify re-checks a written acceptance receipt for promotion: the file
// must load, name the expected operation, and record a complete pass.
// A forged, absent or incomplete receipt fails, never reads as success.
func Verify(receiptPath, wantOpID string) error {
	rec, err := cleanup.LoadFile(receiptPath)
	if err != nil {
		return fmt.Errorf("acceptance: verify receipt: %w", err)
	}
	if rec.OperationID != wantOpID {
		return fmt.Errorf("acceptance: receipt for %q, want operation %q", rec.OperationID, wantOpID)
	}
	if !rec.Complete || rec.Outcome != cleanup.OutcomePass {
		return fmt.Errorf("acceptance: receipt is %s/%s, want a complete pass",
			rec.Outcome, reasonOf(rec))
	}
	return nil
}

func reasonOf(rec *cleanup.Receipt) string {
	if rec.Reason == "" {
		return "no-reason"
	}
	return rec.Reason
}

func validOpID(s string) bool {
	if len(s) == 0 || len(s) > 120 {
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

// Accept runs N under T's independent witness and binds every behavior
// into the manifest. Misuse (a nil witness or host, a malformed field)
// returns an error with no manifest; every other outcome is a manifest
// decision, and only a fully bound run completes.
func Accept(cfg Config) (*Manifest, error) {
	if cfg.T == nil {
		return nil, fmt.Errorf("%w: nil witness", ErrConfig)
	}
	if cfg.Host == nil {
		return nil, fmt.Errorf("%w: no explicitly selected host", ErrConfig)
	}
	if !validOpID(cfg.OpID) {
		return nil, fmt.Errorf("%w: malformed operation ID", ErrConfig)
	}
	if cfg.N.Identity.Executable == "" {
		return nil, fmt.Errorf("%w: unattested subject", ErrConfig)
	}
	if cfg.Root == "" || !filepath.IsAbs(cfg.Root) {
		return nil, fmt.Errorf("%w: repository root must be absolute", ErrConfig)
	}
	if cfg.ReceiptPath == "" || !filepath.IsAbs(cfg.ReceiptPath) {
		return nil, fmt.Errorf("%w: receipt path must be absolute", ErrConfig)
	}
	if cfg.CleanupTarget != "" && !filepath.IsAbs(cfg.CleanupTarget) {
		return nil, fmt.Errorf("%w: cleanup target must be absolute", ErrConfig)
	}
	if cfg.CleanupTarget != "" && cfg.Remove == nil {
		return nil, fmt.Errorf("%w: cleanup target without a remover", ErrConfig)
	}
	r := &run{
		cfg:       cfg,
		w:         cfg.T,
		cleanupOp: cfg.OpID + "-cleanup",
		m: &Manifest{
			SchemaVersion: ManifestSchemaVersion,
			N:             cfg.N.Identity,
			T:             cfg.T.ID(),
			Host: HostRecord{
				Adapter:  cfg.Host.AdapterName,
				Envelope: cfg.Host.Envelope.Name,
				Facts:    cfg.Host.Facts,
			},
		},
	}
	return r.accept()
}

type run struct {
	cfg       Config
	w         *Witness
	cleanupOp string
	m         *Manifest
	marks     map[string]bool
}

func (r *run) mark(name string, pass bool, detail string) {
	if r.marks == nil {
		r.marks = make(map[string]bool)
	}
	if r.marks[name] {
		return
	}
	r.marks[name] = true
	r.m.Checks = append(r.m.Checks, Check{Name: name, Pass: pass, Detail: detail})
}

func (r *run) ordered() {
	byName := make(map[string]Check, len(r.m.Checks))
	for _, c := range r.m.Checks {
		byName[c.Name] = c
	}
	r.m.Checks = r.m.Checks[:0]
	for _, name := range checkOrder {
		c, ok := byName[name]
		if !ok {
			c = Check{Name: name, Detail: "not reached"}
		}
		r.m.Checks = append(r.m.Checks, c)
	}
}

// finalize closes the manifest. Complete needs the flow verdict plus
// every check passing: a success claim with any failing check fails
// closed.
func (r *run) finalize(flowOK bool, reason string) (*Manifest, error) {
	r.ordered()
	complete := flowOK
	for _, c := range r.m.Checks {
		complete = complete && c.Pass
	}
	r.m.Complete = complete
	r.m.Reason = reason
	if !complete && reason == "" {
		r.m.Reason = "check failed"
	}
	if complete {
		r.m.Reason = ""
	}
	return r.m, nil
}

// refuse records an incomplete verdict with a refusal receipt from the
// real cleanup authority: the scan decision is negative by construction
// (the body did not earn deletion authority) so the receipt can never
// read as a complete success.
func (r *run) refuse(reason string) (*Manifest, error) {
	rec, _ := cleanup.Perform(r.w.journal, cleanup.Request{
		OperationID: r.cleanupOp,
		Path:        r.cfg.CleanupTarget,
		Self:        r.w.id,
		Allow:       false,
		DenyReason:  reason,
		Prior:       cleanup.OutcomeFailed,
	}, nil)
	if rec != nil {
		_ = cleanup.WriteFile(r.cfg.ReceiptPath, rec)
		r.m.Receipt = ReceiptFacts{
			OperationID: rec.OperationID,
			Outcome:     string(rec.Outcome),
			Reason:      rec.Reason,
			Removed:     rec.Removed,
			Complete:    rec.Complete,
		}
	}
	r.mark(CheckReceipt, false, "refused: "+reason)
	return r.finalize(false, reason)
}

func (r *run) accept() (*Manifest, error) {
	cfg := r.cfg
	if !r.w.Alive() {
		// T's own loss: nobody trustworthy remains to write a
		// receipt, so none is written at all.
		r.mark(CheckWitness, false, "witness lost")
		return r.finalize(false, "witness-lost")
	}
	r.mark(CheckWitness, true, fmt.Sprintf("T pid %d holds its journal, lease and lanes", r.w.id.PID))

	sum := sha256.Sum256([]byte("n-acceptance-cleanup:" + cfg.OpID))
	if _, err := r.w.journal.Reserve(r.cleanupOp, "sha256:"+hex.EncodeToString(sum[:]),
		r.w.id, journal.PathIdentity{Path: cfg.CleanupTarget}, "", 0); err != nil {
		return r.finalize(false, fmt.Sprintf("journal reserve: %v", err))
	}

	if cfg.Host.Foreign {
		r.mark(CheckHostScope, false, fmt.Sprintf(
			"explicit selection of %q recorded; execution stays local", cfg.Host.AdapterName))
		return r.refuse("foreign-host: " + cfg.Host.AdapterName + " needs its own receipt")
	}
	if err := cfg.N.Identity.Verify(); err != nil {
		r.mark(CheckProtocol, false, err.Error())
		return r.refuse("n-attest: " + err.Error())
	}

	binding, err := host.Admit(r.w.admission, cfg.Root, cfg.Request, cfg.Host.qual, cfg.TmpParent)
	if err != nil {
		r.mark(CheckHostScope, false, fmt.Sprintf("admit refused (%s): %v", host.Reason(err), err))
		return r.refuse("admit: " + err.Error())
	}
	scope := binding.Scope()

	mem := cfg.DeclaredMem
	if mem <= 0 {
		mem = DefaultDeclaredMem
	}
	token, err := scope.ChargeProc(false, mem)
	if err != nil {
		binding.Release()
		r.mark(CheckEnforcement, false, fmt.Sprintf("charge refused: %v", err))
		return r.refuse("enforcement: " + err.Error())
	}

	id, err := r.w.owner.Spawn(cfg.OpID, process.Spec{
		Executable: cfg.N.Identity.Executable,
		Args:       cfg.N.Args,
		Env:        cfg.ChildEnv(),
		WithLease:  true,
	}, r.w.lease)
	if err != nil {
		_ = scope.ReleaseProc(token)
		binding.Release()
		return r.refuse("spawn: " + err.Error())
	}
	snap, _ := r.w.owner.Snapshot(id)
	snapSum := sha256.Sum256(snap)

	ackTimeout, eofTimeout, waitTimeout := cfg.AckTimeout, cfg.EofTimeout, cfg.WaitTimeout
	if ackTimeout <= 0 {
		ackTimeout = DefaultAckTimeout
	}
	if eofTimeout <= 0 {
		eofTimeout = DefaultEofTimeout
	}
	if waitTimeout <= 0 {
		waitTimeout = DefaultWaitTimeout
	}
	collectErr := r.w.owner.CollectStatus(id, ackTimeout, eofTimeout)
	_, waitErr := r.w.owner.Wait(id, waitTimeout)
	if waitErr != nil {
		held, _ := process.ProbeLease(r.w.lease.Path())
		facts, _ := r.w.owner.Facts(id)
		_ = r.w.owner.Abort(id)
		r.fillRelease(process.Report{Facts: facts,
			Lease: process.LeaseReport{Path: r.w.lease.Path(), Inherited: true, Held: held}})
		r.mark(CheckProtocol, false, fmt.Sprintf("N still running past wait bound: %v (collect: %v)", waitErr, collectErr))
		r.markJournal()
		_ = scope.ReleaseProc(token)
		binding.Release()
		return r.refuse(fmt.Sprintf("release: child still running; lease held=%v", held))
	}
	rep, err := r.w.owner.Release(id)
	if err != nil {
		_ = r.w.owner.Abort(id)
		_ = scope.ReleaseProc(token)
		binding.Release()
		return r.refuse("release: " + err.Error())
	}
	r.fillRelease(rep)
	if collectErr != nil {
		r.mark(CheckProtocol, false, fmt.Sprintf("status not accepted: %v", collectErr))
	} else {
		r.mark(CheckProtocol, true, fmt.Sprintf(
			"codec status frame accepted plus clean EOF; version facts bound in snapshot sha256:%x",
			snapSum[:8]))
	}
	r.markJournal()
	if !rep.Clean {
		_ = r.w.owner.Abort(id)
		_ = scope.ReleaseProc(token)
		binding.Release()
		return r.refuse("release: " + rep.Reason)
	}

	drep, err := recovery.Drain(r.w.gate, r.w.owner, recovery.DrainConfig{Grace: 2 * time.Second})
	if err != nil {
		_ = scope.ReleaseProc(token)
		binding.Release()
		return r.refuse("recovery: " + err.Error())
	}
	srep := recovery.Scan(r.w.journal, r.w.id, recovery.ScanConfig{})
	allowed := srep.Allowed(r.cleanupOp)
	drained := drep.Complete && r.w.owner.Live() == 0
	if !drained || !allowed {
		_ = scope.ReleaseProc(token)
		binding.Release()
		r.mark(CheckRecovery, false, fmt.Sprintf(
			"drain complete=%v live=%d scan allowed=%v (%s)",
			drep.Complete, r.w.owner.Live(), allowed, srep.ReasonFor(r.cleanupOp)))
		return r.refuse("recovery: drain or scan refused")
	}
	r.mark(CheckRecovery, true, fmt.Sprintf(
		"drain complete, live 0, scan allows %s", r.cleanupOp))

	if err := scope.ReleaseProc(token); err != nil {
		binding.Release()
		r.mark(CheckEnforcement, false, fmt.Sprintf("token release: %v", err))
		return r.refuse("enforcement: " + err.Error())
	}
	if u := scope.Usage(); u != (host.Usage{}) {
		binding.Release()
		r.mark(CheckEnforcement, false, fmt.Sprintf("lingering charge: %+v", u))
		return r.refuse("enforcement: lingering charge")
	}
	if err := binding.Complete(); err != nil {
		r.mark(CheckEnforcement, false, fmt.Sprintf("binding: %v", err))
		r.mark(CheckHostScope, false, fmt.Sprintf("binding: %v", err))
		return r.refuse("enforcement: " + err.Error())
	}
	r.mark(CheckEnforcement, true, fmt.Sprintf(
		"%s scope charged 1 proc (%d mem declared), released, disposed; lane grant completed",
		cfg.Host.Envelope.Name, mem))
	r.mark(CheckHostScope, true, fmt.Sprintf(
		"admitted on %s/%s: GOOS %s NPROC %d mem %d",
		cfg.Host.AdapterName, cfg.Host.Envelope.Name,
		cfg.Host.Facts.GOOS, cfg.Host.Facts.NProcSoft, cfg.Host.Facts.MemBytes))

	rec, err := cleanup.Perform(r.w.journal, cleanup.Request{
		OperationID: r.cleanupOp,
		Path:        cfg.CleanupTarget,
		Self:        r.w.id,
		Allow:       true,
		Prior:       cleanup.OutcomePass,
	}, cfg.Remove)
	if err != nil {
		if rec != nil {
			_ = cleanup.WriteFile(cfg.ReceiptPath, rec)
			r.m.Receipt = ReceiptFacts{
				OperationID: rec.OperationID,
				Outcome:     string(rec.Outcome),
				Reason:      rec.Reason,
				Removed:     rec.Removed,
				Complete:    rec.Complete,
			}
		}
		r.mark(CheckReceipt, false, fmt.Sprintf("cleanup: %v", err))
		return r.finalize(false, "cleanup: "+err.Error())
	}
	if err := cleanup.WriteFile(cfg.ReceiptPath, rec); err != nil {
		r.mark(CheckReceipt, false, fmt.Sprintf("receipt write: %v", err))
		return r.finalize(false, "cleanup: receipt write failed")
	}
	loaded, err := cleanup.LoadFile(cfg.ReceiptPath)
	if err != nil || !loaded.Complete || loaded.Outcome != cleanup.OutcomePass {
		r.mark(CheckReceipt, false, fmt.Sprintf("receipt round-trip: %v", err))
		return r.finalize(false, "cleanup: receipt round-trip failed")
	}
	r.m.Receipt = ReceiptFacts{
		OperationID: loaded.OperationID,
		Outcome:     string(loaded.Outcome),
		Reason:      loaded.Reason,
		Removed:     loaded.Removed,
		Complete:    loaded.Complete,
	}
	r.mark(CheckReceipt, true, fmt.Sprintf(
		"%s records a complete pass (%s)", loaded.OperationID, loaded.Reason))
	return r.finalize(true, "")
}

func (r *run) fillRelease(rep process.Report) {
	f := rep.Facts
	rf := ReleaseFacts{
		Offered: f.Offered, WriterDone: f.WriterDone, WriterErr: f.WriterErr,
		Accepted: f.Accepted, Malformed: f.Malformed, EOF: f.EOF, Reaped: f.Reaped,
		LeaseHeld: rep.Lease.Held, LeaseRelease: rep.Lease.Released,
		Orphan: rep.Orphan, Clean: rep.Clean, Reason: rep.Reason,
	}
	if f.ChildExit != nil {
		rf.ExitCode = f.ChildExit.Code
		rf.Signaled = f.ChildExit.Signaled
		rf.Signal = f.ChildExit.Signal
	}
	r.m.Release = rf
}

func (r *run) markJournal() {
	res, ok := r.w.journal.Lookup(r.cfg.OpID)
	if !ok {
		r.mark(CheckJournal, false, "spawn intent missing from T journal")
		return
	}
	r.mark(CheckJournal, res.State == journal.StateClosed,
		fmt.Sprintf("spawn intent %s in T journal", res.State))
}

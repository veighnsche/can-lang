package process

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// Descriptor numbers. They mirror the driver process boundary: fd 3 is
// the private environment channel and fd 4 is the generation lease.
// Generated (child) code can observe but never renegotiate them.
const (
	FDEnv   = 3
	FDLease = 4
)

// Errors. Callers distinguish them with errors.Is.
var (
	ErrInvalid    = errors.New("process: invalid request")
	ErrNotFound   = errors.New("process: unknown process")
	ErrWrongOwner = errors.New("process: wrong spawn-start identity")
	ErrTimeout    = errors.New("process: timed out")
	ErrReleased   = errors.New("process: already released")
	ErrState      = errors.New("process: illegal state for operation")
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// MaxProcs caps live children of one Owner.
	MaxProcs = 64
	// MaxEnvBytes caps the offered environment snapshot.
	MaxEnvBytes = 1 << 20
	// MaxEnvVars caps the number of environment entries.
	MaxEnvVars = 1024
	// MaxEnvEntry caps one rendered "key=value" entry.
	MaxEnvEntry = 64 << 10
	// MaxArgs caps argv entries past the executable.
	MaxArgs = 256
	// MaxArgLen caps one argv entry.
	MaxArgLen = 64 << 10
	// MaxStatusPayload caps one decoded status frame.
	MaxStatusPayload = 64 << 10
	// DefaultStatusTimeout bounds one status-collection stage when the
	// caller passes a non-positive timeout.
	DefaultStatusTimeout = 10 * time.Second
	// DefaultWaitTimeout bounds Wait when the caller passes a
	// non-positive timeout.
	DefaultWaitTimeout = 30 * time.Second
	// releaseJoinTimeout bounds the writer/status joins inside Release.
	releaseJoinTimeout = 5 * time.Second
)

// Spec is the exact spawn contract: executable, arguments, environment
// allowlist snapshot and descriptor map. Every field is recorded before
// fork and observed by the child exactly as recorded.
type Spec struct {
	// Executable is the absolute child path. It is resolved (symlinks
	// evaluated) at spawn and the resolved path is recorded.
	Executable string
	// Args are the child's argv past argv[0].
	Args []string
	// Env is the allowlist snapshot delivered verbatim on fd 3. The
	// child's startup environment is otherwise empty.
	Env map[string]string
	// WithLease inherits the owner's lease descriptor as fd 4.
	// Without it the child holds no share of the generation lease.
	WithLease bool
	// Detached starts the child in a new session. It stays a direct,
	// reappable child; only its process group changes.
	Detached bool
	// Dir is the child's working directory; "" inherits the owner's.
	Dir string
}

// Identity binds a PID to the start token minted at spawn. The token
// disambiguates PID reuse: operations presenting a live PID with a
// stale or foreign token fail with ErrWrongOwner.
type Identity struct {
	PID        int
	StartToken string
}

// Exit is the reaped child-exit fact.
type Exit struct {
	Code     int
	Signaled bool
	Signal   string
}

// Facts are the separate per-process observations. No fact implies any
// other: a clean release needs all of them at once.
type Facts struct {
	Offered      bool   // full environment snapshot written to fd 3
	OfferedBytes int    // snapshot bytes the kernel accepted
	WriterDone   bool   // delivery goroutine finished
	WriterErr    string // "" when delivery succeeded; e.g. "EPIPE"
	Accepted     bool   // content-bound ack decoded on the status channel
	Malformed    bool   // status bytes failed frame validation
	EOF          bool   // clean EOF observed after the ack
	Reaped       bool   // child reaped; ChildExit is set
	ChildExit    *Exit  // wait status once reaped
}

// LeaseReport is the kernel-level lease witness outcome.
type LeaseReport struct {
	// Path is the lease file probed.
	Path string
	// Inherited reports whether the child was spawned with fd 4.
	Inherited bool
	// Held reports that an independent exclusive probe failed while
	// protection was required: the lock provably existed.
	Held bool
	// Released reports that an independent exclusive probe succeeded
	// after the owner closed its share: nothing still holds the lock.
	Released bool
}

// Report is the release decision with its evidence.
type Report struct {
	ID         Identity
	Executable string
	Facts      Facts
	Lease      LeaseReport
	// Orphan is set when the lease stays held after the direct child
	// was reaped and the owner closed its share: a detached
	// descendant still holds a copy of fd 4.
	Orphan bool
	// Clean is true only when every release clause holds; Reason
	// names the first missing clause otherwise.
	Clean  bool
	Reason string
}

// Edge is one process-tree descendant edge.
type Edge struct {
	ParentPID int
	Child     Identity
	Running   bool
}

type proc struct {
	id       Identity
	opID     string
	spec     Spec
	resolved string
	snapshot []byte // exact bytes offered on fd 3
	expected string // expected ack payload

	cmd    *exec.Cmd
	status io.ReadCloser // child's stdout: the status channel
	stderr *bytes.Buffer

	writerCh   <-chan offerResult
	writerDone chan struct{} // closed when the writer outcome is recorded
	statusCh   <-chan statusResult
	statusOn   bool // collector started

	lease *Lease // owner's share; nil when the child has no lease

	mu           sync.Mutex
	facts        Facts
	exited       bool
	waitCh       chan error // closed with the wait outcome once reaped
	waitErr      error
	signals      []string
	releaseClean bool
	leaseClosed  bool
	heldSeen     bool // an independent probe once failed: protection observed
}

// Owner spawns and owns processes with descriptor authority. It journals
// spawn intent before fork (P07) and retains every child until Release
// or Abort. An Owner is safe for concurrent use.
type Owner struct {
	mu    sync.Mutex
	j     *journal.Journal
	pid   int
	token string
	procs map[int]*proc
}

// New returns an Owner bound to a journal and an owner identity. The
// spawn-start token disambiguates PID reuse; a bare PID is never
// authority.
func New(j *journal.Journal, pid int, startToken string) (*Owner, error) {
	if j == nil {
		return nil, fmt.Errorf("%w: nil journal", ErrInvalid)
	}
	if pid <= 0 || startToken == "" || len(startToken) > journal.MaxStartTokenLen {
		return nil, fmt.Errorf("%w: owner needs positive PID and spawn-start token", ErrInvalid)
	}
	return &Owner{j: j, pid: pid, token: startToken, procs: make(map[int]*proc)}, nil
}

// Identity reports the owner's spawn-start identity.
func (o *Owner) Identity() journal.Owner {
	return journal.Owner{PID: o.pid, StartToken: o.token}
}

func mintToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("process: random start token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func validOpID(s string) bool {
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

// specDigest binds a spawn operation ID to its exact arguments.
func specDigest(spec Spec, resolved string, snapshot []byte) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%q|%q|%x|%t|%t|%q",
		resolved, spec.Args, sha256.Sum256(snapshot), spec.WithLease, spec.Detached, spec.Dir)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func checkSpec(spec Spec) (resolved string, snapshot []byte, err error) {
	if spec.Executable == "" || !filepath.IsAbs(spec.Executable) {
		return "", nil, fmt.Errorf("%w: executable must be an absolute path", ErrInvalid)
	}
	resolved, err = filepath.EvalSymlinks(spec.Executable)
	if err != nil {
		return "", nil, fmt.Errorf("%w: resolve executable: %v", ErrInvalid, err)
	}
	if len(spec.Args) > MaxArgs {
		return "", nil, fmt.Errorf("%w: too many argv entries", ErrInvalid)
	}
	for _, a := range spec.Args {
		if len(a) > MaxArgLen {
			return "", nil, fmt.Errorf("%w: argv entry too long", ErrInvalid)
		}
	}
	if len(spec.Env) > MaxEnvVars {
		return "", nil, fmt.Errorf("%w: too many environment entries", ErrInvalid)
	}
	for k, v := range spec.Env {
		if k == "" || len(k)+1+len(v) > MaxEnvEntry {
			return "", nil, fmt.Errorf("%w: malformed environment entry", ErrInvalid)
		}
		for i := 0; i < len(k); i++ {
			if k[i] == '=' || k[i] == 0 || k[i] == '\n' {
				return "", nil, fmt.Errorf("%w: malformed environment key", ErrInvalid)
			}
		}
	}
	snapshot, err = snapshotEnv(spec.Env)
	if err != nil {
		return "", nil, err
	}
	if len(snapshot) > MaxEnvBytes {
		return "", nil, fmt.Errorf("%w: environment snapshot exceeds bound", ErrInvalid)
	}
	if spec.Dir != "" {
		if !filepath.IsAbs(spec.Dir) {
			return "", nil, fmt.Errorf("%w: working directory must be absolute", ErrInvalid)
		}
		if fi, serr := os.Stat(spec.Dir); serr != nil || !fi.IsDir() {
			return "", nil, fmt.Errorf("%w: working directory unavailable", ErrInvalid)
		}
	}
	return resolved, snapshot, nil
}

// Spawn records journal intent, then starts one child with the exact
// executable/environment/fd map: stdin is /dev/null, stdout is the
// status channel, stderr is captured opaquely, fd 3 carries the
// environment snapshot and fd 4 (when lease is non-nil and
// spec.WithLease) carries the generation lease. lease may be nil only
// when spec.WithLease is false. Environment delivery to fd 3 starts
// immediately; its outcome lands in Facts.
func (o *Owner) Spawn(opID string, spec Spec, lease *Lease) (Identity, error) {
	if !validOpID(opID) {
		return Identity{}, fmt.Errorf("%w: malformed operation ID", ErrInvalid)
	}
	resolved, snapshot, err := checkSpec(spec)
	if err != nil {
		return Identity{}, err
	}
	if spec.WithLease && lease == nil {
		return Identity{}, fmt.Errorf("%w: WithLease needs a lease", ErrInvalid)
	}
	if !spec.WithLease && lease != nil {
		return Identity{}, fmt.Errorf("%w: lease without WithLease", ErrInvalid)
	}
	digest := specDigest(spec, resolved, snapshot)
	leasePath := ""
	if lease != nil {
		leasePath = lease.Path()
	}
	if _, err := o.j.Reserve(opID, digest, journal.Owner{PID: o.pid, StartToken: o.token},
		journal.PathIdentity{Path: leasePath}, "", 0); err != nil {
		return Identity{}, fmt.Errorf("process: journal reserve: %w", err)
	}
	fail := func(label string) (Identity, error) {
		_, _ = o.j.Advance(opID, journal.StateFailed, label)
		return Identity{}, fmt.Errorf("%w: %s", ErrInvalid, label)
	}

	o.mu.Lock()
	if len(o.procs) >= MaxProcs {
		o.mu.Unlock()
		return fail("owner process table full")
	}
	o.mu.Unlock()

	envRead, envWrite, err := os.Pipe()
	if err != nil {
		return fail("env pipe unavailable")
	}
	cleanupPipe := true
	defer func() {
		if cleanupPipe {
			envRead.Close()
			envWrite.Close()
		}
	}()

	null, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err != nil {
		return fail("null stdin unavailable")
	}
	defer null.Close()

	token, err := mintToken()
	if err != nil {
		return fail("start token unavailable")
	}
	cmd := exec.Command(resolved, spec.Args...)
	cmd.Dir = spec.Dir
	// The startup environment is empty by construction: application
	// state travels on fd 3 only, never on argv, disk or environ.
	cmd.Env = []string{}
	cmd.Stdin = null
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// The status channel is a manual pipe, never cmd.StdoutPipe: Go's
	// cmd.Wait closes StdoutPipe read ends once the child exits, which
	// would fail late collection on perfect child behavior with "file
	// already closed". The owner holds the read end until EOF is
	// observed or the child is aborted, however late that comes.
	statusRead, statusWrite, err := os.Pipe()
	if err != nil {
		return fail("status pipe unavailable")
	}
	cmd.Stdout = statusWrite
	cmd.ExtraFiles = []*os.File{envRead}
	if spec.WithLease {
		cmd.ExtraFiles = append(cmd.ExtraFiles, lease.File())
	}
	if spec.Detached {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		statusRead.Close()
		statusWrite.Close()
		return fail("child failed to start")
	}
	// The parent drops its copy of the status write end so the
	// child's close delivers EOF; the read end stays open until the
	// owner observes EOF or aborts.
	statusWrite.Close()
	status := statusRead
	// The owner keeps the write end for delivery; the read end belongs
	// to the child now.
	envRead.Close()
	cleanupPipe = false // envWrite ownership moves to the writer below

	p := &proc{
		id:       Identity{PID: cmd.Process.Pid, StartToken: token},
		opID:     opID,
		spec:     spec,
		resolved: resolved,
		snapshot: snapshot,
		expected: statusPayload(opID, envDigest(snapshot)),
		cmd:      cmd,
		status:   status,
		stderr:   &stderr,
		lease:    lease,
		waitCh:   make(chan error, 1),
	}
	p.writerDone = make(chan struct{})
	p.writerCh = offerEnv(envWrite, snapshot)
	go func() { p.waitCh <- cmd.Wait() }()

	o.mu.Lock()
	o.procs[p.id.PID] = p
	o.mu.Unlock()
	for _, st := range []journal.State{journal.StateOpening, journal.StateLive} {
		if _, aerr := o.j.Advance(opID, st); aerr != nil {
			_ = p.cmd.Process.Kill()
			<-p.waitCh
			o.remove(p.id.PID)
			envWrite.Close()
			_ = status.Close()
			return Identity{}, fmt.Errorf("process: journal advance: %w", aerr)
		}
	}
	return p.id, nil
}

// lookup returns the live child, verifying the spawn-start token: a
// stale or foreign token is ErrWrongOwner even when the PID is live
// (possibly reused by an unrelated process).
func (o *Owner) lookup(id Identity) (*proc, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p, ok := o.procs[id.PID]
	if !ok {
		return nil, fmt.Errorf("%w: pid %d", ErrNotFound, id.PID)
	}
	if p.id.StartToken != id.StartToken {
		return nil, fmt.Errorf("%w: pid %d presents a foreign start token", ErrWrongOwner, id.PID)
	}
	return p, nil
}

func (o *Owner) remove(pid int) {
	o.mu.Lock()
	delete(o.procs, pid)
	o.mu.Unlock()
}

// Snapshot returns the exact environment bytes offered to the child.
func (o *Owner) Snapshot(id Identity) ([]byte, error) {
	p, err := o.lookup(id)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), p.snapshot...), nil
}

// Executable returns the resolved executable recorded at spawn.
func (o *Owner) Executable(id Identity) (string, error) {
	p, err := o.lookup(id)
	if err != nil {
		return "", err
	}
	return p.resolved, nil
}

// FDMap returns the exact descriptor map recorded at spawn.
func (o *Owner) FDMap(id Identity) (map[int]string, error) {
	p, err := o.lookup(id)
	if err != nil {
		return nil, err
	}
	m := map[int]string{0: "null", 1: "status-pipe", 2: "stderr-buffer", FDEnv: "env"}
	if p.spec.WithLease {
		m[FDLease] = "lease"
	}
	return m, nil
}

// Facts returns a copy of the current separate facts without blocking.
func (o *Owner) Facts(id Identity) (Facts, error) {
	p, err := o.lookup(id)
	if err != nil {
		return Facts{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.facts
	if p.facts.ChildExit != nil {
		cp := *p.facts.ChildExit
		out.ChildExit = &cp
	}
	return out, nil
}

// Stderr returns the child's captured stderr verbatim. It is observed
// opaquely and never decoded: forged bytes there cannot become facts.
func (o *Owner) Stderr(id Identity) ([]byte, error) {
	p, err := o.lookup(id)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]byte(nil), p.stderr.Bytes()...), nil
}

// Tree returns the current descendant edges of this owner.
func (o *Owner) Tree() []Edge {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []Edge
	for pid, p := range o.procs {
		p.mu.Lock()
		running := !p.exited
		out = append(out, Edge{ParentPID: o.pid, Child: Identity{PID: pid, StartToken: p.id.StartToken}, Running: running})
		p.mu.Unlock()
	}
	return out
}

// Live reports the number of retained children.
func (o *Owner) Live() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.procs)
}

// Signal delivers sig to a running child. Signalling a reaped child
// fails; the signal never reaches a PID-reusing stranger because the
// token is verified first.
func (o *Owner) Signal(id Identity, sig os.Signal) error {
	p, err := o.lookup(id)
	if err != nil {
		return err
	}
	p.mu.Lock()
	exited := p.exited
	p.mu.Unlock()
	if exited {
		return fmt.Errorf("%w: child already reaped", ErrState)
	}
	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("process: signal: %w", err)
	}
	p.mu.Lock()
	p.signals = append(p.signals, sig.String())
	p.mu.Unlock()
	return nil
}

// Kill terminates the direct child with SIGKILL. It is a cleanup
// primitive, not a release path: a killed child cannot produce clean
// release. Kill reaches only the direct child: a grandchild that
// inherited the status pipe keeps it open past the direct child's
// death and pins Wait until EOF, which is why Release reports EOF and
// child-exit as separate facts.
func (o *Owner) Kill(id Identity) error {
	p, err := o.lookup(id)
	if err != nil {
		return err
	}
	_ = p.cmd.Process.Kill()
	return nil
}

// Wait blocks until the child exits or timeout elapses, then records
// the reaped child-exit fact. A timeout leaves the child running for
// the caller to signal or kill. Waiting twice joins the same outcome.
func (o *Owner) Wait(id Identity, timeout time.Duration) (Exit, error) {
	p, err := o.lookup(id)
	if err != nil {
		return Exit{}, err
	}
	p.mu.Lock()
	if p.exited && p.facts.ChildExit != nil {
		exit := *p.facts.ChildExit
		p.mu.Unlock()
		return exit, nil
	}
	p.mu.Unlock()
	if timeout <= 0 {
		timeout = DefaultWaitTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case werr := <-p.waitCh:
		return p.finishWait(werr), nil
	case <-timer.C:
		return Exit{}, fmt.Errorf("%w: pid %d still running", ErrTimeout, id.PID)
	}
}

// joinWait consumes a pending wait outcome without blocking.
func (p *proc) joinWait() {
	select {
	case werr := <-p.waitCh:
		p.finishWait(werr)
	default:
	}
}

func (p *proc) finishWait(waitErr error) Exit {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		if p.facts.ChildExit != nil {
			return *p.facts.ChildExit
		}
		return Exit{}
	}
	p.exited = true
	exit := Exit{}
	if waitErr == nil {
		exit.Code = 0
	} else if ee, ok := waitErr.(*exec.ExitError); ok {
		exit.Code = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			exit.Signaled = true
			exit.Signal = ws.Signal().String()
		}
	} else {
		exit.Code = -1
	}
	p.facts.Reaped = true
	p.facts.ChildExit = &exit
	return exit
}

// joinWriter records the writer-exit fact, waiting up to timeout.
// The writer delivers once; concurrent joiners observe the recorded
// outcome through writerDone instead of starving on the consumed
// channel, so a second Release never waits out a spurious timeout.
func (p *proc) joinWriter(timeout time.Duration) {
	p.mu.Lock()
	if p.facts.WriterDone {
		p.mu.Unlock()
		return
	}
	done := p.writerDone
	p.mu.Unlock()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-p.writerCh:
		p.recordWriter(res)
	case <-done:
	case <-timer.C:
	}
}

// recordWriter stores one writer outcome exactly once and wakes every
// joiner waiting in joinWriter.
func (p *proc) recordWriter(res offerResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.facts.WriterDone {
		return
	}
	p.facts.WriterDone = true
	p.facts.OfferedBytes = res.wrote
	p.facts.Offered = res.wrote == len(p.snapshot)
	if res.err != nil {
		p.facts.WriterErr = res.err.Error()
	}
	close(p.writerDone)
}

// CollectStatus reads the status channel in two bounded stages: one ack
// frame naming the expected content-bound payload, then clean EOF. It
// records the accepted/malformed/EOF facts and returns nil only when
// the ack matched and EOF followed. Timeouts and malformed bytes are
// recorded facts, reported as errors, and never retried implicitly:
// call again with fresh bounds to continue observing.
func (o *Owner) CollectStatus(id Identity, ackTimeout, eofTimeout time.Duration) error {
	p, err := o.lookup(id)
	if err != nil {
		return err
	}
	p.mu.Lock()
	if !p.statusOn {
		p.statusCh = collectStatus(p.status, MaxStatusPayload)
		p.statusOn = true
	}
	alreadyAccepted := p.facts.Accepted
	p.mu.Unlock()

	if !alreadyAccepted {
		res, ok := awaitStatus(p.statusCh, ackTimeout)
		if !ok {
			return fmt.Errorf("%w: waiting for status ack", ErrTimeout)
		}
		p.mu.Lock()
		switch {
		case res.malformed != nil:
			p.facts.Malformed = true
			p.mu.Unlock()
			return fmt.Errorf("process: malformed status frame: %w", res.malformed)
		case res.eof:
			p.facts.EOF = true
			p.mu.Unlock()
			return fmt.Errorf("process: status ended before ack")
		case string(res.payload) != p.expected:
			p.facts.Malformed = true
			p.mu.Unlock()
			return fmt.Errorf("process: status ack names foreign content")
		default:
			p.facts.Accepted = true
			p.mu.Unlock()
		}
	}
	res, ok := awaitStatus(p.statusCh, eofTimeout)
	if !ok {
		return fmt.Errorf("%w: waiting for status EOF", ErrTimeout)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if res.malformed != nil {
		p.facts.Malformed = true
		return fmt.Errorf("process: malformed status trailer: %w", res.malformed)
	}
	if !res.eof {
		p.facts.Malformed = true
		return fmt.Errorf("process: status trailer is not EOF")
	}
	p.facts.EOF = true
	return nil
}

// ExpectedAck renders the exact status frame bytes the child must emit
// to be accepted. Tests embed it in well-behaved child scripts; it
// stands in for a child that read fd 3 and bound its ack to the digest
// of the offered bytes.
func (o *Owner) ExpectedAck(id Identity) ([]byte, error) {
	p, err := o.lookup(id)
	if err != nil {
		return nil, err
	}
	return encodeStatusFrame(p.expected)
}

// Release decides whether the process released cleanly. It joins the
// writer and a pending wait outcome without blocking, then witnesses
// the lease with independent probes: protection must have been held
// while required and must be gone after the owner closes its share.
// Clean needs every clause at once:
//
//	offered, accepted, clean EOF, reaped exit 0, live spawn-start
//	identity, lease inherited, held and released, and no orphan.
//
// A not-clean Report is a decision, not an error: errors are reserved
// for misuse (unknown PID, foreign token, double clean release).
// Release is idempotent until it reports clean.
func (o *Owner) Release(id Identity) (Report, error) {
	p, err := o.lookup(id)
	if err != nil {
		return Report{}, err
	}
	p.mu.Lock()
	if p.releaseClean {
		p.mu.Unlock()
		return Report{}, fmt.Errorf("%w: pid %d", ErrReleased, id.PID)
	}
	p.mu.Unlock()

	p.joinWriter(releaseJoinTimeout)
	p.joinWait()

	rep := Report{ID: p.id, Executable: p.resolved}
	p.mu.Lock()
	rep.Facts = p.facts
	if p.facts.ChildExit != nil {
		cp := *p.facts.ChildExit
		rep.Facts.ChildExit = &cp
	}
	leaseClosed := p.leaseClosed
	p.mu.Unlock()

	rep.Lease.Inherited = p.spec.WithLease
	if p.lease != nil {
		rep.Lease.Path = p.lease.Path()
		if !leaseClosed {
			held, perr := ProbeLease(p.lease.Path())
			if perr != nil {
				return Report{}, perr
			}
			if held {
				p.mu.Lock()
				p.heldSeen = true
				p.mu.Unlock()
			}
			rep.Lease.Held = held || p.heldNow()
			if held && rep.Facts.Reaped {
				// The direct child is gone; close the owner's
				// share so the next probe sees only
				// descendants. A live child keeps its share.
				p.mu.Lock()
				p.leaseClosed = true
				p.mu.Unlock()
				_ = p.lease.Close()
				leaseClosed = true
			}
		} else {
			rep.Lease.Held = p.heldNow()
		}
		if leaseClosed {
			held, perr := ProbeLease(p.lease.Path())
			if perr != nil {
				return Report{}, perr
			}
			rep.Lease.Released = !held
			rep.Orphan = held && rep.Facts.Reaped
		}
	}

	rep.Clean, rep.Reason = cleanDecision(rep)
	if rep.Clean {
		p.mu.Lock()
		p.releaseClean = true
		p.mu.Unlock()
		for _, st := range []journal.State{journal.StateClosing, journal.StateClosed} {
			_, _ = o.j.Advance(p.opID, st)
		}
		_ = p.status.Close()
		o.remove(id.PID)
	} else if terminalReason(rep.Reason) {
		// Transient causes (a running child, a pending EOF, a
		// temporarily held lease) leave the journal live so a
		// later Release can still decide clean.
		_, _ = o.j.Advance(p.opID, journal.StateFailed, truncatePartial(rep.Reason))
	}
	return rep, nil
}

// heldNow reports the persisted protection observation.
func (p *proc) heldNow() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.heldSeen
}

// terminalReason reports whether a refusal cause can never resolve by
// waiting: spawn-time facts, content failures and exit status are
// terminal, while liveness and lease-hold observations are transient.
func terminalReason(reason string) bool {
	switch reason {
	case "environment not offered",
		"status ack not accepted",
		"malformed status frame",
		"malformed status trailer",
		"status trailer is not EOF",
		"child did not exit 0",
		"lease never inherited",
		"lease protection never observed":
		return true
	}
	return false
}

// CloseLeaseEarly drops the owner's share of the generation lease
// before the child exits. It simulates a launcher exit: protection
// continues only while an inheriting descendant still holds fd 4, and
// a release that never witnessed protection cannot be clean.
func (o *Owner) CloseLeaseEarly(id Identity) error {
	p, err := o.lookup(id)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lease == nil || p.leaseClosed {
		return fmt.Errorf("%w: no open lease share", ErrState)
	}
	p.leaseClosed = true
	return p.lease.Close()
}

func cleanDecision(rep Report) (bool, string) {
	f := rep.Facts
	switch {
	case !f.Offered:
		return false, "environment not offered"
	case !f.WriterDone:
		return false, "environment writer still blocked"
	case !f.Accepted:
		return false, "status ack not accepted"
	case f.Malformed:
		return false, "malformed status frame"
	case !f.EOF:
		return false, "status EOF not observed"
	case !f.Reaped:
		return false, "child still running"
	case f.ChildExit == nil || f.ChildExit.Code != 0 || f.ChildExit.Signaled:
		return false, "child did not exit 0"
	case !rep.Lease.Inherited:
		return false, "lease never inherited"
	case !rep.Lease.Held:
		return false, "lease protection never observed"
	case !rep.Lease.Released:
		if rep.Orphan {
			return false, "orphaned descendant still holds the lease"
		}
		return false, "lease not released"
	case rep.Orphan:
		return false, "orphaned descendant still holds the lease"
	}
	return true, ""
}

func truncatePartial(s string) string {
	if len(s) > journal.MaxPartialLen {
		return s[:journal.MaxPartialLen]
	}
	if s == "" {
		return "release refused"
	}
	return s
}

// Abort is the cleanup path for children that cannot release cleanly:
// it kills a running child, reaps it, closes the owner's lease share
// and forgets the child. The journal records the failure. Abort never
// reports clean.
func (o *Owner) Abort(id Identity) error {
	p, err := o.lookup(id)
	if err != nil {
		return err
	}
	_ = p.cmd.Process.Kill()
	p.mu.Lock()
	already := p.exited
	p.mu.Unlock()
	if !already {
		select {
		case werr := <-p.waitCh:
			p.finishWait(werr)
		case <-time.After(DefaultWaitTimeout):
			return fmt.Errorf("%w: aborting pid %d", ErrTimeout, id.PID)
		}
	}
	p.joinWriter(releaseJoinTimeout)
	p.mu.Lock()
	leaseClosed := p.leaseClosed
	if !leaseClosed && p.lease != nil {
		p.leaseClosed = true
	}
	p.mu.Unlock()
	if p.lease != nil && !leaseClosed {
		_ = p.lease.Close()
	}
	_ = p.status.Close()
	_, _ = o.j.Advance(p.opID, journal.StateFailed, "aborted")
	o.remove(id.PID)
	return nil
}

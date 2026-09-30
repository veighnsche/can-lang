package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Transitional bounds. Every field is finite; nothing here means unlimited.
const (
	// DescriptorsPerChild is the stdio descriptor budget charged per child
	// (stdin, stdout, stderr).
	DescriptorsPerChild = 3
	// DefaultStopGrace is the cooperative SIGTERM grace when Stop is asked
	// with a non-positive grace.
	DefaultStopGrace = 200 * time.Millisecond
	// MaxStopGrace caps one Stop grace; the final SIGKILL reap wait is
	// bounded by the same grace again.
	MaxStopGrace = 10 * time.Second
	// logFileName is the append-only registration record inside the witness
	// directory, retained for recovery if T is lost.
	logFileName = "t.jsonl"
)

var (
	ErrInvalid     = errors.New("bootstrap: invalid request")
	ErrConflict    = errors.New("bootstrap: conflicting registration")
	ErrExhausted   = errors.New("bootstrap: finite authority exhausted")
	ErrNotFound    = errors.New("bootstrap: unknown child or scratch")
	ErrAlready     = errors.New("bootstrap: child already started")
	ErrTimeout     = errors.New("bootstrap: deadline exceeded")
	ErrWitnessLost = errors.New("bootstrap: witness lost")
	ErrPriorState  = errors.New("bootstrap: prior witness state present")
)

// Limits is the finite authority envelope pre-registered before any effect.
type Limits struct {
	MaxChildren       int   // >0
	MaxDescriptors    int   // >0; each child charges DescriptorsPerChild
	MaxScratchBytes   int64 // >0 total across scratch roots
	MaxScratchEntries int   // >0 total across scratch roots
}

func (l Limits) valid() bool {
	return l.MaxChildren > 0 && l.MaxDescriptors > 0 && l.MaxScratchBytes > 0 && l.MaxScratchEntries > 0
}

// ScratchSpec pre-registers one owned scratch root and its budgets.
type ScratchSpec struct {
	ID         string
	Path       string // absolute; verified at registration, owned by the disposer
	MaxBytes   int64  // >0
	MaxEntries int    // >0
}

// ChildSpec pre-registers one child process before it is spawned.
type ChildSpec struct {
	ID        string
	Path      string   // executable; resolved at registration for stable identity
	Args      []string // literal argv
	ScratchID string   // optional; must name a registered scratch root
}

// ExitFact is the independently observed terminal fact for a child.
type ExitFact struct {
	Exited bool     // an exit or terminating signal was observed
	Code   int      // exit code when Exited without a signal
	Signal string   // terminating signal name when signaled, else ""
	Reaped bool     // Wait completed; the zombie is gone
	Forced []string // forced actions taken by Stop, in order
}

// ReleaseClaim is a receipt's claimed releases to verify against facts.
type ReleaseClaim struct {
	Children []string // child IDs claimed exited and reaped
	Scratch  []string // scratch IDs claimed removed
}

// MismatchError lists every release-claim problem found by the witness.
// A nil error from CheckRelease with a live witness is the only green.
type MismatchError struct {
	Problems []string
}

func (e *MismatchError) Error() string {
	return "bootstrap: release mismatch: " + strings.Join(e.Problems, "; ")
}

// ChildView is a read-only snapshot of one registered child.
type ChildView struct {
	ID         string
	PID        int
	StartToken string
	Started    bool
	Fact       ExitFact
}

// ScratchView is a read-only snapshot of one registered scratch root.
type ScratchView struct {
	ID         string
	Path       string
	MaxBytes   int64
	MaxEntries int
}

// Snapshot is a read-only view of witness state for tests and recovery.
// WitnessLost reports that T is gone; such a record can never receipt.
type Snapshot struct {
	WitnessLost  bool
	LogFailed    bool
	Children     map[string]ChildView
	Scratch      map[string]ScratchView
	UsedChildren int
	UsedScratchB int64
	UsedScratchE int
}

type child struct {
	spec       ChildSpec
	resolved   string
	cmd        *exec.Cmd
	pid        int
	startToken string
	started    bool
	done       chan struct{}
	forced     []string
	fact       ExitFact
}

type scratchRoot struct {
	spec ScratchSpec
}

// logLine is one JSONL registration record.
type logLine struct {
	Op       string   `json:"op"` // init|scratch|child|start|exit|stop
	ID       string   `json:"id,omitempty"`
	Exec     string   `json:"exec,omitempty"`
	Args     []string `json:"args,omitempty"`
	Scratch  string   `json:"scratch,omitempty"`
	PID      int      `json:"pid,omitempty"`
	Token    string   `json:"token,omitempty"`
	Code     int      `json:"code,omitempty"`
	Signal   string   `json:"signal,omitempty"`
	Forced   []string `json:"forced,omitempty"`
	AtWallMs int64    `json:"atWallMs"`
}

// Witness is the bootstrap parent T. It parents fixture children from
// outside their killable subtree, pre-registers finite authority before
// effects, retains stop/reap, and independently verifies release claims.
// A Witness is safe for concurrent use.
type Witness struct {
	mu           sync.Mutex
	dir          string
	lim          Limits
	scratch      map[string]*scratchRoot
	children     map[string]*child
	usedScratchB int64
	usedScratchE int
	lost         bool
	logFailed    bool
	log          *os.File
}

// New creates a witness owning dir, which must live outside every killable
// test subtree. A non-empty prior record refuses startup with ErrPriorState:
// recovery inspects the old record and starts a new witness, never resumes
// the old run in place.
func New(dir string, lim Limits) (*Witness, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty witness directory", ErrInvalid)
	}
	if !lim.valid() {
		return nil, fmt.Errorf("%w: limits must all be positive", ErrInvalid)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("bootstrap: create directory: %w", err)
	}
	p := filepath.Join(dir, logFileName)
	if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
		return nil, fmt.Errorf("%w: %s", ErrPriorState, p)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("bootstrap: stat record: %w", err)
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: open record: %w", err)
	}
	t := &Witness{
		dir: dir, lim: lim,
		scratch:  make(map[string]*scratchRoot),
		children: make(map[string]*child),
		log:      f,
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.appendLocked(logLine{Op: "init", AtWallMs: time.Now().UnixMilli()}); err != nil {
		f.Close()
		return nil, err
	}
	return t, nil
}

// validID matches the shared wire ID shape.
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

func sameArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func wallMs() int64 { return time.Now().UnixMilli() }

// appendLocked records one line and fsyncs. Any failure latches logFailed
// so later receipt checks fail closed.
func (t *Witness) appendLocked(line logLine) error {
	if t.log == nil {
		t.logFailed = true
		return fmt.Errorf("%w: record unavailable", ErrWitnessLost)
	}
	raw, err := json.Marshal(line)
	if err != nil {
		t.logFailed = true
		return fmt.Errorf("bootstrap: encode record: %w", err)
	}
	raw = append(raw, '\n')
	if _, err := t.log.Write(raw); err != nil {
		t.logFailed = true
		return fmt.Errorf("bootstrap: append record: %w", err)
	}
	if err := t.log.Sync(); err != nil {
		t.logFailed = true
		return fmt.Errorf("bootstrap: sync record: %w", err)
	}
	return nil
}

// RegisterScratch pre-registers one scratch root and charges it against the
// finite envelope. Repeating an identical spec rejoins; a conflicting spec
// under the same ID is rejected.
func (t *Witness) RegisterScratch(spec ScratchSpec) error {
	if !validID(spec.ID) {
		return fmt.Errorf("%w: malformed scratch ID", ErrInvalid)
	}
	if spec.Path == "" || !filepath.IsAbs(spec.Path) {
		return fmt.Errorf("%w: scratch path must be absolute", ErrInvalid)
	}
	if spec.MaxBytes <= 0 || spec.MaxEntries <= 0 {
		return fmt.Errorf("%w: scratch budgets must be positive", ErrInvalid)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lost {
		return ErrWitnessLost
	}
	if prev, dup := t.scratch[spec.ID]; dup {
		if prev.spec == spec {
			return nil
		}
		return fmt.Errorf("%w: scratch %q", ErrConflict, spec.ID)
	}
	if t.usedScratchB+spec.MaxBytes > t.lim.MaxScratchBytes ||
		t.usedScratchE+spec.MaxEntries > t.lim.MaxScratchEntries {
		return fmt.Errorf("%w: scratch envelope", ErrExhausted)
	}
	if err := t.appendLocked(logLine{Op: "scratch", ID: spec.ID, Exec: spec.Path, AtWallMs: wallMs()}); err != nil {
		return err
	}
	t.scratch[spec.ID] = &scratchRoot{spec: spec}
	t.usedScratchB += spec.MaxBytes
	t.usedScratchE += spec.MaxEntries
	return nil
}

// Register pre-registers one child before it may start. The executable is
// resolved now for stable identity; nothing is spawned.
func (t *Witness) Register(spec ChildSpec) error {
	if !validID(spec.ID) {
		return fmt.Errorf("%w: malformed child ID", ErrInvalid)
	}
	if spec.Path == "" {
		return fmt.Errorf("%w: empty executable", ErrInvalid)
	}
	resolved, err := exec.LookPath(spec.Path)
	if err != nil {
		return fmt.Errorf("%w: unresolvable executable %q", ErrInvalid, spec.Path)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lost {
		return ErrWitnessLost
	}
	if prev, dup := t.children[spec.ID]; dup {
		if prev.spec.Path == spec.Path && sameArgs(prev.spec.Args, spec.Args) && prev.spec.ScratchID == spec.ScratchID {
			return nil
		}
		return fmt.Errorf("%w: child %q", ErrConflict, spec.ID)
	}
	if len(t.children) >= t.lim.MaxChildren {
		return fmt.Errorf("%w: child count", ErrExhausted)
	}
	if (len(t.children)+1)*DescriptorsPerChild > t.lim.MaxDescriptors {
		return fmt.Errorf("%w: descriptor budget", ErrExhausted)
	}
	if spec.ScratchID != "" {
		if _, ok := t.scratch[spec.ScratchID]; !ok {
			return fmt.Errorf("%w: scratch %q", ErrNotFound, spec.ScratchID)
		}
	}
	if err := t.appendLocked(logLine{Op: "child", ID: spec.ID, Exec: resolved, Args: spec.Args, Scratch: spec.ScratchID, AtWallMs: wallMs()}); err != nil {
		return err
	}
	t.children[spec.ID] = &child{spec: spec, resolved: resolved, done: make(chan struct{})}
	return nil
}

// Admit is the explicit admission gate: it registers exactly like Register,
// and fails with ErrWitnessLost once T is gone. Loss of T grants no new
// admission.
func (t *Witness) Admit(spec ChildSpec) error {
	return t.Register(spec)
}

// Start spawns a pre-registered child and records its start identity
// (PID plus spawn-start token). A second Start never re-executes.
func (t *Witness) Start(id string) error {
	t.mu.Lock()
	if t.lost {
		t.mu.Unlock()
		return ErrWitnessLost
	}
	c, ok := t.children[id]
	if !ok {
		t.mu.Unlock()
		return fmt.Errorf("%w: child %q", ErrNotFound, id)
	}
	if c.started {
		t.mu.Unlock()
		return fmt.Errorf("%w: child %q", ErrAlready, id)
	}
	dir := ""
	if c.spec.ScratchID != "" {
		s, ok := t.scratch[c.spec.ScratchID]
		if !ok {
			t.mu.Unlock()
			return fmt.Errorf("%w: scratch %q", ErrNotFound, c.spec.ScratchID)
		}
		// The registered authority names the directory; Starting into a
		// missing directory would silently relocate the child.
		if fi, err := os.Stat(s.spec.Path); err != nil || !fi.IsDir() {
			t.mu.Unlock()
			return fmt.Errorf("%w: scratch %q unavailable", ErrInvalid, c.spec.ScratchID)
		}
		dir = s.spec.Path
	}
	cmd := exec.Command(c.resolved, c.spec.Args...)
	cmd.Dir = dir
	// Nil stdio connects the child to the null device: bounded fixtures
	// can never block T on a full pipe.
	if err := cmd.Start(); err != nil {
		t.mu.Unlock()
		return fmt.Errorf("bootstrap: start child %q: %w", id, err)
	}
	c.cmd = cmd
	c.pid = cmd.Process.Pid
	c.startToken = fmt.Sprintf("%d/%d", cmd.Process.Pid, time.Now().UnixNano())
	c.started = true
	if err := t.appendLocked(logLine{Op: "start", ID: id, PID: c.pid, Token: c.startToken, AtWallMs: wallMs()}); err != nil {
		// Effect without a record is unacceptable: kill what we started.
		// No reap goroutine is running on this path, so record inline.
		proc := cmd.Process
		t.mu.Unlock()
		_ = proc.Kill()
		_ = cmd.Wait()
		t.mu.Lock()
		c.fact = ExitFact{Exited: true, Signal: "killed", Reaped: true, Forced: []string{"SIGKILL"}}
		close(c.done)
		t.mu.Unlock()
		return err
	}
	t.mu.Unlock()
	go t.reap(id, cmd, c.done)
	return nil
}

// reap waits for one child, records the independent exit fact, and releases
// the zombie. It never applies case policy; it only observes.
func (t *Witness) reap(id string, cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()
	code, signal, exited := exitOf(err)
	reaped := exited // Wait returned, so the zombie is gone when known.
	t.mu.Lock()
	defer t.mu.Unlock()
	c, ok := t.children[id]
	if !ok {
		return
	}
	c.fact = ExitFact{Exited: exited, Code: code, Signal: signal, Reaped: reaped, Forced: append([]string(nil), c.forced...)}
	_ = t.appendLocked(logLine{Op: "exit", ID: id, PID: c.pid, Code: code, Signal: signal, Forced: c.forced, AtWallMs: wallMs()})
	select {
	case <-done:
	default:
		close(done)
	}
}

func exitOf(err error) (code int, signal string, exited bool) {
	if err == nil {
		return 0, "", true
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return -1, "", false
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
		if ws.Signaled() {
			return -1, strings.ToLower(ws.Signal().String()), true
		}
		return ws.ExitStatus(), "", true
	}
	return ee.ExitCode(), "", true
}

// Wait blocks until the child's exit fact is recorded or the timeout
// elapses. A timeout is a deadline on the wait only; the child stays owned
// and CheckRelease will refuse to receipt it.
func (t *Witness) Wait(id string, timeout time.Duration) (ExitFact, error) {
	t.mu.Lock()
	if t.lost {
		t.mu.Unlock()
		return ExitFact{}, ErrWitnessLost
	}
	c, ok := t.children[id]
	if !ok {
		t.mu.Unlock()
		return ExitFact{}, fmt.Errorf("%w: child %q", ErrNotFound, id)
	}
	if !c.started {
		t.mu.Unlock()
		return ExitFact{}, fmt.Errorf("%w: child %q not started", ErrInvalid, id)
	}
	done := c.done
	t.mu.Unlock()
	if timeout <= 0 {
		timeout = DefaultStopGrace
	}
	select {
	case <-done:
		t.mu.Lock()
		defer t.mu.Unlock()
		f := t.children[id].fact
		f.Forced = append([]string(nil), f.Forced...)
		return f, nil
	case <-time.After(timeout):
		return ExitFact{}, fmt.Errorf("%w: wait for child %q", ErrTimeout, id)
	}
}

// Stop applies bounded TERM-then-KILL escalation and joins the reap.
// Repeating Stop on an exited child joins the same disposal.
func (t *Witness) Stop(id string, grace time.Duration) error {
	if grace <= 0 {
		grace = DefaultStopGrace
	}
	if grace > MaxStopGrace {
		grace = MaxStopGrace
	}
	t.mu.Lock()
	if t.lost {
		t.mu.Unlock()
		return ErrWitnessLost
	}
	c, ok := t.children[id]
	if !ok {
		t.mu.Unlock()
		return fmt.Errorf("%w: child %q", ErrNotFound, id)
	}
	if !c.started {
		t.mu.Unlock()
		return fmt.Errorf("%w: child %q not started", ErrInvalid, id)
	}
	select {
	case <-c.done:
		t.mu.Unlock()
		return nil // already exited: join the recorded disposal
	default:
	}
	proc := c.cmd.Process
	done := c.done
	t.mu.Unlock()

	if err := proc.Signal(syscall.SIGTERM); err == nil {
		t.recordForced(id, "SIGTERM")
	}
	select {
	case <-done:
		t.logStop(id)
		return nil
	case <-time.After(grace):
	}
	if err := proc.Kill(); err == nil {
		t.recordForced(id, "SIGKILL")
	}
	select {
	case <-done:
		t.logStop(id)
		return nil
	case <-time.After(grace):
		return fmt.Errorf("%w: stop child %q; still owned and unresolved", ErrTimeout, id)
	}
}

func (t *Witness) recordForced(id, action string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.children[id]; ok {
		c.forced = append(c.forced, action)
	}
}

func (t *Witness) logStop(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.children[id]; ok {
		_ = t.appendLocked(logLine{Op: "stop", ID: id, PID: c.pid, Forced: c.forced, AtWallMs: wallMs()})
	}
}

// CheckRelease independently verifies a claimed receipt against actual
// release facts: every claimed child must show observed exit plus reap,
// every claimed scratch must be observably absent, and every registered
// authority must be claimed. Anything else — including a lost witness or
// a failed record — refuses the green.
func (t *Witness) CheckRelease(claim ReleaseClaim) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lost {
		return ErrWitnessLost
	}
	if t.logFailed {
		return errors.New("bootstrap: release witness unavailable: record failed")
	}
	var problems []string
	claimedChildren := make(map[string]bool, len(claim.Children))
	for _, id := range claim.Children {
		claimedChildren[id] = true
		c, ok := t.children[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("unknown child %q", id))
			continue
		}
		if !c.fact.Exited || !c.fact.Reaped {
			problems = append(problems, fmt.Sprintf("child %q not released (exited=%v reaped=%v)", id, c.fact.Exited, c.fact.Reaped))
		}
	}
	claimedScratch := make(map[string]bool, len(claim.Scratch))
	for _, id := range claim.Scratch {
		claimedScratch[id] = true
		s, ok := t.scratch[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("unknown scratch %q", id))
			continue
		}
		fi, err := os.Stat(s.spec.Path)
		if err == nil && fi != nil {
			problems = append(problems, fmt.Sprintf("scratch %q still present", id))
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			problems = append(problems, fmt.Sprintf("scratch %q unverifiable: %v", id, err))
		}
	}
	for id := range t.children {
		if !claimedChildren[id] {
			problems = append(problems, fmt.Sprintf("child %q unclaimed", id))
		}
	}
	for id := range t.scratch {
		if !claimedScratch[id] {
			problems = append(problems, fmt.Sprintf("scratch %q unclaimed", id))
		}
	}
	if len(problems) > 0 {
		return &MismatchError{Problems: problems}
	}
	return nil
}

// Snapshot copies current witness state for tests and recovery tooling.
// It never grants authority.
func (t *Witness) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	snap := Snapshot{
		WitnessLost:  t.lost,
		LogFailed:    t.logFailed,
		Children:     make(map[string]ChildView, len(t.children)),
		Scratch:      make(map[string]ScratchView, len(t.scratch)),
		UsedChildren: len(t.children),
		UsedScratchB: t.usedScratchB,
		UsedScratchE: t.usedScratchE,
	}
	for id, c := range t.children {
		f := c.fact
		f.Forced = append([]string(nil), f.Forced...)
		snap.Children[id] = ChildView{ID: id, PID: c.pid, StartToken: c.startToken, Started: c.started, Fact: f}
	}
	for id, s := range t.scratch {
		snap.Scratch[id] = ScratchView{ID: id, Path: s.spec.Path, MaxBytes: s.spec.MaxBytes, MaxEntries: s.spec.MaxEntries}
	}
	return snap
}

// Close marks T lost and releases the record. This simulates witness loss:
// afterwards Admit/Register/Start/Stop/Wait/CheckRelease all fail with
// ErrWitnessLost, so the loss can neither issue a green receipt nor grant
// new admission. Running children are left to the test process; reap
// observers already in flight still record their facts for inspection.
func (t *Witness) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lost {
		return nil
	}
	t.lost = true
	if t.log == nil {
		return nil
	}
	serr := t.log.Sync()
	cerr := t.log.Close()
	t.log = nil
	if serr != nil {
		return fmt.Errorf("bootstrap: sync on close: %w", serr)
	}
	if cerr != nil {
		return fmt.Errorf("bootstrap: close: %w", cerr)
	}
	return nil
}

// Inspect replays a retained witness record without taking ownership. The
// returned snapshot always reports WitnessLost: a dead T's record supports
// recovery bookkeeping only and can never receipt or admit.
func Inspect(dir string) (Snapshot, error) {
	snap := Snapshot{
		WitnessLost: true,
		Children:    make(map[string]ChildView),
		Scratch:     make(map[string]ScratchView),
	}
	raw, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		return Snapshot{}, fmt.Errorf("bootstrap: read record: %w", err)
	}
	endsNL := len(raw) > 0 && raw[len(raw)-1] == '\n'
	start := 0
	for start < len(raw) {
		end := start
		for end < len(raw) && raw[end] != '\n' {
			end++
		}
		line := raw[start:end]
		isLast := end == len(raw)
		if len(line) == 0 {
			start = end + 1
			continue
		}
		var ll logLine
		if err := json.Unmarshal(line, &ll); err != nil {
			if isLast && !endsNL {
				break // torn tail from a crash mid-write
			}
			return Snapshot{}, fmt.Errorf("bootstrap: corrupt record at offset %d: %v", start, err)
		}
		switch ll.Op {
		case "init":
		case "scratch":
			snap.Scratch[ll.ID] = ScratchView{ID: ll.ID, Path: ll.Exec}
		case "child":
			snap.Children[ll.ID] = ChildView{ID: ll.ID}
		case "start":
			v := snap.Children[ll.ID]
			v.Started = true
			v.PID = ll.PID
			v.StartToken = ll.Token
			snap.Children[ll.ID] = v
		case "exit":
			v := snap.Children[ll.ID]
			v.Fact = ExitFact{Exited: true, Code: ll.Code, Signal: ll.Signal, Reaped: true, Forced: ll.Forced}
			snap.Children[ll.ID] = v
		case "stop":
			// Disposal marker only; exit facts carry the release proof.
		default:
			return Snapshot{}, fmt.Errorf("bootstrap: corrupt record op %q", ll.Op)
		}
		start = end + 1
	}
	snap.UsedChildren = len(snap.Children)
	return snap, nil
}

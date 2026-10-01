// Owner-side handle for the independent observer child process.
//
// The N/T owner spawns the observer with StartObserver and drives the
// interval over the wire: samples, ticks, driver-death notes,
// corrections, interruption, and the disposal seal. The handle detects
// child death (clean or crash) and refuses further commands with
// observer-lost; after loss the owner replays the durable journal with
// ReplayAfterLoss, which recovers acknowledged facts and marks the
// interval unknown. A missing observer blocks launch: StartObserver
// returns no handle and no attestation unless the child boots, binds its
// granted scope, and attests the launch over the real channel.
package browser_observer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// maxChildStderr caps captured child diagnostics.
const maxChildStderr = 64 * 1024

// Handle drives one observer child over the wire. It is safe for
// concurrent use; requests serialize in call order.
type Handle struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	stderr   *boundedBuffer
	token    string
	attest   LaunchAttestation
	spec     ChildSpec
	dead     chan struct{}
	deadOnce sync.Once
	waitErr  error
	closed   bool
}

// StartObserver spawns the observer child and binds the launch. binary
// is the observer host executable; the bounded controls pass the test
// binary for re-exec, and production passes the owner-built observer
// binary. The child shares no driver code, flags, environment, or
// process state: it is the owner's direct child and the driver's
// sibling, never a driver descendant.
func StartObserver(binary string, spec ChildSpec) (*Handle, error) {
	encoded, err := EncodeChildSpec(spec)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(), ChildEnvGate+"=1", ChildEnvSpec+"="+encoded)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("browser_observer: observer stdin: %w", err)
	}
	pipeOut, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("browser_observer: observer stdout: %w", err)
	}
	stderr := &boundedBuffer{limit: maxChildStderr}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("browser_observer: start observer: %w", err)
	}
	h := &Handle{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(pipeOut),
		stderr: stderr,
		spec:   spec,
		dead:   make(chan struct{}),
	}
	go func() {
		h.waitErr = cmd.Wait()
		h.deadOnce.Do(func() { close(h.dead) })
	}()
	ready, err := h.readReady()
	if err != nil {
		h.kill()
		<-h.dead
		stdin.Close()
		return nil, err
	}
	h.token = ready.Token
	h.attest = ready.Attestation
	return h, nil
}

func (h *Handle) readReady() (readyReport, error) {
	line, err := readLine(h.stdout)
	if err != nil {
		return readyReport{}, h.bootFailure(fmt.Errorf("browser_observer: observer boot: %w", err))
	}
	var ready readyReport
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ready); err != nil || dec.More() || !ready.Ready || ready.Token == "" {
		return readyReport{}, h.bootFailure(fmt.Errorf("browser_observer: observer boot report is corrupt"))
	}
	return ready, nil
}

func (h *Handle) bootFailure(err error) error {
	if captured := h.stderr.String(); captured != "" {
		return fmt.Errorf("%w (child diagnostics: %s)", err, captured)
	}
	return err
}

// Attestation reports the launch attestation received over the channel.
func (h *Handle) Attestation() LaunchAttestation { return h.attest }

// JournalPath reports the durable publication path for this interval.
func (h *Handle) JournalPath() string { return h.spec.JournalPath }

// Pid reports the observer child pid. Bounded controls use it to prove
// the observer is the owner's direct child and outside the driver's
// killable subtree.
func (h *Handle) Pid() int {
	if h.cmd.Process == nil {
		return -1
	}
	return h.cmd.Process.Pid
}

// Alive reports whether the child is still running.
func (h *Handle) Alive() bool {
	select {
	case <-h.dead:
		return false
	default:
		return true
	}
}

// SetTokenForTest swaps the presented launch token so bounded controls
// can prove forged tokens reject over the real channel.
func (h *Handle) SetTokenForTest(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.token = token
}

// roundTrip sends one request and reads one response. Any channel
// failure (broken pipe, EOF, corrupt frame, child death) is observer
// loss: the interval can no longer be observed.
func (h *Handle) roundTrip(op, channel, state string, tick int, after string) (json.RawMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.dead:
		return nil, obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: observer child is gone", ErrDenied))
	default:
	}
	line, err := encodeRequest(wireRequest{Op: op, Token: h.token, Channel: channel, State: state, Tick: tick, After: after})
	if err != nil {
		return nil, err
	}
	if _, err := h.stdin.Write(line); err != nil {
		return nil, h.lostLocked(fmt.Errorf("browser_observer: observer write: %w", err))
	}
	respLine, err := readLine(h.stdout)
	if err != nil {
		return nil, h.lostLocked(fmt.Errorf("browser_observer: observer read: %w", err))
	}
	resp, err := decodeResponse(respLine)
	if err != nil {
		return nil, h.lostLocked(err)
	}
	if !resp.Ok {
		return nil, responseError(resp)
	}
	return resp.Body, nil
}

func (h *Handle) lostLocked(err error) error {
	h.deadOnce.Do(func() { close(h.dead) })
	return obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: %v", ErrDenied, err))
}

func decodeBody(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return obsErr(LayerObserver, CodeObserverLost, fmt.Errorf("%w: corrupt wire facts", ErrDenied))
	}
	return nil
}

// Sample records one host-UI sensor sample at the open tick over the
// wire. In the bounded controls the bytes come from a pipe-fed stub;
// the qualified host profile wires the real host-UI sensor here.
func (h *Handle) Sample(channel, state string) (TickFacts, error) {
	raw, err := h.roundTrip(OpSample, channel, state, 0, "")
	if err != nil {
		return TickFacts{}, err
	}
	var facts TickFacts
	if err := decodeBody(raw, &facts); err != nil {
		return TickFacts{}, err
	}
	return facts, nil
}

// EndTick closes the open tick over the wire.
func (h *Handle) EndTick() (TickFacts, error) {
	raw, err := h.roundTrip(OpEndTick, "", "", 0, "")
	if err != nil {
		return TickFacts{}, err
	}
	var facts TickFacts
	if err := decodeBody(raw, &facts); err != nil {
		return TickFacts{}, err
	}
	return facts, nil
}

// NoteDriverDeath notes the driver's death over the wire. The interval
// continues: late host effects still observe through disposal.
func (h *Handle) NoteDriverDeath() error {
	_, err := h.roundTrip(OpNoteDriverDeath, "", "", 0, "")
	return err
}

// PublishCorrection publishes one durable correction over the wire.
func (h *Handle) PublishCorrection(tick int, channel, after string) (CorrectionFacts, error) {
	raw, err := h.roundTrip(OpPublishCorr, channel, "", tick, after)
	if err != nil {
		return CorrectionFacts{}, err
	}
	var facts CorrectionFacts
	if err := decodeBody(raw, &facts); err != nil {
		return CorrectionFacts{}, err
	}
	return facts, nil
}

// MarkUnknown publishes a durable unknown over the wire.
func (h *Handle) MarkUnknown(tick int, channel string) (CorrectionFacts, error) {
	raw, err := h.roundTrip(OpMarkUnknown, channel, "", tick, "")
	if err != nil {
		return CorrectionFacts{}, err
	}
	var facts CorrectionFacts
	if err := decodeBody(raw, &facts); err != nil {
		return CorrectionFacts{}, err
	}
	return facts, nil
}

// SealDisposal seals the interval through confirmed disposal over the
// wire.
func (h *Handle) SealDisposal() (DisposalReceipt, error) {
	raw, err := h.roundTrip(OpSealDisposal, "", "", 0, "")
	if err != nil {
		return DisposalReceipt{}, err
	}
	var receipt DisposalReceipt
	if err := decodeBody(raw, &receipt); err != nil {
		return DisposalReceipt{}, err
	}
	return receipt, nil
}

// IntervalFacts reads the interval evidence over the wire.
func (h *Handle) IntervalFacts() (IntervalFacts, error) {
	raw, err := h.roundTrip(OpIntervalFacts, "", "", 0, "")
	if err != nil {
		return IntervalFacts{}, err
	}
	var facts IntervalFacts
	if err := decodeBody(raw, &facts); err != nil {
		return IntervalFacts{}, err
	}
	return facts, nil
}

// Interrupt suspends observation over the wire.
func (h *Handle) Interrupt() error {
	_, err := h.roundTrip(OpInterrupt, "", "", 0, "")
	return err
}

// Resume resumes an interrupted observer over the wire.
func (h *Handle) Resume() error {
	_, err := h.roundTrip(OpResume, "", "", 0, "")
	return err
}

// Kill SIGKILLs the observer child, simulating a crash. The journal
// keeps every acknowledged mutation; the interval becomes unknown.
func (h *Handle) Kill() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.kill()
}

func (h *Handle) kill() error {
	if h.cmd.Process == nil {
		return nil
	}
	if err := h.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("browser_observer: kill observer: %w", err)
	}
	return nil
}

// Close shuts the child down cleanly and reaps it. It is safe to call
// Close after Kill; the child is always reaped exactly once by the
// waiter goroutine.
func (h *Handle) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		<-h.dead
		return nil
	}
	h.closed = true
	select {
	case <-h.dead:
		h.mu.Unlock()
		// The child is already gone, but the stdin pipe still needs
		// exactly one close (the closed flag makes this path once-only).
		h.stdin.Close()
		<-h.dead
		return nil
	default:
	}
	line, err := encodeRequest(wireRequest{Op: OpShutdown, Token: h.token})
	if err != nil {
		h.mu.Unlock()
		return err
	}
	_, _ = h.stdin.Write(line)
	h.mu.Unlock()
	// A failed shutdown write means the child is already gone; reaping
	// is still required and still bounded by the waiter.
	<-h.dead
	h.stdin.Close()
	return nil
}

// LostInterval replays the durable journal after observer loss, marks
// the recovered interval observer-lost, and seals it. Acknowledged ticks
// and corrections come back; the loss itself makes the interval unknown,
// and any journal truncation is reported instead of hidden. An interval
// already disposed before the loss keeps its original receipt.
func (h *Handle) LostInterval() (IntervalFacts, DisposalReceipt, ReplayResult, error) {
	engine, token, result, err := Replay(h.spec.JournalPath)
	if err != nil {
		return IntervalFacts{}, DisposalReceipt{}, ReplayResult{}, err
	}
	if lerr := engine.LoseObserver(h.spec.Owner, h.spec.Scope); lerr != nil {
		return IntervalFacts{}, DisposalReceipt{}, ReplayResult{}, lerr
	}
	facts, err := engine.IntervalFacts(h.spec.Owner, h.spec.Scope, h.spec.LaunchID)
	if err != nil {
		return IntervalFacts{}, DisposalReceipt{}, ReplayResult{}, err
	}
	var receipt DisposalReceipt
	if facts.Disposed {
		receipt, err = engine.DisposalReceipt(h.spec.Owner, h.spec.Scope, h.spec.LaunchID)
	} else {
		receipt, err = engine.SealDisposal(h.spec.Owner, h.spec.Scope, h.spec.LaunchID, token)
	}
	if err != nil {
		return IntervalFacts{}, DisposalReceipt{}, ReplayResult{}, err
	}
	return facts, receipt, result, nil
}

type boundedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	room := b.limit - b.buf.Len()
	if room <= 0 {
		return len(p), nil
	}
	if len(p) > room {
		p = p[:room]
	}
	return b.buf.Write(p)
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

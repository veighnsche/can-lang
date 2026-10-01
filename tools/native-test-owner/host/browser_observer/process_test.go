// Independent-observer mechanism controls over the real channel.
//
// Every control here spawns real OS processes (re-execed test binary,
// no fixture binaries, no network, no live hosts): the observer child
// as the owner's direct child and sibling of the driver stub, never a
// driver descendant. Signals are real (SIGKILL), pipes are real
// (framed JSON over stdin/stdout), and publication is a real fsync'd
// file. Cleanup reaps every child; journals live in t.TempDir.
package browser_observer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func procOwner() Owner {
	return Owner{PID: os.Getpid(), StartToken: "proc-start-1"}
}

func procSpec(t *testing.T) ChildSpec {
	t.Helper()
	return ChildSpec{
		Owner:       procOwner(),
		Scope:       "host-ui",
		Channels:    []string{"window", "icon"},
		Limits:      DefaultLimits(),
		LaunchID:    "launch-1",
		JournalPath: filepath.Join(t.TempDir(), "pub.log"),
	}
}

func startObserver(t *testing.T, spec ChildSpec) *Handle {
	t.Helper()
	h, err := StartObserver(os.Args[0], spec)
	if err != nil {
		t.Fatalf("StartObserver: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	if !h.Alive() {
		t.Fatal("observer not alive after boot")
	}
	return h
}

// driverStub is the killable driver subtree in miniature: a real
// sibling process that blocks until killed.
type driverStub struct {
	cmd *exec.Cmd
}

func startDriverStub(t *testing.T) *driverStub {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), DriverEnvGate+"=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("driver stdin: %v", err)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start driver: %v", err)
	}
	d := &driverStub{cmd: cmd}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = stdin.Close()
	})
	return d
}

func (d *driverStub) kill(t *testing.T) {
	t.Helper()
	if err := d.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill driver: %v", err)
	}
	if err := d.cmd.Wait(); err == nil {
		t.Fatal("killed driver reaped clean")
	}
}

func journalLineCount(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatalf("journal has a torn tail: %q", raw)
	}
	return strings.Count(string(raw), "\n")
}

func verifyIntervalDigest(t *testing.T, facts IntervalFacts) {
	t.Helper()
	death, disposed, unknown := "0", "0", "0"
	if facts.DriverDeathNoted {
		death = "1"
	}
	if facts.Disposed {
		disposed = "1"
	}
	if facts.Unknown {
		unknown = "1"
	}
	flags := "driver-death:" + death + "|disposed:" + disposed + "|unknown:" + unknown
	if got := DigestInterval(facts.LaunchID, facts.Ticks, facts.Corrections, flags); got != facts.Digest {
		t.Fatalf("interval digest does not re-verify: %s != %s", got, facts.Digest)
	}
}

func TestObserverChannelRoundTrip(t *testing.T) {
	spec := procSpec(t)
	h := startObserver(t, spec)
	attest := h.Attestation()
	if attest.LaunchID != "launch-1" || attest.Scope != "host-ui" || attest.Tick != 0 {
		t.Fatalf("bad attestation over the channel: %+v", attest)
	}
	if attest.Owner.PID != os.Getpid() || attest.Owner.StartToken != "proc-start-1" {
		t.Fatalf("attestation names the wrong owner: %+v", attest.Owner)
	}
	for _, digest := range []string{attest.HandleDigest, attest.ObserverDigest, attest.ScopeDigest} {
		if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
			t.Fatalf("attestation carries no digest-only handle: %q", digest)
		}
	}
	if h.Pid() <= 0 || h.Pid() == os.Getpid() {
		t.Fatalf("observer pid is not a distinct child: %d", h.Pid())
	}
	// Tick 0 over the wire, then a late tick, then the seal.
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	sealed, err := h.EndTick()
	if err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	if !sealed.Closed || sealed.Tick != 0 {
		t.Fatalf("bad sealed tick over the channel: %+v", sealed)
	}
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	receipt, err := h.SealDisposal()
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if receipt.Ticks != 2 || receipt.Unknown || receipt.Corrections != 0 {
		t.Fatalf("bad disposal receipt over the channel: %+v", receipt)
	}
	facts, err := h.IntervalFacts()
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	verifyIntervalDigest(t, facts)
	verdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: facts})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if verdict.UISeen || verdict.Unknown {
		t.Fatalf("clean channeled interval misjudged: %+v", verdict)
	}
	// Durable publication: header + 4 samples + end-tick + seal.
	if n := journalLineCount(t, spec.JournalPath); n != 7 {
		t.Fatalf("journal holds %d lines, want 7", n)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if h.Alive() {
		t.Fatal("observer alive after clean shutdown")
	}
}

func TestObservationSurvivesDriverDeath(t *testing.T) {
	spec := procSpec(t)
	h := startObserver(t, spec)
	driver := startDriverStub(t)
	if h.Pid() == driver.cmd.Process.Pid {
		t.Fatal("observer and driver share a pid")
	}
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.EndTick(); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	// Real SIGKILL of the driver mid-interval. The observer is the
	// driver's sibling, never its descendant, so it survives.
	driver.kill(t)
	if err := h.NoteDriverDeath(); err != nil {
		t.Fatalf("NoteDriverDeath after real driver death: %v", err)
	}
	if !h.Alive() {
		t.Fatal("observer died with the driver")
	}
	// Late host effects after driver death still observe.
	if _, err := h.Sample("window", StateSeen); err != nil {
		t.Fatalf("late Sample: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err != nil {
		t.Fatalf("late Sample: %v", err)
	}
	receipt, err := h.SealDisposal()
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if !receipt.DriverDeathNoted || receipt.Unknown {
		t.Fatalf("bad post-death receipt: %+v", receipt)
	}
	facts, err := h.IntervalFacts()
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	verifyIntervalDigest(t, facts)
	verdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: facts})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if !verdict.UISeen || verdict.Unknown {
		t.Fatalf("late effect after driver death misjudged: %+v", verdict)
	}
	// Header + 2 samples + end-tick + death note + 2 late samples + seal.
	if n := journalLineCount(t, spec.JournalPath); n != 8 {
		t.Fatalf("journal holds %d lines, want 8", n)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestObserverLossMakesIntervalUnknown(t *testing.T) {
	spec := procSpec(t)
	h := startObserver(t, spec)
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if _, err := h.EndTick(); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	// Real SIGKILL of the observer: no graceful shutdown, no flush
	// beyond the fsync'd journal.
	if err := h.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close after kill: %v", err)
	}
	if h.Alive() {
		t.Fatal("observer alive after SIGKILL")
	}
	// The dead channel refuses every command as observer loss.
	if _, err := h.Sample("window", StateAbsent); err == nil {
		t.Fatal("dead observer sampled")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	if _, err := h.EndTick(); err == nil {
		t.Fatal("dead observer ended a tick")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	if err := h.NoteDriverDeath(); err == nil {
		t.Fatal("dead observer noted a death")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	// The journal replays the acknowledged prefix; the loss marks the
	// interval unknown and the seal says so honestly.
	facts, receipt, result, err := h.LostInterval()
	if err != nil {
		t.Fatalf("LostInterval: %v", err)
	}
	if result.Truncated || result.Entries != 4 {
		t.Fatalf("loss replay wrong: %+v", result)
	}
	if !facts.Unknown || facts.Disposed {
		t.Fatalf("lost interval not unknown: %+v", facts)
	}
	if len(facts.Ticks) != 2 {
		t.Fatalf("acknowledged prefix lost: %+v", facts)
	}
	// The acknowledged sample survived; the unobserved channel went
	// unknown through the loss itself.
	seen := map[string]ChannelFacts{}
	for _, entry := range facts.Ticks[1].Channels {
		seen[entry.Channel] = entry
	}
	if seen["window"].State != StateAbsent || seen["window"].Origin != "observed" {
		t.Fatalf("acknowledged sample lost: %+v", facts.Ticks[1])
	}
	if seen["icon"].State != StateUnknown || seen["icon"].Origin != "loss" {
		t.Fatalf("loss did not mark the unobserved channel: %+v", facts.Ticks[1])
	}
	if !receipt.Unknown || receipt.Ticks != 2 {
		t.Fatalf("lost interval sealed known: %+v", receipt)
	}
	verifyIntervalDigest(t, facts)
	verdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: facts})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if verdict.UISeen || !verdict.Unknown {
		t.Fatalf("lost interval judged known: %+v", verdict)
	}
}

func TestDurableCorrectionSurvivesObserverCrash(t *testing.T) {
	spec := procSpec(t)
	h := startObserver(t, spec)
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	tick0, err := h.EndTick()
	if err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	for _, entry := range tick0.Channels {
		if entry.Channel == "icon" && entry.State != StateGap {
			t.Fatalf("silence over the channel is not a gap: %+v", entry)
		}
	}
	first, err := h.PublishCorrection(0, "icon", StateAbsent)
	if err != nil {
		t.Fatalf("PublishCorrection: %v", err)
	}
	if first.CorrectionID != "c1" || first.Before != StateGap {
		t.Fatalf("bad correction over the channel: %+v", first)
	}
	// Crash without any graceful shutdown: the acknowledged correction
	// must still publish durably.
	if err := h.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close after kill: %v", err)
	}
	facts, receipt, result, err := h.LostInterval()
	if err != nil {
		t.Fatalf("LostInterval: %v", err)
	}
	if result.Truncated || result.Entries != 3 {
		t.Fatalf("correction replay wrong: %+v", result)
	}
	if len(facts.Corrections) != 1 || facts.Corrections[0].CorrectionID != "c1" {
		t.Fatalf("correction not durable across the crash: %+v", facts)
	}
	if facts.Corrections[0].Before != StateGap || facts.Corrections[0].After != StateAbsent {
		t.Fatalf("correction content lost: %+v", facts.Corrections[0])
	}
	if !facts.Unknown || !receipt.Unknown {
		t.Fatalf("crashed interval sealed known: %+v / %+v", facts, receipt)
	}
	// Single-shot survives the crash too: replaying again cannot
	// re-correct, because replay rebuilds from the same journal.
	again, _, _, err := h.LostInterval()
	if err != nil {
		t.Fatalf("LostInterval again: %v", err)
	}
	if len(again.Corrections) != 1 {
		t.Fatalf("replay diverged: %+v", again)
	}
}

func TestMissingObserverProcessBlocksLaunch(t *testing.T) {
	// No binary, no observer, no attestation.
	if _, err := StartObserver(filepath.Join(t.TempDir(), "no-such-binary"), procSpec(t)); err == nil {
		t.Fatal("launch admitted with no observer binary")
	}
	// A child that cannot bind (uncreatable journal) attests nothing.
	bad := procSpec(t)
	bad.JournalPath = filepath.Join(t.TempDir(), "no-such-dir", "pub.log")
	if _, err := StartObserver(os.Args[0], bad); err == nil {
		t.Fatal("launch admitted by an unbindable observer")
	}
	// A dead observer admits nothing further: every command refuses.
	h := startObserver(t, procSpec(t))
	if err := h.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := h.Sample("window", StateAbsent); err == nil {
		t.Fatal("dead observer sampled")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	if _, err := h.IntervalFacts(); err == nil {
		t.Fatal("dead observer read facts")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	if _, err := h.SealDisposal(); err == nil {
		t.Fatal("dead observer sealed")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
}

func TestWireNegativesOverChannel(t *testing.T) {
	h := startObserver(t, procSpec(t))
	// Unknown effect keys and out-of-scope channels reject over the wire
	// without effect.
	for _, bad := range []string{"", "cursor", "WINDOW", "gpu-usage"} {
		if _, err := h.Sample(bad, StateAbsent); err == nil {
			t.Fatalf("wire accepted unknown effect %q", bad)
		} else {
			requireObsErr(t, err, LayerInterval, CodeUnknownEffect)
		}
	}
	for _, foreign := range []string{"notification", "focus"} {
		if _, err := h.Sample(foreign, StateAbsent); err == nil {
			t.Fatalf("wire accepted out-of-scope channel %q", foreign)
		} else {
			requireObsErr(t, err, LayerInterval, CodeOutOfScope)
		}
	}
	if _, err := h.Sample("window", "maybe"); err == nil {
		t.Fatal("wire accepted a derived state")
	} else {
		requireObsErr(t, err, LayerInterval, CodeUnknownEffect)
	}
	// Interruption suspends the channeled observer; the open tick's
	// silence becomes unknown and observation resumes after.
	if _, err := h.Sample("window", StateAbsent); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if err := h.Interrupt(); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	if _, err := h.Sample("icon", StateAbsent); err == nil {
		t.Fatal("interrupted observer sampled")
	} else {
		requireObsErr(t, err, LayerObserver, CodeInterrupted)
	}
	if err := h.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	facts, err := h.IntervalFacts()
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if !facts.Unknown {
		t.Fatal("interrupted interval is not unknown")
	}
	// The forged-token control runs last: it wedges this handle by
	// design, since the true token never crosses back.
	h.SetTokenForTest("lnch-forged")
	if _, err := h.Sample("window", StateAbsent); err == nil {
		t.Fatal("wire accepted a forged token")
	} else {
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
	}
	if _, err := h.SealDisposal(); err == nil {
		t.Fatal("wire sealed on a forged token")
	} else {
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
	}
}

func TestCorruptWireFrameFailsClosed(t *testing.T) {
	spec := procSpec(t)
	encoded, err := EncodeChildSpec(spec)
	if err != nil {
		t.Fatalf("EncodeChildSpec: %v", err)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), ChildEnvGate+"=1", ChildEnvSpec+"="+encoded)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin: %v", err)
	}
	pipeOut, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout: %v", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	out := bufio.NewReader(pipeOut)
	line, err := readLine(out)
	if err != nil || len(line) == 0 {
		t.Fatalf("no boot report: %v", err)
	}
	// One corrupt frame: the child exits fail-closed instead of
	// inventing a reply, and the owner reads EOF (loss), not facts.
	if _, err := fmt.Fprintln(stdin, "this is not a frame {{{"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := readLine(out); err == nil {
		t.Fatal("corrupt frame drew a reply")
	}
	_ = stdin.Close()
	werr := cmd.Wait()
	if werr == nil {
		t.Fatal("child exited clean on a corrupt frame")
	}
	if exit, ok := werr.(*exec.ExitError); !ok || exit.ExitCode() != ChildExitChannel {
		t.Fatalf("child exit is not fail-closed: %v", werr)
	}
	// The interval still replays its acknowledged (empty) prefix and
	// seals unknown: corruption never invents observations.
	h := &Handle{spec: spec}
	facts, receipt, result, err := h.LostInterval()
	if err != nil {
		t.Fatalf("LostInterval: %v", err)
	}
	if result.Truncated || result.Entries != 0 {
		t.Fatalf("empty-prefix replay wrong: %+v", result)
	}
	if !facts.Unknown || !receipt.Unknown {
		t.Fatalf("corrupt-channel interval sealed known: %+v", receipt)
	}
}

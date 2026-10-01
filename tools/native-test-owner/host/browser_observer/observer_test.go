// K07 browser-observer tests: admission, attestation, the interval
// through disposal, gaps, durable corrections, interruption, loss, and
// the no-UI proof gate. Local doubles only: no browsers, processes,
// networks, or live runtimes.
package browser_observer

import (
	"errors"
	"strings"
	"testing"
)

var (
	testOwner  = Owner{PID: 4242, StartToken: "start-token-1"}
	otherOwner = Owner{PID: 4243, StartToken: "start-token-2"}
)

func testObserver(t *testing.T) *Observer {
	t.Helper()
	o, err := New([]Grant{
		{Owner: testOwner, Scope: "host-ui", Channels: []string{"window", "icon"}},
		{Owner: testOwner, Scope: "host-ui-full", Channels: []string{"window", "icon", "notification", "focus"}},
	}, DefaultLimits())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := o.OpenScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	return o
}

func testLaunch(t *testing.T, o *Observer) string {
	t.Helper()
	if _, err := o.AdmitLaunch(testOwner, "host-ui", "launch-1"); err != nil {
		t.Fatalf("AdmitLaunch: %v", err)
	}
	token, err := o.TokenForTest(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("TokenForTest: %v", err)
	}
	return token
}

func requireObsErr(t *testing.T, err error, layer Layer, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want [%s:%s], got nil", layer, code)
	}
	var oerr *ObserverError
	if !errors.As(err, &oerr) {
		t.Fatalf("want *ObserverError, got %T (%v)", err, err)
	}
	if oerr.Layer != layer || oerr.Code != code {
		t.Fatalf("want [%s:%s], got [%s:%s] (%v)", layer, code, oerr.Layer, oerr.Code, err)
	}
	if !strings.HasPrefix(err.Error(), "["+string(layer)+":"+code+"]") {
		t.Fatalf("error lacks layer prefix: %v", err)
	}
}

func observeFull(t *testing.T, o *Observer, token, launch string) {
	t.Helper()
	for _, channel := range []string{"window", "icon"} {
		if _, err := o.Observe(testOwner, "host-ui", launch, token, channel, StateAbsent); err != nil {
			t.Fatalf("Observe %s: %v", channel, err)
		}
	}
}

func TestScopeReceiptBindsOwnedGrant(t *testing.T) {
	o, err := New([]Grant{{Owner: testOwner, Scope: "host-ui", Channels: []string{"window", "icon"}}}, DefaultLimits())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	receipt, err := o.OpenScope(testOwner, "host-ui")
	if err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	if receipt.Scope != "host-ui" || !receipt.Owner.same(testOwner) || receipt.Launches != 0 {
		t.Fatalf("bad receipt: %+v", receipt)
	}
	if !strings.HasPrefix(receipt.HandleDigest, "sha256:") || len(receipt.HandleDigest) != len("sha256:")+64 {
		t.Fatalf("receipt carries no digest-only handle: %q", receipt.HandleDigest)
	}
	again, err := o.OpenScope(testOwner, "host-ui")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if again.HandleDigest != receipt.HandleDigest {
		t.Fatal("reopen changed the handle digest")
	}
	for _, foreign := range []string{"host-ui-2", "host", "", "host-ui/full"} {
		_, err := o.OpenScope(testOwner, foreign)
		requireObsErr(t, err, LayerObserver, CodeUnknownScope)
	}
	_, err = o.OpenScope(otherOwner, "host-ui")
	requireObsErr(t, err, LayerObserver, CodeUnknownScope)
	// A second identity granted the same scope name cannot join it.
	shared, err := New([]Grant{
		{Owner: testOwner, Scope: "host-ui", Channels: []string{"window"}},
		{Owner: otherOwner, Scope: "host-ui", Channels: []string{"window"}},
	}, DefaultLimits())
	if err != nil {
		t.Fatalf("New shared: %v", err)
	}
	if _, err := shared.OpenScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	_, err = shared.OpenScope(otherOwner, "host-ui")
	requireObsErr(t, err, LayerObserver, CodeWrongOwner)
}

func TestMissingObserverBlocksLaunch(t *testing.T) {
	o, err := New([]Grant{{Owner: testOwner, Scope: "host-ui", Channels: []string{"window"}}}, DefaultLimits())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Never opened, ungranted, and foreign all refuse identically.
	_, err = o.AdmitLaunch(testOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	_, err = o.AdmitLaunch(testOwner, "elsewhere", "launch-1")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	if _, err := o.OpenScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	_, err = o.AdmitLaunch(otherOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	// A live binding attests.
	attestation, err := o.AdmitLaunch(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("AdmitLaunch: %v", err)
	}
	if attestation.LaunchID != "launch-1" || attestation.Tick != 0 {
		t.Fatalf("bad attestation: %+v", attestation)
	}
	for _, digest := range []string{attestation.HandleDigest, attestation.ObserverDigest, attestation.ScopeDigest} {
		if !strings.HasPrefix(digest, "sha256:") {
			t.Fatalf("attestation carries no digest: %q", digest)
		}
	}
	joined, err := o.AdmitLaunch(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("re-admit: %v", err)
	}
	if joined.HandleDigest != attestation.HandleDigest {
		t.Fatal("re-admit changed the handle digest")
	}
}

func TestForbiddenProofsRejected(t *testing.T) {
	if len(ForbiddenProofKinds) != 4 {
		t.Fatalf("want 4 forbidden kinds, got %d", len(ForbiddenProofKinds))
	}
	for _, kind := range ForbiddenProofKinds {
		_, err := AssertNoUiProof(ProofClaim{Kind: kind, Detail: "seed"})
		requireObsErr(t, err, LayerAdmission, CodeForbiddenProof)
	}
	for _, kind := range []string{"driver-receipt", "screenshot-hash", "", "observer-attestation-x"} {
		_, err := AssertNoUiProof(ProofClaim{Kind: kind, Detail: "x"})
		requireObsErr(t, err, LayerAdmission, CodeForbiddenProof)
	}
	// A forged attestation digest never verifies.
	o := testObserver(t)
	token := testLaunch(t, o)
	observeFull(t, o, token, "launch-1")
	if _, err := o.EndTick(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	observeFull(t, o, token, "launch-1")
	interval, err := o.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	forged := interval
	forged.Digest = "sha256:" + strings.Repeat("0", 64)
	_, err = AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: forged})
	requireObsErr(t, err, LayerAdmission, CodeForbiddenProof)
	// An interval with an open tick never proves.
	_, err = AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: interval})
	requireObsErr(t, err, LayerAdmission, CodeForbiddenProof)
}

func TestObserverAttestationJudgesUiEffects(t *testing.T) {
	// Clean: zero seen effects over a fully observed interval proves no-UI.
	o := testObserver(t)
	token := testLaunch(t, o)
	observeFull(t, o, token, "launch-1")
	if _, err := o.EndTick(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	observeFull(t, o, token, "launch-1")
	if _, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	interval, err := o.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if !interval.Disposed || interval.Unknown {
		t.Fatalf("bad interval: %+v", interval)
	}
	verdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: interval})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if verdict.UISeen || verdict.Unknown {
		t.Fatalf("clean interval misjudged: %+v", verdict)
	}
	// Dirty: the observer attests UI effects from its own observations.
	dirty := testObserver(t)
	dirtyToken := testLaunch(t, dirty)
	if _, err := dirty.Observe(testOwner, "host-ui", "launch-1", dirtyToken, "window", StateSeen); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if _, err := dirty.Observe(testOwner, "host-ui", "launch-1", dirtyToken, "icon", StateAbsent); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if _, err := dirty.SealDisposal(testOwner, "host-ui", "launch-1", dirtyToken); err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	dirtyInterval, err := dirty.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	dirtyVerdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: dirtyInterval})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if !dirtyVerdict.UISeen || dirtyVerdict.Unknown {
		t.Fatalf("dirty interval misjudged: %+v", dirtyVerdict)
	}
}

func TestLaunchTokensOpaqueAndVerified(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	for _, forged := range []string{"", "lnch-deadbeef", token[:len(token)-1] + "x"} {
		_, err := o.Observe(testOwner, "host-ui", "launch-1", forged, "window", StateAbsent)
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
		_, err = o.EndTick(testOwner, "host-ui", "launch-1", forged)
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
		err = o.NoteDriverDeath(testOwner, "host-ui", "launch-1", forged)
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
		_, err = o.SealDisposal(testOwner, "host-ui", "launch-1", forged)
		requireObsErr(t, err, LayerAdmission, CodeForgedToken)
	}
	_, err := o.Observe(testOwner, "host-ui", "launch-9", token, "window", StateAbsent)
	requireObsErr(t, err, LayerAdmission, CodeUnknownLaunch)
	if _, err := o.AdmitLaunch(testOwner, "host-ui", "launch-2"); err != nil {
		t.Fatalf("AdmitLaunch: %v", err)
	}
	token2, err := o.TokenForTest(testOwner, "host-ui", "launch-2")
	if err != nil {
		t.Fatalf("TokenForTest: %v", err)
	}
	if token2 == token {
		t.Fatal("launch tokens collide")
	}
	_, err = o.Observe(testOwner, "host-ui", "launch-2", token, "window", StateAbsent)
	requireObsErr(t, err, LayerAdmission, CodeForgedToken)
}

func TestIntervalThroughDisposal(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	observeFull(t, o, token, "launch-1")
	sealed, err := o.EndTick(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	if !sealed.Closed || sealed.Tick != 0 {
		t.Fatalf("bad sealed tick: %+v", sealed)
	}
	// The driver dies mid-interval: observation continues through late
	// host effects, and the dead driver never seals on its own.
	if err := o.NoteDriverDeath(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("NoteDriverDeath: %v", err)
	}
	if _, err := o.Observe(testOwner, "host-ui", "launch-1", token, "window", StateSeen); err != nil {
		t.Fatalf("Observe late: %v", err)
	}
	if _, err := o.Observe(testOwner, "host-ui", "launch-1", token, "icon", StateAbsent); err != nil {
		t.Fatalf("Observe late: %v", err)
	}
	_, err = o.DisposalReceipt(testOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerInterval, CodeDisposalOpen)
	receipt, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if receipt.Ticks != 2 || !receipt.DriverDeathNoted || receipt.Unknown {
		t.Fatalf("bad receipt: %+v", receipt)
	}
	again, err := o.DisposalReceipt(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("DisposalReceipt: %v", err)
	}
	if again != receipt {
		t.Fatal("disposal receipt is not durable")
	}
	_, err = o.Observe(testOwner, "host-ui", "launch-1", token, "window", StateAbsent)
	requireObsErr(t, err, LayerAdmission, CodeLaunchClosed)
	_, err = o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	requireObsErr(t, err, LayerInterval, CodeAlreadyGone)
	_, err = o.AdmitLaunch(testOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerAdmission, CodeLaunchClosed)
	after, err := o.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if !after.Disposed || len(after.Ticks) != 2 {
		t.Fatalf("post-disposal facts lost: %+v", after)
	}
}

func TestGapAndCorrectionKeys(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	if _, err := o.Observe(testOwner, "host-ui", "launch-1", token, "window", StateAbsent); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	tick0, err := o.EndTick(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	found := false
	for _, entry := range tick0.Channels {
		if entry.Channel == "icon" {
			found = true
			if entry.State != StateGap || entry.Origin != "gap" || entry.Corrected {
				t.Fatalf("silence is not an explicit gap: %+v", entry)
			}
		}
	}
	if !found {
		t.Fatal("gap channel missing from sealed tick")
	}
	observeFull(t, o, token, "launch-1")
	_, err = o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	requireObsErr(t, err, LayerInterval, CodeGapOpen)
	// The gap corrects exactly once; replay and rewrite refuse.
	first, err := o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 0, "icon", StateAbsent)
	if err != nil {
		t.Fatalf("PublishCorrection: %v", err)
	}
	if first.CorrectionID != "c1" || first.Before != StateGap || first.After != StateAbsent {
		t.Fatalf("bad correction: %+v", first)
	}
	_, err = o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 0, "icon", StateSeen)
	requireObsErr(t, err, LayerInterval, CodeCorrectionCold)
	_, err = o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 0, "window", StateSeen)
	requireObsErr(t, err, LayerInterval, CodeCorrectionCold)
	_, err = o.MarkUnknown(testOwner, "host-ui", "launch-1", token, 0, "window")
	requireObsErr(t, err, LayerInterval, CodeCorrectionCold)
	_, err = o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 9, "icon", StateAbsent)
	requireObsErr(t, err, LayerInterval, CodeCorrectionMiss)
	_, err = o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 1, "icon", StateAbsent)
	requireObsErr(t, err, LayerInterval, CodeCorrectionMiss)
	receipt, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if receipt.Unknown || receipt.Corrections != 1 {
		t.Fatalf("bad sealed receipt: %+v", receipt)
	}
	log, err := o.CorrectionLog(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("CorrectionLog: %v", err)
	}
	if len(log) != 1 || log[0].CorrectionID != "c1" {
		t.Fatalf("correction journal not durable: %+v", log)
	}
}

func TestInterruptionAndLoss(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	if _, err := o.Observe(testOwner, "host-ui", "launch-1", token, "window", StateAbsent); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if err := o.InterruptScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("InterruptScope: %v", err)
	}
	_, err := o.AdmitLaunch(testOwner, "host-ui", "launch-2")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	_, err = o.Observe(testOwner, "host-ui", "launch-1", token, "icon", StateAbsent)
	requireObsErr(t, err, LayerObserver, CodeInterrupted)
	_, err = o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	requireObsErr(t, err, LayerObserver, CodeInterrupted)
	during, err := o.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if !during.Unknown {
		t.Fatal("interrupted interval is not unknown")
	}
	if err := o.ResumeScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("ResumeScope: %v", err)
	}
	if err := o.ResumeScope(testOwner, "host-ui"); err == nil {
		t.Fatal("double resume succeeds")
	} else {
		requireObsErr(t, err, LayerObserver, CodeInterrupted)
	}
	// The interrupted entry corrects durably; the seal runs clean.
	fix, err := o.PublishCorrection(testOwner, "host-ui", "launch-1", token, 0, "icon", StateAbsent)
	if err != nil {
		t.Fatalf("PublishCorrection: %v", err)
	}
	if fix.Before != StateUnknown {
		t.Fatalf("interruption did not mark unknown: %+v", fix)
	}
	observeFull(t, o, token, "launch-1")
	receipt, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if receipt.Unknown {
		t.Fatalf("corrected interval still unknown: %+v", receipt)
	}

	// Loss: the interval seals unknown and new launches block.
	lost := testObserver(t)
	lostToken := testLaunch(t, lost)
	observeFull(t, lost, lostToken, "launch-1")
	if _, err := lost.EndTick(testOwner, "host-ui", "launch-1", lostToken); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	if err := lost.LoseObserver(testOwner, "host-ui"); err != nil {
		t.Fatalf("LoseObserver: %v", err)
	}
	_, err = lost.AdmitLaunch(testOwner, "host-ui", "launch-2")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	_, err = lost.Observe(testOwner, "host-ui", "launch-1", lostToken, "window", StateAbsent)
	requireObsErr(t, err, LayerObserver, CodeObserverLost)
	if err := lost.ResumeScope(testOwner, "host-ui"); err == nil {
		t.Fatal("lost observer resumes")
	} else {
		requireObsErr(t, err, LayerObserver, CodeObserverLost)
	}
	lostReceipt, err := lost.SealDisposal(testOwner, "host-ui", "launch-1", lostToken)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if !lostReceipt.Unknown {
		t.Fatalf("lost interval sealed known: %+v", lostReceipt)
	}
}

func TestLossWithFullyObservedTicksSealsUnknown(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	// Every tick fully observed, but the observer dies before the seal:
	// late host effects between the last observation and disposal are
	// unobserved, so the interval is unknown.
	observeFull(t, o, token, "launch-1")
	if _, err := o.EndTick(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("EndTick: %v", err)
	}
	observeFull(t, o, token, "launch-1")
	if err := o.LoseObserver(testOwner, "host-ui"); err != nil {
		t.Fatalf("LoseObserver: %v", err)
	}
	during, err := o.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if !during.Unknown {
		t.Fatal("lost interval with fully observed ticks is not unknown")
	}
	receipt, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if !receipt.Unknown {
		t.Fatalf("lost interval sealed known: %+v", receipt)
	}
	verdict, err := AssertNoUiProof(ProofClaim{Kind: "observer-attestation", Interval: during})
	if err != nil {
		t.Fatalf("AssertNoUiProof: %v", err)
	}
	if !verdict.Unknown {
		t.Fatalf("lost interval judged known: %+v", verdict)
	}
}

func TestUnknownScopeHandling(t *testing.T) {
	o := testObserver(t)
	token := testLaunch(t, o)
	for _, bad := range []string{"", "cursor", "window ", "WINDOW", "gpu-usage"} {
		_, err := o.Observe(testOwner, "host-ui", "launch-1", token, bad, StateAbsent)
		requireObsErr(t, err, LayerInterval, CodeUnknownEffect)
	}
	for _, foreign := range []string{"notification", "focus"} {
		_, err := o.Observe(testOwner, "host-ui", "launch-1", token, foreign, StateAbsent)
		requireObsErr(t, err, LayerInterval, CodeOutOfScope)
	}
	if _, err := o.OpenScope(testOwner, "host-ui-full"); err != nil {
		t.Fatalf("OpenScope full: %v", err)
	}
	if _, err := o.AdmitLaunch(testOwner, "host-ui-full", "launch-f"); err != nil {
		t.Fatalf("AdmitLaunch full: %v", err)
	}
	tokenF, err := o.TokenForTest(testOwner, "host-ui-full", "launch-f")
	if err != nil {
		t.Fatalf("TokenForTest: %v", err)
	}
	for _, channel := range EffectChannels {
		if _, err := o.Observe(testOwner, "host-ui-full", "launch-f", tokenF, channel, StateAbsent); err != nil {
			t.Fatalf("Observe %s: %v", channel, err)
		}
	}
	for _, bad := range []string{StateGap, StateUnknown, "", "maybe"} {
		_, err := o.Observe(testOwner, "host-ui", "launch-1", token, "window", bad)
		requireObsErr(t, err, LayerInterval, CodeUnknownEffect)
	}
	_, err = o.Observe(otherOwner, "host-ui", "launch-1", token, "window", StateAbsent)
	requireObsErr(t, err, LayerObserver, CodeWrongOwner)
	_, err = o.IntervalFacts(otherOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerObserver, CodeWrongOwner)
	if err := o.CloseScope(testOwner, "host-ui"); err == nil {
		t.Fatal("close with a live launch succeeds")
	} else {
		requireObsErr(t, err, LayerInterval, CodeDisposalOpen)
	}
	observeFull(t, o, token, "launch-1")
	if _, err := o.SealDisposal(testOwner, "host-ui", "launch-1", token); err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if err := o.CloseScope(testOwner, "host-ui"); err != nil {
		t.Fatalf("CloseScope: %v", err)
	}
	_, err = o.AdmitLaunch(testOwner, "host-ui", "launch-9")
	requireObsErr(t, err, LayerAdmission, CodeNoObserver)
	_, err = o.IntervalFacts(testOwner, "host-ui", "launch-1")
	requireObsErr(t, err, LayerObserver, CodeScopeClosed)
}

func TestEveryFailureNamesItsLayer(t *testing.T) {
	if len(Codes) != 18 {
		t.Fatalf("want 18 closed codes, got %d", len(Codes))
	}
	seen := make(map[string]bool, len(Codes))
	for _, code := range Codes {
		if seen[code] {
			t.Fatalf("duplicate code: %s", code)
		}
		seen[code] = true
		layer, ok := LayerOfCode(code)
		if !ok {
			t.Fatalf("code maps to no layer: %s", code)
		}
		err := &ObserverError{Layer: layer, Code: code, Err: ErrDenied}
		if !strings.HasPrefix(err.Error(), "["+string(layer)+":"+code+"]") {
			t.Fatalf("error lacks layer prefix: %v", err)
		}
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("Unwrap lost the cause: %v", err)
		}
	}
	if _, ok := LayerOfCode("no-such-code"); ok {
		t.Fatal("unknown code maps to a layer")
	}
	if _, err := New(nil, DefaultLimits()); err == nil {
		t.Fatal("empty grants construct")
	}
	tiny, err := New(
		[]Grant{{Owner: testOwner, Scope: "s", Channels: []string{"window"}}},
		Limits{MaxScopes: 1, MaxLaunches: 1, MaxTicksPerLaunch: 1, MaxCorrectionsPerLaunch: 1},
	)
	if err != nil {
		t.Fatalf("New tiny: %v", err)
	}
	if _, err := tiny.OpenScope(testOwner, "s"); err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	if _, err := tiny.AdmitLaunch(testOwner, "s", "l"); err != nil {
		t.Fatalf("AdmitLaunch: %v", err)
	}
	_, err = tiny.AdmitLaunch(testOwner, "s", "l2")
	requireObsErr(t, err, LayerObserver, CodeCapacity)
	tok, err := tiny.TokenForTest(testOwner, "s", "l")
	if err != nil {
		t.Fatalf("TokenForTest: %v", err)
	}
	if _, err := tiny.Observe(testOwner, "s", "l", tok, "window", StateAbsent); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	_, err = tiny.EndTick(testOwner, "s", "l", tok)
	requireObsErr(t, err, LayerObserver, CodeCapacity)
}

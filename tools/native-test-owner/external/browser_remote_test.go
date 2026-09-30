// Executed K16 remote-authority tests. The worker verified this behavior with
// a throwaway probe and removed it; the integrator keeps the maintained
// coverage here: server preservation, owned connections/contexts, worker
// death, the request/confirm/seal lifecycle, lost confirmations, the
// witness seam, and the cleanup judge.

package external

import (
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func remoteOwner() journal.Owner { return journal.Owner{PID: 4242, StartToken: "start-token-1"} }

func remoteAuthority(t *testing.T) *RemoteAuthority {
	t.Helper()
	a, err := NewRemoteAuthority([]RemoteServerGrant{{Owner: remoteOwner(), Server: "fx-server"}})
	if err != nil {
		t.Fatalf("NewRemoteAuthority: %v", err)
	}
	return a
}

func requireRemoteErr(t *testing.T, err error, layer BrowserRemoteLayer, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want [%s:%s], got nil", layer, code)
	}
	lerr, ok := err.(*BrowserRemoteError)
	if !ok {
		t.Fatalf("want *BrowserRemoteError, got %T (%v)", err, err)
	}
	if lerr.Layer != layer || lerr.Code != code {
		t.Fatalf("want [%s:%s], got [%s:%s] (%v)", layer, code, lerr.Layer, lerr.Code, err)
	}
}

func TestRemoteServerPreserved(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	before, err := a.ServerFacts("fx-server")
	if err != nil {
		t.Fatalf("ServerFacts: %v", err)
	}
	if !before.Live {
		t.Fatal("server not live")
	}
	// A case close-server attempt rejects before any effect.
	if err := a.CloseServer(self, "fx-server"); err == nil {
		t.Fatal("close-server succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerRemote, BrowserRemoteCodeForbiddenServerClose)
	}
	after, err := a.ServerFacts("fx-server")
	if err != nil {
		t.Fatalf("ServerFacts: %v", err)
	}
	if !after.Live || after.Digest != before.Digest || len(after.Connections) != 0 {
		t.Fatalf("server touched: %+v", after)
	}
	if err := a.CloseServer(self, "ghost"); err == nil {
		t.Fatal("close-server on foreign server succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer)
	}
}

func TestRemoteConnectionOwnedSeparately(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	if _, _, err := a.DeclareConnection(self, "ghost", "c1"); err == nil {
		t.Fatal("foreign server declares")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerService, BrowserRemoteCodeUnknownServer)
	}
	facts, tok, err := a.DeclareConnection(self, "fx-server", "c1")
	if err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	if facts.TokenDigest == "" || facts.TokenDigest == tok {
		t.Fatalf("facts leak or lack digest: %+v", facts)
	}
	// Identical re-declare joins with the same token.
	_, tok2, err := a.DeclareConnection(self, "fx-server", "c1")
	if err != nil {
		t.Fatalf("re-declare: %v", err)
	}
	if tok2 != tok {
		t.Fatal("re-declare minted a new token")
	}
	foreign := journal.Owner{PID: 9999, StartToken: "other"}
	if _, _, err := a.DeclareConnection(foreign, "fx-server", "c1"); err == nil {
		t.Fatal("foreign owner joins connection")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerService, BrowserRemoteCodeWrongOwner)
	}
}

func TestRemoteWorkerDeathGatesOpens(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	if _, _, err := a.DeclareConnection(self, "fx-server", "c1"); err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	// Connection token is verified before death is noted.
	if err := a.NoteWorkerDeath(self, "c1", "forged"); err == nil {
		t.Fatal("forged token notes death")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerRemote, BrowserRemoteCodeForgedToken)
	}
	_, ctok, err := a.DeclareConnection(self, "fx-server", "c3")
	if err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	// Open one context, then note death: opens stop, facts stay readable.
	if _, _, err := a.OpenContext(self, "c3", ctok, "x1"); err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	if err := a.NoteWorkerDeath(self, "c3", ctok); err != nil {
		t.Fatalf("NoteWorkerDeath: %v", err)
	}
	if _, _, err := a.OpenContext(self, "c3", ctok, "x2"); err == nil {
		t.Fatal("open after death succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerRemote, BrowserRemoteCodeWorkerDead)
	}
	facts, err := a.ContextFacts(self, "c3", "x1")
	if err != nil {
		t.Fatalf("ContextFacts: %v", err)
	}
	if !facts.WorkerDead || facts.Closed {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestRemoteRequestConfirmSeal(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	_, ctok, err := a.DeclareConnection(self, "fx-server", "c1")
	if err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	_, xtok, err := a.OpenContext(self, "c1", ctok, "x1")
	if err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	// Confirm without a request refuses.
	if _, _, err := a.ConfirmClose(self, "c1", "x1", xtok, "whatever"); err == nil {
		t.Fatal("confirm without request succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed)
	}
	req, closeTok, err := a.RequestClose(self, "c1", "x1", xtok)
	if err != nil {
		t.Fatalf("RequestClose: %v", err)
	}
	if req.CloseState != RemoteClosePending || req.Closed {
		t.Fatalf("request = %+v", req)
	}
	// Re-request joins with the same close token.
	_, closeTok2, err := a.RequestClose(self, "c1", "x1", xtok)
	if err != nil {
		t.Fatalf("RequestClose re-request: %v", err)
	}
	if closeTok2 != closeTok {
		t.Fatal("re-request minted a new close token")
	}
	// Forged close token refuses.
	if _, _, err := a.ConfirmClose(self, "c1", "x1", xtok, "forged"); err == nil {
		t.Fatal("forged close token confirms")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose)
	}
	wf, wtok, err := a.ConfirmClose(self, "c1", "x1", xtok, closeTok)
	if err != nil {
		t.Fatalf("ConfirmClose: %v", err)
	}
	if !wf.Closed {
		t.Fatalf("witness = %+v", wf)
	}
	// Double confirm refuses; receipt reads the seal.
	if _, _, err := a.ConfirmClose(self, "c1", "x1", xtok, closeTok); err == nil {
		t.Fatal("double confirm succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeAlreadyClosed)
	}
	rc, err := a.CloseReceipt(self, "c1", "x1")
	if err != nil {
		t.Fatalf("CloseReceipt: %v", err)
	}
	if rc.Digest == "" || rc.ContextID != "x1" {
		t.Fatalf("receipt = %+v", rc)
	}
	// Witness verification: unknown and cross-context tokens refuse.
	if _, err := a.VerifyWitness("c1", "x1", "bogus"); err == nil {
		t.Fatal("unknown witness verifies")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose)
	}
	if _, err := a.VerifyWitness("c1", "other", wtok); err == nil {
		t.Fatal("cross-context witness verifies")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeForgedClose)
	}
	got, err := a.VerifyWitness("c1", "x1", wtok)
	if err != nil {
		t.Fatalf("VerifyWitness: %v", err)
	}
	if !got.Closed {
		t.Fatalf("verified = %+v", got)
	}
}

func TestRemoteLostConfirmationUnresolved(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	_, ctok, err := a.DeclareConnection(self, "fx-server", "c1")
	if err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	_, xtok, err := a.OpenContext(self, "c1", ctok, "x1")
	if err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	if err := a.LoseConfirmation(self, "c1", "x1", xtok); err == nil {
		t.Fatal("loss without pending request succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed)
	}
	_, closeTok, err := a.RequestClose(self, "c1", "x1", xtok)
	if err != nil {
		t.Fatalf("RequestClose: %v", err)
	}
	if err := a.LoseConfirmation(self, "c1", "x1", xtok); err != nil {
		t.Fatalf("LoseConfirmation: %v", err)
	}
	facts, err := a.ContextFacts(self, "c1", "x1")
	if err != nil {
		t.Fatalf("ContextFacts: %v", err)
	}
	if facts.CloseState != RemoteCloseUnknown || facts.Closed {
		t.Fatalf("facts = %+v", facts)
	}
	// Seal with the previously valid token, receipt, and re-request all
	// refuse; the loss is terminal.
	if _, _, err := a.ConfirmClose(self, "c1", "x1", xtok, closeTok); err == nil {
		t.Fatal("confirm after loss succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed)
	}
	if _, err := a.CloseReceipt(self, "c1", "x1"); err == nil {
		t.Fatal("receipt after loss succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed)
	}
	if _, _, err := a.RequestClose(self, "c1", "x1", xtok); err == nil {
		t.Fatal("re-request after loss succeeds")
	} else {
		requireRemoteErr(t, err, BrowserRemoteLayerCleanup, BrowserRemoteCodeCloseUnconfirmed)
	}
}

func TestRemoteCleanupJudge(t *testing.T) {
	self := remoteOwner()
	a := remoteAuthority(t)
	_, ctok, err := a.DeclareConnection(self, "fx-server", "c1")
	if err != nil {
		t.Fatalf("DeclareConnection: %v", err)
	}
	_, xtok, err := a.OpenContext(self, "c1", ctok, "x1")
	if err != nil {
		t.Fatalf("OpenContext: %v", err)
	}
	_, closeTok, err := a.RequestClose(self, "c1", "x1", xtok)
	if err != nil {
		t.Fatalf("RequestClose: %v", err)
	}
	if _, _, err := a.ConfirmClose(self, "c1", "x1", xtok, closeTok); err != nil {
		t.Fatalf("ConfirmClose: %v", err)
	}
	rc, err := a.CloseReceipt(self, "c1", "x1")
	if err != nil {
		t.Fatalf("CloseReceipt: %v", err)
	}
	for _, kind := range append(append([]string(nil), ForbiddenCloseKinds...), "obituary", "") {
		if _, err := AssertRemoteCleanup(RemoteCleanupClaim{Kind: kind, Detail: "dead"}); err == nil {
			t.Fatalf("kind %q judges", kind)
		}
	}
	tampered := rc
	tampered.Digest = "sha256:dead"
	if _, err := AssertRemoteCleanup(RemoteCleanupClaim{Kind: "external-close", Receipt: tampered}); err == nil {
		t.Fatal("tampered receipt verifies")
	}
	verdict, err := AssertRemoteCleanup(RemoteCleanupClaim{Kind: "external-close", Receipt: rc})
	if err != nil {
		t.Fatalf("AssertRemoteCleanup: %v", err)
	}
	if !verdict.Witnessed {
		t.Fatalf("verdict = %+v", verdict)
	}
}

// Executed K08 local-authority tests. The worker verified this behavior with
// a throwaway probe and removed it; the integrator keeps the maintained
// coverage here: pinned admission, the observer gate, report/observe/
// containment, externally witnessed reclaim, and the cleanup judge.

package external

import (
	"errors"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func localOwner() journal.Owner { return journal.Owner{PID: 4242, StartToken: "start-token-1"} }

func localAuthority(t *testing.T) *LocalAuthority {
	t.Helper()
	self := localOwner()
	a, err := NewLocalAuthority(
		[]LocalDriverGrant{{Owner: self, Driver: "firefox", Launcher: "firefox-launcher"}},
		[]LocalObserverGrant{{Owner: self, Scope: "host-ui"}},
	)
	if err != nil {
		t.Fatalf("NewLocalAuthority: %v", err)
	}
	if _, err := a.BindObserver(self, "host-ui"); err != nil {
		t.Fatalf("BindObserver: %v", err)
	}
	return a
}

func requireLocalErr(t *testing.T, err error, layer BrowserLocalLayer, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want [%s:%s], got nil", layer, code)
	}
	var lerr *BrowserLocalError
	if !errors.As(err, &lerr) {
		t.Fatalf("want *BrowserLocalError, got %T (%v)", err, err)
	}
	if lerr.Layer != layer || lerr.Code != code {
		t.Fatalf("want [%s:%s], got [%s:%s] (%v)", layer, code, lerr.Layer, lerr.Code, err)
	}
	if !strings.HasPrefix(err.Error(), "["+string(layer)+":"+code+"]") {
		t.Fatalf("error lacks layer prefix: %v", err)
	}
}

func TestLocalAuthorityConstruction(t *testing.T) {
	self := localOwner()
	if _, err := NewLocalAuthority(nil, []LocalObserverGrant{{Owner: self, Scope: "s"}}); err == nil {
		t.Fatal("empty drivers admitted")
	}
	if _, err := NewLocalAuthority(
		[]LocalDriverGrant{{Owner: self, Driver: "d", Launcher: "l"}},
		nil,
	); err == nil {
		t.Fatal("empty observers admitted")
	}
	// Duplicate grants refuse.
	drivers := []LocalDriverGrant{
		{Owner: self, Driver: "d", Launcher: "l"},
		{Owner: self, Driver: "d", Launcher: "l"},
	}
	if _, err := NewLocalAuthority(drivers, []LocalObserverGrant{{Owner: self, Scope: "s"}}); err == nil {
		t.Fatal("duplicate drivers admitted")
	}
	// Foreign scope binding refuses.
	a, err := NewLocalAuthority(
		[]LocalDriverGrant{{Owner: self, Driver: "d", Launcher: "l"}},
		[]LocalObserverGrant{{Owner: self, Scope: "s"}},
	)
	if err != nil {
		t.Fatalf("NewLocalAuthority: %v", err)
	}
	if _, err := a.BindObserver(self, "elsewhere"); err == nil {
		t.Fatal("ungranted scope binds")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerService, BrowserLocalCodeNoObserver)
	}
	foreign := journal.Owner{PID: 9999, StartToken: "other"}
	if _, err := a.BindObserver(foreign, "s"); err == nil {
		t.Fatal("foreign owner binds")
	}
}

func TestLocalObserverGate(t *testing.T) {
	self := localOwner()
	a, err := NewLocalAuthority(
		[]LocalDriverGrant{{Owner: self, Driver: "firefox", Launcher: "firefox-launcher"}},
		[]LocalObserverGrant{{Owner: self, Scope: "host-ui"}},
	)
	if err != nil {
		t.Fatalf("NewLocalAuthority: %v", err)
	}
	// Never-bound scope blocks declaration.
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "p"); err == nil {
		t.Fatal("declaration without binding succeeds")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerService, BrowserLocalCodeNoObserver)
	}
	if _, err := a.BindObserver(self, "host-ui"); err != nil {
		t.Fatalf("BindObserver: %v", err)
	}
	// Lost binding blocks new declarations; facts stay readable.
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "p"); err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	if err := a.LoseObserver(self, "host-ui"); err != nil {
		t.Fatalf("LoseObserver: %v", err)
	}
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l2", "firefox", "firefox-launcher", "p"); err == nil {
		t.Fatal("declaration after loss succeeds")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerService, BrowserLocalCodeNoObserver)
	}
	if _, err := a.Discover(self, "l1"); err != nil {
		t.Fatalf("Discover after loss: %v", err)
	}
}

func TestLocalDeclareLaunch(t *testing.T) {
	self := localOwner()
	a := localAuthority(t)
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l1", "chromium", "chromium-launcher", "p"); err == nil {
		t.Fatal("foreign driver declares")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerService, BrowserLocalCodeUnknownDriver)
	}
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "other-launcher", "p"); err == nil {
		t.Fatal("unpinned launcher declares")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerService, BrowserLocalCodeLauncherUnpinned)
	}
	facts, tok, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	if facts.TokenDigest == "" || strings.Contains(facts.TokenDigest, tok) {
		t.Fatalf("facts leak or lack digest: %+v", facts)
	}
	// Identical re-declare joins with the same token.
	_, tok2, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("identical re-declare: %v", err)
	}
	if tok2 != tok {
		t.Fatal("re-declare minted a new token")
	}
	// Same ID with differing arguments refuses.
	if _, _, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "other"); err == nil {
		t.Fatal("re-declare with new profile joins")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerLaunch, BrowserLocalCodeUnknownLaunch)
	}
}

func TestLocalReportObserveContain(t *testing.T) {
	self := localOwner()
	a := localAuthority(t)
	_, tok, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	facts, err := a.ReportChildren(self, "l1", tok, []string{"c1", "c2"})
	if err != nil {
		t.Fatalf("ReportChildren: %v", err)
	}
	if len(facts.Reported) != 2 {
		t.Fatalf("reported = %+v", facts)
	}
	if _, err := a.ReportChildren(self, "l1", "forged", []string{"c1"}); err == nil {
		t.Fatal("forged token reports")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerLaunch, BrowserLocalCodeForgedToken)
	}
	// External observation is additive and continues after death.
	if _, err := a.ObserveChildren(self, "l1", tok, []string{"c3"}); err != nil {
		t.Fatalf("ObserveChildren: %v", err)
	}
	if err := a.NoteDriverDeath(self, "l1", tok); err != nil {
		t.Fatalf("NoteDriverDeath: %v", err)
	}
	if _, err := a.ReportChildren(self, "l1", tok, []string{"c4"}); err == nil {
		t.Fatal("dead launcher reports")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerLaunch, BrowserLocalCodeDriverDead)
	}
	if _, err := a.ObserveChildren(self, "l1", tok, []string{"c4"}); err != nil {
		t.Fatalf("ObserveChildren after death: %v", err)
	}
	if err := a.Contain(self, "l1", tok, "ghost"); err == nil {
		t.Fatal("unknown child contains")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerReclaim, BrowserLocalCodeUnknownChild)
	}
	for _, c := range []string{"c1", "c2", "c3", "c4"} {
		if err := a.Contain(self, "l1", tok, c); err != nil {
			t.Fatalf("Contain %s: %v", c, err)
		}
	}
}

func TestLocalReclaimNamesOrphans(t *testing.T) {
	self := localOwner()
	a := localAuthority(t)
	_, tok, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	if _, err := a.ReportChildren(self, "l1", tok, []string{"c1", "c2"}); err != nil {
		t.Fatalf("ReportChildren: %v", err)
	}
	if err := a.Contain(self, "l1", tok, "c1"); err != nil {
		t.Fatalf("Contain: %v", err)
	}
	if _, _, err := a.Reclaim(self, "l1", tok); err == nil {
		t.Fatal("reclaim with open orphan succeeds")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerReclaim, BrowserLocalCodeOrphanOpen)
		if !strings.Contains(err.Error(), "c2") {
			t.Fatalf("orphan not named: %v", err)
		}
	}
	if err := a.Contain(self, "l1", tok, "c2"); err != nil {
		t.Fatalf("Contain: %v", err)
	}
	wf, wtok, err := a.Reclaim(self, "l1", tok)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if !wf.Contained || len(wf.Children) != 2 {
		t.Fatalf("witness = %+v", wf)
	}
	if _, _, err := a.Reclaim(self, "l1", tok); err == nil {
		t.Fatal("double reclaim succeeds")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerReclaim, BrowserLocalCodeAlreadyReclaimed)
	}
	// Witness lookup: unknown and cross-launch tokens never verify.
	if _, err := a.VerifyWitness("l1", "bogus"); err == nil {
		t.Fatal("unknown witness verifies")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerReclaim, BrowserLocalCodeForgedWitness)
	}
	if _, err := a.VerifyWitness("other", wtok); err == nil {
		t.Fatal("cross-launch witness verifies")
	} else {
		requireLocalErr(t, err, BrowserLocalLayerReclaim, BrowserLocalCodeForgedWitness)
	}
	got, err := a.VerifyWitness("l1", wtok)
	if err != nil {
		t.Fatalf("VerifyWitness: %v", err)
	}
	if !got.Contained || len(got.Children) != 2 {
		t.Fatalf("verified = %+v", got)
	}
}

func TestLocalDeathMidLaunchReclaimsWithoutOrphans(t *testing.T) {
	self := localOwner()
	a := localAuthority(t)
	_, tok, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	if _, err := a.ReportChildren(self, "l1", tok, []string{"c1"}); err != nil {
		t.Fatalf("ReportChildren: %v", err)
	}
	if err := a.NoteDriverDeath(self, "l1", tok); err != nil {
		t.Fatalf("NoteDriverDeath: %v", err)
	}
	// The authority discovers the death-time child externally and
	// contains everything without the driver.
	if _, err := a.ObserveChildren(self, "l1", tok, []string{"c2"}); err != nil {
		t.Fatalf("ObserveChildren: %v", err)
	}
	for _, c := range []string{"c1", "c2"} {
		if err := a.Contain(self, "l1", tok, c); err != nil {
			t.Fatalf("Contain %s: %v", c, err)
		}
	}
	wf, _, err := a.Reclaim(self, "l1", tok)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if len(wf.Children) != 2 {
		t.Fatalf("witness = %+v", wf)
	}
	rc, err := a.ReclaimReceipt(self, "l1")
	if err != nil {
		t.Fatalf("ReclaimReceipt: %v", err)
	}
	if !rc.DriverDead || len(rc.Children) != 2 {
		t.Fatalf("receipt = %+v", rc)
	}
	verdict, err := AssertLocalCleanup(LocalCleanupClaim{Kind: "external-witness", Receipt: rc})
	if err != nil {
		t.Fatalf("AssertLocalCleanup: %v", err)
	}
	if !verdict.Witnessed {
		t.Fatalf("verdict = %+v", verdict)
	}
}

func TestLocalDriverReceiptNeverWitnesses(t *testing.T) {
	for _, kind := range append(append([]string(nil), ForbiddenWitnessKinds...), "obituary", "") {
		_, err := AssertLocalCleanup(LocalCleanupClaim{Kind: kind, Detail: "dead"})
		if err == nil {
			t.Fatalf("kind %q judges", kind)
		}
		var lerr *BrowserLocalError
		if !errors.As(err, &lerr) || lerr.Code != BrowserLocalCodeForbiddenWitness {
			t.Fatalf("kind %q: %v", kind, err)
		}
	}
	// A tampered receipt digest never verifies.
	self := localOwner()
	a := localAuthority(t)
	_, tok, err := a.DeclareLaunch(self, "host-ui", "l1", "firefox", "firefox-launcher", "prof")
	if err != nil {
		t.Fatalf("DeclareLaunch: %v", err)
	}
	if _, _, err := a.Reclaim(self, "l1", tok); err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	rc, err := a.ReclaimReceipt(self, "l1")
	if err != nil {
		t.Fatalf("ReclaimReceipt: %v", err)
	}
	rc.Digest = "sha256:dead"
	if _, err := AssertLocalCleanup(LocalCleanupClaim{Kind: "external-witness", Receipt: rc}); err == nil {
		t.Fatal("tampered receipt verifies")
	}
}

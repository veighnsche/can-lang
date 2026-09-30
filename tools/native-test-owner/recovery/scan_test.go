package recovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func TestScanClassification(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-scan-proc")
	foreign := spawnSleeper(t, o, "scan-foreign", 43)
	defer func() {
		_ = o.Abort(foreign)
		assertNoStraySleepers(t, "sleep 43")
	}()
	dead := deadPID(t, o, "scan")
	self := testSelf()
	intact := writeTempFile(t, "intact.txt", "intact-target")
	intactID, err := journal.StatPath(intact)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	if !intactID.HasFileID {
		t.Fatalf("StatPath reports no file identity for %q", intact)
	}
	bogus := intactID
	bogus.Dev++
	foreignOwner := journal.Owner{PID: foreign.PID, StartToken: "foreign-token"}
	deadOwner := journal.Owner{PID: dead, StartToken: "dead-token"}

	proof := func(op string, p LivenessProof) map[string]LivenessProof {
		return map[string]LivenessProof{op: p}
	}
	goodProof := func(op string) map[string]LivenessProof {
		return proof(op, LivenessProof{SchemaVersion: "1", OperationID: op, PID: self.PID, StartToken: self.StartToken})
	}

	type scanCase struct {
		name       string
		op         string
		owner      journal.Owner
		path       journal.PathIdentity
		cfg        ScanConfig
		self       journal.Owner
		wantAllow  bool
		wantReason string
	}
	cases := []scanCase{
		{"allow-no-path", "scanop0", self, journal.PathIdentity{}, ScanConfig{}, self, true, ""},
		{"allow-intact-path", "scanop1", self, intactID, ScanConfig{}, self, true, ""},
		{"owner-dead-intact-target", "scanop2", deadOwner, intactID, ScanConfig{}, self, false, ReasonOwnerMismatch},
		{"foreign-process-live", "scanop3", foreignOwner, journal.PathIdentity{}, ScanConfig{}, self, false, ReasonForeignLive},
		{"foreign-live-beats-path", "scanop4", foreignOwner, intactID, ScanConfig{}, self, false, ReasonForeignLive},
		{"replaced-path", "scanop5", self, bogus, ScanConfig{}, self, false, ReasonReplacedPath},
		{"unknown-schema", "scanop6", self, journal.PathIdentity{}, ScanConfig{ObservedSchema: "99"}, self, false, ReasonUnknownSchema},
		{"unknown-schema-beats-owner", "scanop7", self, intactID, ScanConfig{ObservedSchema: "0"}, self, false, ReasonUnknownSchema},
		{"proof-required-missing", "scanop8", self, journal.PathIdentity{}, ScanConfig{Proofs: map[string]LivenessProof{}}, self, false, ReasonIncompleteProof},
		{"proof-unknown-schema", "scanop9", self, journal.PathIdentity{}, ScanConfig{Proofs: proof("scanop9", LivenessProof{SchemaVersion: "99", OperationID: "scanop9", PID: self.PID, StartToken: self.StartToken})}, self, false, ReasonUnknownSchema},
		{"proof-foreign-live", "scanop10", self, journal.PathIdentity{}, ScanConfig{Proofs: proof("scanop10", LivenessProof{SchemaVersion: "1", OperationID: "scanop10", PID: foreign.PID, StartToken: "foreign-token"})}, self, false, ReasonForeignLive},
		{"proof-incomplete-dead", "scanop11", self, journal.PathIdentity{}, ScanConfig{Proofs: proof("scanop11", LivenessProof{SchemaVersion: "1", OperationID: "scanop11", PID: dead, StartToken: "dead-token"})}, self, false, ReasonIncompleteProof},
		{"proof-complete", "scanop12", self, journal.PathIdentity{}, ScanConfig{Proofs: goodProof("scanop12")}, self, true, ""},
		{"stranger-refused", "scanop13", self, intactID, ScanConfig{}, testStranger(), false, ReasonForeignLive},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			j := journal.OpenMemory()
			defer j.Close()
			reserve(t, j, c.op, c.owner, c.path)
			before := pendingStates(j)
			rep := Scan(j, c.self, c.cfg)
			if len(rep.Decisions) != 1 {
				t.Fatalf("decisions = %d, want 1", len(rep.Decisions))
			}
			d := rep.Decisions[0]
			if d.OperationID != c.op {
				t.Fatalf("decision op = %q, want %q", d.OperationID, c.op)
			}
			if d.Allow != c.wantAllow || d.Reason != c.wantReason {
				t.Fatalf("decision = (allow=%v reason=%q), want (allow=%v reason=%q)",
					d.Allow, d.Reason, c.wantAllow, c.wantReason)
			}
			if rep.Allowed(c.op) != c.wantAllow {
				t.Fatalf("Allowed(%q) = %v, want %v", c.op, !c.wantAllow, c.wantAllow)
			}
			if got := rep.ReasonFor(c.op); got != c.wantReason {
				t.Fatalf("ReasonFor(%q) = %q, want %q", c.op, got, c.wantReason)
			}
			if rep.Allowed("ghost-op") {
				t.Fatalf("Allowed reports an unknown operation allowed")
			}
			if got := rep.ReasonFor("ghost-op"); got != ReasonDenied {
				t.Fatalf("ReasonFor(unknown) = %q, want %q", got, ReasonDenied)
			}
			// The scan mutates nothing: no advances, no signals, no
			// deletions.
			if after := pendingStates(j); !sameStates(before, after) {
				t.Fatalf("scan mutated journal states: %v -> %v", before, after)
			}
			if got := readFile(t, intact); got != "intact-target" {
				t.Fatalf("scan touched the target: %q", got)
			}
			if !ProbeAlive(foreign.PID) {
				t.Fatalf("scan killed the foreign sleeper (pid %d)", foreign.PID)
			}
		})
	}
	// Nil journals and reports fail closed without panicking.
	if rep := Scan(nil, self, ScanConfig{}); len(rep.Decisions) != 0 {
		t.Fatalf("nil journal scan reports %d decisions", len(rep.Decisions))
	}
	var nilRep *Report
	if nilRep.Allowed("x") || nilRep.ReasonFor("x") != ReasonDenied {
		t.Fatalf("nil report reads as authority")
	}
}

// TestScanNeverSignalsDeletesOrPasses is the acceptance clause in one
// place: a dead owner with an intact target, scanned by a stranger while
// a foreign process is live, authorizes no signal, no deletion and no
// old-run pass.
func TestScanNeverSignalsDeletesOrPasses(t *testing.T) {
	o := procOwner(t, journal.OpenMemory(), "p11-nosignal-proc")
	foreign := spawnSleeper(t, o, "nosignal-foreign", 45)
	defer func() {
		_ = o.Abort(foreign)
		assertNoStraySleepers(t, "sleep 45")
	}()
	dead := deadPID(t, o, "nosignal")
	target := writeTempFile(t, "target.txt", "old-run-bytes")
	targetID, err := journal.StatPath(target)
	if err != nil {
		t.Fatalf("StatPath: %v", err)
	}
	j := journal.OpenMemory()
	defer j.Close()
	reserve(t, j, "nosignal1", journal.Owner{PID: dead, StartToken: "dead-owner"}, targetID)
	before := pendingStates(j)

	rep := Scan(j, testStranger(), ScanConfig{})
	if len(rep.Decisions) != 1 {
		t.Fatalf("decisions = %d, want 1", len(rep.Decisions))
	}
	d := rep.Decisions[0]
	if d.Allow || d.Reason != ReasonOwnerMismatch {
		t.Fatalf("decision = (allow=%v reason=%q), want (allow=false reason=%q)",
			d.Allow, d.Reason, ReasonOwnerMismatch)
	}
	if !ProbeAlive(foreign.PID) {
		t.Fatalf("scan signaled the foreign sleeper (pid %d)", foreign.PID)
	}
	if got := readFile(t, target); got != "old-run-bytes" {
		t.Fatalf("scan touched the target: %q", got)
	}
	if after := pendingStates(j); !sameStates(before, after) {
		t.Fatalf("scan mutated journal states: %v -> %v", before, after)
	}
}

// TestScanDirSchemaMarker covers old unknown schemas on disk: a journal
// directory carrying a marker this scanner does not understand refuses
// every pending operation, while the current marker (or none) scans
// normally.
func TestScanDirSchemaMarker(t *testing.T) {
	dir := t.TempDir()
	j, err := journal.OpenDir(dir)
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}
	reserve(t, j, "schemamark1", testSelf(), journal.PathIdentity{})
	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	marker := filepath.Join(dir, schemaMarkerName)

	writeMarker := func(body string) {
		t.Helper()
		if err := os.WriteFile(marker, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile marker: %v", err)
		}
	}
	scan := func() *Report {
		t.Helper()
		rep, err := ScanDir(dir, testSelf(), ScanConfig{})
		if err != nil {
			t.Fatalf("ScanDir: %v", err)
		}
		return rep
	}

	writeMarker("99")
	if rep := scan(); len(rep.Decisions) != 1 || rep.Decisions[0].Allow || rep.Decisions[0].Reason != ReasonUnknownSchema {
		t.Fatalf("schema 99 decisions = %+v, want one unknown-schema refusal", rep.Decisions)
	}
	writeMarker("1\n")
	if rep := scan(); len(rep.Decisions) != 1 || !rep.Decisions[0].Allow {
		t.Fatalf("schema 1 decisions = %+v, want one allow", rep.Decisions)
	}
	writeMarker("")
	if rep := scan(); len(rep.Decisions) != 1 || rep.Decisions[0].Allow {
		t.Fatalf("empty marker decisions = %+v, want a refusal", rep.Decisions)
	}
	writeMarker("1" + strings.Repeat("0", maxSchemaMarkerBytes))
	if rep := scan(); len(rep.Decisions) != 1 || rep.Decisions[0].Allow || rep.Decisions[0].Reason != ReasonUnknownSchema {
		t.Fatalf("oversize marker decisions = %+v, want an unknown-schema refusal", rep.Decisions)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatalf("Remove marker: %v", err)
	}
	if rep := scan(); len(rep.Decisions) != 1 || !rep.Decisions[0].Allow {
		t.Fatalf("absent marker decisions = %+v, want one allow", rep.Decisions)
	}
	// The scans advanced nothing.
	j2, err := journal.OpenDir(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer j2.Close()
	r, ok := j2.Lookup("schemamark1")
	if !ok || r.State != journal.StateReserved {
		t.Fatalf("scan advanced the journal: %+v", r)
	}
	if _, err := ScanDir("", testSelf(), ScanConfig{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty dir: got %v, want ErrInvalid", err)
	}
}

// TestScanReplacedDirectory covers replaced directories deterministically:
// a vanished target and a symlink-swapped target both refuse deletion
// authority with the replaced-path reason.
func TestScanReplacedDirectory(t *testing.T) {
	self := testSelf()

	t.Run("vanished", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "target")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		id, err := journal.StatPath(dir)
		if err != nil {
			t.Fatalf("StatPath: %v", err)
		}
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "replaced1", self, id)
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
		rep := Scan(j, self, ScanConfig{})
		if len(rep.Decisions) != 1 || rep.Decisions[0].Allow || rep.Decisions[0].Reason != ReasonReplacedPath {
			t.Fatalf("decisions = %+v, want one replaced-path refusal", rep.Decisions)
		}
	})

	t.Run("symlink-swap", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "target")
		other := filepath.Join(root, "other")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		if err := os.Mkdir(other, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		id, err := journal.StatPath(dir)
		if err != nil {
			t.Fatalf("StatPath: %v", err)
		}
		j := journal.OpenMemory()
		defer j.Close()
		reserve(t, j, "replaced2", self, id)
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
		if err := os.Symlink(other, dir); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		rep := Scan(j, self, ScanConfig{})
		if len(rep.Decisions) != 1 || rep.Decisions[0].Allow || rep.Decisions[0].Reason != ReasonReplacedPath {
			t.Fatalf("decisions = %+v, want one replaced-path refusal", rep.Decisions)
		}
	})
}

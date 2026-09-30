package f1

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

const (
	testShell = "/bin/sh"
	testCat   = "/bin/cat"
	testSleep = "/bin/sleep"
)

// requireBin skips when a bounded local child binary is unavailable.
func requireBin(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			t.Skipf("bounded child binary %s unavailable: %v", p, err)
		}
	}
}

func testOwner(t *testing.T) *process.Owner {
	t.Helper()
	o, err := process.New(journal.OpenMemory(), os.Getpid(), "f1-test-owner")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		for _, e := range o.Tree() {
			_ = o.Abort(e.Child)
		}
		if n := o.Live(); n != 0 {
			t.Errorf("Live() = %d, want 0 (stray processes)", n)
		}
	})
	return o
}

func testLease(t *testing.T, name string) *process.Lease {
	t.Helper()
	l, err := process.AcquireLease(filepath.Join(t.TempDir(), "lease-"+name))
	if err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func shArgs(script string, extras ...string) []string {
	return append([]string{"-c", script, "sh"}, extras...)
}

// bigSnapshotEnv renders a ~512KB snapshot from 16 entries of 32KB
// each: every entry fits MaxEnvEntry and the total fits MaxEnvBytes,
// while no undrained pipe buffer can hold it, forcing EPIPE delivery
// failure against a non-reader.
func bigSnapshotEnv() map[string]string {
	env := make(map[string]string, 16)
	for i := 0; i < 16; i++ {
		env["F1_BIG_"+string(rune('A'+i))] = strings.Repeat("b", 32<<10)
	}
	return env
}

func mustAbort(o *process.Owner, id process.Identity) { _ = o.Abort(id) }

// The positive baseline: a child that drains fd 3 and emits the
// content-bound ack flips each fact independently across stages, and
// the fixture ack matches Owner.ExpectedAck byte for byte.
func TestPositiveDrainAckEOFSeparated(t *testing.T) {
	requireBin(t, testShell, testCat)
	o := testOwner(t)
	l := testLease(t, "positive")
	opID := "f1positive1"
	env := map[string]string{"F1_ALPHA": "1", "F1_BETA": "two"}
	dir := t.TempDir()
	drained := filepath.Join(dir, "fd3.out")
	script := testCat + ` <&3 > "$1"; printf '%b' '` + process.OctalEscape(MustAckFrameFor(opID, env)) + `'`
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	// The fixture renders exactly what the owner expects: a child that
	// read fd 3 mints this frame and nothing else.
	if want, err := o.ExpectedAck(id); err != nil || !bytes.Equal(want, MustAckFrameFor(opID, env)) {
		t.Fatalf("ExpectedAck = %x err=%v, want fixture frame", want, err)
	}
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus: %v", err)
	}
	f, _ := o.Facts(id)
	if !f.Accepted || f.Malformed || !f.EOF {
		t.Fatalf("after status: facts=%+v, want accepted+EOF, no malformed", f)
	}
	if f.Reaped || f.ChildExit != nil {
		t.Fatalf("after status: facts=%+v, want child-exit not yet recorded", f)
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.Code != 0 || exit.Signaled {
		t.Fatalf("exit = %+v, want code 0", exit)
	}
	f, _ = o.Facts(id)
	if !f.Reaped || f.ChildExit == nil || f.ChildExit.Code != 0 {
		t.Fatalf("after wait: facts=%+v, want reaped exit 0", f)
	}
	wantSnap := MustRenderSnapshot(env)
	if got, err := os.ReadFile(drained); err != nil || !bytes.Equal(got, wantSnap) {
		t.Fatalf("child drained %q err=%v, want %q", got, err, wantSnap)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if !rep.Clean {
		t.Fatalf("Clean=false reason=%q facts=%+v", rep.Reason, rep.Facts)
	}
	if rep.Facts.OfferedBytes != len(wantSnap) || !rep.Facts.Offered || !rep.Facts.WriterDone {
		t.Fatalf("facts=%+v, want full offer recorded", rep.Facts)
	}
}

// Offered bytes and accepted bytes stay separate in both directions: a
// full drain with a foreign ack delivers every byte yet accepts none,
// and an instant-exiting non-reader accepts nothing however much the
// writer lands.
func TestOfferedAcceptedBytesSeparate(t *testing.T) {
	requireBin(t, testShell, testCat)
	t.Run("drained-but-foreign", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "foreign")
		opID := "f1foreign1"
		env := map[string]string{"F1_K": "v"}
		dir := t.TempDir()
		drained := filepath.Join(dir, "fd3.out")
		// Bound to another operation: the bytes are right, the binding
		// is wrong.
		foreign := MustAckFrameFor("someone-else", env)
		script := testCat + ` <&3 > "$1"; printf '%b' '` + process.OctalEscape(foreign) + `'`
		id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want foreign-ack error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true for foreign ack")
		}
		wantSnap := MustRenderSnapshot(env)
		if got, _ := os.ReadFile(drained); !bytes.Equal(got, wantSnap) {
			t.Fatalf("child drained %q, want %q", got, wantSnap)
		}
		f := rep.Facts
		if !f.Offered || f.OfferedBytes != len(wantSnap) {
			t.Fatalf("facts=%+v, want every byte offered", f)
		}
		if f.Accepted || !f.Malformed {
			t.Fatalf("facts=%+v, want accepted=false malformed=true", f)
		}
	})
	t.Run("instant-non-reader", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "instant")
		env := map[string]string{"F1_K": "v"}
		id, err := o.Spawn("f1instant1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
			t.Fatalf("CollectStatus: want error, got nil")
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		rep, err := o.Release(id)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true for non-reader")
		}
		f := rep.Facts
		if f.Accepted || !f.EOF || !f.WriterDone {
			t.Fatalf("facts=%+v, want accepted=false EOF=true writer-joined", f)
		}
		if f.OfferedBytes < 0 || f.OfferedBytes > len(MustRenderSnapshot(env)) {
			t.Fatalf("facts=%+v, want bounded offered bytes", f)
		}
		if f.Offered && f.WriterErr != "" {
			t.Fatalf("facts=%+v, want no writer error on a full offer", f)
		}
	})
}

// EOF is observed separately from child exit: a child that acks then
// withholds EOF is accepted-but-open while alive; killing it records
// the exit while EOF stays false until the collector observes the pipe
// close afterwards.
func TestEOFSeparateFromChildExit(t *testing.T) {
	requireBin(t, testShell, testSleep)
	o := testOwner(t)
	l := testLease(t, "withheld")
	opID := "f1withheld1"
	env := map[string]string{"F1_K": "v"}
	script := `printf '%b' '` + process.OctalEscape(MustAckFrameFor(opID, env)) + `'; exec ` + testSleep + ` 30`
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if err := o.CollectStatus(id, 5*time.Second, 300*time.Millisecond); err == nil {
		t.Fatalf("CollectStatus: want EOF timeout, got nil")
	}
	f, _ := o.Facts(id)
	if !f.Accepted || f.EOF {
		t.Fatalf("withheld: facts=%+v, want accepted=true EOF=false", f)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean || rep.Reason != "status EOF not observed" {
		t.Fatalf("Clean=%v reason=%q, want EOF refusal", rep.Clean, rep.Reason)
	}
	if err := o.Kill(id); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !exit.Signaled {
		t.Fatalf("exit=%+v, want signaled", exit)
	}
	// The exit is recorded; EOF is still not: reaping never implies EOF.
	f, _ = o.Facts(id)
	if !f.Reaped || f.ChildExit == nil {
		t.Fatalf("after kill: facts=%+v, want reaped exit recorded", f)
	}
	if f.EOF {
		t.Fatalf("after kill: facts=%+v, want EOF still false", f)
	}
	// Only now does the collector observe the pipe close as EOF.
	if err := o.CollectStatus(id, time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus after kill: %v", err)
	}
	f, _ = o.Facts(id)
	if !f.EOF {
		t.Fatalf("after recollect: facts=%+v, want EOF=true", f)
	}
	rep, err = o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean {
		t.Fatalf("Clean=true for killed child")
	}
}

// The writer exit is its own fact, EPIPE included: a 512KB snapshot
// cannot fit the pipe buffer without a draining reader, so an
// instant-exiting child forces delivery to fail while the child itself
// exits 0.
func TestWriterExitRecordedSeparately(t *testing.T) {
	requireBin(t, testShell)
	o := testOwner(t)
	l := testLease(t, "epipe")
	env := bigSnapshotEnv()
	id, err := o.Spawn("f1epipe1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.Code != 0 {
		t.Fatalf("exit = %+v, want code 0", exit)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean {
		t.Fatalf("Clean=true despite failed delivery")
	}
	f := rep.Facts
	if !f.WriterDone {
		t.Fatalf("facts=%+v, want writer-exit recorded", f)
	}
	if f.Offered {
		t.Fatalf("facts=%+v, want offered=false for undrainable snapshot", f)
	}
	if !strings.Contains(strings.ToLower(f.WriterErr), "pipe") {
		t.Fatalf("facts=%+v, want EPIPE writer error", f)
	}
	// Child green, writer red: neither fact implies the other.
	if f.ChildExit == nil || f.ChildExit.Code != 0 {
		t.Fatalf("facts=%+v, want child exit 0 recorded alongside", f)
	}
}

// The child result stays separate from acceptance: full drain, valid
// ack and clean EOF still refuse release when the child exits nonzero.
func TestChildResultSeparate(t *testing.T) {
	requireBin(t, testShell, testCat)
	o := testOwner(t)
	l := testLease(t, "exit3")
	opID := "f1exit31"
	env := map[string]string{"F1_K": "v"}
	script := testCat + ` <&3 >/dev/null; printf '%b' '` + process.OctalEscape(MustAckFrameFor(opID, env)) + `'; exit 3`
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err != nil {
		t.Fatalf("CollectStatus: %v", err)
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.Code != 3 || exit.Signaled {
		t.Fatalf("exit = %+v, want code 3", exit)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean || rep.Reason != "child did not exit 0" {
		t.Fatalf("Clean=%v reason=%q, want child-exit refusal", rep.Clean, rep.Reason)
	}
	f := rep.Facts
	if !f.Accepted || f.Malformed || !f.EOF {
		t.Fatalf("facts=%+v, want accepted+EOF green beside the red exit", f)
	}
}

// A child reading the wrong descriptor observes the wrong bytes: fd 0
// (null stdin) yields nothing while the snapshot waits on fd 3, so no
// content-bound ack can follow and acceptance stays red.
func TestWrongFDReadsWrongBytes(t *testing.T) {
	requireBin(t, testShell, testCat)
	o := testOwner(t)
	l := testLease(t, "wrongfd")
	opID := "f1wrongfd1"
	env := map[string]string{"F1_K": "v"}
	dir := t.TempDir()
	fd0 := filepath.Join(dir, "fd0.out")
	fd3 := filepath.Join(dir, "fd3.out")
	script := testCat + ` <&0 > "$1"; ` + testCat + ` <&3 > "$2"; exit 0`
	id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script, fd0, fd3), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if _, err := o.Wait(id, 10*time.Second); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got, err := os.ReadFile(fd0); err != nil || len(got) != 0 {
		t.Fatalf("fd 0 yielded %q err=%v, want empty null stdin", got, err)
	}
	wantSnap := MustRenderSnapshot(env)
	if got, err := os.ReadFile(fd3); err != nil || !bytes.Equal(got, wantSnap) {
		t.Fatalf("fd 3 yielded %q err=%v, want %q", got, err, wantSnap)
	}
	fdMap, err := o.FDMap(id)
	if err != nil {
		t.Fatalf("FDMap: %v", err)
	}
	if fdMap[process.FDEnv] != "env" {
		t.Fatalf("FDMap = %v, want fd 3 recorded as the env channel", fdMap)
	}
	// No ack was (or could be) minted from the wrong bytes.
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
		t.Fatalf("CollectStatus: want error, got nil")
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean || rep.Facts.Accepted {
		t.Fatalf("Clean=%v facts=%+v, want acceptance red", rep.Clean, rep.Facts)
	}
}

// Truncated status bytes are malformed, never accepted: a header cut
// short and a payload cut short both fail frame validation.
func TestTruncationNeverAccepted(t *testing.T) {
	requireBin(t, testShell)
	for _, v := range []struct {
		name string
		raw  []byte
	}{
		{"header", MalformedTruncatedHeader},
		{"payload", MalformedTruncatedPayload},
	} {
		t.Run(v.name, func(t *testing.T) {
			o := testOwner(t)
			l := testLease(t, "trunc")
			script := `printf '%b' '` + process.OctalEscape(v.raw) + `'; exit 0`
			id, err := o.Spawn("f1trunc-"+v.name, process.Spec{Executable: testShell, Args: shArgs(script), Env: map[string]string{"F1_K": "v"}, WithLease: true}, l)
			if err != nil {
				t.Fatalf("Spawn: %v", err)
			}
			defer mustAbort(o, id)
			if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
				t.Fatalf("CollectStatus: want truncation error, got nil")
			}
			if _, err := o.Wait(id, 10*time.Second); err != nil {
				t.Fatalf("Wait: %v", err)
			}
			rep, err := o.Release(id)
			if err != nil {
				t.Fatalf("Release: %v", err)
			}
			if rep.Clean || !rep.Facts.Malformed || rep.Facts.Accepted {
				t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
			}
		})
	}
}

// Every malformed status vector refuses clean release on its content
// failure: head-position vectors are never accepted, trailer-position
// vectors keep their ack but are still malformed.
func TestMalformedStatusVectorsRefuse(t *testing.T) {
	requireBin(t, testShell)
	t.Run("head", func(t *testing.T) {
		for _, v := range MalformedHeadVectors() {
			// Truncation has its own control above; the rest run here.
			if v.Name == "truncated-header" || v.Name == "truncated-payload" {
				continue
			}
			t.Run(v.Name, func(t *testing.T) {
				o := testOwner(t)
				l := testLease(t, "malhead")
				script := `printf '%b' '` + process.OctalEscape(v.Bytes) + `'; exit 0`
				id, err := o.Spawn("f1malhead-"+v.Name, process.Spec{Executable: testShell, Args: shArgs(script), Env: map[string]string{"F1_K": "v"}, WithLease: true}, l)
				if err != nil {
					t.Fatalf("Spawn: %v", err)
				}
				defer mustAbort(o, id)
				if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
					t.Fatalf("CollectStatus: want error, got nil")
				}
				if _, err := o.Wait(id, 10*time.Second); err != nil {
					t.Fatalf("Wait: %v", err)
				}
				rep, err := o.Release(id)
				if err != nil {
					t.Fatalf("Release: %v", err)
				}
				if rep.Clean || !rep.Facts.Malformed || rep.Facts.Accepted {
					t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
				}
			})
		}
		t.Run("gap-head", func(t *testing.T) {
			o := testOwner(t)
			l := testLease(t, "gaphead")
			script := `printf '%b' '` + process.OctalEscape(GapHeadFrame()) + `'; exit 0`
			id, err := o.Spawn("f1gaphead1", process.Spec{Executable: testShell, Args: shArgs(script), Env: map[string]string{"F1_K": "v"}, WithLease: true}, l)
			if err != nil {
				t.Fatalf("Spawn: %v", err)
			}
			defer mustAbort(o, id)
			if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
				t.Fatalf("CollectStatus: want gap error, got nil")
			}
			if _, err := o.Wait(id, 10*time.Second); err != nil {
				t.Fatalf("Wait: %v", err)
			}
			rep, err := o.Release(id)
			if err != nil {
				t.Fatalf("Release: %v", err)
			}
			if rep.Clean || !rep.Facts.Malformed || rep.Facts.Accepted {
				t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, rep.Facts)
			}
		})
	})
	t.Run("trailer", func(t *testing.T) {
		env := map[string]string{"F1_K": "v"}
		cases := []struct {
			name    string
			trailer func(opID string) []byte
		}{
			{"second-frame", func(opID string) []byte { return SequenceFrame(1, []byte("extra")) }},
			{"duplicate-seq", func(opID string) []byte { return SequenceFrame(0, MustAckFrameFor(opID, env)) }},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				o := testOwner(t)
				l := testLease(t, "maltrail")
				opID := "f1maltrail-" + c.name
				ack := process.OctalEscape(MustAckFrameFor(opID, env))
				script := `printf '%b' '` + ack + `'; printf '%b' '` + process.OctalEscape(c.trailer(opID)) + `'; exit 0`
				id, err := o.Spawn(opID, process.Spec{Executable: testShell, Args: shArgs(script), Env: env, WithLease: true}, l)
				if err != nil {
					t.Fatalf("Spawn: %v", err)
				}
				defer mustAbort(o, id)
				if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
					t.Fatalf("CollectStatus: want trailer error, got nil")
				}
				if _, err := o.Wait(id, 10*time.Second); err != nil {
					t.Fatalf("Wait: %v", err)
				}
				rep, err := o.Release(id)
				if err != nil {
					t.Fatalf("Release: %v", err)
				}
				f := rep.Facts
				if rep.Clean || !f.Malformed {
					t.Fatalf("Clean=%v facts=%+v, want refused+malformed", rep.Clean, f)
				}
				if !f.Accepted {
					t.Fatalf("facts=%+v, want the leading ack kept", f)
				}
			})
		}
	})
}

// Sanitation: Unicode names and hostile values round-trip exactly,
// missing names stay absent, hostile names and oversize entries are
// rejected at spawn, and invalid UTF-8 in a value is pinned to the
// JSON U+FFFD replacement the owner renders.
func TestSanitation(t *testing.T) {
	requireBin(t, testShell, testCat)
	t.Run("unicode-empty-missing", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "sanuni")
		env := map[string]string{EmptyValueKey: ""}
		for _, name := range UnicodeEnvNames {
			env[name] = "v-" + name
		}
		dir := t.TempDir()
		drained := filepath.Join(dir, "fd3.out")
		script := testCat + ` <&3 > "$1"; exit 0`
		id, err := o.Spawn("f1sanuni1", process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		wantSnap := MustRenderSnapshot(env)
		got, err := os.ReadFile(drained)
		if err != nil || !bytes.Equal(got, wantSnap) {
			t.Fatalf("child drained %q err=%v, want %q", got, err, wantSnap)
		}
		if bytes.Contains(got, []byte(MissingName)) {
			t.Fatalf("snapshot %q names %s, want absent", got, MissingName)
		}
		if snap, err := o.Snapshot(id); err != nil || !bytes.Equal(snap, wantSnap) {
			t.Fatalf("Snapshot = %q err=%v, want %q", snap, err, wantSnap)
		}
	})
	t.Run("hostile-values-round-trip", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "sanval")
		env := map[string]string{"F1_HOSTILE": "a\nb\rc%sd\x00eÜ"}
		dir := t.TempDir()
		drained := filepath.Join(dir, "fd3.out")
		script := testCat + ` <&3 > "$1"; exit 0`
		id, err := o.Spawn("f1sanval1", process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		wantSnap := MustRenderSnapshot(env)
		if got, err := os.ReadFile(drained); err != nil || !bytes.Equal(got, wantSnap) {
			t.Fatalf("child drained %q err=%v, want %q", got, err, wantSnap)
		}
	})
	t.Run("invalid-utf8-pinned", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "sanutf8")
		env := map[string]string{"F1_BIN": "a\xffb"}
		dir := t.TempDir()
		drained := filepath.Join(dir, "fd3.out")
		script := testCat + ` <&3 > "$1"; exit 0`
		id, err := o.Spawn("f1sanutf81", process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		// encoding/json replaces the undecodable byte with U+FFFD.
		want := []byte("{\"F1_BIN\":\"a\xef\xbf\xbdb\"}")
		if got, err := os.ReadFile(drained); err != nil || !bytes.Equal(got, want) {
			t.Fatalf("child drained %q err=%v, want %q", got, err, want)
		}
	})
	t.Run("hostile-names-rejected", func(t *testing.T) {
		o := testOwner(t)
		for i, key := range HostileEnvKeys {
			_, err := o.Spawn("f1sankey1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: map[string]string{key: "v"}}, nil)
			if !errors.Is(err, process.ErrInvalid) {
				t.Fatalf("key[%d] %q: Spawn err=%v, want ErrInvalid", i, key, err)
			}
		}
		if n := o.Live(); n != 0 {
			t.Fatalf("Live() = %d, want 0 (rejected spawns retain nothing)", n)
		}
	})
	t.Run("entry-bound-exact", func(t *testing.T) {
		requireBin(t, testShell, testCat)
		o := testOwner(t)
		l := testLease(t, "sanbound")
		over := strings.Repeat("v", process.MaxEnvEntry-1) // 1+1+len > bound
		if _, err := o.Spawn("f1sanbound-over", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: map[string]string{"K": over}}, nil); !errors.Is(err, process.ErrInvalid) {
			t.Fatalf("oversize entry: Spawn err=%v, want ErrInvalid", err)
		}
		edge := strings.Repeat("v", process.MaxEnvEntry-2) // 1+1+len == bound
		dir := t.TempDir()
		drained := filepath.Join(dir, "fd3.out")
		script := testCat + ` <&3 > "$1"; exit 0`
		id, err := o.Spawn("f1sanbound-edge", process.Spec{Executable: testShell, Args: shArgs(script, drained), Env: map[string]string{"K": edge}, WithLease: true}, l)
		if err != nil {
			t.Fatalf("boundary entry: Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		wantSnap := MustRenderSnapshot(map[string]string{"K": edge})
		if got, err := os.ReadFile(drained); err != nil || !bytes.Equal(got, wantSnap) {
			t.Fatalf("boundary drain mismatch: len=%d err=%v, want %d", len(got), err, len(wantSnap))
		}
	})
}

// An unused snapshot never disturbs the child: the offer may land in
// the pipe, but nothing is consumed or accepted, the child still exits
// 0, and release still refuses on the missing acceptance.
func TestUnusedSnapshot(t *testing.T) {
	requireBin(t, testShell)
	o := testOwner(t)
	l := testLease(t, "unused")
	env := UnusedSnapshotEnv()
	id, err := o.Spawn("f1unused1", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: env, WithLease: true}, l)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer mustAbort(o, id)
	if err := o.CollectStatus(id, 5*time.Second, 5*time.Second); err == nil {
		t.Fatalf("CollectStatus: want error, got nil")
	}
	exit, err := o.Wait(id, 10*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.Code != 0 {
		t.Fatalf("exit = %+v, want code 0", exit)
	}
	rep, err := o.Release(id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if rep.Clean {
		t.Fatalf("Clean=true for unused snapshot")
	}
	f := rep.Facts
	if f.Accepted || !f.EOF || !f.WriterDone {
		t.Fatalf("facts=%+v, want accepted=false EOF=true writer-joined", f)
	}
	if f.ChildExit == nil || f.ChildExit.Code != 0 {
		t.Fatalf("facts=%+v, want child exit 0 recorded", f)
	}
}

// A non-reader never strands the writer: the delivery goroutine always
// joins with a recorded outcome (full offer or EPIPE) inside a bounded
// wait, whether the child exits instantly or is killed mid-delivery.
func TestNonReaderDoesNotStrandWriter(t *testing.T) {
	requireBin(t, testShell, testSleep)
	t.Run("instant-exit", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "strand-instant")
		id, err := o.Spawn("f1strand-instant", process.Spec{Executable: testShell, Args: shArgs(`exit 0`), Env: UnusedSnapshotEnv(), WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		_ = o.CollectStatus(id, 5*time.Second, 5*time.Second)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		start := time.Now()
		rep, err := o.Release(id)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true for non-reader")
		}
		if !rep.Facts.WriterDone {
			t.Fatalf("facts=%+v, want writer-exit recorded", rep.Facts)
		}
		if elapsed > 30*time.Second {
			t.Fatalf("Release took %v, want bounded writer join", elapsed)
		}
	})
	t.Run("killed-mid-delivery", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "strand-mid")
		env := bigSnapshotEnv()
		id, err := o.Spawn("f1strand-mid", process.Spec{Executable: testShell, Args: shArgs(`exec ` + testSleep + ` 30`), Env: env, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		// Let the writer block on the full pipe behind the live
		// non-reader, then kill: delivery must fail closed, not hang.
		time.Sleep(500 * time.Millisecond)
		if err := o.Kill(id); err != nil {
			t.Fatalf("Kill: %v", err)
		}
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		start := time.Now()
		rep, err := o.Release(id)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if rep.Clean {
			t.Fatalf("Clean=true for killed non-reader")
		}
		f := rep.Facts
		if !f.WriterDone {
			t.Fatalf("facts=%+v, want writer-exit recorded", f)
		}
		if f.Offered || !strings.Contains(strings.ToLower(f.WriterErr), "pipe") {
			t.Fatalf("facts=%+v, want EPIPE writer outcome", f)
		}
		if elapsed > 30*time.Second {
			t.Fatalf("Release took %v, want bounded writer join", elapsed)
		}
	})
}

// Binary stimuli traverse the shell-embedding path exactly: every
// hostile vector round-trips through OctalEscape + printf %b to a
// fixture child byte for byte.
func TestBinaryStimuliDeliverExactly(t *testing.T) {
	requireBin(t, testShell)
	for _, v := range BinaryStdinVectors() {
		t.Run(v.Name, func(t *testing.T) {
			o := testOwner(t)
			l := testLease(t, "binstim")
			dir := t.TempDir()
			out := filepath.Join(dir, "stim.out")
			script := `printf '%b' '` + process.OctalEscape(v.Bytes) + `' > "$1"`
			id, err := o.Spawn("f1binstim-"+v.Name, process.Spec{Executable: testShell, Args: shArgs(script, out), Env: map[string]string{"F1_K": "v"}, WithLease: true}, l)
			if err != nil {
				t.Fatalf("Spawn: %v", err)
			}
			defer mustAbort(o, id)
			if _, err := o.Wait(id, 10*time.Second); err != nil {
				t.Fatalf("Wait: %v", err)
			}
			if got, err := os.ReadFile(out); err != nil || !bytes.Equal(got, v.Bytes) {
				t.Fatalf("child wrote %q err=%v, want %q", got, err, v.Bytes)
			}
		})
	}
}

// Startup isolation: a fixture child observes exactly the recorded fds
// (0-3, plus 4 only with a lease) and no private owner fd above them.
func TestNoPrivateOwnerFD(t *testing.T) {
	requireBin(t, testShell, testCat)
	// Probing with cat: a closed fd fails at once; an open one would be
	// drained, which is why the probe range starts above every recorded
	// fd. The Wait timeout plus deferred Abort bound a leak finding.
	probe := func(closed string) string {
		return `for f in ` + closed + `; do if ` + testCat + ` <&$f >/dev/null 2>&1; then echo "OPEN $f" >> "$1"; fi; done; echo done >> "$1"`
	}
	t.Run("with-lease", func(t *testing.T) {
		o := testOwner(t)
		l := testLease(t, "nofd-lease")
		dir := t.TempDir()
		out := filepath.Join(dir, "fds.out")
		id, err := o.Spawn("f1nofd-lease", process.Spec{Executable: testShell, Args: shArgs(probe("5 6 7 8 9"), out), Env: map[string]string{"F1_K": "v"}, WithLease: true}, l)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if got, err := os.ReadFile(out); err != nil || string(got) != "done\n" {
			t.Fatalf("probe = %q err=%v, want only done (no fd 5-9 open)", got, err)
		}
		fdMap, err := o.FDMap(id)
		if err != nil {
			t.Fatalf("FDMap: %v", err)
		}
		if len(fdMap) != 5 || fdMap[process.FDEnv] != "env" || fdMap[process.FDLease] != "lease" {
			t.Fatalf("FDMap = %v, want exactly 0-4 with 3=env 4=lease", fdMap)
		}
	})
	t.Run("without-lease", func(t *testing.T) {
		o := testOwner(t)
		dir := t.TempDir()
		out := filepath.Join(dir, "fds.out")
		id, err := o.Spawn("f1nofd-nolease", process.Spec{Executable: testShell, Args: shArgs(probe("4 5 6 7 8 9"), out), Env: map[string]string{"F1_K": "v"}}, nil)
		if err != nil {
			t.Fatalf("Spawn: %v", err)
		}
		defer mustAbort(o, id)
		if _, err := o.Wait(id, 10*time.Second); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if got, err := os.ReadFile(out); err != nil || string(got) != "done\n" {
			t.Fatalf("probe = %q err=%v, want only done (no fd 4-9 open)", got, err)
		}
		fdMap, err := o.FDMap(id)
		if err != nil {
			t.Fatalf("FDMap: %v", err)
		}
		if len(fdMap) != 4 {
			t.Fatalf("FDMap = %v, want exactly 0-3", fdMap)
		}
	})
}

// The subject of record is the existing Go CLI, the canlc compiler
// launcher: this pins its identity from source (read-only, no build)
// and records why descriptor delivery is driven through the P10 owner
// with fixture children instead — the CLI never mints content-bound
// status acks, so it cannot speak the fd-3 protocol.
func TestSubjectOfRecord(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller: unavailable")
	}
	// Ascend to the module root (marked by go.mod), wherever the
	// checkout lives.
	root := filepath.Dir(self)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	mainPath := filepath.Join(root, "compiler", "main.go")
	raw, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("subject compiler/main.go unreadable: %v", err)
	}
	src := string(raw)
	if !strings.Contains(src, "package main") {
		t.Fatalf("%s: want package main", mainPath)
	}
	if !strings.Contains(src, "canlc") {
		t.Fatalf("%s: want canlc usage identity", mainPath)
	}
	// A content-bound ack always carries the "ACK:" prefix; a child
	// that never emits it cannot be Accepted, however it exits.
	if strings.Contains(src, "ACK:") {
		t.Fatalf("%s: mints status acks; K18 subject resolution stale", mainPath)
	}
	if strings.Contains(src, "native-test-owner/protocol/codec") {
		t.Fatalf("%s: speaks the status frame protocol; K18 subject resolution stale", mainPath)
	}
}

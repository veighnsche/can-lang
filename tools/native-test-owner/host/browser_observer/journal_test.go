// Durable publication controls: append, fsync, replay, and honest
// truncation. Journal files live in t.TempDir; no fixture binaries, no
// network, no live hosts.
package browser_observer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func journalHeaderForTest() journalHeader {
	return journalHeader{
		Owner: testOwner, Scope: "host-ui",
		Channels: []string{"window", "icon"},
		LaunchID: "launch-1", Limits: DefaultLimits(),
	}
}

func writeTestJournal(t *testing.T, path string, ops ...journalEntry) {
	t.Helper()
	j, err := CreateJournal(path)
	if err != nil {
		t.Fatalf("CreateJournal: %v", err)
	}
	defer j.Close()
	if err := j.WriteHeader(journalHeaderForTest()); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	for _, op := range ops {
		if err := j.Append(op.Op, op.Channel, op.State, op.Tick, op.After); err != nil {
			t.Fatalf("Append %v: %v", op, err)
		}
	}
}

func sampleEntry(channel, state string) journalEntry {
	return journalEntry{Op: OpSample, Channel: channel, State: state}
}

func TestJournalRefusesStaleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pub.log")
	if err := os.WriteFile(path, []byte("stale\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := CreateJournal(path); err == nil {
		t.Fatal("publication appended to a stale journal")
	}
}

func TestJournalReplayRecoversInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pub.log")
	writeTestJournal(t, path,
		sampleEntry("window", StateAbsent),
		journalEntry{Op: OpEndTick},
		sampleEntry("window", StateSeen),
		sampleEntry("icon", StateAbsent),
		journalEntry{Op: OpNoteDriverDeath},
		journalEntry{Op: OpPublishCorr, Channel: "icon", Tick: 0, After: StateAbsent},
	)
	engine, token, result, err := Replay(path)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if result.Truncated || result.Entries != 6 {
		t.Fatalf("clean journal misreplayed: %+v", result)
	}
	if result.Header.LaunchID != "launch-1" || result.Header.Scope != "host-ui" {
		t.Fatalf("header lost: %+v", result.Header)
	}
	facts, err := engine.IntervalFacts(testOwner, "host-ui", "launch-1")
	if err != nil {
		t.Fatalf("IntervalFacts: %v", err)
	}
	if facts.Unknown || !facts.DriverDeathNoted || len(facts.Corrections) != 1 {
		t.Fatalf("replayed interval wrong: %+v", facts)
	}
	if facts.Corrections[0].CorrectionID != "c1" || facts.Corrections[0].Before != StateGap {
		t.Fatalf("correction not durable: %+v", facts.Corrections[0])
	}
	// The replayed engine seals: the token survived with the facts.
	receipt, err := engine.SealDisposal(testOwner, "host-ui", "launch-1", token)
	if err != nil {
		t.Fatalf("SealDisposal: %v", err)
	}
	if receipt.Unknown || receipt.Corrections != 1 || receipt.Ticks != 2 {
		t.Fatalf("replayed seal wrong: %+v", receipt)
	}
}

func TestJournalTruncationIsHonest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pub.log")
	writeTestJournal(t, path,
		sampleEntry("window", StateAbsent),
		journalEntry{Op: OpEndTick},
	)
	// A torn tail: the last line never finished writing.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := f.WriteString(`{"seq":3,"op":"sam`); err != nil {
		t.Fatalf("tear: %v", err)
	}
	f.Close()
	_, _, result, err := Replay(path)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if !result.Truncated || result.Entries != 2 {
		t.Fatalf("torn tail hidden: %+v", result)
	}
}

func TestJournalCorruptionStopsReplay(t *testing.T) {
	cases := map[string]string{
		"corrupt line":  "{\"seq\":2,\"op\":}\n",
		"sequence gap":  "{\"seq\":9,\"op\":\"end-tick\"}\n",
		"unknown op":    "{\"seq\":2,\"op\":\"reboot-observer\"}\n",
		"unknown field": "{\"seq\":2,\"op\":\"end-tick\",\"smuggled\":true}\n",
		"unappliable":   "{\"seq\":2,\"op\":\"publish-correction\",\"tick\":7,\"channel\":\"icon\",\"after\":\"absent\"}\n",
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pub.log")
			writeTestJournal(t, path, sampleEntry("window", StateAbsent))
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if _, err := f.WriteString(extra); err != nil {
				t.Fatalf("append: %v", err)
			}
			f.Close()
			engine, _, result, err := Replay(path)
			if err != nil {
				t.Fatalf("Replay: %v", err)
			}
			if !result.Truncated || result.Entries != 1 {
				t.Fatalf("corruption hidden: %+v", result)
			}
			facts, err := engine.IntervalFacts(testOwner, "host-ui", "launch-1")
			if err != nil {
				t.Fatalf("IntervalFacts: %v", err)
			}
			if len(facts.Ticks) != 1 || len(facts.Ticks[0].Channels) != 1 {
				t.Fatalf("prefix facts lost: %+v", facts)
			}
		})
	}
}

func TestJournalRefusesBadHeader(t *testing.T) {
	// Wrong-grants journals fail at bind time, not silently.
	path := filepath.Join(t.TempDir(), "pub.log")
	j, err := CreateJournal(path)
	if err != nil {
		t.Fatalf("CreateJournal: %v", err)
	}
	defer j.Close()
	h := journalHeaderForTest()
	h.Channels = []string{"not-a-channel"}
	if err := j.WriteHeader(h); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, _, _, err := Replay(path); err == nil {
		t.Fatal("journal with unbindable grants replayed")
	}
	// Empty and malformed headers refuse.
	for _, content := range []string{"", "hello\n", "{\"seq\":1,\"op\":\"end-tick\"}\n"} {
		path := filepath.Join(t.TempDir(), "pub.log")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if _, _, _, err := Replay(path); err == nil {
			t.Fatalf("bad header replayed: %q", content)
		}
	}
}

func TestJournalHeaderIsFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pub.log")
	j, err := CreateJournal(path)
	if err != nil {
		t.Fatalf("CreateJournal: %v", err)
	}
	defer j.Close()
	if err := j.WriteHeader(journalHeaderForTest()); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := j.WriteHeader(journalHeaderForTest()); err == nil {
		t.Fatal("second header accepted")
	}
	if err := j.Append("shutdown", "", "", 0, ""); err == nil {
		t.Fatal("read op journaled")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") || strings.Count(string(raw), "\n") != 1 {
		t.Fatalf("journal framing wrong: %q", raw)
	}
}

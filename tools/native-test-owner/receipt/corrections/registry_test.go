package corrections

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registryPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func readRaw(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %q: %v", path, err)
	}
	return string(raw)
}

func appendMust(t *testing.T, path string, e Entry) Record {
	t.Helper()
	rec, err := Append(path, e)
	if err != nil {
		t.Fatalf("Append %+v: %v", e, err)
	}
	return rec
}

// TestAppendGrowthIsTheDigestChainControl is the headline acceptance
// clause: each correction commits to the previous digest, the old bytes
// survive the append verbatim, and the new head differs while chaining
// to the old one.
func TestAppendGrowthIsTheDigestChainControl(t *testing.T) {
	path := registryPath(t, "chain.jsonl")

	before, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if before.Head() != GenesisPrev {
		t.Fatalf("empty head = %q, want %q", before.Head(), GenesisPrev)
	}

	first := appendMust(t, path, Entry{CaseID: "process/nonzero-result", Outcome: OutcomeFailed, Note: "exit-code drift"})
	if first.Seq != 0 || first.PrevDigest != GenesisPrev {
		t.Fatalf("genesis record = %+v, want seq 0 prev genesis", first)
	}
	if first.Digest == "" || first.Digest == GenesisPrev {
		t.Fatalf("genesis digest = %q, want a fresh digest", first.Digest)
	}
	afterOne := readRaw(t, path)

	second := appendMust(t, path, Entry{CaseID: "process/nonzero-result", Outcome: OutcomePass, Note: "re-run clean"})
	if second.Seq != 1 || second.PrevDigest != first.Digest {
		t.Fatalf("second record = %+v, want seq 1 prev %q", second, first.Digest)
	}
	if second.Digest == first.Digest {
		t.Fatalf("second digest repeats the first: %q", second.Digest)
	}
	afterTwo := readRaw(t, path)
	if !strings.HasPrefix(afterTwo, afterOne) {
		t.Fatalf("append rewrote history:\nbefore %q\nafter %q", afterOne, afterTwo)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Records) != 2 || loaded.Head() != second.Digest {
		t.Fatalf("loaded = %+v, want 2 records headed %q", loaded, second.Digest)
	}
	if loaded.Records[0] != first || loaded.Records[1] != second {
		t.Fatalf("loaded records drifted from appended records")
	}

	// The before/after digest pair for the P22 evidence file.
	t.Logf("P22-CHAIN before=%s after=%s", first.Digest, second.Digest)

	// A fresh pass never erases the old failure.
	res, err := loaded.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Standing != OutcomeFailed || res.Head != second.Digest || res.Count != 2 {
		t.Fatalf("resolution = %+v, want failed standing at the new head", res)
	}
}

// TestDigestDeterminism pins stable evidence: the same entries built in
// two directories carry identical digests, because records hold no
// timestamps or process identities.
func TestDigestDeterminism(t *testing.T) {
	entries := []Entry{
		{CaseID: "a", Outcome: OutcomeFailed, Note: "first"},
		{CaseID: "a", Outcome: OutcomePass, Note: "second"},
	}
	build := func(t *testing.T) *Registry {
		t.Helper()
		path := registryPath(t, "det.jsonl")
		for _, e := range entries {
			appendMust(t, path, e)
		}
		reg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		return reg
	}
	left, right := build(t), build(t)
	if left.Head() != right.Head() {
		t.Fatalf("heads differ: %q vs %q", left.Head(), right.Head())
	}
	if len(left.Records) != 2 || left.Records[0].Digest != right.Records[0].Digest {
		t.Fatalf("genesis digests differ: %+v vs %+v", left.Records, right.Records)
	}
}

func TestResolveStickyTable(t *testing.T) {
	cases := []struct {
		name     string
		outcomes []Outcome
		want     Outcome
	}{
		{"empty-resolves-pass", nil, OutcomePass},
		{"single-pass", []Outcome{OutcomePass}, OutcomePass},
		{"all-pass", []Outcome{OutcomePass, OutcomePass, OutcomePass}, OutcomePass},
		{"single-failed", []Outcome{OutcomeFailed}, OutcomeFailed},
		{"fresh-pass-keeps-old-failure", []Outcome{OutcomeFailed, OutcomePass}, OutcomeFailed},
		{"fresh-pass-keeps-old-failure-late", []Outcome{OutcomePass, OutcomeFailed, OutcomePass}, OutcomeFailed},
		{"single-incomplete", []Outcome{OutcomeIncomplete}, OutcomeIncomplete},
		{"fresh-pass-keeps-old-incomplete", []Outcome{OutcomeIncomplete, OutcomePass}, OutcomeIncomplete},
		{"failed-beats-incomplete", []Outcome{OutcomeIncomplete, OutcomeFailed, OutcomePass}, OutcomeFailed},
		{"incomplete-beats-pass", []Outcome{OutcomePass, OutcomeIncomplete}, OutcomeIncomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := registryPath(t, "sticky.jsonl")
			for i, o := range c.outcomes {
				appendMust(t, path, Entry{CaseID: "a", Outcome: o, Note: strings.Repeat("n", i+1)})
			}
			reg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			res, err := reg.Resolve()
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if res.Standing != c.want || res.Count != len(c.outcomes) || res.Head != reg.Head() {
				t.Fatalf("resolution = %+v, want standing %q count %d", res, c.want, len(c.outcomes))
			}
			if len(c.outcomes) == 0 && res.Head != GenesisPrev {
				t.Fatalf("empty head = %q, want %q", res.Head, GenesisPrev)
			}
		})
	}
}

// TestTamperRefusalTable mutates a healthy two-record chain in every
// structural way: each mutation refuses load and resolution, and a
// further append onto the tampered file is refused without writing.
func TestTamperRefusalTable(t *testing.T) {
	healthy := func(t *testing.T) string {
		t.Helper()
		path := registryPath(t, "tamper.jsonl")
		appendMust(t, path, Entry{CaseID: "a", Outcome: OutcomeFailed, Note: "first"})
		appendMust(t, path, Entry{CaseID: "a", Outcome: OutcomePass, Note: "second"})
		return path
	}
	mutate := func(t *testing.T, path, old, new string, count int) {
		t.Helper()
		raw := readRaw(t, path)
		if !strings.Contains(raw, old) {
			t.Fatalf("mutation anchor %q not found", old)
		}
		changed := strings.Replace(raw, old, new, count)
		if changed == raw {
			t.Fatalf("mutation %q -> %q changed nothing", old, new)
		}
		if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	cases := []struct {
		name string
		want error
		edit func(t *testing.T, path string)
	}{
		{"digest-flip", ErrBrokenChain, func(t *testing.T, path string) {
			raw := readRaw(t, path)
			anchor := `"digest":"sha256:`
			i := strings.Index(raw, anchor) + len(anchor)
			flip := raw[i : i+1]
			swap := "a"
			if flip == "a" {
				swap = "b"
			}
			mutate(t, path, anchor+flip, anchor+swap, 1)
		}},
		{"prev-flip", ErrBrokenChain, func(t *testing.T, path string) {
			raw := readRaw(t, path)
			anchor := `"prevDigest":"sha256:`
			i := strings.Index(raw, anchor) + len(anchor)
			flip := raw[i : i+1]
			swap := "a"
			if flip == "a" {
				swap = "b"
			}
			mutate(t, path, anchor+flip, anchor+swap, 1)
		}},
		{"genesis-prev-flip", ErrBrokenChain, func(t *testing.T, path string) {
			mutate(t, path, `"prevDigest":"genesis"`, `"prevDigest":"genesiX"`, 1)
		}},
		{"resequence", ErrBrokenChain, func(t *testing.T, path string) {
			mutate(t, path, `"seq":1`, `"seq":7`, 1)
		}},
		{"swapped-order", ErrBrokenChain, func(t *testing.T, path string) {
			raw := readRaw(t, path)
			lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
			if len(lines) != 2 {
				t.Fatalf("want 2 lines, got %d", len(lines))
			}
			if err := os.WriteFile(path, []byte(lines[1]+"\n"+lines[0]+"\n"), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
		}},
		{"long-note-on-disk", ErrInvalid, func(t *testing.T, path string) {
			mutate(t, path, `"note":"second"`, `"note":"second`+strings.Repeat("n", MaxNoteLen)+`"`, 1)
		}},
		{"unknown-outcome", ErrInvalid, func(t *testing.T, path string) {
			mutate(t, path, `"outcome":"pass"`, `"outcome":"bogus"`, 1)
		}},
		{"unknown-schema", ErrUnknownSchema, func(t *testing.T, path string) {
			mutate(t, path, `"schemaVersion":"1"`, `"schemaVersion":"99"`, 1)
		}},
		{"malformed-json", ErrInvalid, func(t *testing.T, path string) {
			mutate(t, path, `"note":"second"`, `"note":`, 1)
		}},
		{"blank-then-resequence", ErrBrokenChain, func(t *testing.T, path string) {
			// Blank lines are skipped, so smuggling one in cannot
			// renumber the chain; the reseq still refuses.
			raw := readRaw(t, path)
			if err := os.WriteFile(path, []byte("\n"+strings.Replace(raw, `"seq":1`, `"seq":1`, 1)), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			mutate(t, path, `"seq":1`, `"seq":2`, 1)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := healthy(t)
			c.edit(t, path)
			tampered := readRaw(t, path)
			if _, err := Load(path); !errors.Is(err, c.want) {
				t.Fatalf("Load: got %v, want %v", err, c.want)
			}
			if _, err := Append(path, Entry{CaseID: "a", Outcome: OutcomePass, Note: "late"}); !errors.Is(err, c.want) {
				t.Fatalf("Append onto tampered chain: got %v, want %v", err, c.want)
			}
			if got := readRaw(t, path); got != tampered {
				t.Fatalf("refused append rewrote the file:\nbefore %q\nafter %q", tampered, got)
			}
		})
	}
}

func TestAppendRefusalsTable(t *testing.T) {
	path := registryPath(t, "refuse.jsonl")
	appendMust(t, path, Entry{CaseID: "a", Outcome: OutcomePass, Note: "first"})
	pristine := readRaw(t, path)
	cases := []struct {
		name  string
		entry Entry
	}{
		{"empty-case", Entry{Outcome: OutcomePass}},
		{"malformed-case", Entry{CaseID: "../x", Outcome: OutcomePass}},
		{"long-case", Entry{CaseID: strings.Repeat("c", MaxCaseIDLen+1), Outcome: OutcomePass}},
		{"unknown-outcome", Entry{CaseID: "a", Outcome: "bogus"}},
		{"empty-outcome", Entry{CaseID: "a"}},
		{"long-note", Entry{CaseID: "a", Outcome: OutcomePass, Note: strings.Repeat("n", MaxNoteLen+1)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Append(path, c.entry); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Append: got %v, want ErrInvalid", err)
			}
		})
	}
	if _, err := Append("", Entry{CaseID: "a", Outcome: OutcomePass}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty path Append: got %v, want ErrInvalid", nil)
	}
	if got := readRaw(t, path); got != pristine {
		t.Fatalf("refused appends rewrote the file:\nbefore %q\nafter %q", pristine, got)
	}
	// The chain still loads and resolves after the refusals.
	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	res, err := reg.Resolve()
	if err != nil || res.Standing != OutcomePass || res.Count != 1 {
		t.Fatalf("Resolve = (%+v, %v), want pass count 1", res, err)
	}
}

func TestLoadRefusalsTable(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.jsonl")); err != nil {
		t.Fatalf("missing file is a healthy empty chain, got %v", err)
	}
	empty := registryPath(t, "empty.jsonl")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if reg, err := Load(empty); err != nil || len(reg.Records) != 0 {
		t.Fatalf("empty file = (%+v, %v), want healthy empty chain", reg, err)
	}
	oversize := registryPath(t, "big.jsonl")
	if err := os.WriteFile(oversize, []byte(strings.Repeat("x", MaxRegistryBytes+8)), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(oversize); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversize Load: got %v, want ErrInvalid", err)
	}
	if _, err := Append(oversize, Entry{CaseID: "a", Outcome: OutcomePass}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversize Append: got %v, want ErrInvalid", err)
	}
	badJSON := registryPath(t, "bad.jsonl")
	if err := os.WriteFile(badJSON, []byte("{not json}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(badJSON); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed Load: got %v, want ErrInvalid", err)
	}
}

// TestResolveReverifies pins that resolution never trusts an unverified
// in-memory chain: a hand-built broken vector refuses even though it was
// never loaded from disk.
func TestResolveReverifies(t *testing.T) {
	broken := &Registry{Records: []Record{
		{SchemaVersion: SchemaVersion, Seq: 0, PrevDigest: GenesisPrev, Digest: "sha256:dead", CaseID: "a", Outcome: OutcomePass},
	}}
	if _, err := broken.Resolve(); !errors.Is(err, ErrBrokenChain) {
		t.Fatalf("Resolve broken: got %v, want ErrBrokenChain", nil)
	}
	if _, err := (*Registry)(nil).Resolve(); err != nil {
		t.Fatalf("nil registry Resolve: got %v, want healthy empty", err)
	}
	var nilReg *Registry
	if nilReg.Head() != GenesisPrev {
		t.Fatalf("nil head = %q, want %q", nilReg.Head(), GenesisPrev)
	}
}

// TestCorrectionNoteRecorded pins that corrections carry their audit
// trail: the note survives the append/load round trip verbatim.
func TestCorrectionNoteRecorded(t *testing.T) {
	path := registryPath(t, "note.jsonl")
	want := "harness bug 7: worker env leaked; re-run clean"
	rec := appendMust(t, path, Entry{CaseID: "process/nonzero-result", Outcome: OutcomePass, Note: want})
	if rec.Note != want {
		t.Fatalf("appended note = %q, want %q", rec.Note, want)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(reg.Records) != 1 || reg.Records[0].Note != want {
		t.Fatalf("loaded = %+v, want the recorded note", reg.Records)
	}
}

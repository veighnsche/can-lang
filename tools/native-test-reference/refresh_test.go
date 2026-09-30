package reference

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

var (
	refreshPredDigest = strings.Repeat("1", 64)
	refreshSeedDigest = strings.Repeat("2", 64)
	refreshCandDigest = strings.Repeat("3", 64)
	refreshFileA      = strings.Repeat("a", 64)
	refreshFileAV2    = strings.Repeat("b", 64)
	refreshFileB      = strings.Repeat("c", 64)
	refreshFileNew    = strings.Repeat("d", 64)
)

// refreshPromotionFixture builds a fully consistent happy-path promotion:
// predecessor R1 seal, refresh candidate advancing bin/canlc plus one runtime
// file and one added file (all bridge-declared), frozen suite, one prior
// acceptance record re-observed by R1, and one new-feature record observed
// by the refresh seed.
func refreshPromotionFixture() Promotion {
	predecessor := SealIdentity{
		Kind:    "can.native-test.seed",
		Version: "r-seed-1",
		BundleFiles: map[string]string{
			"bin/canlc":    refreshPredDigest,
			"runtime/a.ts": refreshFileA,
			"runtime/b.ts": refreshFileB,
		},
	}
	candidate := Manifest{
		SchemaVersion: 1,
		Kind:          "can.native-test.seed",
		Version:       "r-seed-2",
		BundleFiles: map[string]string{
			"bin/canlc":    refreshSeedDigest,
			"runtime/a.ts": refreshFileAV2,
			"runtime/b.ts": refreshFileB,
			"runtime/c.ts": refreshFileNew,
		},
	}
	suiteFiles := map[string]string{"suite/one.can": refreshFileA, "suite/two.can": refreshFileB}
	suite := DigestSuite(suiteFiles)
	live := Manifest{
		SchemaVersion: 1,
		Kind:          predecessor.Kind,
		Version:       predecessor.Version,
		BundleFiles: map[string]string{
			"bin/canlc":    refreshPredDigest,
			"runtime/a.ts": refreshFileA,
			"runtime/b.ts": refreshFileB,
		},
	}
	rollback := SealIdentityOf(live)
	refreshSeed := Producer{Path: "/seed/r2/bin/canlc", SHA256: refreshSeedDigest}
	candidateCompiler := Producer{Path: "/tmp/candidate/canlc", SHA256: refreshCandDigest}
	return Promotion{
		Proposal: RefreshProposal{
			SchemaVersion: RefreshSchemaVersion,
			Kind:          RefreshKind,
			Predecessor:   predecessor,
			Suite:         suite,
			Bridge: []BridgeAssumption{
				{Path: "bin/canlc", Change: BridgeChanged, Reason: "rebuilt seed compiler for the new binding"},
				{Path: "runtime/a.ts", Change: BridgeChanged, Reason: "runtime support for the new binding"},
				{Path: "runtime/c.ts", Change: BridgeAdded, Reason: "new runtime module for the new syntax"},
			},
			Fixed: []FixedObservation{
				{Name: "new.binding", Stdout: "ok\n", Stderr: "", Exit: 0},
			},
			Rollback: rollback,
		},
		LivePredecessor:   live,
		Candidate:         candidate,
		RefreshSeed:       refreshSeed,
		CandidateCompiler: candidateCompiler,
		NewResults: map[string]ObservedResult{
			"new.binding": {Stdout: []byte("ok\n"), Stderr: []byte{}, Exit: 0, Producer: refreshSeed},
		},
		PriorFixed: []FixedObservation{
			{Name: "pass.parse", Stdout: "parsed\n", Stderr: "", Exit: 0},
		},
		PriorResults: map[string]ObservedResult{
			"pass.parse": {
				Stdout:   []byte("parsed\n"),
				Stderr:   []byte{},
				Exit:     0,
				Producer: Producer{Path: "/seed/r1/bin/canlc", SHA256: refreshPredDigest},
			},
		},
		Suite: suite,
	}
}

func TestDiffBundle(t *testing.T) {
	predecessor := SealIdentity{BundleFiles: map[string]string{"same": "1", "chg": "1", "del": "1"}}
	candidate := Manifest{BundleFiles: map[string]string{"same": "1", "chg": "2", "add": "3"}}
	delta := DiffBundle(predecessor, candidate)
	join := func(names []string) string { return strings.Join(names, ",") }
	if join(delta.Added) != "add" || join(delta.Changed) != "chg" || join(delta.Removed) != "del" {
		t.Fatalf("delta = %+v", delta)
	}
	if (DiffBundle(predecessor, Manifest{BundleFiles: map[string]string{"same": "1", "chg": "1", "del": "1"}})).Empty() != true {
		t.Fatal("identical manifests report a delta")
	}
}

func TestComparePredecessorCompatible(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*SealIdentity, *Manifest, *[]BridgeAssumption)
		wantErr string
	}{
		{name: "declared refresh accepted", mutate: func(*SealIdentity, *Manifest, *[]BridgeAssumption) {}},
		{
			name: "kind drift rejected",
			mutate: func(_ *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				c.Kind = "can.native-test.other"
			},
			wantErr: "kind",
		},
		{
			name: "same version rejected",
			mutate: func(p *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				c.Version = p.Version
			},
			wantErr: "does not advance",
		},
		{
			name: "empty version rejected",
			mutate: func(_ *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				c.Version = ""
			},
			wantErr: "no version",
		},
		{
			name: "undeclared change rejected",
			mutate: func(_ *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				c.BundleFiles["runtime/b.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "undeclared drift",
		},
		{
			name: "undeclared removal rejected",
			mutate: func(_ *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				delete(c.BundleFiles, "runtime/b.ts")
			},
			wantErr: "undeclared drift",
		},
		{
			name: "missing bridge entry rejected",
			mutate: func(_ *SealIdentity, _ *Manifest, b *[]BridgeAssumption) {
				*b = (*b)[:2]
			},
			wantErr: "no bridge assumption",
		},
		{
			name: "orphan assumption rejected",
			mutate: func(_ *SealIdentity, _ *Manifest, b *[]BridgeAssumption) {
				*b = append(*b, BridgeAssumption{Path: "runtime/ghost.ts", Change: BridgeAdded, Reason: "x"})
			},
			wantErr: "orphan assumption",
		},
		{
			name: "wrong classification rejected",
			mutate: func(_ *SealIdentity, _ *Manifest, b *[]BridgeAssumption) {
				(*b)[2].Change = BridgeChanged
			},
			wantErr: "actual delta",
		},
		{
			name: "empty reason rejected",
			mutate: func(_ *SealIdentity, _ *Manifest, b *[]BridgeAssumption) {
				(*b)[0].Reason = "  "
			},
			wantErr: "no reason",
		},
		{
			name: "duplicate bridge path rejected",
			mutate: func(_ *SealIdentity, _ *Manifest, b *[]BridgeAssumption) {
				*b = append(*b, (*b)[0])
			},
			wantErr: "duplicated",
		},
		{
			name: "empty predecessor pin rejected",
			mutate: func(p *SealIdentity, _ *Manifest, _ *[]BridgeAssumption) {
				p.BundleFiles = map[string]string{}
			},
			wantErr: "predecessor pin holds no",
		},
		{
			name: "empty candidate rejected",
			mutate: func(_ *SealIdentity, c *Manifest, _ *[]BridgeAssumption) {
				c.BundleFiles = map[string]string{}
			},
			wantErr: "candidate holds no",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := refreshPromotionFixture()
			predecessor, candidate, bridge := p.Proposal.Predecessor, p.Candidate, p.Proposal.Bridge
			// Deep-copy the maps so mutations stay within this case.
			predecessor.BundleFiles = copyDigestMap(predecessor.BundleFiles)
			candidate.BundleFiles = copyDigestMap(candidate.BundleFiles)
			bridge = append([]BridgeAssumption(nil), bridge...)
			tc.mutate(&predecessor, &candidate, &bridge)
			err := ComparePredecessorCompatible(predecessor, candidate, bridge)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("accepted refresh rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

func copyDigestMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestVerifyPredecessorPin(t *testing.T) {
	p := refreshPromotionFixture()
	cases := []struct {
		name    string
		mutate  func(*Manifest)
		wantErr string
	}{
		{name: "exact pin accepted", mutate: func(*Manifest) {}},
		{
			name:    "version drift rejected",
			mutate:  func(m *Manifest) { m.Version = "r-seed-2" },
			wantErr: "drifts from pinned",
		},
		{
			name:    "digest drift rejected",
			mutate:  func(m *Manifest) { m.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64) },
			wantErr: "drifts from pinned digest",
		},
		{
			name:    "dropped entry rejected",
			mutate:  func(m *Manifest) { delete(m.BundleFiles, "runtime/b.ts") },
			wantErr: "holds 2 files, pinned holds 3",
		},
		{
			name:    "extra entry rejected",
			mutate:  func(m *Manifest) { m.BundleFiles["runtime/extra.ts"] = refreshFileNew },
			wantErr: "holds 4 files, pinned holds 3",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			live := p.LivePredecessor
			live.BundleFiles = copyDigestMap(live.BundleFiles)
			tc.mutate(&live)
			err := VerifyPredecessorPin(live, p.Proposal.Predecessor)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("exact pin rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

func TestDigestSuiteDeterministic(t *testing.T) {
	a := DigestSuite(map[string]string{"s/a.can": refreshFileA, "s/b.can": refreshFileB})
	b := DigestSuite(map[string]string{"s/b.can": refreshFileB, "s/a.can": refreshFileA})
	if a != b {
		t.Fatalf("suite digest order-dependent: %v vs %v", a, b)
	}
	if a.Files != 2 || len(a.Digest) != 64 {
		t.Fatalf("suite pin = %+v", a)
	}
	edited := DigestSuite(map[string]string{"s/a.can": refreshFileAV2, "s/b.can": refreshFileB})
	if edited == a {
		t.Fatal("provisional suite edit keeps the pinned digest")
	}
	if err := VerifySuiteFrozen(a, edited); err == nil {
		t.Fatal("edited suite accepted as frozen")
	} else if !strings.Contains(err.Error(), "suite rewrite rejected") {
		t.Fatalf("err = %v, want suite rewrite fault", err)
	} else {
		t.Logf("detected: %v", err)
	}
	removed := DigestSuite(map[string]string{"s/a.can": refreshFileA})
	if err := VerifySuiteFrozen(a, removed); err == nil {
		t.Fatal("shrunk suite accepted as frozen")
	}
	if err := VerifySuiteFrozen(a, b); err != nil {
		t.Fatalf("identical suite rejected: %v", err)
	}
}

func TestVerifyRollback(t *testing.T) {
	p := refreshPromotionFixture()
	cases := []struct {
		name    string
		mutate  func(*RefreshProposal, *Manifest)
		wantErr string
	}{
		{name: "exact rollback accepted", mutate: func(*RefreshProposal, *Manifest) {}},
		{
			name: "rollback digest drift rejected",
			mutate: func(proposal *RefreshProposal, _ *Manifest) {
				proposal.Rollback.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "drifts from predecessor digest",
		},
		{
			name: "rollback version drift rejected",
			mutate: func(proposal *RefreshProposal, _ *Manifest) {
				proposal.Rollback.Version = "r-seed-2"
			},
			wantErr: "is not the predecessor",
		},
		{
			name: "restored seed drift rejected",
			mutate: func(_ *RefreshProposal, live *Manifest) {
				live.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "does not match rollback target",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proposal := p.Proposal
			proposal.Rollback.BundleFiles = copyDigestMap(proposal.Rollback.BundleFiles)
			proposal.Predecessor.BundleFiles = copyDigestMap(proposal.Predecessor.BundleFiles)
			live := p.LivePredecessor
			live.BundleFiles = copyDigestMap(live.BundleFiles)
			tc.mutate(&proposal, &live)
			err := VerifyRollback(proposal, live)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("exact rollback rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

func TestVerifyRefreshProducer(t *testing.T) {
	seed := Producer{Path: "/seed/r2/bin/canlc", SHA256: refreshSeedDigest}
	candidate := Producer{Path: "/tmp/candidate/canlc", SHA256: refreshCandDigest}
	cases := []struct {
		name     string
		producer Producer
		wantErr  string
	}{
		{name: "refresh seed accepted", producer: seed},
		{name: "candidate rejected as self-promotion", producer: candidate, wantErr: "self-promotion"},
		{
			name:     "unknown binary rejected",
			producer: Producer{Path: "/tmp/other/canlc", SHA256: refreshSeedDigest},
			wantErr:  "untrusted",
		},
		{
			name:     "right path wrong digest rejected",
			producer: Producer{Path: seed.Path, SHA256: refreshCandDigest},
			wantErr:  "untrusted",
		},
		{
			name:     "bare name rejected",
			producer: Producer{Path: "canlc", SHA256: refreshSeedDigest},
			wantErr:  "non-absolute",
		},
		{
			name:     "malformed digest rejected",
			producer: Producer{Path: seed.Path, SHA256: "not-hex"},
			wantErr:  "not a SHA-256",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyRefreshProducer(tc.producer, seed, candidate)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("refresh seed rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

func TestCheckRefreshObservation(t *testing.T) {
	seed := Producer{Path: "/seed/r2/bin/canlc", SHA256: refreshSeedDigest}
	candidate := Producer{Path: "/tmp/candidate/canlc", SHA256: refreshCandDigest}
	fixed := FixedObservation{Name: "new.binding", Stdout: "ok\n", Stderr: "", Exit: 0}
	good := ObservedResult{Stdout: []byte("ok\n"), Stderr: []byte{}, Exit: 0, Producer: seed}
	cases := []struct {
		name    string
		got     ObservedResult
		wantErr string
	}{
		{name: "exact observation accepted", got: good},
		{
			name:    "exit drift rejected",
			got:     ObservedResult{Stdout: []byte("ok\n"), Stderr: []byte{}, Exit: 1, Producer: seed},
			wantErr: "exit 1, want 0",
		},
		{
			name:    "stdout drift rejected",
			got:     ObservedResult{Stdout: []byte("nope\n"), Stderr: []byte{}, Exit: 0, Producer: seed},
			wantErr: "stdout",
		},
		{
			name:    "stderr drift rejected",
			got:     ObservedResult{Stdout: []byte("ok\n"), Stderr: []byte("warn\n"), Exit: 0, Producer: seed},
			wantErr: "stderr",
		},
		{
			name:    "candidate producer rejected",
			got:     ObservedResult{Stdout: []byte("ok\n"), Stderr: []byte{}, Exit: 0, Producer: candidate},
			wantErr: "self-promotion",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckRefreshObservation(fixed, tc.got, seed, candidate)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("exact observation rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

func TestPromote(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Promotion)
		wantErr string
	}{
		{name: "declared refresh promotes", mutate: func(*Promotion) {}},
		{
			name:    "wrong proposal kind rejected",
			mutate:  func(p *Promotion) { p.Proposal.Kind = "can.native-test.seed" },
			wantErr: "proposal kind",
		},
		{
			name:    "missing fixed observations rejected",
			mutate:  func(p *Promotion) { p.Proposal.Fixed = nil },
			wantErr: "no fixed new-feature observations",
		},
		{
			name:    "seed-is-candidate lane rejected",
			mutate:  func(p *Promotion) { p.RefreshSeed = p.CandidateCompiler },
			wantErr: "refresh seed is the candidate",
		},
		{
			name: "live predecessor drift rejected",
			mutate: func(p *Promotion) {
				p.LivePredecessor.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "drifts from pinned digest",
		},
		{
			name: "undeclared candidate drift rejected",
			mutate: func(p *Promotion) {
				p.Candidate.BundleFiles["runtime/b.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "undeclared drift",
		},
		{
			name: "rollback drift rejected",
			mutate: func(p *Promotion) {
				p.Proposal.Rollback.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64)
			},
			wantErr: "drifts from predecessor digest",
		},
		{
			name: "provisional suite edit rejected",
			mutate: func(p *Promotion) {
				p.Suite = DigestSuite(map[string]string{"suite/one.can": refreshFileAV2, "suite/two.can": refreshFileB})
			},
			wantErr: "suite rewrite rejected",
		},
		{
			name: "prior acceptance regression rejected",
			mutate: func(p *Promotion) {
				got := p.PriorResults["pass.parse"]
				got.Stdout = []byte("changed\n")
				p.PriorResults["pass.parse"] = got
			},
			wantErr: "regressed",
		},
		{
			name: "prior re-observed by candidate rejected",
			mutate: func(p *Promotion) {
				got := p.PriorResults["pass.parse"]
				got.Producer = p.CandidateCompiler
				p.PriorResults["pass.parse"] = got
			},
			wantErr: "self-promotion",
		},
		{
			name: "prior re-observed by unknown binary rejected",
			mutate: func(p *Promotion) {
				got := p.PriorResults["pass.parse"]
				got.Producer = Producer{Path: "/tmp/other/canlc", SHA256: refreshCandDigest}
				p.PriorResults["pass.parse"] = got
			},
			wantErr: "no longer produced by the predecessor",
		},
		{
			name: "missing prior re-observation rejected",
			mutate: func(p *Promotion) {
				delete(p.PriorResults, "pass.parse")
			},
			wantErr: "not re-observed",
		},
		{
			name: "new observation rewriting prior name rejected",
			mutate: func(p *Promotion) {
				p.Proposal.Fixed = []FixedObservation{{Name: "pass.parse", Stdout: "ok\n"}}
				p.NewResults = map[string]ObservedResult{
					"pass.parse": {Stdout: []byte("ok\n"), Stderr: []byte{}, Exit: 0, Producer: p.RefreshSeed},
				}
			},
			wantErr: "rewrites predecessor acceptance",
		},
		{
			name: "missing new observation rejected",
			mutate: func(p *Promotion) {
				delete(p.NewResults, "new.binding")
			},
			wantErr: "want 1 fixed records",
		},
		{
			name: "candidate-produced new feature rejected",
			mutate: func(p *Promotion) {
				got := p.NewResults["new.binding"]
				got.Producer = p.CandidateCompiler
				p.NewResults["new.binding"] = got
			},
			wantErr: "self-promotion",
		},
		{
			name: "wrong new-feature output rejected",
			mutate: func(p *Promotion) {
				got := p.NewResults["new.binding"]
				got.Stdout = []byte("nope\n")
				p.NewResults["new.binding"] = got
			},
			wantErr: "wrong observation",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := refreshPromotionFixture()
			tc.mutate(&p)
			err := Promote(p)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("declared refresh rejected: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				t.Logf("detected: %v", err)
			}
		})
	}
}

// TestPromoteRejectsSelfPromotion is the acceptance control for the first P06
// clause: a new binding/syntax observed through the candidate compiler —
// byte-identical output, wrong producer — must not self-promote.
func TestPromoteRejectsSelfPromotion(t *testing.T) {
	p := refreshPromotionFixture()
	got := p.NewResults["new.binding"]
	got.Producer = p.CandidateCompiler
	p.NewResults["new.binding"] = got
	err := Promote(p)
	if err == nil {
		t.Fatal("candidate-produced new feature promoted")
	}
	if !strings.Contains(err.Error(), "self-promotion") {
		t.Fatalf("err = %v, want self-promotion fault", err)
	}
	t.Logf("detected: %v", err)
}

// TestPromoteRejectsSuiteRewrite is the acceptance control for the second P06
// clause: a provisional suite edit (same promotion, moved suite digest)
// must not rewrite R acceptance.
func TestPromoteRejectsSuiteRewrite(t *testing.T) {
	p := refreshPromotionFixture()
	p.Suite = DigestSuite(map[string]string{"suite/one.can": refreshFileAV2, "suite/two.can": refreshFileB})
	err := Promote(p)
	if err == nil {
		t.Fatal("promotion over an edited suite accepted")
	}
	if !strings.Contains(err.Error(), "suite rewrite rejected") {
		t.Fatalf("err = %v, want suite rewrite fault", err)
	}
	t.Logf("detected: %v", err)
}

// TestRefreshRollbackRestoresPredecessor is the simulated rollback control: a
// live seed matching the pinned rollback target verifies, and a seed with
// one drifted byte does not.
func TestRefreshRollbackRestoresPredecessor(t *testing.T) {
	p := refreshPromotionFixture()
	if err := VerifyRollback(p.Proposal, p.LivePredecessor); err != nil {
		t.Fatalf("exact rollback rejected: %v", err)
	}
	drifted := p.LivePredecessor
	drifted.BundleFiles = copyDigestMap(drifted.BundleFiles)
	drifted.BundleFiles["runtime/a.ts"] = strings.Repeat("e", 64)
	if err := VerifyRollback(p.Proposal, drifted); err == nil {
		t.Fatal("drifted rollback accepted")
	} else {
		t.Logf("detected: %v", err)
	}
}

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	// Sealed R1 field shape: the loader must understand the P04 seal layout.
	doc := map[string]any{
		"schema_version": 1,
		"kind":           "can.native-test.seed",
		"version":        "r-seed-1",
		"seed_root":      "/seed/r1",
		"selection":      map[string]any{},
		"bundle_files":   map[string]any{"bin/canlc": refreshPredDigest},
		"go_version":     "go1",
		"host":           "h",
		"owner_uid":      1,
		"owner_gid":      2,
		"staged_utc":     "1970-01-01T00:00:00Z",
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "manifest.json")
	writeFile(t, path, string(encoded))
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Kind != "can.native-test.seed" || m.BundleFiles["bin/canlc"] != refreshPredDigest {
		t.Fatalf("manifest = %+v", m)
	}
	if _, err := LoadManifest(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("missing manifest loaded")
	}
	writeFile(t, filepath.Join(dir, "bad.json"), "{not json")
	if _, err := LoadManifest(filepath.Join(dir, "bad.json")); err == nil {
		t.Fatal("malformed manifest loaded")
	}
}

func TestLoadRefreshProposal(t *testing.T) {
	dir := t.TempDir()
	p := refreshPromotionFixture().Proposal
	encoded, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "refresh.json")
	writeFile(t, path, string(encoded))
	loaded, err := LoadRefreshProposal(path)
	if err != nil {
		t.Fatalf("LoadRefreshProposal: %v", err)
	}
	if loaded.Kind != RefreshKind || len(loaded.Fixed) != 1 || len(loaded.Bridge) != 3 {
		t.Fatalf("proposal = %+v", loaded)
	}
	p.Kind = "can.native-test.seed"
	encoded, err = json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "wrong-kind.json"), string(encoded))
	if _, err := LoadRefreshProposal(filepath.Join(dir, "wrong-kind.json")); err == nil {
		t.Fatal("wrong-kind proposal loaded")
	}
	if _, err := LoadRefreshProposal(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("missing proposal loaded")
	}
}

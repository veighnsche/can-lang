// Capability refresh (R promotion lane) for the native-test reference seed.
//
// The initial seal (seal.go) fixes R1 forever: mutation after selection
// invalidates. A capability refresh — a new binding or syntax the judge must
// execute — therefore never edits R1 in place. It travels this separate
// promotion lane, which promotes a refresh-build manifest to the new R only
// when every gate below holds:
//
//   - Predecessor pin: the live predecessor seed still matches its pinned
//     seal identity exactly (kind, version, every bundle digest).
//   - Predecessor-compatible comparison: the candidate manifest keeps the
//     predecessor kind, advances the version, and every bundle delta
//     (added, changed, removed) is covered by exactly one explicit bridge
//     assumption. Undeclared drift is rejected.
//   - Independent new-feature observations: each fixed new-feature record
//     matches the refresh seed's actual output byte-for-byte, and the
//     producer of every observation is the refresh seed binary itself.
//     Anything produced by the candidate compiler is self-promotion and is
//     rejected: a new binding/syntax cannot qualify itself via C.
//   - Suite freeze: the suite digest at promotion time must equal the pinned
//     suite digest, and predecessor acceptance observations must still match
//     byte-for-byte under names the refresh does not reuse. Provisional
//     suite edits cannot rewrite R acceptance.
//   - Rollback identity: the rollback target is pinned like a seal and must
//     equal the predecessor identity exactly, so a failed refresh restores
//     the exact predecessor bytes.
//
// This package only verifies promotion inputs; it never builds compilers,
// never stages seeds, and never runs the candidate.
package reference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// RefreshKind is the only accepted capability-refresh proposal kind.
	RefreshKind = "can.native-test.seed-refresh"
	// RefreshSchemaVersion is the only accepted refresh proposal version.
	RefreshSchemaVersion = 1
	// refreshCompilerEntry is the sealed compiler digest entry, mirroring
	// the P05 acceptance controls.
	refreshCompilerEntry = "bin/canlc"
)

// Bridge change classifications. Every bundle delta must carry exactly one.
const (
	BridgeAdded   = "added"
	BridgeChanged = "changed"
	BridgeRemoved = "removed"
)

// SealIdentity is the seal-like pinned identity of one sealed seed: exact
// kind, version and every bundle digest. It is the comparison baseline for
// predecessor checks and the rollback target shape.
type SealIdentity struct {
	Kind        string            `json:"kind"`
	Version     string            `json:"version"`
	BundleFiles map[string]string `json:"bundle_files"`
}

// SealIdentityOf projects a sealed Manifest to its pinned identity. The
// digest map is copied so later mutation of the manifest cannot move the pin.
func SealIdentityOf(m Manifest) SealIdentity {
	files := make(map[string]string, len(m.BundleFiles))
	for name, sum := range m.BundleFiles {
		files[name] = sum
	}
	return SealIdentity{Kind: m.Kind, Version: m.Version, BundleFiles: files}
}

// BridgeAssumption is one explicit assumption covering one bundle delta
// between the predecessor seal and the refresh candidate. Deltas without an
// assumption are undeclared drift; assumptions without a delta are orphans.
// Both are rejected.
type BridgeAssumption struct {
	// Path is the bundle-relative file the delta touches.
	Path string `json:"path"`
	// Change is one of BridgeAdded, BridgeChanged, BridgeRemoved.
	Change string `json:"change"`
	// Reason is the mandatory human rationale for accepting this delta.
	Reason string `json:"reason"`
}

// SuitePin freezes the suite sources R acceptance was recorded against: a
// digest over the sorted per-file hashes plus the file count. Any
// provisional edit moves the digest and fails promotion.
type SuitePin struct {
	Digest string `json:"digest"`
	Files  int    `json:"files"`
}

// DigestSuite pins suite sources given as path -> SHA-256 hex. The digest is
// the SHA-256 over lines of "path\x00sha256\x00" in sorted path order, so it
// is deterministic regardless of map iteration order.
func DigestSuite(files map[string]string) SuitePin {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write([]byte(files[name]))
		h.Write([]byte{0})
	}
	return SuitePin{Digest: hex.EncodeToString(h.Sum(nil)), Files: len(names)}
}

// Producer identifies the binary that produced an observation.
type Producer struct {
	// Path is the absolute path of the producing binary. Bare names are
	// rejected so a PATH entry or candidate can never qualify silently.
	Path string `json:"path"`
	// SHA256 is the hex digest of the producing binary.
	SHA256 string `json:"sha256"`
}

// FixedObservation is one pinned new-feature record the refresh must
// reproduce exactly.
type FixedObservation struct {
	Name   string `json:"name"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Exit   int    `json:"exit"`
}

// ObservedResult is one actual observation plus the binary that produced it.
type ObservedResult struct {
	Stdout   []byte
	Stderr   []byte
	Exit     int
	Producer Producer
}

// RefreshProposal is the separate promotion lane input: everything a
// capability refresh needs beyond the two manifests, pinned up front.
type RefreshProposal struct {
	SchemaVersion int                `json:"schema_version"`
	Kind          string             `json:"kind"`
	Predecessor   SealIdentity       `json:"predecessor"`
	Suite         SuitePin           `json:"suite"`
	Bridge        []BridgeAssumption `json:"bridge"`
	Fixed         []FixedObservation `json:"fixed_observations"`
	Rollback      SealIdentity       `json:"rollback"`
}

// Promotion is one simulated or live refresh promotion attempt: the pinned
// proposal plus the live inputs promotion verifies against it.
type Promotion struct {
	Proposal RefreshProposal
	// LivePredecessor is the seal read from the staged predecessor seed.
	LivePredecessor Manifest
	// Candidate is the refresh-build manifest awaiting promotion.
	Candidate Manifest
	// RefreshSeed identifies the refresh seed binary: the only valid
	// producer of new-feature observations.
	RefreshSeed Producer
	// CandidateCompiler identifies the candidate under test: it must never
	// produce a refresh observation.
	CandidateCompiler Producer
	// NewResults maps fixed new-feature observation name -> actual result.
	NewResults map[string]ObservedResult
	// PriorFixed and PriorResults are the predecessor acceptance records
	// and their re-observed results: the regression gate.
	PriorFixed   []FixedObservation
	PriorResults map[string]ObservedResult
	// Suite is the suite digest observed at promotion time.
	Suite SuitePin
}

// LoadManifest reads a sealed seed manifest (predecessor seal or refresh
// candidate) from path.
func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("reference: read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("reference: decode manifest %s: %w", path, err)
	}
	return m, nil
}

// LoadRefreshProposal reads a refresh proposal from path and validates its
// kind and schema version.
func LoadRefreshProposal(path string) (RefreshProposal, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RefreshProposal{}, fmt.Errorf("reference: read refresh proposal %s: %w", path, err)
	}
	var p RefreshProposal
	if err := json.Unmarshal(raw, &p); err != nil {
		return RefreshProposal{}, fmt.Errorf("reference: decode refresh proposal %s: %w", path, err)
	}
	if err := validateProposalHeader(p); err != nil {
		return RefreshProposal{}, err
	}
	return p, nil
}

func validateProposalHeader(p RefreshProposal) error {
	if p.Kind != RefreshKind {
		return fmt.Errorf("reference: refresh proposal kind %q, want %q", p.Kind, RefreshKind)
	}
	if p.SchemaVersion != RefreshSchemaVersion {
		return fmt.Errorf("reference: refresh proposal schema %d, want %d", p.SchemaVersion, RefreshSchemaVersion)
	}
	return nil
}

// BundleDelta is the file-level difference between the predecessor seal and
// the refresh candidate. Paths are bundle-relative and sorted.
type BundleDelta struct {
	Added   []string
	Changed []string
	Removed []string
}

// Empty reports whether the delta touches no files.
func (d BundleDelta) Empty() bool {
	return len(d.Added)+len(d.Changed)+len(d.Removed) == 0
}

// DiffBundle compares the candidate manifest against the predecessor seal
// identity entry by entry.
func DiffBundle(predecessor SealIdentity, candidate Manifest) BundleDelta {
	var delta BundleDelta
	for name, want := range predecessor.BundleFiles {
		got, ok := candidate.BundleFiles[name]
		if !ok {
			delta.Removed = append(delta.Removed, name)
			continue
		}
		if got != want {
			delta.Changed = append(delta.Changed, name)
		}
	}
	for name := range candidate.BundleFiles {
		if _, ok := predecessor.BundleFiles[name]; !ok {
			delta.Added = append(delta.Added, name)
		}
	}
	sort.Strings(delta.Added)
	sort.Strings(delta.Changed)
	sort.Strings(delta.Removed)
	return delta
}

// VerifyPredecessorPin requires the live predecessor seal to equal its pinned
// identity on kind, version and every bundle digest, mirroring the P05
// committed-anchor semantics: without this anchor a tampered predecessor
// could attest its own refresh.
func VerifyPredecessorPin(live Manifest, pinned SealIdentity) error {
	if live.Kind != pinned.Kind || live.Version != pinned.Version {
		return fmt.Errorf("reference: live predecessor %s/%s drifts from pinned %s/%s",
			live.Kind, live.Version, pinned.Kind, pinned.Version)
	}
	if len(live.BundleFiles) != len(pinned.BundleFiles) {
		return fmt.Errorf("reference: live predecessor holds %d files, pinned holds %d",
			len(live.BundleFiles), len(pinned.BundleFiles))
	}
	for name, want := range pinned.BundleFiles {
		got, ok := live.BundleFiles[name]
		if !ok {
			return fmt.Errorf("reference: live predecessor drops pinned entry %s", name)
		}
		if got != want {
			return fmt.Errorf("reference: live predecessor entry %s drifts from pinned digest", name)
		}
	}
	return nil
}

// ComparePredecessorCompatible verifies the refresh candidate against the
// predecessor seal: same kind, advanced version, and every bundle delta
// covered by exactly one explicit bridge assumption with a non-empty reason.
func ComparePredecessorCompatible(predecessor SealIdentity, candidate Manifest, bridge []BridgeAssumption) error {
	if len(predecessor.BundleFiles) == 0 {
		return fmt.Errorf("reference: predecessor pin holds no bundle files")
	}
	if len(candidate.BundleFiles) == 0 {
		return fmt.Errorf("reference: refresh candidate holds no bundle files")
	}
	if candidate.Kind != predecessor.Kind {
		return fmt.Errorf("reference: refresh candidate kind %q drifts from predecessor %q",
			candidate.Kind, predecessor.Kind)
	}
	if candidate.Version == "" {
		return fmt.Errorf("reference: refresh candidate has no version")
	}
	if candidate.Version == predecessor.Version {
		return fmt.Errorf("reference: refresh candidate version %q does not advance predecessor %q",
			candidate.Version, predecessor.Version)
	}
	delta := DiffBundle(predecessor, candidate)
	want := make(map[string]string, len(delta.Added)+len(delta.Changed)+len(delta.Removed))
	for _, name := range delta.Added {
		want[name] = BridgeAdded
	}
	for _, name := range delta.Changed {
		want[name] = BridgeChanged
	}
	for _, name := range delta.Removed {
		want[name] = BridgeRemoved
	}
	seen := make(map[string]bool, len(bridge))
	for i, assumption := range bridge {
		if assumption.Path == "" {
			return fmt.Errorf("reference: bridge assumption %d has no path", i)
		}
		if seen[assumption.Path] {
			return fmt.Errorf("reference: bridge assumption for %s is duplicated", assumption.Path)
		}
		seen[assumption.Path] = true
		change, ok := want[assumption.Path]
		if !ok {
			return fmt.Errorf("reference: bridge assumption for %s matches no predecessor delta (orphan assumption)", assumption.Path)
		}
		if assumption.Change != change {
			return fmt.Errorf("reference: bridge assumption for %s claims %q, actual delta is %q",
				assumption.Path, assumption.Change, change)
		}
		if strings.TrimSpace(assumption.Reason) == "" {
			return fmt.Errorf("reference: bridge assumption for %s carries no reason", assumption.Path)
		}
	}
	for name, change := range want {
		if !seen[name] {
			return fmt.Errorf("reference: predecessor delta %s (%s) has no bridge assumption (undeclared drift)", name, change)
		}
	}
	return nil
}

// VerifySuiteFrozen requires the suite observed at promotion time to equal
// the pinned suite digest exactly. Provisional suite edits move the digest
// and are rejected: they must never rewrite R acceptance.
func VerifySuiteFrozen(pin, got SuitePin) error {
	if got.Files != pin.Files {
		return fmt.Errorf("reference: suite rewrite rejected: %d files, pinned %d", got.Files, pin.Files)
	}
	if got.Digest != pin.Digest {
		return fmt.Errorf("reference: suite rewrite rejected: digest %s, pinned %s", got.Digest, pin.Digest)
	}
	return nil
}

// VerifyRollbackTarget requires the rollback target to be pinned exactly like
// the predecessor seal: same kind, version and every bundle digest. A
// rollback that restores anything but the exact predecessor bytes is
// rejected.
func VerifyRollbackTarget(rollback, predecessor SealIdentity) error {
	if rollback.Kind != predecessor.Kind || rollback.Version != predecessor.Version {
		return fmt.Errorf("reference: rollback target %s/%s is not the predecessor %s/%s",
			rollback.Kind, rollback.Version, predecessor.Kind, predecessor.Version)
	}
	if len(rollback.BundleFiles) != len(predecessor.BundleFiles) {
		return fmt.Errorf("reference: rollback target holds %d files, predecessor holds %d",
			len(rollback.BundleFiles), len(predecessor.BundleFiles))
	}
	for name, want := range predecessor.BundleFiles {
		got, ok := rollback.BundleFiles[name]
		if !ok {
			return fmt.Errorf("reference: rollback target drops predecessor entry %s", name)
		}
		if got != want {
			return fmt.Errorf("reference: rollback target entry %s drifts from predecessor digest", name)
		}
	}
	return nil
}

// VerifyRollback gates a rollback after a failed or retired refresh: the
// proposal's rollback target must equal the predecessor pin, and the live
// restored seed must match the rollback target exactly.
func VerifyRollback(proposal RefreshProposal, live Manifest) error {
	if err := VerifyRollbackTarget(proposal.Rollback, proposal.Predecessor); err != nil {
		return err
	}
	liveIdentity := SealIdentityOf(live)
	if err := VerifyRollbackTarget(proposal.Rollback, liveIdentity); err != nil {
		return fmt.Errorf("reference: restored seed does not match rollback target: %v", err)
	}
	return nil
}

func checkProducerWellFormed(producer Producer, what string) error {
	if !filepath.IsAbs(producer.Path) {
		return fmt.Errorf("reference: refusing non-absolute %s path %q", what, producer.Path)
	}
	if len(producer.SHA256) != 64 {
		return fmt.Errorf("reference: %s digest %q is not a SHA-256 hex string", what, producer.SHA256)
	}
	if _, err := hex.DecodeString(producer.SHA256); err != nil {
		return fmt.Errorf("reference: %s digest %q is not hex: %v", what, producer.SHA256, err)
	}
	return nil
}

// VerifyRefreshProducer gates one observation producer: it must be the
// refresh seed binary. A producer equal to the candidate compiler is
// self-promotion — the new binding/syntax qualifying itself via C — and is
// rejected; any other unexpected producer is rejected as untrusted.
func VerifyRefreshProducer(producer, refreshSeed, candidate Producer) error {
	if err := checkProducerWellFormed(producer, "observation producer"); err != nil {
		return err
	}
	if producer == candidate {
		return fmt.Errorf("reference: self-promotion rejected: observation produced by the candidate compiler %s", producer.Path)
	}
	if producer != refreshSeed {
		return fmt.Errorf("reference: untrusted observation producer %s: want refresh seed %s", producer.Path, refreshSeed.Path)
	}
	return nil
}

// CheckRefreshObservation compares one actual new-feature observation
// byte-for-byte against its fixed record (stdout, stderr, exit) and gates
// its producer through VerifyRefreshProducer.
func CheckRefreshObservation(fixed FixedObservation, got ObservedResult, refreshSeed, candidate Producer) error {
	if err := VerifyRefreshProducer(got.Producer, refreshSeed, candidate); err != nil {
		return fmt.Errorf("reference: wrong observation %s: %v", fixed.Name, err)
	}
	if got.Exit != fixed.Exit {
		return fmt.Errorf("reference: wrong observation %s: exit %d, want %d", fixed.Name, got.Exit, fixed.Exit)
	}
	if !bytes.Equal(got.Stdout, []byte(fixed.Stdout)) {
		return fmt.Errorf("reference: wrong observation %s stdout (%d bytes, want %d)",
			fixed.Name, len(got.Stdout), len(fixed.Stdout))
	}
	if !bytes.Equal(got.Stderr, []byte(fixed.Stderr)) {
		return fmt.Errorf("reference: wrong observation %s stderr (%d bytes, want %d)",
			fixed.Name, len(got.Stderr), len(fixed.Stderr))
	}
	return nil
}

// checkPriorObservation re-verifies one predecessor acceptance record: the
// result must still match byte-for-byte, and its producer digest must still
// be the predecessor compiler the seal pins. A drifted or re-produced prior
// observation means R acceptance was rewritten, not refreshed.
func checkPriorObservation(fixed FixedObservation, got ObservedResult, predecessor SealIdentity, candidate Producer) error {
	if !filepath.IsAbs(got.Producer.Path) {
		return fmt.Errorf("reference: prior observation %s: refusing non-absolute producer path %q", fixed.Name, got.Producer.Path)
	}
	if got.Producer == candidate {
		return fmt.Errorf("reference: prior observation %s: self-promotion rejected: re-observed by the candidate compiler", fixed.Name)
	}
	want, ok := predecessor.BundleFiles[refreshCompilerEntry]
	if !ok {
		return fmt.Errorf("reference: prior observation %s: predecessor seal lacks compiler entry %q", fixed.Name, refreshCompilerEntry)
	}
	if got.Producer.SHA256 != want {
		return fmt.Errorf("reference: prior observation %s no longer produced by the predecessor compiler", fixed.Name)
	}
	if got.Exit != fixed.Exit {
		return fmt.Errorf("reference: prior observation %s regressed: exit %d, want %d", fixed.Name, got.Exit, fixed.Exit)
	}
	if !bytes.Equal(got.Stdout, []byte(fixed.Stdout)) {
		return fmt.Errorf("reference: prior observation %s regressed: stdout drift", fixed.Name)
	}
	if !bytes.Equal(got.Stderr, []byte(fixed.Stderr)) {
		return fmt.Errorf("reference: prior observation %s regressed: stderr drift", fixed.Name)
	}
	return nil
}

// Promote verifies a full capability-refresh promotion attempt. Every gate
// must hold; the first failure rejects the promotion.
func Promote(p Promotion) error {
	if err := validateProposalHeader(p.Proposal); err != nil {
		return err
	}
	if len(p.Proposal.Fixed) == 0 {
		return fmt.Errorf("reference: refresh promotion carries no fixed new-feature observations")
	}
	// Lane producers must be well-formed and distinct: the refresh seed is
	// never the candidate under test.
	if err := checkProducerWellFormed(p.RefreshSeed, "refresh seed"); err != nil {
		return err
	}
	if err := checkProducerWellFormed(p.CandidateCompiler, "candidate compiler"); err != nil {
		return err
	}
	if p.RefreshSeed == p.CandidateCompiler {
		return fmt.Errorf("reference: refresh lane misconfigured: refresh seed is the candidate compiler")
	}
	// Predecessor anchor, compatibility, rollback and suite gates.
	if err := VerifyPredecessorPin(p.LivePredecessor, p.Proposal.Predecessor); err != nil {
		return err
	}
	if err := ComparePredecessorCompatible(p.Proposal.Predecessor, p.Candidate, p.Proposal.Bridge); err != nil {
		return err
	}
	if err := VerifyRollbackTarget(p.Proposal.Rollback, p.Proposal.Predecessor); err != nil {
		return err
	}
	if err := VerifySuiteFrozen(p.Proposal.Suite, p.Suite); err != nil {
		return err
	}
	// Regression gate: predecessor acceptance still holds, observed by R.
	priorNames := make(map[string]bool, len(p.PriorFixed))
	for _, fixed := range p.PriorFixed {
		if fixed.Name == "" {
			return fmt.Errorf("reference: prior observation has no name")
		}
		if priorNames[fixed.Name] {
			return fmt.Errorf("reference: prior observation %q is duplicated", fixed.Name)
		}
		priorNames[fixed.Name] = true
		got, ok := p.PriorResults[fixed.Name]
		if !ok {
			return fmt.Errorf("reference: prior observation %s was not re-observed", fixed.Name)
		}
		if err := checkPriorObservation(fixed, got, p.Proposal.Predecessor, p.CandidateCompiler); err != nil {
			return err
		}
	}
	// New-feature gate: every fixed record reproduced by the refresh seed,
	// under names that do not rewrite predecessor acceptance.
	if len(p.NewResults) != len(p.Proposal.Fixed) {
		return fmt.Errorf("reference: %d new-feature results, want %d fixed records",
			len(p.NewResults), len(p.Proposal.Fixed))
	}
	for _, fixed := range p.Proposal.Fixed {
		if fixed.Name == "" {
			return fmt.Errorf("reference: fixed new-feature observation has no name")
		}
		if priorNames[fixed.Name] {
			return fmt.Errorf("reference: new-feature observation %s rewrites predecessor acceptance", fixed.Name)
		}
		got, ok := p.NewResults[fixed.Name]
		if !ok {
			return fmt.Errorf("reference: new-feature observation %s was not observed", fixed.Name)
		}
		if err := CheckRefreshObservation(fixed, got, p.RefreshSeed, p.CandidateCompiler); err != nil {
			return err
		}
	}
	return nil
}

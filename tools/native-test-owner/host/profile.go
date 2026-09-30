package host

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ProfileSchemaVersion versions the host-profile acceptance record.
const ProfileSchemaVersion = 1

// EnvelopeProfile is the acceptance record for one envelope: mechanism
// table, overshoot bounds, disposal reserve and the full check table.
type EnvelopeProfile struct {
	Envelope   Envelope      `json:"envelope"`
	Mechanisms []Mechanism   `json:"mechanisms"`
	Overshoot  Overshoot     `json:"overshoot"`
	Checks     []CheckResult `json:"checks"`
	Valid      bool          `json:"valid"`
}

// NegativeControl records one must-fail adapter and the checks that
// failed it. Valid must be false: a passing negative invalidates the
// battery, not the adapter.
type NegativeControl struct {
	Name          string   `json:"name"`
	Adapter       string   `json:"adapter"`
	Valid         bool     `json:"valid"`
	FailingChecks []string `json:"failingChecks"`
}

// ProfileRecord is the host-profile acceptance data the integrator
// commits separately: per-envelope mechanism tables with bounds and
// reserve, host facts, negative-control outcomes, and the honest
// limitations. Tests emit it as JSON (see TestProfileArtifact); it is
// never written under docs/ by this package.
type ProfileRecord struct {
	SchemaVersion int               `json:"schemaVersion"`
	Host          HostFacts         `json:"host"`
	Adapter       string            `json:"adapter"`
	Envelopes     []EnvelopeProfile `json:"envelopes"`
	Negatives     []NegativeControl `json:"negatives"`
	Limitations   []string          `json:"limitations"`
}

// ProfileLimitations states what the darwin-strict profile does not
// claim. Qualification scope: P14 binds N acceptance to this profile,
// and another host requires its own receipt.
func ProfileLimitations() []string {
	return []string{
		"memory is declared-peak ledger only: RLIMIT_AS and RLIMIT_DATA are unsettable on darwin arm64 (EINVAL, shared-cache mapping), so no kernel RSS backstop exists; runtime RSS beyond declaration is bounded only by host commit (hw.memsize fit demonstrated at admission)",
		"tmp bypass bytes between decision-point reconciles are bounded physically by probed free disk, not by the envelope; no admission or disposal verdict is issued on unreconciled state, and overflow past the max latches fail-closed until disposal",
		"NPROC fit uses the soft limit at probe time; Admit re-probes fresh facts and refuses on drift, but a limit lowered mid-run is detected only at the next admission",
		"darwin-only: off darwin the adapter probes unavailable and every envelope stays blocked",
	}
}

// BuildProfile assembles the acceptance record from qualification
// outcomes. quals holds one qualification per admitted envelope;
// negatives holds name plus qualification per must-fail adapter.
func BuildProfile(adapter string, facts HostFacts, quals []*Qualification, negatives map[string]*Qualification) *ProfileRecord {
	r := &ProfileRecord{
		SchemaVersion: ProfileSchemaVersion,
		Host:          facts,
		Adapter:       adapter,
		Limitations:   ProfileLimitations(),
	}
	for _, q := range quals {
		if q == nil {
			continue
		}
		r.Envelopes = append(r.Envelopes, EnvelopeProfile{
			Envelope: q.Envelope, Mechanisms: q.Mechanisms,
			Overshoot: q.Overshoot, Checks: q.Checks, Valid: q.Valid(),
		})
	}
	names := make([]string, 0, len(negatives))
	for name := range negatives {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		q := negatives[name]
		nc := NegativeControl{Name: name, Valid: q.Valid(), FailingChecks: q.Failing()}
		if q != nil {
			nc.Adapter = q.AdapterName
		}
		r.Negatives = append(r.Negatives, nc)
	}
	return r
}

// JSON renders the record for the test-emitted artifact.
func (r *ProfileRecord) JSON() string {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"schemaVersion":%d,"error":"encode failed"}`, ProfileSchemaVersion)
	}
	return string(raw)
}

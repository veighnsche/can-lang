package acceptance

import (
	"errors"
	"fmt"

	"github.com/veighnsche/can-lang/tools/native-test-owner/host"
)

// Host selection errors. Callers distinguish them with errors.Is.
var (
	// ErrHostNotQualified reports a missing or invalid qualification.
	ErrHostNotQualified = errors.New("acceptance: host not qualified")
	// ErrHostDrift reports re-probed facts that no longer match the
	// exact qualified host profile.
	ErrHostDrift = errors.New("acceptance: host facts drifted from qualification")
	// ErrForeignHost reports an explicitly selected non-local host:
	// execution stays put and that host needs its own receipt.
	ErrForeignHost = errors.New("acceptance: foreign host requires explicit selection and its own receipt")
)

// SelectedHost is one explicitly selected host profile. A local
// selection carries its valid qualification plus freshly re-probed
// facts; a foreign selection carries only the explicit record and can
// never execute locally.
type SelectedHost struct {
	AdapterName string
	Facts       host.HostFacts
	Envelope    host.Envelope
	Foreign     bool

	qual *host.Qualification
}

// Select binds the exact qualified host profile: the qualification must
// be valid, the adapter must be the qualified one, enforcement must
// still probe available, and the freshly re-probed stable facts must
// equal the qualified ones. Tmp free disk is checked by envelope fit,
// not by equality, because it legitimately moves between probe and
// selection. Any drift or unavailability refuses.
func Select(a host.Adapter, q *host.Qualification, tmpParent string) (*SelectedHost, error) {
	if q == nil || !q.Valid() {
		return nil, fmt.Errorf("%w: acceptance needs a valid qualification", ErrHostNotQualified)
	}
	if a == nil || a.Name() != q.AdapterName {
		got := "<nil>"
		if a != nil {
			got = a.Name()
		}
		return nil, fmt.Errorf("%w: adapter %q is not the qualified %q",
			ErrHostNotQualified, got, q.AdapterName)
	}
	caps := a.Probe(tmpParent)
	if !caps.Available {
		return nil, fmt.Errorf("%w: %s", host.ErrUnavailable, caps.Detail)
	}
	if err := checkDrift(q.Facts, caps.Facts, q.Envelope); err != nil {
		return nil, err
	}
	return &SelectedHost{
		AdapterName: a.Name(),
		Facts:       caps.Facts,
		Envelope:    q.Envelope,
		qual:        q,
	}, nil
}

// SelectOther explicitly records another host profile. The record is
// explicit by construction — name, facts and envelope travel together —
// and Accept refuses to execute it locally: that host needs its own
// selection and its own receipt where it runs.
func SelectOther(name string, facts host.HostFacts, env host.Envelope) (*SelectedHost, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: another host needs an explicit name", ErrForeignHost)
	}
	return &SelectedHost{AdapterName: name, Facts: facts, Envelope: env, Foreign: true}, nil
}

// checkDrift compares the stable fitted dimensions exactly. Tmp free
// disk only needs to keep fitting the envelope.
func checkDrift(qualified, fresh host.HostFacts, env host.Envelope) error {
	switch {
	case fresh.GOOS != qualified.GOOS:
		return fmt.Errorf("%w: GOOS %q, qualified %q", ErrHostDrift, fresh.GOOS, qualified.GOOS)
	case fresh.NProcSoft != qualified.NProcSoft || fresh.NProcHard != qualified.NProcHard:
		return fmt.Errorf("%w: NPROC %d/%d, qualified %d/%d",
			ErrHostDrift, fresh.NProcSoft, fresh.NProcHard, qualified.NProcSoft, qualified.NProcHard)
	case fresh.AddrSpaceCapped != qualified.AddrSpaceCapped:
		return fmt.Errorf("%w: AddrSpaceCapped %v, qualified %v",
			ErrHostDrift, fresh.AddrSpaceCapped, qualified.AddrSpaceCapped)
	case fresh.FileSizeCapped != qualified.FileSizeCapped:
		return fmt.Errorf("%w: FileSizeCapped %v, qualified %v",
			ErrHostDrift, fresh.FileSizeCapped, qualified.FileSizeCapped)
	case fresh.MemBytes != qualified.MemBytes:
		return fmt.Errorf("%w: mem %d, qualified %d", ErrHostDrift, fresh.MemBytes, qualified.MemBytes)
	case uint64(env.MaxTmpBytes) > fresh.TmpFreeBytes:
		return fmt.Errorf("%w: tmp free %d no longer fits envelope %d",
			ErrHostDrift, fresh.TmpFreeBytes, env.MaxTmpBytes)
	}
	return nil
}

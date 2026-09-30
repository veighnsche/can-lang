package host

import (
	"encoding/json"
	"runtime"
	"testing"
)

// TestProfileArtifact emits the host-profile acceptance record as JSON:
// per-envelope mechanism tables with overshoot bounds and disposal
// reserve, host facts, negative-control outcomes and limitations. The
// integrator commits this record separately; the package writes
// nothing under docs/ itself. Run with -v to read the artifact.
func TestProfileArtifact(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only profile")
	}
	tmp := t.TempDir()
	qQuick, err := Qualify(Darwin(), Quick(), QualifyOpts{TmpParent: tmp})
	if err != nil || !qQuick.Valid() {
		t.Fatalf("quick did not qualify: %v %v", err, qQuick.Failing())
	}
	qBounded, err := Qualify(Darwin(), BoundedJob(), QualifyOpts{TmpParent: tmp})
	if err != nil || !qBounded.Valid() {
		t.Fatalf("bounded-job did not qualify: %v %v", err, qBounded.Failing())
	}
	negatives := map[string]*Qualification{}
	for name, a := range map[string]Adapter{
		"poll-and-kill": pollKillAdapter{},
		"sampled-peak":  samplingAdapter{},
		"lying-labels":  lyingAdapter{},
		"unavailable":   Unavailable("profile: no mechanism"),
	} {
		q, qerr := Qualify(a, Quick(), QualifyOpts{TmpParent: tmp})
		if qerr != nil {
			t.Fatalf("%s: battery did not run: %v", name, qerr)
		}
		if q.Valid() {
			t.Fatalf("%s: negative control qualified", name)
		}
		negatives[name] = q
	}
	rec := BuildProfile(Darwin().Name(), Darwin().Probe(tmp).Facts,
		[]*Qualification{qQuick, qBounded}, negatives)
	if rec.SchemaVersion != ProfileSchemaVersion {
		t.Fatalf("schema %d, want %d", rec.SchemaVersion, ProfileSchemaVersion)
	}
	if len(rec.Envelopes) != 2 || len(rec.Negatives) != 4 || len(rec.Limitations) == 0 {
		t.Fatalf("record shape wrong: %+v", rec)
	}
	for _, e := range rec.Envelopes {
		if !e.Valid {
			t.Fatalf("envelope %s invalid in record", e.Envelope.Name)
		}
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("record not JSON-clean: %v", err)
	}
	_ = raw
	t.Logf("HOST_PROFILE_ARTIFACT:\n%s", rec.JSON())
}

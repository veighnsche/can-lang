package check

import (
	"strings"
	"testing"
)

// C9.1: bound standard catches bind the opaque standard_failure snapshot.
// Snapshots project kind/message/occurrence_id, persist as data and aggregate
// leaves, but cannot be constructed, updated, wire-encoded, or emitted.
func TestStandardSnapshotContracts(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [codec, bytes]", 1)
	work := `fn int work
    emits []
    asserts
        sample: => ok 1
    ok 1
`
	t.Run("store and project", func(t *testing.T) {
		text := header + work + `fn str observed
    emits []
    asserts
        sample: => ok "none"
    match call work()
        [_] as standard_failure failure => do
            standard_failure kept = failure
            str kind = kept.kind
            str message = kept.message
            int id = kept.occurrence_id
            ok kind + message
        ok int value => ok "none"
` + programMain + "    ok\n"
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
			t.Fatalf("snapshot store/project rejected: %v", err)
		}
	})
	t.Run("aggregate leaf projects", func(t *testing.T) {
		text := header + work + `variant failure
    codec::invalid_data
    standard_failure
fn str leaf_kind
    emits []
    asserts
        sample: => ok "none"
    match call work()
        [_] as standard_failure failure => do
            failure observed = failure
            match observed
                codec::invalid_data => ok "domain"
                standard_failure => ok observed.kind
        ok int value => ok "none"
` + programMain + "    ok\n"
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
			t.Fatalf("aggregate leaf projection rejected: %v", err)
		}
	})
	for name, body := range map[string]string{
		"construct": `fn standard_failure make
    emits []
    asserts
        sample: => ok
    standard_failure()
`,
		"emits": `fn void leak
    emits [standard_failure]
    asserts
        sample: => ok
    ok
`,
		"wire": `fn int ship
    emits []
    asserts
        sample: => ok 0
    match call work()
        [_] as standard_failure failure => match call codec::encode_json<standard_failure>(failure)
            [_]
            ok bytes::buffer encoded => ok 0
        ok int value => ok value
`,
		"update": `fn str forge
    emits []
    asserts
        sample: => ok "none"
    match call work()
        [_] as standard_failure failure => do
            standard_failure rewritten = failure with (message = "forged")
            ok rewritten.message
        ok int value => ok "none"
`,
	} {
		t.Run(name, func(t *testing.T) {
			text := header + work + body + programMain + "    ok\n"
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatalf("snapshot %s admitted", name)
			}
		})
	}
}

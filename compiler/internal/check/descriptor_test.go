package check

import (
	"strings"
	"testing"
)

func descriptorElisionFixture() string {
	return "package app\n    provides []\n    uses [descriptor, test]\nfn void down\n    emits {test::stale_handle}\n    given\n        test::owner o\n        descriptor::launch l\n    asserts\n        sample: => ok\n    match call descriptor::kill_child(o, l)\n        when\n            sample: o, l => ok\n        test::stale_handle\n        ok => ok\n" + programMain + "    ok\n"
}

func TestDescriptorAssertionScopeElision(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": descriptorElisionFixture()}); err != nil {
		t.Fatalf("rejected elided owner/launch rows: %v", err)
	}
	original := descriptorElisionFixture()
	for _, tc := range []struct{ name, old, replacement string }{
		{"extra scope argument", "sample: => ok", "sample: \"x\" => ok"},
		{"supplied scope value", "sample: => ok", "sample: l => ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(original, tc.old, tc.replacement, 1)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted invalid scope row: %s", tc.name)
			}
		})
	}
}

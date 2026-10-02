package check

import (
	"strings"
	"testing"
)

// probe_lease takes the I04-admitted test owner plus a dynamic path:
// assert rows elide the owner and supply the path. The path stays
// dynamic (validated at runtime with descriptor_fault); only the
// owner is scope. The lease_report result reuses the I11 record
// shape, which stays constructible for expected vectors.

func descriptorProbeLeaseFixture() string {
	report := "descriptor::lease_report(\"/tmp/can-fd4.lease\", true, true, false)"
	return "package app\n    provides []\n    uses [descriptor, test]\nfn descriptor::lease_report probe\n    emits {test::stale_handle, descriptor::descriptor_fault}\n    given\n        test::owner o\n        str path\n    asserts\n        sample: \"/tmp/can-fd4.lease\" => ok " + report + "\n    match call descriptor::probe_lease(o, path)\n        when\n            sample: o, \"/tmp/can-fd4.lease\" => ok " + report + "\n        test::stale_handle\n        descriptor::descriptor_fault\n        ok descriptor::lease_report got => ok got\n" + programMain + "    ok\n"
}

func TestDescriptorProbeLeaseAdmits(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": descriptorProbeLeaseFixture()}); err != nil {
		t.Fatalf("rejected elided-owner probe row: %v", err)
	}
}

func TestDescriptorProbeLeaseRefusals(t *testing.T) {
	original := descriptorProbeLeaseFixture()
	for _, tc := range []struct{ name, old, replacement string }{
		{"supplied scope owner", "sample: \"/tmp/can-fd4.lease\" => ok", "sample: o, \"/tmp/can-fd4.lease\" => ok"},
		{"elided dynamic path", "sample: \"/tmp/can-fd4.lease\" => ok", "sample: => ok"},
		{"missing fault emission", "emits {test::stale_handle, descriptor::descriptor_fault}", "emits {test::stale_handle}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(original, tc.old, tc.replacement, 1)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatalf("accepted invalid probe row: %s", tc.name)
			}
		})
	}
}

package check

import (
	"strings"
	"testing"
)

func httpPeerElisionFixture() string {
	return "package app\n    provides []\n    uses [http_peer, test]\nfn http_peer::listener_close_receipt shut\n    emits {test::stale_handle, test::closed_handle, http_peer::peer_fault}\n    given\n        test::owner owner\n        http_peer::listener listener\n    asserts\n        drained: => ok http_peer::listener_close_receipt(2, 1, 0)\n        stale: => test::stale_handle{\"owner\"}\n    match call http_peer::close_listener(owner, listener)\n        when\n            drained: owner, listener => ok http_peer::listener_close_receipt(2, 1, 0)\n            stale: owner, listener => test::stale_handle{\"owner\"}\n        test::stale_handle\n        test::closed_handle\n        http_peer::peer_fault\n        ok http_peer::listener_close_receipt got => ok got\nfn void main\n    emits {}\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n"
}

func TestHttpPeerAssertionScopeElision(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": httpPeerElisionFixture()}); err != nil {
		t.Fatalf("rejected elided owner/listener rows: %v", err)
	}
	original := httpPeerElisionFixture()
	for _, tc := range []struct{ name, old, replacement string }{
		{"extra scope argument", "drained: => ok", "drained: \"x\" => ok"},
		{"supplied scope value", "drained: => ok", "drained: owner => ok"},
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

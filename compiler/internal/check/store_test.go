package check

import (
	"strings"
	"testing"
)

// The NT-I17 store operations carry no closed vocabulary: every
// K27 input (grant owners, prefixes, sessions, tokens, keys,
// payloads, write ids, limits, continuations) is dynamic and
// validated at runtime with precise store_fault failures. These
// fixtures pin admission itself: owner-first calls with
// bytes::buffer and option::value<str> inputs check, void
// closes check, the two pure ops emit stale_handle only, and
// missing fault arms plus mistyped inputs refuse exactly.

func storeOpenPrefixFixture(dropFault bool, grantOwner string) string {
	emits := "test::stale_handle, store::store_fault"
	arms := "        test::stale_handle\n        store::store_fault\n"
	if dropFault {
		emits = "test::stale_handle"
		arms = "        test::stale_handle\n"
	}
	stub := "store::prefix_receipt(\"tenant-a/owned\", \"owner-a\", \"prefix-handle:owner-a:tenant-a/owned\", \"sha256:stub\", 0)"
	return "package app\n    provides []\n    uses [store, test]\nfn store::prefix_receipt open\n    emits {" + emits + "}\n    given\n        test::owner o\n        str grant_owner\n        str prefix\n    asserts\n        first: " + grantOwner + ", \"tenant-a/owned\" => ok " + stub + "\n    match call store::open_prefix(o, grant_owner, prefix)\n        when\n            first: o, grant_owner, prefix => ok " + stub + "\n" + arms + "        ok store::prefix_receipt got => ok got\n" + programMain + "    ok\n"
}

func TestStoreOpenPrefixDynamicAdmits(t *testing.T) {
	fixture := storeOpenPrefixFixture(false, "\"owner-a\"")
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected dynamic store inputs: %v", err)
	}
}

func TestStoreMissingFaultArmRefused(t *testing.T) {
	_, err := programFixture(t, map[string]string{"src/main.can": storeOpenPrefixFixture(true, "\"owner-a\"")})
	if err == nil {
		t.Fatal("accepted store call missing its store_fault arm")
	}
	if !strings.Contains(err.Error(), "store::store_fault") {
		t.Fatalf("wrong refusal for missing store_fault arm: %v", err)
	}
}

func TestStoreWrongInputTypeRefused(t *testing.T) {
	_, err := programFixture(t, map[string]string{"src/main.can": storeOpenPrefixFixture(false, "7")})
	if err == nil {
		t.Fatal("accepted int store grant_owner")
	}
	if !strings.Contains(err.Error(), "does not fit expected type") {
		t.Fatalf("wrong refusal for int store grant_owner: %v", err)
	}
}

func TestStorePutBytesAdmits(t *testing.T) {
	stub := "store::write_facts(\"w1\", \"tenant-a/owned/a\", \"sha256:stub\", false)"
	fixture := "package app\n    provides []\n    uses [bytes, store, test]\nfn store::write_facts stage\n    emits {test::stale_handle, store::store_fault}\n    given\n        test::owner o\n        str grant_owner\n        str prefix\n        str session\n        str token\n        str key\n    asserts\n        first: \"owner-a\", \"tenant-a/owned\", \"s0\", \"tok-stub\", \"tenant-a/owned/a\" => ok " + stub + "\n    match chain\n        call bytes::empty() as bytes::buffer payload\n        call store::put(o, grant_owner, prefix, session, token, key, payload) as store::write_facts staged\n        test::stale_handle\n        store::store_fault\n        ok => ok staged\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected store put with bytes payload: %v", err)
	}
}

func TestStoreListOptionAdmits(t *testing.T) {
	stub := "store::page_facts(\"tenant-a/owned\", [], 0, true, option::none(), 0, \"sha256:stub\")"
	fixture := "package app\n    provides []\n    uses [option, store, test]\nfn store::page_facts scan\n    emits {test::stale_handle, store::store_fault}\n    given\n        test::owner o\n        str grant_owner\n        str prefix\n        str session\n        str token\n        int limit\n        option::value<str> continuation\n    asserts\n        first: \"owner-a\", \"tenant-a/owned\", \"s0\", \"tok-stub\", 8, option::none() => ok " + stub + "\n    match call store::list(o, grant_owner, prefix, session, token, limit, continuation)\n        when\n            first: o, grant_owner, prefix, session, token, limit, continuation => ok " + stub + "\n        test::stale_handle\n        store::store_fault\n        ok store::page_facts got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected store list with option continuation: %v", err)
	}
}

func TestStoreVoidCloseAdmits(t *testing.T) {
	fixture := "package app\n    provides []\n    uses [store, test]\nfn void shut\n    emits {test::stale_handle, store::store_fault}\n    given\n        test::owner o\n        str grant_owner\n        str prefix\n    asserts\n        first: \"owner-a\", \"tenant-a/owned\" => ok\n    match chain\n        call store::close_prefix(o, grant_owner, prefix)\n        test::stale_handle\n        store::store_fault\n        ok => ok\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": fixture}); err != nil {
		t.Fatalf("rejected void store close: %v", err)
	}
}

func TestStorePureOpsStaleOnlyEmits(t *testing.T) {
	compare := "package app\n    provides []\n    uses [bytes, store, test]\nfn bool equal\n    emits {test::stale_handle}\n    given\n        test::owner o\n    asserts\n        pair: => ok true\n    match chain\n        call bytes::empty() as bytes::buffer first\n        call bytes::empty() as bytes::buffer second\n        call store::compare_bytes(o, first, second) as bool out\n        test::stale_handle\n        ok => ok out\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": compare}); err != nil {
		t.Fatalf("rejected stale-only store::compare_bytes call: %v", err)
	}
	digest := "package app\n    provides []\n    uses [store, test]\nfn str foreign\n    emits {test::stale_handle}\n    given\n        test::owner o\n    asserts\n        first: => ok \"sha256:stub\"\n    match call store::shared_digest(o)\n        when\n            first: o => ok \"sha256:stub\"\n        test::stale_handle\n        ok str got => ok got\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": digest}); err != nil {
		t.Fatalf("rejected stale-only store::shared_digest call: %v", err)
	}
}

package emit

import "testing"

// A05: bulk builder identities resolve to the factory methods the
// runtime publishes; existing mappings are unchanged.
func TestCollectionMethodNameBulkBuilders(t *testing.T) {
	for identity, want := range map[string]string{
		"can.std.collections@1::build_map": "build_map",
		"can.std.collections@1::build_set": "build_set",
		"can.std.collections@1::insert":    "insert",
		"can.std.collections@1::add":       "add",
		"can.std.collections@1::empty_map": "empty",
		"can.std.collections@1::empty_set": "empty",
	} {
		if method := collectionMethodName(identity); method != want {
			t.Fatalf("%s mapped to %s, want %s", identity, method, want)
		}
	}
}

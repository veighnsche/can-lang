package catalogue

import (
	"reflect"
	"testing"
)

// TestFetchJSONCatalogueContract pins the T23 browser JSON fetch surface:
// the bodyless GET and wire-body POST operations, their static action name,
// their finite failure bounds, and the native fetch recipe. Per-action
// capture contracts are derived by the checker; the catalogue carries the
// shared failure vocabulary only.
func TestFetchJSONCatalogueContract(t *testing.T) {
	c := Builtin()
	get, err := c.Operation("http::fetch_json_get", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(get.Parameters) != 1 || get.Parameters[0].Name != "Result" || get.Result != "Result" {
		t.Fatalf("http::fetch_json_get descriptor differs: %+v", get)
	}
	if !reflect.DeepEqual(get.StaticInputs, []string{"action"}) {
		t.Fatalf("http::fetch_json_get static inputs differ: %+v", get.StaticInputs)
	}
	if !reflect.DeepEqual(get.Emits, []string{"http::transport_failed", "http::invalid_request", "http::status_error", "codec::invalid_data"}) {
		t.Fatalf("http::fetch_json_get bound differs: %+v", get.Emits)
	}
	post, err := c.Operation("http::fetch_json_post", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Parameters) != 2 || post.Parameters[0].Name != "Result" || post.Parameters[1].Name != "Wire" || post.Result != "Result" {
		t.Fatalf("http::fetch_json_post descriptor differs: %+v", post)
	}
	if !reflect.DeepEqual(post.StaticInputs, []string{"action"}) {
		t.Fatalf("http::fetch_json_post static inputs differ: %+v", post.StaticInputs)
	}
	if !reflect.DeepEqual(post.Emits, []string{"http::transport_failed", "http::invalid_request", "http::body_limit", "http::status_error", "codec::invalid_data"}) {
		t.Fatalf("http::fetch_json_post bound differs: %+v", post.Emits)
	}
	for _, op := range []Operation{get, post} {
		if op.Lowering.Task != "T23" || op.Assertion != "real" {
			t.Fatalf("%s evidence contract differs: %+v", op.Name, op.Lowering)
		}
		found := false
		for _, native := range op.Lowering.Native {
			if native == "fetch" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s names no fetch recipe: %+v", op.Name, op.Lowering.Native)
		}
		args := map[string]string{"Result": "str"}
		if op.Name == "http::fetch_json_post" {
			args["Wire"] = "str"
		}
		spec, err := c.Resolve(op.Name, c.Inventory().TargetID, c.Inventory().Revision, args, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(spec.Emits) != len(op.Emits) {
			t.Fatalf("%s specialization bound differs: %+v", op.Name, spec.Emits)
		}
	}
}

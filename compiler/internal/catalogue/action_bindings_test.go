package catalogue

import (
	"reflect"
	"testing"
)

// TestActionBindingsCatalogueContract pins the UP08 symbol-based action
// consumer surface: the mount binder, the URL builder and the bodyless GET
// and wire-body POST clients. The action operand is a checked declaration
// symbol resolved statically by the checker, never a string value; the
// catalogue carries the result shapes and the shared failure vocabulary,
// while per-action capture, body and callable contracts derive from the
// checked action table.
func TestActionBindingsCatalogueContract(t *testing.T) {
	c := Builtin()
	target, revision := c.Inventory().TargetID, c.Inventory().Revision
	operation := func(name string) Operation {
		t.Helper()
		op, err := c.Operation(name, target, revision)
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	declaration, ok := c.Type("action::declaration")
	if !ok || declaration.Kind != "opaque" {
		t.Fatalf("action::declaration marker differs: %+v", declaration)
	}
	invalidPath, ok := c.Error("action::invalid_path")
	if !ok || len(invalidPath.Fields) != 1 || invalidPath.Fields[0].Name != "reason" || invalidPath.Fields[0].Type != "str" {
		t.Fatalf("action::invalid_path differs: %+v", invalidPath)
	}
	mount := operation("action::mount")
	if mount.Result != "http::route" || len(mount.Inputs) != 1 || mount.Inputs[0].Type != "action::declaration" {
		t.Fatalf("action::mount descriptor differs: %+v", mount)
	}
	if !reflect.DeepEqual(mount.Emits, []string{"http::invalid_route"}) {
		t.Fatalf("action::mount bound differs: %+v", mount.Emits)
	}
	url := operation("action::url")
	if url.Result != "str" || len(url.Inputs) != 1 || url.Inputs[0].Type != "action::declaration" {
		t.Fatalf("action::url descriptor differs: %+v", url)
	}
	if !reflect.DeepEqual(url.Emits, []string{"action::invalid_path"}) {
		t.Fatalf("action::url bound differs: %+v", url.Emits)
	}
	request := operation("action::request")
	if len(request.Parameters) != 1 || request.Parameters[0].Name != "Result" || request.Result != "Result" {
		t.Fatalf("action::request descriptor differs: %+v", request)
	}
	if !reflect.DeepEqual(request.Emits, []string{"http::transport_failed", "http::invalid_request", "http::status_error", "codec::invalid_data"}) {
		t.Fatalf("action::request bound differs: %+v", request.Emits)
	}
	post := operation("action::post")
	if len(post.Parameters) != 2 || post.Parameters[0].Name != "Result" || post.Parameters[1].Name != "Wire" || post.Result != "Result" {
		t.Fatalf("action::post descriptor differs: %+v", post)
	}
	if !reflect.DeepEqual(post.Emits, []string{"http::transport_failed", "http::invalid_request", "http::body_limit", "http::status_error", "codec::invalid_data"}) {
		t.Fatalf("action::post bound differs: %+v", post.Emits)
	}
	for _, op := range []Operation{mount, url, request, post} {
		if op.Lowering.Task != "I32" || op.Assertion != "real" {
			t.Fatalf("%s evidence contract differs: %+v", op.Name, op.Lowering)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe: %+v", op.Name, op.Lowering)
		}
		args := map[string]string{}
		for _, p := range op.Parameters {
			args[p.Name] = "str"
		}
		spec, err := c.Resolve(op.Name, target, revision, args, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(spec.Emits) != len(op.Emits) {
			t.Fatalf("%s specialization bound differs: %+v", op.Name, spec.Emits)
		}
	}
}

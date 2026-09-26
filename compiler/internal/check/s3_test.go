package check

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	s3Client       = "can.std.s3@1::client"
	s3Metadata     = "can.std.s3@1::metadata"
	s3Upload       = "can.std.s3@1::upload"
	s3Presigned    = "can.std.s3@1::presigned"
	s3Continuation = "can.std.s3@1::continuation"
)

func s3Fixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/current/s3/objects.can")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func s3Declarations(t *testing.T, list []*types.Type) []string {
	t.Helper()
	out := make([]string, 0, len(list))
	for _, typ := range list {
		if typ == nil {
			t.Fatal("missing checked type")
		}
		out = append(out, typ.Declaration())
	}
	return out
}

func TestS3FixtureAdmitsOperationContracts(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": s3Fixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	contracts := map[string]struct {
		inputs []string
		result string
		errors []string
	}{
		"can.std.s3@1::client_open": {
			inputs: []string{"str", "str", "str", "str", "str"},
			result: s3Client,
			errors: []string{"can.std.s3@1::invalid_config"},
		},
		"can.std.s3@1::read_bytes": {
			inputs: []string{s3Client, "str", "int"},
			result: "can.std.bytes@1::buffer",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::missing_key", "can.std.s3@1::access_denied", "can.std.s3@1::service_error", "can.std.s3@1::over_limit"},
		},
		"can.std.s3@1::read_range": {
			inputs: []string{s3Client, "str", "int", "int"},
			result: "can.std.bytes@1::buffer",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::missing_key", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::read_stream": {
			inputs: []string{s3Client, "str", "int"},
			result: "can.std.stream@1::reader",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::missing_key", "can.std.s3@1::access_denied", "can.std.s3@1::service_error", "can.std.s3@1::over_limit"},
		},
		"can.std.s3@1::write_bytes": {
			inputs: []string{s3Client, "str", "can.std.bytes@1::buffer", "can.std.s3@1::write_options"},
			result: s3Metadata,
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::write_stream": {
			inputs: []string{s3Client, "str", "can.std.stream@1::reader", "can.std.s3@1::write_options", "int", "int"},
			result: s3Metadata,
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::access_denied", "can.std.s3@1::service_error", "can.std.s3@1::over_limit", "can.std.stream@1::read_failed", "can.std.stream@1::cancelled"},
		},
		"can.std.s3@1::stat": {
			inputs: []string{s3Client, "str"},
			result: s3Metadata,
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::missing_key", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::exists": {
			inputs: []string{s3Client, "str"},
			result: "bool",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::delete": {
			inputs: []string{s3Client, "str"},
			result: "void",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::list": {
			inputs: []string{s3Client, "can.std.s3@1::list_options"},
			result: "can.std.s3@1::page",
			errors: []string{"can.std.s3@1::invalid_config", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::presign": {
			inputs: []string{s3Client, "can.std.s3@1::method", "str", "int", "can.std.option@1::value"},
			result: s3Presigned,
			errors: []string{"can.std.s3@1::invalid_config"},
		},
		"can.std.s3@1::describe": {
			inputs: []string{s3Presigned},
			result: "can.std.s3@1::presigned_info",
			errors: []string{},
		},
		"can.std.s3@1::begin_upload": {
			inputs: []string{s3Client, "str", "can.std.s3@1::upload_options"},
			result: s3Upload,
			errors: []string{"can.std.s3@1::invalid_config"},
		},
		"can.std.s3@1::upload_write": {
			inputs: []string{s3Upload, "can.std.bytes@1::buffer"},
			result: "int",
			errors: []string{"can.std.s3@1::upload_closed", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::upload_finish": {
			inputs: []string{s3Upload},
			result: s3Metadata,
			errors: []string{"can.std.s3@1::upload_closed", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
		"can.std.s3@1::discard_upload": {
			inputs: []string{s3Upload},
			result: "void",
			errors: []string{"can.std.s3@1::upload_closed", "can.std.s3@1::access_denied", "can.std.s3@1::service_error"},
		},
	}
	for identity, want := range contracts {
		contract := p.Intrinsics[identity]
		if contract == nil {
			t.Fatalf("missing s3 intrinsic %s", identity)
		}
		got := s3Declarations(t, contract.Inputs())
		if strings.Join(got, ",") != strings.Join(want.inputs, ",") {
			t.Fatalf("%s inputs = %v, want %v", identity, got, want.inputs)
		}
		if contract.Result() == nil {
			t.Fatalf("missing result for %s", identity)
		}
		if result := contract.Result().Declaration(); result != want.result {
			t.Fatalf("%s result = %s, want %s", identity, result, want.result)
		}
		errors := s3Declarations(t, contract.Errors())
		sort.Strings(errors)
		sorted := append([]string(nil), want.errors...)
		sort.Strings(sorted)
		if strings.Join(errors, ",") != strings.Join(sorted, ",") {
			t.Fatalf("%s errors = %v, want %v", identity, errors, want.errors)
		}
	}
	// Clients, uploads, presigned handles and continuations cross as
	// opaque handles while metadata stays a data record.
	seen := map[string]bool{}
	for _, typ := range p.Model.Types() {
		switch typ.Declaration() {
		case s3Client:
			seen["client"] = typ.Kind() == types.Opaque
		case s3Upload:
			seen["upload"] = typ.Kind() == types.Opaque
		case s3Presigned:
			seen["presigned"] = typ.Kind() == types.Opaque
		case s3Continuation:
			seen["continuation"] = typ.Kind() == types.Opaque
		case s3Metadata:
			seen["metadata"] = typ.Kind() == types.Record
		}
	}
	for name, ok := range seen {
		if !ok {
			t.Fatalf("s3 %s has the wrong kind", name)
		}
	}
	if len(seen) != 5 {
		t.Fatalf("s3 handle types missing: %v", seen)
	}
}

func TestS3SuppliedFixturesAttachToWireBoundaries(t *testing.T) {
	p, err := programFixture(t, map[string]string{"src/main.can": s3Fixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string][]int{}
	var match func(*ir.Match)
	var completion func(*ir.Completion)
	var block func(*ir.Block)
	steps := func(call *ir.Invocation) {
		if call == nil {
			return
		}
		for i := range call.Steps {
			step := &call.Steps[i]
			if step.Fixtures == nil {
				continue
			}
			rows[step.Identity] = append(rows[step.Identity], len(step.Fixtures.Rows))
			if !strings.HasSuffix(step.Fixtures.Identity, "/when") {
				t.Fatalf("fixture table %q lacks a call-site identity", step.Fixtures.Identity)
			}
		}
	}
	completion = func(c *ir.Completion) {
		if c == nil {
			return
		}
		steps(c.Call)
		block(c.Block)
		match(c.Match)
	}
	block = func(b *ir.Block) {
		if b == nil {
			return
		}
		for i := range b.Steps {
			steps(b.Steps[i].Call)
		}
		completion(b.Terminal)
	}
	match = func(m *ir.Match) {
		if m == nil {
			return
		}
		steps(m.Call)
		for i := range m.Arms {
			completion(m.Arms[i].Body)
		}
	}
	for _, fn := range p.Functions {
		if fn.Region == nil {
			continue
		}
		block(fn.Region.Body)
	}
	// Every wire boundary carries fixtures while client construction
	// runs for real: no client_open table may appear.
	if len(rows) == 0 {
		t.Fatal("expected supplied fixture tables")
	}
	sawDescribe := false
	for identity, counts := range rows {
		if strings.Contains(identity, "client_open") {
			t.Fatalf("client_open must run for real without fixtures")
		}
		if strings.Contains(identity, "describe") {
			sawDescribe = true
		}
		for _, count := range counts {
			if count < 1 || count > 3 {
				t.Fatalf("fixture table %q carries %d rows", identity, count)
			}
		}
	}
	if !sawDescribe {
		t.Fatalf("expected a bound describe fixture table")
	}
}

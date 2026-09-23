package check

import (
	"os"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func TestCodecFormatSpecializations(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/codec/formats.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	p, err := programFixture(t, map[string]string{"src/main.can": source})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Codecs) != 5 {
		t.Fatalf("expected five format codec contracts, got %d", len(p.Codecs))
	}
	for key, codec := range p.Codecs {
		if codec.Schema.Root != codec.Data.Identity() || codec.Contract == nil {
			t.Fatal("missing checked schema")
		}
		if strings.HasPrefix(key, "can.std.codec@1::consume_jsonl<") {
			inputs := codec.Contract.Inputs()
			if len(inputs) != 2 || codec.Contract.Result().Declaration() != "int" {
				t.Fatalf("consume contract binds (reader, handler) -> int, got %v", codec.Contract)
			}
		}
		if strings.HasPrefix(key, "can.std.codec@1::decode_jsonl<") {
			if codec.Contract.Result().Kind() != types.Array {
				t.Fatalf("jsonl decode returns an array, got %v", codec.Contract)
			}
		}
	}
	for _, tc := range []struct{ name, old, replacement, want string }{
		{"opaque", "codec::decode_toml<service>", "codec::decode_toml<bytes::buffer>", "not codec-admissible"},
		{"missing argument", "codec::decode_yaml<waypoint>", "codec::decode_yaml", "one explicit"},
		{"wrong input", "codec::decode_json5<waypoint>(encoded)", "codec::decode_json5<int>(encoded)", "type"},
		{"fallible handler", "fn void handle_waypoint\n    emits []", "fn void handle_waypoint\n    emits [codec::invalid_data]", "type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := strings.Replace(source, tc.old, tc.replacement, 1)
			if modified == source {
				t.Fatal("ineffective mutation")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": modified})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s refusal, got %v", tc.want, err)
			}
		})
	}
}

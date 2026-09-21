package check

import (
	"os"
	"strings"
	"testing"
)

func TestCodecExplicitSpecializations(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/codec/roundtrip.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	p, err := programFixture(t, map[string]string{"src/main.can": source})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Codecs) != 5 {
		t.Fatalf("expected shared concrete codec contracts, got %d", len(p.Codecs))
	}
	for _, codec := range p.Codecs {
		if codec.Schema.Root != codec.Data.Identity() || codec.Contract == nil {
			t.Fatal("missing checked schema")
		}
	}
	for _, tc := range []struct{ name, old, replacement, want string }{
		{"opaque", "codec::encode_json<receipt>", "codec::encode_json<bytes::buffer>", "not codec-admissible"},
		{"callable", "codec::encode_json<receipt>", "codec::encode_json<callable int () emits []>", "not codec-admissible"},
		{"missing argument", "codec::encode_json<receipt>", "codec::encode_json", "one explicit"},
		{"wrong arity", "codec::encode_json<receipt>", "codec::encode_json<int, str>", "one explicit"},
		{"wrong input", "codec::encode_json<receipt>(original)", "codec::encode_json<int>(original)", "type"},
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

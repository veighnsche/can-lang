package check

import (
	"os"
	"strings"
	"testing"
)

func TestBytesCatalogueAdmission(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/bytes/roundtrip.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if _, err = programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"opaque constructor", "bytes::buffer empty = call bytes::empty()", "bytes::buffer empty = bytes::buffer()"},
		{"opaque indexing", "ok empty.length", "ok empty[0]"},
		{"length is projection", "ok empty.length", "ok call empty.length()"},
		{"wrong octet type", "call bytes::from_ints(values)", "call bytes::from_ints([1.0])"},
		{"wrong decode input", "call bytes::to_utf8(original)", "call bytes::to_utf8(3)"},
		{"error bound", "fn int[] integers\n    emits [codec::invalid_data]", "fn int[] integers\n    emits []"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(source, tc.old, tc.replacement, 1)
			if changed == source {
				t.Fatal("ineffective mutation")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": changed}); err == nil {
				t.Fatal("invalid bytes use accepted")
			}
		})
	}
}

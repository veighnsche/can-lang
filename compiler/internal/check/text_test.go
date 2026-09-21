package check

import (
	"os"
	"strings"
	"testing"
)

func TestTextCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../../std/text/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"value.includes(part)", "value.includes(1)"},
		{"value.to_lower_case()", "value.locale_lower_case()"},
		{"text::from_scalars(items)", "text::from_scalars([1.0])"},
		{"fn str[] split\n    emits [text::empty_separator]", "fn str[] split\n    emits []"},
	} {
		t.Run(change[1], func(t *testing.T) {
			text := strings.Replace(string(source), change[0], change[1], 1)
			if text == string(source) {
				t.Fatal("mutation missed source")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatal("unsupported coercion, operation or error bound admitted")
			}
		})
	}
}

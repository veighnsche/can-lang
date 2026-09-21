package check

import (
	"os"
	"strings"
	"testing"
)

func TestNumericCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../../std/scalars/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"number::floor(value)", "number::floor(1)"},
		{"text::from_int(value)", "text::from_int(1.0)"},
		{"number::bool_to_int(value)", "number::bool_to_int(1)"},
		{"text::to_float(value)", "text::to_float(1.0)"},
		{"number::floor(value)", "number::sqrt(value)"},
		{"fn float floor_value", "fn dec floor_value"},
		{"fn float int_float\n    emits [number::inexact]", "fn float int_float\n    emits []"},
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

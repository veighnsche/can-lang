package check

import (
	"os"
	"strings"
	"testing"
)

func TestExactAmountCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../../std/ratio/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"number::divmod(numerator, denominator)", "number::divmod(1.0, denominator)"},
		{"number::round_ratio_half_even(numerator, denominator)", "number::round_ratio_half_even(numerator, 2.0)"},
		{"fn number::division truncating\n    emits [number::zero_divisor]", "fn number::division truncating\n    emits []"},
		{"fn number::rounded rounded", "fn dec rounded"},
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

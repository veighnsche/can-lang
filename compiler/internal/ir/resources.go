package ir

import "github.com/veighnsche/can-lang/compiler/internal/types"

// ResourceCaptureIndices is conservative checked evidence for runtime capture
// retention. Callables can hide captured handles regardless of their signature.
// Recursive ordinary data is visited finitely; opaque provenance stays runtime
// private, and this adds no authored ownership/effect type syntax.
func ResourceCaptureIndices(inputs []*Expression) []int {
	var out []int
	for i, input := range inputs {
		if mayRetainResource(input.Type, map[string]bool{}) {
			out = append(out, i)
		}
	}
	return out
}
func mayRetainResource(t *types.Type, seen map[string]bool) bool {
	if seen[t.Identity()] {
		return false
	}
	seen[t.Identity()] = true
	switch t.Kind() {
	case types.Opaque:
		return t.Declaration() != "can.std.bytes@1::buffer" && t.Declaration() != "can.prelude@1::standard_failure"
	case types.Callable, types.ChoiceArm:
		return true
	case types.Array:
		return mayRetainResource(t.Element(), seen)
	case types.Record, types.Error:
		for _, field := range t.Fields() {
			if mayRetainResource(field.Type, seen) {
				return true
			}
		}
	case types.Variant:
		for _, leaf := range t.Leaves() {
			if mayRetainResource(leaf, seen) {
				return true
			}
		}
	}
	return false
}

// Predecessor completion layout. Current data identity lives in internal/types;
// current completion regions and protected payload boxes are implemented in I10.
package main

// scalarSuccess identifies the four primitive scalar types.
func scalarSuccess(ret string) bool {
	switch ret {
	case "int", "str", "bool", "dec":
		return true
	}
	return false
}

// Value successes use the existing Ok protocol with exactly one field,
// value. A variant keeps its own tag inside that field; its tag never
// doubles as the outcome discriminant. Records keep their field payloads.
func valueOkFields(ret string) [][2]string {
	return [][2]string{{"value", ret}}
}

// valueSuccess distinguishes one-value successes from flattened records.
// nominalValue means a declared brand or variant. Containment and known-type
// validation remain the caller's responsibility: a source factory can return
// Fn, but invocation successes and extern signatures must still be data-only.
func valueSuccess(ret string, nominalValue bool) bool {
	_, seq := seqElemName(ret)
	_, _, _, fn := fnTypeShape(ret)
	return nominalValue || scalarSuccess(ret) || ret == "Bytes" || seq || fn
}

func (c *tycker) valueSuccess(ret string) bool {
	return valueSuccess(ret, c.variants[ret] || c.brands[ret])
}

// successFields is the shared declared success contract. It never infers a
// shape from examples, nor treats an arbitrary unknown name as a value type.
func successFields(ret string, records map[string][][2]string, nominalValue bool) ([][2]string, bool) {
	if valueSuccess(ret, nominalValue) {
		return valueOkFields(ret), true
	}
	fields, ok := records[ret]
	return fields, ok
}

// successBinderType keeps untyped on-Ok call/invoke binders on the same ABI:
// records expose their fields; value successes expose exactly .value.
// Explicit on Ok<T> binders instead use patternSuccessType.
func (c *tycker) successBinderType(ret string) string {
	if c.valueSuccess(ret) {
		// Lazy installation also covers instantiated Seq and Fn spellings.
		c.recs[valueOkType(ret)] = valueOkFields(ret)
		return valueOkType(ret)
	}
	return ret
}

// valueOkType names a checker-only shape for on-Ok binders. The colon
// cannot occur in a source type name: this is not a first-class Outcome
// type or an implicitly declared public record.
func valueOkType(ret string) string {
	return "ok:" + ret
}

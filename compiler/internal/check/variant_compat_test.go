package check

import (
	"strings"
	"testing"
)

const variantTaggedDeclarations = `record unit
variant tagged<item>
    unit
variant bridge
    unit
`

func TestGenericVariantDirectBridgeLeafConversion(t *testing.T) {
	bodies := map[string]string{
		"direct": "    ok value\n",
		"bridge": "    bridge alias = value\n    ok alias\n",
		"leaf":   "    match value\n        unit => ok value\n",
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			text := programHeader + variantTaggedDeclarations + `fn tagged<str> convert
    emits []
    given
        tagged<int> value
    asserts
        sample: unit() => ok unit()
` + body + programMain + "    ok\n"
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
				t.Fatalf("%s conversion rejected: %v", name, err)
			}
		})
	}
}

func TestNestedGenericVariantConversion(t *testing.T) {
	declarations := `record unit
record extra
variant inner<item>
    unit
variant outer<item>
    inner<item>
    extra
fn outer<str> convert
    emits []
    given
        outer<int> value
    asserts
        sample: unit() => ok unit()
    ok value
`
	text := programHeader + declarations + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
		t.Fatal(err)
	}
}

func TestOptionConversion(t *testing.T) {
	header := strings.Replace(programHeader, "uses []", "uses [option]", 1)
	positive := header + `fn option::value<int> wrap
    emits []
    given
        int value
    asserts
        sample: 3 => ok option::some(3)
    ok option::some(value)
fn option::value<int> absent
    emits []
    asserts
        sample: => ok option::none()
    ok option::none()
` + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": positive}); err != nil {
		t.Fatalf("option leaf inclusion rejected: %v", err)
	}
	for name, fn := range map[string]string{
		"variant to variant": `fn option::value<str> convert
    emits []
    given
        option::value<int> value
    asserts
        sample: option::none() => ok option::none()
    ok value
`,
		"leaf to wrong specialization": `fn option::value<str> convert
    emits []
    given
        int value
    asserts
        sample: 3 => ok option::none()
    ok option::some(value)
`,
	} {
		t.Run(name, func(t *testing.T) {
			text := header + fn + programMain + "    ok\n"
			err := mustReject(t, text)
			if !strings.Contains(err, "expression type does not fit expected type at byte ") {
				t.Fatalf("wrong option diagnostic: %s", err)
			}
		})
	}
}

func TestGenericVariantIncompatibleRecords(t *testing.T) {
	for name, fn := range map[string]string{
		"unlisted leaf": `record unit
record other
variant tagged<item>
    unit
fn tagged<str> convert
    emits []
    given
        other value
    asserts
        sample: other() => ok unit()
    ok value
`,
		"generic record": `record box<item>
    item value
fn box<str> convert
    emits []
    given
        box<int> value
    asserts
        sample: box(3) => ok box("x")
    ok value
`,
		"distinct generic leaves": `record box<item>
    item value
variant wrap<item>
    box<item>
fn wrap<str> convert
    emits []
    given
        wrap<int> value
    asserts
        sample: box(3) => ok box("x")
    ok value
`,
	} {
		t.Run(name, func(t *testing.T) {
			text := programHeader + fn + programMain + "    ok\n"
			err := mustReject(t, text)
			if !strings.Contains(err, "expression type does not fit expected type at byte ") {
				t.Fatalf("wrong incompatible-record diagnostic: %s", err)
			}
		})
	}
}

func TestGenericVariantExhaustiveness(t *testing.T) {
	declarations := `record paid
record declined
variant payment<item>
    paid
    declined
`
	complete := programHeader + declarations + `fn str describe
    emits []
    given
        payment<int> value
    asserts
        sample: paid() => ok "paid"
    match value
        paid => ok "paid"
        declined => ok "declined"
` + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": complete}); err != nil {
		t.Fatalf("exhaustive generic match rejected: %v", err)
	}
	partial := programHeader + declarations + `fn str describe
    emits []
    given
        payment<int> value
    asserts
        sample: paid() => ok "paid"
    match value
        paid => ok "paid"
` + programMain + "    ok\n"
	if err := mustReject(t, partial); !strings.Contains(err, "ordinary match is not exhaustive") {
		t.Fatalf("missing leaf admitted: %s", err)
	}
}

func mustReject(t *testing.T, text string) string {
	t.Helper()
	_, err := programFixture(t, map[string]string{"src/main.can": text})
	if err == nil {
		t.Fatal("invalid conversion admitted")
	}
	return err.Error()
}

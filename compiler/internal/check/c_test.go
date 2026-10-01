package check

import (
	"strings"
	"testing"
)

func cKindFixture(kind string) string {
	manifest := "c::manifest(\"c\", [], [], c::manifest_entry(\"src/main.js\", \".*\", []), c::manifest_entry(\"src/assert.js\", \".*\", []), c::case_requirements([], [], []))"
	return "package app\n    provides []\n    uses [c, test]\nfn str[] check\n    emits {test::stale_handle}\n    given\n        test::owner o\n        c::manifest m\n        str path\n        str source\n    asserts\n        sample: " + manifest + ", \"src/main.js\", \"export const a = 1\" => ok []\n    match call c::check_module(o, m, " + kind + ", path, source)\n        when\n            sample: o, " + manifest + ", " + kind + ", \"src/main.js\", \"export const a = 1\" => ok []\n        test::stale_handle\n        ok str[] got => ok got\n" + programMain + "    ok\n"
}

func TestCCheckKindLiteral(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": cKindFixture("\"executable\"")}); err != nil {
		t.Fatalf("rejected literal executable kind: %v", err)
	}
}

func TestCCheckKindRefusals(t *testing.T) {
	original := cKindFixture("\"executable\"")
	for _, tc := range []struct{ name, kind, want string }{
		{"unknown kind", "\"executables\"", "is not admitted"},
		{"empty kind", "\"\"", "is not admitted"},
		{"dynamic kind", "path", "must be a static literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := cKindFixture(tc.kind)
			if source == original {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": source})
			if err == nil {
				t.Fatalf("accepted invalid kind: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong refusal for %s: %v", tc.name, err)
			}
		})
	}
}

func cParseElisionFixture() string {
	return "package app\n    provides []\n    uses [c, test]\nfn c::parsed_module parse\n    emits {test::stale_handle}\n    given\n        test::owner o\n        str source\n    asserts\n        sample: \"export const a = 1\" => ok c::parsed_module([], [\"a\"], false, \"export const a = 1\")\n    match call c::parse_module(o, source)\n        when\n            sample: o, source => ok c::parsed_module([], [\"a\"], false, \"export const a = 1\")\n        test::stale_handle\n        ok c::parsed_module got => ok got\n" + programMain + "    ok\n"
}

func TestCParseOwnerElision(t *testing.T) {
	if _, err := programFixture(t, map[string]string{"src/main.can": cParseElisionFixture()}); err != nil {
		t.Fatalf("rejected elided owner row: %v", err)
	}
}

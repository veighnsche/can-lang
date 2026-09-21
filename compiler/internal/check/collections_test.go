package check

import (
	"os"
	"strings"
	"testing"
)

func TestCollectionsCatalogue(t *testing.T) {
	source, err := os.ReadFile("../../../std/map/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = programFixture(t, map[string]string{"src/main.can": string(source)}); err != nil {
		t.Fatal(err)
	}
}

func TestCollectionsRejectInvalidContracts(t *testing.T) {
	source, err := os.ReadFile("../../../std/map/current/src/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"collections::empty_set<int>()", "collections::empty_set<float>()"},
		{"collections::empty_map<int,str>()", "collections::empty_map<collections::entry<int,str>,str>()"},
		{"collections::empty_set<int>()", "collections::set<int>()"},
		{"collections::contains(first, 1)", "collections::contains(first, 1.0)"},
		{"callable collections::add\n", "callable collections::contains\n"},
		{"emits [collections::key_absent]\n    asserts", "emits []\n    asserts"},
		{"collections::add(empty, false)", "collections::add(empty, 1)"},
	} {
		t.Run(change[1], func(t *testing.T) {
			text := strings.Replace(string(source), change[0], change[1], 1)
			if text == string(source) {
				t.Fatal("mutation missed source")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatal("invalid collection contract admitted")
			}
		})
	}
}

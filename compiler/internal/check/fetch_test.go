package check

import (
	"os"
	"strings"
	"testing"
)

func TestNamedFetchModes(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	program, err := programFixture(t, map[string]string{"src/main.can": string(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Natives) != 8 {
		t.Fatal("missing named fetch plans")
	}
	for _, native := range program.Natives {
		if native.Fetch == nil {
			t.Fatal("fetch not retained")
		}
	}
	for _, change := range [][2]string{
		{"body text payload", "body json payload\n    body text payload"},
		{"post \"/text\"", "get \"/text\""},
		{"fetch str head_text", "fetch receipt head_text"},
		{"fetch receipt load_json", "fetch int load_json"},
		{"labels = [\"a b\", \"+\"]", "labels = [1]"},
		{"body text payload", "body bytes payload"},
		{"    body json payload", "    headers\n        content_type = \"text/plain\"\n    body json payload"},
		{"    body json payload", "    headers\n        content_type = \"\"\n    body json payload"},
		{"http::timeout, ", ""},
	} {
		t.Run(change[1], func(t *testing.T) {
			text := strings.Replace(string(source), change[0], change[1], 1)
			if text == string(source) {
				t.Fatal("missed mutation")
			}
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatal("invalid fetch admitted")
			}
		})
	}
}

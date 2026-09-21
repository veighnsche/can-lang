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
		{"body text (call [[payload]].map(callable extract))[0]", "body json payload\n    body text (call [[payload]].map(callable extract))[0]"},
		{"post \"/text\"", "get \"/text\""},
		{"fetch str head_text", "fetch receipt head_text"},
		{"fetch receipt load_json", "fetch int load_json"},
		{"labels = call [[\"a b\"], [\"+\"]].map(callable extract)", "labels = [1]"},
		{"body text (call [[payload]].map(callable extract))[0]", "body bytes payload"},
		{`content_type = ([...([("application/json")])])`, `content_type = "text/plain"`},
		{`content_type = ([...([("application/json")])])`, `content_type = ""`},
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

func TestFetchRejectsLiteralHeaderValuesInsideArrays(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`"a\r\nb"`, `["a\r\nb"]`, `(["a\nb"])`, `["ok", ...(["a\0b"])]`, `[...([("Ā")])]`} {
		t.Run(bad, func(t *testing.T) {
			text := strings.Replace(string(source), `x_probe = call [["yes"]].map(callable extract)`, `x_probe = `+bad, 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), "invalid literal request header value") {
				t.Fatalf("literal header admitted or wrong failure: %v", err)
			}
		})
	}
}

func TestFetchGroupedEmptyEntriesAndLiteralContentTypes(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, entry := range []string{`labels = call [["a b"], ["+"]].map(callable extract)`, `x_probe = call [["yes"]].map(callable extract)`} {
		for _, value := range []string{"[]", "([])", "(([]))", "([...([])])"} {
			t.Run(entry+value, func(t *testing.T) {
				name := strings.Split(entry, " = ")[0]
				if _, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(source, entry, name+" = "+value, 1)}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	for _, value := range []string{`("text/plain")`, `(("text/plain"))`, `[...["text/plain"]]`, `([...([("text/plain")])])`, `["application/json", ...(["text/plain"])]`} {
		t.Run(value, func(t *testing.T) {
			text := strings.Replace(source, `content_type = ([...([("application/json")])])`, "content_type = "+value, 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err == nil || !strings.Contains(err.Error(), "JSON fetch body requires") {
				t.Fatalf("known conflict: %v", err)
			}
		})
	}
	for _, value := range []string{`("application/json")`, `([...(["application/json; charset=utf-8"])])`, `([])`, `([...([])])`} {
		t.Run("valid "+value, func(t *testing.T) {
			text := strings.Replace(source, `content_type = ([...([("application/json")])])`, "content_type = "+value, 1)
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFetchContentTypeEmptyOverrideAndDynamicAdmission(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	base := strings.Replace(string(data), "    max_body_bytes 8192", "    max_body_bytes 8192\n    headers\n        content_type = \"text/plain\"", 1)
	for _, value := range []string{`[]`, `(([]))`, `([...([])])`, `("application/json")`, `call ["text/plain"].map(callable pass_header)`} {
		t.Run(value, func(t *testing.T) {
			text := strings.Replace(base, `content_type = ([...([("application/json")])])`, "content_type = "+value, 1)
			text += `fn str pass_header
    emits []
    given
        str value
    asserts
        sample: "text/plain" => ok "text/plain"
    ok value
`
			if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
				t.Fatal(err)
			}
		})
	}
	base = strings.Replace(base, `    headers
        content_type = ([...([("application/json")])])
`, "", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": base}); err == nil || !strings.Contains(err.Error(), "JSON fetch body requires") {
		t.Fatalf("unreplaced default conflict: %v", err)
	}
}

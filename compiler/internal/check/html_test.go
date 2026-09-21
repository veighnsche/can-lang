package check

import (
	"strings"
	"testing"
)

func TestHTMLHasOnlyCatalogueConstructors(t *testing.T) {
	header := "package app\n    provides []\n    uses [html, htmx]\n"
	main := "fn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n"
	for _, body := range []string{
		"    html::safe forged = html::safe(\"<script>bad</script>\")\n    ok\n",
		"    html::node forged = html::node(\"markup\")\n    ok\n",
		"    html::attribute raw = call html::raw(\"onclick\", \"bad\")\n    ok\n",
		"    html::node script = call html::script(\"bad\")\n    ok\n",
	} {
		if _, err := programFixture(t, map[string]string{"src/main.can": header + main + body}); err == nil {
			t.Fatalf("accepted forged HTML:\n%s", body)
		}
	}
	for _, legacy := range []string{"brand Html__Safe is str rev 1\n", "asset_bridge Approved, Policy from schema via trusted@1 for script\n"} {
		if _, err := programFixture(t, map[string]string{"src/main.can": header + legacy + main + "    ok\n"}); err == nil {
			t.Fatal("legacy HTML authority accepted")
		}
	}
	source := header + strings.Replace(main, "emits []", "emits [html::invalid_structure]", 1) + "    match call html::make_tag(\"div\")\n        ok html::tag tag => ok\n        html::invalid_structure\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
}

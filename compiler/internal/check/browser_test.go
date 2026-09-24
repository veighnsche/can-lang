package check

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const browserSurfaceFixture = `package app
    provides []
    uses [browser]
fn void on_click
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("click", "save", "", "") => ok
    ok
fn void on_tick
    emits []
    asserts
        sample: => ok
    ok
fn int demo
    emits [browser::missing_root, browser::disposed, browser::rejected, browser::stale_version]
    given
        str root
    asserts
        sample: "app" => ok 1
    match call browser::mount(root)
        browser::missing_root
        ok browser::app app => match call browser::root(app)
            browser::disposed
            ok browser::node anchor => match call browser::open_view(app)
                browser::disposed
                ok browser::view view => match call browser::create_element(view, "div")
                    browser::disposed
                    browser::rejected
                    ok browser::node box => match call browser::create_text(view, "hi")
                        browser::disposed
                        ok browser::node label => match call browser::set_text(label, "hello")
                            browser::disposed
                            ok => match call browser::set_attribute(box, "id", "panel")
                                browser::disposed
                                browser::rejected
                                ok => match call browser::append_child(box, label)
                                    browser::disposed
                                    browser::rejected
                                    ok => match call browser::append_child(anchor, box)
                                        browser::disposed
                                        browser::rejected
                                        ok => match call browser::focus(box)
                                            browser::disposed
                                            ok => match call browser::on_event(view, box, "click", callable on_click)
                                                browser::disposed
                                                browser::rejected
                                                ok => match call browser::set_timeout(view, 250, callable on_tick)
                                                    browser::disposed
                                                    browser::rejected
                                                    ok => match call browser::create_state(view, 7)
                                                        browser::disposed
                                                        ok browser::state<int> cell => match call browser::read_state(cell)
                                                            browser::disposed
                                                            ok browser::snapshot<int> snap => match call browser::replace_state(cell, snap.version, 8)
                                                                browser::disposed
                                                                browser::stale_version
                                                                ok int next => match call browser::remove_attribute(box, "id")
                                                                    browser::disposed
                                                                    browser::rejected
                                                                    ok => match call browser::remove_node(label)
                                                                        browser::disposed
                                                                        browser::rejected
                                                                        ok => match call browser::create_state<str>(view, "draft")
                                                                            browser::disposed
                                                                            ok browser::state<str> named => match call browser::read_state(named)
                                                                                browser::disposed
                                                                                ok browser::snapshot<str> seen => match call browser::dispose_view(view)
                                                                                    ok => match call browser::dispose_app(app)
                                                                                        ok => ok next
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestBrowserSurfaceAdmitsFullCatalogue(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/main.can": browserSurfaceFixture})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.BrowserStates) != 5 {
		t.Fatalf("expected five browser state specializations, got %d", len(program.BrowserStates))
	}
	seen := map[string]int{}
	for key, special := range program.BrowserStates {
		if special.Contract == nil || special.Data == nil {
			t.Fatalf("missing browser state contract for %s", key)
		}
		seen[special.Operation]++
	}
	for _, operation := range []string{browserCreateState, browserReadState, browserReplaceState} {
		if seen[operation] == 0 {
			t.Fatalf("missing browser state specialization for %s", operation)
		}
	}
	if seen[browserCreateState] != 2 {
		t.Fatalf("expected int and str create_state specializations, got %d", seen[browserCreateState])
	}
	if seen[browserReadState] != 2 || seen[browserReplaceState] != 1 {
		t.Fatalf("expected two read_state and one replace_state specializations, got %d and %d", seen[browserReadState], seen[browserReplaceState])
	}
	identities := map[string]bool{}
	var walk func(*ir.Region)
	walk = func(region *ir.Region) {
		if region == nil {
			return
		}
		var block func(*ir.Block)
		var completion func(*ir.Completion)
		steps := func(call *ir.Invocation) {
			if call == nil {
				return
			}
			for i := range call.Steps {
				identities[call.Steps[i].Identity] = true
			}
		}
		completion = func(done *ir.Completion) {
			if done == nil {
				return
			}
			steps(done.Call)
			block(done.Block)
			if done.Match != nil {
				steps(done.Match.Call)
				for i := range done.Match.Arms {
					completion(done.Match.Arms[i].Body)
				}
			}
		}
		block = func(b *ir.Block) {
			if b == nil {
				return
			}
			for i := range b.Steps {
				steps(b.Steps[i].Call)
			}
			completion(b.Terminal)
		}
		block(region.Body)
	}
	for _, fn := range program.Functions {
		walk(fn.Region)
	}
	for _, identity := range []string{
		browserMount, browserRoot, browserOpenView, browserDisposeView, browserDisposeApp,
		browserCreateElement, browserCreateText, browserSetText, browserSetAttribute,
		browserRemoveAttribute, browserAppendChild, browserRemoveNode, browserFocus,
		browserOnEvent, browserSetTimeout,
	} {
		if !identities[identity] {
			t.Fatalf("checked IR omits %s", identity)
		}
	}
}

func TestBrowserStaticAdmissionRefusals(t *testing.T) {
	dynamic := strings.Replace(browserSurfaceFixture, `call browser::create_element(view, "div")`, `call browser::create_element(view, tag)`, 1)
	dynamic = strings.Replace(dynamic, "        str root\n", "        str root\n        str tag\n", 1)
	dynamic = strings.Replace(dynamic, `sample: "app" => ok 1`, `sample: "app", "div" => ok 1`, 1)
	if dynamic == browserSurfaceFixture {
		t.Fatal("invalid dynamic-tag fixture")
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": dynamic}); err != nil {
		t.Fatalf("dynamic tag rejected statically: %v", err)
	}
	for _, tc := range []struct{ name, from, to, want string }{
		{"script tag", `create_element(view, "div")`, `create_element(view, "script")`, `browser tag "script" is not admitted`},
		{"style tag", `create_element(view, "div")`, `create_element(view, "STYLE")`, `browser tag "STYLE" is not admitted`},
		{"unknown tag", `create_element(view, "div")`, `create_element(view, "marquee")`, `browser tag "marquee" is not admitted`},
		{"empty tag", `create_element(view, "div")`, `create_element(view, "")`, `browser tag "" is not admitted`},
		{"handler attribute", `set_attribute(box, "id", "panel")`, `set_attribute(box, "onclick", "x()")`, `browser attribute "onclick" is not admitted`},
		{"style attribute", `set_attribute(box, "id", "panel")`, `set_attribute(box, "style", "color:red")`, `browser attribute "style" is not admitted`},
		{"remove handler attribute", `remove_attribute(box, "id")`, `remove_attribute(box, "onload")`, `browser attribute "onload" is not admitted`},
		{"javascript url", `set_attribute(box, "id", "panel")`, `set_attribute(box, "href", "javascript:alert(1)")`, `browser URL attribute value is not admitted`},
		{"protocol relative url", `set_attribute(box, "id", "panel")`, `set_attribute(box, "src", "//example.com/x.js")`, `browser URL attribute value is not admitted`},
		{"plain http url", `set_attribute(box, "id", "panel")`, `set_attribute(box, "href", "http://example.com/")`, `browser URL attribute value is not admitted`},
		{"unknown event", `on_event(view, box, "click", callable on_click)`, `on_event(view, box, "mouseover", callable on_click)`, `browser event "mouseover" is not admitted`},
		{"call result callback", `on_event(view, box, "click", callable on_click)`, `on_event(view, box, "click", call on_tick())`, `browser event callback must be a named reference`},
		{"timer result callback", `set_timeout(view, 250, callable on_tick)`, `set_timeout(view, 250, call on_tick())`, `browser timer callback must be a named reference`},
		{"negative delay", `set_timeout(view, 250, callable on_tick)`, `set_timeout(view, -5, callable on_tick)`, `browser delay must be 0-2147483647 milliseconds`},
		{"huge delay", `set_timeout(view, 250, callable on_tick)`, `set_timeout(view, 2147483648, callable on_tick)`, `browser delay must be 0-2147483647 milliseconds`},
		{"forged handles", "fn void main", "fn browser::app forge_app\n    emits []\n    asserts\n        sample: => ok browser::app(\"x\")\n    ok browser::app(\"root\")\nfn browser::node forge_node\n    emits []\n    asserts\n        sample: => ok browser::node(\"x\")\n    ok browser::node(\"text\")\nfn void main", `no eligible constructor browser::app`},
		{"unknown operation", `call browser::focus(box)`, `call browser::raw(box)`, `no eligible call browser::raw`},
		{"fallible handler", "fn void on_click\n    emits []", "fn void on_click\n    emits [browser::rejected]", ``},
		{"replace mistyped value", `replace_state(cell, snap.version, 8)`, `replace_state(cell, snap.version, "x")`, ``},
		{"cross spec snapshot", `ok browser::snapshot<int> snap`, `ok browser::snapshot<str> snap`, ``},
		{"unhandled missing root", "    emits [browser::missing_root, browser::disposed, browser::rejected, browser::stale_version]\n", "    emits [browser::disposed, browser::rejected, browser::stale_version]\n", `browser::missing_root`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(browserSurfaceFixture, tc.from, tc.to, 1)
			if source == browserSurfaceFixture {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": source})
			if err == nil {
				t.Fatalf("accepted invalid browser call: %s", tc.to)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("browser diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestBrowserDiagnosticSpans(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"tag", `create_element(view, "div")`, `create_element(view, "script")`, `"script"`},
		{"attribute", `set_attribute(box, "id", "panel")`, `set_attribute(box, "onclick", "x()")`, `"onclick"`},
		{"event", `"click", callable on_click`, `"mouseover", callable on_click`, `"mouseover"`},
		{"url", `set_attribute(box, "id", "panel")`, `set_attribute(box, "href", "javascript:alert(1)")`, `"javascript:alert(1)"`},
		{"delay", `set_timeout(view, 250, callable on_tick)`, `set_timeout(view, -5, callable on_tick)`, `-5`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(browserSurfaceFixture, tc.from, tc.to, 1)
			if text == browserSurfaceFixture {
				t.Fatal("invalid span fixture")
			}
			_, err := programFixture(t, map[string]string{"src/main.can": text})
			if err == nil {
				t.Fatalf("admitted invalid browser literal: %s", tc.to)
			}
			located, ok := source.AsLocated(err)
			if !ok {
				t.Fatalf("browser failure lost its span: %v", err)
			}
			if got := text[located.Span.Start:located.Span.End]; got != tc.want {
				t.Fatalf("browser span covers %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBrowserAdmissionCorpusMatchesRuntime(t *testing.T) {
	data, err := os.ReadFile("../../../runtime/test/browser-names.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Tags struct {
			Accept []string `json:"accept"`
			Reject []string `json:"reject"`
		} `json:"tags"`
		Attributes struct {
			Accept []string `json:"accept"`
			Reject []string `json:"reject"`
		} `json:"attributes"`
		Events struct {
			Accept []string `json:"accept"`
			Reject []string `json:"reject"`
		} `json:"events"`
		URLs struct {
			Accept []string `json:"accept"`
			Reject []string `json:"reject"`
		} `json:"urls"`
		Delays struct {
			Accept []int64 `json:"accept"`
			Reject []int64 `json:"reject"`
		} `json:"delays"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Tags.Accept) == 0 || len(corpus.Tags.Reject) == 0 {
		t.Fatal("empty shared tag corpus")
	}
	for _, tag := range corpus.Tags.Accept {
		if err := checkBrowserTag(tag); err != nil {
			t.Fatalf("checker rejects runtime-accepted tag %q: %v", tag, err)
		}
	}
	for _, tag := range corpus.Tags.Reject {
		if err := checkBrowserTag(tag); err == nil {
			t.Fatalf("checker admits runtime-rejected tag %q", tag)
		}
	}
	for name := range browserTags {
		found := false
		for _, tag := range corpus.Tags.Accept {
			if strings.ToLower(tag) == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("checker tag %q escapes the shared corpus", name)
		}
	}
	for _, name := range corpus.Attributes.Accept {
		if err := checkBrowserAttributeName(name); err != nil {
			t.Fatalf("checker rejects runtime-accepted attribute %q: %v", name, err)
		}
	}
	for _, name := range corpus.Attributes.Reject {
		if err := checkBrowserAttributeName(name); err == nil {
			t.Fatalf("checker admits runtime-rejected attribute %q", name)
		}
	}
	for name := range browserAttributes {
		found := false
		for _, accept := range corpus.Attributes.Accept {
			if strings.ToLower(accept) == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("checker attribute %q escapes the shared corpus", name)
		}
	}
	for _, kind := range corpus.Events.Accept {
		if err := checkBrowserEvent(kind); err != nil {
			t.Fatalf("checker rejects runtime-accepted event %q: %v", kind, err)
		}
	}
	for _, kind := range corpus.Events.Reject {
		if err := checkBrowserEvent(kind); err == nil {
			t.Fatalf("checker admits runtime-rejected event %q", kind)
		}
	}
	for _, value := range corpus.URLs.Accept {
		if err := checkBrowserURLValue(value); err != nil {
			t.Fatalf("checker rejects runtime-accepted URL %q: %v", value, err)
		}
	}
	for _, value := range corpus.URLs.Reject {
		if err := checkBrowserURLValue(value); err == nil {
			t.Fatalf("checker admits runtime-rejected URL %q", value)
		}
	}
	for _, delay := range corpus.Delays.Accept {
		if delay < 0 || delay > browserMaxDelayMs {
			t.Fatalf("checker rejects runtime-accepted delay %d", delay)
		}
	}
	for _, delay := range corpus.Delays.Reject {
		if delay >= 0 && delay <= browserMaxDelayMs {
			t.Fatalf("checker admits runtime-rejected delay %d", delay)
		}
	}
}

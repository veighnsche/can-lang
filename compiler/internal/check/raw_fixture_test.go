package check

import (
	"strings"
	"testing"
)

const rawFixtureValid = `{
  "schema": "can.native-fixture.v1",
  "target": "app::load",
  "environment": {"TOKEN": "fake"},
  "exchange": {
    "request": {
      "method": "GET",
      "url": "https://example.invalid/receipt",
      "headers": [["x-probe", "yes"]],
      "body": {"bytes_base64": ""}
    },
    "outcome": {
      "response": {
        "status": 200,
        "headers": [["content-type", "application/json"]],
        "body_base64": "e30="
      }
    }
  }
}`

func TestParseRawFixtureValid(t *testing.T) {
	fixture, err := ParseRawFixture([]byte(rawFixtureValid))
	if err != nil {
		t.Fatal(err)
	}
	if fixture.Target != "app::load" || fixture.Environment["TOKEN"] != "fake" {
		t.Fatal(fixture)
	}
	exchange := fixture.Exchange
	if exchange == nil || exchange.Method != "GET" || exchange.Response == nil || exchange.Response.Status != 200 {
		t.Fatal("response exchange missing")
	}
	null, err := ParseRawFixture([]byte(`{"schema":"can.native-fixture.v1","target":"app::load","environment":{},"exchange":null}`))
	if err != nil || null.Exchange != nil {
		t.Fatalf("null exchange rejected: %v", err)
	}
}

func TestParseRawFixtureRejects(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"schema": func(s string) string { return strings.Replace(s, "can.native-fixture.v1", "other", 1) },
		"duplicate": func(s string) string {
			return strings.Replace(s, `"target": "app::load",`, `"target": "app::load", "target": "app::load",`, 1)
		},
		"unknown": func(s string) string { return strings.Replace(s, `"exchange": {`, `"extra": 1, "exchange": {`, 1) },
		"method":  func(s string) string { return strings.Replace(s, `"GET"`, `"get"`, 1) },
		"url":     func(s string) string { return strings.Replace(s, "https://example.invalid/receipt", "/relative", 1) },
		"both-body": func(s string) string {
			return strings.Replace(s, `"body": {"bytes_base64": ""}`, `"body": {"bytes_base64": "", "json_utf8": "{}"}`, 1)
		},
		"no-body": func(s string) string {
			return strings.Replace(s, `"body": {"bytes_base64": ""}`, `"body": {}`, 1)
		},
		"base64": func(s string) string { return strings.Replace(s, `"e30="`, `"e30"`, 1) },
		"status": func(s string) string { return strings.Replace(s, `"status": 200`, `"status": 99`, 1) },
		"phase": func(s string) string {
			return strings.Replace(s, `"outcome": {
      "response": {
        "status": 200,
        "headers": [["content-type", "application/json"]],
        "body_base64": "e30="
      }
    }`, `"outcome": {"failure": {"kind": "transport", "phase": "warp"}}`, 1)
		},
		"both-outcome": func(s string) string {
			return strings.Replace(s, `"outcome": {`, `"outcome": {"failure": {"kind": "timeout"}, "response": {`, 1)
		},
		"json-dup": func(s string) string {
			return strings.Replace(s, `"body": {"bytes_base64": ""}`, `"body": {"json_utf8": "{\"a\":1,\"a\":2}"}`, 1)
		},
		"head-body": func(s string) string {
			return strings.Replace(strings.Replace(s, `"GET"`, `"HEAD"`, 1), `"body_base64": "e30="`, `"body_base64": "e30="`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRawFixture([]byte(mutate(rawFixtureValid))); err == nil {
				t.Fatal("invalid fixture admitted")
			}
		})
	}
	if _, err := ParseRawFixture([]byte(rawFixtureValid[:40])); err == nil {
		t.Fatal("truncated fixture admitted")
	}
}

func TestRawFixtureBindingRejects(t *testing.T) {
	header := "package app\n    provides []\n    uses [http, codec]\nconnection service\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n"
	mismatched := header + "fetch str load from service\n    emits [http::request_failed]\n    asserts\n        sample: => ok \"x\"\n            using raw \"fixtures/other.json\"\n    get \"/\"\n" + programMain + "    ok\n"
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": mismatched}, "load", "other")); err == nil || !strings.Contains(err.Error(), "raw fixture targets can.project.root/app::other, not can.project.root/app::load") {
		t.Fatalf("wrong fixture target admitted: %v", err)
	}
	uncovered := header + "fetch str load from service\n    emits [http::request_failed]\n    asserts\n        sample: => ok \"x\"\n    get \"/\"\n" + programMain + "    ok\n"
	if _, err := programFixture(t, map[string]string{"src/main.can": uncovered}); err == nil || !strings.Contains(err.Error(), "lacks request/decoder coverage") {
		t.Fatalf("missing request/decoder coverage admitted: %v", err)
	}
	nullOnly := header + "fetch str load from service\n    emits [http::request_failed]\n    asserts\n        sample: => http::request_failed(http::credentials_missing(\"TOKEN\"))\n            using raw \"fixtures/load.json\"\n    get \"/\"\n" + programMain + "    ok\n"
	files := map[string]string{"src/main.can": nullOnly}
	files["src/fixtures/load.json"] = `{"schema":"can.native-fixture.v1","target":"can.project.root/app::load","environment":{},"exchange":null}`
	if _, err := programFixture(t, files); err == nil || !strings.Contains(err.Error(), "lacks request/decoder coverage") {
		t.Fatalf("null-exchange-only coverage admitted: %v", err)
	}
}

func TestParseRawFixtureFailures(t *testing.T) {
	for _, outcome := range []string{
		`{"failure": {"kind": "transport", "phase": "body"}}`,
		`{"failure": {"kind": "timeout"}}`,
		`{"failure": {"kind": "body_limit"}}`,
	} {
		text := strings.Replace(rawFixtureValid, `"outcome": {
      "response": {
        "status": 200,
        "headers": [["content-type", "application/json"]],
        "body_base64": "e30="
      }
    }`, `"outcome": `+outcome, 1)
		fixture, err := ParseRawFixture([]byte(text))
		if err != nil || fixture.Exchange == nil || fixture.Exchange.Failure == nil {
			t.Fatalf("failure outcome rejected: %v", err)
		}
	}
}

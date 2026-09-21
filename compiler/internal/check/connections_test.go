package check

import (
	"math/big"
	"testing"
)

func TestConnectionPolicy(t *testing.T) {
	text := func(name, value string) ConnectionSetting { return ConnectionSetting{Name: name, Text: &value} }
	integer := func(name string, value int64) ConnectionSetting {
		return ConnectionSetting{Name: name, Integer: big.NewInt(value)}
	}
	base := []ConnectionSetting{text("endpoint", "http://localhost:8080/api/"), integer("timeout_ms", 1000)}
	p, err := CheckConnection(base)
	if err != nil || p.MaxBodyBytes != 8388608 {
		t.Fatalf("defaults: %+v %v", p, err)
	}
	for _, setting := range []ConnectionSetting{
		text("endpoint", "https://example.test/?"), text("endpoint", "https://example.test:65536/"), text("auth", "lowercase"), text("endpoint", "https://user@example.test/"),
		integer("timeout_ms", 0), integer("timeout_ms", 2147483648), integer("max_body_bytes", 67108865),
		{Name: "metadata", Entries: []ConnectionSetting{text("unknown", "x")}},
		{Name: "metadata", Entries: []ConnectionSetting{text("protocol", "typesafe_systemone_v1"), integer("max_output_tokens", 1)}},
		{Name: "headers", Entries: []ConnectionSetting{text("host", "example.test")}},
		{Name: "headers", Entries: []ConnectionSetting{text("x_test", "ok"), text("X_TEST", "duplicate")}},
		{Name: "headers", Entries: []ConnectionSetting{text("x_test", "bad\nvalue")}},
	} {
		settings := append([]ConnectionSetting{}, base...)
		replaced := false
		for i := range settings {
			if settings[i].Name == setting.Name {
				settings[i] = setting
				replaced = true
			}
		}
		if !replaced {
			settings = append(settings, setting)
		}
		if _, err := CheckConnection(settings); err == nil {
			t.Fatalf("accepted %+v", setting)
		}
	}
	settings := append(append([]ConnectionSetting{}, base...), ConnectionSetting{Name: "metadata", Entries: []ConnectionSetting{integer("max_output_tokens", 32768), text("model", "test"), text("protocol", "openai_responses_v1")}})
	p, err = CheckConnection(settings)
	if err != nil || p.CheckAI("openai_responses_v1") != nil || p.MaxOutputTokens != 32768 {
		t.Fatalf("AI profile %+v %v", p, err)
	}
	if p.CheckAI("typesafe_systemone_v1") == nil {
		t.Fatal("accepted incompatible AI profile")
	}
}

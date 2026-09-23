package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// RawSpec renders a validated fixture as a RawFixtureInput object literal.
// Bodies stay in their authored base64/UTF-8 form; the runtime decodes and
// compares them at the owned boundary.
func RawSpec(raw *ir.RawFixture) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("missing raw fixture")
	}
	environment := map[string]string{}
	for name, value := range raw.Environment {
		environment[name] = value
	}
	spec := map[string]any{"environment": environment, "exchange": nil}
	if raw.Exchange != nil {
		exchange := raw.Exchange
		headers := make([][2]string, 0, len(exchange.Headers))
		headers = append(headers, exchange.Headers...)
		body := map[string]any{}
		if exchange.HasBodyJSON {
			body["json_utf8"] = exchange.BodyJSON
		} else {
			body["bytes_base64"] = exchange.BodyBase64
		}
		outcome := map[string]any{}
		if exchange.Response != nil {
			response := exchange.Response
			responseHeaders := make([][2]string, 0, len(response.Headers))
			responseHeaders = append(responseHeaders, response.Headers...)
			outcome["response"] = map[string]any{"status": response.Status, "headers": responseHeaders, "body_base64": response.Body}
		} else if exchange.Failure != nil {
			failure := map[string]any{"kind": exchange.Failure.Kind}
			if exchange.Failure.Kind == "transport" {
				failure["phase"] = exchange.Failure.Phase
			}
			outcome["failure"] = failure
		} else {
			return "", fmt.Errorf("raw fixture exchange needs a response or failure outcome")
		}
		spec["exchange"] = map[string]any{
			"request": map[string]any{"method": exchange.Method, "url": exchange.URL, "headers": headers, "body": body},
			"outcome": outcome,
		}
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

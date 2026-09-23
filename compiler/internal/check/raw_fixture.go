package check

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
)

// maxRawFixtureBytes bounds one staged raw exchange document.
const maxRawFixtureBytes = 8388608

// RawFixture is a validated can.native-fixture.v1 document. Bodies stay in
// their authored base64/UTF-8 form; emit embeds them verbatim.
type RawFixture struct {
	Target      string
	Environment map[string]string
	Exchange    *RawExchange
}
type RawExchange struct {
	Method      string
	URL         string
	Headers     [][2]string
	BodyBase64  string
	BodyJSON    string
	HasBodyJSON bool
	Response    *RawResponse
	Failure     *RawFailure
}
type RawResponse struct {
	Status  int
	Headers [][2]string
	Body    string
}
type RawFailure struct {
	Kind  string
	Phase string
}

var rawMethods = map[string]bool{"GET": true, "HEAD": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "OPTIONS": true}
var rawPhases = map[string]bool{"connect": true, "body": true, "protocol": true, "cancelled": true}

// ParseRawFixture strictly validates one fixture document: UTF-8 JSON,
// duplicate/unknown field rejection, canonical base64, and the exact
// request/outcome shape. Target and environment-key checks run against the
// owning declaration at the call site.
func ParseRawFixture(data []byte) (*RawFixture, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("raw fixture is not valid UTF-8")
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("raw fixture is not valid JSON")
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return nil, err
	}
	fields, err := strictObject(data, []string{"schema", "target", "environment", "exchange"}, nil)
	if err != nil {
		return nil, err
	}
	schema, err := strictText(fields["schema"])
	if err != nil || schema != "can.native-fixture.v1" {
		return nil, fmt.Errorf("raw fixture schema must be can.native-fixture.v1")
	}
	target, err := strictText(fields["target"])
	if err != nil || target == "" {
		return nil, fmt.Errorf("raw fixture target must be a nonempty string")
	}
	environment := map[string]string{}
	env, err := strictDictionary(fields["environment"])
	if err != nil {
		return nil, fmt.Errorf("raw fixture environment: %w", err)
	}
	for name, raw := range env {
		value, err := strictText(raw)
		if err != nil {
			return nil, fmt.Errorf("raw fixture environment %q: %w", name, err)
		}
		environment[name] = value
	}
	fixture := &RawFixture{Target: target, Environment: environment}
	trimmed := bytes.TrimSpace(fields["exchange"])
	if string(trimmed) == "null" {
		return fixture, nil
	}
	exchange, err := strictObject(fields["exchange"], []string{"request", "outcome"}, nil)
	if err != nil {
		return nil, fmt.Errorf("raw fixture exchange: %w", err)
	}
	request, err := strictObject(exchange["request"], []string{"method", "url", "headers", "body"}, nil)
	if err != nil {
		return nil, fmt.Errorf("raw fixture request: %w", err)
	}
	result := &RawExchange{}
	if result.Method, err = strictText(request["method"]); err != nil || !rawMethods[result.Method] {
		return nil, fmt.Errorf("raw fixture method must be an uppercase HTTP method")
	}
	if result.URL, err = strictText(request["url"]); err != nil {
		return nil, fmt.Errorf("raw fixture url: %w", err)
	}
	endpoint, err := url.Parse(result.URL)
	if err != nil || endpoint.Scheme != "http" && endpoint.Scheme != "https" || endpoint.Host == "" {
		return nil, fmt.Errorf("raw fixture url must be an absolute http(s) URL")
	}
	if result.Headers, err = strictPairs(request["headers"]); err != nil {
		return nil, fmt.Errorf("raw fixture request headers: %w", err)
	}
	body, err := strictObject(request["body"], nil, []string{"bytes_base64", "json_utf8"})
	if err != nil {
		return nil, fmt.Errorf("raw fixture body: %w", err)
	}
	_, hasBytes := body["bytes_base64"]
	_, hasJSON := body["json_utf8"]
	if hasBytes == hasJSON {
		return nil, fmt.Errorf("raw fixture body needs exactly one of bytes_base64, json_utf8")
	}
	if hasBytes {
		text, err := strictText(body["bytes_base64"])
		if err != nil {
			return nil, fmt.Errorf("raw fixture body: %w", err)
		}
		if err := checkBase64(text); err != nil {
			return nil, fmt.Errorf("raw fixture body: %w", err)
		}
		result.BodyBase64 = text
	} else {
		text, err := strictText(body["json_utf8"])
		if err != nil {
			return nil, fmt.Errorf("raw fixture body: %w", err)
		}
		if !json.Valid([]byte(text)) {
			return nil, fmt.Errorf("raw fixture json_utf8 is not valid JSON")
		}
		if err := rejectDuplicateKeys([]byte(text)); err != nil {
			return nil, fmt.Errorf("raw fixture json_utf8: %w", err)
		}
		result.BodyJSON, result.HasBodyJSON = text, true
	}
	outcome, err := strictObject(exchange["outcome"], nil, []string{"response", "failure"})
	if err != nil {
		return nil, fmt.Errorf("raw fixture outcome: %w", err)
	}
	_, hasResponse := outcome["response"]
	_, hasFailure := outcome["failure"]
	if hasResponse == hasFailure {
		return nil, fmt.Errorf("raw fixture outcome needs exactly one of response, failure")
	}
	if hasResponse {
		response, err := strictObject(outcome["response"], []string{"status", "headers", "body_base64"}, nil)
		if err != nil {
			return nil, fmt.Errorf("raw fixture response: %w", err)
		}
		status, err := strictInteger(response["status"])
		if err != nil || status < 200 || status > 599 {
			return nil, fmt.Errorf("raw fixture status must be an integer 200..599")
		}
		headers, err := strictPairs(response["headers"])
		if err != nil {
			return nil, fmt.Errorf("raw fixture response headers: %w", err)
		}
		text, err := strictText(response["body_base64"])
		if err != nil {
			return nil, fmt.Errorf("raw fixture response body: %w", err)
		}
		if err := checkBase64(text); err != nil {
			return nil, fmt.Errorf("raw fixture response body: %w", err)
		}
		if (result.Method == "HEAD" || status == 204 || status == 205 || status == 304) && text != "" {
			return nil, fmt.Errorf("raw fixture bodyless response must carry empty bytes")
		}
		result.Response = &RawResponse{Status: status, Headers: headers, Body: text}
	} else {
		failure, err := strictObject(outcome["failure"], []string{"kind"}, []string{"phase"})
		if err != nil {
			return nil, fmt.Errorf("raw fixture failure: %w", err)
		}
		kind, err := strictText(failure["kind"])
		if err != nil {
			return nil, fmt.Errorf("raw fixture failure: %w", err)
		}
		parsed := &RawFailure{Kind: kind}
		switch kind {
		case "transport":
			phase, ok := failure["phase"]
			if !ok {
				return nil, fmt.Errorf("raw fixture transport failure needs a phase")
			}
			name, err := strictText(phase)
			if err != nil || !rawPhases[name] {
				return nil, fmt.Errorf("raw fixture transport phase is not a known A2 phase")
			}
			parsed.Phase = name
		case "timeout", "body_limit":
			if _, ok := failure["phase"]; ok {
				return nil, fmt.Errorf("raw fixture %s failure takes no phase", kind)
			}
		default:
			return nil, fmt.Errorf("raw fixture failure kind must be transport, timeout or body_limit")
		}
		result.Failure = parsed
	}
	fixture.Exchange = result
	return fixture, nil
}

func checkBase64(text string) error {
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return fmt.Errorf("body base64 does not decode")
	}
	if base64.StdEncoding.EncodeToString(raw) != text {
		return fmt.Errorf("body base64 is not canonical")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 256 {
			return fmt.Errorf("raw fixture nesting exceeds 256 levels")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("raw fixture has an invalid object key")
				}
				if seen[name] {
					return fmt.Errorf("raw fixture has duplicate key %q", name)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("raw fixture has an unexpected delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("raw fixture has trailing input")
	}
	return nil
}

func strictObject(data []byte, required, optional []string) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("expected JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, name := range required {
		allowed[name] = true
		if _, ok := fields[name]; !ok {
			return nil, fmt.Errorf("missing field %q", name)
		}
	}
	for _, name := range optional {
		allowed[name] = true
	}
	for name := range fields {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown field %q", name)
		}
	}
	return fields, nil
}

func strictDictionary(data []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("expected JSON object")
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func strictText(data []byte) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return "", fmt.Errorf("expected JSON string")
	}
	var result string
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return "", err
	}
	if !utf8.ValidString(result) || strings.ContainsRune(result, 0) {
		return "", fmt.Errorf("expected Unicode scalar string")
	}
	return result, nil
}

func strictPairs(data []byte) ([][2]string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("expected JSON array")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(trimmed, &rows); err != nil {
		return nil, err
	}
	var result [][2]string
	for _, row := range rows {
		pair, err := strictPair(row)
		if err != nil {
			return nil, err
		}
		result = append(result, pair)
	}
	return result, nil
}

func strictPair(data []byte) ([2]string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return [2]string{}, fmt.Errorf("expected header pair")
	}
	var cells []json.RawMessage
	if err := json.Unmarshal(trimmed, &cells); err != nil || len(cells) != 2 {
		return [2]string{}, fmt.Errorf("expected header pair")
	}
	name, err := strictText(cells[0])
	if err != nil {
		return [2]string{}, err
	}
	value, err := strictText(cells[1])
	if err != nil {
		return [2]string{}, err
	}
	return [2]string{name, value}, nil
}

// RawScope carries assertion-time native evidence: fixture loading confined
// to the owning project plus the checked native declarations and connection
// policies by identity. It rides the completion context so both attached
// assertion rows and lexical when rows resolve raw targets identically.
type RawScope struct {
	Load        func(path string) (*RawFixture, error)
	Natives     map[string]*NativeDeclaration
	Connections map[string]ConnectionPolicy
}

func (c *programChecker) rawScope(file *resolve.File) *RawScope {
	directory := filepath.Dir(file.Source.Path)
	owner := file.Source.Package.Owner
	scope := &RawScope{Natives: map[string]*NativeDeclaration{}, Connections: c.program.Connections}
	for _, native := range c.program.Natives {
		scope.Natives[native.Symbol.ID] = native
	}
	scope.Load = func(path string) (*RawFixture, error) {
		if path == "" || filepath.IsAbs(path) || !utf8.ValidString(path) {
			return nil, fmt.Errorf("raw fixture %q must be a source-relative path", path)
		}
		real, err := filepath.EvalSymlinks(filepath.Join(directory, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("raw fixture %q is not readable", path)
		}
		confined, err := filepath.EvalSymlinks(owner.Root)
		if err != nil {
			return nil, fmt.Errorf("raw fixture %q escapes its project", path)
		}
		if !project.Contains(confined, real) {
			return nil, fmt.Errorf("raw fixture %q escapes its project", path)
		}
		data, err := os.ReadFile(real)
		if err != nil {
			return nil, fmt.Errorf("raw fixture %q is not readable", path)
		}
		if len(data) > maxRawFixtureBytes {
			return nil, fmt.Errorf("raw fixture %q exceeds its size bound", path)
		}
		fixture, err := ParseRawFixture(data)
		if err != nil {
			return nil, fmt.Errorf("raw fixture %q: %w", path, err)
		}
		return fixture, nil
	}
	return scope
}

// CheckRawFixture binds a loaded fixture to its invoked declaration: the
// target must name the declaration exactly, and every environment key must
// be a credential name the inherited connection reads. A missing key
// simulates absence at runtime.
func (s *RawScope) CheckRawFixture(invoked string, fixture *RawFixture) error {
	if fixture.Target != invoked {
		return fmt.Errorf("raw fixture targets %s, not %s", fixture.Target, invoked)
	}
	native, ok := s.Natives[invoked]
	if !ok {
		return fmt.Errorf("raw fixture target %s is not a checked native declaration", invoked)
	}
	policy := s.Connections[native.Connection]
	for name := range fixture.Environment {
		if name != policy.BearerEnvironment {
			return fmt.Errorf("raw fixture environment %q is not read by the inherited connection", name)
		}
	}
	return nil
}

func convertRawFixture(operation string, fixture *RawFixture) *ir.RawFixture {
	converted := &ir.RawFixture{Operation: operation, Environment: fixture.Environment}
	if fixture.Exchange == nil {
		return converted
	}
	exchange := fixture.Exchange
	result := &ir.RawExchange{Method: exchange.Method, URL: exchange.URL, BodyBase64: exchange.BodyBase64, BodyJSON: exchange.BodyJSON, HasBodyJSON: exchange.HasBodyJSON}
	for _, pair := range exchange.Headers {
		result.Headers = append(result.Headers, [2]string{pair[0], pair[1]})
	}
	if exchange.Response != nil {
		response := &ir.RawResponse{Status: exchange.Response.Status, Body: exchange.Response.Body}
		for _, pair := range exchange.Response.Headers {
			response.Headers = append(response.Headers, [2]string{pair[0], pair[1]})
		}
		result.Response = response
	}
	if exchange.Failure != nil {
		result.Failure = &ir.RawFailure{Kind: exchange.Failure.Kind, Phase: exchange.Failure.Phase}
	}
	converted.Exchange = result
	return converted
}

func strictInteger(data []byte) (int, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return 0, fmt.Errorf("expected JSON integer token")
	}
	for _, c := range trimmed {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("expected unsigned JSON integer token")
		}
	}
	var result int
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return 0, fmt.Errorf("JSON integer out of range")
	}
	return result, nil
}

package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// The standard decoder owns JSON syntax. This pass adds duplicate-key and
// Unicode-scalar validation so decoding cannot silently change path identities.
func validateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	if !json.Valid(data) {
		var value any
		err := json.Unmarshal(data, &value)
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			offset := max(0, min(len(data), int(syntaxError.Offset)-1))
			if strings.Contains(syntaxError.Error(), "unexpected end of JSON input") {
				offset = len(data)
			}
			return &JSONError{Span: source.Span{Start: offset, End: min(len(data), offset+1)}, Err: err}
		}
		return err
	}
	scalarErr := validateStringScalars(data)
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	problems := []error{scalarErr}
	var value func(int) error
	value = func(depth int) error {
		if depth > 256 {
			offset := jsonTokenSpan(data, int(d.InputOffset()), len(data)).Start
			return &JSONError{Span: source.Span{Start: offset, End: min(len(data), offset+1)}, Err: fmt.Errorf("JSON nesting exceeds 256 levels")}
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := map[string]source.Span{}
				for d.More() {
					keyStart := int(d.InputOffset())
					key, err := d.Token()
					if err != nil {
						return err
					}
					name, ok := key.(string)
					if !ok {
						return fmt.Errorf("invalid object key")
					}
					if first, exists := seen[name]; exists {
						problems = append(problems, &JSONError{Span: jsonTokenSpan(data, keyStart, int(d.InputOffset())), Related: []source.Span{first}, Err: fmt.Errorf("duplicate JSON key %q", name)})
					} else {
						seen[name] = jsonTokenSpan(data, keyStart, int(d.InputOffset()))
					}
					if err := value(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := value(depth + 1); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unexpected delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := value(0); err != nil {
		return errors.Join(append(problems, err)...)
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON input")
	}
	return errors.Join(problems...)
}

func validateStringScalars(data []byte) error {
	inString := false
	var problems []error
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		start := i
		i++
		if data[i] != 'u' {
			continue
		}
		unit, _ := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		i += 4
		refusal := func(message string) {
			problems = append(problems, &JSONError{Span: source.Span{Start: start, End: i + 1}, Err: fmt.Errorf("%s", message)})
		}
		if unit >= 0xdc00 && unit <= 0xdfff {
			refusal("unpaired low surrogate in JSON string")
			continue
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				refusal("unpaired high surrogate in JSON string")
				continue
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				refusal("unpaired high surrogate in JSON string")
				continue
			}
			i += 6
		}
	}
	return errors.Join(problems...)
}
func object(data []byte, required, optional []string) (map[string]json.RawMessage, error) {
	fields, err := dictionary(data)
	if err != nil {
		return nil, jsonFieldError(data, err)
	}
	allowed := map[string]bool{}
	var problems []error
	for _, name := range required {
		allowed[name] = true
		if _, ok := fields[name]; !ok {
			problems = append(problems, jsonFieldError(data, fmt.Errorf("missing field %q", name), name))
		}
	}
	for _, name := range optional {
		allowed[name] = true
	}
	for _, name := range sortedKeys(fields) {
		if !allowed[name] {
			problems = append(problems, jsonKeyError(data, fmt.Errorf("unknown field %q", name), name))
		}
	}
	return fields, errors.Join(problems...)
}
func dictionary(data []byte) (map[string]json.RawMessage, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '{' {
		return nil, fmt.Errorf("expected JSON object")
	}
	var result map[string]json.RawMessage
	err := json.Unmarshal(trim, &result)
	return result, err
}
func text(data []byte) (string, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '"' {
		return "", fmt.Errorf("expected JSON string")
	}
	var result string
	err := json.Unmarshal(trim, &result)
	return result, err
}
func array(data []byte) ([]json.RawMessage, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '[' {
		return nil, fmt.Errorf("expected JSON array")
	}
	var result []json.RawMessage
	err := json.Unmarshal(trim, &result)
	return result, err
}
func integer(data []byte) (uint64, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 {
		return 0, fmt.Errorf("expected JSON integer token")
	}
	for _, c := range trim {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("expected unsigned JSON integer token")
		}
	}
	n, err := strconv.ParseUint(string(trim), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("JSON integer out of range")
	}
	return n, nil
}

// JSONError retains byte provenance independently of presentation text.
type JSONError struct {
	Span    source.Span
	Related []source.Span
	Err     error
}

func (e *JSONError) Error() string { return e.Err.Error() }
func (e *JSONError) Unwrap() error { return e.Err }
func jsonTokenSpan(data []byte, start, end int) source.Span {
	for start < end && (data[start] == ' ' || data[start] == '\n' || data[start] == '\r' || data[start] == '\t' || data[start] == ',' || data[start] == ':') {
		start++
	}
	return source.Span{Start: start, End: end}
}

// JSON locations are derived structurally from decoder offsets. Array entries
// use decimal indices; object keys may contain arbitrary escaped Unicode.
type jsonLocation struct {
	value, key source.Span
	hasKey     bool
}

func jsonPathKey(path []string) string {
	if len(path) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(path)
	return string(encoded)
}
func jsonLocations(data []byte) map[string]jsonLocation {
	locations := map[string]jsonLocation{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var walk func([]string, source.Span, bool)
	walk = func(at []string, key source.Span, hasKey bool) {
		start := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err != nil {
			return
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				for decoder.More() {
					keyStart := int(decoder.InputOffset())
					name, err := decoder.Token()
					if err != nil {
						return
					}
					keySpan := jsonTokenSpan(data, keyStart, int(decoder.InputOffset()))
					walk(append(append([]string{}, at...), name.(string)), keySpan, true)
				}
			case '[':
				for i := 0; decoder.More(); i++ {
					walk(append(append([]string{}, at...), strconv.Itoa(i)), source.Span{}, false)
				}
			}
			_, _ = decoder.Token()
		}
		locations[jsonPathKey(at)] = jsonLocation{value: jsonTokenSpan(data, start, int(decoder.InputOffset())), key: key, hasKey: hasKey}
	}
	walk(nil, source.Span{}, false)
	return locations
}
func jsonSpan(data []byte, key bool, path ...string) source.Span {
	locations := jsonLocations(data)
	if location, ok := locations[jsonPathKey(path)]; ok {
		if key && location.hasKey {
			return location.key
		}
		return location.value
	}
	// A missing member is an insertion at its nearest existing container's end.
	for len(path) > 0 {
		path = path[:len(path)-1]
		if location, ok := locations[jsonPathKey(path)]; ok {
			end := max(location.value.Start, location.value.End-1)
			return source.Span{Start: end, End: end}
		}
	}
	end := len(bytes.TrimRight(data, " \n\r\t"))
	return source.Span{Start: end, End: end}
}
func JSONFieldSpan(data []byte, path ...string) source.Span { return jsonSpan(data, false, path...) }
func jsonFieldError(data []byte, err error, path ...string) error {
	if err == nil {
		return nil
	}
	return &JSONError{Span: jsonSpan(data, false, path...), Err: err}
}
func jsonKeyError(data []byte, err error, path ...string) error {
	return &JSONError{Span: jsonSpan(data, true, path...), Err: err}
}

// Nested parsers report offsets relative to their own raw JSON value. Preserve
// those offsets when attaching the value to its parent instead of widening it.
func jsonChildError(data []byte, err error, path ...string) error {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var problems []error
		for _, child := range many.Unwrap() {
			problems = append(problems, jsonChildError(data, child, path...))
		}
		return errors.Join(problems...)
	}
	base := JSONFieldSpan(data, path...)
	if located, ok := err.(*JSONError); ok {
		span := source.Span{Start: base.Start + located.Span.Start, End: base.Start + located.Span.End}
		related := make([]source.Span, len(located.Related))
		for i, origin := range located.Related {
			related[i] = source.Span{Start: base.Start + origin.Start, End: base.Start + origin.End}
		}
		return &JSONError{Span: span, Related: related, Err: located.Err}
	}
	return &JSONError{Span: base, Err: err}
}

// Registry.Validate also serves in-memory mutations, so it returns structural
// paths without depending on raw bytes. Parsing binds those paths to source.
type jsonPathError struct {
	path []string
	key  bool
	err  error
}

func (e *jsonPathError) Error() string       { return e.err.Error() }
func (e *jsonPathError) Unwrap() error       { return e.err }
func jsonAt(err error, path ...string) error { return &jsonPathError{path: path, err: err} }
func bindJSONPaths(data []byte, err error) error {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var problems []error
		for _, child := range many.Unwrap() {
			problems = append(problems, bindJSONPaths(data, child))
		}
		return errors.Join(problems...)
	}
	if p, ok := err.(*jsonPathError); ok {
		return &JSONError{Span: jsonSpan(data, p.key, p.path...), Err: p.err}
	}
	return err
}
func configError(path string, data []byte, err error) error {
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var problems []error
		for _, child := range many.Unwrap() {
			problems = append(problems, configError(path, data, child))
		}
		return errors.Join(problems...)
	}
	var located *JSONError
	if errors.As(err, &located) {
		related := make([]source.RelatedSpan, len(located.Related))
		for i, origin := range located.Related {
			related[i] = source.RelatedSpan{File: path, Span: origin, Note: "first declaration here"}
		}
		return &source.LocatedError{File: path, Span: located.Span, Code: "CAN-PROJECT-CONFIG", Related: related, Err: located.Err}
	}
	// Only project I/O failures may lack a token. Never invent a whole-file
	// underline when a producer has not supplied configuration provenance.
	return fmt.Errorf("%s: %w", path, err)
}

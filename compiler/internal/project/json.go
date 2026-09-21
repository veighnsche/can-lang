package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// The standard decoder owns JSON syntax. This pass adds duplicate-key and
// Unicode-scalar validation so decoding cannot silently change path identities.
func validateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON")
	}
	if err := validateStringScalars(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 256 {
			return fmt.Errorf("JSON nesting exceeds 256 levels")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					if err != nil {
						return err
					}
					name, ok := key.(string)
					if !ok {
						return fmt.Errorf("invalid object key")
					}
					if seen[name] {
						return fmt.Errorf("duplicate JSON key %q", name)
					}
					seen[name] = true
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
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON input")
	}
	return nil
}

func validateStringScalars(data []byte) error {
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		i++
		if data[i] != 'u' {
			continue
		}
		unit, _ := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		i += 4
		if unit >= 0xdc00 && unit <= 0xdfff {
			return fmt.Errorf("unpaired low surrogate in JSON string")
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return fmt.Errorf("unpaired high surrogate in JSON string")
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return fmt.Errorf("unpaired high surrogate in JSON string")
			}
			i += 6
		}
	}
	return nil
}

func object(data []byte, required, optional []string) (map[string]json.RawMessage, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '{' {
		return nil, fmt.Errorf("expected JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trim, &fields); err != nil {
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
	for _, name := range sortedKeys(fields) {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown field %q", name)
		}
	}
	return fields, nil
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

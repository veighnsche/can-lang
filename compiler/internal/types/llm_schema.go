package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// GenerationSchema is the deliberately narrower, expanded Responses profile.
// It is independent of the ordinary codec graph used to validate returned data.
type GenerationSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

func LLMSchema(root *Type) (GenerationSchema, error) {
	if !Equal(root, root) || root.Kind() != Record {
		return GenerationSchema{}, fmt.Errorf("LLM schema at /: root requires an ordinary record")
	}
	active := map[*Type]bool{}
	properties := 0
	pathChild := func(path, key string) string {
		return path + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
	}
	var visit func(*Type, string, int) (any, error)
	visit = func(t *Type, path string, depth int) (any, error) {
		fail := func(reason string) (any, error) {
			if path == "" {
				path = "/"
			}
			return nil, fmt.Errorf("LLM schema at %s: %s", path, reason)
		}
		if t.Kind() == Primitive {
			name := map[string]string{"str": "string", "bool": "boolean", "int": "integer", "float": "number"}[t.Declaration()]
			if name == "" {
				return fail("unsupported primitive")
			}
			return map[string]any{"type": name}, nil
		}
		if t.Kind() != Record && t.Kind() != Array {
			return fail("unsupported " + string(t.Kind()))
		}
		depth++
		if depth > 8 {
			return fail("container depth exceeds 8")
		}
		if active[t] {
			return fail("recursive output type")
		}
		active[t] = true
		defer delete(active, t)
		if t.Kind() == Array {
			item, err := visit(t.Element(), path+"/*", depth)
			if err != nil {
				return nil, err
			}
			return map[string]any{"type": "array", "items": item}, nil
		}
		fields := t.Fields()
		properties += len(fields)
		if properties > 1024 {
			return fail("expanded properties exceed 1024")
		}
		props := map[string]any{}
		required := make([]string, 0, len(fields))
		for _, field := range fields {
			node, err := visit(field.Type, pathChild(path, field.Name), depth)
			if err != nil {
				return nil, err
			}
			props[field.Name] = node
			required = append(required, field.Name)
		}
		return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}, nil
	}
	schema, err := visit(root, "", 0)
	if err != nil {
		return GenerationSchema{}, err
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return GenerationSchema{}, err
	}
	if len(data) > 65536 {
		return GenerationSchema{}, fmt.Errorf("LLM schema at /: UTF-8 schema bytes exceed 65536")
	}
	identity, err := json.Marshal([]any{root.Identity(), json.RawMessage(data)})
	if err != nil {
		return GenerationSchema{}, err
	}
	digest := sha256.Sum256(identity)
	return GenerationSchema{Name: fmt.Sprintf("can_%x", digest[:16]), Schema: data}, nil
}

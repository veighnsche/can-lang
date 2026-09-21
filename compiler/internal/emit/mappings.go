package emit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// NUL cannot occur literally in quoted authored strings. These private assembly
// tokens travel with statements and are removed before any TypeScript is parsed.
func mappingMark(id string, span source.Span, operation string) string {
	data, _ := json.Marshal(ir.Mapping{Source: id, Start: span.Start, End: span.End, Operation: operation})
	return "\x00" + base64.RawStdEncoding.EncodeToString(data) + "\x00"
}
func extractMappings(text string) (string, []ir.Mapping, error) {
	var out strings.Builder
	var mappings []ir.Mapping
	line, column := 1, 0
	for len(text) > 0 {
		next := strings.IndexByte(text, 0)
		if next < 0 {
			out.WriteString(text)
			break
		}
		prefix := text[:next]
		out.WriteString(prefix)
		for _, r := range prefix {
			if r == '\n' {
				line++
				column = 0
			} else {
				column += utf16.RuneLen(r)
			}
		}
		text = text[next+1:]
		end := strings.IndexByte(text, 0)
		if end < 0 {
			return "", nil, fmt.Errorf("unfinished mapping token")
		}
		data, err := base64.RawStdEncoding.DecodeString(text[:end])
		if err != nil {
			return "", nil, err
		}
		var mapping ir.Mapping
		if err = json.Unmarshal(data, &mapping); err != nil {
			return "", nil, err
		}
		mapping.Line = line
		mapping.Column = column
		mappings = append(mappings, mapping)
		text = text[end+1:]
	}
	return out.String(), mappings, nil
}

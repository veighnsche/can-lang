package main

import (
	"fmt"
	"strings"
)

func (e *emitter) wholeSuccessBinding(p Pattern, tmp string) (string, error) {
	if p.Name != "Ok" || len(p.TypeArgs) == 0 {
		return tmp, nil
	}
	if len(p.TypeArgs) != 1 {
		return "", fmt.Errorf("typed Ok pattern needs one success type")
	}
	if fields, ok := e.recs[p.TypeArgs[0]]; ok {
		return tsRecordProjection(tmp, fields, false), nil
	}
	if !valueSuccess(p.TypeArgs[0], e.variants[p.TypeArgs[0]] != nil || e.brands[p.TypeArgs[0]] != "") {
		return "", fmt.Errorf("unknown typed Ok success %s", p.TypeArgs[0])
	}
	return tmp + ".value", nil
}

func (e *emitter) emitWholeOk(s *Small) (string, error) {
	if len(s.TypeArgs) != 1 || len(s.Args) != 1 || (s.Args[0].HasName && s.Args[0].Name != "value") {
		return "", fmt.Errorf("typed Ok takes one type and one whole value")
	}
	v, err := e.emitValue(s.Args[0].V)
	if err != nil {
		return "", err
	}
	if fields, ok := e.recs[s.TypeArgs[0]]; ok {
		// Structural parameter annotation needs only the field dependencies
		// already imported for the declared flattened result. No extra nominal
		// record import, spread, or repeated evaluation of the operand.
		var types []string
		for _, f := range fields {
			t, err := tsTypeB(f[1], e.brands, e.recs, e.variants, e.errTypes)
			if err != nil {
				return "", err
			}
			types = append(types, tsField(f[0], t))
		}
		tmp := e.fresh()
		return "((" + tmp + ": { " + strings.Join(types, "; ") + " }) => (" + tsRecordProjection(tmp, fields, true) + "))(" + v + ")", nil
	}
	if !valueSuccess(s.TypeArgs[0], e.variants[s.TypeArgs[0]] != nil || e.brands[s.TypeArgs[0]] != "") {
		return "", fmt.Errorf("unknown typed Ok success %s", s.TypeArgs[0])
	}
	return "{ " + tsTag + `: "ok", value: ` + v + " }", nil
}

func tsRecordProjection(v string, fields [][2]string, outcome bool) string {
	parts := []string{}
	if outcome {
		parts = append(parts, tsTag+`: "ok" as const`)
	}
	for _, f := range fields {
		parts = append(parts, tsField(f[0], v+"."+f[0]))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

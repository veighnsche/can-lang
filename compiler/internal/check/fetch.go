package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"mime"
	"strings"
)

func fetchJSONContentType(text string) bool {
	media, params, err := mime.ParseMediaType(text)
	if err != nil || media != "application/json" || strings.Count(text, ";") != len(params) {
		return false
	}
	for name, value := range params {
		if name != "charset" || !strings.EqualFold(value, "utf-8") {
			return false
		}
	}
	return true
}
func checkFetchContentType(d *syntax.FetchDecl, policy ConnectionPolicy) error {
	if d.BodyEncoding == nil || d.BodyEncoding.Text != "json" {
		return nil
	}
	var current string
	present := false
	for _, header := range policy.Headers {
		if strings.EqualFold(strings.ReplaceAll(header.Name, "_", "-"), "content-type") {
			current = header.Value
			present = true
		}
	}
	for _, header := range d.Headers {
		if !strings.EqualFold(strings.ReplaceAll(header.Name.Text, "_", "-"), "content-type") {
			continue
		}
		switch value := header.Value.(type) {
		case *syntax.LiteralExpr:
			if value.Token.Kind != syntax.String {
				return nil
			}
			current = value.Token.Value
			present = true
		case *syntax.ArrayExpr:
			var parts []string
			for _, item := range value.Elements {
				literal, ok := item.Value.(*syntax.LiteralExpr)
				if !ok || item.Spread || literal.Token.Kind != syntax.String {
					return nil
				}
				parts = append(parts, literal.Token.Value)
			}
			if len(parts) == 0 {
				return nil
			}
			current = strings.Join(parts, ", ")
			present = true
		default:
			return nil
		}
	}
	if present && !fetchJSONContentType(current) {
		return fmt.Errorf("JSON fetch body requires application/json Content-Type with optional UTF-8 charset")
	}
	return nil
}
func finishFetchPlan(plan *ir.Fetch) error {
	result := plan.Result
	if result.Declaration() == "can.std.http@1::response" {
		plan.Envelope = result.Identity()
		result = result.Arguments()[0]
	}
	switch {
	case scalar(result, "str"):
		plan.ResultMode = "text"
	case result.Declaration() == "can.std.bytes@1::buffer":
		plan.ResultMode = "bytes"
	default:
		plan.ResultMode = "json"
		schema, err := types.Schema(result)
		if err != nil {
			return err
		}
		plan.ResultSchema = &schema
	}
	if plan.BodyMode == "json" {
		schema, err := types.Schema(plan.Body.Type)
		if err != nil {
			return err
		}
		plan.BodySchema = &schema
	}
	return nil
}

// Inspect only statically present string values, including grouped arrays and
// literal spreads. Dynamic expressions retain the transport admission check.
func checkLiteralHeaderValues(expr syntax.Expr) error {
	switch value := expr.(type) {
	case *syntax.LiteralExpr:
		if value.Token.Kind == syntax.String {
			for _, r := range value.Token.Value {
				if r > 255 || r == 0 || r == '\r' || r == '\n' {
					return fmt.Errorf("invalid literal request header value")
				}
			}
		}
	case *syntax.GroupExpr:
		return checkLiteralHeaderValues(value.Value)
	case *syntax.ArrayExpr:
		for _, item := range value.Elements {
			if err := checkLiteralHeaderValues(item.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

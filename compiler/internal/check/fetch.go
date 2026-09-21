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
		parts, known := literalFetchHeader(header.Value)
		if !known {
			return nil
		}
		current = strings.Join(parts, ", ")
		present = len(parts) > 0
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

func fetchUngroup(expr syntax.Expr) syntax.Expr {
	for {
		group, ok := expr.(*syntax.GroupExpr)
		if !ok {
			return expr
		}
		expr = group.Value
	}
}

// Return only completely known native header values. An empty array removes
// a connection header; an unknown expression defers admission to transport.
func literalFetchHeader(expr syntax.Expr) ([]string, bool) {
	expr = fetchUngroup(expr)
	if literal, ok := expr.(*syntax.LiteralExpr); ok && literal.Token.Kind == syntax.String {
		return []string{literal.Token.Value}, true
	}
	values, ok := expr.(*syntax.ArrayExpr)
	if !ok {
		return nil, false
	}
	var result []string
	for _, item := range values.Elements {
		if item.Group != nil {
			return nil, false
		}
		child := fetchUngroup(item.Value)
		if item.Spread {
			if _, ok := child.(*syntax.ArrayExpr); !ok {
				return nil, false
			}
			nested, known := literalFetchHeader(child)
			if !known {
				return nil, false
			}
			result = append(result, nested...)
		} else {
			literal, ok := child.(*syntax.LiteralExpr)
			if !ok || literal.Token.Kind != syntax.String {
				return nil, false
			}
			result = append(result, literal.Token.Value)
		}
	}
	return result, true
}

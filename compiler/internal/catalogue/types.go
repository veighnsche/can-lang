package catalogue

import (
	"fmt"
	"strings"
)

// typeRef parses only catalogue descriptor notation, not authored Can syntax.
// Source parsing/type inference remain separate compiler passes.
type typeRef struct {
	name      string
	arguments []typeRef
}

func parseType(text string) (typeRef, error) {
	pos := 0
	var parse func(int) (typeRef, error)
	parse = func(depth int) (typeRef, error) {
		if depth > 64 {
			return typeRef{}, fmt.Errorf("type descriptor nesting limit")
		}
		start := pos
		for pos < len(text) && ((text[pos] >= 'a' && text[pos] <= 'z') || (text[pos] >= 'A' && text[pos] <= 'Z') || (text[pos] >= '0' && text[pos] <= '9') || text[pos] == '_' || text[pos] == ':') {
			pos++
		}
		name := text[start:pos]
		parts := strings.Split(name, "::")
		if len(parts) > 2 || len(parts) == 0 {
			return typeRef{}, fmt.Errorf("invalid type descriptor %q", text)
		}
		for _, p := range parts {
			if !identifier.MatchString(p) {
				return typeRef{}, fmt.Errorf("invalid type descriptor %q", text)
			}
		}
		r := typeRef{name: name}
		if pos < len(text) && text[pos] == '<' {
			pos++
			for {
				arg, err := parse(depth + 1)
				if err != nil {
					return typeRef{}, err
				}
				r.arguments = append(r.arguments, arg)
				if pos >= len(text) {
					return typeRef{}, fmt.Errorf("unterminated type arguments")
				}
				if text[pos] == '>' {
					pos++
					break
				}
				if text[pos] != ',' {
					return typeRef{}, fmt.Errorf("invalid type arguments")
				}
				pos++
			}
		}
		for pos+1 < len(text) && text[pos:pos+2] == "[]" {
			r = typeRef{name: "[]", arguments: []typeRef{r}}
			pos += 2
		}
		return r, nil
	}
	r, err := parse(0)
	if err != nil {
		return r, err
	}
	if pos != len(text) {
		return r, fmt.Errorf("trailing type descriptor input: %q", text)
	}
	return r, nil
}
func (r typeRef) String() string {
	if r.name == "[]" {
		return r.arguments[0].String() + "[]"
	}
	if len(r.arguments) == 0 {
		return r.name
	}
	xs := make([]string, len(r.arguments))
	for i, a := range r.arguments {
		xs[i] = a.String()
	}
	return r.name + "<" + strings.Join(xs, ",") + ">"
}
func (r typeRef) substitute(args map[string]string) typeRef {
	if value, ok := args[r.name]; ok && len(r.arguments) == 0 {
		v, err := parseType(value)
		if err != nil {
			panic(err)
		}
		return v
	}
	out := typeRef{name: r.name}
	for _, a := range r.arguments {
		out.arguments = append(out.arguments, a.substitute(args))
	}
	return out
}
func substitute(text string, args map[string]string) string {
	r, err := parseType(text)
	if err != nil {
		panic(err)
	}
	return r.substitute(args).String()
}
func (c *Catalogue) checkType(text string, ps map[string]bool, allowVoid bool) error {
	r, err := parseType(text)
	if err != nil {
		return err
	}
	var check func(typeRef, bool) error
	check = func(r typeRef, allowVoid bool) error {
		if r.name == "[]" {
			return check(r.arguments[0], false)
		}
		if r.name == "void" {
			if allowVoid && len(r.arguments) == 0 {
				return nil
			}
			return fmt.Errorf("void is not a data type")
		}
		if ps[r.name] || r.name == "int" || r.name == "float" || r.name == "bool" || r.name == "str" {
			if len(r.arguments) > 0 {
				return fmt.Errorf("unexpected generic arguments for %s", r.name)
			}
			return nil
		}
		arity := -1
		if d, ok := c.types[r.name]; ok {
			arity = len(d.Parameters)
		}
		if d, ok := c.errors[r.name]; ok {
			arity = len(d.Parameters)
		}
		if arity < 0 {
			return fmt.Errorf("unknown catalogue type %s", r.name)
		}
		if len(r.arguments) != arity {
			return fmt.Errorf("generic arity mismatch for %s", r.name)
		}
		for _, a := range r.arguments {
			if err := check(a, false); err != nil {
				return err
			}
		}
		return nil
	}
	return check(r, allowVoid)
}

// TypeAdmission is evidence from the compiler's checked project type graph,
// needed only for project-owned nominal types. It is not an extension mechanism
// for primitive keys, opaque types, or catalogue definitions.
type TypeAdmission func(typeName, constraint string) bool

func (c *Catalogue) admits(text, constraint string, project TypeAdmission) bool {
	r, err := parseType(text)
	if err != nil {
		return false
	}
	seen := map[string]bool{}
	var admit func(typeRef, string) bool
	admit = func(r typeRef, constraint string) bool {
		if r.name == "void" {
			return false
		}
		scalar := r.name == "int" || r.name == "float" || r.name == "bool" || r.name == "str"
		if constraint == "map_key" {
			return len(r.arguments) == 0 && (r.name == "int" || r.name == "bool" || r.name == "str")
		}
		if constraint == "sort_key" {
			return len(r.arguments) == 0 && scalar
		}
		if scalar {
			return len(r.arguments) == 0 && (constraint == "data" || constraint == "wire" || constraint == "sql_scalar")
		}
		if r.name == "[]" {
			return (constraint == "data" || constraint == "wire") && admit(r.arguments[0], constraint)
		}
		d, known := c.types[r.name]
		if e, ok := c.errors[r.name]; ok {
			d = TypeDecl{Name: e.Name, Kind: "record", Parameters: e.Parameters, Fields: e.Fields}
			known = true
		}
		if !known {
			return (constraint == "data" || constraint == "wire" || constraint == "form" || constraint == "sql_parameters" || constraint == "sql_row" || constraint == "failure_variant") && project != nil && project(r.String(), constraint)
		}
		if len(d.Parameters) != len(r.arguments) {
			return false
		}
		args := map[string]string{}
		for i, p := range d.Parameters {
			if !admit(r.arguments[i], p.Constraint) {
				return false
			}
			args[p.Name] = r.arguments[i].String()
		}
		if constraint == "data" {
			return true
		}
		if constraint == "sql_scalar" {
			if r.name == "bytes::buffer" {
				return true
			}
			if r.name == "option::value" && len(r.arguments) == 1 && r.arguments[0].name != "option::value" {
				return admit(r.arguments[0], "sql_scalar")
			}
			return false
		}
		if constraint == "failure_variant" {
			return d.Kind == "variant"
		}
		if d.Kind == "opaque" {
			return false
		}
		if constraint == "form" || constraint == "sql_parameters" || constraint == "sql_row" {
			if d.Kind != "record" {
				return false
			}
			for _, f := range d.Fields {
				value, _ := parseType(substitute(f.Type, args))
				if constraint == "form" {
					s := value.String()
					if s != "str" && s != "str[]" && s != "option::value<str>" {
						return false
					}
				} else if !admit(value, "sql_scalar") {
					return false
				}
			}
			return true
		}
		if constraint != "wire" {
			return false
		}
		key := r.String() + ":" + constraint
		if seen[key] {
			return true
		}
		seen[key] = true
		for _, f := range d.Fields {
			value, _ := parseType(substitute(f.Type, args))
			if !admit(value, constraint) {
				return false
			}
		}
		for _, leaf := range d.Leaves {
			value, _ := parseType(substitute(leaf, args))
			if !admit(value, constraint) {
				return false
			}
		}
		return true
	}
	return admit(r, constraint)
}

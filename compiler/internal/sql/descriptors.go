package sql

import (
	"fmt"
	"strconv"
	"strings"
)

// Segment is one piece of a checked statement: static Text, or Param N with
// 1-based parameter number. Segments tile the exact statement bytes; text
// resembling $N inside quotes or comments stays inside Text.
type Segment struct {
	Text  string `json:"text,omitempty"`
	Param int    `json:"param,omitempty"`
}

// Descriptor is a fully checked manifest descriptor statement: cardinality
// shape, contiguous parameters, and template segments. Type correspondence
// with the parameter and row records is resolved by check/sql.go; this
// package never sees Can types.
type Descriptor struct {
	Name        string    `json:"name"`
	Cardinality string    `json:"cardinality"`
	Kind        string    `json:"kind"`
	Segments    []Segment `json:"segments"`
	// Total is the highest parameter number: len(parameters)+1 for
	// row-returning descriptors (limit included), len(parameters) for
	// execute. Limit is the row-limit parameter number, 0 for execute.
	Total   int `json:"total"`
	Limit   int `json:"limit"`
	Version int `json:"version"`
}

func rowReturning(cardinality string) bool { return cardinality != "execute" }

// CheckDescriptor validates one manifest descriptor statement: exactly one
// approved statement, contiguous $1..$M parameters with the top-level LIMIT
// as the exclusive trailing number for row-returning shapes, and no
// RETURNING for mutations. It returns template segments tiling the input.
func CheckDescriptor(name, statement string, parameters []string, cardinality string, rowLimit uint64) (Descriptor, error) {
	out := Descriptor{Name: name, Cardinality: cardinality}
	fail := func(format string, args ...any) (Descriptor, error) {
		return Descriptor{}, fmt.Errorf("sql descriptor %q: %s", name, fmt.Sprintf(format, args...))
	}
	if rowReturning(cardinality) {
		out.Total = len(parameters) + 1
		out.Limit = int(rowLimit)
		if rowLimit != uint64(out.Total) {
			return fail("row_limit_parameter is %d, want %d after %d application parameters", rowLimit, out.Total, len(parameters))
		}
	} else {
		out.Total = len(parameters)
		if rowLimit != 0 {
			return fail("execute must not declare row_limit_parameter")
		}
	}
	stmts, failure, err := Parse(statement)
	if err != nil {
		return Descriptor{}, err
	}
	if failure != nil {
		return fail("%s at character %d", failure.Message, failure.Cursor)
	}
	if len(stmts) != 1 {
		return fail("%d statements, want 1", len(stmts))
	}
	stmt := stmts[0]
	out.Kind = stmt.Kind
	version, err := Version()
	if err != nil {
		return Descriptor{}, err
	}
	out.Version = version
	tokens, failure, err := Scan(statement)
	if err != nil {
		return Descriptor{}, err
	}
	if failure != nil {
		return fail("%s at character %d", failure.Message, failure.Cursor)
	}
	type site struct{ number, start, end int }
	var sites []site
	last := 0
	for _, token := range tokens {
		if token.Start < last {
			return Descriptor{}, fmt.Errorf("sql descriptor %q: scanner tokens out of order", name)
		}
		last = token.Start
		if token.Kind != "PARAM" {
			continue
		}
		text := statement[token.Start:token.End]
		number, err := strconv.Atoi(strings.TrimPrefix(text, "$"))
		if err != nil || "$"+strconv.Itoa(number) != text {
			return Descriptor{}, fmt.Errorf("sql descriptor %q: parameter token %q", name, text)
		}
		sites = append(sites, site{number, token.Start, token.End})
	}
	present := map[int]bool{}
	for _, s := range sites {
		present[s.number] = true
	}
	for n := range present {
		if n < 1 || n > out.Total {
			return fail("$%d has no declared parameter", n)
		}
	}
	if rowReturning(cardinality) {
		if stmt.Kind != "SelectStmt" {
			return fail("cardinality %s requires SELECT, got %s", cardinality, stmt.Kind)
		}
		if !stmt.HasLimit {
			return fail("unbounded SELECT requires LIMIT $%d", out.Limit)
		}
		if stmt.LimitParam != out.Limit {
			return fail("top-level LIMIT must be $%d", out.Limit)
		}
		uses := 0
		for _, s := range sites {
			if s.number == out.Limit {
				uses++
			}
		}
		if uses != 1 {
			return fail("row-limit parameter $%d appears %d times, want once in LIMIT", out.Limit, uses)
		}
	} else {
		switch stmt.Kind {
		case "InsertStmt", "UpdateStmt", "DeleteStmt":
		default:
			return fail("execute requires INSERT, UPDATE, or DELETE, got %s", stmt.Kind)
		}
		if stmt.Returning {
			return fail("RETURNING is not admitted")
		}
	}
	for n := 1; n <= out.Total; n++ {
		if !present[n] {
			if rowReturning(cardinality) && n == out.Total {
				return fail("row-limit parameter $%d is not used", n)
			}
			return fail("parameter %q ($%d) is not used", parameters[n-1], n)
		}
	}
	cursor := 0
	for _, s := range sites {
		if literal := statement[cursor:s.start]; literal != "" {
			out.Segments = append(out.Segments, Segment{Text: literal})
		}
		out.Segments = append(out.Segments, Segment{Param: s.number})
		cursor = s.end
	}
	if literal := statement[cursor:]; literal != "" {
		out.Segments = append(out.Segments, Segment{Text: literal})
	}
	return out, nil
}

package check

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Pattern-matrix usefulness partitions only at written literal/range/length
// boundaries. Constructor fields become columns. It neither evaluates source
// expressions nor guesses arbitrary predicates. A finite work bound diagnoses
// exceptionally large authored products instead of risking compiler exhaustion.
type patternCell struct {
	key     string
	integer *big.Int
	length  int
	fields  []*types.Type
}

func patternsUseful(matrix [][]*ir.Pattern, query []*ir.Pattern, columns []*types.Type) (bool, error) {
	work := 0
	var visit func([][]*ir.Pattern, []*ir.Pattern, []*types.Type) (bool, error)
	visit = func(rows [][]*ir.Pattern, q []*ir.Pattern, ts []*types.Type) (bool, error) {
		work++
		if work > 100000 {
			return false, fmt.Errorf("pattern coverage exceeds finite work budget")
		}
		if len(rows) == 0 {
			return true, nil
		}
		if len(q) == 0 {
			return false, nil
		}
		for _, row := range rows {
			all := true
			for _, p := range row {
				if p.Kind != "any" {
					all = false
					break
				}
			}
			if all {
				return false, nil
			}
		}
		if q[0].Kind == "or" {
			for _, p := range q[0].Children {
				next := append([]*ir.Pattern{p}, q[1:]...)
				yes, err := visit(rows, next, ts)
				if err != nil || yes {
					return yes, err
				}
			}
			return false, nil
		}
		var expanded [][]*ir.Pattern
		var expand func([]*ir.Pattern)
		expand = func(row []*ir.Pattern) {
			if row[0].Kind == "or" {
				for _, p := range row[0].Children {
					expand(append([]*ir.Pattern{p}, row[1:]...))
				}
			} else {
				expanded = append(expanded, row)
			}
		}
		for _, row := range rows {
			expand(row)
		}
		rows = expanded
		allAny := q[0].Kind == "any"
		for _, row := range rows {
			if row[0].Kind != "any" {
				allAny = false
			}
		}
		if allAny {
			rest := make([][]*ir.Pattern, len(rows))
			for i, row := range rows {
				rest[i] = row[1:]
			}
			return visit(rest, q[1:], ts[1:])
		}
		patterns := []*ir.Pattern{q[0]}
		for _, row := range rows {
			patterns = append(patterns, row[0])
		}
		for _, cell := range patternCells(ts[0], patterns) {
			head, ok := specializePattern(q[0], cell)
			if !ok {
				continue
			}
			next := append(head, q[1:]...)
			var reduced [][]*ir.Pattern
			for _, row := range rows {
				if prefix, ok := specializePattern(row[0], cell); ok {
					reduced = append(reduced, append(prefix, row[1:]...))
				}
			}
			nextTypes := append(append([]*types.Type{}, cell.fields...), ts[1:]...)
			yes, err := visit(reduced, next, nextTypes)
			if err != nil || yes {
				return yes, err
			}
		}
		return false, nil
	}
	return visit(matrix, query, columns)
}
func patternCells(t *types.Type, patterns []*ir.Pattern) []patternCell {
	if t.Kind() == types.Record || t.Kind() == types.Error || t.Kind() == types.Variant {
		leaves := []*types.Type{t}
		if t.Kind() == types.Variant {
			leaves = t.Leaves()
		}
		out := make([]patternCell, 0, len(leaves))
		for _, leaf := range leaves {
			cell := patternCell{key: leaf.Identity()}
			for _, f := range leaf.Fields() {
				cell.fields = append(cell.fields, f.Type)
			}
			out = append(out, cell)
		}
		return out
	}
	if t.Kind() == types.Array {
		maximum := 0
		for _, p := range patterns {
			if p.Kind == "array" && len(p.Children) > maximum {
				maximum = len(p.Children)
			}
		}
		out := make([]patternCell, 0, maximum+2)
		for n := 0; n <= maximum+1; n++ {
			cell := patternCell{key: "array", length: n}
			for i := 0; i < n; i++ {
				cell.fields = append(cell.fields, t.Element())
			}
			out = append(out, cell)
		}
		return out
	}
	if scalar(t, "int") {
		edges := map[string]*big.Int{}
		for _, p := range patterns {
			if p.Kind != "literal" && p.Kind != "range" {
				continue
			}
			lo, _ := new(big.Int).SetString(p.Text, 10)
			hi := new(big.Int).Set(lo)
			if p.Kind == "range" {
				hi, _ = new(big.Int).SetString(p.Upper, 10)
			}
			hi.Add(hi, big.NewInt(1))
			edges[lo.String()] = lo
			edges[hi.String()] = hi
		}
		sorted := make([]*big.Int, 0, len(edges))
		for _, x := range edges {
			sorted = append(sorted, x)
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Cmp(sorted[j]) < 0 })
		if len(sorted) == 0 {
			return []patternCell{{integer: big.NewInt(0)}}
		}
		out := []patternCell{{integer: new(big.Int).Sub(sorted[0], big.NewInt(1))}}
		for _, x := range sorted {
			out = append(out, patternCell{integer: x})
		}
		return out
	}
	if scalar(t, "bool") {
		return []patternCell{{key: "literal:true"}, {key: "literal:false"}}
	}
	if scalar(t, "str") || scalar(t, "float") {
		keys := map[string]bool{}
		for _, p := range patterns {
			if p.Kind == "literal" {
				keys["literal:"+p.Text] = true
			}
		}
		names := make([]string, 0, len(keys))
		for key := range keys {
			names = append(names, key)
		}
		sort.Strings(names)
		out := []patternCell{{key: "other"}}
		for _, key := range names {
			out = append(out, patternCell{key: key})
		}
		return out
	}
	return []patternCell{{key: t.Identity()}}
}
func specializePattern(p *ir.Pattern, cell patternCell) ([]*ir.Pattern, bool) {
	if p.Kind == "any" {
		children := make([]*ir.Pattern, len(cell.fields))
		for i, t := range cell.fields {
			children[i] = &ir.Pattern{Kind: "any", Type: t}
		}
		return children, true
	}
	switch p.Kind {
	case "nominal":
		if p.Text == cell.key {
			return append([]*ir.Pattern{}, p.Children...), true
		}
	case "array":
		if cell.key != "array" || p.Rest == nil && cell.length != len(p.Children) || p.Rest != nil && cell.length < len(p.Children) {
			return nil, false
		}
		children := append([]*ir.Pattern{}, p.Children...)
		for len(children) < cell.length {
			children = append(children, &ir.Pattern{Kind: "any", Type: p.Type.Element()})
		}
		return children, true
	case "literal", "range":
		if cell.integer != nil {
			lo, _ := new(big.Int).SetString(p.Text, 10)
			hi := lo
			if p.Kind == "range" {
				hi, _ = new(big.Int).SetString(p.Upper, 10)
			}
			return nil, cell.integer.Cmp(lo) >= 0 && cell.integer.Cmp(hi) <= 0
		}
		return nil, cell.key == "literal:"+p.Text
	}
	return nil, false
}

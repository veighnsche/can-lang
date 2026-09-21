package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	resolution "github.com/veighnsche/can-lang/compiler/internal/resolve"
	checkedtypes "github.com/veighnsche/can-lang/compiler/internal/types"
)

type typeReport struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Kind          string               `json:"kind"`
	Types         []concreteTypeReport `json:"types"`
}
type concreteTypeReport struct {
	Identity    string            `json:"identity"`
	Declaration string            `json:"declaration,omitempty"`
	Kind        checkedtypes.Kind `json:"kind"`
	Arguments   []string          `json:"arguments,omitempty"`
	Fields      []typeFieldReport `json:"fields,omitempty"`
	Leaves      []string          `json:"leaves,omitempty"`
	Element     string            `json:"element,omitempty"`
	Result      string            `json:"result,omitempty"`
	Inputs      []string          `json:"inputs,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	Equality    bool              `json:"equality"`
}
type typeFieldReport struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// inspect-types is explicitly declaration inspection, not a claim that arbitrary
// expressions or completion paths have passed their later checking passes.
func runInspectTypes(stdout, stderr io.Writer, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: canlc inspect-types PROJECT_DIRECTORY")
		return 2
	}
	graph, err := project.Load(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	world, err := resolution.Build(graph)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	model, err := checkedtypes.CheckDeclarations(world)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := check.ErrorDeclarations(world); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := typeReport{SchemaVersion: 1, Kind: "can.declaration-types", Types: []concreteTypeReport{}}
	for _, typ := range model.Types() {
		item := concreteTypeReport{Identity: typ.Identity(), Declaration: typ.Declaration(), Kind: typ.Kind(), Equality: checkedtypes.EqualityEligible(typ)}
		for _, arg := range typ.Arguments() {
			item.Arguments = append(item.Arguments, arg.Identity())
		}
		for _, field := range typ.Fields() {
			item.Fields = append(item.Fields, typeFieldReport{field.Name, field.Type.Identity()})
		}
		for _, leaf := range typ.Leaves() {
			item.Leaves = append(item.Leaves, leaf.Identity())
		}
		if typ.Element() != nil {
			item.Element = typ.Element().Identity()
		}
		if typ.Result() != nil {
			item.Result = typ.Result().Identity()
		}
		for _, input := range typ.Inputs() {
			item.Inputs = append(item.Inputs, input.Identity())
		}
		for _, failure := range typ.Errors() {
			item.Errors = append(item.Errors, failure.Identity())
		}
		report.Types = append(report.Types, item)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

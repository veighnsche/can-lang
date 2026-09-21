package check

import (
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type FailureField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type FailureShape struct {
	Identity    string         `json:"identity"`
	Kind        types.Kind     `json:"kind"`
	Declaration string         `json:"declaration,omitempty"`
	Arguments   []string       `json:"arguments"`
	Fields      []FailureField `json:"fields"`
	Leaves      []string       `json:"leaves"`
	Element     string         `json:"element,omitempty"`
	Result      string         `json:"result,omitempty"`
	Inputs      []string       `json:"inputs"`
	Errors      []string       `json:"errors"`
}
type ErrorPlan struct {
	Declarations []ErrorDeclaration `json:"declarations"`
	Shapes       []FailureShape     `json:"shapes"`
}

// Plan serializes checked type evidence, not source type spellings or a second
// authored grammar. The same nominal data identities reach emitted error values.
func (r *ErrorRegistry) Plan(bound ErrorBound) ErrorPlan {
	shapes := map[string]FailureShape{}
	var visit func(*types.Type)
	visit = func(t *types.Type) {
		if _, ok := shapes[t.Identity()]; ok {
			return
		}
		s := FailureShape{Identity: t.Identity(), Kind: t.Kind(), Declaration: t.Declaration(), Arguments: []string{}, Fields: []FailureField{}, Leaves: []string{}, Inputs: []string{}, Errors: []string{}}
		shapes[t.Identity()] = s
		for _, a := range t.Arguments() {
			s.Arguments = append(s.Arguments, a.Identity())
			visit(a)
		}
		for _, f := range t.Fields() {
			s.Fields = append(s.Fields, FailureField{f.Name, f.Type.Identity()})
			visit(f.Type)
		}
		for _, leaf := range t.Leaves() {
			s.Leaves = append(s.Leaves, leaf.Identity())
			visit(leaf)
		}
		if t.Element() != nil {
			s.Element = t.Element().Identity()
			visit(t.Element())
		}
		if t.Result() != nil {
			s.Result = t.Result().Identity()
			visit(t.Result())
		}
		for _, input := range t.Inputs() {
			s.Inputs = append(s.Inputs, input.Identity())
			visit(input)
		}
		for _, failure := range t.Errors() {
			s.Errors = append(s.Errors, failure.Identity())
			visit(failure)
		}
		shapes[t.Identity()] = s
	}
	for _, error := range bound.entries {
		visit(error.Type)
	}
	plan := ErrorPlan{Declarations: r.Declarations(), Shapes: []FailureShape{}}
	for _, shape := range shapes {
		plan.Shapes = append(plan.Shapes, shape)
	}
	sort.Slice(plan.Shapes, func(i, j int) bool { return plan.Shapes[i].Identity < plan.Shapes[j].Identity })
	return plan
}

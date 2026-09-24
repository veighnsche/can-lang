package emit

import (
	"strings"
	"testing"
)

// T24 lowers elided browser handles to the shared harness scope value,
// like every other ingress scope.
const browserElisionSource = `package app
    provides []
    uses [browser]
fn void on_press
    emits []
    given
        browser::event e
        near browser::state<int> cell
    asserts
        sample: browser::event("click", "save", "", "") => ok
    match call browser::read_state(cell)
        browser::disposed => ok
        ok browser::snapshot<int> snap => ok
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func TestBrowserElisionLowersScopeRequest(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserElisionSource})
	dependencies := httpDependencies(t)
	artifacts, err := AssertionModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	joined := strings.Join(bodies, "\n")
	if !strings.Contains(joined, "$canScopeRequest($canContext)") {
		t.Fatal("expected elided browser handles to lower through the harness scope request")
	}
}

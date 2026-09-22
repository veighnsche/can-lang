package driver

import (
	"context"
	"fmt"
	"io"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// Assert checks all declarations before applying the requested root selector.
// Selectors are empty (all), package/name, or package/declaration/name.
func (r *Runtime) Assert(ctx context.Context, directory string, selector, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if r == nil {
		return fmt.Errorf("assertions require a bundled runtime")
	}
	store, err := BeginOutput(directory)
	if err != nil {
		return err
	}
	defer store.Close()
	program, err := check.CheckAssertionProgram(store.Graph)
	if err != nil {
		return err
	}
	if len(selector) != 0 {
		if len(selector) != 2 && len(selector) != 3 {
			return fmt.Errorf("assertion selector requires package/name or package/declaration/name")
		}
		var selected []*ir.Assertion
		for _, test := range program.Assertions {
			if test.Root.Package != selector[0] || test.Root.Name != selector[len(selector)-1] {
				continue
			}
			if len(selector) == 3 && test.Root.Declaration != selector[1] {
				continue
			}
			selected = append(selected, test)
		}
		if len(selected) != 1 {
			return fmt.Errorf("assertion selector resolves to %d roots; supply full package/declaration/name identity", len(selected))
		}
		program.Assertions = selected
	}
	buildID, _, err := r.stageProgram(ctx, store, program, true)
	if err != nil {
		return err
	}
	lease, err := store.AcquireGeneration(buildID)
	if err != nil {
		return err
	}
	defer lease.Close()
	if err = store.Close(); err != nil {
		return err
	}
	return r.RunOutput(ctx, lease, nil, environment, stdin, stdout, stderr)
}

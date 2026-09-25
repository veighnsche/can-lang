package driver

import (
	"context"
	"fmt"
	"io"
)

// Run builds from the current source snapshot, then retains a generation lease
// for the child while releasing the writer lock. Other builds and clean may
// proceed, but cannot prune the program that is still running.
func (r *Runtime) Run(ctx context.Context, projectDirectory string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if r == nil {
		return fmt.Errorf("run requires a bundled runtime")
	}
	store, err := BeginOutput(projectDirectory)
	if err != nil {
		return err
	}
	defer store.Close()
	// Run verifies like build under the default P15.1 budget; only build
	// and assert accept a configured timeout.
	if _, err = r.build(ctx, store, environment, stdin, stderr, DefaultAssertTimeoutMs, ""); err != nil {
		return err
	}
	lease, err := store.AcquireCurrent()
	if err != nil {
		return err
	}
	defer lease.Close()
	if err = store.Close(); err != nil {
		return err
	}
	return r.RunOutput(ctx, lease, args, environment, stdin, stdout, stderr)
}

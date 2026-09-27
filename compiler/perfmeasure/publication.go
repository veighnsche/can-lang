package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func sameArtifacts(actual, expected []ir.Artifact) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("artifact inventory differs")
	}
	for i, a := range actual {
		if a.Path != expected[i].Path || !bytes.Equal(a.Bytes, expected[i].Bytes) {
			return fmt.Errorf("artifact bytes/path differ: %s", a.Path)
		}
	}
	return nil
}

// New-publication probes use only the independent flat fixture. Copying inputs
// is setup, while the real owned-output store writes all production payloads.
func copyFlatInputs(from, to string, g *project.Graph) error {
	if len(g.Projects) != 1 {
		return fmt.Errorf("fresh-publication fixture requires one project")
	}
	names := []string{"can.project.json", "can.errors.json"}
	for _, p := range g.Projects {
		for _, s := range p.Sources {
			relative, e := filepath.Rel(from, s.Path)
			if e != nil {
				return e
			}
			names = append(names, relative)
		}
	}
	for _, name := range names {
		data, e := os.ReadFile(filepath.Join(from, name))
		if e != nil {
			return e
		}
		destination := filepath.Join(to, name)
		if e = os.MkdirAll(filepath.Dir(destination), 0700); e != nil {
			return e
		}
		if e = os.WriteFile(destination, data, 0600); e != nil {
			return e
		}
	}
	return nil
}

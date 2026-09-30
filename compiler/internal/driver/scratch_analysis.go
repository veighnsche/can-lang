package driver

import (
	"context"
	"errors"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"path/filepath"
)

// CheckScratchSnapshot analyzes a standalone source entirely in memory. Its
// package can refer to itself and the catalogue; no workspace imports or
// persisted error allocations are invented.
func CheckScratchSnapshot(ctx context.Context, path, text string) (*Snapshot, error) {
	root := filepath.Dir(path)
	graph := &project.Graph{Projects: map[string]*project.Project{}, Packages: map[string]*project.Package{}, Inputs: map[string][]byte{path: []byte(text)}}
	owner := &project.Project{ID: "can.scratch", Root: root, Registry: project.Registry{Active: []string{}, Retired: []string{}}, Dependencies: map[string]*project.Project{}}
	graph.Root = owner
	graph.Projects[""] = owner
	file, err := source.New(path, text)
	if err != nil {
		return analyzeGraph(ctx, root, graph, err, nil)
	}
	parsed := syntax.Parse(file)
	if !parsed.OK() {
		graph.Errors = append(graph.Errors, &project.SourceError{Path: path, File: file, Diagnostics: parsed.Diagnostics, Message: parsed.Diagnostics[0].Format(file)})
	}
	src := &project.Source{Name: filepath.Base(path), Path: path, RelativePath: filepath.Base(path), Bytes: []byte(text), Syntax: parsed.File}
	owner.Sources = []*project.Source{src}
	if parsed.File.Header.Name.Text != "" {
		pkg := &project.Package{Name: parsed.File.Header.Name.Text, ID: owner.ID + "/" + parsed.File.Header.Name.Text, Directory: root, Owner: owner, Sources: []*project.Source{src}}
		src.Package, src.ID = pkg, pkg.ID+"/"+src.Name
		owner.Packages = []*project.Package{pkg}
		graph.Packages[pkg.ID] = pkg
	}
	return analyzeGraph(ctx, root, graph, errors.Join(graph.Errors...), nil)
}

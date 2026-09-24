package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	resolution "github.com/veighnsche/can-lang/compiler/internal/resolve"
)

type projectReport struct {
	SchemaVersion     int                    `json:"schemaVersion"`
	Kind              string                 `json:"kind"`
	CatalogueRevision int                    `json:"catalogueRevision"`
	Projects          []projectOwnerReport   `json:"projects"`
	Packages          []projectPackageReport `json:"packages"`
}
type projectOwnerReport struct {
	ID             string `json:"id"`
	Lineage        string `json:"lineage"`
	ManifestSHA256 string `json:"manifestSHA256"`
	SourceSHA256   string `json:"sourceSHA256"`
}
type projectPackageReport struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	OutputDirectory string                `json:"outputDirectory"`
	Sources         []projectSourceReport `json:"sources"`
	Symbols         []projectSymbolReport `json:"symbols"`
}
type projectSourceReport struct {
	ID         string            `json:"id"`
	SourcePath string            `json:"sourcePath"`
	OutputPath string            `json:"outputPath"`
	Imports    map[string]string `json:"imports"`
}
type projectSymbolReport struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Kind   resolution.Kind `json:"kind"`
	Public bool            `json:"public"`
}

// The inspection command is an inert boundary for the new loader and resolver.
// It is not a compile/check/run alias: body typing and emission are later passes.
func runInspectProject(stdout, stderr io.Writer, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: canlc inspect-project PROJECT_DIRECTORY")
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
	report := projectReport{SchemaVersion: 2, Kind: "can.package-resolution", CatalogueRevision: catalogue.GeneratedRevision, Projects: []projectOwnerReport{}, Packages: []projectPackageReport{}}
	for _, owner := range graph.Projects {
		report.Projects = append(report.Projects, projectOwnerReport{ID: owner.ID, Lineage: owner.Lineage, ManifestSHA256: owner.ManifestSHA256, SourceSHA256: owner.SourceSHA256})
	}
	sort.Slice(report.Projects, func(i, j int) bool { return report.Projects[i].ID < report.Projects[j].ID })
	for _, pkg := range graph.Packages {
		item := projectPackageReport{ID: pkg.ID, Name: pkg.Name, OutputDirectory: pkg.OutputDirectory, Sources: []projectSourceReport{}, Symbols: []projectSymbolReport{}}
		for _, source := range pkg.Sources {
			imports := map[string]string{}
			for alias, target := range world.Files[source].Imports {
				imports[alias] = target.ID
			}
			item.Sources = append(item.Sources, projectSourceReport{ID: source.ID, SourcePath: source.RelativePath, OutputPath: source.OutputPath, Imports: imports})
		}
		sort.Slice(item.Sources, func(i, j int) bool { return item.Sources[i].ID < item.Sources[j].ID })
		for _, symbol := range world.Packages[pkg.ID].Scope.Symbols {
			item.Symbols = append(item.Symbols, projectSymbolReport{ID: symbol.ID, Name: symbol.Name, Kind: symbol.Kind, Public: symbol.Public})
		}
		sort.Slice(item.Symbols, func(i, j int) bool { return item.Symbols[i].ID < item.Symbols[j].ID })
		report.Packages = append(report.Packages, item)
	}
	sort.Slice(report.Packages, func(i, j int) bool { return report.Packages[i].ID < report.Packages[j].ID })
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

type mapSegment struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Source string `json:"source"`
	Name   string `json:"name"`
}
type mapModule struct {
	Path     string       `json:"path"`
	Segments []mapSegment `json:"segments"`
}

// encodeMappedArtifacts is a benchmark-only adapter to the production source-map
// tool protocol. The driver API does not expose encodeSourceMaps; assembly below
// uses the checked IR's real spans. The qualified production encoder creates the
// maps, and ValidateOutput verifies the resulting index/map/module relationship.
// Preparation excludes this adapter. The artifacts.source-maps case includes
// its assembly and the real tool invocation, with that scope named explicitly.
func encodeMappedArtifacts(r *driver.Runtime, p *check.Program, artifacts []ir.Artifact) ([]ir.Artifact, error) {
	sources := map[string]map[string]any{}
	files := map[string]*source.File{}
	for src := range p.World.Files {
		files[src.ID] = src.Syntax.Source
		sources[src.ID] = map[string]any{"id": src.ID, "path": src.RelativePath, "spans": map[string]any{}}
	}
	modules := []mapModule{}
	for _, a := range artifacts {
		if a.Runtime || !strings.HasSuffix(a.Path, ".ts") {
			continue
		}
		segments := []mapSegment{}
		for _, m := range a.Mappings {
			f := files[m.Source]
			if f == nil || m.Operation == "" {
				return nil, fmt.Errorf("invalid source mapping")
			}
			span := source.Span{Start: m.Start, End: m.End}
			if err := f.Validate(span); err != nil {
				return nil, err
			}
			start, _ := f.MapPosition(span.Start)
			end, _ := f.MapPosition(span.End)
			name := fmt.Sprintf("%s:%d:%d", m.Operation, m.Start, m.End)
			sources[m.Source]["spans"].(map[string]any)[name] = map[string]any{"start": m.Start, "end": m.End, "line": start.Line, "column": start.Column, "endLine": end.Line, "endColumn": end.Column, "operation": m.Operation}
			segments = append(segments, mapSegment{m.Line, m.Column, m.Source, name})
		}
		modules = append(modules, mapModule{a.Path, segments})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	ids := []string{}
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ordered := []map[string]any{}
	for _, id := range ids {
		ordered = append(ordered, sources[id])
	}
	request, err := json.Marshal(map[string]any{"schemaVersion": 1, "kind": "can.source-index", "sources": ordered, "modules": modules})
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	if err = r.RunTool(context.Background(), "tools/runtime/source-maps.ts", nil, nil, bytes.NewReader(request), &stdout, &stderr); err != nil {
		return nil, fmt.Errorf("source map encoder: %w: %s", err, stderr.String())
	}
	var response struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Kind          string                     `json:"kind"`
		RequestSHA256 string                     `json:"requestSHA256"`
		Maps          map[string]json.RawMessage `json:"maps"`
	}
	if err = json.Unmarshal(stdout.Bytes(), &response); err != nil || response.SchemaVersion != 1 || response.Kind != "can.source-maps" || len(response.Maps) != len(modules) || response.RequestSHA256 != fmt.Sprintf("%x", sha256.Sum256(request)) {
		return nil, fmt.Errorf("invalid map encoder response")
	}
	result := append([]ir.Artifact{}, artifacts...)
	for i := range result {
		a := &result[i]
		if a.Runtime || !strings.HasSuffix(a.Path, ".ts") {
			continue
		}
		a.Bytes = append(append([]byte{}, a.Bytes...), []byte("//# sourceMappingURL="+path.Base(a.Path)+".map\n")...)
	}
	for _, m := range modules {
		name := m.Path
		result = append(result, ir.Artifact{Path: name + ".map", Bytes: response.Maps[name]})
	}
	result = append(result, ir.Artifact{Path: "diagnostics/source-index.json", Bytes: request})
	return result, nil
}

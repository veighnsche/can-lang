package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

type diagnosticSpan struct {
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Operation string `json:"operation"`
}
type diagnosticSource struct {
	ID    string                    `json:"id"`
	Path  string                    `json:"path"`
	Spans map[string]diagnosticSpan `json:"spans"`
}
type diagnosticSegment struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Source string `json:"source"`
	Name   string `json:"name"`
}
type diagnosticModule struct {
	Path     string              `json:"path"`
	Segments []diagnosticSegment `json:"segments"`
}
type sourceIndex struct {
	SchemaVersion int                `json:"schemaVersion"`
	Kind          string             `json:"kind"`
	Sources       []diagnosticSource `json:"sources"`
	Modules       []diagnosticModule `json:"modules"`
}

func (r *Runtime) encodeSourceMaps(ctx context.Context, program *check.Program, artifacts []ir.Artifact) ([]ir.Artifact, error) {
	index := sourceIndex{1, "can.source-index", []diagnosticSource{}, []diagnosticModule{}}
	sourceFiles := map[string]*source.File{}
	entries := map[string]*diagnosticSource{}
	for src := range program.World.Files {
		sourceFiles[src.ID] = src.Syntax.Source
		entries[src.ID] = &diagnosticSource{src.ID, src.RelativePath, map[string]diagnosticSpan{}}
	}
	for _, artifact := range artifacts {
		if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		module := diagnosticModule{artifact.Path, []diagnosticSegment{}}
		for _, segment := range artifact.Mappings {
			file := sourceFiles[segment.Source]
			if file == nil || segment.Operation == "" {
				return nil, fmt.Errorf("invalid mapping source or operation")
			}
			span := source.Span{Start: segment.Start, End: segment.End}
			if err := file.Validate(span); err != nil {
				return nil, err
			}
			start, _ := file.MapPosition(span.Start)
			end, _ := file.MapPosition(span.End)
			name := fmt.Sprintf("%s:%d:%d", segment.Operation, span.Start, span.End)
			entries[segment.Source].Spans[name] = diagnosticSpan{span.Start, span.End, start.Line, start.Column, end.Line, end.Column, segment.Operation}
			module.Segments = append(module.Segments, diagnosticSegment{segment.Line, segment.Column, segment.Source, name})
		}
		index.Modules = append(index.Modules, module)
	}
	for _, id := range sortedOutputKeys(entries) {
		index.Sources = append(index.Sources, *entries[id])
	}
	sort.Slice(index.Modules, func(i, j int) bool { return index.Modules[i].Path < index.Modules[j].Path })
	request, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	if err = r.RunTool(ctx, "tools/runtime/source-maps.ts", nil, nil, bytes.NewReader(request), &stdout, &stderr); err != nil {
		return nil, fmt.Errorf("source map encoding failed: %w", err)
	}
	var response struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Kind          string                     `json:"kind"`
		RequestSHA256 string                     `json:"requestSHA256"`
		Maps          map[string]json.RawMessage `json:"maps"`
	}
	if err = decodeOutput(stdout.Bytes(), &response); err != nil || response.SchemaVersion != 1 || response.Kind != "can.source-maps" || response.RequestSHA256 != hashBytes(request) || len(response.Maps) != len(index.Modules) {
		return nil, fmt.Errorf("invalid source map helper report")
	}
	result := append([]ir.Artifact{}, artifacts...)
	for i := range result {
		artifact := &result[i]
		if artifact.Runtime || !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		data := response.Maps[artifact.Path]
		if len(data) == 0 {
			return nil, fmt.Errorf("missing encoded source map")
		}
		artifact.Bytes = append(append([]byte{}, artifact.Bytes...), []byte("//# sourceMappingURL="+path.Base(artifact.Path)+".map\n")...)
	}
	for _, module := range index.Modules {
		result = append(result, ir.Artifact{Path: module.Path + ".map", Bytes: response.Maps[module.Path]})
	}
	result = append(result, ir.Artifact{Path: "diagnostics/source-index.json", Bytes: request})
	return result, nil
}

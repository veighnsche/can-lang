package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// maps validates the sealed source-map set carried by a browser generation:
// the per-module .ts.map files, the sourceMappingURL trailers, and the
// diagnostics/source-index.json table. Validation is structural and
// two-sided: map bytes must decode as source-map v3, index entries must be
// well-formed, and the decoded segments must agree with the index exactly,
// so a tampered map or a stale index fails before publication.
func (audit *graphAudit) sealedMaps() error {
	_, hasIndex := audit.byPath[SourceIndexPath]
	if len(audit.maps) == 0 && !hasIndex {
		// Pre-map stage: unit-test generations seal bytes before maps
		// exist. The post-bundle audit enforces final map presence.
		return nil
	}
	rawIndex, ok := audit.byPath[SourceIndexPath]
	if !ok {
		return fmt.Errorf("browser audit: sealed maps without %s", SourceIndexPath)
	}
	index, err := parseSourceIndex(rawIndex.Bytes)
	if err != nil {
		return fmt.Errorf("browser audit: %s: %w", SourceIndexPath, err)
	}
	if err := scanCanaries(SourceIndexPath, rawIndex.Bytes, nil); err != nil {
		return err
	}
	indexed := map[string]sealedModule{}
	for _, module := range index.Modules {
		if _, dup := indexed[module.Path]; dup {
			return fmt.Errorf("browser audit: %s lists %s twice", SourceIndexPath, module.Path)
		}
		indexed[module.Path] = module
	}
	if len(indexed) != len(audit.generated) {
		return fmt.Errorf("browser audit: %s covers %d modules, generation holds %d", SourceIndexPath, len(indexed), len(audit.generated))
	}
	for _, name := range audit.generated {
		module, ok := indexed[name]
		if !ok {
			return fmt.Errorf("browser audit: %s omits generated module %s", SourceIndexPath, name)
		}
		if err := audit.sealedModule(name, module, index); err != nil {
			return err
		}
	}
	for name := range audit.maps {
		module := strings.TrimSuffix(name, ".map")
		if _, ok := indexed[module]; !ok {
			return fmt.Errorf("browser audit: orphan source map %s with no generated module", name)
		}
	}
	return nil
}

type sealedSpan struct {
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Operation string `json:"operation"`
}

type sealedSource struct {
	ID    string                `json:"id"`
	Path  string                `json:"path"`
	Spans map[string]sealedSpan `json:"spans"`
}

type sealedSegment struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Source string `json:"source"`
	Name   string `json:"name"`
}

type sealedModule struct {
	Path     string          `json:"path"`
	Segments []sealedSegment `json:"segments"`
}

type sealedIndex struct {
	SchemaVersion int            `json:"schemaVersion"`
	Kind          string         `json:"kind"`
	Sources       []sealedSource `json:"sources"`
	Modules       []sealedModule `json:"modules"`
}

func parseSourceIndex(raw []byte) (sealedIndex, error) {
	var index sealedIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return sealedIndex{}, fmt.Errorf("invalid source index: %w", err)
	}
	if index.SchemaVersion != 1 || index.Kind != "can.source-index" {
		return sealedIndex{}, fmt.Errorf("invalid source index identity")
	}
	seen := map[string]bool{}
	for _, source := range index.Sources {
		if source.ID == "" || source.Path == "" {
			return sealedIndex{}, fmt.Errorf("source index carries an empty source identity")
		}
		if seen[source.ID] {
			return sealedIndex{}, fmt.Errorf("source index lists source %s twice", source.ID)
		}
		seen[source.ID] = true
		for name, span := range source.Spans {
			if name == "" || span.Operation == "" {
				return sealedIndex{}, fmt.Errorf("source index carries an empty span or operation")
			}
			if span.Start < 0 || span.End < span.Start || span.Line < 1 || span.Column < 0 || span.EndLine < span.Line {
				return sealedIndex{}, fmt.Errorf("source index span %q of %s is malformed", name, source.ID)
			}
			if span.EndLine == span.Line && span.EndColumn < span.Column {
				return sealedIndex{}, fmt.Errorf("source index span %q of %s is malformed", name, source.ID)
			}
		}
	}
	return index, nil
}

func (audit *graphAudit) sealedModule(name string, module sealedModule, index sealedIndex) error {
	artifact := audit.byPath[name]
	lines := splitLines(artifact.Bytes)
	spans := map[string]map[string]sealedSpan{}
	for _, source := range index.Sources {
		spans[source.ID] = source.Spans
	}
	for _, segment := range module.Segments {
		table, ok := spans[segment.Source]
		if !ok {
			return fmt.Errorf("browser audit: %s segment names unknown source %s", name, segment.Source)
		}
		if _, ok := table[segment.Name]; !ok {
			return fmt.Errorf("browser audit: %s segment names unknown span %q of %s", name, segment.Name, segment.Source)
		}
		if err := checkBounds(name, lines, segment.Line, segment.Column); err != nil {
			return err
		}
	}
	mapName := name + ".map"
	raw, ok := audit.byPath[mapName]
	if !ok {
		return fmt.Errorf("browser audit: generated module %s lacks its sealed source map %s", name, mapName)
	}
	trailer := "//# sourceMappingURL=" + path.Base(name) + ".map\n"
	if !strings.HasSuffix(string(artifact.Bytes), trailer) {
		return fmt.Errorf("browser audit: %s lacks its %q trailer", name, strings.TrimSuffix(trailer, "\n"))
	}
	if err := scanCanaries(mapName, raw.Bytes, nil); err != nil {
		return err
	}
	decoded, err := parseSourceMap(raw.Bytes)
	if err != nil {
		return fmt.Errorf("browser audit: %s: %w", mapName, err)
	}
	if decoded.File != path.Base(name) {
		return fmt.Errorf("browser audit: %s names file %q, want %q", mapName, decoded.File, path.Base(name))
	}
	segments, err := decodeMappings(decoded.Mappings)
	if err != nil {
		return fmt.Errorf("browser audit: %s: %w", mapName, err)
	}
	return checkMapAgreement(mapName, name, lines, decoded, segments, module, spans)
}

func checkMapAgreement(mapName, module string, lines []string, decoded decodedSourceMap, segments [][]mapSegment, indexed sealedModule, spans map[string]map[string]sealedSpan) error {
	for _, source := range decoded.Sources {
		if _, ok := spans[source]; !ok {
			return fmt.Errorf("browser audit: %s names source %s outside the source index", mapName, source)
		}
	}
	if len(decoded.Sources) == 0 {
		for _, line := range segments {
			if len(line) != 0 {
				return fmt.Errorf("browser audit: %s carries segments with no sources", mapName)
			}
		}
	}
	type key struct {
		line, column int
		source       string
		name         string
		srcLine      int
		srcColumn    int
	}
	want := map[key]int{}
	for _, segment := range indexed.Segments {
		span := spans[segment.Source][segment.Name]
		want[key{segment.Line, segment.Column, segment.Source, segment.Name, span.Line, span.Column}]++
	}
	got := map[key]int{}
	for lineIndex, line := range segments {
		for _, segment := range line {
			if err := checkBounds(module, lines, lineIndex+1, segment.GenColumn); err != nil {
				return fmt.Errorf("browser audit: %s: %w", mapName, err)
			}
			if segment.Source < 0 {
				continue
			}
			if segment.Source >= len(decoded.Sources) {
				return fmt.Errorf("browser audit: %s segment cites source %d of %d", mapName, segment.Source, len(decoded.Sources))
			}
			source := decoded.Sources[segment.Source]
			name := ""
			if segment.Name >= 0 {
				if segment.Name >= len(decoded.Names) {
					return fmt.Errorf("browser audit: %s segment cites name %d of %d", mapName, segment.Name, len(decoded.Names))
				}
				name = decoded.Names[segment.Name]
			}
			got[key{lineIndex + 1, segment.GenColumn, source, name, segment.SrcLine + 1, segment.SrcColumn}]++
		}
	}
	if len(got) != len(want) {
		return fmt.Errorf("browser audit: %s disagrees with the source index on %s: %d mapped positions against %d indexed", mapName, module, len(got), len(want))
	}
	for key, count := range want {
		if got[key] != count {
			return fmt.Errorf("browser audit: %s disagrees with the source index on %s at %d:%d", mapName, module, key.line, key.column)
		}
	}
	return nil
}

// checkBounds requires a generated coordinate to land inside the module.
// Columns are UTF-16 units while lines are split as bytes; every byte
// contributes at most one column unit, so the byte length is a sound
// (slightly loose) upper bound that can never false-fail.
func checkBounds(name string, lines []string, line, column int) error {
	if line < 1 || line > len(lines) {
		return fmt.Errorf("browser audit: %s coordinate %d:%d lands outside the module", name, line, column)
	}
	if column < 0 || column > len(lines[line-1]) {
		return fmt.Errorf("browser audit: %s coordinate %d:%d lands outside the module", name, line, column)
	}
	return nil
}

func splitLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
}

// bindAsset requires the asset manifest to bind exactly the generated
// module digests: no module missing, none extra, none tampered.
func (audit *graphAudit) bindAsset(asset ir.Artifact) error {
	files := map[string]string{}
	for _, name := range audit.generated {
		sum := sha256.Sum256(audit.byPath[name].Bytes)
		files[name] = hex.EncodeToString(sum[:])
	}
	manifest, err := ParseAsset(asset.Bytes)
	if err != nil {
		return fmt.Errorf("browser audit: %w", err)
	}
	if len(manifest.Files) != len(files) {
		return fmt.Errorf("browser audit: asset binds %d modules, generation holds %d", len(manifest.Files), len(files))
	}
	for name, digest := range files {
		if manifest.Files[name] != digest {
			return fmt.Errorf("browser audit: asset digest mismatch for %s", name)
		}
	}
	return nil
}

// mapSegment is one decoded source-map segment: a generated column with an
// optional original source position and name. Source, SrcLine, SrcColumn
// and Name hold -1 when the segment carries no original position.
type mapSegment struct {
	GenColumn int
	Source    int
	SrcLine   int
	SrcColumn int
	Name      int
}

type decodedSourceMap struct {
	File           string
	Sources        []string
	SourcesContent []*string
	Names          []string
	Mappings       string
}

func parseSourceMap(raw []byte) (decodedSourceMap, error) {
	var encoded struct {
		Version        int       `json:"version"`
		File           string    `json:"file"`
		Sources        []string  `json:"sources"`
		SourcesContent []*string `json:"sourcesContent"`
		Names          []string  `json:"names"`
		Mappings       string    `json:"mappings"`
	}
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return decodedSourceMap{}, fmt.Errorf("invalid source map: %w", err)
	}
	if encoded.Version != 3 {
		return decodedSourceMap{}, fmt.Errorf("unsupported source map version %d", encoded.Version)
	}
	if encoded.Sources == nil || encoded.Names == nil {
		return decodedSourceMap{}, fmt.Errorf("source map lacks sources or names")
	}
	if encoded.SourcesContent != nil && len(encoded.SourcesContent) != len(encoded.Sources) {
		return decodedSourceMap{}, fmt.Errorf("source map sourcesContent disagrees with sources")
	}
	return decodedSourceMap{
		File:           encoded.File,
		Sources:        encoded.Sources,
		SourcesContent: encoded.SourcesContent,
		Names:          encoded.Names,
		Mappings:       encoded.Mappings,
	}, nil
}

const vlqAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var vlqValues = func() map[byte]int {
	values := map[byte]int{}
	for index := 0; index < len(vlqAlphabet); index++ {
		values[vlqAlphabet[index]] = index
	}
	return values
}()

// decodeMappings decodes a source-map mappings string into per-line
// segments. Generated lines are 1-based by position; columns, source
// lines and source columns follow the source-map base (0-based).
func decodeMappings(mappings string) ([][]mapSegment, error) {
	lines := strings.Split(mappings, ";")
	decoded := make([][]mapSegment, 0, len(lines))
	prevSource, prevSrcLine, prevSrcColumn, prevName := 0, 0, 0, 0
	for _, line := range lines {
		var segments []mapSegment
		if line != "" {
			parts := strings.Split(line, ",")
			genColumn := 0
			for _, part := range parts {
				if part == "" {
					return nil, fmt.Errorf("malformed mappings segment")
				}
				fields, err := decodeSegment(part)
				if err != nil {
					return nil, err
				}
				if len(fields) != 1 && len(fields) != 4 && len(fields) != 5 {
					return nil, fmt.Errorf("malformed mappings segment")
				}
				genColumn += fields[0]
				segment := mapSegment{GenColumn: genColumn, Source: -1, SrcLine: -1, SrcColumn: -1, Name: -1}
				if len(fields) >= 4 {
					prevSource += fields[1]
					prevSrcLine += fields[2]
					prevSrcColumn += fields[3]
					segment.Source, segment.SrcLine, segment.SrcColumn = prevSource, prevSrcLine, prevSrcColumn
				}
				if len(fields) == 5 {
					prevName += fields[4]
					segment.Name = prevName
				}
				if segment.GenColumn < 0 || segment.Source < -1 || segment.SrcLine < -1 || segment.SrcColumn < -1 || segment.Name < -1 {
					return nil, fmt.Errorf("malformed mappings segment")
				}
				segments = append(segments, segment)
			}
		}
		decoded = append(decoded, segments)
	}
	return decoded, nil
}

func decodeSegment(part string) ([]int, error) {
	var fields []int
	offset := 0
	for offset < len(part) {
		value, width, err := decodeVLQ(part[offset:])
		if err != nil {
			return nil, err
		}
		offset += width
		fields = append(fields, value)
	}
	return fields, nil
}

func decodeVLQ(part string) (int, int, error) {
	value, shift, width := 0, 0, 0
	for _, char := range []byte(part) {
		digit, ok := vlqValues[char]
		if !ok {
			return 0, 0, fmt.Errorf("malformed mappings encoding")
		}
		width++
		value |= (digit & 31) << shift
		shift += 5
		if digit&32 == 0 {
			if value&1 == 0 {
				return value >> 1, width, nil
			}
			return -(value >> 1), width, nil
		}
	}
	return 0, 0, fmt.Errorf("malformed mappings encoding")
}

// mapOrigin resolves a generated coordinate through decoded segments to
// its original source position, or "" when the position is unmapped.
// Comparison is by nearest segment at or before the column on the line.
func mapOrigin(sources []string, segments [][]mapSegment, line, column int) string {
	if line < 1 || line > len(segments) {
		return ""
	}
	best := -1
	for index, segment := range segments[line-1] {
		if segment.GenColumn <= column && segment.Source >= 0 {
			if best < 0 || segment.GenColumn > segments[line-1][best].GenColumn {
				best = index
			}
		}
	}
	if best < 0 {
		return ""
	}
	segment := segments[line-1][best]
	if segment.Source >= len(sources) {
		return ""
	}
	return fmt.Sprintf("; originating %s [%d:%d]", sources[segment.Source], segment.SrcLine+1, segment.SrcColumn)
}

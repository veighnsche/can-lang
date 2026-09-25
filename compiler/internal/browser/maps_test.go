package browser

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func TestDecodeMappingsVectors(t *testing.T) {
	segments, err := decodeMappings("A")
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 1 || len(segments[0]) != 1 || segments[0][0].GenColumn != 0 || segments[0][0].Source != -1 {
		t.Fatalf("unmapped segment = %+v", segments)
	}
	segments, err = decodeMappings("AAAA")
	if err != nil {
		t.Fatal(err)
	}
	got := segments[0][0]
	if got.GenColumn != 0 || got.Source != 0 || got.SrcLine != 0 || got.SrcColumn != 0 || got.Name != -1 {
		t.Fatalf("mapped segment = %+v", got)
	}
	segments, err = decodeMappings("AACIA")
	if err != nil {
		t.Fatal(err)
	}
	got = segments[0][0]
	if got.GenColumn != 0 || got.Source != 0 || got.SrcLine != 1 || got.SrcColumn != 4 || got.Name != 0 {
		t.Fatalf("named segment = %+v", got)
	}
	segments, err = decodeMappings("A,C")
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 1 || len(segments[0]) != 2 || segments[0][1].GenColumn != 1 {
		t.Fatalf("relative columns = %+v", segments)
	}
	segments, err = decodeMappings(";AACIA;")
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 3 || len(segments[0]) != 0 || len(segments[1]) != 1 || len(segments[2]) != 0 {
		t.Fatalf("multiline segments = %+v", segments)
	}
	segments, err = decodeMappings("")
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 1 || len(segments[0]) != 0 {
		t.Fatalf("empty mappings = %+v", segments)
	}
	for _, bad := range []string{"A,", ",", "!", "g", "AA", "AAAAAA"} {
		if _, err := decodeMappings(bad); err == nil {
			t.Fatalf("malformed mappings admitted: %q", bad)
		}
	}
}

func TestMapOriginResolves(t *testing.T) {
	segments, err := decodeMappings("AACIA")
	if err != nil {
		t.Fatal(err)
	}
	if got := mapOrigin([]string{"app/main.can"}, segments, 1, 9); got != "; originating app/main.can [2:4]" {
		t.Fatalf("origin = %q", got)
	}
	if got := mapOrigin([]string{"app/main.can"}, segments, 2, 0); got != "" {
		t.Fatalf("unmapped line origin = %q", got)
	}
	unmapped, err := decodeMappings("A")
	if err != nil {
		t.Fatal(err)
	}
	if got := mapOrigin([]string{"app/main.can"}, unmapped, 1, 0); got != "" {
		t.Fatalf("unmapped segment origin = %q", got)
	}
}

func TestParseSourceMapValidation(t *testing.T) {
	good := `{"version":3,"file":"app.js","sources":["a.can"],"sourcesContent":[null],"names":[],"mappings":"AAAA"}`
	if _, err := parseSourceMap([]byte(good)); err != nil {
		t.Fatalf("valid map rejected: %v", err)
	}
	for name, raw := range map[string]string{
		"not json":             `{`,
		"bad version":          `{"version":2,"file":"app.js","sources":[],"names":[],"mappings":""}`,
		"missing sources":      `{"version":3,"file":"app.js","names":[],"mappings":""}`,
		"missing names":        `{"version":3,"file":"app.js","sources":[],"mappings":""}`,
		"content disagreement": `{"version":3,"file":"app.js","sources":["a.can"],"sourcesContent":[],"names":[],"mappings":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSourceMap([]byte(raw)); err == nil {
				t.Fatalf("invalid map admitted: %s", raw)
			}
		})
	}
}

// sealedMapFixture is a complete pre-bundle sealed-map set: two generated
// modules with trailers, one mapped and one empty module map, and the
// source index binding both.
type sealedMapFixture struct {
	entryBody string
	stateBody string
	entryMap  string
	stateMap  string
	index     string
}

func baseSealedFixture() sealedMapFixture {
	return sealedMapFixture{
		entryBody: "import \"./program/state.ts\";\nexport const x = 1;\n//# sourceMappingURL=browser.ts.map\n",
		stateBody: "export function $canInitialize(): void {}\n//# sourceMappingURL=state.ts.map\n",
		entryMap:  `{"version":3,"file":"browser.ts","sources":["can.project.root/app/main.can"],"names":["call:10:20"],"mappings":";AACIA"}`,
		stateMap:  `{"version":3,"file":"state.ts","sources":[],"names":[],"mappings":""}`,
		index: `{"schemaVersion":1,"kind":"can.source-index",` +
			`"sources":[{"id":"can.project.root/app/main.can","path":"app/main.can",` +
			`"spans":{"call:10:20":{"start":10,"end":20,"line":2,"column":4,"endLine":2,"endColumn":14,"operation":"call"}}}],` +
			`"modules":[{"path":"browser.ts","segments":[{"line":2,"column":0,"source":"can.project.root/app/main.can","name":"call:10:20"}]},` +
			`{"path":"program/state.ts","segments":[]}]}`,
	}
}

func (fixture sealedMapFixture) seal(t *testing.T, drop ...string) []ir.Artifact {
	t.Helper()
	dropped := map[string]bool{}
	for _, name := range drop {
		dropped[name] = true
	}
	modules := []graphModule{
		{path: BrowserEntry, body: fixture.entryBody, imports: []string{"./program/state.ts"}},
		{path: "program/state.ts", body: fixture.stateBody},
	}
	var extra []ir.Artifact
	add := func(path, body string) {
		if !dropped[path] {
			extra = append(extra, ir.Artifact{Path: path, Bytes: []byte(body)})
		}
	}
	add(BrowserEntry+".map", fixture.entryMap)
	add("program/state.ts.map", fixture.stateMap)
	add(SourceIndexPath, fixture.index)
	return sealGraph(t, modules, extra...)
}

func TestSealedMapsAcceptCompleteSet(t *testing.T) {
	if err := AuditArtifacts(baseSealedFixture().seal(t)); err != nil {
		t.Fatalf("complete sealed set rejected: %v", err)
	}
}

func TestSealedMapsRejectDefects(t *testing.T) {
	base := baseSealedFixture()
	emptyModules := `{"schemaVersion":1,"kind":"can.source-index",` +
		`"sources":[{"id":"can.project.root/app/main.can","path":"app/main.can","spans":{}}],` +
		`"modules":[{"path":"browser.ts","segments":[]},{"path":"program/state.ts","segments":[]}]}`
	cases := map[string]struct {
		mutate func(*sealedMapFixture) []string
		want   []string
	}{
		"missing module map": {
			mutate: func(fixture *sealedMapFixture) []string { return []string{"program/state.ts.map"} },
			want:   []string{"lacks its sealed source map", "program/state.ts.map"},
		},
		"missing index": {
			mutate: func(fixture *sealedMapFixture) []string { return []string{SourceIndexPath} },
			want:   []string{"sealed maps without", SourceIndexPath},
		},
		"missing trailer": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryBody = strings.TrimSuffix(fixture.entryBody, "//# sourceMappingURL=browser.ts.map\n")
				return nil
			},
			want: []string{"lacks its", "sourceMappingURL=browser.ts.map"},
		},
		"tampered mappings": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = strings.Replace(fixture.entryMap, `";AACIA"`, `"AAAA"`, 1)
				return nil
			},
			want: []string{"disagrees with the source index"},
		},
		"segment outside module": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.index = strings.Replace(fixture.index, `{"line":2,`, `{"line":99,`, 1)
				return nil
			},
			want: []string{"outside the module"},
		},
		"unknown segment source": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.index = strings.Replace(fixture.index, `"source":"can.project.root/app/main.can","name"`, `"source":"can.project.root/gone.can","name"`, 1)
				return nil
			},
			want: []string{"unknown source"},
		},
		"unknown span name": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.index = strings.Replace(fixture.index, `"name":"call:10:20"`, `"name":"call:1:2"`, 1)
				return nil
			},
			want: []string{"unknown span"},
		},
		"map names wrong file": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = strings.Replace(fixture.entryMap, `"file":"browser.ts"`, `"file":"other.ts"`, 1)
				return nil
			},
			want: []string{"names file"},
		},
		"map bad version": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = strings.Replace(fixture.entryMap, `"version":3`, `"version":2`, 1)
				return nil
			},
			want: []string{"unsupported source map version"},
		},
		"map malformed mappings": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = strings.Replace(fixture.entryMap, `";AACIA"`, `"A,"`, 1)
				return nil
			},
			want: []string{"malformed mappings"},
		},
		"map source outside index": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = strings.Replace(fixture.entryMap, `"sources":["can.project.root/app/main.can"]`, `"sources":["can.project.root/app/main.can","can.project.root/extra.can"]`, 1)
				return nil
			},
			want: []string{"outside the source index"},
		},
		"index omits module": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.index = strings.Replace(fixture.index, `,{"path":"program/state.ts","segments":[]}`, ``, 1)
				return nil
			},
			want: []string{"covers 1 modules, generation holds 2"},
		},
		"canary in map": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.stateMap = `{"version":3,"file":"state.ts","sources":[],"names":[],"mappings":"","x":"__CAN_SECRET_CANARY__"}`
				return nil
			},
			want: []string{"secret canary"},
		},
		"canary in index": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.index = strings.Replace(fixture.index, `"operation":"call"`, `"operation":"call CANARY-SECRET-DO-NOT-SHIP"`, 1)
				return nil
			},
			want: []string{"secret canary"},
		},
		"empty map agreement": {
			mutate: func(fixture *sealedMapFixture) []string {
				fixture.entryMap = `{"version":3,"file":"browser.ts","sources":[],"names":[],"mappings":""}`
				fixture.index = emptyModules
				return nil
			},
			want: nil,
		},
	}
	for name, fixture := range cases {
		t.Run(name, func(t *testing.T) {
			edited := base
			drop := fixture.mutate(&edited)
			artifacts := edited.seal(t, drop...)
			if fixture.want == nil {
				if err := AuditArtifacts(artifacts); err != nil {
					t.Fatalf("empty sealed set rejected: %v", err)
				}
				return
			}
			auditFails(t, artifacts, fixture.want...)
		})
	}
}

func TestSealedMapsRejectOrphanMap(t *testing.T) {
	artifacts := baseSealedFixture().seal(t)
	artifacts = append(artifacts, ir.Artifact{Path: "packages/p-9/s-9.ts.map", Bytes: []byte(`{"version":3,"file":"s-9.ts","sources":[],"names":[],"mappings":""}`)})
	auditFails(t, artifacts, "orphan source map")
}

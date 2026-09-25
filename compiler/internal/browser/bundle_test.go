package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

type bundleFixture struct {
	entry string
	table string
	files map[string]string
}

func baseBundleFixture() bundleFixture {
	return bundleFixture{
		entry: "app.js",
		table: "diagnostics/table.json",
		files: map[string]string{
			"app.js":                 "\"use strict\";\nexport const answer = 41 + 1;\n//# sourceMappingURL=app.js.map\n",
			"app.js.map":             `{"version":3,"file":"app.js","sources":["../src/main.can"],"names":[],"mappings":"AAAA"}`,
			"diagnostics/table.json": `{"schemaVersion":1,"kind":"can.diagnostic-table","records":[{"category":"startup","phase":"boot","file":"app/main.can","line":2,"column":4}]}`,
		},
	}
}

func (fixture bundleFixture) bundle(t *testing.T, secrets ...string) Bundle {
	t.Helper()
	assembled := Bundle{Entry: fixture.entry, Table: fixture.table, Digests: map[string]string{}, Secrets: secrets}
	for name, body := range fixture.files {
		assembled.Files = append(assembled.Files, BundleFile{Path: name, Bytes: []byte(body)})
		sum := sha256.Sum256([]byte(body))
		assembled.Digests[name] = hex.EncodeToString(sum[:])
	}
	return assembled
}

func bundleFails(t *testing.T, bundle Bundle, wants ...string) {
	t.Helper()
	err := AuditBundle(bundle)
	if err == nil {
		t.Fatal("bundle violation admitted")
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "s3cr3t-build-matter") {
		t.Fatalf("diagnostic leaks build secret: %q", err.Error())
	}
}

func TestAuditBundleAcceptsCleanBundle(t *testing.T) {
	if err := AuditBundle(baseBundleFixture().bundle(t)); err != nil {
		t.Fatalf("clean bundle rejected: %v", err)
	}
}

func TestAuditBundleAcceptsAccountedChunks(t *testing.T) {
	fixture := baseBundleFixture()
	fixture.files["app.js"] = "import \"./chunk.js\";\nexport const answer = 41 + 1;\n//# sourceMappingURL=app.js.map\n"
	fixture.files["chunk.js"] = "export const chunk = 7;\n//# sourceMappingURL=chunk.js.map\n"
	fixture.files["chunk.js.map"] = `{"version":3,"file":"chunk.js","sources":["../src/chunk.can"],"names":[],"mappings":"AAAA"}`
	if err := AuditBundle(fixture.bundle(t)); err != nil {
		t.Fatalf("chunked bundle rejected: %v", err)
	}
}

func TestAuditBundleRejectsHostOperations(t *testing.T) {
	base := baseBundleFixture()
	cases := map[string]struct {
		line string
		want []string
	}{
		"bun call": {
			line: "export const out = Bun.write(bundle, line);\n",
			want: []string{"app.js:1:", "host operation Bun.write", "originating ../src/main.can [1:0]"},
		},
		"process member": {
			line: "export const home = process.env.HOME;\n",
			want: []string{"host operation process.env"},
		},
		"dynamic import": {
			line: "export const lazy = await import(\"./chunk.js\");\n",
			want: []string{"unaccounted edge import()"},
		},
		"require call": {
			line: "export const fs = require(\"node:fs\");\n",
			want: []string{"unaccounted edge require"},
		},
		"eval call": {
			line: "export const v = eval(\"1+1\");\n",
			want: []string{"host operation eval"},
		},
		"function ctor": {
			line: "export const f = new Function(\"return 1\");\n",
			want: []string{"host operation Function"},
		},
	}
	for name, fixture := range cases {
		t.Run(name, func(t *testing.T) {
			edited := base
			edited.files = map[string]string{}
			for path, body := range base.files {
				edited.files[path] = body
			}
			edited.files["app.js"] = fixture.line + "//# sourceMappingURL=app.js.map\n"
			bundleFails(t, edited.bundle(t), fixture.want...)
		})
	}
}

func TestAuditBundleAdmitsQuotedData(t *testing.T) {
	fixture := baseBundleFixture()
	fixture.files["app.js"] = "export const a = \"Bun.\";\nexport const b = 'require(';\n" +
		"export const c = `node: introduction`;\n// Bun.write(x)\n//# sourceMappingURL=app.js.map\n"
	if err := AuditBundle(fixture.bundle(t)); err != nil {
		t.Fatalf("quoted bundle data rejected: %v", err)
	}
}

func TestAuditBundleRejectsEdgeViolations(t *testing.T) {
	base := baseBundleFixture()
	withLine := func(line string) bundleFixture {
		edited := base
		edited.files = map[string]string{}
		for path, body := range base.files {
			edited.files[path] = body
		}
		edited.files["app.js"] = line + "//# sourceMappingURL=app.js.map\n"
		return edited
	}
	t.Run("missing chunk", func(t *testing.T) {
		bundleFails(t, withLine("import \"./gone.js\";\n").bundle(t), "unaccounted edge", "no published chunk")
	})
	t.Run("native edge", func(t *testing.T) {
		bundleFails(t, withLine("import \"node:fs\";\n").bundle(t), "unaccounted edge", "node:fs")
	})
	t.Run("absolute edge", func(t *testing.T) {
		bundleFails(t, withLine("import \"/etc/app.js\";\n").bundle(t), "unaccounted edge")
	})
	t.Run("remote edge", func(t *testing.T) {
		bundleFails(t, withLine("import \"https://example.invalid/app.js\";\n").bundle(t), "unaccounted edge")
	})
	t.Run("non chunk target", func(t *testing.T) {
		edited := withLine("import \"./diagnostics/table.json\";\n")
		bundleFails(t, edited.bundle(t), "unaccounted edge")
	})
}

func TestAuditBundleRejectsManifestDefects(t *testing.T) {
	t.Run("digest mismatch", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		for i := range assembled.Files {
			if assembled.Files[i].Path == "app.js" {
				assembled.Files[i].Bytes = []byte("export const answer = 42;\n//# sourceMappingURL=app.js.map\n")
			}
		}
		bundleFails(t, assembled, "digest mismatch", "app.js")
	})
	t.Run("missing digest", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		delete(assembled.Digests, "app.js.map")
		bundleFails(t, assembled, "no manifest digest", "app.js.map")
	})
	t.Run("extra digest", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		assembled.Digests["ghost.js"] = strings.Repeat("0", 64)
		bundleFails(t, assembled, "unpublished file", "ghost.js")
	})
	t.Run("malformed digest", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		assembled.Digests["app.js"] = "zz"
		bundleFails(t, assembled, "not SHA-256 hex")
	})
	t.Run("unaccounted file", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["notes.txt"] = "stray\n"
		bundleFails(t, fixture.bundle(t), "unaccounted published file", "notes.txt")
	})
	t.Run("missing map", func(t *testing.T) {
		fixture := baseBundleFixture()
		delete(fixture.files, "app.js.map")
		bundleFails(t, fixture.bundle(t), "lacks its source map")
	})
	t.Run("missing table", func(t *testing.T) {
		fixture := baseBundleFixture()
		delete(fixture.files, "diagnostics/table.json")
		bundleFails(t, fixture.bundle(t), "diagnostic table", "not published")
	})
	t.Run("orphan map", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["ghost.js.map"] = `{"version":3,"file":"ghost.js","sources":["x"],"names":[],"mappings":""}`
		bundleFails(t, fixture.bundle(t), "orphan source map")
	})
	t.Run("missing trailer", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["app.js"] = "export const answer = 42;\n"
		bundleFails(t, fixture.bundle(t), "lacks its", "sourceMappingURL=app.js.map")
	})
	t.Run("empty script", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["app.js"] = ""
		bundleFails(t, fixture.bundle(t), "is empty")
	})
	t.Run("bad entry", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		assembled.Entry = "app.ts"
		bundleFails(t, assembled, "entry must be")
	})
	t.Run("bad table", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		assembled.Table = "diagnostics/table.txt"
		bundleFails(t, assembled, "table must be")
	})
	t.Run("absolute path", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["/app.js"] = fixture.files["app.js"]
		delete(fixture.files, "app.js")
		bundleFails(t, fixture.bundle(t), "not normalized")
	})
	t.Run("duplicate file", func(t *testing.T) {
		assembled := baseBundleFixture().bundle(t)
		assembled.Files = append(assembled.Files, assembled.Files[0])
		bundleFails(t, assembled, "duplicate published file")
	})
	t.Run("no files", func(t *testing.T) {
		bundleFails(t, Bundle{Entry: "app.js", Table: "diagnostics/table.json"}, "publishes no files")
	})
}

func TestAuditBundleRejectsMapDefects(t *testing.T) {
	base := baseBundleFixture()
	withMap := func(raw string) Bundle {
		edited := base
		edited.files = map[string]string{}
		for path, body := range base.files {
			edited.files[path] = body
		}
		edited.files["app.js.map"] = raw
		return edited.bundle(t)
	}
	t.Run("not json", func(t *testing.T) {
		bundleFails(t, withMap(`{`), "invalid source map")
	})
	t.Run("bad version", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":2,"file":"app.js","sources":["x"],"names":[],"mappings":""}`), "unsupported source map version")
	})
	t.Run("file mismatch", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"other.js","sources":["x"],"names":[],"mappings":""}`), "names file")
	})
	t.Run("empty sources", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":[],"names":[],"mappings":""}`), "no sources")
	})
	t.Run("remote source", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":["https://example.invalid/a.js"],"names":[],"mappings":""}`), "remote source")
	})
	t.Run("file source", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":["file:///etc/a.js"],"names":[],"mappings":""}`), "remote source")
	})
	t.Run("malformed mappings", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":["x"],"names":[],"mappings":"A,"}`), "malformed mappings")
	})
	t.Run("segment outside script", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":["x"],"names":[],"mappings":";;;;;;;;AAAA"}`), "outside the module")
	})
	t.Run("source index overflow", func(t *testing.T) {
		bundleFails(t, withMap(`{"version":3,"file":"app.js","sources":["x"],"names":[],"mappings":"ACAA"}`), "cites source 1 of 1")
	})
}

func TestAuditBundleRejectsTableDefects(t *testing.T) {
	base := baseBundleFixture()
	withTable := func(raw string) Bundle {
		edited := base
		edited.files = map[string]string{}
		for path, body := range base.files {
			edited.files[path] = body
		}
		edited.files["diagnostics/table.json"] = raw
		return edited.bundle(t)
	}
	t.Run("not json", func(t *testing.T) {
		bundleFails(t, withTable(`{`), "not JSON")
	})
	for _, key := range []string{"stack", "stackTrace", "cause", "nativeCause", "rawInput", "raw", "secret", "secretBytes"} {
		t.Run("excluded "+key, func(t *testing.T) {
			bundleFails(t, withTable(`{"records":[{"category":"startup","`+key+`":"x"}]}`), "excluded diagnostic data", key)
		})
	}
	t.Run("nested excluded key", func(t *testing.T) {
		bundleFails(t, withTable(`{"records":[{"detail":{"stackTrace":"x"}}]}`), "excluded diagnostic data", "stackTrace")
	})
	t.Run("array table", func(t *testing.T) {
		if err := AuditBundle(withTable(`[{"category":"startup","phase":"boot","file":"a","line":1,"column":0}]`)); err != nil {
			t.Fatalf("array table rejected: %v", err)
		}
	})
}

func TestAuditBundleRejectsSecrets(t *testing.T) {
	t.Run("canary in script string", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["app.js"] = "export const leak = \"__CAN_SECRET_CANARY__\";\n//# sourceMappingURL=app.js.map\n"
		bundleFails(t, fixture.bundle(t), "secret canary", "__CAN_SECRET_CANARY__")
	})
	t.Run("canary in map content", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["app.js.map"] = `{"version":3,"file":"app.js","sources":["x"],"sourcesContent":["CANARY-SECRET-DO-NOT-SHIP"],"names":[],"mappings":""}`
		bundleFails(t, fixture.bundle(t), "secret canary")
	})
	t.Run("canary in table", func(t *testing.T) {
		fixture := baseBundleFixture()
		fixture.files["diagnostics/table.json"] = `{"records":[{"note":"-----BEGIN PRIVATE KEY-----"}]}`
		bundleFails(t, fixture.bundle(t), "secret canary")
	})
	t.Run("build secret in script", func(t *testing.T) {
		fixture := baseBundleFixture()
		secret := "https://s3cr3t-build-matter.invalid"
		fixture.files["app.js"] = "export const endpoint = \"" + secret + "\";\n//# sourceMappingURL=app.js.map\n"
		bundleFails(t, fixture.bundle(t, secret), "build-known secret material", "app.js:1:")
	})
	t.Run("empty secret ignored", func(t *testing.T) {
		if err := AuditBundle(baseBundleFixture().bundle(t, "")); err != nil {
			t.Fatalf("empty secret rejected: %v", err)
		}
	})
}

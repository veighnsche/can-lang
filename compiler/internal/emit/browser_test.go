package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func browserEmitProgram(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
	}
	for name, text := range files {
		all[name] = text
	}
	for name, text := range all {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

const browserPureSource = `package app
    provides []
    uses [text, codec, bytes]
record point
    int x
fn point load
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"x\":1}" => ok point(1)
    match chain
        call bytes::from_utf8(text) as bytes::buffer raw
        call codec::decode_json<point>(raw) as point found
        codec::invalid_data
        ok => ok found
fn void main
    emits [codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call load("{\"x\":1}")
        codec::invalid_data
        ok point found => ok
`

func TestBrowserModulesEmitDistinctRoot(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	artifacts, err := BrowserModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatalf("browser emission rejected: %v", err)
	}
	byPath := map[string]ir.Artifact{}
	for _, artifact := range artifacts {
		byPath[artifact.Path] = artifact
	}
	entry, ok := byPath[browser.BrowserEntry]
	if !ok {
		t.Fatalf("missing %s; paths: %v", browser.BrowserEntry, pathsOf(artifacts))
	}
	if _, forbidden := byPath["entry.ts"]; forbidden {
		t.Fatal("browser generation carries the bun entry")
	}
	body := string(entry.Bytes)
	if !strings.Contains(body, `BROWSER_PROFILE = "browser-main"`) {
		t.Fatalf("entry lacks the profile marker:\n%s", body)
	}
	if !strings.Contains(body, "$canBrowserMain") || !strings.Contains(body, "$canInitialize()") {
		t.Fatalf("entry lacks the browser main lifecycle:\n%s", body)
	}
	for _, token := range []string{"process.", "Bun.", "require(", "node:"} {
		if strings.Contains(body, token) {
			t.Fatalf("entry contains host token %q", token)
		}
	}
	for _, artifact := range artifacts {
		if !strings.HasSuffix(artifact.Path, ".ts") {
			continue
		}
		for _, edge := range artifact.Imports {
			for _, denied := range []string{"/platform/sql/", "/platform/process/", "/platform/files/", "/platform/env.ts", "/platform/crypto/", "/platform/s3", "/platform/server.ts", "/platform/io.ts", "/environment.ts", "/platform/cli.ts", "/ai/questions.ts"} {
				if strings.Contains(edge, denied) {
					t.Fatalf("%s reaches %s via %q", artifact.Path, denied, edge)
				}
			}
		}
		text := string(artifact.Bytes)
		for _, factory := range []string{"$canSQL", "$canCrypto", "$canPasswords", "$canFiles", "$canProcess", "$canStream", "$canWebSocket", "$canS3", "$canIO", "$canEnv", "$canCLI", "$canServer", "$canSHA256", "$canFetch", "$canResponses", "$canAI"} {
			if strings.Contains(text, factory) {
				t.Fatalf("%s references server factory %s", artifact.Path, factory)
			}
		}
	}
}

func pathsOf(artifacts []ir.Artifact) []string {
	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}
	return paths
}

func TestBrowserModulesRejectForbiddenProgram(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": `package app
    provides []
    uses [env, http]
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call env::required("HOME")
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`})
	if _, err := BrowserModules(program, "runtime", httpDependencies(t)); err == nil {
		t.Fatal("browser emission admitted a server capability")
	} else if !strings.Contains(err.Error(), "can.std.env@1::required") {
		t.Fatalf("emission diagnostic %q loses the operation", err.Error())
	}
}

func TestBrowserSharesStrictWireCodec(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	bun, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	browserArtifacts, err := BrowserModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	codecOf := func(artifacts []ir.Artifact) string {
		for _, artifact := range artifacts {
			if artifact.Path == "program/state.ts" {
				text := string(artifact.Bytes)
				for _, line := range strings.Split(text, "\n") {
					if strings.Contains(line, "$canCreateCodec") {
						return line
					}
				}
			}
		}
		return ""
	}
	bunCodec, browserCodec := codecOf(bun), codecOf(browserArtifacts)
	if bunCodec == "" || browserCodec == "" {
		t.Fatal("missing codec specialization constant")
	}
	if bunCodec != browserCodec {
		t.Fatalf("codec constants diverge:\nbun:     %s\nbrowser: %s", bunCodec, browserCodec)
	}
}

func TestBunProfileKeepsFullState(t *testing.T) {
	program := browserEmitProgram(t, map[string]string{"src/main.can": browserPureSource})
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if artifact.Path != "program/state.ts" {
			continue
		}
		text := string(artifact.Bytes)
		for _, factory := range []string{"$canEnv", "$canCLI", "$canServer", "$canSQL", "$canCrypto"} {
			if !strings.Contains(text, factory) {
				t.Fatalf("bun state lost %s", factory)
			}
		}
		return
	}
	t.Fatal("missing state module")
}

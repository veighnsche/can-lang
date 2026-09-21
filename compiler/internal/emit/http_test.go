package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

func httpDependencies(t *testing.T) []ir.Artifact {
	t.Helper()
	var dependencies []ir.Artifact
	root := "../../../runtime"
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		dependencies = append(dependencies, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", relative))})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return dependencies
}

func httpProgram(t *testing.T) string {
	t.Helper()
	program := sourceProgram(t, "../../testdata/current/http/main.can")
	dependencies := httpDependencies(t)
	programArtifacts, err := ProgramModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	assertionArtifacts, err := AssertionModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range append(programArtifacts, assertionArtifacts...) {
		bodies = append(bodies, string(artifact.Bytes))
	}
	return strings.Join(bodies, "\n")
}

func TestHTTPEmissionBindsRequestsResponsesAndRouter(t *testing.T) {
	joined := httpProgram(t)
	for _, want := range []string{
		"$canHTTPRequests.method",
		"$canHTTPRequests.path",
		"$canHTTPRequests.headers",
		"$canHTTPRequests.queryOne",
		"$canHTTPRequests.queryAll",
		"$canHTTPRequests.body",
		"$canHTTPResponses.makeStatus",
		"$canHTTPResponses.makeBodyStatus",
		"$canHTTPResponses.ok",
		"$canHTTPResponses.unprocessable",
		"$canHTTPResponses.emptyHeaders",
		"$canHTTPResponses.text",
		"$canHTTPResponses.html",
		"$canRouter.get",
		"$canRouter.post",
		"$canRouter.make",
		"$canScopeRequest($canContext)",
		"platform/http.ts",
		"platform/router.ts",
		"$canIsHTTP($canHTTPKinds[identity],value)",
		"$canIsRouter($canHTTPKinds[identity],value)",
		"$canHTTP0.decode",
		"$canHTTP1.decode",
		"$canHTTP2.encode",
		`"name":"query","kind":"str"`,
		`"name":"note","kind":"optional"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("generated HTTP program omits %s", want)
		}
	}
	if strings.Contains(joined, "$canResponses.json") || strings.Contains(joined, "$canResponses.form") {
		t.Fatal("HTTP descriptors reuse the AI response alias")
	}
}

func TestHTTPEmissionAvoidsAIResponseAliases(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/native/declarations.can")
	dependencies := httpDependencies(t)
	if _, err := ProgramModules(program, "runtime", dependencies); err != nil {
		t.Fatalf("AI program modules conflict with HTTP aliases: %v", err)
	}
	if _, err := AssertionModules(program, "runtime", dependencies); err != nil {
		t.Fatalf("AI assertion modules conflict with HTTP aliases: %v", err)
	}
}

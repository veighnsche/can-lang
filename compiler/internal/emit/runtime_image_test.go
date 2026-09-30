package emit

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

const imageInspectEmitSource = "package app\n" +
	"    provides []\n" +
	"    uses [bytes, codec, image]\n" +
	"fn image::metadata inspect_image\n" +
	"    emits [codec::invalid_data, image::invalid_image]\n" +
	"    asserts\n" +
	"        sample: => ok image::metadata(\"png\", 1, 1)\n" +
	"    match chain\n" +
	"        call bytes::decode_base64(\"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/+ioAAAAASUVORK5CYII=\") as bytes::buffer source\n" +
	"        call image::inspect(source, 100) as image::metadata metadata\n" +
	"        codec::invalid_data\n" +
	"        image::invalid_image\n" +
	"        ok => ok metadata\n" +
	"fn void main\n" +
	"    emits [codec::invalid_data, image::invalid_image]\n" +
	"    given\n" +
	"        str[] args\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    match call inspect_image()\n" +
	"        codec::invalid_data\n" +
	"        image::invalid_image\n" +
	"        ok image::metadata found => ok\n"

func TestImageInspectionBindingTargetsNativeAdapter(t *testing.T) {
	const identity = "can.std.image@1::inspect"
	if got := coreOperationBindings().functions[identity]; got != "$canImage.inspect" {
		t.Fatalf("image inspection binding = %q, want %q", got, "$canImage.inspect")
	}
}

func TestImageInspectEmissionImportsStateImage(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/main.can": imageInspectEmitSource})
	dependencies := httpDependencies(t)

	production, err := ProgramModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	assertImageCallHasStateImport(t, production, "bun production")

	assertions, err := AssertionModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	assertImageCallHasStateImport(t, assertions, "bun assertion")
	cases := 0
	for _, artifact := range assertions {
		if !strings.HasPrefix(artifact.Path, "assertions/") {
			continue
		}
		cases++
		if !strings.Contains(string(artifact.Bytes), "$canImage as $canImage") {
			t.Fatalf("bun assertion case %s omits the $canImage state import", artifact.Path)
		}
	}
	if cases == 0 {
		t.Fatal("bun assertion emission has no assertion cases")
	}

	browserArtifacts, err := BrowserModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	assertImageCallHasStateImport(t, browserArtifacts, "browser")
}

func assertImageCallHasStateImport(t *testing.T, artifacts []ir.Artifact, profile string) {
	t.Helper()
	found := 0
	for _, artifact := range artifacts {
		body := string(artifact.Bytes)
		if !strings.Contains(body, "$canImage.inspect") {
			continue
		}
		if artifact.Path == programStatePath {
			continue
		}
		found++
		if !strings.Contains(body, "$canImage as $canImage") {
			t.Fatalf("%s module %s calls $canImage.inspect without importing $canImage", profile, artifact.Path)
		}
	}
	if found == 0 {
		t.Fatalf("%s emission has no $canImage.inspect call", profile)
	}
}

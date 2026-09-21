package driver

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestPackagedOutputValidationAndExecution(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged native output validation")
	}
	if os.Getenv("CAN_OUTPUT_OFFLINE_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", os.Args[0], "-test.run", "^TestPackagedOutputValidationAndExecution$", "-test.v")
		cmd.Env = append(os.Environ(), "CAN_OUTPUT_OFFLINE_CHILD=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("offline output integration: %v\n%s", err, output)
		}
		t.Log(string(output))
		return
	}
	sourceRoot, _ := filepath.Abs("../../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "output-test")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(bundle, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := resolve(filepath.Join(bundle, "bin/canlc"), distribution.Hash(raw))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := runtime.PrivateArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	root := outputProject(t)
	store := outputBegin(t, root)
	inputs := store.BuildInputs(strings.Repeat("a", 64), strings.Repeat("b", 64), assets.Identity, strings.Repeat("c", 64))
	makeOutput := func(source string, imports []string, additional ...OutputArtifact) *PreparedOutput {
		artifacts := append([]OutputArtifact{}, assets.Files...)
		artifacts = append(artifacts, additional...)
		artifacts = append(artifacts, OutputArtifact{Path: "entry.ts", Bytes: []byte(source), Imports: imports})
		p, err := PrepareOutput(inputs, "entry.ts", artifacts)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, source := range []string{`const target = "node:fs"; await import(target);`,
		"const target = 'fs'; await import(`node:${target}`);",
		`export async function dormant(target: string) { return import(/* hidden edge */ target); }`,
		`export const nested = () => (() => import(String("node:fs")));`, `const x = ;`, `import {readFileSync} from "node:fs"; console.log(readFileSync);`} {
		p := makeOutput(source, nil)
		if err = runtime.ValidateOutput(ctx, p); err == nil {
			t.Fatal("invalid syntax/import inventory admitted")
		}
		if _, err = store.Publish(p); err == nil {
			t.Fatal("failed validation published")
		}
	}
	specifier := "./" + assets.Directory + "/failure.ts"
	program := `import {captureStandard,standardFailureOccurrenceID} from "` + specifier + `";
import {makeFailure} from "./packages/p-helper/helper.ts";
const imported=makeFailure();if(captureStandard(imported,{source:"outer",start:0,end:1,invocation:[]})!==imported)throw new Error("private runtime duplicated");
const a=captureStandard("same",{source:"app::main",start:0,end:1,invocation:[]});
const b=captureStandard("same",{source:"app::main",start:0,end:1,invocation:[]});
if(standardFailureOccurrenceID(a)===standardFailureOccurrenceID(b))throw new Error("merged occurrence");
console.log("published generation executed");`
	helper := OutputArtifact{Path: "packages/p-helper/helper.ts", Imports: []string{"../../" + assets.Directory + "/failure.ts"}, Bytes: []byte(`import {captureStandard} from "../../` + assets.Directory + `/failure.ts"; export function makeFailure(){return captureStandard("helper",{source:"helper",start:0,end:1,invocation:[]})}`)}
	p := makeOutput(program, []string{specifier, "./packages/p-helper/helper.ts"}, helper)
	if err = runtime.ValidateOutput(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Publish(p); err != nil {
		t.Fatal(err)
	}
	lease, err := store.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err = runtime.RunOutput(ctx, lease, nil, []string{"UNUSED_LARGE_SNAPSHOT=" + strings.Repeat("x", 1024*1024)}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "published generation executed" {
		t.Fatal(stdout.String())
	}
	// A child retains an inherited kernel lease after its parent closes its copy.
	lease, err = store.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, runtime.Executable, "--no-env-file", "--no-macros", "--no-install", "-e", `console.log("lease ready"); await Bun.sleep(60000);`)
	command.Dir = t.TempDir()
	command.Env = []string{"HOME=" + command.Dir, "XDG_CONFIG_HOME=" + command.Dir, "PATH=/nonexistent"}
	command.ExtraFiles = []*os.File{lease.File()}
	pipe, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	defer command.Process.Kill()
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() || scanner.Text() != "lease ready" {
		t.Fatal("lease child did not start")
	}
	directory := lease.Directory
	lease.Close()
	if err = store.Clean(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); err != nil {
		t.Fatal("inherited lease lost", err)
	}
	command.Process.Kill()
	command.Wait()
	if err = store.Prune(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("released child generation retained", err)
	}
	t.Log("native parser validated exact imports; published entry executed; inherited child lease survived launcher close")
}

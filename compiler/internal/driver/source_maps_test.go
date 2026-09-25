package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/distribution"
)

func TestPackagedSourceMaps(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline source map qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if os.Getenv("CAN_MAPS_OFFLINE_CHILD") != "1" {
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", os.Args[0], "-test.run", "^TestPackagedSourceMaps$", "-test.v")
		cmd.Env = append(os.Environ(), "CAN_MAPS_OFFLINE_CHILD=1")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("offline maps: %v\n%s", err, data)
		}
		t.Log(string(data))
		return
	}
	sourceRoot, _ := filepath.Abs("../../..")
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "source-maps")
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
	root := outputProject(t)
	text := `package app
    provides []
    uses []
fn str first
    emits []
    given
        str[] values
    asserts
        sample: ["x"] => ok "😀x"
    ok "😀" + values[0]
fn void main
    emits []
    given
        str[] args
    asserts
        sample: ["x"] => ok
    match call first(args)
        ok str s => ok
`
	if err = os.WriteFile(filepath.Join(root, "src/main.can"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	var buildDiagnostics bytes.Buffer
	if _, err = runtime.Build(ctx, root, os.Environ(), nil, &buildDiagnostics, DefaultAssertTimeoutMs, ""); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
	if err != nil {
		t.Fatal(err)
	}
	store := outputBegin(t, root)
	lease, err := store.AcquireCurrent()
	if err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if err = runtime.RunOutput(ctx, lease, nil, []string{"SECRET=must-not-disclose"}, nil, &out, &diag); err == nil {
		t.Fatal("bounds failure exited successfully")
	}
	var report struct {
		Category string `json:"category"`
		Frames   []struct {
			Source, File, Operation  string
			Start, End, Line, Column int
			Synthetic                bool
		} `json:"frames"`
	}
	if err = json.Unmarshal(diag.Bytes(), &report); err != nil {
		t.Fatal(err, diag.String())
	}
	file, _ := source.New("main.can", text)
	start := strings.Index(text, "values[0]")
	position, _ := file.MapPosition(start)
	found := false
	for _, frame := range report.Frames {
		if frame.Start == start && frame.End == start+len("values[0]") && frame.Line == position.Line && frame.Column == position.Column+1 && frame.Operation == "index" && !frame.Synthetic && frame.File == "main.can" {
			found = true
		}
	}
	if report.Category != "bounds" || !found || out.Len() != 0 {
		t.Fatalf("wrong Can span: %s %s", out.String(), diag.String())
	}
	for _, secret := range []string{root, bundle, "must-not-disclose", "Error:", "primitive.ts"} {
		if strings.Contains(diag.String(), secret) {
			t.Fatal("diagnostic leaked", secret)
		}
	}
	t.Log("mapped native failure:", diag.String())
	program, err := check.CheckProgram(store.Graph)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := runtime.PrivateArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	emitted, err := emit.ProgramModules(program, assets.Directory, assets.Files)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := runtime.encodeSourceMaps(ctx, program, emitted)
	if err != nil {
		t.Fatal(err)
	}
	inputs := store.BuildInputs(strings.Repeat("a", 64), strings.Repeat("b", 64), assets.Identity, strings.Repeat("c", 64))
	for _, mode := range []string{"missing-map", "missing-index", "corrupt-map", "unknown-source"} {
		t.Run(mode, func(t *testing.T) {
			var changed []ir.Artifact
			mutated := false
			for _, a := range artifacts {
				if mode == "missing-index" && a.Path == "diagnostics/source-index.json" {
					continue
				}
				if strings.HasSuffix(a.Path, ".ts.map") && !mutated {
					mutated = true
					if mode == "missing-map" {
						continue
					}
					if mode == "corrupt-map" {
						var m map[string]any
						json.Unmarshal(a.Bytes, &m)
						m["mappings"] = "?"
						a.Bytes, _ = json.Marshal(m)
					}
				}
				if mode == "unknown-source" && a.Path == "diagnostics/source-index.json" {
					var index sourceIndex
					json.Unmarshal(a.Bytes, &index)
					for i := range index.Modules {
						if len(index.Modules[i].Segments) > 0 {
							index.Modules[i].Segments[0].Source = "missing"
							break
						}
					}
					a.Bytes, _ = json.Marshal(index)
				}
				changed = append(changed, a)
			}
			p, err := PrepareOutput(inputs, "entry.ts", changed)
			if err == nil {
				err = runtime.ValidateOutput(ctx, p)
				if err == nil {
					t.Fatal("bad map admitted")
				}
				if _, err = store.Publish(p); err == nil {
					t.Fatal("bad map published")
				}
			}
			after, _ := os.ReadFile(filepath.Join(root, "dist/current.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("failed mapping changed current")
			}
		})
	}
	broken := *runtime
	broken.Executable = "/usr/bin/false"
	if _, err = broken.encodeSourceMaps(ctx, program, emitted); err == nil {
		t.Fatal("helper failure ignored")
	}
	after, _ := os.ReadFile(filepath.Join(root, "dist/current.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("helper failure changed current")
	}
}

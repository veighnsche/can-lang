package check_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

func errorFixture(t *testing.T) (*resolve.World, *check.ErrorRegistry, map[string]*types.Type) {
	t.Helper()
	root := t.TempDir()
	var packages []string
	for _, p := range catalogue.Builtin().Inventory().Packages {
		packages = append(packages, p.Name)
	}
	text := "package app\n    provides []\n    uses [" + strings.Join(packages, ", ") + "]\nerror failed<item>(item value)\nerror nested(option::value<int> value)\nvariant failure\n    failed<int>\n    standard_failure\n"
	for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":["app::failed","app::nested"],"retired":["app::legacy"]}`, "src/main.can": text} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	world, err := resolve.Build(graph)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := check.ErrorDeclarations(world)
	if err != nil {
		t.Fatal(err)
	}
	builder := types.NewBuilder(world)
	if err = builder.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	var file *resolve.File
	for _, f := range world.Files {
		file = f
	}
	names := []string{"failed<int>", "failed<str>", "nested", "all_failed<failure>", "standard_failure", "int", "str"}
	for _, e := range catalogue.Builtin().Inventory().Errors {
		if len(e.Parameters) == 0 {
			names = append(names, e.Name)
		}
	}
	out := map[string]*types.Type{}
	for _, name := range names {
		src, _ := source.New("annotation.can", name)
		node, ds := syntax.ParseType(src)
		if len(ds) != 0 {
			t.Fatal(ds)
		}
		out[name], err = builder.Resolve(file, node, nil, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = builder.Finish(); err != nil {
		t.Fatal(err)
	}
	return world, registry, out
}
func bound(t *testing.T, r *check.ErrorRegistry, ts ...*types.Type) check.ErrorBound {
	t.Helper()
	b, err := r.Bound(ts)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestExactErrorBoundsAndAllocation(t *testing.T) {
	_, r, ts := errorFixture(t)
	a := bound(t, r, ts["failed<int>"])
	b := bound(t, r, ts["failed<str>"])
	both := bound(t, r, ts["failed<int>"], ts["failed<str>"])
	if a.Entries()[0].Declaration.Identity != b.Entries()[0].Declaration.Identity || a.Entries()[0].TypeIdentity == b.Entries()[0].TypeIdentity {
		t.Fatal("generic identity was lost")
	}
	if a.Entries()[0].Declaration.Name != "app::failed" {
		t.Fatalf("qualified kind lost: %+v", a.Entries()[0].Declaration)
	}
	if err := both.CheckEscaping(a); err != nil {
		t.Fatal(err)
	}
	if err := a.CheckEscaping(b); err == nil {
		t.Fatal("payload specialization widened")
	}
	if err := bound(t, r).CheckEscaping(a); err == nil {
		t.Fatal("undeclared domain error escaped")
	}
	if _, err := r.Bound([]*types.Type{ts["failed<int>"], ts["failed<int>"]}); err == nil {
		t.Fatal("duplicate authored emits")
	}
	for _, name := range []string{"int", "standard_failure"} {
		if _, err := r.Bound([]*types.Type{ts[name]}); err == nil {
			t.Fatal("non-domain bound", name)
		}
	}
	id := a.Entries()[0].Declaration.Identity
	if _, err := a.ResolveBareArm(id); err != nil {
		t.Fatal(err)
	}
	if _, err := both.ResolveBareArm(id); err == nil {
		t.Fatal("ambiguous bare error pattern")
	}
	if _, err := a.ResolveBareArm("unknown"); err == nil {
		t.Fatal("unknown arm")
	}
	entries := a.Entries()
	entries[0].Arguments[0] = "tampered"
	if a.Entries()[0].Arguments[0] == "tampered" {
		t.Fatal("mutable bound evidence")
	}
	first, _ := json.Marshal(r.Plan(both))
	second, _ := json.Marshal(r.Plan(both))
	if string(first) != string(second) {
		t.Fatal("nondeterministic failure plan")
	}
}

func TestNumberedErrorDeclarationsRejected(t *testing.T) {
	for _, spelling := range []string{"1000000", "0xf4240", "0b11110100001001000000", "0o3641100"} {
		t.Run(spelling, func(t *testing.T) {
			root := t.TempDir()
			for name, data := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":["app::failed"],"retired":[]}`, "src/main.can": "package app\n    provides []\n    uses []\nerror " + spelling + " failed(item value)\n"} {
				p := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := project.Load(root); err == nil {
				t.Fatalf("numbered error %q admitted", spelling)
			}
		})
	}
}

func TestErrorRegistryRejectsBrokenWorldAgreement(t *testing.T) {
	for _, mode := range []string{"duplicate", "mismatch", "missing", "retired", "redeclare", "owner", "chain", "unchained"} {
		t.Run(mode, func(t *testing.T) {
			w, _, _ := errorFixture(t)
			owner := w.Graph.Projects[""]
			switch mode {
			case "duplicate":
				owner.Registry.Active = append(owner.Registry.Active, owner.Registry.Active[0])
			case "mismatch":
				owner.Registry.Active[0] = "app::renamed"
			case "missing":
				owner.Registry.Active = owner.Registry.Active[1:]
			case "retired":
				owner.Registry.Active = append(owner.Registry.Active, "app::legacy")
			case "redeclare":
				owner.Registry.Active = []string{"app::nested"}
				owner.Registry.Retired = []string{"app::failed"}
			case "owner":
				owner.Registry.Active[0] = "other::failed"
			case "chain":
				owner.Registry.Predecessors = map[string][]string{"app::failed": {"app::unknown"}}
			case "unchained":
				owner.Registry.Predecessors = map[string][]string{"app::missing": {"app::legacy"}}
			}
			if _, err := check.ErrorDeclarations(w); err == nil {
				t.Fatal("broken allocation admitted")
			}
		})
	}
}

func TestEmittedFailurePlanOnQualifiedBun(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to execute checked failure plan")
	}
	target := distribution.PinnedTarget()
	binary, err := os.ReadFile(bun)
	if !filepath.IsAbs(bun) || err != nil || distribution.Hash(binary) != target.Runtime.SHA256 || runtime.GOOS != target.Runtime.Platform || runtime.GOARCH != target.Runtime.Architecture {
		t.Fatal("CAN_BUN is not the qualified target")
	}
	_, r, ts := errorFixture(t)
	var errors []*types.Type
	for _, typ := range ts {
		if typ.Kind() == types.Error {
			errors = append(errors, typ)
		}
	}
	plan := r.Plan(bound(t, r, errors...))
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	runtimePath := filepath.Join(cwd, "../../../runtime")
	quote := func(s string) string { v, _ := json.Marshal(s); return string(v) }
	// Parse/check/lower projections as authored expressions, retaining the opaque
	// boundary rather than emitting a JavaScript property access to private data.
	c := check.Expressions{Scalars: ts, Value: func(_ syntax.QualifiedName) (check.ValueBinding, error) {
		return check.ValueBinding{Identity: "failure", Type: ts["standard_failure"]}, nil
	}}
	var projections strings.Builder
	for _, name := range []string{"kind", "message", "occurrence_id"} {
		src, _ := source.New("projection.can", "failure."+name)
		node, ds := syntax.ParseExpression(src)
		if len(ds) != 0 {
			t.Fatal(ds)
		}
		checked, err := c.Check(node, nil)
		if err != nil {
			t.Fatal(err)
		}
		emitter := emit.ExpressionEmitter{Bindings: map[string]string{"failure": "standard"}}
		lowered, err := emitter.Lower(checked)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"kind": `"native_exception"`, "message": `"Error: safe"`, "occurrence_id": "standardFailureOccurrenceID(standard)"}[name]
		projections.WriteString("{\n" + lowered.Statements + "assert.equal(" + lowered.Value + ", " + want + ");\n}\n")
	}
	for _, name := range []string{"cause", "stack", "origin"} {
		src, _ := source.New("hidden.can", "failure."+name)
		node, _ := syntax.ParseExpression(src)
		if _, err := c.Check(node, nil); err == nil {
			t.Fatal("private failure projection admitted", name)
		}
	}
	program := `import { strict as assert } from "node:assert";
import { createDomainRuntime, domainFailureDiagnostics } from ` + quote(filepath.Join(runtimePath, "domain.ts")) + `;
import { record } from ` + quote(filepath.Join(runtimePath, "data.ts")) + `;
import { captureStandard, standardFailureOccurrenceID } from ` + quote(filepath.Join(runtimePath, "failure.ts")) + `;` + emit.FailureImports(filepath.Join(runtimePath, "failure.ts")) + `
const plan=` + string(encoded) + `;
const runtime=createDomainRuntime(plan);
const shapes=new Map(plan.shapes.map(s=>[s.identity,s]));
const make=(id)=>{const s=shapes.get(id);switch(s.kind){
 case "primitive":return s.declaration==="int"?1n:s.declaration==="str"?"safe":s.declaration==="bool"?true:1.5;
 case "array":return Object.freeze([]);
 case "variant":return make(s.leaves[0]);
 case "opaque":return captureStandard(new Error("safe"),origin);
 case "record":case "error":return record(id,s.fields.map(f=>[f.name,make(f.type)]));
 default:throw new Error("unsupported test shape");}};
const origin={source:"app::run",start:3,end:9,invocation:["root"]};
for(const s of plan.shapes.filter(s=>s.kind==="error")){
 const value=make(s.identity);assert(runtime.accepts(s.identity,value));
 const first=runtime.create(s.identity,value,origin),second=runtime.create(s.identity,value,origin);
 assert.notEqual(domainFailureDiagnostics(first).occurrenceID,domainFailureDiagnostics(second).occurrenceID);
 assert.equal(runtime.checkBound(first,[s.identity]),first);assert.throws(()=>runtime.checkBound(first,[]));
 assert.equal(domainFailureDiagnostics(first).origin.source,"app::run");
 if(s.fields.length){assert(!runtime.accepts(s.identity,record(s.identity,[])));assert(!runtime.accepts(s.identity,record(s.identity,s.fields.map(f=>[f.name,null]))));}
}
const standard=captureStandard(new Error("safe"),origin);
` + projections.String() + `console.log("checked error plans and projections passed");`
	file := filepath.Join(t.TempDir(), "failures.ts")
	if err = os.WriteFile(file, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(bun, "--no-env-file", "--no-macros", "--no-install", file).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}

func TestResolveExactArm(t *testing.T) {
	_, r, ts := errorFixture(t)
	both := bound(t, r, ts["failed<int>"], ts["failed<str>"])
	hit, err := both.ResolveExactArm(ts["failed<int>"].Identity())
	if err != nil || hit.TypeIdentity != ts["failed<int>"].Identity() {
		t.Fatalf("exact specialization missed: %+v %v", hit, err)
	}
	if _, err = both.ResolveExactArm(ts["nested"].Identity()); err == nil {
		t.Fatal("foreign specialization admitted")
	}
	if _, err = both.ResolveBareArm(hit.Declaration.Identity); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("bare head over two specializations admitted: %v", err)
	} else if !strings.Contains(err.Error(), ts["failed<int>"].Identity()) || !strings.Contains(err.Error(), ts["failed<str>"].Identity()) {
		t.Fatalf("ambiguity hides alternatives: %v", err)
	}
}

package emit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func programImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/assert/fixtures.ts", Names: []ImportName{{"withFixture", "$canWithFixture"}}},
		{Target: runtime + "/completion.ts", Names: []ImportName{{"success", "$canSuccess"}, {"failure", "$canFailure"}, {"value", "$canValue"}, {"invoke", "$canInvoke"}, {"caught", "$canCaught"}, {"errorType", "$canErrorType"}, {"errorPayload", "$canErrorPayload"}}},
		{Target: runtime + "/completion.ts", TypeOnly: true, Names: []ImportName{{"Completion", "$canCompletion"}, {"AssertionContext", "$canAssertionContext"}}},
		{Target: runtime + "/data.ts", Names: []ImportName{{"record", "$canRecord"}, {"update", "$canUpdate"}, {"array", "$canArray"}, {"recordIdentity", "$canRecordIdentity"}}},
		{Target: runtime + "/primitive.ts", Names: []ImportName{{"intDivide", "$canIntDivide"}, {"intRemainder", "$canIntRemainder"}, {"intPower", "$canIntPower"}, {"index", "$canIndex"}, {"slice", "$canSlice"}}},
		{Target: runtime + "/failure.ts", Names: []ImportName{{"captureStandard", "$canCaptureStandard"}, {"isStandardFailure", "$canIsStandardFailure"}, {"standardFailureMessage", "$canStandardMessage"}, {"standardFailureKind", "$canFailureKind"}, {"standardFailureMessage", "$canFailureMessage"}, {"standardFailureOccurrenceID", "$canFailureOccurrenceID"}}},
	}
}

// ProgramModules keeps every source module inert. One shared initialization
// function creates error evidence and evaluates the ordered top-level values;
// the entry supervisor invokes it before calling the checked main region.
func ProgramModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, false)
}
func AssertionModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, true)
}
func programModules(program *check.Program, runtime string, dependencies []ir.Artifact, assertions bool) ([]ir.Artifact, error) {
	if program == nil || (!assertions && (program.Entry == nil || program.Entry.Region == nil)) {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	const statePath = "program/state.ts"
	functions := map[string]string{
		"can.std.bytes@1::from_utf8": "$canCLI.fromUTF8",
		"can.std.io@1::stdout_write": "$canCLI.stdoutWrite",
		"can.std.io@1::stderr_write": "$canCLI.stderrWrite",
	}
	bindings := map[string]string{}
	for i, fn := range program.Functions {
		functions[fn.Symbol.ID] = fmt.Sprintf("$canFunction%d", i)
	}
	for _, value := range program.Initializers {
		bindings[value.Identity] = "($canValues[" + quote(value.Identity) + "] as " + TypeName(value.Type) + ")"
	}
	var errorTypes []*types.Type
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Error {
			errorTypes = append(errorTypes, typ)
		}
	}
	bound, err := program.Registry.Bound(errorTypes)
	if err != nil {
		return nil, err
	}
	plan, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		return nil, err
	}
	initial, err := Initialization(program.Initializers, nil, true)
	if err != nil {
		return nil, err
	}
	declarations, err := NativeTypeDeclarations(program.Model.Types())
	if err != nil {
		return nil, err
	}
	var state strings.Builder
	state.WriteString(declarations)
	state.WriteString("export let $canCLI: ReturnType<typeof $canCreateCLI>;\nexport let $canDomain: ReturnType<typeof $canCreateDomain>;\nexport const $canValues: Record<string, unknown> = Object.create(null);\nexport function $canInitialize(): void {\n")
	var invalidData, writeFailed, bytesID string
	for _, typ := range program.Model.Types() {
		switch typ.Declaration() {
		case "can.std.codec@1::invalid_data":
			invalidData = typ.Identity()
		case "can.std.io@1::write_failed":
			writeFailed = typ.Identity()
		case "can.std.bytes@1::buffer":
			bytesID = typ.Identity()
		}
	}
	fmt.Fprintf(&state, "$canDomain = $canCreateDomain(%s, (identity, value) => identity === %s && $canIsBytes(value));\n", plan, quote(bytesID))
	fmt.Fprintf(&state, "$canCLI = $canCreateCLI($canDomain, {invalidData: %s, writeFailed: %s});\n", quote(invalidData), quote(writeFailed))
	state.WriteString(initial.Code)
	for _, value := range program.Initializers {
		fmt.Fprintf(&state, "$canValues[%s] = %s;\n", quote(value.Identity), initial.Bindings[value.Identity])
	}
	state.WriteString("Object.freeze($canValues);\n}\n")
	imports := append(programImports(runtime), ModuleImport{Target: runtime + "/domain.ts", Names: []ImportName{{"createDomainRuntime", "$canCreateDomain"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/cli.ts", Names: []ImportName{{"createCLI", "$canCreateCLI"}}}, ModuleImport{Target: runtime + "/bytes.ts", Names: []ImportName{{"isBytes", "$canIsBytes"}}})
	modules := []Module{{Path: statePath, Imports: imports, Body: state.String()}}
	byPath := map[string][]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		path := fn.Symbol.Source.OutputPath
		byPath[path] = append(byPath[path], fn)
	}
	var paths []string
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		var body strings.Builder
		var regions []*ir.Region
		for _, fn := range byPath[path] {
			regions = append(regions, fn.Region)
		}
		// Initializer references may name types absent from function signatures.
		allTypes := append(program.Model.Types(), regionTypes(regions...)...)
		localTypes, err := NativeTypeDeclarations(allTypes)
		if err != nil {
			return nil, err
		}
		body.WriteString(localTypes)
		for _, fn := range byPath[path] {
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID}
			code, err := emitter.Function(functions[fn.Symbol.ID], fn.Region)
			if err != nil {
				return nil, err
			}
			body.WriteString("export ")
			body.WriteString(code)
		}
		imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}}})
		for _, fn := range program.Functions {
			target := fn.Symbol.Source.OutputPath
			if target != path {
				imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{functions[fn.Symbol.ID], functions[fn.Symbol.ID]}}})
			}
		}
		modules = append(modules, Module{Path: path, Imports: imports, Body: body.String()})
	}
	if assertions {
		if len(program.Assertions) == 0 {
			return nil, fmt.Errorf("project has no concrete assertions")
		}
		entry := Module{Path: "entry.ts", Imports: []ModuleImport{
			{Target: runtime + "/assert/runner.ts", Names: []ImportName{{"runAssertions", "$canRunAssertions"}}},
			{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
			{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		}}
		var cases []string
		for i, test := range program.Assertions {
			rootJSON, err := json.Marshal(test.Root)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(append([]byte("can-assertion-root-v1\x00"), rootJSON...))
			path := fmt.Sprintf("assertions/%x.ts", digest)
			imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}}})
			for _, fn := range program.Functions {
				imports = append(imports, ModuleImport{Target: fn.Symbol.Source.OutputPath, Names: []ImportName{{functions[fn.Symbol.ID], functions[fn.Symbol.ID]}}})
			}
			graph := append(program.Model.Types(), regionTypes(test.Actual, test.Expected)...)
			body, err := NativeTypeDeclarations(graph)
			if err != nil {
				return nil, err
			}
			sourceID := ""
			for src := range program.World.Files {
				if src.Syntax.Source.Name() == test.Actual.Source {
					sourceID = src.ID
				}
			}
			if sourceID == "" {
				return nil, fmt.Errorf("assertion source is not indexed")
			}
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: sourceID}
			actual, err := emitter.Function("$canActual", test.Actual)
			if err != nil {
				return nil, err
			}
			expected, err := emitter.Function("$canExpected", test.Expected)
			if err != nil {
				return nil, err
			}
			body += actual + expected + fmt.Sprintf("export const $canCase = Object.freeze({root: Object.freeze(%s), actual: $canActual, expected: $canExpected});\n", rootJSON)
			modules = append(modules, Module{Path: path, Imports: imports, Body: body})
			name := fmt.Sprintf("$canCase%d", i)
			cases = append(cases, name)
			entry.Imports = append(entry.Imports, ModuleImport{Target: path, Names: []ImportName{{"$canCase", name}}})
		}
		entry.Body = "process.exitCode = await $canRunAssertions([" + strings.Join(cases, ",") + "], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();});\n"
		modules = append(modules, entry)
		return Modules(modules, dependencies...)
	}
	main := program.Entry
	modules = append(modules, Module{Path: "entry.ts", Imports: []ModuleImport{
		{Target: runtime + "/entry.ts", Names: []ImportName{{"runEntry", "$canRunEntry"}}},
		{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{functions[main.Symbol.ID], "$canMain"}}},
	}, Body: "process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));\n"})
	return Modules(modules, dependencies...)
}

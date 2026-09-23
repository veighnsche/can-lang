package emit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// emitAssertionModules lowers every attached assertion into its own module
// plus the assertion-root entry module.
func emitAssertionModules(assembly *programAssembly, runtime string) ([]Module, error) {
	program := assembly.program
	if len(program.Assertions) == 0 {
		return nil, fmt.Errorf("project has no concrete assertions")
	}
	modules := []Module{}
	entry := Module{Path: "entry.ts", Imports: []ModuleImport{
		{Target: runtime + "/assert/runner.ts", Names: []ImportName{{"runAssertionRoot", "$canRunAssertionRoot"}}},
		{Target: programStatePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
	}}
	var cases []string
	for i, test := range program.Assertions {
		caseModule, err := emitAssertionCase(assembly, runtime, test)
		if err != nil {
			return nil, err
		}
		modules = append(modules, caseModule)
		name := fmt.Sprintf("$canCase%d", i)
		cases = append(cases, name)
		entry.Imports = append(entry.Imports, ModuleImport{Target: caseModule.Path, Names: []ImportName{{"$canCase", name}}})
	}
	entry.Body = "process.exitCode = await $canRunAssertionRoot([" + strings.Join(cases, ",") + "], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, process.argv.slice(2));\n"
	modules = append(modules, entry)
	return modules, nil
}

// emitAssertionCase lowers one assertion root, its actual/expected regions
// and its optional raw-fixture or injected policy.
func emitAssertionCase(assembly *programAssembly, runtime string, test *ir.Assertion) (Module, error) {
	program := assembly.program
	rootJSON, err := json.Marshal(test.Root)
	if err != nil {
		return Module{}, err
	}
	digest := sha256.Sum256(append([]byte("can-assertion-root-v1\x00"), rootJSON...))
	path := fmt.Sprintf("assertions/%x.ts", digest)
	imports := append(programImports(runtime), ModuleImport{Target: programStatePath, Names: stateValueImportNames()})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/crypto/primitives.ts", Names: []ImportName{{"sha256", "$canSHA256"}}})
	for _, id := range assembly.collectionIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.collectionNames[id], assembly.collectionNames[id]}}})
	}
	for _, id := range assembly.codecIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.codecNames[id], assembly.codecNames[id]}}})
	}
	for _, id := range assembly.httpIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.httpNames[id], assembly.httpNames[id]}}})
	}
	for _, id := range assembly.sqlIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.sqlNames[id], assembly.sqlNames[id]}}})
	}
	for _, id := range assembly.txIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.txNames[id], assembly.txNames[id]}}})
	}
	for _, fn := range program.Functions {
		imports = append(imports, ModuleImport{Target: fn.Symbol.Source.OutputPath, Names: []ImportName{{assembly.functions[fn.Identity()], assembly.functions[fn.Identity()]}}})
	}
	for _, id := range assembly.nativeIDs {
		target := assembly.nativePaths[id]
		if name := assembly.functions[id]; name != "" {
			imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{name, name}}})
		}
	}
	graph := append(program.Model.Types(), regionTypes(test.Actual, test.Expected)...)
	body, err := NativeTypeDeclarations(graph)
	if err != nil {
		return Module{}, err
	}
	sourceID := ""
	for src := range program.World.Files {
		if src.Syntax.Source.Name() == test.Actual.Source {
			sourceID = src.ID
		}
	}
	if sourceID == "" {
		return Module{}, fmt.Errorf("assertion source is not indexed")
	}
	emitter := RegionEmitter{Bindings: assembly.bindings, Functions: assembly.functions, DomainRuntime: "$canDomain", SourceID: sourceID}
	actual, err := emitter.Function("$canActual", test.Actual)
	if err != nil {
		return Module{}, err
	}
	expected, err := emitter.Function("$canExpected", test.Expected)
	if err != nil {
		return Module{}, err
	}
	actualName := "$canActual"
	if test.Raw != nil {
		spec, err := RawSpec(test.Raw)
		if err != nil {
			return Module{}, err
		}
		imports = append(imports, ModuleImport{Target: runtime + "/assert/provider.ts", Names: []ImportName{{"provideRawHTTP", "$canProvideRaw"}}})
		body += fmt.Sprintf("async function $canRawActual($canContext: $canAssertionContext): Promise<$canCompletion<%s>> {\n$canProvideRaw($canContext, %s, %s);\nreturn $canActual($canContext);\n}\n", TypeName(test.Actual.Result), quote(test.Raw.Operation), spec)
		actualName = "$canRawActual"
	}
	if test.Injected != nil {
		if _, err := emitter.configure(&ir.Region{ID: test.Actual.ID + "/injected", Result: test.Injected.Value.Type, Span: test.Actual.Span}); err != nil {
			return Module{}, err
		}
		lowered, err := emitter.expression.Lower(test.Injected.Value)
		if err != nil {
			return Module{}, err
		}
		body += fmt.Sprintf("async function $canInjectActual($canContext: $canAssertionContext): Promise<$canCompletion<%s>> {\nlet $canOrigin = %s;\n%s$canPolicyProvide($canContext, %s, %s, %s, %s, %s, $canDomain, $canOrigin);\nreturn $canActual($canContext);\n}\n", TypeName(test.Actual.Result), emitter.origin(test.Actual.Span), lowered.Statements, quote(test.Injected.Operation), quote(test.Injected.Origin), quote(test.Injected.Identity), lowered.Value, quote(test.Injected.Failed))
		actualName = "$canInjectActual"
	}
	body += actual + expected + fmt.Sprintf("export const $canCase = Object.freeze({root: Object.freeze(%s), actual: %s, expected: $canExpected});\n", rootJSON, actualName)
	return Module{Path: path, Imports: imports, Body: body}, nil
}

// emitExecutableEntry builds the production entry module invoking the
// checked main region through the entry supervisor.
func emitExecutableEntry(assembly *programAssembly, runtime string) Module {
	main := assembly.program.Entry
	return Module{Path: "entry.ts", Imports: []ModuleImport{
		{Target: runtime + "/entry.ts", Names: []ImportName{{"runEntry", "$canRunEntry"}}},
		{Target: programStatePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{assembly.functions[main.Identity()], "$canMain"}}},
	}, Body: "process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));\n"}
}

package emit

import (
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// emitAuthoredModules lowers every checked function and native declaration
// into its authored output module, grouped by output path.
func emitAuthoredModules(assembly *programAssembly, runtime string) ([]Module, error) {
	program := assembly.program
	modules := []Module{}
	byPath := map[string][]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		path := fn.Symbol.Source.OutputPath
		byPath[path] = append(byPath[path], fn)
	}
	for _, path := range assembly.nativePaths {
		if _, ok := byPath[path]; !ok {
			byPath[path] = nil
		}
	}
	var paths []string
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		module, err := emitAuthoredModule(assembly, runtime, path, byPath[path])
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

// emitAuthoredModule lowers the functions and natives sharing one output path.
func emitAuthoredModule(assembly *programAssembly, runtime, path string, fns []*check.ProgramFunction) (Module, error) {
	program := assembly.program
	var body strings.Builder
	var regions []*ir.Region
	var descriptors []*ir.Expression
	for _, fn := range fns {
		regions = append(regions, fn.Region)
	}
	for _, native := range program.Natives {
		if native.Symbol.Source.OutputPath == path && (native.Question != nil || native.Judge != nil || native.Fetch != nil || native.LLM != nil || native.ArmDescription != nil || native.Wrapper != nil) {
			regions = append(regions, native.Regions...)
			if f := native.Fetch; f != nil {
				descriptors = append(descriptors, f.Path, f.Body)
				for _, entry := range f.Query {
					descriptors = append(descriptors, entry.Value)
				}
				for _, entry := range f.Headers {
					descriptors = append(descriptors, entry.Value)
				}
			}
			if l := native.LLM; l != nil {
				descriptors = append(descriptors, l.Instructions)
			}
			if q := native.Question; q != nil {
				descriptors = append(descriptors, q.Instructions, q.Minimum)
				for _, option := range q.Options {
					descriptors = append(descriptors, option.Description, option.Spread)
				}
			}
			if j := native.Judge; j != nil {
				for _, registration := range j.Registrations {
					descriptors = append(descriptors, registration.Arguments...)
					for _, step := range registration.Prepare {
						descriptors = append(descriptors, step.Value)
					}
				}
			}
		}
	}
	// Initializer references may name types absent from function signatures.
	allTypes := append(program.Model.Types(), checkedTypes(regions, descriptors)...)
	localTypes, err := NativeTypeDeclarations(allTypes)
	if err != nil {
		return Module{}, err
	}
	body.WriteString(localTypes)
	for _, fn := range fns {
		emitter := RegionEmitter{Bindings: assembly.bindings, Functions: assembly.functions, DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID}
		code, err := emitter.Function(assembly.functions[fn.Identity()], fn.Region)
		if err != nil {
			return Module{}, err
		}
		body.WriteString("export ")
		body.WriteString(code)
	}
	for _, native := range program.Natives {
		if native.Symbol.Source.OutputPath != path || native.Question == nil && native.Judge == nil && native.Fetch == nil && native.LLM == nil && native.ArmDescription == nil && native.Wrapper == nil {
			continue
		}
		for _, region := range native.Regions {
			emitter := RegionEmitter{Bindings: assembly.bindings, Functions: assembly.functions, RuleNames: assembly.nativeNames, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
			render := emitter.Function
			if native.Wrapper != nil {
				render = emitter.WrapperRule
			}
			code, err := render(assembly.nativeNames[region.ID], region)
			if err != nil {
				return Module{}, err
			}
			body.WriteString("export " + code)
		}
		emitter := RegionEmitter{Bindings: assembly.bindings, Functions: assembly.functions, RuleNames: assembly.nativeNames, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
		var code string
		var err error
		if native.Question != nil {
			code, err = emitter.QuestionPreparation(assembly.nativeNames[native.Symbol.ID], native.Question, assembly.nativeNames)
		} else if native.ArmDescription != nil {
			continue
		} else if native.Wrapper != nil {
			code, err = emitter.Wrapper(assembly.nativeNames[native.Symbol.ID], native, assembly.functions, assembly.nativeNames)
		} else if native.LLM != nil {
			policy := program.Connections[native.Connection]
			code, err = emitter.LLM(assembly.nativeNames[native.Symbol.ID], native.LLM, assembly.connectionNames[native.Connection], policy.Model, policy.MaxOutputTokens)
		} else if native.Fetch != nil {
			code, err = emitter.Fetch(assembly.nativeNames[native.Symbol.ID], native.Fetch, assembly.connectionNames[native.Connection])
		} else {
			policy := program.Connections[native.Connection]
			connectionName := assembly.connectionNames[native.Connection]
			code, err = emitter.Judge(assembly.nativeNames[native.Symbol.ID], native.Judge, assembly.questions, assembly.nativeNames, connectionName, policy.Model)
		}
		if err != nil {
			return Module{}, err
		}
		body.WriteString("export " + code)
	}
	return Module{Path: path, Imports: authoredModuleImports(assembly, runtime, path), Body: body.String()}, nil
}

// authoredModuleImports lists the runtime, state and cross-module imports
// for one authored output module.
func authoredModuleImports(assembly *programAssembly, runtime, path string) []ModuleImport {
	program := assembly.program
	values := stateValueImportNames()
	if assembly.browser {
		values = browserStateValueImportNames()
	}
	imports := append(programImports(runtime), ModuleImport{Target: programStatePath, Names: values})
	if !assembly.browser {
		imports = append(imports, ModuleImport{Target: runtime + "/platform/crypto/primitives.ts", Names: []ImportName{{"sha256", "$canSHA256"}}})
		imports = append(imports, ModuleImport{Target: runtime + "/ai/questions.ts", TypeOnly: true, Names: []ImportName{{"PreparedQuestion", "$canPreparedQuestion"}, {"Answer", "$canAnswer"}}})
	}
	if assembly.fetches {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{"$canFetch", "$canFetch"}}})
	}
	if assembly.llms {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{"$canResponses", "$canResponses"}}})
	}
	if assembly.judges {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{"$canAI", "$canAI"}}})
	}
	if assembly.program.ActionRoutes {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{"$canActionRoutes", "$canActionRoutes"}}})
	}
	if assembly.program.ActionClient {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{"$canActionClient", "$canActionClient"}}})
	}
	for _, id := range assembly.connectionIDs {
		name := assembly.connectionNames[id]
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{name, name}}})
	}
	for _, id := range assembly.nativeIDs {
		target := assembly.nativePaths[id]
		if target != path {
			imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{assembly.nativeNames[id], assembly.nativeNames[id]}}})
		}
	}
	for _, id := range assembly.collectionIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.collectionNames[id], assembly.collectionNames[id]}}})
	}
	for _, id := range assembly.browserStateIDs {
		imports = append(imports, ModuleImport{Target: programStatePath, Names: []ImportName{{assembly.browserStateNames[id], assembly.browserStateNames[id]}}})
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
		target := fn.Symbol.Source.OutputPath
		if target != path {
			imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{assembly.functions[fn.Identity()], assembly.functions[fn.Identity()]}}})
		}
	}
	return imports
}

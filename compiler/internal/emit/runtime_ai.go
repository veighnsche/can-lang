package emit

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// aiBindings wires native question/judge/fetch/LLM/wrapper declarations:
// per-symbol runtime names, checked dispatch targets, connection names and
// the flags selecting which AI factories the state module initializes.
func (assembly *programAssembly) aiBindings() bindingContribution {
	program := assembly.program
	functions := map[string]string{}
	assembly.nativeNames = map[string]string{}
	assembly.nativePaths = map[string]string{}
	assembly.questions = map[string]*ir.Question{}
	for i, native := range program.Natives {
		if native.Question == nil && native.Judge == nil && native.Fetch == nil && native.LLM == nil && native.ArmDescription == nil && native.Wrapper == nil {
			continue
		}
		name := fmt.Sprintf("$canNative%d", i)
		if native.ArmDescription == nil {
			assembly.nativeNames[native.Symbol.ID] = name
			assembly.nativePaths[native.Symbol.ID] = native.Symbol.Source.OutputPath
		}
		if native.Question != nil {
			assembly.questions[native.Symbol.ID] = native.Question
		}
		if native.LLM != nil {
			functions[native.Symbol.ID] = name
			assembly.llms = true
		}
		if native.Fetch != nil {
			functions[native.Symbol.ID] = name
			assembly.fetches = true
		}
		if native.Judge != nil {
			functions[native.Symbol.ID] = name
			assembly.judges = true
		}
		if native.Wrapper != nil {
			functions[native.Symbol.ID] = name
		}
		for j, region := range native.Regions {
			assembly.nativeNames[region.ID] = fmt.Sprintf("%sHandler%d", name, j)
			assembly.nativePaths[region.ID] = native.Symbol.Source.OutputPath
		}
	}
	connectionNames := map[string]string{}
	for _, native := range program.Natives {
		if native.Judge != nil || native.Fetch != nil || native.LLM != nil || native.Wrapper != nil {
			connectionNames[native.Connection] = ""
		}
	}
	assembly.connectionIDs = make([]string, 0, len(connectionNames))
	for id := range connectionNames {
		assembly.connectionIDs = append(assembly.connectionIDs, id)
	}
	sort.Strings(assembly.connectionIDs)
	assembly.connectionNames = map[string]string{}
	for i, id := range assembly.connectionIDs {
		assembly.connectionNames[id] = fmt.Sprintf("$canConnection%d", i)
	}
	assembly.nativeIDs = make([]string, 0, len(assembly.nativePaths))
	for id := range assembly.nativePaths {
		assembly.nativeIDs = append(assembly.nativeIDs, id)
	}
	sort.Strings(assembly.nativeIDs)
	assembly.armHandlers = map[string]string{}
	for _, native := range program.Natives {
		if native.ArmDescription != nil {
			assembly.armHandlers[native.Symbol.ID] = assembly.nativeNames[native.Regions[0].ID]
		}
	}
	return bindingContribution{domain: "ai", functions: functions}
}

// aiStateImports lists the native AI factories needed by this program. Only
// the selected transports are imported.
func (assembly *programAssembly) aiStateImports(runtime string) []ModuleImport {
	imports := []ModuleImport{}
	if assembly.judges {
		imports = append(imports, ModuleImport{Target: runtime + "/ai/typesafe.ts", Names: []ImportName{{"createTypeSafe", "$canCreateTypeSafe"}}})
	}
	if assembly.fetches {
		imports = append(imports, ModuleImport{Target: runtime + "/transport/named.ts", Names: []ImportName{{"createNamedFetch", "$canCreateNamedFetch"}}})
	}
	if assembly.llms {
		imports = append(imports, ModuleImport{Target: runtime + "/ai/responses.ts", Names: []ImportName{{"createResponses", "$canCreateResponses"}}})
	}
	return imports
}

// armDescriptionImports imports authored arm-description handlers into the
// shared state module.
func (assembly *programAssembly) armDescriptionImports() []ModuleImport {
	imports := []ModuleImport{}
	for _, native := range assembly.program.Natives {
		if native.ArmDescription != nil {
			name := assembly.armHandlers[native.Symbol.ID]
			imports = append(imports, ModuleImport{Target: native.Symbol.Source.OutputPath, Names: []ImportName{{name, name}}})
		}
	}
	return imports
}

// declareAIState emits the selected native AI factory bindings.
func (builder *stateBuilder) declareAIState() {
	if builder.assembly.fetches {
		builder.out.WriteString("export let $canFetch: ReturnType<typeof $canCreateNamedFetch>;\n")
	}
	if builder.assembly.llms {
		builder.out.WriteString("export let $canResponses: ReturnType<typeof $canCreateResponses>;\n")
	}
	if builder.assembly.judges {
		builder.out.WriteString("export let $canAI: ReturnType<typeof $canCreateTypeSafe>;\n")
	}
}

// initializeAIState constructs the selected native AI factories inside the
// shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeAIState() error {
	if builder.assembly.fetches {
		ids := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "failed": "can.std.http@1::request_failed"} {
			ids[name] = builder.numberIDs[declaration]
		}
		encoded, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder.out, "$canFetch=$canCreateNamedFetch($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	if builder.assembly.llms {
		ids := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "refused": "can.std.llm@1::refused", "truncated": "can.std.llm@1::truncated", "invalidResponse": "can.std.llm@1::invalid_response"} {
			ids[name] = builder.numberIDs[declaration]
		}
		encoded, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder.out, "$canResponses=$canCreateResponses($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	if builder.assembly.judges {
		ids := map[string]string{}
		for _, typ := range builder.assembly.program.Model.Types() {
			ids[typ.Declaration()] = typ.Identity()
		}
		fields := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "invalidQuestion": "can.std.ai@1::invalid_question", "invalidAnswer": "can.std.ai@1::invalid_answer", "failed": "can.std.http@1::request_failed"} {
			fields[name] = ids[declaration]
		}
		encoded, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder.out, "$canAI = $canCreateTypeSafe($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	return nil
}

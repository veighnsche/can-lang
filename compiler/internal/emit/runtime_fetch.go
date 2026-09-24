package emit

import (
	"fmt"
	"sort"
	"strings"
)

// fetchSpecializationBindings assigns the checked JSON fetch
// specializations of this program to their adapter targets. Contracts
// carry no per-type runtime data, so every get shares its operation
// target and every post shares its own; the frozen per-action contract
// splices per call site.
func (assembly *programAssembly) fetchSpecializationBindings() bindingContribution {
	functions := map[string]string{}
	ids := make([]string, 0, len(assembly.program.Fetches))
	for id := range assembly.program.Fetches {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		operation, _, _ := strings.Cut(id, "<")
		switch operation {
		case "can.std.http@1::fetch_json_get":
			functions[id] = "$canActionFetch.get"
		case "can.std.http@1::fetch_json_post":
			functions[id] = "$canActionFetch.post"
		}
	}
	return bindingContribution{domain: "fetch-specializations", functions: functions}
}

// fetchStateImports lists the JSON action adapter module the shared state
// module needs in both profiles.
func (assembly *programAssembly) fetchStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/action-json.ts", Names: []ImportName{{"createJsonActionFetch", "$canCreateActionFetch"}}},
	}
}

// fetchCatalogueValueImportNames lists the fetch adapter factory value
// authored and assertion modules import from the state module.
func fetchCatalogueValueImportNames() []ImportName {
	return []ImportName{{"$canActionFetch", "$canActionFetch"}}
}

// declareFetchState emits the JSON fetch adapter factory binding.
func (builder *stateBuilder) declareFetchState() {
	builder.out.WriteString("export let $canActionFetch:ReturnType<typeof $canCreateActionFetch>;\n")
}

// initializeFetchState constructs the JSON fetch adapter inside the
// shared initializer, after the domain runtime exists. The adapter maps
// native fetch outcomes into the fixed failure bound: cancelled
// transports, codec violations and unexpected statuses never surface as
// domain cases.
func (builder *stateBuilder) initializeFetchState() {
	fmt.Fprintf(&builder.out, "$canActionFetch=$canCreateActionFetch($canDomain,{transport:%s,invalidRequest:%s,bodyLimit:%s,statusError:%s,invalidData:%s,header:%s});\n", quote(builder.numberIDs["can.std.http@1::transport_failed"]), quote(builder.numberIDs["can.std.http@1::invalid_request"]), quote(builder.numberIDs["can.std.http@1::body_limit"]), quote(builder.numberIDs["can.std.http@1::status_error"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.http@1::header"]))
}

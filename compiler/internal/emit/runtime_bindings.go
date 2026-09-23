package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// bindingContribution is one domain's operation-identity to runtime-target
// mappings. Feature files construct their own contribution; this file only
// combines them in explicit deterministic order.
type bindingContribution struct {
	domain    string
	functions map[string]string
}

// combineBindingContributions merges domain mappings and rejects duplicate
// operation identities instead of letting one domain silently overwrite
// another. The caller fixes the order; contributions are never re-sorted.
func combineBindingContributions(contributions ...bindingContribution) (map[string]string, error) {
	combined := map[string]string{}
	owners := map[string]string{}
	for _, contribution := range contributions {
		for id, target := range contribution.functions {
			if owner, exists := owners[id]; exists {
				return nil, fmt.Errorf("duplicate operation binding %s from %s and %s", id, owner, contribution.domain)
			}
			owners[id] = contribution.domain
			combined[id] = target
		}
	}
	return combined, nil
}

// programAssembly holds every computed name/binding table shared by the state,
// authored-module and entry emitters. It carries no source semantics; the
// checker owns those.
type programAssembly struct {
	program   *check.Program
	functions map[string]string
	bindings  map[string]string

	httpIDs    []string
	httpNames  map[string]string
	sqlIDs     []string
	sqlNames   map[string]string
	txIDs      []string
	txNames    map[string]string
	codecIDs   []string
	codecNames map[string]string

	collectionIDs   []string
	collectionNames map[string]string
	collectionTypes map[string]*check.CollectionSpecialization

	nativeNames map[string]string
	nativePaths map[string]string
	questions   map[string]*ir.Question
	judges      bool
	fetches     bool
	llms        bool

	connectionIDs   []string
	connectionNames map[string]string
	nativeIDs       []string
	armHandlers     map[string]string
}

// assembleProgramBindings computes the full operation map and every name table
// in one deterministic pass. Domain order is fixed: core library operations,
// per-program specializations, collections, native AI wiring, then authored
// function identities.
func assembleProgramBindings(program *check.Program) (*programAssembly, error) {
	assembly := &programAssembly{program: program}
	contributions := []bindingContribution{
		coreOperationBindings(),
		sqlOperationBindings(),
		cryptoOperationBindings(),
		utilitiesOperationBindings(),
		assembly.specializationBindings(),
		assembly.sqlSpecializationBindings(),
		assembly.collectionBindings(),
		assembly.aiBindings(),
		fileOperationBindings(),
		processOperationBindings(),
		streamOperationBindings(),
		assembly.streamSpecializationBindings(),
		assembly.functionBindings(),
	}
	functions, err := combineBindingContributions(contributions...)
	if err != nil {
		return nil, err
	}
	assembly.functions = functions
	assembly.bindInitializers()
	return assembly, nil
}

// specializationBindings assigns deterministic runtime names to the checked
// HTTP/codec specializations of this program. SQL and transaction
// specializations bind through sqlSpecializationBindings in runtime_sql.go.
func (assembly *programAssembly) specializationBindings() bindingContribution {
	program := assembly.program
	functions := map[string]string{}

	assembly.httpIDs = make([]string, 0, len(program.HTTPs))
	for id := range program.HTTPs {
		assembly.httpIDs = append(assembly.httpIDs, id)
	}
	sort.Strings(assembly.httpIDs)
	assembly.httpNames = map[string]string{}
	for i, id := range assembly.httpIDs {
		name := fmt.Sprintf("$canHTTP%d", i)
		assembly.httpNames[id] = name
		method := "decode"
		if program.HTTPs[id].Operation == "can.std.http@1::response_json" {
			method = "encode"
		}
		functions[id] = name + "." + method
	}

	assembly.codecIDs = make([]string, 0, len(program.Codecs))
	for id := range program.Codecs {
		assembly.codecIDs = append(assembly.codecIDs, id)
	}
	sort.Strings(assembly.codecIDs)
	assembly.codecNames = map[string]string{}
	for i, id := range assembly.codecIDs {
		name := fmt.Sprintf("$canCodec%d", i)
		assembly.codecNames[id] = name
		method := "encode"
		if program.Codecs[id].Operation == "can.std.codec@1::decode_json" {
			method = "decode"
		}
		functions[id] = name + "." + method
	}

	return bindingContribution{domain: "specializations", functions: functions}
}

// functionBindings maps authored function identities to their emitted names.
func (assembly *programAssembly) functionBindings() bindingContribution {
	functions := map[string]string{}
	for i, fn := range assembly.program.Functions {
		functions[fn.Identity()] = fmt.Sprintf("$canFunction%d", i)
	}
	return bindingContribution{domain: "functions", functions: functions}
}

// bindInitializers exposes ordered top-level values through the shared
// value table.
func (assembly *programAssembly) bindInitializers() {
	assembly.bindings = map[string]string{}
	for _, value := range assembly.program.Initializers {
		assembly.bindings[value.Identity] = "($canValues[" + quote(value.Identity) + "] as " + TypeName(value.Type) + ")"
	}
}

// collectionMethodName maps a collection operation identity to its factory
// method. It is shared by binding computation only.
func collectionMethodName(operation string) string {
	method := strings.Split(operation, "::")[1]
	if method == "empty_map" || method == "empty_set" {
		method = "empty"
	}
	return method
}

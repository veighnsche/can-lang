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
// checker owns those. The browser flag selects the trimmed browser profile;
// the Bun profile is byte-identical with it unset.
type programAssembly struct {
	program   *check.Program
	functions map[string]string
	bindings  map[string]string
	browser   bool
	// authoredProof pairs each checked program.Functions identity with its
	// actual emitted binding. It is built from the same checked list and
	// resolved table as emission, separate from the mixed operation map,
	// and is shared read-only by this assembly's region emitters only.
	authoredProof map[string]string
	// collectionAsyncProof pairs each checked map/set specialization key
	// with its actual emitted receiver.method binding, but only for the
	// finite audited canonical operations with correct Entry-based
	// factory pairing. It is built after the final contribution merge
	// from checked program.Collections, never inferred from names or
	// types, and is shared read-only by assembly region emitters only.
	collectionAsyncProof map[string]string
	// coreAsyncProof pairs whitelisted canonical text/byte/check
	// identities with their actual emitted receiver.method bindings. It
	// is built after the final contribution merge from the finite exact
	// table, never inferred, and is shared read-only by assembly region
	// emitters only.
	coreAsyncProof map[string]string
	// integerWorkers proves concrete integer functions with synchronous
	// bigint companions. It is built after final binding resolution from
	// checked program functions, never inferred, and is shared read-only
	// by assembly region emitters only.
	integerWorkers map[string]*IntegerWorkerProof
	// mapLeaves proves concrete map-leaf functions with synchronous
	// Completion companions. It is built after final binding resolution
	// from checked program functions, never inferred, and is shared
	// read-only by assembly region emitters only.
	mapLeaves map[string]*MapLeafProof
	// mapBatches proves concrete batch-transition functions with
	// synchronous bigint value companions. It is built after final
	// binding resolution from checked program functions, never
	// inferred, and is shared read-only by assembly region emitters
	// only.
	mapBatches map[string]*MapBatchProof
	// closedRecoveries proves concrete closed-recovery functions with
	// a native boolean branch. It is built after final binding
	// resolution from checked program functions, never inferred, and
	// is shared read-only by assembly region emitters only.
	closedRecoveries map[string]*ClosedRecoveryProof

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

	browserStateIDs   []string
	browserStateNames map[string]string

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

	// reachedFunctions and reachedInitializers prune browser production to
	// the entry closure. They are nil for Bun, which keeps every checked
	// declaration. Specialization ID lists above are filtered in place for
	// browser after assembly.
	reachedFunctions    map[string]bool
	reachedInitializers map[string]bool

	// browserPairing binds one verified browser build into the emitted
	// server asset table. It is nil for unpaired builds and for the
	// browser profile itself.
	browserPairing *BrowserPairing
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
		testOperationBindings(),
		httpPeerOperationBindings(),
		nativeValuesOperationBindings(),
		streamOperationBindings(),
		assembly.streamSpecializationBindings(),
		websocketOperationBindings(),
		cookiesOperationBindings(),
		s3OperationBindings(),
		markdownOperationBindings(),
		formOperationBindings(),
		assembly.formSpecializationBindings(),
		assembly.fetchSpecializationBindings(),
		assembly.actionBindings(),
		browserOperationBindings(),
		assembly.browserSpecializationBindings(),
		assembly.functionBindings(),
	}
	functions, err := combineBindingContributions(contributions...)
	if err != nil {
		return nil, err
	}
	assembly.functions = functions
	assembly.authoredProof = assembly.checkedAuthoredProof()
	assembly.collectionAsyncProof = assembly.checkedCollectionProof()
	assembly.coreAsyncProof = assembly.checkedCoreProof()
	assembly.integerWorkers = assembly.checkedIntegerWorkers()
	assembly.mapLeaves = assembly.checkedMapLeaves()
	assembly.mapBatches = assembly.checkedMapBatches()
	assembly.closedRecoveries = assembly.checkedClosedRecoveries()
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
		switch program.Codecs[id].Operation {
		case "can.std.codec@1::decode_json":
			method = "decode"
		case "can.std.codec@1::decode_toml":
			method = "decodeToml"
		case "can.std.codec@1::decode_yaml":
			method = "decodeYaml"
		case "can.std.codec@1::decode_json5":
			method = "decodeJson5"
		case "can.std.codec@1::decode_jsonl":
			method = "decodeJsonl"
		case "can.std.codec@1::consume_jsonl":
			method = "consume"
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

// checkedAuthoredProof pairs every checked authored function identity,
// including concrete generic instances, with its actual emitted binding.
// Identities without a resolved binding are omitted so they can never
// qualify for proof-selected emission.
func (assembly *programAssembly) checkedAuthoredProof() map[string]string {
	proof := make(map[string]string, len(assembly.program.Functions))
	for _, fn := range assembly.program.Functions {
		identity := fn.Identity()
		if bound := assembly.functions[identity]; bound != "" {
			proof[identity] = bound
		}
	}
	return proof
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

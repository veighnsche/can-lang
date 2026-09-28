package emit

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/check"
)

// collectionBindings assigns deterministic factory names to the checked map
// and set specializations of this program.
func (assembly *programAssembly) collectionBindings() bindingContribution {
	assembly.collectionNames = map[string]string{}
	assembly.collectionTypes = map[string]*check.CollectionSpecialization{}
	assembly.collectionIDs = []string{}
	for _, special := range assembly.program.Collections {
		id := special.Collection.Identity()
		if assembly.collectionTypes[id] == nil {
			assembly.collectionIDs = append(assembly.collectionIDs, id)
			assembly.collectionTypes[id] = special
		}
	}
	sort.Strings(assembly.collectionIDs)
	for i, id := range assembly.collectionIDs {
		assembly.collectionNames[id] = fmt.Sprintf("$canCollection%d", i)
	}
	functions := map[string]string{}
	for id, special := range assembly.program.Collections {
		functions[id] = assembly.collectionNames[special.Collection.Identity()] + "." + collectionMethodName(special.Operation)
	}
	return bindingContribution{domain: "collections", functions: functions}
}

// collectionAsyncMethod is one audited canonical operation: the factory
// kind it requires ("map" or "set") and its native async method name.
type collectionAsyncMethod struct {
	kind   string
	method string
}

// collectionAsyncMethods is the finite frozen whitelist of native async
// collection operations. Map entries require Entry-based pairing
// (Entry != nil); set entries require Entry == nil. Anything outside
// this table never qualifies for context-bypass emission.
var collectionAsyncMethods = map[string]collectionAsyncMethod{
	"can.std.collections@1::empty_map":    {kind: "map", method: "empty"},
	"can.std.collections@1::build_map":    {kind: "map", method: "build_map"},
	"can.std.collections@1::get":          {kind: "map", method: "get"},
	"can.std.collections@1::insert":       {kind: "map", method: "insert"},
	"can.std.collections@1::replace":      {kind: "map", method: "replace"},
	"can.std.collections@1::remove":       {kind: "map", method: "remove"},
	"can.std.collections@1::entries":      {kind: "map", method: "entries"},
	"can.std.collections@1::empty_set":    {kind: "set", method: "empty"},
	"can.std.collections@1::build_set":    {kind: "set", method: "build_set"},
	"can.std.collections@1::contains":     {kind: "set", method: "contains"},
	"can.std.collections@1::add":          {kind: "set", method: "add"},
	"can.std.collections@1::union":        {kind: "set", method: "union"},
	"can.std.collections@1::intersection": {kind: "set", method: "intersection"},
	"can.std.collections@1::difference":   {kind: "set", method: "difference"},
}

// checkedCollectionProof pairs every checked map/set specialization key
// whose canonical operation, Entry-based factory kind and final merged
// target agree exactly with the actual collectionNames receiver and the
// audited native async method. Unknown operations, mismatched kinds,
// missing bindings and rebound targets are omitted so they can never
// qualify for proof-selected emission.
func (assembly *programAssembly) checkedCollectionProof() map[string]string {
	proof := make(map[string]string, len(assembly.program.Collections))
	for key, special := range assembly.program.Collections {
		if key == "" || special == nil || special.Collection == nil {
			continue
		}
		want, ok := collectionAsyncMethods[special.Operation]
		if !ok {
			continue
		}
		if (special.Entry != nil) != (want.kind == "map") {
			continue
		}
		receiver, ok := assembly.collectionNames[special.Collection.Identity()]
		if !ok || receiver == "" {
			continue
		}
		resolved, ok := assembly.functions[key]
		if !ok || resolved == "" || resolved != receiver+"."+want.method {
			continue
		}
		proof[key] = resolved
	}
	return proof
}

// collectionStateImports lists the runtime map/set factories the shared
// state module needs.
func (assembly *programAssembly) collectionStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/collections/map.ts", Names: []ImportName{{"createMap", "$canCreateMap"}, {"isMap", "$canIsMap"}}},
		{Target: runtime + "/collections/set.ts", Names: []ImportName{{"createSet", "$canCreateSet"}, {"isSet", "$canIsSet"}}},
	}
}

// declareCollectionState emits one factory binding per map/set
// specialization used by the program.
func (builder *stateBuilder) declareCollectionState() {
	for _, id := range builder.assembly.collectionIDs {
		special := builder.assembly.collectionTypes[id]
		args := special.Collection.Arguments()
		factory := "$canCreateSet<" + TypeName(args[0]) + ">"
		if special.Entry != nil {
			factory = "$canCreateMap<" + TypeName(args[0]) + "," + TypeName(args[1]) + ">"
		}
		fmt.Fprintf(&builder.out, "export let %s: ReturnType<typeof %s>;\n", builder.assembly.collectionNames[id], factory)
	}
}

// initializeCollectionState constructs the map/set factories inside the
// shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeCollectionState() {
	for _, id := range builder.assembly.collectionIDs {
		special := builder.assembly.collectionTypes[id]
		args := special.Collection.Arguments()
		if special.Entry != nil {
			fmt.Fprintf(&builder.out, "%s = $canCreateMap<%s,%s>($canDomain,{map:%s,entry:%s,absent:%s,exists:%s},%s);\n", builder.assembly.collectionNames[id], TypeName(args[0]), TypeName(args[1]), quote(id), quote(special.Entry.Identity()), quote(builder.numberIDs["can.std.collections@1::key_absent"]), quote(builder.numberIDs["can.std.collections@1::key_exists"]), quote(args[0].Declaration()))
		} else {
			fmt.Fprintf(&builder.out, "%s = $canCreateSet<%s>(%s,%s);\n", builder.assembly.collectionNames[id], TypeName(args[0]), quote(id), quote(args[0].Declaration()))
		}
	}
}

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

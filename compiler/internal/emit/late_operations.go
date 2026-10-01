package emit

import "fmt"

// lateOperationBindings maps the NT-I03 late-occurrence operations to
// the single $canLate contribution: owner-admitted calls over one
// adapter-held K06 service with verbatim late_fault mapping. The
// adapter shares the $canTest.owner table, wired once in the shared
// state module; every method is async and takes the trailing call
// context. The catalogue arm_gate rename (gate is the participant
// gate noun) maps to the adapter armGate service call.
func lateOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.late@1::select":           "$canLate.select",
		"can.std.late@1::enroll":           "$canLate.enroll",
		"can.std.late@1::arm_gate":         "$canLate.armGate",
		"can.std.late@1::emit":             "$canLate.emit",
		"can.std.late@1::witness_terminal": "$canLate.witnessTerminal",
		"can.std.late@1::observe":          "$canLate.observe",
		"can.std.late@1::grant_lease":      "$canLate.grantLease",
		"can.std.late@1::observe_lease":    "$canLate.observeLease",
		"can.std.late@1::release_lease":    "$canLate.releaseLease",
		"can.std.late@1::reconcile":        "$canLate.reconcile",
		"can.std.late@1::kill_worker":      "$canLate.killWorker",
		"can.std.late@1::read_outcome":     "$canLate.readOutcome",
		"can.std.late@1::read_counters":    "$canLate.readCounters",
	}
	return bindingContribution{domain: "late", functions: functions}
}

// lateStateImports lists the late-occurrence adapter factory module the
// shared state module needs.
func (assembly *programAssembly) lateStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i03/late.ts", Names: []ImportName{{"createLateOccurrence", "$canCreateLateOccurrence"}}},
	}
}

// lateStateValueImportNames lists the late-occurrence factory value
// authored and assertion modules import from the state module.
func lateStateValueImportNames() []ImportName {
	return []ImportName{{"$canLate", "$canLate"}}
}

// declareLateState emits the late-occurrence factory binding.
func (builder *stateBuilder) declareLateState() {
	builder.out.WriteString("export let $canLate:ReturnType<typeof $canCreateLateOccurrence>;\n")
}

// initializeLateState constructs the late-occurrence adapter inside the
// shared initializer, after the test factory exists: every op takes the
// owner first and the adapter admits the grant before any compute, so a
// foreign or released owner fails stale_handle without touching the
// service. Result records reuse the nominal identities the catalogue
// fact types reference.
func (builder *stateBuilder) initializeLateState() {
	fmt.Fprintf(&builder.out, "$canLate=$canCreateLateOccurrence($canDomain,{staleHandle:%s,lateFault:%s,some:%s,none:%s,selectFacts:%s,enrollFacts:%s,gateFacts:%s,eventFacts:%s,terminalFacts:%s,observationFacts:%s,leaseFacts:%s,releaseFacts:%s,reconcileFacts:%s,outcomeFacts:%s,counters:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.late@1::late_fault"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.late@1::select_facts"]),
		quote(builder.numberIDs["can.std.late@1::enroll_facts"]),
		quote(builder.numberIDs["can.std.late@1::gate_facts"]),
		quote(builder.numberIDs["can.std.late@1::event_facts"]),
		quote(builder.numberIDs["can.std.late@1::terminal_facts"]),
		quote(builder.numberIDs["can.std.late@1::observation_facts"]),
		quote(builder.numberIDs["can.std.late@1::lease_facts"]),
		quote(builder.numberIDs["can.std.late@1::release_facts"]),
		quote(builder.numberIDs["can.std.late@1::reconcile_facts"]),
		quote(builder.numberIDs["can.std.late@1::outcome_facts"]),
		quote(builder.numberIDs["can.std.late@1::counters"]))
}

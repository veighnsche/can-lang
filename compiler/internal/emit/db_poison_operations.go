package emit

import "fmt"

// dbPoisonOperationBindings maps the NT-I15 poison operations to
// the single $canDbPoison contribution: owner-admitted calls over
// one adapter-held K25 service with verbatim db_fault mapping.
// The adapter shares the $canTest.owner table, wired once in the
// shared state module; every method is async and takes the
// trailing call context. The catalogue snake_case names map to
// the adapter camelCase calls.
func dbPoisonOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.db@1::begin_poison_attempt":         "$canDbPoison.beginPoisonAttempt",
		"can.std.db@1::begin_control_attempt":        "$canDbPoison.beginControlAttempt",
		"can.std.db@1::attempt_record":               "$canDbPoison.attemptRecord",
		"can.std.db@1::record_poison_error":          "$canDbPoison.recordPoisonError",
		"can.std.db@1::error_facts":                  "$canDbPoison.errorFacts",
		"can.std.db@1::record_callback_report":       "$canDbPoison.recordCallbackReport",
		"can.std.db@1::callback_facts":               "$canDbPoison.callbackFacts",
		"can.std.db@1::record_poison_settlement":     "$canDbPoison.recordPoisonSettlement",
		"can.std.db@1::poison_settlement_facts":      "$canDbPoison.poisonSettlementFacts",
		"can.std.db@1::record_fresh_read":            "$canDbPoison.recordFreshRead",
		"can.std.db@1::fresh_facts":                  "$canDbPoison.freshFacts",
		"can.std.db@1::record_replay_read":           "$canDbPoison.recordReplayRead",
		"can.std.db@1::replay_facts":                 "$canDbPoison.replayFacts",
		"can.std.db@1::compare_fresh_to_replay":      "$canDbPoison.compareFreshToReplay",
		"can.std.db@1::sentinel_present_in_fresh":    "$canDbPoison.sentinelPresentInFresh",
		"can.std.db@1::sentinel_present_in_replay":   "$canDbPoison.sentinelPresentInReplay",
		"can.std.db@1::settlement_agrees_with_reads": "$canDbPoison.settlementAgreesWithReads",
	}
	return bindingContribution{domain: "dbpoison", functions: functions}
}

// dbPoisonStateImports lists the poison-observer adapter factory
// module the shared state module needs.
func (assembly *programAssembly) dbPoisonStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i15/db_poison.ts", Names: []ImportName{{"createDbPoison", "$canCreateDbPoison"}}},
	}
}

// dbPoisonStateValueImportNames lists the poison-observer factory
// value authored and assertion modules import from the state
// module.
func dbPoisonStateValueImportNames() []ImportName {
	return []ImportName{{"$canDbPoison", "$canDbPoison"}}
}

// declareDbPoisonState emits the poison-observer factory binding.
func (builder *stateBuilder) declareDbPoisonState() {
	builder.out.WriteString("export let $canDbPoison:ReturnType<typeof $canCreateDbPoison>;\n")
}

// initializeDbPoisonState constructs the poison-observer adapter
// inside the shared initializer, after the test factory exists:
// every op takes the owner first and the adapter admits the grant
// before any compute, so a foreign or released owner fails
// stale_handle without touching the service. Result records
// reuse the nominal identities the catalogue fact types
// reference.
func (builder *stateBuilder) initializeDbPoisonState() {
	fmt.Fprintf(&builder.out, "$canDbPoison=$canCreateDbPoison($canDomain,{staleHandle:%s,dbFault:%s,numberCell:%s,textCell:%s,bytesCell:%s,nullCell:%s,optionSome:%s,optionNone:%s,poisonAttemptRecord:%s,poisonErrorFacts:%s,poisonCallbackFacts:%s,poisonSettlementRecord:%s,sentinelRowFacts:%s,freshSentinelFacts:%s,replaySentinelFacts:%s,sentinelComparison:%s,settlementAgreement:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.db@1::db_fault"]),
		quote(builder.numberIDs["can.std.db@1::number_cell"]),
		quote(builder.numberIDs["can.std.db@1::text_cell"]),
		quote(builder.numberIDs["can.std.db@1::bytes_cell"]),
		quote(builder.numberIDs["can.std.db@1::null_cell"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.db@1::poison_attempt_record"]),
		quote(builder.numberIDs["can.std.db@1::poison_error_facts"]),
		quote(builder.numberIDs["can.std.db@1::poison_callback_facts"]),
		quote(builder.numberIDs["can.std.db@1::poison_settlement_record"]),
		quote(builder.numberIDs["can.std.db@1::sentinel_row_facts"]),
		quote(builder.numberIDs["can.std.db@1::fresh_sentinel_facts"]),
		quote(builder.numberIDs["can.std.db@1::replay_sentinel_facts"]),
		quote(builder.numberIDs["can.std.db@1::sentinel_comparison"]),
		quote(builder.numberIDs["can.std.db@1::settlement_agreement"]))
}

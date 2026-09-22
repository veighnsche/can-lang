package emit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func programImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/collections/array.ts", Names: arrayImports()},
		{Target: runtime + "/assert/context.ts", Names: []ImportName{{"callContext", "$canCallContext"}, {"scopeRequest", "$canScopeRequest"}}},
		{Target: runtime + "/coordination.ts", Names: []ImportName{{"settle", "$canCoordinateSettle"}, {"handle", "$canCoordinateHandle"}, {"aggregate", "$canCoordinateAggregate"}}},
		{Target: runtime + "/owner.ts", TypeOnly: true, Names: []ImportName{{"Participant", "$canParticipant"}}},
		{Target: runtime + "/bytes.ts", Names: []ImportName{{"byteLength", "$canByteLength"}}},
		{Target: runtime + "/callable.ts", Names: []ImportName{{"ownCallable", "$canOwnCallable"}, {"callableInstance", "$canCallableInstance"}}},
		{Target: runtime + "/assert/fixtures.ts", Names: []ImportName{{"withFixture", "$canWithFixture"}}},
		{Target: runtime + "/completion.ts", Names: []ImportName{{"success", "$canSuccess"}, {"failure", "$canFailure"}, {"value", "$canValue"}, {"invoke", "$canInvoke"}, {"caught", "$canCaught"}, {"errorType", "$canErrorType"}, {"errorPayload", "$canErrorPayload"}}},
		{Target: runtime + "/completion.ts", TypeOnly: true, Names: []ImportName{{"Completion", "$canCompletion"}, {"AssertionContext", "$canAssertionContext"}}},
		{Target: runtime + "/data.ts", Names: []ImportName{{"record", "$canRecord"}, {"update", "$canUpdate"}, {"array", "$canArray"}, {"recordIdentity", "$canRecordIdentity"}}},
		{Target: runtime + "/primitive.ts", Names: []ImportName{{"intDivide", "$canIntDivide"}, {"intRemainder", "$canIntRemainder"}, {"intPower", "$canIntPower"}, {"index", "$canIndex"}, {"slice", "$canSlice"}}},
		{Target: runtime + "/failure.ts", Names: []ImportName{{"captureStandard", "$canCaptureStandard"}, {"isStandardFailure", "$canIsStandardFailure"}, {"standardFailureKind", "$canFailureKind"}, {"standardFailureMessage", "$canFailureMessage"}, {"standardFailureOccurrenceID", "$canFailureOccurrenceID"}}},
	}
}

// ProgramModules keeps every source module inert. One shared initialization
// function creates error evidence and evaluates the ordered top-level values;
// the entry supervisor invokes it before calling the checked main region.
func ProgramModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, false)
}
func AssertionModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, true)
}
func programModules(program *check.Program, runtime string, dependencies []ir.Artifact, assertions bool) ([]ir.Artifact, error) {
	if program == nil || (!assertions && (program.Entry == nil || program.Entry.Region == nil)) {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	const statePath = "program/state.ts"
	functions := map[string]string{
		"can.std.bytes@1::from_utf8": "$canBytes.fromUTF8",
		"can.std.bytes@1::to_utf8":   "$canBytes.toUTF8",
		"can.std.bytes@1::from_ints": "$canBytes.fromInts",
		"can.std.bytes@1::to_ints":   "$canBytes.toInts",
		"can.std.bytes@1::empty":     "$canBytes.empty",
		"can.std.io@1::stdout_write": "$canCLI.stdoutWrite",
		"can.std.io@1::stderr_write": "$canCLI.stderrWrite",
		"can.std.io@1::stdin_bytes":  "$canIO.stdinBytes",
		"can.std.io@1::stdin_text":   "$canIO.stdinText",
		"can.std.env@1::required":    "$canEnv.required",
		"can.std.env@1::optional":    "$canEnv.optional",
	}
	functions["can.std.html@1::make_tag"] = "$canHTML.makeTag"
	functions["can.std.html@1::text"] = "$canHTML.text"
	functions["can.std.html@1::text_fragment"] = "$canHTML.textFragment"
	functions["can.std.html@1::parse_url"] = "$canHTML.parseURL"
	functions["can.std.html@1::text_attribute"] = "$canHTML.textAttribute"
	functions["can.std.html@1::url_attribute"] = "$canHTML.urlAttribute"
	functions["can.std.html@1::element"] = "$canHTML.element"
	functions["can.std.html@1::fragment"] = "$canHTML.fragment"
	functions["can.std.html@1::stylesheet"] = "$canHTML.stylesheet"
	functions["can.std.html@1::meta_viewport"] = "$canHTML.metaViewport"
	functions["can.std.html@1::document"] = "$canHTML.document"
	functions["can.std.htmx@1::get"] = "$canHTML.get"
	functions["can.std.htmx@1::post"] = "$canHTML.post"
	functions["can.std.htmx@1::target_id"] = "$canHTML.targetID"
	functions["can.std.htmx@1::target_attribute"] = "$canHTML.targetAttribute"
	functions["can.std.htmx@1::indicator_id"] = "$canHTML.indicatorID"
	functions["can.std.htmx@1::swap_inner"] = "$canHTML.swapInner"
	functions["can.std.htmx@1::swap_outer"] = "$canHTML.swapOuter"
	functions["can.std.htmx@1::trigger_change"] = "$canHTML.triggerChange"
	functions["can.std.htmx@1::trigger_input_changed"] = "$canHTML.triggerInputChanged"
	functions["can.std.htmx@1::trigger_every"] = "$canHTML.triggerEvery"
	functions["can.std.htmx@1::disable_this"] = "$canHTML.disableThis"
	functions["can.std.htmx@1::runtime_head"] = "$canHTML.runtimeHead"
	functions["can.std.http@1::request_method"] = "$canHTTPRequests.method"
	functions["can.std.http@1::request_path"] = "$canHTTPRequests.path"
	functions["can.std.http@1::request_headers"] = "$canHTTPRequests.headers"
	functions["can.std.http@1::query_one"] = "$canHTTPRequests.queryOne"
	functions["can.std.http@1::query_all"] = "$canHTTPRequests.queryAll"
	functions["can.std.http@1::request_body"] = "$canHTTPRequests.body"
	functions["can.std.http@1::make_status"] = "$canHTTPResponses.makeStatus"
	functions["can.std.http@1::make_body_status"] = "$canHTTPResponses.makeBodyStatus"
	functions["can.std.http@1::status_ok"] = "$canHTTPResponses.ok"
	functions["can.std.http@1::status_unprocessable"] = "$canHTTPResponses.unprocessable"
	functions["can.std.http@1::status_internal"] = "$canHTTPResponses.internal"
	functions["can.std.http@1::status_unavailable"] = "$canHTTPResponses.unavailable"
	functions["can.std.http@1::make_server_headers"] = "$canHTTPResponses.makeHeaders"
	functions["can.std.http@1::empty_server_headers"] = "$canHTTPResponses.emptyHeaders"
	functions["can.std.http@1::response_empty"] = "$canHTTPResponses.empty"
	functions["can.std.http@1::response_bytes"] = "$canHTTPResponses.bytes"
	functions["can.std.http@1::response_text"] = "$canHTTPResponses.text"
	functions["can.std.http@1::response_html"] = "$canHTTPResponses.html"
	functions["can.std.http@1::route_get"] = "$canRouter.get"
	functions["can.std.http@1::route_post"] = "$canRouter.post"
	functions["can.std.http@1::make_router"] = "$canRouter.make"
	functions["can.std.http@1::make_server_config"] = "$canServer.makeConfig"
	functions["can.std.http@1::server_start"] = "$canServer.start"
	functions["can.std.http@1::server_stop"] = "$canServer.stop"
	functions["can.std.http@1::server_wait"] = "$canServer.wait"
	functions["can.std.sql@1::pool_open"] = "$canSQLPools.open"
	functions["can.std.sql@1::pool_close"] = "$canSQLPools.close"
	functions["can.std.clock@1::wall_millis"] = "$canClock.wallMillis"
	functions["can.std.clock@1::monotonic_millis"] = "$canClock.monotonicMillis"
	functions["can.std.clock@1::sleep_millis"] = "$canClock.sleepMillis"
	functions["can.std.random@1::secure_bytes"] = "$canRandom.secureBytes"
	functions["can.std.random@1::uuid_v4"] = "$canRandom.uuidV4"
	functions["can.std.crypto@1::sha256"] = "$canSHA256"
	functions["can.std.log@1::write_info"] = "$canLog.writeInfo"
	functions["can.std.log@1::write_error"] = "$canLog.writeError"
	functions["can.intrinsic.str@1::includes"] = "$canText.includes"
	functions["can.intrinsic.str@1::starts_with"] = "$canText.startsWith"
	functions["can.intrinsic.str@1::ends_with"] = "$canText.endsWith"
	functions["can.intrinsic.str@1::to_lower_case"] = "$canText.toLowerCase"
	functions["can.intrinsic.str@1::to_upper_case"] = "$canText.toUpperCase"
	functions["can.intrinsic.str@1::trim"] = "$canText.trim"
	functions["can.intrinsic.str@1::slice"] = "$canText.slice"
	functions["can.intrinsic.str@1::split"] = "$canText.split"
	functions["can.intrinsic.str@1::replace_all"] = "$canText.replaceAll"
	functions["can.std.text@1::join"] = "$canText.join"
	functions["can.std.text@1::scalars"] = "$canText.scalars"
	functions["can.std.text@1::from_scalars"] = "$canText.fromScalars"
	functions["can.std.text@1::graphemes"] = "$canText.graphemes"
	functions["can.std.text@1::normalize_nfc"] = "$canText.normalizeNFC"
	functions["can.std.number@1::divmod"] = "$canAmounts.divmod"
	functions["can.std.number@1::euclidean_divmod"] = "$canAmounts.euclideanDivmod"
	functions["can.std.number@1::round_ratio_half_even"] = "$canAmounts.roundRatioHalfEven"
	functions["can.std.text@1::from_int"] = "$canNumbers.fromInt"
	functions["can.std.text@1::from_float"] = "$canNumbers.fromFloat"
	functions["can.std.text@1::from_bool"] = "$canNumbers.fromBool"
	functions["can.std.text@1::to_int"] = "$canNumbers.toInt"
	functions["can.std.text@1::to_float"] = "$canNumbers.toFloat"
	functions["can.std.text@1::to_bool"] = "$canNumbers.toBool"
	functions["can.std.number@1::int_to_float"] = "$canNumbers.intToFloat"
	functions["can.std.number@1::float_to_int"] = "$canNumbers.floatToInt"
	functions["can.std.number@1::bool_to_int"] = "$canNumbers.boolToInt"
	functions["can.std.number@1::int_to_bool"] = "$canNumbers.intToBool"
	functions["can.std.number@1::floor"] = "$canNumbers.floor"
	functions["can.std.number@1::ceil"] = "$canNumbers.ceil"
	functions["can.std.number@1::trunc"] = "$canNumbers.trunc"
	functions["can.std.number@1::round"] = "$canNumbers.round"
	functions["can.std.number@1::is_finite"] = "$canNumbers.isFinite"
	functions["can.std.number@1::is_nan"] = "$canNumbers.isNaN"
	httpIDs := make([]string, 0, len(program.HTTPs))
	for id := range program.HTTPs {
		httpIDs = append(httpIDs, id)
	}
	sort.Strings(httpIDs)
	httpNames := map[string]string{}
	for i, id := range httpIDs {
		name := fmt.Sprintf("$canHTTP%d", i)
		httpNames[id] = name
		method := "decode"
		if program.HTTPs[id].Operation == "can.std.http@1::response_json" {
			method = "encode"
		}
		functions[id] = name + "." + method
	}
	sqlIDs := make([]string, 0, len(program.SQLs))
	for id := range program.SQLs {
		sqlIDs = append(sqlIDs, id)
	}
	sort.Strings(sqlIDs)
	sqlNames := map[string]string{}
	for i, id := range sqlIDs {
		sqlNames[id] = fmt.Sprintf("$canSQLQuery%d", i)
		functions[id] = sqlNames[id] + ".run"
	}
	txIDs := make([]string, 0, len(program.Transactions))
	for id := range program.Transactions {
		txIDs = append(txIDs, id)
	}
	sort.Strings(txIDs)
	txNames := map[string]string{}
	for i, id := range txIDs {
		txNames[id] = fmt.Sprintf("$canSQLTransaction%d", i)
		functions[id] = txNames[id] + ".run"
	}
	codecIDs := make([]string, 0, len(program.Codecs))
	for id := range program.Codecs {
		codecIDs = append(codecIDs, id)
	}
	sort.Strings(codecIDs)
	codecNames := map[string]string{}
	for i, id := range codecIDs {
		name := fmt.Sprintf("$canCodec%d", i)
		codecNames[id] = name
		method := "encode"
		if program.Codecs[id].Operation == "can.std.codec@1::decode_json" {
			method = "decode"
		}
		functions[id] = name + "." + method
	}

	collectionNames := map[string]string{}
	collectionTypes := map[string]*check.CollectionSpecialization{}
	collectionIDs := []string{}
	for _, special := range program.Collections {
		id := special.Collection.Identity()
		if collectionTypes[id] == nil {
			collectionIDs = append(collectionIDs, id)
			collectionTypes[id] = special
		}
	}
	sort.Strings(collectionIDs)
	for i, id := range collectionIDs {
		collectionNames[id] = fmt.Sprintf("$canCollection%d", i)
	}
	for id, special := range program.Collections {
		method := strings.Split(special.Operation, "::")[1]
		if method == "empty_map" || method == "empty_set" {
			method = "empty"
		}
		functions[id] = collectionNames[special.Collection.Identity()] + "." + method
	}
	bindings := map[string]string{}
	nativeNames := map[string]string{}
	nativePaths := map[string]string{}
	questions := map[string]*ir.Question{}
	judges := false
	fetches := false
	llms := false
	for i, native := range program.Natives {
		if native.Question == nil && native.Judge == nil && native.Fetch == nil && native.LLM == nil && native.ArmDescription == nil {
			continue
		}
		name := fmt.Sprintf("$canNative%d", i)
		if native.ArmDescription == nil {
			nativeNames[native.Symbol.ID] = name
			nativePaths[native.Symbol.ID] = native.Symbol.Source.OutputPath
		}
		if native.Question != nil {
			questions[native.Symbol.ID] = native.Question
		}
		if native.LLM != nil {
			functions[native.Symbol.ID] = name
			llms = true
		}
		if native.Fetch != nil {
			functions[native.Symbol.ID] = name
			fetches = true
		}
		if native.Judge != nil {
			functions[native.Symbol.ID] = name
			judges = true
		}
		for j, region := range native.Regions {
			nativeNames[region.ID] = fmt.Sprintf("%sHandler%d", name, j)
			nativePaths[region.ID] = native.Symbol.Source.OutputPath
		}
	}
	connectionNames := map[string]string{}
	for _, native := range program.Natives {
		if native.Judge != nil || native.Fetch != nil || native.LLM != nil {
			connectionNames[native.Connection] = ""
		}
	}
	connectionIDs := make([]string, 0, len(connectionNames))
	for id := range connectionNames {
		connectionIDs = append(connectionIDs, id)
	}
	sort.Strings(connectionIDs)
	for i, id := range connectionIDs {
		connectionNames[id] = fmt.Sprintf("$canConnection%d", i)
	}
	nativeIDs := make([]string, 0, len(nativePaths))
	for id := range nativePaths {
		nativeIDs = append(nativeIDs, id)
	}
	sort.Strings(nativeIDs)
	for i, fn := range program.Functions {
		functions[fn.Identity()] = fmt.Sprintf("$canFunction%d", i)
	}
	for _, value := range program.Initializers {
		bindings[value.Identity] = "($canValues[" + quote(value.Identity) + "] as " + TypeName(value.Type) + ")"
	}
	var errorTypes []*types.Type
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Error {
			errorTypes = append(errorTypes, typ)
		}
	}
	bound, err := program.Registry.Bound(errorTypes)
	if err != nil {
		return nil, err
	}
	plan, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		return nil, err
	}
	armHandlers := map[string]string{}
	for _, native := range program.Natives {
		if native.ArmDescription != nil {
			armHandlers[native.Symbol.ID] = nativeNames[native.Regions[0].ID]
		}
	}
	initial, err := initialization(program.Initializers, nil, armHandlers, true)
	if err != nil {
		return nil, err
	}
	declarations, err := NativeTypeDeclarations(program.Model.Types())
	if err != nil {
		return nil, err
	}
	optionResult := program.Intrinsics["can.std.env@1::optional"].Result()
	optionType := TypeName(optionResult)
	optionIDs := map[string]string{}
	for _, leaf := range optionResult.Leaves() {
		optionIDs[leaf.Declaration()] = leaf.Identity()
	}
	headerType := ""
	for _, typ := range program.Model.Types() {
		if typ.Declaration() == "can.std.http@1::header" {
			headerType = TypeName(typ)
		}
	}
	if headerType == "" {
		return nil, fmt.Errorf("HTTP header type is not in the checked model")
	}
	var state strings.Builder
	state.WriteString(declarations)
	state.WriteString("export let $canHTML:ReturnType<typeof $canCreateHTML>;\nexport let $canSQL:ReturnType<typeof $canCreateSQLDescriptors>;\nexport let $canSQLPools:ReturnType<typeof $canCreateSQLPools>;\nexport let $canTransactions:ReturnType<typeof $canCreateSQLTransactions>;\n")
	fmt.Fprintf(&state, "export let $canHTTPRequests: ReturnType<typeof $canCreateRequests<%s>>;\nexport let $canHTTPResponses: ReturnType<typeof $canCreateHTTPResponses>;\nexport let $canRouter: ReturnType<typeof $canCreateRouter>;\nexport let $canServer: ReturnType<typeof $canCreateServer>;\n", headerType)
	state.WriteString("export let $canClock:ReturnType<typeof $canCreateClock>;\nexport let $canRandom:ReturnType<typeof $canCreateRandom>;\nexport let $canLog:ReturnType<typeof $canCreateLog>;\n")
	fmt.Fprintf(&state, "export let $canIO: ReturnType<typeof $canCreateIO>;\nexport let $canEnv: ReturnType<typeof $canCreateEnv<%s>>;\n", optionType)
	for _, id := range connectionIDs {
		policy := program.Connections[id]
		headers := make([]map[string]string, 0, len(policy.Headers))
		for _, header := range policy.Headers {
			// Checked policy uses wire names; transport entries use authored identifiers.
			headers = append(headers, map[string]string{"name": strings.ReplaceAll(header.Name, "-", "_"), "value": header.Value})
		}
		connection := map[string]any{"endpoint": policy.Endpoint, "timeoutMilliseconds": policy.TimeoutMilliseconds, "maxBodyBytes": policy.MaxBodyBytes, "headers": headers}
		if policy.BearerEnvironment != "" {
			connection["bearerEnvironment"] = policy.BearerEnvironment
		}
		encoded, e := json.Marshal(connection)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "export const %s = Object.freeze(%s);\n", connectionNames[id], encoded)
	}

	if fetches {
		state.WriteString("export let $canFetch: ReturnType<typeof $canCreateNamedFetch>;\n")
	}
	if llms {
		state.WriteString("export let $canResponses: ReturnType<typeof $canCreateResponses>;\n")
	}
	if judges {
		state.WriteString("export let $canAI: ReturnType<typeof $canCreateTypeSafe>;\n")
	}
	for _, id := range codecIDs {
		fmt.Fprintf(&state, "export let %s: ReturnType<typeof $canCreateCodec<%s>>;\n", codecNames[id], TypeName(program.Codecs[id].Data))
	}
	for _, id := range collectionIDs {
		special := collectionTypes[id]
		args := special.Collection.Arguments()
		factory := "$canCreateSet<" + TypeName(args[0]) + ">"
		if special.Entry != nil {
			factory = "$canCreateMap<" + TypeName(args[0]) + "," + TypeName(args[1]) + ">"
		}
		fmt.Fprintf(&state, "export let %s: ReturnType<typeof %s>;\n", collectionNames[id], factory)
	}
	state.WriteString("export let $canText: ReturnType<typeof $canCreateText>;\nexport let $canAmounts: ReturnType<typeof $canCreateExactAmounts>;\nexport let $canNumbers: ReturnType<typeof $canCreateNumbers>;\nexport let $canBytes: ReturnType<typeof $canCreateBytes>;\nexport let $canCLI: ReturnType<typeof $canCreateCLI>;\nexport let $canDomain: ReturnType<typeof $canCreateDomain>;\nexport const $canValues: Record<string, unknown> = Object.create(null);\nexport function $canInitialize(): void {\n")
	var invalidData, writeFailed, bytesID string
	for _, typ := range program.Model.Types() {
		switch typ.Declaration() {
		case "can.std.codec@1::invalid_data":
			invalidData = typ.Identity()
		case "can.std.io@1::write_failed":
			writeFailed = typ.Identity()
		case "can.std.bytes@1::buffer":
			bytesID = typ.Identity()
		}
	}
	htmlKinds := map[string]string{}
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Opaque && (strings.HasPrefix(typ.Declaration(), "can.std.html@1::") || strings.HasPrefix(typ.Declaration(), "can.std.htmx@1::")) {
			htmlKinds[typ.Identity()] = strings.Split(typ.Declaration(), "::")[1]
		}
	}
	htmlKindsJSON, err := json.Marshal(htmlKinds)
	if err != nil {
		return nil, err
	}
	httpKinds := map[string]string{}
	for _, typ := range program.Model.Types() {
		if typ.Kind() != types.Opaque || !strings.HasPrefix(typ.Declaration(), "can.std.http@1::") {
			continue
		}
		switch kind := strings.Split(typ.Declaration(), "::")[1]; kind {
		case "request", "status", "body_status", "server_headers", "server_response", "route", "router", "server_config", "server":
			httpKinds[typ.Identity()] = kind
		}
	}
	httpKindsJSON, err := json.Marshal(httpKinds)
	if err != nil {
		return nil, err
	}
	sqlKinds := map[string]string{}
	for _, typ := range program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.sql@1::pool":
			sqlKinds[typ.Identity()] = "pool"
		case "can.std.sql@1::transaction":
			sqlKinds[typ.Identity()] = "transaction"
		}
	}
	sqlKindsJSON, err := json.Marshal(sqlKinds)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&state, "const $canHTMLKinds:Readonly<Record<string,string>>=%s;\n", htmlKindsJSON)
	fmt.Fprintf(&state, "const $canHTTPKinds:Readonly<Record<string,string>>=%s;\n", httpKindsJSON)
	fmt.Fprintf(&state, "const $canSQLKinds:Readonly<Record<string,string>>=%s;\n", sqlKindsJSON)
	fmt.Fprintf(&state, "$canDomain = $canCreateDomain(%s, (identity, value) => (identity === %s && $canIsBytes(value)) || $canIsMap(identity,value) || $canIsSet(identity,value) || $canIsHTML($canHTMLKinds[identity],value) || $canIsHTTP($canHTTPKinds[identity],value) || $canIsRouter($canHTTPKinds[identity],value) || $canIsServer($canHTTPKinds[identity],value) || $canIsSQLPool($canSQLKinds[identity],value) || $canIsSQLTransaction($canSQLKinds[identity],value));\n", plan, quote(bytesID))
	fmt.Fprintf(&state, "$canBytes = $canCreateBytes($canDomain, %s);\n$canCLI = $canCreateCLI($canDomain, {writeFailed: %s});\n", quote(invalidData), quote(writeFailed))
	numberIDs := map[string]string{}
	for _, typ := range program.Model.Types() {
		numberIDs[typ.Declaration()] = typ.Identity()
	}
	assetTable, assetURLs, assetFiles, err := assetBundle(program)
	if err != nil {
		return nil, err
	}
	encodedTable, err := json.Marshal(assetTable)
	if err != nil {
		return nil, err
	}
	encodedURLs, err := json.Marshal(assetURLs)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&state, "const $canAssetTable:Parameters<typeof $canCreateAssets>[0]=%s;\n", encodedTable)
	fmt.Fprintf(&state, "$canHTML=$canCreateHTML($canDomain,{structure:%s,url:%s,target:%s,interval:%s},%s);\n", quote(numberIDs["can.std.html@1::invalid_structure"]), quote(numberIDs["can.std.html@1::invalid_url"]), quote(numberIDs["can.std.htmx@1::invalid_target"]), quote(numberIDs["can.std.htmx@1::invalid_interval"]), encodedURLs)
	fmt.Fprintf(&state, "const $canAssets=$canCreateAssets($canAssetTable, new URL(\"../\", import.meta.url));\n")
	sqlDescriptors, err := sqlTable(program)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&state, "$canSQL=$canCreateSQLDescriptors(%s);\n", sqlDescriptors)
	fmt.Fprintf(&state, "$canSQLPools=$canCreateSQLPools($canDomain,{credentialsMissing:%s,connectionFailed:%s,queryFailed:%s,rowMissing:%s,rowCount:%s,schemaMismatch:%s,constraintFailed:%s,closeFailed:%s,rowLimit:%s,unsupportedValue:%s},$canOriginalEnvironment,$canSQL);\n", quote(numberIDs["can.std.http@1::credentials_missing"]), quote(numberIDs["can.std.sql@1::connection_failed"]), quote(numberIDs["can.std.sql@1::query_failed"]), quote(numberIDs["can.std.sql@1::row_missing"]), quote(numberIDs["can.std.sql@1::row_count"]), quote(numberIDs["can.std.sql@1::schema_mismatch"]), quote(numberIDs["can.std.sql@1::constraint_failed"]), quote(numberIDs["can.std.sql@1::close_failed"]), quote(numberIDs["can.std.sql@1::row_limit"]), quote(numberIDs["can.std.sql@1::unsupported_value"]))
	fmt.Fprintf(&state, "$canTransactions=$canCreateSQLTransactions($canDomain,{connectionFailed:%s,queryFailed:%s,rowMissing:%s,rowCount:%s,schemaMismatch:%s,constraintFailed:%s,rowLimit:%s,unsupportedValue:%s,transactionFailed:%s,commitUnknown:%s},$canSQL);\n", quote(numberIDs["can.std.sql@1::connection_failed"]), quote(numberIDs["can.std.sql@1::query_failed"]), quote(numberIDs["can.std.sql@1::row_missing"]), quote(numberIDs["can.std.sql@1::row_count"]), quote(numberIDs["can.std.sql@1::schema_mismatch"]), quote(numberIDs["can.std.sql@1::constraint_failed"]), quote(numberIDs["can.std.sql@1::row_limit"]), quote(numberIDs["can.std.sql@1::unsupported_value"]), quote(numberIDs["can.std.sql@1::transaction_failed"]), quote(numberIDs["can.std.sql@1::commit_unknown"]))
	fmt.Fprintf(&state, "$canHTTPRequests=$canCreateRequests<%s>($canDomain,{invalid:%s,limit:%s,invalidData:%s,header:%s});\n", headerType, quote(numberIDs["can.std.http@1::invalid_request"]), quote(numberIDs["can.std.http@1::body_limit"]), quote(invalidData), quote(numberIDs["can.std.http@1::header"]))
	fmt.Fprintf(&state, "$canHTTPResponses=$canCreateHTTPResponses($canDomain,{invalid:%s,invalidData:%s});\n", quote(numberIDs["can.std.http@1::invalid_request"]), quote(invalidData))
	fmt.Fprintf(&state, "$canRouter=$canCreateRouter($canDomain,{invalid:%s,duplicate:%s,ambiguous:%s});\n", quote(numberIDs["can.std.http@1::invalid_route"]), quote(numberIDs["can.std.http@1::duplicate_route"]), quote(numberIDs["can.std.http@1::ambiguous_route"]))
	fmt.Fprintf(&state, "$canServer=$canCreateServer($canDomain,{invalidConfig:%s,bindFailed:%s,shutdownFailed:%s},$canAssets);\n", quote(numberIDs["can.std.http@1::invalid_server_config"]), quote(numberIDs["can.std.http@1::bind_failed"]), quote(numberIDs["can.std.http@1::shutdown_failed"]))
	fmt.Fprintf(&state, "$canClock=$canCreateClock($canDomain,%s);\n$canRandom=$canCreateRandom($canDomain,%s);\n$canLog=$canCreateLog($canDomain,%s);\n", quote(numberIDs["can.std.clock@1::invalid_duration"]), quote(numberIDs["can.std.random@1::invalid_length"]), quote(numberIDs["can.std.log@1::write_failed"]))
	fmt.Fprintf(&state, "$canIO=$canCreateIO($canDomain,{readFailed:%s,limit:%s,invalidData:%s});\n$canEnv=$canCreateEnv<%s>($canDomain,{invalidName:%s,missing:%s,some:%s,none:%s},$canOriginalEnvironment);\n", quote(numberIDs["can.std.io@1::read_failed"]), quote(numberIDs["can.std.io@1::limit_exceeded"]), quote(invalidData), optionType, quote(numberIDs["can.std.env@1::invalid_name"]), quote(numberIDs["can.std.http@1::credentials_missing"]), quote(optionIDs["can.std.option@1::some"]), quote(optionIDs["can.std.option@1::none"]))
	fmt.Fprintf(&state, "$canNumbers = $canCreateNumbers($canDomain, {inexact:%s,invalidNumber:%s,invalidTextBool:%s,invalidIntBool:%s});\n", quote(numberIDs["can.std.number@1::inexact"]), quote(numberIDs["can.std.text@1::invalid_number"]), quote(numberIDs["can.std.text@1::invalid_bool"]), quote(numberIDs["can.std.number@1::invalid_bool"]))
	fmt.Fprintf(&state, "$canAmounts = $canCreateExactAmounts($canDomain, {zeroDivisor:%s,division:%s,rounded:%s});\n", quote(numberIDs["can.std.number@1::zero_divisor"]), quote(numberIDs["can.std.number@1::division"]), quote(numberIDs["can.std.number@1::rounded"]))
	fmt.Fprintf(&state, "$canText = $canCreateText($canDomain, {emptySeparator:%s,emptyPattern:%s,invalidUnicode:%s});\n", quote(numberIDs["can.std.text@1::empty_separator"]), quote(numberIDs["can.std.text@1::empty_pattern"]), quote(numberIDs["can.std.text@1::invalid_unicode"]))
	for _, id := range collectionIDs {
		special := collectionTypes[id]
		args := special.Collection.Arguments()
		if special.Entry != nil {
			fmt.Fprintf(&state, "%s = $canCreateMap<%s,%s>($canDomain,{map:%s,entry:%s,absent:%s,exists:%s},%s);\n", collectionNames[id], TypeName(args[0]), TypeName(args[1]), quote(id), quote(special.Entry.Identity()), quote(numberIDs["can.std.collections@1::key_absent"]), quote(numberIDs["can.std.collections@1::key_exists"]), quote(args[0].Declaration()))
		} else {
			fmt.Fprintf(&state, "%s = $canCreateSet<%s>(%s,%s);\n", collectionNames[id], TypeName(args[0]), quote(id), quote(args[0].Declaration()))
		}
	}
	if fetches {
		ids := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data"} {
			ids[name] = numberIDs[declaration]
		}
		encoded, e := json.Marshal(ids)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "$canFetch=$canCreateNamedFetch($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	if llms {
		ids := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "refused": "can.std.llm@1::refused", "truncated": "can.std.llm@1::truncated", "invalidResponse": "can.std.llm@1::invalid_response"} {
			ids[name] = numberIDs[declaration]
		}
		encoded, e := json.Marshal(ids)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "$canResponses=$canCreateResponses($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	if judges {
		ids := map[string]string{}
		for _, typ := range program.Model.Types() {
			ids[typ.Declaration()] = typ.Identity()
		}
		fields := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "invalidQuestion": "can.std.ai@1::invalid_question", "invalidAnswer": "can.std.ai@1::invalid_answer"} {
			fields[name] = ids[declaration]
		}
		encoded, e := json.Marshal(fields)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "$canAI = $canCreateTypeSafe($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	for _, id := range codecIDs {
		encoded, e := json.Marshal(program.Codecs[id].Schema)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "%s = $canCreateCodec<%s>(%s, $canDomain, %s);\n", codecNames[id], TypeName(program.Codecs[id].Data), encoded, quote(invalidData))
	}
	state.WriteString(initial.Code)
	for _, value := range program.Initializers {
		fmt.Fprintf(&state, "$canValues[%s] = %s;\n", quote(value.Identity), initial.Bindings[value.Identity])
	}
	state.WriteString("Object.freeze($canValues);\n}\n")
	for _, id := range httpIDs {
		special := program.HTTPs[id]
		var encoded []byte
		var e error
		shape := ""
		switch special.Operation {
		case "can.std.http@1::request_json", "can.std.http@1::response_json":
			encoded, e = json.Marshal(special.Schema)
		case "can.std.http@1::request_form":
			encoded, e = json.Marshal(special.Form)
		default:
			e = fmt.Errorf("unknown HTTP specialization %s", special.Operation)
		}
		if e != nil {
			return nil, e
		}
		if special.Operation == "can.std.http@1::response_json" {
			shape = "encode:(status:unknown,headers:unknown,body:unknown,$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>$canHTTPResponses.json(" + string(encoded) + ",status,headers,body,$canContext)"
		} else {
			method := "json"
			if special.Operation == "can.std.http@1::request_form" {
				method = "form"
			}
			shape = "decode:(request:unknown,limit:bigint,$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>$canHTTPRequests." + method + "(" + string(encoded) + ",request,limit,$canContext)"
		}
		fmt.Fprintf(&state, "export const %s = Object.freeze({%s});\n", httpNames[id], shape)
	}
	for _, id := range sqlIDs {
		special := program.SQLs[id]
		method, err := sqlMethod(special.Operation)
		if err != nil {
			return nil, err
		}
		plan, err := sqlPlan(special)
		if err != nil {
			return nil, err
		}
		// The static descriptor literal stays in the lowered arguments for
		// fixture matching; the bound descriptor value arrives spliced per
		// call site and is the only value the runtime method consumes.
		descriptor := "Parameters<typeof $canSQL.template>[0]"
		receiver := sqlReceiver(special.Operation)
		shape := "run:(pool:unknown,_name:unknown,params:unknown,descriptor:" + descriptor + ",$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>" + receiver + "." + method + "(descriptor," + plan + ",pool,params,$canContext)"
		if special.Operation == "can.std.sql@1::query_rows" || special.Operation == "can.std.sql@1::transaction_query_rows" {
			shape = "run:(pool:unknown,_name:unknown,params:unknown,maxRows:bigint,descriptor:" + descriptor + ",$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>" + receiver + "." + method + "(descriptor," + plan + ",pool,params,maxRows,$canContext)"
		}
		fmt.Fprintf(&state, "export const %s = Object.freeze({%s});\n", sqlNames[id], shape)
	}
	for _, id := range txIDs {
		special := program.Transactions[id]
		// The commit/rollback leaves are nominal identities, so the runtime
		// classifies the callback decision without consulting bindings.
		shape := "run:(pool:unknown,callback:unknown,$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>$canTransactions.withTransaction(pool,callback,{commit:" + quote(special.Commit) + ",rollback:" + quote(special.Rollback) + "},$canContext)"
		fmt.Fprintf(&state, "export const %s = Object.freeze({%s});\n", txNames[id], shape)
	}
	imports := append(programImports(runtime), ModuleImport{Target: runtime + "/domain.ts", Names: []ImportName{{"createDomainRuntime", "$canCreateDomain"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/cli.ts", Names: []ImportName{{"createCLI", "$canCreateCLI"}}}, ModuleImport{Target: runtime + "/bytes.ts", Names: []ImportName{{"isBytes", "$canIsBytes"}, {"createBytes", "$canCreateBytes"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/collections/map.ts", Names: []ImportName{{"createMap", "$canCreateMap"}, {"isMap", "$canIsMap"}}}, ModuleImport{Target: runtime + "/collections/set.ts", Names: []ImportName{{"createSet", "$canCreateSet"}, {"isSet", "$canIsSet"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/text.ts", Names: []ImportName{{"createText", "$canCreateText"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/number.ts", Names: []ImportName{{"createNumbers", "$canCreateNumbers"}, {"createExactAmounts", "$canCreateExactAmounts"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/codec/json.ts", Names: []ImportName{{"createCodec", "$canCreateCodec"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/clock.ts", Names: []ImportName{{"createClock", "$canCreateClock"}}}, ModuleImport{Target: runtime + "/platform/random.ts", Names: []ImportName{{"createRandom", "$canCreateRandom"}}}, ModuleImport{Target: runtime + "/platform/log.ts", Names: []ImportName{{"createLog", "$canCreateLog"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/html.ts", Names: []ImportName{{"createHTML", "$canCreateHTML"}, {"isHTMLValue", "$canIsHTML"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/assets.ts", Names: []ImportName{{"createAssets", "$canCreateAssets"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/sql-descriptor.ts", Names: []ImportName{{"createSQLDescriptors", "$canCreateSQLDescriptors"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/sql.ts", Names: []ImportName{{"createSQLPools", "$canCreateSQLPools"}, {"isSQLPoolValue", "$canIsSQLPool"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/transaction.ts", Names: []ImportName{{"createSQLTransactions", "$canCreateSQLTransactions"}, {"isSQLTransactionValue", "$canIsSQLTransaction"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/http.ts", Names: []ImportName{{"createRequests", "$canCreateRequests"}, {"createResponses", "$canCreateHTTPResponses"}, {"isHTTPValue", "$canIsHTTP"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/router.ts", Names: []ImportName{{"createRouter", "$canCreateRouter"}, {"isRouterValue", "$canIsRouter"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/server.ts", Names: []ImportName{{"createServer", "$canCreateServer"}, {"isServerValue", "$canIsServer"}}})
	if judges {
		imports = append(imports, ModuleImport{Target: runtime + "/ai/typesafe.ts", Names: []ImportName{{"createTypeSafe", "$canCreateTypeSafe"}}})
	}
	if fetches {
		imports = append(imports, ModuleImport{Target: runtime + "/transport/named.ts", Names: []ImportName{{"createNamedFetch", "$canCreateNamedFetch"}}})
	}
	if llms {
		imports = append(imports, ModuleImport{Target: runtime + "/ai/responses.ts", Names: []ImportName{{"createResponses", "$canCreateResponses"}}})
	}
	imports = append(imports, ModuleImport{Target: runtime + "/environment.ts", Names: []ImportName{{"originalEnvironment", "$canOriginalEnvironment"}}}, ModuleImport{Target: runtime + "/platform/io.ts", Names: []ImportName{{"createIO", "$canCreateIO"}}}, ModuleImport{Target: runtime + "/platform/env.ts", Names: []ImportName{{"createEnvironment", "$canCreateEnv"}}})

	for _, native := range program.Natives {
		if native.ArmDescription != nil {
			name := armHandlers[native.Symbol.ID]
			imports = append(imports, ModuleImport{Target: native.Symbol.Source.OutputPath, Names: []ImportName{{name, name}}})
		}
	}
	modules := []Module{{Path: statePath, Imports: imports, Body: state.String()}}
	byPath := map[string][]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		path := fn.Symbol.Source.OutputPath
		byPath[path] = append(byPath[path], fn)
	}
	for _, path := range nativePaths {
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
		var body strings.Builder
		var regions []*ir.Region
		var descriptors []*ir.Expression
		for _, fn := range byPath[path] {
			regions = append(regions, fn.Region)
		}
		for _, native := range program.Natives {
			if native.Symbol.Source.OutputPath == path && (native.Question != nil || native.Judge != nil || native.Fetch != nil || native.LLM != nil || native.ArmDescription != nil) {
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
			return nil, err
		}
		body.WriteString(localTypes)
		for _, fn := range byPath[path] {
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID}
			code, err := emitter.Function(functions[fn.Identity()], fn.Region)
			if err != nil {
				return nil, err
			}
			body.WriteString("export ")
			body.WriteString(code)
		}
		for _, native := range program.Natives {
			if native.Symbol.Source.OutputPath != path || native.Question == nil && native.Judge == nil && native.Fetch == nil && native.LLM == nil && native.ArmDescription == nil {
				continue
			}
			for _, region := range native.Regions {
				emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
				code, e := emitter.Function(nativeNames[region.ID], region)
				if e != nil {
					return nil, e
				}
				body.WriteString("export " + code)
			}
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
			var code string
			var e error
			if native.Question != nil {
				code, e = emitter.QuestionPreparation(nativeNames[native.Symbol.ID], native.Question, nativeNames)
			} else if native.ArmDescription != nil {
				continue
			} else if native.LLM != nil {
				policy := program.Connections[native.Connection]
				code, e = emitter.LLM(nativeNames[native.Symbol.ID], native.LLM, connectionNames[native.Connection], policy.Model, policy.MaxOutputTokens)
			} else if native.Fetch != nil {
				code, e = emitter.Fetch(nativeNames[native.Symbol.ID], native.Fetch, connectionNames[native.Connection])
			} else {
				policy := program.Connections[native.Connection]
				connectionName := connectionNames[native.Connection]
				code, e = emitter.Judge(nativeNames[native.Symbol.ID], native.Judge, questions, nativeNames, connectionName, policy.Model)
			}
			if e != nil {
				return nil, e
			}
			body.WriteString("export " + code)
		}
		imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canHTML", "$canHTML"}, {"$canClock", "$canClock"}, {"$canRandom", "$canRandom"}, {"$canLog", "$canLog"}, {"$canIO", "$canIO"}, {"$canEnv", "$canEnv"}, {"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}, {"$canBytes", "$canBytes"}, {"$canHTTPRequests", "$canHTTPRequests"}, {"$canHTTPResponses", "$canHTTPResponses"}, {"$canRouter", "$canRouter"}, {"$canServer", "$canServer"}, {"$canSQL", "$canSQL"}, {"$canSQLPools", "$canSQLPools"}}})
		imports = append(imports, ModuleImport{Target: runtime + "/platform/crypto.ts", Names: []ImportName{{"sha256", "$canSHA256"}}})
		imports = append(imports, ModuleImport{Target: runtime + "/ai/questions.ts", TypeOnly: true, Names: []ImportName{{"PreparedQuestion", "$canPreparedQuestion"}, {"Answer", "$canAnswer"}}})
		if fetches {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{"$canFetch", "$canFetch"}}})
		}
		if llms {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{"$canResponses", "$canResponses"}}})
		}
		if judges {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{"$canAI", "$canAI"}}})
		}
		for _, id := range connectionIDs {
			name := connectionNames[id]
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{name, name}}})
		}
		for _, id := range nativeIDs {
			target := nativePaths[id]
			if target != path {
				imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{nativeNames[id], nativeNames[id]}}})
			}
		}
		for _, id := range collectionIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{collectionNames[id], collectionNames[id]}}})
		}
		for _, id := range codecIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{codecNames[id], codecNames[id]}}})
		}
		for _, id := range httpIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{httpNames[id], httpNames[id]}}})
		}
		for _, id := range sqlIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{sqlNames[id], sqlNames[id]}}})
		}
		for _, id := range txIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{txNames[id], txNames[id]}}})
		}
		for _, fn := range program.Functions {
			target := fn.Symbol.Source.OutputPath
			if target != path {
				imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{functions[fn.Identity()], functions[fn.Identity()]}}})
			}
		}
		modules = append(modules, Module{Path: path, Imports: imports, Body: body.String()})
	}
	if assertions {
		if len(program.Assertions) == 0 {
			return nil, fmt.Errorf("project has no concrete assertions")
		}
		entry := Module{Path: "entry.ts", Imports: []ModuleImport{
			{Target: runtime + "/assert/runner.ts", Names: []ImportName{{"runAssertionRoot", "$canRunAssertionRoot"}}},
			{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
			{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		}}
		var cases []string
		for i, test := range program.Assertions {
			rootJSON, err := json.Marshal(test.Root)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(append([]byte("can-assertion-root-v1\x00"), rootJSON...))
			path := fmt.Sprintf("assertions/%x.ts", digest)
			imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canHTML", "$canHTML"}, {"$canClock", "$canClock"}, {"$canRandom", "$canRandom"}, {"$canLog", "$canLog"}, {"$canIO", "$canIO"}, {"$canEnv", "$canEnv"}, {"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}, {"$canBytes", "$canBytes"}, {"$canHTTPRequests", "$canHTTPRequests"}, {"$canHTTPResponses", "$canHTTPResponses"}, {"$canRouter", "$canRouter"}, {"$canServer", "$canServer"}, {"$canSQL", "$canSQL"}, {"$canSQLPools", "$canSQLPools"}}})
			imports = append(imports, ModuleImport{Target: runtime + "/platform/crypto.ts", Names: []ImportName{{"sha256", "$canSHA256"}}})
			for _, id := range collectionIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{collectionNames[id], collectionNames[id]}}})
			}
			for _, id := range codecIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{codecNames[id], codecNames[id]}}})
			}
			for _, id := range httpIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{httpNames[id], httpNames[id]}}})
			}
			for _, id := range sqlIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{sqlNames[id], sqlNames[id]}}})
			}
			for _, id := range txIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{txNames[id], txNames[id]}}})
			}
			for _, fn := range program.Functions {
				imports = append(imports, ModuleImport{Target: fn.Symbol.Source.OutputPath, Names: []ImportName{{functions[fn.Identity()], functions[fn.Identity()]}}})
			}
			for _, id := range nativeIDs {
				target := nativePaths[id]
				if name := functions[id]; name != "" {
					imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{name, name}}})
				}
			}
			graph := append(program.Model.Types(), regionTypes(test.Actual, test.Expected)...)
			body, err := NativeTypeDeclarations(graph)
			if err != nil {
				return nil, err
			}
			sourceID := ""
			for src := range program.World.Files {
				if src.Syntax.Source.Name() == test.Actual.Source {
					sourceID = src.ID
				}
			}
			if sourceID == "" {
				return nil, fmt.Errorf("assertion source is not indexed")
			}
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: sourceID}
			actual, err := emitter.Function("$canActual", test.Actual)
			if err != nil {
				return nil, err
			}
			expected, err := emitter.Function("$canExpected", test.Expected)
			if err != nil {
				return nil, err
			}
			body += actual + expected + fmt.Sprintf("export const $canCase = Object.freeze({root: Object.freeze(%s), actual: $canActual, expected: $canExpected});\n", rootJSON)
			modules = append(modules, Module{Path: path, Imports: imports, Body: body})
			name := fmt.Sprintf("$canCase%d", i)
			cases = append(cases, name)
			entry.Imports = append(entry.Imports, ModuleImport{Target: path, Names: []ImportName{{"$canCase", name}}})
		}
		entry.Body = "process.exitCode = await $canRunAssertionRoot([" + strings.Join(cases, ",") + "], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, process.argv.slice(2));\n"
		modules = append(modules, entry)
		artifacts, err := Modules(modules, dependencies...)
		if err != nil {
			return nil, err
		}
		return append(artifacts, assetFiles...), nil
	}
	main := program.Entry
	modules = append(modules, Module{Path: "entry.ts", Imports: []ModuleImport{
		{Target: runtime + "/entry.ts", Names: []ImportName{{"runEntry", "$canRunEntry"}}},
		{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{functions[main.Identity()], "$canMain"}}},
	}, Body: "process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));\n"})
	artifacts, err := Modules(modules, dependencies...)
	if err != nil {
		return nil, err
	}
	return append(artifacts, assetFiles...), nil
}

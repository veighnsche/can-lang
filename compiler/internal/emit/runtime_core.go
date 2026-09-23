package emit

// programImports lists the runtime modules every authored module needs. The
// state module extends this list with factory imports per domain.
func programImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/collections/array.ts", Names: arrayImports()},
		{Target: runtime + "/assert/context.ts", Names: []ImportName{{"callContext", "$canCallContext"}, {"scopeRequest", "$canScopeRequest"}}},
		{Target: runtime + "/coordination.ts", Names: []ImportName{{"settle", "$canCoordinateSettle"}, {"handle", "$canCoordinateHandle"}, {"aggregate", "$canCoordinateAggregate"}}},
		{Target: runtime + "/owner.ts", TypeOnly: true, Names: []ImportName{{"Participant", "$canParticipant"}}},
		{Target: runtime + "/bytes.ts", Names: []ImportName{{"byteLength", "$canByteLength"}}},
		{Target: runtime + "/callable.ts", Names: []ImportName{{"ownCallable", "$canOwnCallable"}, {"callableInstance", "$canCallableInstance"}}},
		{Target: runtime + "/assert/fixtures.ts", Names: []ImportName{{"withFixture", "$canWithFixture"}}},
		{Target: runtime + "/assert/policy.ts", Names: []ImportName{{"policyBase", "$canPolicyBase"}, {"policyKey", "$canPolicyKey"}, {"policyLeaf", "$canPolicyLeaf"}, {"provideInjection", "$canPolicyProvide"}}},
		{Target: runtime + "/completion.ts", Names: []ImportName{{"success", "$canSuccess"}, {"failure", "$canFailure"}, {"value", "$canValue"}, {"invoke", "$canInvoke"}, {"caught", "$canCaught"}, {"errorType", "$canErrorType"}, {"errorPayload", "$canErrorPayload"}}},
		{Target: runtime + "/completion.ts", TypeOnly: true, Names: []ImportName{{"Completion", "$canCompletion"}, {"AssertionContext", "$canAssertionContext"}}},
		{Target: runtime + "/data.ts", Names: []ImportName{{"record", "$canRecord"}, {"update", "$canUpdate"}, {"array", "$canArray"}, {"recordIdentity", "$canRecordIdentity"}}},
		{Target: runtime + "/primitive.ts", Names: []ImportName{{"intDivide", "$canIntDivide"}, {"intRemainder", "$canIntRemainder"}, {"intPower", "$canIntPower"}, {"index", "$canIndex"}, {"slice", "$canSlice"}}},
		{Target: runtime + "/failure.ts", Names: []ImportName{{"captureStandard", "$canCaptureStandard"}, {"isStandardFailure", "$canIsStandardFailure"}, {"standardFailureKind", "$canFailureKind"}, {"standardFailureMessage", "$canFailureMessage"}, {"standardFailureOccurrenceID", "$canFailureOccurrenceID"}}},
	}
}

// coreOperationBindings maps the stable pre-B1 library operations to their
// state-module targets. New B1 capabilities add their own binding files;
// this table keeps only behavior owned before the Bun milestone. Existing
// HTTP/crypto entries move to their feature files when B1-06 and B1-08
// begin; SQL entries live in runtime_sql.go.
func coreOperationBindings() bindingContribution {
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
	functions["can.std.http@1::route_put"] = "$canRouter.put"
	functions["can.std.http@1::route_patch"] = "$canRouter.patch"
	functions["can.std.http@1::route_delete"] = "$canRouter.delete"
	functions["can.std.http@1::route_options"] = "$canRouter.options"
	functions["can.std.http@1::route_head"] = "$canRouter.head"
	functions["can.std.http@1::make_router"] = "$canRouter.make"
	functions["can.std.http@1::route_stream"] = "$canRouter.stream"
	functions["can.std.http@1::request_body_stream"] = "$canHTTPRequests.bodyStream"
	functions["can.std.http@1::make_server_config"] = "$canServer.makeConfig"
	functions["can.std.http@1::server_start"] = "$canServer.start"
	functions["can.std.http@1::make_tls_config"] = "$canServer.makeTlsConfig"
	functions["can.std.http@1::server_start_tls"] = "$canServer.startTls"
	functions["can.std.http@1::server_stop"] = "$canServer.stop"
	functions["can.std.http@1::server_wait"] = "$canServer.wait"
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
	functions["can.std.checks@1::require"] = "$canChecks.require"
	return bindingContribution{domain: "core", functions: functions}
}

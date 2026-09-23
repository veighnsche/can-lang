package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// utilitiesOperationBindings maps the concrete URL, time, text-regex,
// and byte-encoding operations to their state-module targets. Regex and
// encodings reuse the core text and bytes factories; only URL and time
// own new factory values.
func utilitiesOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.url@1::parse":                      "$canURLs.parseURL",
		"can.std.url@1::resolve":                    "$canURLs.resolveURL",
		"can.std.url@1::to_string":                  "$canURLs.urlToString",
		"can.std.url@1::query_all":                  "$canURLs.queryAll",
		"can.std.url@1::query_pairs":                "$canURLs.queryPairs",
		"can.std.url@1::with_query":                 "$canURLs.withQuery",
		"can.std.time@1::instant_from_epoch_millis": "$canDateTimes.instantFromEpochMillis",
		"can.std.time@1::instant_epoch_millis":      "$canDateTimes.instantEpochMillis",
		"can.std.time@1::format_in_zone":            "$canDateTimes.formatInZone",
		"can.std.time@1::resolve_zoned_time":        "$canDateTimes.resolveZonedTime",
		"can.std.text@1::compile_regex":             "$canText.compileRegex",
		"can.std.text@1::matches":                   "$canText.findMatches",
		"can.std.bytes@1::encode_base64":            "$canBytes.encodeBase64",
		"can.std.bytes@1::decode_base64":            "$canBytes.decodeBase64",
		"can.std.bytes@1::encode_hex":               "$canBytes.encodeHex",
		"can.std.bytes@1::decode_hex":               "$canBytes.decodeHex",
	}
	return bindingContribution{domain: "utilities", functions: functions}
}

// utilitiesStateImports lists the URL and datetime factory modules the
// shared state module needs.
func (assembly *programAssembly) utilitiesStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/url.ts", Names: []ImportName{{"createURLs", "$canCreateURLs"}}},
		{Target: runtime + "/platform/datetime.ts", Names: []ImportName{{"createDateTimes", "$canCreateDateTimes"}, {"isTimeInstantValue", "$canIsTimeInstant"}}},
	}
}

// utilitiesStateValueImportNames lists the URL and datetime factory
// values authored and assertion modules import from the state module.
func utilitiesStateValueImportNames() []ImportName {
	return []ImportName{{"$canURLs", "$canURLs"}, {"$canDateTimes", "$canDateTimes"}}
}

// declareUtilitiesState emits the URL and datetime factory bindings.
func (builder *stateBuilder) declareUtilitiesState() {
	builder.out.WriteString("export let $canURLs:ReturnType<typeof $canCreateURLs>;\nexport let $canDateTimes:ReturnType<typeof $canCreateDateTimes>;\n")
}

// initializeUtilitiesState creates the URL and datetime factories inside
// the shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeUtilitiesState() {
	fmt.Fprintf(&builder.out, "$canURLs=$canCreateURLs($canDomain,{invalidURL:%s,parts:%s,pair:%s});\n", quote(builder.numberIDs["can.std.url@1::invalid_url"]), quote(builder.numberIDs["can.std.url@1::parts"]), quote(builder.numberIDs["can.std.url@1::query_pair"]))
	fmt.Fprintf(&builder.out, "$canDateTimes=$canCreateDateTimes($canDomain,{outOfRange:%s,invalidZone:%s,nonexistent:%s,invalidOption:%s});\n", quote(builder.numberIDs["can.std.time@1::out_of_range"]), quote(builder.numberIDs["can.std.time@1::invalid_zone"]), quote(builder.numberIDs["can.std.time@1::nonexistent_time"]), quote(builder.numberIDs["can.std.time@1::invalid_option"]))
}

// emitUtilitiesKinds emits the opaque-handle kind table the domain
// predicate uses to recognize regex and instant values at boundaries.
func (builder *stateBuilder) emitUtilitiesKinds() error {
	utilitiesKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.text@1::regex":
			utilitiesKinds[typ.Identity()] = "regex"
		case "can.std.time@1::instant":
			utilitiesKinds[typ.Identity()] = "instant"
		}
	}
	utilitiesKindsJSON, err := json.Marshal(utilitiesKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canUtilitiesKinds:Readonly<Record<string,string>>=%s;\n", utilitiesKindsJSON)
	return nil
}

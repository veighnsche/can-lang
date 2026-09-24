package emit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// programStatePath is the generation-relative path of the shared
// initialization/state module every other module imports.
const programStatePath = "program/state.ts"

// stateBuilder assembles the shared state module: type declarations,
// factory bindings, the ordered initializer and per-program
// specialization constants. Domain fragments (collections, AI) live in
// their own files and are invoked here in the fixed dependency order;
// existing HTTP/SQL/codec fragments move to their feature files when
// B1-02, B1-06 and B1-11 begin.
type stateBuilder struct {
	assembly     *programAssembly
	out          strings.Builder
	numberIDs    map[string]string
	headerType   string
	optionType   string
	optionIDs    map[string]string
	plan         []byte
	initial      InitializedValues
	declarations string
}

// emitStateModule builds the shared state module and returns it with the
// asset file artifacts produced alongside it.
func emitStateModule(assembly *programAssembly, runtime string) (Module, []ir.Artifact, error) {
	builder := &stateBuilder{assembly: assembly}
	if err := builder.prerequisites(); err != nil {
		return Module{}, nil, err
	}
	browser := builder.assembly.browser
	builder.out.WriteString(builder.declarations)
	builder.declareCoreState()
	if !browser {
		if err := builder.declareConnections(); err != nil {
			return Module{}, nil, err
		}
		builder.declareAIState()
	}
	builder.declareCodecState()
	builder.declareCollectionState()
	builder.declareBrowserState()
	builder.declareBrowserStateSpecializations()
	builder.declareFetchState()
	if !browser {
		builder.declareFileState()
		builder.declareProcessState()
		builder.declareStreamState()
		builder.declareWebSocketState()
	}
	builder.declareCookiesState()
	if !browser {
		builder.declareS3State()
	}
	builder.declareMarkdownState()
	if browser {
		builder.out.WriteString("export let $canText: ReturnType<typeof $canCreateText>;\nexport let $canAmounts: ReturnType<typeof $canCreateExactAmounts>;\nexport let $canNumbers: ReturnType<typeof $canCreateNumbers>;\nexport let $canChecks: ReturnType<typeof $canCreateChecks>;\nexport let $canBytes: ReturnType<typeof $canCreateBytes>;\nexport let $canDomain: ReturnType<typeof $canCreateDomain>;\nexport const $canValues: Record<string, unknown> = Object.create(null);\nexport function $canInitialize(): void {\n")
	} else {
		builder.out.WriteString("export let $canText: ReturnType<typeof $canCreateText>;\nexport let $canAmounts: ReturnType<typeof $canCreateExactAmounts>;\nexport let $canNumbers: ReturnType<typeof $canCreateNumbers>;\nexport let $canChecks: ReturnType<typeof $canCreateChecks>;\nexport let $canBytes: ReturnType<typeof $canCreateBytes>;\nexport let $canCLI: ReturnType<typeof $canCreateCLI>;\nexport let $canDomain: ReturnType<typeof $canCreateDomain>;\nexport const $canValues: Record<string, unknown> = Object.create(null);\nexport function $canInitialize(): void {\n")
	}
	if err := builder.initializeDomain(); err != nil {
		return Module{}, nil, err
	}
	builder.collectNumberIDs()
	assetFiles, err := builder.initializeAssets()
	if err != nil {
		return Module{}, nil, err
	}
	if !browser {
		if err := builder.initializeSQLState(); err != nil {
			return Module{}, nil, err
		}
		builder.initializeCryptoState()
	}
	builder.initializeUtilitiesState()
	builder.initializeCoreState()
	if !browser {
		builder.initializeFileState()
		builder.initializeProcessState()
		builder.initializeStreamState()
		builder.initializeWebSocketState()
	}
	builder.initializeCookiesState()
	if !browser {
		builder.initializeS3State()
	}
	builder.initializeMarkdownState()
	builder.initializeBrowserState()
	builder.initializeBrowserStateSpecializations()
	builder.initializeFetchState()
	builder.initializeCollectionState()
	if !browser {
		if err := builder.initializeAIState(); err != nil {
			return Module{}, nil, err
		}
	}
	if err := builder.initializeCodecs(); err != nil {
		return Module{}, nil, err
	}
	builder.initializeValues()
	if err := builder.emitSpecializationConstants(); err != nil {
		return Module{}, nil, err
	}
	if err := builder.emitActionConstants(); err != nil {
		return Module{}, nil, err
	}
	return Module{Path: programStatePath, Imports: builder.stateImports(runtime), Body: builder.out.String()}, assetFiles, nil
}

// prerequisites computes the error plan, top-level initializers, shared
// type declarations and the option/header types every later fragment needs.
func (builder *stateBuilder) prerequisites() error {
	program := builder.assembly.program
	var errorTypes []*types.Type
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Error {
			errorTypes = append(errorTypes, typ)
		}
	}
	bound, err := program.Registry.Bound(errorTypes)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		return err
	}
	builder.plan = plan
	initial, err := initialization(program.Initializers, nil, builder.assembly.armHandlers, true)
	if err != nil {
		return err
	}
	builder.initial = initial
	declarations, err := NativeTypeDeclarations(program.Model.Types())
	if err != nil {
		return err
	}
	builder.declarations = declarations
	optionResult := program.Intrinsics["can.std.env@1::optional"].Result()
	builder.optionType = TypeName(optionResult)
	builder.optionIDs = map[string]string{}
	for _, leaf := range optionResult.Leaves() {
		builder.optionIDs[leaf.Declaration()] = leaf.Identity()
	}
	builder.headerType = ""
	for _, typ := range program.Model.Types() {
		if typ.Declaration() == "can.std.http@1::header" {
			builder.headerType = TypeName(typ)
		}
	}
	if builder.headerType == "" {
		return fmt.Errorf("HTTP header type is not in the checked model")
	}
	return nil
}

// declareCoreState emits the pre-B1 factory bindings. HTTP entries move to
// their feature file with B1-06; SQL entries live in runtime_sql.go.
func (builder *stateBuilder) declareCoreState() {
	builder.out.WriteString("export let $canHTML:ReturnType<typeof $canCreateHTML>;\nexport let $canForm:ReturnType<typeof $canCreateForm>;\nexport let $canFormActions:ReturnType<typeof $canCreateFormActions>;\n")
	if !builder.assembly.browser {
		builder.declareSQLState()
		builder.declareCryptoState()
	}
	builder.declareUtilitiesState()
	if builder.assembly.browser {
		fmt.Fprintf(&builder.out, "export let $canHTTPRequests: ReturnType<typeof $canCreateRequests<%s>>;\nexport let $canHTTPResponses: ReturnType<typeof $canCreateHTTPResponses>;\nexport let $canRouter: ReturnType<typeof $canCreateRouter>;\n", builder.headerType)
	} else {
		fmt.Fprintf(&builder.out, "export let $canHTTPRequests: ReturnType<typeof $canCreateRequests<%s>>;\nexport let $canHTTPResponses: ReturnType<typeof $canCreateHTTPResponses>;\nexport let $canRouter: ReturnType<typeof $canCreateRouter>;\nexport let $canServer: ReturnType<typeof $canCreateServer>;\n", builder.headerType)
	}
	builder.out.WriteString("export let $canClock:ReturnType<typeof $canCreateClock>;\nexport let $canRandom:ReturnType<typeof $canCreateRandom>;\nexport let $canLog:ReturnType<typeof $canCreateLog>;\n")
	if !builder.assembly.browser {
		fmt.Fprintf(&builder.out, "export let $canIO: ReturnType<typeof $canCreateIO>;\nexport let $canEnv: ReturnType<typeof $canCreateEnv<%s>>;\n", builder.optionType)
	}
}

// declareCodecState emits one codec binding per JSON and document
// specialization used by the program.
func (builder *stateBuilder) declareCodecState() {
	for _, id := range builder.assembly.codecIDs {
		fmt.Fprintf(&builder.out, "export let %s: ReturnType<typeof $canCreateCodec<%s>>;\n", builder.assembly.codecNames[id], TypeName(builder.assembly.program.Codecs[id].Data))
	}
}

// initializeDomain creates the shared domain runtime, bytes and CLI
// factories that later fragments build on.
func (builder *stateBuilder) initializeDomain() error {
	var invalidData, writeFailed, bytesID string
	for _, typ := range builder.assembly.program.Model.Types() {
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
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() == types.Opaque && (strings.HasPrefix(typ.Declaration(), "can.std.html@1::") || strings.HasPrefix(typ.Declaration(), "can.std.htmx@1::")) {
			htmlKinds[typ.Identity()] = strings.Split(typ.Declaration(), "::")[1]
		}
	}
	htmlKindsJSON, err := json.Marshal(htmlKinds)
	if err != nil {
		return err
	}
	httpKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque || !strings.HasPrefix(typ.Declaration(), "can.std.http@1::") {
			continue
		}
		switch kind := strings.Split(typ.Declaration(), "::")[1]; kind {
		case "request", "status", "body_status", "server_headers", "server_response", "route", "router", "server_config", "tls_config", "server":
			httpKinds[typ.Identity()] = kind
		}
	}
	httpKindsJSON, err := json.Marshal(httpKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canHTMLKinds:Readonly<Record<string,string>>=%s;\n", htmlKindsJSON)
	fmt.Fprintf(&builder.out, "const $canHTTPKinds:Readonly<Record<string,string>>=%s;\n", httpKindsJSON)
	if !builder.assembly.browser {
		if err := builder.emitSQLKinds(); err != nil {
			return err
		}
		if err := builder.emitCryptoKinds(); err != nil {
			return err
		}
	}
	if err := builder.emitUtilitiesKinds(); err != nil {
		return err
	}
	if !builder.assembly.browser {
		if err := builder.emitStreamKinds(); err != nil {
			return err
		}
		if err := builder.emitWebSocketKinds(); err != nil {
			return err
		}
	}
	if err := builder.emitCookiesKinds(); err != nil {
		return err
	}
	if err := builder.emitBrowserKinds(); err != nil {
		return err
	}
	if !builder.assembly.browser {
		if err := builder.emitS3Kinds(); err != nil {
			return err
		}
	}
	if builder.assembly.browser {
		fmt.Fprintf(&builder.out, "$canDomain = $canCreateDomain(%s, (identity, value) => (identity === %s && $canIsBytes(value)) || $canIsMap(identity,value) || $canIsSet(identity,value) || $canIsHTML($canHTMLKinds[identity],value) || $canIsHTTP($canHTTPKinds[identity],value) || $canIsRouter($canHTTPKinds[identity],value) || $canIsTextRegex($canUtilitiesKinds[identity],value) || $canIsTimeInstant($canUtilitiesKinds[identity],value) || $canIsCookie($canCookiesKinds[identity],value) || $canIsBrowser($canBrowserKinds[identity],value) || $canIsBrowserState(identity,value));\n", builder.plan, quote(bytesID))
		fmt.Fprintf(&builder.out, "$canBytes = $canCreateBytes($canDomain, %s);\n", quote(invalidData))
	} else {
		fmt.Fprintf(&builder.out, "$canDomain = $canCreateDomain(%s, (identity, value) => (identity === %s && $canIsBytes(value)) || $canIsMap(identity,value) || $canIsSet(identity,value) || $canIsHTML($canHTMLKinds[identity],value) || $canIsHTTP($canHTTPKinds[identity],value) || $canIsRouter($canHTTPKinds[identity],value) || $canIsServer($canHTTPKinds[identity],value) || $canIsSQLPool($canSQLKinds[identity],value) || $canIsSQLTransaction($canSQLKinds[identity],value) || $canIsCryptoKey($canCryptoKinds[identity],value) || $canIsTextRegex($canUtilitiesKinds[identity],value) || $canIsTimeInstant($canUtilitiesKinds[identity],value) || $canIsStream($canStreamKinds[identity],value) || $canIsWebSocket($canWebSocketKinds[identity],value) || $canIsCookie($canCookiesKinds[identity],value) || $canIsS3($canS3Kinds[identity],value) || $canIsBrowser($canBrowserKinds[identity],value) || $canIsBrowserState(identity,value));\n", builder.plan, quote(bytesID))
		fmt.Fprintf(&builder.out, "$canBytes = $canCreateBytes($canDomain, %s);\n$canCLI = $canCreateCLI($canDomain, {writeFailed: %s});\n", quote(invalidData), quote(writeFailed))
	}
	builder.numberIDs = map[string]string{}
	return nil
}

// collectNumberIDs indexes checked type identities by declaration for the
// factory-argument fragments below.
func (builder *stateBuilder) collectNumberIDs() {
	for _, typ := range builder.assembly.program.Model.Types() {
		builder.numberIDs[typ.Declaration()] = typ.Identity()
	}
}

// initializeAssets creates the HTML factory and the asset bundle. It
// returns the asset file artifacts emitted alongside the program.
func (builder *stateBuilder) initializeAssets() ([]ir.Artifact, error) {
	assetTable, assetURLs, assetFiles, err := assetBundle(builder.assembly.program)
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
	fmt.Fprintf(&builder.out, "const $canAssetTable:Parameters<typeof $canCreateAssets>[0]=%s;\n", encodedTable)
	fmt.Fprintf(&builder.out, "$canHTML=$canCreateHTML($canDomain,{structure:%s,url:%s,target:%s,interval:%s},%s);\n", quote(builder.numberIDs["can.std.html@1::invalid_structure"]), quote(builder.numberIDs["can.std.html@1::invalid_url"]), quote(builder.numberIDs["can.std.htmx@1::invalid_target"]), quote(builder.numberIDs["can.std.htmx@1::invalid_interval"]), encodedURLs)
	fmt.Fprintf(&builder.out, "const $canAssets=$canCreateAssets($canAssetTable, new URL(\"../\", import.meta.url));\n")
	return assetFiles, nil
}

// initializeCoreState creates the remaining pre-B1 factories: HTTP, clock,
// random, log, IO, environment, numbers, checks, amounts and text.
func (builder *stateBuilder) initializeCoreState() {
	fmt.Fprintf(&builder.out, "$canHTTPRequests=$canCreateRequests<%s>($canDomain,{invalid:%s,limit:%s,invalidData:%s,header:%s,close:%s,writeFailed:%s,multipartForm:%s,multipartField:%s,multipartFile:%s});\n", builder.headerType, quote(builder.numberIDs["can.std.http@1::invalid_request"]), quote(builder.numberIDs["can.std.http@1::body_limit"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.http@1::header"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]), quote(builder.numberIDs["can.std.stream@1::write_failed"]), quote(builder.numberIDs["can.std.http@1::multipart_form"]), quote(builder.numberIDs["can.std.http@1::multipart_field"]), quote(builder.numberIDs["can.std.http@1::multipart_file"]))
	fmt.Fprintf(&builder.out, "$canHTTPResponses=$canCreateHTTPResponses($canDomain,{invalid:%s,invalidData:%s,close:%s,writeFailed:%s,limit:%s});\n", quote(builder.numberIDs["can.std.http@1::invalid_request"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]), quote(builder.numberIDs["can.std.stream@1::write_failed"]), quote(builder.numberIDs["can.std.http@1::body_limit"]))
	fmt.Fprintf(&builder.out, "$canRouter=$canCreateRouter($canDomain,{invalid:%s,duplicate:%s,ambiguous:%s});\n", quote(builder.numberIDs["can.std.http@1::invalid_route"]), quote(builder.numberIDs["can.std.http@1::duplicate_route"]), quote(builder.numberIDs["can.std.http@1::ambiguous_route"]))
	fmt.Fprintf(&builder.out, "$canForm=$canCreateForm($canDomain,{unknownField:%s,invalidName:%s});\n", quote(builder.numberIDs["can.std.form@1::unknown_field"]), quote(builder.numberIDs["can.std.form@1::invalid_name"]))
	fmt.Fprintf(&builder.out, "$canFormActions=$canCreateFormActions($canDomain,{invalidRoute:%s});\n", quote(builder.numberIDs["can.std.http@1::invalid_route"]))
	if !builder.assembly.browser {
		fmt.Fprintf(&builder.out, "$canServer=$canCreateServer($canDomain,{invalidConfig:%s,bindFailed:%s,shutdownFailed:%s},$canAssets);\n", quote(builder.numberIDs["can.std.http@1::invalid_server_config"]), quote(builder.numberIDs["can.std.http@1::bind_failed"]), quote(builder.numberIDs["can.std.http@1::shutdown_failed"]))
	}
	fmt.Fprintf(&builder.out, "$canClock=$canCreateClock($canDomain,%s);\n$canRandom=$canCreateRandom($canDomain,%s);\n$canLog=$canCreateLog($canDomain,%s);\n", quote(builder.numberIDs["can.std.clock@1::invalid_duration"]), quote(builder.numberIDs["can.std.random@1::invalid_length"]), quote(builder.numberIDs["can.std.log@1::write_failed"]))
	if !builder.assembly.browser {
		fmt.Fprintf(&builder.out, "$canIO=$canCreateIO($canDomain,{readFailed:%s,limit:%s,invalidData:%s});\n$canEnv=$canCreateEnv<%s>($canDomain,{invalidName:%s,missing:%s,some:%s,none:%s},$canOriginalEnvironment);\n", quote(builder.numberIDs["can.std.io@1::read_failed"]), quote(builder.numberIDs["can.std.io@1::limit_exceeded"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), builder.optionType, quote(builder.numberIDs["can.std.env@1::invalid_name"]), quote(builder.numberIDs["can.std.http@1::credentials_missing"]), quote(builder.optionIDs["can.std.option@1::some"]), quote(builder.optionIDs["can.std.option@1::none"]))
	}
	fmt.Fprintf(&builder.out, "$canNumbers = $canCreateNumbers($canDomain, {inexact:%s,invalidNumber:%s,invalidTextBool:%s,invalidIntBool:%s});\n", quote(builder.numberIDs["can.std.number@1::inexact"]), quote(builder.numberIDs["can.std.text@1::invalid_number"]), quote(builder.numberIDs["can.std.text@1::invalid_bool"]), quote(builder.numberIDs["can.std.number@1::invalid_bool"]))
	fmt.Fprintf(&builder.out, "$canChecks = $canCreateChecks($canDomain, {failed:%s});\n", quote(builder.numberIDs["can.std.checks@1::failed"]))
	fmt.Fprintf(&builder.out, "$canAmounts = $canCreateExactAmounts($canDomain, {zeroDivisor:%s,division:%s,rounded:%s});\n", quote(builder.numberIDs["can.std.number@1::zero_divisor"]), quote(builder.numberIDs["can.std.number@1::division"]), quote(builder.numberIDs["can.std.number@1::rounded"]))
	fmt.Fprintf(&builder.out, "$canText = $canCreateText($canDomain, {emptySeparator:%s,emptyPattern:%s,invalidUnicode:%s,invalidRegex:%s,invalidLimit:%s,match:%s});\n", quote(builder.numberIDs["can.std.text@1::empty_separator"]), quote(builder.numberIDs["can.std.text@1::empty_pattern"]), quote(builder.numberIDs["can.std.text@1::invalid_unicode"]), quote(builder.numberIDs["can.std.text@1::invalid_regex"]), quote(builder.numberIDs["can.std.text@1::invalid_limit"]), quote(builder.numberIDs["can.std.text@1::regex_match"]))
}

// initializeCodecs constructs the JSON and document codec specializations
// of this program. Consume shares the schema-bound factory: its reader,
// handler and count contracts resolve in the checker.
func (builder *stateBuilder) initializeCodecs() error {
	for _, id := range builder.assembly.codecIDs {
		encoded, err := json.Marshal(builder.assembly.program.Codecs[id].Schema)
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder.out, "%s = $canCreateCodec<%s>(%s, $canDomain, {invalidData:%s,readFailed:%s,cancelled:%s});\n", builder.assembly.codecNames[id], TypeName(builder.assembly.program.Codecs[id].Data), encoded, quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.stream@1::read_failed"]), quote(builder.numberIDs["can.std.stream@1::cancelled"]))
	}
	return nil
}

// initializeValues evaluates the ordered top-level initializers into the
// frozen shared value table and closes the initializer.
func (builder *stateBuilder) initializeValues() {
	builder.out.WriteString(builder.initial.Code)
	for _, value := range builder.assembly.program.Initializers {
		fmt.Fprintf(&builder.out, "$canValues[%s] = %s;\n", quote(value.Identity), builder.initial.Bindings[value.Identity])
	}
	builder.out.WriteString("Object.freeze($canValues);\n}\n")
}

// emitSpecializationConstants freezes the per-program HTTP dispatch records
// after the initializer, then delegates SQL/transaction records to the SQL
// feature file.
func (builder *stateBuilder) emitSpecializationConstants() error {
	for _, id := range builder.assembly.httpIDs {
		special := builder.assembly.program.HTTPs[id]
		var encoded []byte
		var err error
		shape := ""
		switch special.Operation {
		case "can.std.http@1::request_json", "can.std.http@1::response_json":
			encoded, err = json.Marshal(special.Schema)
		case "can.std.http@1::request_form":
			encoded, err = json.Marshal(special.Form)
		default:
			err = fmt.Errorf("unknown HTTP specialization %s", special.Operation)
		}
		if err != nil {
			return err
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
		fmt.Fprintf(&builder.out, "export const %s = Object.freeze({%s});\n", builder.assembly.httpNames[id], shape)
	}
	if builder.assembly.browser {
		return nil
	}
	return builder.emitSQLSpecializationConstants()
}

// stateImports lists the factory modules the shared state module needs, in
// the fixed order the initializer depends on.
func (builder *stateBuilder) stateImports(runtime string) []ModuleImport {
	browser := builder.assembly.browser
	imports := append(programImports(runtime), ModuleImport{Target: runtime + "/domain.ts", Names: []ImportName{{"createDomainRuntime", "$canCreateDomain"}}})
	if browser {
		imports = append(imports, ModuleImport{Target: runtime + "/bytes.ts", Names: []ImportName{{"isBytes", "$canIsBytes"}, {"createBytes", "$canCreateBytes"}}})
	} else {
		imports = append(imports, ModuleImport{Target: runtime + "/platform/cli.ts", Names: []ImportName{{"createCLI", "$canCreateCLI"}}}, ModuleImport{Target: runtime + "/bytes.ts", Names: []ImportName{{"isBytes", "$canIsBytes"}, {"createBytes", "$canCreateBytes"}}})
	}
	imports = append(imports, builder.assembly.collectionStateImports(runtime)...)
	imports = append(imports, ModuleImport{Target: runtime + "/text.ts", Names: []ImportName{{"createText", "$canCreateText"}, {"isTextRegexValue", "$canIsTextRegex"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/number.ts", Names: []ImportName{{"createNumbers", "$canCreateNumbers"}, {"createExactAmounts", "$canCreateExactAmounts"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/checks.ts", Names: []ImportName{{"createChecks", "$canCreateChecks"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/codec/json.ts", Names: []ImportName{{"createCodec", "$canCreateCodec"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/clock.ts", Names: []ImportName{{"createClock", "$canCreateClock"}}}, ModuleImport{Target: runtime + "/platform/random.ts", Names: []ImportName{{"createRandom", "$canCreateRandom"}}}, ModuleImport{Target: runtime + "/platform/log.ts", Names: []ImportName{{"createLog", "$canCreateLog"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/html.ts", Names: []ImportName{{"createHTML", "$canCreateHTML"}, {"isHTMLValue", "$canIsHTML"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/form.ts", Names: []ImportName{{"createForm", "$canCreateForm"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/assets.ts", Names: []ImportName{{"createAssets", "$canCreateAssets"}}})
	if !browser {
		imports = append(imports, builder.assembly.sqlStateImports(runtime)...)
		imports = append(imports, builder.assembly.cryptoStateImports(runtime)...)
	}
	imports = append(imports, builder.assembly.utilitiesStateImports(runtime)...)
	imports = append(imports, ModuleImport{Target: runtime + "/platform/http.ts", Names: []ImportName{{"createRequests", "$canCreateRequests"}, {"createResponses", "$canCreateHTTPResponses"}, {"isHTTPValue", "$canIsHTTP"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/router.ts", Names: []ImportName{{"createRouter", "$canCreateRouter"}, {"createFormActions", "$canCreateFormActions"}, {"isRouterValue", "$canIsRouter"}}})
	imports = append(imports, builder.assembly.fetchStateImports(runtime)...)
	if !browser {
		imports = append(imports, ModuleImport{Target: runtime + "/platform/server.ts", Names: []ImportName{{"createServer", "$canCreateServer"}, {"isServerValue", "$canIsServer"}}})
		imports = append(imports, builder.assembly.fileStateImports(runtime)...)
		imports = append(imports, builder.assembly.processStateImports(runtime)...)
		imports = append(imports, builder.assembly.streamStateImports(runtime)...)
		imports = append(imports, builder.assembly.websocketStateImports(runtime)...)
	}
	imports = append(imports, builder.assembly.cookiesStateImports(runtime)...)
	if !browser {
		imports = append(imports, builder.assembly.s3StateImports(runtime)...)
	}
	imports = append(imports, builder.assembly.markdownStateImports(runtime)...)
	imports = append(imports, builder.assembly.browserStateImports(runtime)...)
	if !browser {
		imports = append(imports, builder.assembly.aiStateImports(runtime)...)
		imports = append(imports, ModuleImport{Target: runtime + "/environment.ts", Names: []ImportName{{"originalEnvironment", "$canOriginalEnvironment"}}}, ModuleImport{Target: runtime + "/platform/io.ts", Names: []ImportName{{"createIO", "$canCreateIO"}}}, ModuleImport{Target: runtime + "/platform/env.ts", Names: []ImportName{{"createEnvironment", "$canCreateEnv"}}})
		imports = append(imports, builder.assembly.armDescriptionImports()...)
	}
	return imports
}

// stateValueImportNames lists the shared factory values every authored and
// assertion module imports from the state module.
func stateValueImportNames() []ImportName {
	names := []ImportName{{"$canHTML", "$canHTML"}, {"$canForm", "$canForm"}, {"$canFormActions", "$canFormActions"}, {"$canClock", "$canClock"}, {"$canRandom", "$canRandom"}, {"$canLog", "$canLog"}, {"$canIO", "$canIO"}, {"$canEnv", "$canEnv"}, {"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canChecks", "$canChecks"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}, {"$canBytes", "$canBytes"}, {"$canHTTPRequests", "$canHTTPRequests"}, {"$canHTTPResponses", "$canHTTPResponses"}, {"$canRouter", "$canRouter"}, {"$canServer", "$canServer"}}
	names = append(names, sqlStateValueImportNames()...)
	names = append(names, cryptoStateValueImportNames()...)
	names = append(names, utilitiesStateValueImportNames()...)
	names = append(names, fileStateValueImportNames()...)
	names = append(names, processStateValueImportNames()...)
	names = append(names, streamStateValueImportNames()...)
	names = append(names, websocketStateValueImportNames()...)
	names = append(names, cookiesStateValueImportNames()...)
	names = append(names, s3StateValueImportNames()...)
	names = append(names, markdownStateValueImportNames()...)
	names = append(names, fetchCatalogueValueImportNames()...)
	return append(names, browserCatalogueValueImportNames()...)
}

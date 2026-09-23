package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// cookiesOperationBindings maps the B1-09 cookie and CSRF operations to
// their state-module targets. Cookie handles are opaque validated
// wrappers; CSRF tokens cross as plain strings.
func cookiesOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.cookie@1::parse":     "$canCookies.parse",
		"can.std.cookie@1::get":       "$canCookies.get",
		"can.std.cookie@1::make":      "$canCookies.make",
		"can.std.cookie@1::serialize": "$canCookies.serialize",
		"can.std.cookie@1::expire":    "$canCookies.remove",
		"can.std.csrf@1::generate":    "$canCSRF.generate",
		"can.std.csrf@1::verify":      "$canCSRF.verify",
	}
	return bindingContribution{domain: "cookies", functions: functions}
}

// cookiesStateImports lists the cookie and CSRF factory modules the shared
// state module needs.
func (assembly *programAssembly) cookiesStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/cookies.ts", Names: []ImportName{{"createCookies", "$canCreateCookies"}, {"isCookieValue", "$canIsCookie"}}},
		{Target: runtime + "/platform/csrf.ts", Names: []ImportName{{"createCSRF", "$canCreateCSRF"}}},
	}
}

// cookiesStateValueImportNames lists the cookie and CSRF factory values
// authored and assertion modules import from the state module.
func cookiesStateValueImportNames() []ImportName {
	return []ImportName{{"$canCookies", "$canCookies"}, {"$canCSRF", "$canCSRF"}}
}

// declareCookiesState emits the cookie and CSRF factory bindings.
func (builder *stateBuilder) declareCookiesState() {
	builder.out.WriteString("export let $canCookies:ReturnType<typeof $canCreateCookies>;\n")
	builder.out.WriteString("export let $canCSRF:ReturnType<typeof $canCreateCSRF>;\n")
}

// initializeCookiesState constructs the cookie and CSRF factories inside
// the shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeCookiesState() {
	fmt.Fprintf(&builder.out, "$canCookies=$canCreateCookies($canDomain,{invalid:%s,collection:%s,pair:%s,some:%s,none:%s,samesiteStrict:%s,samesiteLax:%s,samesiteNone:%s});\n", quote(builder.numberIDs["can.std.cookie@1::invalid_cookie"]), quote(builder.numberIDs["can.std.cookie@1::collection"]), quote(builder.numberIDs["can.std.cookie@1::pair"]), quote(builder.optionIDs["can.std.option@1::some"]), quote(builder.optionIDs["can.std.option@1::none"]), quote(builder.numberIDs["can.std.cookie@1::strict"]), quote(builder.numberIDs["can.std.cookie@1::lax"]), quote(builder.numberIDs["can.std.cookie@1::none"]))
	fmt.Fprintf(&builder.out, "$canCSRF=$canCreateCSRF($canDomain,{invalid:%s});\n", quote(builder.numberIDs["can.std.csrf@1::invalid_config"]))
}

// emitCookiesKinds emits the opaque-handle kind table the domain predicate
// uses to recognize cookie values at boundaries.
func (builder *stateBuilder) emitCookiesKinds() error {
	cookiesKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		if typ.Declaration() == "can.std.cookie@1::cookie" {
			cookiesKinds[typ.Identity()] = "cookie"
		}
	}
	cookiesKindsJSON, err := json.Marshal(cookiesKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canCookiesKinds:Readonly<Record<string,string>>=%s;\n", cookiesKindsJSON)
	return nil
}

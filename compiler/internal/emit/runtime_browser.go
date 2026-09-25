package emit

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// browserOperationBindings maps the T22 bounded browser catalogue
// operations to their state-module targets. State operations specialize
// per concrete data type and bind through browserSpecializationBindings.
// Query/cancel bindings are the UP11 concrete call interface consumed by
// UP13's platform adapter; the method names below are the contract.
func browserOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.browser@1::mount":            "$canBrowser.mount",
		"can.std.browser@1::root":             "$canBrowser.root",
		"can.std.browser@1::open_view":        "$canBrowser.openView",
		"can.std.browser@1::dispose_view":     "$canBrowser.disposeView",
		"can.std.browser@1::dispose_app":      "$canBrowser.disposeApp",
		"can.std.browser@1::create_element":   "$canBrowser.createElement",
		"can.std.browser@1::create_text":      "$canBrowser.createText",
		"can.std.browser@1::set_text":         "$canBrowser.setText",
		"can.std.browser@1::set_attribute":    "$canBrowser.setAttribute",
		"can.std.browser@1::remove_attribute": "$canBrowser.removeAttribute",
		"can.std.browser@1::append_child":     "$canBrowser.appendChild",
		"can.std.browser@1::remove_node":      "$canBrowser.removeNode",
		"can.std.browser@1::focus":            "$canBrowser.focus",
		"can.std.browser@1::on_event":         "$canBrowser.onEvent",
		"can.std.browser@1::set_timeout":      "$canBrowser.setTimeout",
		"can.std.browser@1::query_parameter":  "$canBrowser.queryParameter",
		"can.std.browser@1::on_cancel_key":    "$canBrowser.onCancelKey",
		"can.std.browser@1::on_cancel_event":  "$canBrowser.onCancelEvent",
	}
	return bindingContribution{domain: "browser", functions: functions}
}

// browserSpecializationBindings assigns deterministic runtime names to the
// checked browser state specializations of this program. Each key binds
// to its per-type state value, which brands tokens and snapshot records
// with the concrete identities.
func (assembly *programAssembly) browserSpecializationBindings() bindingContribution {
	assembly.browserStateIDs = make([]string, 0, len(assembly.program.BrowserStates))
	for id := range assembly.program.BrowserStates {
		assembly.browserStateIDs = append(assembly.browserStateIDs, id)
	}
	sort.Strings(assembly.browserStateIDs)
	assembly.browserStateNames = map[string]string{}
	functions := map[string]string{}
	for i, id := range assembly.browserStateIDs {
		name := fmt.Sprintf("$canBrowserState%d", i)
		assembly.browserStateNames[id] = name
		var method string
		switch assembly.program.BrowserStates[id].Operation {
		case "can.std.browser@1::create_state":
			method = "createState"
		case "can.std.browser@1::read_state":
			method = "readState"
		case "can.std.browser@1::replace_state":
			method = "replaceState"
		}
		functions[id] = name + "." + method
	}
	return bindingContribution{domain: "browser-states", functions: functions}
}

// browserStateImports lists the browser factory module the shared state
// module needs in both profiles.
func (assembly *programAssembly) browserStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/browser.ts", Names: []ImportName{{"createBrowser", "$canCreateBrowser"}, {"createBrowserState", "$canCreateBrowserState"}, {"isBrowserValue", "$canIsBrowser"}, {"isBrowserStateValue", "$canIsBrowserState"}}},
	}
}

// browserCatalogueValueImportNames lists the browser factory value
// authored and assertion modules import from the state module.
func browserCatalogueValueImportNames() []ImportName {
	return []ImportName{{"$canBrowser", "$canBrowser"}}
}

// declareBrowserState emits the browser factory binding.
func (builder *stateBuilder) declareBrowserState() {
	builder.out.WriteString("export let $canBrowser:ReturnType<typeof $canCreateBrowser>;\n")
}

// declareBrowserStateSpecializations emits one state factory binding per
// checked browser state specialization used by the program.
func (builder *stateBuilder) declareBrowserStateSpecializations() {
	for _, id := range builder.assembly.browserStateIDs {
		fmt.Fprintf(&builder.out, "export let %s: ReturnType<typeof $canCreateBrowserState>;\n", builder.assembly.browserStateNames[id])
	}
}

// browserQueryOptionIDs seals the option::value<str> leaf identities the
// query_parameter adapter brands its some/none results with. Like the
// env/cookies adapters, the compiler resolves the concrete identities
// from the operation's own checked result; the none leaf takes no type
// arguments but still carries a digest identity.
func browserQueryOptionIDs(builder *stateBuilder) (some, none string) {
	intrinsic := builder.assembly.program.Intrinsics["can.std.browser@1::query_parameter"]
	if intrinsic == nil {
		return "", ""
	}
	for _, leaf := range intrinsic.Result().Leaves() {
		switch leaf.Declaration() {
		case "can.std.option@1::some":
			some = leaf.Identity()
		case "can.std.option@1::none":
			none = leaf.Identity()
		}
	}
	return some, none
}

// initializeBrowserState constructs the browser factory inside the shared
// initializer, after the domain runtime exists. The factory resolves the
// live document lazily, so Bun executions fail closed with missing_root
// while browser bundles bind the real document. The invalidQuery contract
// is the UP11 addition for query_parameter; UP13 consumes it in the
// platform adapter together with the sealed some/none option leaves.
func (builder *stateBuilder) initializeBrowserState() {
	some, none := browserQueryOptionIDs(builder)
	fmt.Fprintf(&builder.out, "$canBrowser=$canCreateBrowser($canDomain,{missingRoot:%s,disposed:%s,rejected:%s,event:%s,invalidQuery:%s,some:%s,none:%s});\n", quote(builder.numberIDs["can.std.browser@1::missing_root"]), quote(builder.numberIDs["can.std.browser@1::disposed"]), quote(builder.numberIDs["can.std.browser@1::rejected"]), quote(builder.numberIDs["can.std.browser@1::event"]), quote(builder.numberIDs["can.std.browser@1::invalid_query"]), quote(some), quote(none))
}

// initializeBrowserStateSpecializations constructs the per-type browser
// state values inside the shared initializer.
func (builder *stateBuilder) initializeBrowserStateSpecializations() {
	for _, id := range builder.assembly.browserStateIDs {
		special := builder.assembly.program.BrowserStates[id]
		fmt.Fprintf(&builder.out, "%s=$canCreateBrowserState($canDomain,{disposed:%s,stale:%s,state:%s,snapshot:%s});\n", builder.assembly.browserStateNames[id], quote(builder.numberIDs["can.std.browser@1::disposed"]), quote(builder.numberIDs["can.std.browser@1::stale_version"]), quote(special.State.Identity()), quote(special.Snapshot.Identity()))
	}
}

// emitBrowserKinds emits the opaque-handle kind table the domain predicate
// uses to recognize app, view and node values at boundaries. State
// identities stay out of the kinds map: state tokens compare their
// concrete creation identity exactly, like maps.
func (builder *stateBuilder) emitBrowserKinds() error {
	browserKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.browser@1::app":
			browserKinds[typ.Identity()] = "app"
		case "can.std.browser@1::view":
			browserKinds[typ.Identity()] = "view"
		case "can.std.browser@1::node":
			browserKinds[typ.Identity()] = "node"
		}
	}
	browserKindsJSON, err := json.Marshal(browserKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canBrowserKinds:Readonly<Record<string,string>>=%s;\n", browserKindsJSON)
	return nil
}

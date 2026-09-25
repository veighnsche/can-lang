package emit

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// BrowserModules lowers a capability-gated program to the browser profile:
// the trimmed state module, authored modules without server edges, and the
// distinct main-thread root. The browser capability gate runs first so
// direct callers fail closed even when they bypass the driver.
func BrowserModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	if err := browser.CheckProgram(program); err != nil {
		return nil, err
	}
	return browserModules(program, runtime, dependencies)
}

// browserModules mirrors programModules with the browser profile selected.
// Assertion roots never ship in a browser generation: they execute under
// Bun during the verified build. Production emission prunes to the entry
// closure; the checker retains full source checks.
func browserModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	if program == nil || program.Entry == nil || program.Entry.Region == nil {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		return nil, err
	}
	assembly.browser = true
	pruneBrowserAssembly(assembly, reachableBrowserNodes(program))
	if err := assertBrowserAssembly(assembly); err != nil {
		return nil, err
	}
	state, assetFiles, err := emitStateModule(assembly, runtime)
	if err != nil {
		return nil, err
	}
	modules := []Module{state}
	authored, err := emitAuthoredModules(assembly, runtime)
	if err != nil {
		return nil, err
	}
	modules = append(modules, authored...)
	modules = append(modules, emitBrowserEntry(assembly, runtime))
	artifacts, err := Modules(modules, dependencies...)
	if err != nil {
		return nil, err
	}
	return append(artifacts, assetFiles...), nil
}

// assertBrowserAssembly is defense in depth behind the capability gate: the
// trimmed browser state module declares no server factory, so any server
// specialization, connection or native reference would lower to a dangling
// binding instead of failing here.
func assertBrowserAssembly(assembly *programAssembly) error {
	program := assembly.program
	if assembly.fetches || assembly.llms || assembly.judges {
		return fmt.Errorf("browser emission admits no native AI requests")
	}
	if len(assembly.sqlIDs)+len(assembly.txIDs)+len(assembly.connectionIDs)+len(assembly.nativeIDs) != 0 {
		return fmt.Errorf("browser emission admits no SQL, connection or native references")
	}
	if len(program.Natives)+len(program.Connections)+len(program.SQL)+len(program.SQLs)+len(program.Transactions)+len(program.Streams) != 0 {
		return fmt.Errorf("browser emission requires a capability-gated program")
	}
	return nil
}

// emitBrowserEntry builds the distinct main-thread root: the compiler-owned
// once-only DOM-ready startup for the checked zero-argument main. It
// initializes Can state inside the explicit owner root and reports a startup
// fault once through the sealed reporter; no test-authored boot module
// invokes main. The diagnostic table below is the UP11 placeholder shape
// (empty sealed index); UP15 seals it with checked source locations and
// publishes it as a verified asset. No host process reference appears.
func emitBrowserEntry(assembly *programAssembly, runtime string) Module {
	main := assembly.program.Entry
	body := "export const BROWSER_PROFILE = " + quote(browser.Profile) + ";\n" +
		"const $canTable = Object.freeze({index: Object.freeze({schemaVersion: 1, kind: \"can.source-index\", sources: Object.freeze([]), modules: Object.freeze([])}), maps: Object.freeze({})});\n" +
		"let $canStarted = false;\n" +
		"export async function $canBrowserMain(): Promise<void> {\n" +
		"if ($canStarted) return;\n" +
		"$canStarted = true;\n" +
		"await $canRunBrowserEntry({table: $canTable, main: async ($canCtx) => {await $canVerifyErrorPlan($canErrorPlan);$canInitialize();\n" +
		"return $canMain($canCtx);\n" +
		"}});\n" +
		"}\n" +
		"void $canBrowserMain();\n"
	return Module{Path: browser.BrowserEntry, Imports: []ModuleImport{
		{Target: programStatePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}, {"$canErrorPlan", "$canErrorPlan"}}},
		{Target: runtime + "/browser/entry.ts", Names: []ImportName{{"runBrowserEntry", "$canRunBrowserEntry"}}},
		{Target: runtime + "/browser/domain.ts", Names: []ImportName{{"verifyErrorPlan", "$canVerifyErrorPlan"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{assembly.functions[main.Identity()], "$canMain"}}},
	}, Body: body}
}

// browserStateValueImportNames lists the shared factory values browser
// authored modules import from the trimmed state module: the server
// factories (SQL, crypto, files, processes, streams, websockets, S3,
// IO, environment, CLI and server) are absent.
func browserStateValueImportNames() []ImportName {
	names := []ImportName{{"$canHTML", "$canHTML"}, {"$canForm", "$canForm"}, {"$canFormActions", "$canFormActions"}, {"$canClock", "$canClock"}, {"$canRandom", "$canRandom"}, {"$canLog", "$canLog"}, {"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canChecks", "$canChecks"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canBytes", "$canBytes"}, {"$canHTTPRequests", "$canHTTPRequests"}, {"$canHTTPResponses", "$canHTTPResponses"}, {"$canRouter", "$canRouter"}}
	names = append(names, utilitiesStateValueImportNames()...)
	names = append(names, cookiesStateValueImportNames()...)
	names = append(names, markdownStateValueImportNames()...)
	names = append(names, fetchCatalogueValueImportNames()...)
	return append(names, browserCatalogueValueImportNames()...)
}

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
// Bun during the verified build.
func browserModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	if program == nil || program.Entry == nil || program.Entry.Region == nil {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		return nil, err
	}
	assembly.browser = true
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

// emitBrowserEntry builds the distinct main-thread root: an inert module
// exporting the browser profile marker and the checked main through the
// same configure-then-initialize lifecycle as the Bun entry, without any
// host process reference.
func emitBrowserEntry(assembly *programAssembly, runtime string) Module {
	main := assembly.program.Entry
	return Module{Path: browser.BrowserEntry, Imports: []ModuleImport{
		{Target: programStatePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{assembly.functions[main.Identity()], "$canMain"}}},
	}, Body: "export const BROWSER_PROFILE = " + quote(browser.Profile) + ";\nexport async function $canBrowserMain(...$canArgs: Parameters<typeof $canMain>): Promise<Awaited<ReturnType<typeof $canMain>>> {\n$canConfigureDiagnostics(import.meta.url);\n$canInitialize();\nreturn $canMain(...$canArgs);\n}\n"}
}

// browserStateValueImportNames lists the shared factory values browser
// authored modules import from the trimmed state module: the server
// factories (SQL, crypto, files, processes, streams, websockets, S3,
// IO, environment, CLI and server) are absent.
func browserStateValueImportNames() []ImportName {
	names := []ImportName{{"$canHTML", "$canHTML"}, {"$canClock", "$canClock"}, {"$canRandom", "$canRandom"}, {"$canLog", "$canLog"}, {"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canChecks", "$canChecks"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canBytes", "$canBytes"}, {"$canHTTPRequests", "$canHTTPRequests"}, {"$canHTTPResponses", "$canHTTPResponses"}, {"$canRouter", "$canRouter"}}
	names = append(names, utilitiesStateValueImportNames()...)
	names = append(names, cookiesStateValueImportNames()...)
	return append(names, markdownStateValueImportNames()...)
}

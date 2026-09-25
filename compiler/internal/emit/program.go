package emit

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// ProgramModules keeps every source module inert. One shared initialization
// function creates error evidence and evaluates the ordered top-level values;
// the entry supervisor invokes it before calling the checked main region.
func ProgramModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, false, nil)
}

// AssertionModules emits the same program with assertion roots instead of a
// production entry point.
func AssertionModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, true, nil)
}

// ProgramModulesPaired emits a production program with one verified browser
// pairing bound into its asset table. A nil pairing emits unpaired output.
func ProgramModulesPaired(program *check.Program, runtime string, dependencies []ir.Artifact, pairing *BrowserPairing) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, false, pairing)
}

// AssertionModulesPaired emits assertion roots with one verified browser
// pairing bound into their asset table. A nil pairing emits unpaired output.
func AssertionModulesPaired(program *check.Program, runtime string, dependencies []ir.Artifact, pairing *BrowserPairing) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, true, pairing)
}

// programModules assembles a program from its shared state module, authored
// modules and entry artifacts in that fixed order. Domain bindings live in
// runtime_*.go, the state module in program_state.go, authored modules in
// program_modules.go and entries in program_entry.go; no feature body belongs
// here.
func programModules(program *check.Program, runtime string, dependencies []ir.Artifact, assertions bool, pairing *BrowserPairing) ([]ir.Artifact, error) {
	if program == nil || (!assertions && (program.Entry == nil || program.Entry.Region == nil)) {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	assembly, err := assembleProgramBindings(program)
	if err != nil {
		return nil, err
	}
	assembly.browserPairing = pairing
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
	if assertions {
		entries, err := emitAssertionModules(assembly, runtime)
		if err != nil {
			return nil, err
		}
		modules = append(modules, entries...)
		artifacts, err := Modules(modules, dependencies...)
		if err != nil {
			return nil, err
		}
		return append(artifacts, assetFiles...), nil
	}
	modules = append(modules, emitExecutableEntry(assembly, runtime))
	artifacts, err := Modules(modules, dependencies...)
	if err != nil {
		return nil, err
	}
	return append(artifacts, assetFiles...), nil
}

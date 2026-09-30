package check

import (
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"maps"
	"reflect"
)

// unitCheckpoint isolates additions produced while checking one executable
// unit. Sealed types are immutable; a specializer fork holds new type evidence
// until the unit succeeds. Previously published maps retain their identity for
// contexts that already reference them.
func (c *programChecker) unitCheckpoint() func() {
	if !c.recovering || c.program == nil || c.specializer == nil {
		return func() {}
	}
	prior := *c.program
	originalSpecializer := c.specializer
	c.specializer = c.specializer.Fork()
	restores := []func(){
		saveMap(&c.bindings), saveMap(&c.callables), saveMap(&c.variadic), saveMap(&c.templates), saveMap(&c.instances),
		saveMap(&c.codecs), saveMap(&c.codecParts), saveMap(&c.https), saveMap(&c.httpParts), saveMap(&c.forms), saveMap(&c.fetches),
		saveMap(&c.program.Intrinsics), saveMap(&c.program.Collections), saveMap(&c.program.BrowserStates), saveMap(&c.program.Streams),
		saveMap(&c.program.Codecs), saveMap(&c.program.HTTPs), saveMap(&c.program.Forms), saveMap(&c.program.Fetches),
		saveMap(&c.program.SQLs), saveMap(&c.program.Transactions), saveMap(&c.program.Connections),
	}
	for _, annotations := range c.annotations {
		local := annotations
		restores = append(restores, saveMap(&local))
	}
	for _, fn := range c.program.Functions {
		value := *fn
		value.Requests = append([]string(nil), fn.Requests...)
		restores = append(restores, func() { *fn = value })
	}
	for _, native := range c.program.Natives {
		value := *native
		restores = append(restores, func() { *native = value })
	}
	for _, special := range c.codecs {
		value := *special
		restores = append(restores, func() { *special = value })
	}
	for _, special := range c.https {
		value := *special
		restores = append(restores, func() { *special = value })
	}
	for _, special := range c.forms {
		value := *special
		restores = append(restores, func() { *special = value })
	}
	for _, special := range c.fetches {
		value := *special
		restores = append(restores, func() { *special = value })
	}
	sites := len(c.sqlSites)
	return func() {
		for _, restore := range restores {
			restore()
		}
		*c.program = prior
		c.specializer = originalSpecializer
		c.sqlSites = c.sqlSites[:sites]
	}
}

func saveMap[K comparable, V any](target *map[K]V) func() {
	original := *target
	saved := maps.Clone(original)
	return func() {
		clear(original)
		maps.Copy(original, saved)
		*target = original
	}
}

// blockDependentRegions closes invalid executable evidence over canonical IR
// call/reference identities, including references created by specialization.
func (c *programChecker) blockDependentRegions() {
	invalid := map[string]bool{}
	dependencies := map[string][]string{}
	clearNative := func(native *NativeDeclaration) {
		native.Regions = nil
		native.Registrations = nil
		native.Fetch, native.LLM, native.Judge, native.Question = nil, nil, nil, nil
		native.ArmDescription, native.Wrapper = nil, nil
	}
	for _, fn := range c.program.Functions {
		dependencies[fn.Identity()] = irDependencies(reflect.ValueOf(fn.Region))
		if fn.Region == nil || c.world.Invalid[fn.Symbol.Declaration] != nil {
			fn.Region = nil
			invalid[fn.Identity()] = true
		}
	}
	for _, native := range c.program.Natives {
		id := native.Symbol.ID
		for _, plan := range []any{native.Regions, native.Registrations, native.Fetch, native.LLM, native.Judge, native.Question, native.ArmDescription} {
			dependencies[id] = append(dependencies[id], irDependencies(reflect.ValueOf(plan))...)
		}
		if native.Wrapper != nil {
			dependencies[id] = append(dependencies[id], native.Wrapper.Base, native.Wrapper.Root)
			for _, rule := range native.Wrapper.Rules {
				dependencies[id] = append(dependencies[id], irDependencies(reflect.ValueOf(rule.Region))...)
			}
		}
		if err := c.world.Invalid[native.Symbol.Declaration]; err != nil {
			native.Symbol.Invalid = err
		}
		if native.Symbol.Invalid != nil {
			invalid[id] = true
			clearNative(native)
		}
	}
	blockedBy := func(id string) string {
		if invalid[id] {
			return ""
		}
		for _, dep := range dependencies[id] {
			if invalid[dep] {
				return dep
			}
		}
		return ""
	}
	for changed := true; changed; {
		changed = false
		for _, fn := range c.program.Functions {
			if dep := blockedBy(fn.Identity()); dep != "" {
				invalid[fn.Identity()] = true
				changed = true
				fn.Region = nil
				c.world.Invalid[fn.Symbol.Declaration] = &source.BlockedError{Dependency: dep}
			}
		}
		for _, native := range c.program.Natives {
			if dep := blockedBy(native.Symbol.ID); dep != "" {
				invalid[native.Symbol.ID] = true
				changed = true
				native.Symbol.Invalid = &source.BlockedError{Dependency: dep}
				c.world.Invalid[native.Symbol.Declaration] = native.Symbol.Invalid
				clearNative(native)
			}
		}
	}
	validAssertions := c.program.Assertions[:0]
	for _, assertion := range c.program.Assertions {
		valid := !invalid[assertion.Root.Declaration]
		for _, dep := range irDependencies(reflect.ValueOf(assertion)) {
			if invalid[dep] {
				valid = false
				break
			}
		}
		if valid {
			validAssertions = append(validAssertions, assertion)
		}
	}
	c.program.Assertions = validAssertions
}

func irDependencies(value reflect.Value) []string {
	var result []string
	visited := map[any]bool{}
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface {
			if !v.IsNil() {
				visit(v.Elem())
			}
			return
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() || v.Type().Elem().PkgPath() != "github.com/veighnsche/can-lang/compiler/internal/ir" {
				return
			}
			pointer := v.Interface()
			if visited[pointer] {
				return
			}
			visited[pointer] = true
			visit(v.Elem())
			return
		}
		if v.Kind() == reflect.Struct {
			if v.Type().PkgPath() != "github.com/veighnsche/can-lang/compiler/internal/ir" {
				return
			}
			switch node := v.Interface().(type) {
			case ir.Expression:
				if node.Kind == ir.Call && node.Text != "" {
					result = append(result, node.Text)
				}
			case ir.JudgeRegistration:
				if node.Question != "" {
					result = append(result, node.Question)
				}
			case ir.InvocationStep:
				if node.Identity != "" {
					result = append(result, node.Identity)
				}
			case ir.Callable:
				if node.Target != "" {
					result = append(result, node.Target)
				}
			}
			for i := 0; i < v.NumField(); i++ {
				if v.Field(i).CanInterface() {
					visit(v.Field(i))
				}
			}
		} else if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	visit(value)
	return result
}

// childCheckpoint prevents failed arguments and arms from leaking local or
// specialization evidence into their independently checkable siblings.
func (c *regionChecker) childCheckpoint() func() {
	if !c.context.Recover {
		return func() {}
	}
	serial, aggregate := c.serial, c.aggregate
	var restoreAggregate func()
	if aggregate != nil {
		prior := *aggregate
		restoreAggregate = func() { *aggregate = prior }
	}
	var restoreRegion func()
	if c.region != nil {
		region, prior := c.region, *c.region
		restoreRegion = func() { *region = prior }
	}
	restores := []func(){saveMap(&c.locals), saveMap(&c.uses.Names), saveMap(&c.uses.Captures)}
	restores = append(restores, func() {
		c.serial, c.aggregate = serial, aggregate
		if restoreAggregate != nil {
			restoreAggregate()
		}
		if restoreRegion != nil {
			restoreRegion()
		}
	})
	if c.context.Checkpoint != nil {
		restores = append(restores, c.context.Checkpoint())
	}
	return func() {
		for _, restore := range restores {
			restore()
		}
	}
}

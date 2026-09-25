package check

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Exported generic declarations check once under symbolic type variables, in
// addition to the concrete per-instance checks every reachable specialization
// still receives. A type variable is opaque in the declaration body: values
// of variable type only pass through (bind, return, construct composites,
// supply matching variable slots), while every representation-requiring
// operation must arrive as an explicit named callable input or as a callable
// held by a named dictionary-record input. Arithmetic, ordering, equality,
// field access, method calls, indexing, slicing and matching on a bare
// variable all fail at the declaring file, so a dependency body edit cannot
// silently narrow the contract its consumers rely on.
//
// A public generic may also call a public, symbolically validated generic by
// canonical declaration identity: the caller's symbolic type expressions are
// substituted into the callee's checked signature, and ordinary argument,
// result and finite emits obligations are checked against that substituted
// contract. The callee body is never rechecked under the caller's variables.
// Unexported (local) generic declarations keep template semantics: only
// their concrete instances check, and a private template is never a symbolic
// callee. There is no implicit trait search and no error-set type parameter;
// a bare variable in `emits` is not a nominal error.
//
// Public declarations validate in dependency order over strongly connected
// components: a component's members check under provisional internal
// contracts and no proof is committed until every member body and internal
// call succeeds. On every internal cyclic edge each type argument must be a
// bare caller formal or a fully closed type; permutation, duplication and
// dropping of bare formals stay finite, while any constructor around an
// opaque formal rejects with the call chain. Acyclic nested calls such as
// G<box<T>> are legal once G's proof is committed. This bounds declaration
// proof and concrete instance discovery, not runtime recursion depth.
func (c *programChecker) checkExportedGenerics(files []*resolve.File) error {
	var units []*symbolicUnit
	for _, file := range files {
		for _, node := range file.Source.Syntax.Declarations {
			d, ok := node.(*syntax.FunctionDecl)
			if !ok || len(d.Parameters) == 0 {
				continue
			}
			symbol := file.Package.Scope.Symbols[d.Name.Text]
			if symbol == nil || !symbol.Public {
				continue
			}
			units = append(units, &symbolicUnit{file: file, symbol: symbol, declaration: d})
		}
	}
	sort.Slice(units, func(i, j int) bool { return units[i].symbol.ID < units[j].symbol.ID })
	c.symbolicComponent = map[string]int{}
	c.symbolicProofs = map[string]bool{}
	c.symbolicScan = nil
	byID := map[string]*symbolicUnit{}
	for _, unit := range units {
		byID[unit.symbol.ID] = unit
	}
	graph := map[string][]string{}
	for _, unit := range units {
		seen := map[string]bool{}
		for _, edge := range c.scanSymbolicEdges(unit) {
			target, ok := byID[edge.callee]
			if !ok || target == unit {
				continue
			}
			c.symbolicScan = append(c.symbolicScan, edge)
			if !seen[edge.callee] {
				seen[edge.callee] = true
				graph[unit.symbol.ID] = append(graph[unit.symbol.ID], edge.callee)
			}
		}
	}
	return c.checkSymbolicComponents(units, graph)
}

// symbolicUnit is one public generic declaration awaiting proof.
type symbolicUnit struct {
	file        *resolve.File
	symbol      *resolve.Symbol
	declaration *syntax.FunctionDecl
}

// symbolicScanEdge is one syntactic call edge between public generic
// declarations, resolved by canonical identity for the component graph.
type symbolicScanEdge struct {
	caller string
	callee string
	site   string
}

// scanSymbolicEdges resolves the named generic callees a public generic body
// may call, by canonical declaration identity. Only direct named calls and
// callable references resolve syntactically; receiver-typed method
// applications cannot resolve without a receiver type and stay fail-closed
// at check time (an uncommitted callee from another component rejects).
func (c *programChecker) scanSymbolicEdges(unit *symbolicUnit) []symbolicScanEdge {
	var edges []symbolicScanEdge
	scope := c.world.Functions[unit.declaration]
	visit := func(name syntax.QualifiedName, span source.Span) {
		symbol, err := unit.file.Lookup(scope, name, resolve.CallUse)
		if err != nil {
			return
		}
		declaration, ok := symbol.Declaration.(*syntax.FunctionDecl)
		if !ok || len(symbol.Parameters) == 0 || !symbol.Public {
			return
		}
		_ = declaration
		edges = append(edges, symbolicScanEdge{caller: unit.symbol.ID, callee: symbol.ID, site: unit.file.Source.Syntax.Source.Name() + " byte " + fmt.Sprint(span.Start)})
	}
	var walk func(value reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			switch node := value.Interface().(type) {
			case *syntax.CallExpr:
				if callee, ok := node.Invocation.Callee.(*syntax.NameExpr); ok {
					visit(callee.Name, node.Invocation.Span)
				}
			case *syntax.ReferenceExpr:
				if callee, ok := node.Callee.(*syntax.NameExpr); ok {
					visit(callee.Name, node.Span)
				}
			}
			walk(value.Elem())
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				walk(value.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(unit.declaration.Body))
	return edges
}

// symbolicOrder groups units into strongly connected components and orders
// components so callees validate before callers. Tarjan and the dependency
// walk both iterate in symbol-ID order, so the proof order is deterministic.
func symbolicOrder(units []*symbolicUnit, graph map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var components [][]string
	counter := 0
	var visit func(id string)
	visit = func(id string) {
		index[id] = counter
		low[id] = counter
		counter++
		stack = append(stack, id)
		onStack[id] = true
		neighbors := append([]string(nil), graph[id]...)
		sort.Strings(neighbors)
		for _, next := range neighbors {
			if _, seen := index[next]; !seen {
				visit(next)
				if low[next] < low[id] {
					low[id] = low[next]
				}
			} else if onStack[next] && index[next] < low[id] {
				low[id] = index[next]
			}
		}
		if low[id] == index[id] {
			var component []string
			for {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[top] = false
				component = append(component, top)
				if top == id {
					break
				}
			}
			sort.Strings(component)
			components = append(components, component)
		}
	}
	for _, unit := range units {
		if _, seen := index[unit.symbol.ID]; !seen {
			visit(unit.symbol.ID)
		}
	}
	member := map[string]int{}
	for i, component := range components {
		for _, id := range component {
			member[id] = i
		}
	}
	depends := make([][]int, len(components))
	for from, neighbors := range graph {
		for _, to := range neighbors {
			a, b := member[from], member[to]
			if a == b {
				continue
			}
			found := false
			for _, have := range depends[a] {
				if have == b {
					found = true
					break
				}
			}
			if !found {
				depends[a] = append(depends[a], b)
			}
		}
	}
	for i := range depends {
		sort.Ints(depends[i])
	}
	order := []int{}
	done := map[int]bool{}
	var emit func(i int)
	emit = func(i int) {
		if done[i] {
			return
		}
		done[i] = true
		for _, next := range depends[i] {
			emit(next)
		}
		order = append(order, i)
	}
	keys := make([]int, len(components))
	for i := range components {
		keys[i] = i
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.Join(components[keys[i]], "\x00") < strings.Join(components[keys[j]], "\x00")
	})
	for _, key := range keys {
		emit(key)
	}
	ordered := make([][]string, len(components))
	for i, key := range order {
		ordered[i] = components[key]
	}
	return ordered
}

// checkSymbolicComponents validates each dependency component atomically:
// provisional contracts are installed for every member, every body checks,
// and proofs commit only when the entire component passes. The first
// component failure aborts the proof with its root diagnosis; later
// dependents never observe a partial proof.
func (c *programChecker) checkSymbolicComponents(units []*symbolicUnit, graph map[string][]string) error {
	byID := map[string]*symbolicUnit{}
	for _, unit := range units {
		byID[unit.symbol.ID] = unit
	}
	for number, component := range symbolicOrder(units, graph) {
		for _, id := range component {
			c.symbolicComponent[id] = number
		}
		functions := map[string]*ProgramFunction{}
		for _, id := range component {
			unit := byID[id]
			fn, err := c.installSymbolicContract(unit.file, unit.symbol, unit.declaration)
			if err != nil {
				return err
			}
			functions[id] = fn
		}
		saved := c.current
		for _, id := range component {
			unit := byID[id]
			c.current = functions[id]
			if err := c.checkSymbolicBody(unit.file, unit.symbol, unit.declaration, functions[id]); err != nil {
				c.current = saved
				return err
			}
		}
		c.current = saved
		for _, id := range component {
			c.symbolicProofs[id] = true
		}
	}
	return nil
}

// installSymbolicContract resolves a public generic's own symbolic signature
// and publishes its provisional declaration contract. The symbolic identity
// keys declaration-only contracts. It is never a specialization key, never
// enters the instance cache or emitted functions, and its checked region is
// discarded after validation.
func (c *programChecker) installSymbolicContract(file *resolve.File, symbol *resolve.Symbol, d *syntax.FunctionDecl) (*ProgramFunction, error) {
	defFile := file.Source.Syntax.Source.Name()
	parameters := map[string]*types.Type{}
	arguments := make([]*types.Type, len(symbol.Parameters))
	for i, name := range symbol.Parameters {
		param, err := types.SymbolicParameter(symbol.ID, name)
		if err != nil {
			return nil, source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
		}
		parameters[name] = param
		arguments[i] = param
	}
	signature, descriptor, fields, err := genericSignature(d)
	if err != nil {
		return nil, source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	identity := symbol.ID + "/symbolic"
	contract, err := c.specializer.Resolve(file, signature, parameters, true)
	if err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		return nil, source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	descriptor.Contract = contract
	c.callables[identity] = descriptor
	c.bindings[identity] = contract
	for i, field := range fields {
		c.bindings[identity+"/input/"+field.Name.Text] = contract.Inputs()[i]
	}
	if len(d.Inputs) > 0 && d.Inputs[len(d.Inputs)-1].Variadic {
		c.variadic[identity] = true
	}
	return &ProgramFunction{Symbol: symbol, Instance: identity, TypeArguments: arguments, Parameters: parameters}, nil
}

func (c *programChecker) checkSymbolicBody(file *resolve.File, symbol *resolve.Symbol, d *syntax.FunctionDecl, fn *ProgramFunction) error {
	if err := c.gatherBody(file, reflect.ValueOf(d.Body)); err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		err = source.Relate(file.Source.Syntax.Source.Name(), d.DeclSpan(), "exported generic declared here", err)
		return stampCode(err, "CAN-CHECK-EXPORTED-GENERIC")
	}
	context, err := c.functionContext(fn)
	if err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		return source.LocateCode(file.Source.Syntax.Source.Name(), d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	if _, err := CheckRegion(context, d.Body); err != nil {
		err = fmt.Errorf("exported generic function %s requires operations on type variables to be explicit named callable or dictionary inputs: %w", symbol.ID, err)
		err = source.Relate(file.Source.Syntax.Source.Name(), d.DeclSpan(), "exported generic declared here", err)
		return stampCode(err, "CAN-CHECK-EXPORTED-GENERIC")
	}
	return nil
}

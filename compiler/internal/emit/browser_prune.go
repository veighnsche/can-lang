package emit

import (
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// browserReachable collects the production closure reachable from the browser
// entry through calls, callbacks, coordination and initializers. The checker
// retains full source checks; the browser emitter prunes unreachable
// declarations, callbacks, concrete specializations and initializers so
// shared unused form/server code adds no browser runtime edge. Bun emission
// keeps every checked declaration.
type browserReachable struct {
	functions       map[string]bool
	specializations map[string]bool
	initializers    map[string]bool
}

func reachableBrowserNodes(program *check.Program) *browserReachable {
	reached := &browserReachable{
		functions:       map[string]bool{},
		specializations: map[string]bool{},
		initializers:    map[string]bool{},
	}
	if program == nil || program.Entry == nil {
		return reached
	}
	functions := map[string]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		functions[fn.Identity()] = fn
	}
	initializers := map[string]ir.Initializer{}
	for _, init := range program.Initializers {
		initializers[init.Identity] = init
	}
	walker := &browserReachWalker{
		program:      program,
		functions:    functions,
		initializers: initializers,
		reached:      reached,
		scopes:       map[string]bool{},
	}
	walker.function(program.Entry)
	for _, init := range program.Initializers {
		if reached.initializers[init.Identity] && init.Value != nil {
			walker.expression(init.Value)
		}
	}
	return reached
}

type browserReachWalker struct {
	program      *check.Program
	functions    map[string]*check.ProgramFunction
	initializers map[string]ir.Initializer
	reached      *browserReachable
	scopes       map[string]bool
}

func (w *browserReachWalker) function(fn *check.ProgramFunction) {
	if fn == nil || w.reached.functions[fn.Identity()] {
		return
	}
	w.reached.functions[fn.Identity()] = true
	w.region(fn.Region)
}

func (w *browserReachWalker) operation(identity string) {
	if identity == "" {
		return
	}
	if fn := w.functions[identity]; fn != nil {
		w.function(fn)
		return
	}
	if w.program.Codecs[identity] != nil || w.program.HTTPs[identity] != nil ||
		w.program.Forms[identity] != nil || w.program.Fetches[identity] != nil ||
		w.program.BrowserStates[identity] != nil || w.program.Collections[identity] != nil {
		w.reached.specializations[identity] = true
		return
	}
}

func (w *browserReachWalker) region(region *ir.Region) {
	if region == nil {
		return
	}
	if region.ID != "" {
		if w.scopes[region.ID] {
			return
		}
		w.scopes[region.ID] = true
	}
	w.block(region.Body)
}

func (w *browserReachWalker) block(block *ir.Block) {
	if block == nil {
		return
	}
	for i := range block.Steps {
		w.statement(&block.Steps[i])
	}
	w.completion(block.Terminal)
}

func (w *browserReachWalker) statement(step *ir.Statement) {
	if step == nil {
		return
	}
	if step.Coordination != nil {
		w.coordination(step.Coordination)
	}
	if step.Value != nil {
		w.expression(step.Value)
	}
	w.invocation(step.Call)
}

func (w *browserReachWalker) completion(done *ir.Completion) {
	if done == nil {
		return
	}
	if done.Value != nil {
		w.expression(done.Value)
	}
	if done.Call != nil {
		w.invocation(done.Call)
	}
	if done.Block != nil {
		w.block(done.Block)
	}
	w.match(done.Match)
}

func (w *browserReachWalker) match(selected *ir.Match) {
	if selected == nil {
		return
	}
	for _, value := range selected.Values {
		w.expression(value)
	}
	w.invocation(selected.Call)
	for i := range selected.Arms {
		arm := &selected.Arms[i]
		if arm.Value != nil {
			w.expression(arm.Value)
		}
		w.completion(arm.Body)
	}
}

func (w *browserReachWalker) expression(node *ir.Expression) {
	if node == nil {
		return
	}
	if node.Kind == ir.Binding && node.Text != "" {
		if _, ok := w.initializers[node.Text]; ok {
			if !w.reached.initializers[node.Text] {
				w.reached.initializers[node.Text] = true
				if init := w.initializers[node.Text]; init.Value != nil {
					w.expression(init.Value)
				}
			}
		}
	}
	if node.Callable != nil && node.Callable.Target != "" {
		w.operation(node.Callable.Target)
	}
	if node.Coordination != nil {
		w.coordination(node.Coordination)
	}
	w.invocation(node.Invocation)
	w.match(node.Match)
	for _, input := range node.Inputs {
		w.expression(input)
	}
}

func (w *browserReachWalker) invocation(call *ir.Invocation) {
	if call == nil {
		return
	}
	for i := range call.Steps {
		w.step(&call.Steps[i])
	}
}

func (w *browserReachWalker) step(step *ir.InvocationStep) {
	if step == nil {
		return
	}
	w.operation(step.Identity)
	if step.FormAction != nil && step.FormAction.Handler != "" {
		w.operation(step.FormAction.Handler)
	}
	w.expression(step.Callee)
	w.expression(step.Native)
	for _, argument := range step.Arguments {
		w.expression(argument)
	}
	for i := range step.Prepare {
		w.expression(step.Prepare[i].Value)
	}
	if step.Fixtures != nil {
		for _, row := range step.Fixtures.Rows {
			for _, binding := range row.Prepare {
				w.expression(binding.Value)
			}
			for _, arg := range row.Arguments {
				w.expression(arg)
			}
		}
	}
}

func (w *browserReachWalker) coordination(coordination *ir.Coordination) {
	if coordination == nil {
		return
	}
	for i := range coordination.Entries {
		entry := &coordination.Entries[i]
		w.invocation(entry.Call)
		w.expression(entry.Spread)
		w.outcome(entry.Handler)
	}
	w.outcome(coordination.Aggregate)
	w.outcome(coordination.Shared)
}

func (w *browserReachWalker) outcome(handler *ir.OutcomeHandler) {
	if handler == nil {
		return
	}
	w.region(handler.Region)
	for i := range handler.Arms {
		w.completion(handler.Arms[i].Body)
	}
}

// pruneBrowserAssembly filters specialization ID lists and records the
// function/initializer closure for browser emission. Bun keeps every
// checked declaration.
func pruneBrowserAssembly(assembly *programAssembly, reached *browserReachable) {
	if assembly == nil || reached == nil {
		return
	}
	assembly.reachedFunctions = reached.functions
	assembly.reachedInitializers = reached.initializers
	keep := func(ids []string) []string {
		out := ids[:0]
		for _, id := range ids {
			if reached.specializations[id] {
				out = append(out, id)
			}
		}
		return out
	}
	assembly.httpIDs = keep(assembly.httpIDs)
	assembly.codecIDs = keep(assembly.codecIDs)
	assembly.collectionIDs = keep(assembly.collectionIDs)
	assembly.browserStateIDs = keep(assembly.browserStateIDs)
}

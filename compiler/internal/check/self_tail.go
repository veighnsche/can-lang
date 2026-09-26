package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// codeNotLowered is the advisory note for a recursive relay the self-tail
// proof did not lower. Recursion keeps its current nested-call behavior;
// the note names the first exclusion so authors can reshape the frame.
const codeNotLowered = "CAN-CHECK-NOT-LOWERED"

// Self-tail hazard operations by exact catalogue identity. A region holding
// any of these never lowers: the frame owns state a loop iteration must
// not reuse. One-shot settled operations (plain SQL queries, one-shot S3
// byte calls, sleeps) are absent deliberately: they complete before the
// relay and hold nothing across it.
var selfTailHazards = map[string]string{
	// Pending timers and handlers: callbacks outlive the relay.
	"can.std.browser@1::set_timeout":     "pending timer",
	"can.std.browser@1::on_event":        "pending browser handler",
	"can.std.browser@1::on_cancel_key":   "pending browser handler",
	"can.std.browser@1::on_cancel_event": "pending browser handler",
	// Live leases: transaction work stays open across the relay.
	"can.std.sql@1::with_transaction":           "live lease (transaction)",
	"can.std.sql@1::transaction_query_one":      "live lease (transaction)",
	"can.std.sql@1::transaction_query_optional": "live lease (transaction)",
	"can.std.sql@1::transaction_query_rows":     "live lease (transaction)",
	"can.std.sql@1::transaction_execute":        "live lease (transaction)",
	// Drain-owned values: readers, writers and uploads need their drain.
	"can.std.stream@1::read_many":     "drain-owned value (stream)",
	"can.std.stream@1::write_some":    "drain-owned value (stream)",
	"can.std.stream@1::close_reader":  "drain-owned value (stream)",
	"can.std.stream@1::close_writer":  "drain-owned value (stream)",
	"can.std.stream@1::cancel_reader": "drain-owned value (stream)",
	"can.std.stream@1::cancel_writer": "drain-owned value (stream)",
	"can.std.s3@1::read_stream":       "drain-owned value (s3 stream)",
	"can.std.s3@1::write_stream":      "drain-owned value (s3 stream)",
	"can.std.s3@1::begin_upload":      "drain-owned value (s3 upload)",
	"can.std.s3@1::upload_write":      "drain-owned value (s3 upload)",
	"can.std.s3@1::upload_finish":     "drain-owned value (s3 upload)",
	"can.std.s3@1::cancel_upload":     "drain-owned value (s3 upload)",
	// Deferred completions bound to later requests.
	"can.std.action@1::mount": "deferred completion (action mount)",
}

// tailHazard maps a step identity to its lowering hazard, stripping the
// generic specialization suffix: with_transaction<int> holds its lease
// under every instantiation.
func tailHazard(identity string) (string, bool) {
	if reason, ok := selfTailHazards[identity]; ok {
		return reason, true
	}
	if i := strings.Index(identity, "/instance/"); i >= 0 {
		reason, ok := selfTailHazards[identity[:i]]
		return reason, ok
	}
	return "", false
}

// directSelfCall reports whether the relay is one plain call to the
// enclosing function with positionally aligned arguments: no method
// chain, dynamic callee, native/array/asset/site step, fixture table or
// arity mismatch. Anything else keeps nested-call behavior.
func directSelfCall(call *ir.Invocation, region *ir.Region) bool {
	if region == nil || region.Kind != ir.FunctionRegion || call == nil || len(call.Steps) != 1 {
		return false
	}
	step := call.Steps[0]
	if step.Identity != region.ID || step.Callee != nil || step.Native != nil || step.Array != nil || step.Asset != nil {
		return false
	}
	if step.SQL != nil || step.FormAction != nil || step.JSONFetch != nil || step.Action != nil || step.Fixtures != nil {
		return false
	}
	return len(step.Arguments) == len(region.Inputs)
}

// tailFrame walks one region tree in source order, tracking the enclosing
// region across coordination handler nests. It reports the first lowering
// hazard plus every relay completion with its enclosing region.
type tailFrame struct {
	reason string
	relays []tailRelay
}

type tailRelay struct {
	completion *ir.Completion
	enclosing  *ir.Region
}

func scanTailFrame(region *ir.Region) tailFrame {
	frame := tailFrame{}
	var expression func(node *ir.Expression, enclosing *ir.Region)
	var invocation func(call *ir.Invocation, enclosing *ir.Region)
	var completion func(node *ir.Completion, enclosing *ir.Region)
	var match func(node *ir.Match, enclosing *ir.Region)
	var coordination func(node *ir.Coordination, enclosing *ir.Region)
	hazard := func(reason string) {
		if frame.reason == "" {
			frame.reason = reason
		}
	}
	expression = func(node *ir.Expression, enclosing *ir.Region) {
		if node == nil {
			return
		}
		// A callable value defers its completion past creation; without
		// escape proof it may observe reassigned parameters.
		if node.Kind == ir.CallableValue {
			hazard("deferred completion (callable value)")
		}
		for _, input := range node.Inputs {
			expression(input, enclosing)
		}
		invocation(node.Invocation, enclosing)
		match(node.Match, enclosing)
		coordination(node.Coordination, enclosing)
	}
	invocation = func(call *ir.Invocation, enclosing *ir.Region) {
		if call == nil {
			return
		}
		for i := range call.Steps {
			step := &call.Steps[i]
			if reason, ok := tailHazard(step.Identity); ok {
				hazard(reason)
			}
			if step.Fixtures != nil {
				hazard("fixture table")
				for j := range step.Fixtures.Rows {
					row := &step.Fixtures.Rows[j]
					for _, prepared := range row.Prepare {
						expression(prepared.Value, enclosing)
					}
					for _, argument := range row.Arguments {
						expression(argument, enclosing)
					}
					completion(row.Expected, enclosing)
				}
			}
			expression(step.Callee, enclosing)
			for _, prepared := range step.Prepare {
				expression(prepared.Value, enclosing)
			}
			expression(step.Native, enclosing)
			for _, argument := range step.Arguments {
				expression(argument, enclosing)
			}
		}
	}
	completion = func(node *ir.Completion, enclosing *ir.Region) {
		if node == nil {
			return
		}
		if node.Kind == ir.RelayCompletion {
			frame.relays = append(frame.relays, tailRelay{completion: node, enclosing: enclosing})
		}
		expression(node.Value, enclosing)
		invocation(node.Call, enclosing)
		if node.Block != nil {
			for _, statement := range node.Block.Steps {
				expression(statement.Value, enclosing)
				invocation(statement.Call, enclosing)
				coordination(statement.Coordination, enclosing)
			}
			completion(node.Block.Terminal, enclosing)
		}
		match(node.Match, enclosing)
	}
	match = func(node *ir.Match, enclosing *ir.Region) {
		if node == nil {
			return
		}
		for _, value := range node.Values {
			expression(value, enclosing)
		}
		invocation(node.Call, enclosing)
		for _, arm := range node.Arms {
			completion(arm.Body, enclosing)
			expression(arm.Value, enclosing)
		}
	}
	coordination = func(node *ir.Coordination, enclosing *ir.Region) {
		if node == nil {
			return
		}
		hazard("deferred completion (coordination)")
		handler := func(outcome *ir.OutcomeHandler) {
			if outcome == nil {
				return
			}
			nested := enclosing
			if outcome.Region != nil {
				nested = outcome.Region
			}
			for _, arm := range outcome.Arms {
				completion(arm.Body, nested)
				expression(arm.Value, nested)
			}
			if outcome.Region != nil && outcome.Region.Body != nil {
				for _, statement := range outcome.Region.Body.Steps {
					expression(statement.Value, nested)
					invocation(statement.Call, nested)
					coordination(statement.Coordination, nested)
				}
				completion(outcome.Region.Body.Terminal, nested)
			}
		}
		for i := range node.Entries {
			entry := &node.Entries[i]
			invocation(entry.Call, enclosing)
			expression(entry.Spread, enclosing)
			handler(entry.Handler)
		}
		handler(node.Aggregate)
		handler(node.Shared)
	}
	if region != nil && region.Body != nil {
		for _, statement := range region.Body.Steps {
			expression(statement.Value, region)
			invocation(statement.Call, region)
			coordination(statement.Coordination, region)
		}
		completion(region.Body.Terminal, region)
	}
	return frame
}

// proveSelfTail marks proven self-tail relays in a checked region. Only
// direct self calls in hazard-free function frames lower; anything else
// keeps nested-call behavior, with the first exclusion recorded for the
// cycle-note pass.
func proveSelfTail(region *ir.Region) {
	if region == nil || region.Kind != ir.FunctionRegion {
		return
	}
	frame := scanTailFrame(region)
	var candidates []*ir.Completion
	for _, relay := range frame.relays {
		if relay.enclosing == region && directSelfCall(relay.completion.Call, region) {
			candidates = append(candidates, relay.completion)
		}
	}
	if len(candidates) == 0 {
		return
	}
	if frame.reason != "" {
		region.TailExclusion = frame.reason
		return
	}
	for _, candidate := range candidates {
		candidate.SelfTail = true
	}
}

// tailGraph is the static call graph over checked regions: relay and call
// targets by identity. Native operations are sinks without out-edges.
type tailGraph struct {
	edges map[string]map[string]bool
}

func (g tailGraph) reaches(from, to string) bool {
	seen := map[string]bool{from: true}
	stack := []string{from}
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for next := range g.edges[current] {
			if next == to {
				return true
			}
			if !seen[next] {
				seen[next] = true
				stack = append(stack, next)
			}
		}
	}
	return false
}

// noteCyclicRelays reports every unlowered relay on a static recursion
// cycle: failed self proofs carry their exclusion, and mutual relays
// name their non-self target. Ordinary acyclic relays stay silent.
func noteCyclicRelays(program *Program, warn func(Warning)) {
	if program == nil || warn == nil {
		return
	}
	var regions []*ir.Region
	for _, fn := range program.Functions {
		if fn.Region != nil {
			regions = append(regions, fn.Region)
		}
	}
	for _, native := range program.Natives {
		regions = append(regions, native.Regions...)
	}
	graph := tailGraph{edges: map[string]map[string]bool{}}
	type relaySite struct {
		region     *ir.Region
		completion *ir.Completion
		target     string
	}
	var sites []relaySite
	for _, region := range regions {
		if region == nil {
			continue
		}
		add := func(target string) {
			if target == "" {
				return
			}
			if graph.edges[region.ID] == nil {
				graph.edges[region.ID] = map[string]bool{}
			}
			graph.edges[region.ID][target] = true
		}
		var expression func(node *ir.Expression)
		var invocation func(call *ir.Invocation)
		var completion func(node *ir.Completion)
		var match func(node *ir.Match)
		var coordination func(node *ir.Coordination)
		expression = func(node *ir.Expression) {
			if node == nil {
				return
			}
			if node.Kind == ir.Call {
				add(node.Text)
			}
			for _, input := range node.Inputs {
				expression(input)
			}
			invocation(node.Invocation)
			match(node.Match)
			coordination(node.Coordination)
		}
		invocation = func(call *ir.Invocation) {
			if call == nil {
				return
			}
			for i := range call.Steps {
				step := &call.Steps[i]
				add(step.Identity)
				expression(step.Callee)
				for _, prepared := range step.Prepare {
					expression(prepared.Value)
				}
				expression(step.Native)
				for _, argument := range step.Arguments {
					expression(argument)
				}
				if step.Fixtures != nil {
					for j := range step.Fixtures.Rows {
						row := &step.Fixtures.Rows[j]
						for _, prepared := range row.Prepare {
							expression(prepared.Value)
						}
						for _, argument := range row.Arguments {
							expression(argument)
						}
						completion(row.Expected)
					}
				}
			}
		}
		completion = func(node *ir.Completion) {
			if node == nil {
				return
			}
			expression(node.Value)
			invocation(node.Call)
			if node.Block != nil {
				for _, statement := range node.Block.Steps {
					expression(statement.Value)
					invocation(statement.Call)
					coordination(statement.Coordination)
				}
				completion(node.Block.Terminal)
			}
			match(node.Match)
		}
		match = func(node *ir.Match) {
			if node == nil {
				return
			}
			for _, value := range node.Values {
				expression(value)
			}
			invocation(node.Call)
			for _, arm := range node.Arms {
				completion(arm.Body)
				expression(arm.Value)
			}
		}
		coordination = func(node *ir.Coordination) {
			if node == nil {
				return
			}
			handler := func(outcome *ir.OutcomeHandler) {
				if outcome == nil {
					return
				}
				for _, arm := range outcome.Arms {
					completion(arm.Body)
					expression(arm.Value)
				}
				if outcome.Region != nil && outcome.Region.Body != nil {
					for _, statement := range outcome.Region.Body.Steps {
						expression(statement.Value)
						invocation(statement.Call)
						coordination(statement.Coordination)
					}
					completion(outcome.Region.Body.Terminal)
				}
			}
			for i := range node.Entries {
				entry := &node.Entries[i]
				invocation(entry.Call)
				expression(entry.Spread)
				handler(entry.Handler)
			}
			handler(node.Aggregate)
			handler(node.Shared)
		}
		if region.Body != nil {
			for _, statement := range region.Body.Steps {
				expression(statement.Value)
				invocation(statement.Call)
				coordination(statement.Coordination)
			}
			completion(region.Body.Terminal)
		}
		for _, relay := range scanTailFrame(region).relays {
			if relay.completion.SelfTail || relay.completion.Call == nil || len(relay.completion.Call.Steps) != 1 {
				continue
			}
			sites = append(sites, relaySite{region: relay.enclosing, completion: relay.completion, target: relay.completion.Call.Steps[0].Identity})
		}
	}
	ordered := append([]relaySite(nil), sites...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].region.Source != ordered[j].region.Source {
			return ordered[i].region.Source < ordered[j].region.Source
		}
		return ordered[i].completion.Span.Start < ordered[j].completion.Span.Start
	})
	for _, site := range ordered {
		if !graph.reaches(site.target, site.region.ID) {
			continue
		}
		reason := site.region.TailExclusion
		if site.target != site.region.ID || !directSelfCall(site.completion.Call, site.region) {
			reason = fmt.Sprintf("relay targets %s, not the enclosing function", site.target)
		}
		if reason == "" {
			reason = "relay is not a direct self call"
		}
		warn(cyclicRelayWarning(program, site.region, site.completion, reason))
	}
}

// cyclicRelayWarning locates the note on the relay span. Positions resolve
// against the diagnosing graph's bytes; an unresolvable span keeps the
// region source with the explicit unavailable marker from Format.
func cyclicRelayWarning(program *Program, region *ir.Region, completion *ir.Completion, reason string) Warning {
	line, column := 1, 1
	if program != nil && program.World != nil && program.World.Graph != nil {
		for _, owner := range program.World.Graph.Projects {
			for _, s := range owner.Sources {
				if s.Path != region.Source {
					continue
				}
				if file, err := source.New(s.Path, string(s.Bytes)); err == nil {
					if position, err := file.Position(completion.Span.Start); err == nil {
						line, column = position.Line, position.Column
					}
				}
			}
		}
	}
	return Warning{Severity: SeverityNote, Code: codeNotLowered, File: region.Source, Line: line, Column: column, Span: completion.Span, Message: fmt.Sprintf("recursive relay not lowered: %s", reason)}
}

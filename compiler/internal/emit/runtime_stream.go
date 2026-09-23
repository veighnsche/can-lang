package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// streamOperationBindings maps the B1-05 pull-stream operations and the
// file stream producers to their state-module targets.
func streamOperationBindings() bindingContribution {
	// Generic reader operations resolve per specialization key in
	// streamSpecializationBindings; only concrete operations bind here.
	functions := map[string]string{
		"can.std.stream@1::write_some":       "$canStreamWrites.writeSome",
		"can.std.stream@1::close_writer":     "$canStreamWrites.closeWriter",
		"can.std.stream@1::cancel_writer":    "$canStreamWrites.cancelWriter",
		"can.std.files@1::read_stream":       "$canFileStreams.readStream",
		"can.std.files@1::read_lines_stream": "$canFileStreams.readLinesStream",
		"can.std.files@1::write_stream":      "$canFileStreams.writeStream",
	}
	return bindingContribution{domain: "stream", functions: functions}
}

// streamSpecializationBindings maps each checked reader specialization key
// to its shared factory method. The runtime dispatches on the handle's
// recorded item kind, so specializations share one implementation.
func (assembly *programAssembly) streamSpecializationBindings() bindingContribution {
	targets := map[string]string{
		"can.std.stream@1::read_many":     "$canStreamReads.readMany",
		"can.std.stream@1::close_reader":  "$canStreamReads.closeReader",
		"can.std.stream@1::cancel_reader": "$canStreamReads.cancelReader",
	}
	functions := map[string]string{}
	for key, special := range assembly.program.Streams {
		if target, ok := targets[special.Operation]; ok {
			functions[key] = target
		}
	}
	return bindingContribution{domain: "stream-specializations", functions: functions}
}

// streamStateImports lists the pull-stream factory modules the shared state
// module needs.
func (assembly *programAssembly) streamStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/transport/stream/readable.ts", Names: []ImportName{{"createStreamReads", "$canCreateStreamReads"}}},
		{Target: runtime + "/transport/stream/writable.ts", Names: []ImportName{{"createStreamWrites", "$canCreateStreamWrites"}}},
		{Target: runtime + "/transport/stream/lifecycle.ts", Names: []ImportName{{"isStreamHandleValue", "$canIsStream"}}},
		{Target: runtime + "/platform/files/stream.ts", Names: []ImportName{{"createFileStreams", "$canCreateFileStreams"}}},
	}
}

// streamStateValueImportNames lists the pull-stream factory values authored
// and assertion modules import from the state module.
func streamStateValueImportNames() []ImportName {
	return []ImportName{{"$canStreamReads", "$canStreamReads"}, {"$canStreamWrites", "$canStreamWrites"}, {"$canFileStreams", "$canFileStreams"}}
}

// declareStreamState emits the pull-stream factory bindings.
func (builder *stateBuilder) declareStreamState() {
	builder.out.WriteString("export let $canStreamReads:ReturnType<typeof $canCreateStreamReads>;\nexport let $canStreamWrites:ReturnType<typeof $canCreateStreamWrites>;\nexport let $canFileStreams:ReturnType<typeof $canCreateFileStreams>;\n")
}

// initializeStreamState constructs the pull-stream factories inside the
// shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeStreamState() {
	fmt.Fprintf(&builder.out, "$canStreamReads=$canCreateStreamReads($canDomain,{readFailed:%s,cancelled:%s,closeFailed:%s,limitExceeded:%s});\n", quote(builder.numberIDs["can.std.stream@1::read_failed"]), quote(builder.numberIDs["can.std.stream@1::cancelled"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]))
	fmt.Fprintf(&builder.out, "$canStreamWrites=$canCreateStreamWrites($canDomain,{writeFailed:%s,closeFailed:%s});\n", quote(builder.numberIDs["can.std.stream@1::write_failed"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]))
	fmt.Fprintf(&builder.out, "$canFileStreams=$canCreateFileStreams($canDomain,{notFound:%s,denied:%s,invalidPath:%s,unexpectedKind:%s,limitExceeded:%s,ioError:%s,closeFailed:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::invalid_path"]), quote(builder.numberIDs["can.std.files@1::unexpected_kind"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]), quote(builder.numberIDs["can.std.files@1::io_error"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]))
}

// emitStreamKinds emits the opaque-handle kind table the domain predicate
// uses to recognize reader and writer values at boundaries.
func (builder *stateBuilder) emitStreamKinds() error {
	streamKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.stream@1::reader":
			streamKinds[typ.Identity()] = "stream-reader"
		case "can.std.stream@1::writer":
			streamKinds[typ.Identity()] = "stream-writer"
		}
	}
	streamKindsJSON, err := json.Marshal(streamKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canStreamKinds:Readonly<Record<string,string>>=%s;\n", streamKindsJSON)
	return nil
}

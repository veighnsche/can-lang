package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// websocketOperationBindings maps the B1-07 websocket operations to their
// state-module targets. Event reads bind through the B1-05 reader
// specializations; only concrete session operations bind here.
func websocketOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.ws@1::connect":    "$canWebSockets.connect",
		"can.std.ws@1::accept":     "$canWebSockets.accept",
		"can.std.ws@1::send_text":  "$canWebSockets.sendText",
		"can.std.ws@1::send_bytes": "$canWebSockets.sendBytes",
		"can.std.ws@1::close":      "$canWebSockets.close",
	}
	return bindingContribution{domain: "websocket", functions: functions}
}

// websocketStateImports lists the websocket factory module the shared state
// module needs.
func (assembly *programAssembly) websocketStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/websocket.ts", Names: []ImportName{{"createWebSockets", "$canCreateWebSockets"}, {"isWebSocketValue", "$canIsWebSocket"}}},
	}
}

// websocketStateValueImportNames lists the websocket factory value authored
// and assertion modules import from the state module.
func websocketStateValueImportNames() []ImportName {
	return []ImportName{{"$canWebSockets", "$canWebSockets"}}
}

// declareWebSocketState emits the websocket factory binding.
func (builder *stateBuilder) declareWebSocketState() {
	builder.out.WriteString("export let $canWebSockets:ReturnType<typeof $canCreateWebSockets>;\n")
}

// initializeWebSocketState constructs the websocket factory inside the
// shared initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeWebSocketState() {
	fmt.Fprintf(&builder.out, "$canWebSockets=$canCreateWebSockets($canDomain,{connectFailed:%s,upgradeFailed:%s,unsupportedProtocol:%s,sendFailed:%s,invalidClose:%s,limitExceeded:%s,invalidUrl:%s,invalidProtocol:%s,closeFailed:%s,text:%s,binary:%s,drain:%s,close:%s,connection:%s});\n", quote(builder.numberIDs["can.std.ws@1::connect_failed"]), quote(builder.numberIDs["can.std.ws@1::upgrade_failed"]), quote(builder.numberIDs["can.std.ws@1::unsupported_protocol"]), quote(builder.numberIDs["can.std.ws@1::send_failed"]), quote(builder.numberIDs["can.std.ws@1::invalid_close"]), quote(builder.numberIDs["can.std.ws@1::limit_exceeded"]), quote(builder.numberIDs["can.std.ws@1::invalid_url"]), quote(builder.numberIDs["can.std.ws@1::invalid_protocol"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]), quote(builder.numberIDs["can.std.ws@1::text"]), quote(builder.numberIDs["can.std.ws@1::binary"]), quote(builder.numberIDs["can.std.ws@1::drain"]), quote(builder.numberIDs["can.std.ws@1::closed"]), quote(builder.numberIDs["can.std.ws@1::connection"]))
}

// emitWebSocketKinds emits the opaque-handle kind table the domain predicate
// uses to recognize session values at boundaries.
func (builder *stateBuilder) emitWebSocketKinds() error {
	websocketKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		if typ.Declaration() == "can.std.ws@1::session" {
			websocketKinds[typ.Identity()] = "ws-session"
		}
	}
	websocketKindsJSON, err := json.Marshal(websocketKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canWebSocketKinds:Readonly<Record<string,string>>=%s;\n", websocketKindsJSON)
	return nil
}

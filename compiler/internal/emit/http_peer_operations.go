package emit

import "fmt"

// httpPeerOperationBindings maps the NT-I04 http_peer operations to the
// single $canHttpPeer contribution: peer listener/dial/connection calls on
// $canHttpPeer.peer and HTTP request/response calls on $canHttpPeer.http.
// Both adapters share the $canTest.owner table, wired once in the shared
// state module; every method is async and takes the trailing call context.
func httpPeerOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.http_peer@1::open_listener":         "$canHttpPeer.peer.openListener",
		"can.std.http_peer@1::close_listener":        "$canHttpPeer.peer.closeListener",
		"can.std.http_peer@1::dial_peer":             "$canHttpPeer.peer.dialPeer",
		"can.std.http_peer@1::retry_dial":            "$canHttpPeer.peer.retryDial",
		"can.std.http_peer@1::accept":                "$canHttpPeer.peer.accept",
		"can.std.http_peer@1::write":                 "$canHttpPeer.peer.write",
		"can.std.http_peer@1::read":                  "$canHttpPeer.peer.read",
		"can.std.http_peer@1::half_close":            "$canHttpPeer.peer.halfClose",
		"can.std.http_peer@1::close_connection":      "$canHttpPeer.peer.closeConnection",
		"can.std.http_peer@1::read_listener_facts":   "$canHttpPeer.peer.readListenerFacts",
		"can.std.http_peer@1::read_connection_facts": "$canHttpPeer.peer.readConnectionFacts",
		"can.std.http_peer@1::read_dial_facts":       "$canHttpPeer.peer.readDialFacts",
		"can.std.http_peer@1::open_request":          "$canHttpPeer.http.openRequest",
		"can.std.http_peer@1::add_header":            "$canHttpPeer.http.addHeader",
		"can.std.http_peer@1::send_body_chunk":       "$canHttpPeer.http.sendBodyChunk",
		"can.std.http_peer@1::end_upload":            "$canHttpPeer.http.endUpload",
		"can.std.http_peer@1::deliver_response":      "$canHttpPeer.http.deliverResponse",
		"can.std.http_peer@1::read_body_chunk":       "$canHttpPeer.http.readBodyChunk",
		"can.std.http_peer@1::reissue":               "$canHttpPeer.http.reissue",
		"can.std.http_peer@1::close_request":         "$canHttpPeer.http.closeRequest",
		"can.std.http_peer@1::read_request_facts":    "$canHttpPeer.http.readRequestFacts",
		"can.std.http_peer@1::read_response_facts":   "$canHttpPeer.http.readResponseFacts",
	}
	return bindingContribution{domain: "http_peer", functions: functions}
}

// httpPeerStateImports lists the http_peer adapter factory modules the
// shared state module needs.
func (assembly *programAssembly) httpPeerStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i04/peer.ts", Names: []ImportName{{"createHttpPeerPeer", "$canCreateHttpPeerPeer"}}},
		{Target: runtime + "/test-support/slices/i04/http.ts", Names: []ImportName{{"createHttpPeerHttp", "$canCreateHttpPeerHttp"}}},
	}
}

// httpPeerStateValueImportNames lists the http_peer factory value authored
// and assertion modules import from the state module.
func httpPeerStateValueImportNames() []ImportName {
	return []ImportName{{"$canHttpPeer", "$canHttpPeer"}}
}

// declareHttpPeerState emits the http_peer factory binding.
func (builder *stateBuilder) declareHttpPeerState() {
	builder.out.WriteString("export let $canHttpPeer:{peer:ReturnType<typeof $canCreateHttpPeerPeer>,http:ReturnType<typeof $canCreateHttpPeerHttp>};\n")
}

// initializeHttpPeerState constructs the peer and http adapters inside the
// shared initializer, after the test factory exists: both adapters share
// $canTest.owner, so a grant admitted through test::grant_admit admits
// http_peer calls and a foreign owner fails stale_handle admission-first.
// Header records reuse the http::header nominal identity the catalogue
// facts types reference.
func (builder *stateBuilder) initializeHttpPeerState() {
	fmt.Fprintf(&builder.out, "$canHttpPeer={peer:$canCreateHttpPeerPeer($canDomain,{staleHandle:%s,closedHandle:%s,peerFault:%s,some:%s,none:%s,listenerFacts:%s,connectionFacts:%s,dialFacts:%s,dialResult:%s,writeReceipt:%s,readResult:%s,connectionCloseReceipt:%s,listenerCloseReceipt:%s},$canTest.owner),http:$canCreateHttpPeerHttp($canDomain,{staleHandle:%s,closedHandle:%s,httpFault:%s,readResult:%s,some:%s,none:%s,header:%s,requestFacts:%s,redirect:%s,responseFacts:%s,headerReceipt:%s,bodyChunkReceipt:%s,requestCloseReceipt:%s},$canTest.owner)};\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.test@1::closed_handle"]),
		quote(builder.numberIDs["can.std.http_peer@1::peer_fault"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.http_peer@1::listener_facts"]),
		quote(builder.numberIDs["can.std.http_peer@1::connection_facts"]),
		quote(builder.numberIDs["can.std.http_peer@1::dial_facts"]),
		quote(builder.numberIDs["can.std.http_peer@1::dial_result"]),
		quote(builder.numberIDs["can.std.http_peer@1::write_receipt"]),
		quote(builder.numberIDs["can.std.http_peer@1::read_result"]),
		quote(builder.numberIDs["can.std.http_peer@1::connection_close_receipt"]),
		quote(builder.numberIDs["can.std.http_peer@1::listener_close_receipt"]),
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.test@1::closed_handle"]),
		quote(builder.numberIDs["can.std.http_peer@1::http_fault"]),
		quote(builder.numberIDs["can.std.http_peer@1::read_result"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.http@1::header"]),
		quote(builder.numberIDs["can.std.http_peer@1::request_facts"]),
		quote(builder.numberIDs["can.std.http_peer@1::redirect"]),
		quote(builder.numberIDs["can.std.http_peer@1::response_facts"]),
		quote(builder.numberIDs["can.std.http_peer@1::header_receipt"]),
		quote(builder.numberIDs["can.std.http_peer@1::body_chunk_receipt"]),
		quote(builder.numberIDs["can.std.http_peer@1::request_close_receipt"]))
}

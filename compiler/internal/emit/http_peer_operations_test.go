package emit

import (
	"strings"
	"testing"
)

func TestHttpPeerWsEmission(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/http-peer-ws/main.can")
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
	}
	for _, want := range []string{
		"$canHttpPeer.ws.connect",
		"$canHttpPeer.ws.send",
		"$canHttpPeer.ws.deliverEvent",
		"$canHttpPeer.ws.pollEvent",
		"$canHttpPeer.ws.close",
		"$canHttpPeer.ws.deliverRemoteClose",
		"$canHttpPeer.ws.readConnectionFacts",
		"$canCreateHttpPeerWs",
		"export let $canHttpPeer:",
		"test-support/slices/i05/websocket.ts",
		"$canTest.owner",
		"peerFault:",
	} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestHttpPeerEmission(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/http-peer/peer.can")
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
	}
	for _, want := range []string{
		"$canHttpPeer.peer.openListener",
		"$canHttpPeer.peer.closeListener",
		"$canHttpPeer.peer.dialPeer",
		"$canHttpPeer.peer.retryDial",
		"$canHttpPeer.peer.accept",
		"$canHttpPeer.peer.write",
		"$canHttpPeer.peer.read",
		"$canHttpPeer.peer.halfClose",
		"$canHttpPeer.peer.closeConnection",
		"$canHttpPeer.peer.readListenerFacts",
		"$canHttpPeer.peer.readConnectionFacts",
		"$canHttpPeer.peer.readDialFacts",
		"$canCreateHttpPeerPeer",
		"export let $canHttpPeer:",
		"test-support/slices/i04/peer.ts",
		"$canTest.owner",
		"peerFault:",
	} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestHttpRequestEmission(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/http-peer/http.can")
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
	}
	for _, want := range []string{
		"$canHttpPeer.http.openRequest",
		"$canHttpPeer.http.addHeader",
		"$canHttpPeer.http.sendBodyChunk",
		"$canHttpPeer.http.endUpload",
		"$canHttpPeer.http.deliverResponse",
		"$canHttpPeer.http.readBodyChunk",
		"$canHttpPeer.http.reissue",
		"$canHttpPeer.http.closeRequest",
		"$canHttpPeer.http.readRequestFacts",
		"$canHttpPeer.http.readResponseFacts",
		"$canCreateHttpPeerHttp",
		"test-support/slices/i04/http.ts",
		"httpFault:",
	} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
}

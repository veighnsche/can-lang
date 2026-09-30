// Package external implements generic (engine-agnostic) external-service
// resource ownership for native-test resource ownership (task P29).
//
// The registry journals service admission before any effect: every listener,
// remote connection, remote context, database namespace, database credential
// and object-store prefix admission reserves journal intent (P07) before the
// local double is created. Listener, destination, namespace and prefix sets
// are finite and declared up front; anything outside them is foreign and
// rejected. Grants are unforgeable capabilities minted here and verified by
// table lookup: caller-invented tokens, guessed IDs and cross-resource
// tokens are never authority.
//
// Database credentials are opaque handles: facts carry a digest of the
// handle, never a raw secret. Remote connections and contexts bind to the
// owner's spawn-start identity (P10); a live token under a foreign identity
// is rejected. Dispatch runs each effect at most once; an uncertain
// dispatch is never redispatched. Unsettled resource charges are retained,
// never silently dropped, and only an explicit Release settles them.
// Quiescence stops admission and fencing acknowledges the owned live set
// for recovery, which touches only evidenced owned resources and never
// auto-resolves a missing release.
//
// All doubles are in-memory fakes: no real network listeners, databases or
// cloud services. Every service leg is recorded UNQUALIFIED in facts, and
// CleanupProven is always false: engine-specific releases belong to later
// tasks, so generic grants alone never claim cleanup proven.
package external

// K16: clean remote browser contexts after worker death.
//
// An N-owned remote-context service over in-memory doubles only: no
// browser, no process, no flags, no environment, no sampling, no network,
// no I/O. The service owns remote connections and their contexts
// separately from the shared launch server: cases connect to a declared
// shared server, open contexts under their own connection, and clean each
// context through an independent N close authority. The shared launch
// server is preserved: a case close-server attempt rejects before any
// effect and the server stays provably live and untouched.
//
// Worker death is noted as a fact, not cleanup: after death no new
// context opens, while close requests, confirmation tracking, and the
// confirmed seal proceed under independent ownership. A lost close
// confirmation remains unresolved: the context stays explicitly
// pending/unknown, is never treated as closed, and no later seal or
// receipt reads it as closed.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, network state, or process state. The only strings that may
//   coincide are the server/connection/context names under test (supplied
//   by the test, not by the driver) and the vocabulary this file uses for
//   facts. Close tokens are verified by lookup against the injected
//   authority and never parsed.
//   PROVES: shared-server admission and preservation, connection/context
//   ownership separate from the server, worker-death noting, close
//   request/confirmation separation, externally confirmed seals, and error
//   provenance (every failure names its layer). Driver claims are compared
//   AGAINST these facts; the service never derives a cleanup fact FROM a
//   driver claim.
//   Consequently a worker receipt can never prove cleanup: only an
//   independent close confirmation over the exact context proves it.
//
// Split with the Go mirror
// (tools/native-test-owner/external/browser_remote.go):
//   TS owns the typed service mechanics, the close-verifier seam, and the
//   bounded self-check. The Go mirror owns the same contract outside the
//   service under independent N ownership: shared-server admission,
//   connection/context ownership, worker-death noting, close minting, and
//   confirmation. Both enforce the same contract: declared servers, opaque
//   tokens verified by lookup, digest-only frozen facts, and layered
//   errors. Neither contacts a browser.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, networks, or live runtimes.
// Local controls only. Live host-dependent controls wait for the
// corresponding qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_REMOTE_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative, mirrored in browser_remote.go)
// ---------------------------------------------------------------------------

export const BROWSER_REMOTE_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, network state, or process state; proves shared-server " +
  "admission and preservation, connection/context ownership separate from " +
  "the server, worker-death noting, close request/confirmation separation, " +
  "externally confirmed seals, and layered error provenance from its own " +
  "seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_REMOTE_LAYERS = ["service", "remote", "cleanup"] as const;
export type BrowserRemoteLayer = (typeof BROWSER_REMOTE_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   service: declared servers, ownership, capacity.
//   remote: connection/context binding, tokens, lifecycle, worker death,
//     and the server-close refusal.
//   cleanup: close-authority admission, close tokens, confirmation, and
//     the close seal.
export const BROWSER_REMOTE_CODES = [
  "unknown-server",
  "wrong-owner",
  "capacity-exhausted",
  "forged-token",
  "unknown-connection",
  "unknown-context",
  "context-closed",
  "worker-dead",
  "forbidden-server-close",
  "forbidden-close",
  "forged-close",
  "close-unconfirmed",
  "orphan-open",
  "already-closed",
] as const;
export type BrowserRemoteCode = (typeof BROWSER_REMOTE_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserRemoteCode, BrowserRemoteLayer>> = {
  "unknown-server": "service",
  "wrong-owner": "service",
  "capacity-exhausted": "service",
  "forged-token": "remote",
  "unknown-connection": "remote",
  "unknown-context": "remote",
  "context-closed": "remote",
  "worker-dead": "remote",
  "forbidden-server-close": "remote",
  "forbidden-close": "cleanup",
  "forged-close": "cleanup",
  "close-unconfirmed": "cleanup",
  "orphan-open": "cleanup",
  "already-closed": "cleanup",
};

export function layerOfRemoteCode(code: BrowserRemoteCode): BrowserRemoteLayer {
  return CODE_LAYER[code];
}

export class BrowserRemoteError extends Error {
  readonly layer: BrowserRemoteLayer;
  readonly code: BrowserRemoteCode;
  constructor(code: BrowserRemoteCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserRemoteError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserRemoteCode, message: string): never {
  throw new BrowserRemoteError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserRemoteLimits = Readonly<{
  maxServers: number;
  maxConnections: number;
  maxContextsPerConnection: number;
}>;

const LIMIT_KEYS = ["maxServers", "maxConnections", "maxContextsPerConnection"] as const;

export function checkRemoteLimits(value: unknown): BrowserRemoteLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserRemoteError("unknown-server", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserRemoteError("unknown-server", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserRemoteError("unknown-server", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserRemoteLimits;
}

const MAX_NAME_LEN = 128;

function checkName(value: unknown, what: string, code: BrowserRemoteCode): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail(code, `${what} must be a non-empty name of at most ${MAX_NAME_LEN} chars`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail(code, `${what} carries illegal characters: ${value}`);
  }
  return value;
}

function checkOwner(value: unknown): string {
  return checkName(value, "owner", "wrong-owner");
}

function checkServerName(value: unknown): string {
  return checkName(value, "server", "unknown-server");
}

function checkConnectionName(value: unknown): string {
  return checkName(value, "connection", "unknown-connection");
}

function checkContextName(value: unknown): string {
  return checkName(value, "context", "unknown-context");
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only) and the independent close authority seam
// ---------------------------------------------------------------------------

// The close lifecycle of one owned context. open: no close requested.
// close-pending: a close was requested and the confirmation is awaited.
// closed: the independent authority confirmed the close. unknown: the
// confirmation was lost; the close is unresolved and the context is never
// treated as closed.
export const CLOSE_STATES = ["open", "close-pending", "closed", "unknown"] as const;
export type CloseState = (typeof CLOSE_STATES)[number];

export type ServerFacts = Readonly<{
  server: string;
  owner: string;
  live: true;
  connections: readonly string[];
  digest: string;
}>;

export type ConnectionAttestation = Readonly<{
  connectionId: string;
  owner: string;
  server: string;
  handleDigest: string;
}>;

export type ConnectionFacts = Readonly<{
  connectionId: string;
  owner: string;
  server: string;
  workerDeathNoted: boolean;
  contexts: readonly string[];
  digest: string;
}>;

export type ContextAttestation = Readonly<{
  contextId: string;
  connectionId: string;
  owner: string;
  server: string;
  connectionDigest: string;
  handleDigest: string;
}>;

export type ContextFacts = Readonly<{
  contextId: string;
  connectionId: string;
  owner: string;
  server: string;
  closeState: CloseState;
  closed: boolean;
  workerDeathNoted: boolean;
  digest: string;
}>;

export type CloseRequestFacts = Readonly<{
  contextId: string;
  connectionId: string;
  owner: string;
  closeState: "close-pending";
  closed: false;
  handleDigest: string;
  digest: string;
}>;

// Independently confirmed close facts, returned by the injected
// authority. The service shape-checks every field: a lying verifier
// fails lookup.
export type CloseFacts = Readonly<{
  contextId: string;
  closed: boolean;
  digest: string;
}>;

// The independent N remote-context cleanup authority. The service holds
// it as an opaque verifier: close tokens are verified by lookup through
// this seam and never parsed. Local controls inject a seeded double; QB5
// wires the qualified authority.
export type ExternalCloseAuthority = {
  readonly authority: string;
  verifyClose(contextId: string, token: string): CloseFacts;
};

export type CloseReceipt = Readonly<{
  contextId: string;
  connectionId: string;
  owner: string;
  server: string;
  workerDeathNoted: boolean;
  digest: string;
}>;

// A cleanup proof claim. Only external-close is admissible; a worker
// receipt and every other kind reject with forbidden-close before any
// verdict is read.
export const FORBIDDEN_CLOSE_KINDS = ["worker-receipt", "server-log", "exit-code"] as const;
export type ForbiddenCloseKind = (typeof FORBIDDEN_CLOSE_KINDS)[number];

export type RemoteCleanupProofClaim =
  | Readonly<{ kind: "external-close"; receipt: CloseReceipt }>
  | Readonly<{ kind: ForbiddenCloseKind; detail: string }>;

export type RemoteCleanupVerdict = Readonly<{
  contextId: string;
  connectionId: string;
  witnessed: boolean;
  digest: string;
}>;

type ConnectionRecord = {
  connectionId: string;
  owner: string;
  server: string;
  token: string;
  workerDeathNoted: boolean;
  contexts: Map<string, ContextRecord>;
};

type ContextRecord = {
  contextId: string;
  connectionId: string;
  owner: string;
  server: string;
  token: string;
  closeState: CloseState;
  receipt: CloseReceipt | null;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// The close digest is recomputed from carried fields, so
// assertRemoteCleanupWitnessed re-verifies a presented receipt instead of
// trusting its digest string.
export function digestRemoteClose(
  contextId: string,
  connectionId: string,
  owner: string,
  server: string,
  workerDeathNoted: boolean,
): string {
  const hash = createHash("sha256");
  hash.update(contextId, "utf8");
  hash.update("\0", "utf8");
  hash.update(connectionId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(server, "utf8");
  hash.update("\0", "utf8");
  hash.update(workerDeathNoted ? "dead" : "live", "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned connections/contexts, a preserved server, confirmed closes
// ---------------------------------------------------------------------------

export type ServerGrant = Readonly<{
  owner: string;
  server: string;
}>;

export class BrowserRemoteService {
  private readonly limits: BrowserRemoteLimits;
  private readonly servers = new Map<string, ServerGrant>();
  private readonly connections = new Map<string, ConnectionRecord>();
  private readonly authority: ExternalCloseAuthority;

  constructor(
    servers: readonly ServerGrant[],
    limits: BrowserRemoteLimits,
    authority: ExternalCloseAuthority,
  ) {
    if (
      authority === null ||
      typeof authority !== "object" ||
      typeof authority.verifyClose !== "function"
    ) {
      throw new BrowserRemoteError("forbidden-close", "an independent close authority is required");
    }
    if (servers.length === 0) {
      throw new BrowserRemoteError("unknown-server", "declare at least one shared server");
    }
    if (servers.length > limits.maxServers) {
      throw new BrowserRemoteError("capacity-exhausted", "declared servers exceed the cap");
    }
    for (const grant of servers) {
      checkOwner(grant.owner);
      checkServerName(grant.server);
      if (this.servers.has(grant.server)) {
        fail("unknown-server", `duplicate shared server: ${grant.server}`);
      }
      this.servers.set(grant.server, Object.freeze({ owner: grant.owner, server: grant.server }));
    }
    this.limits = limits;
    this.authority = authority;
  }

  get connectionCount(): number {
    return this.connections.size;
  }

  // Shared-server facts stay readable and live forever: nothing in this
  // service closes, kills, or otherwise touches the server.
  serverFacts(server: string): ServerFacts {
    checkServerName(server);
    const grant = this.servers.get(server);
    if (grant === undefined) {
      fail("unknown-server", `no declared shared server: ${server}`);
    }
    const connections: string[] = [];
    for (const connection of this.connections.values()) {
      if (connection.server === server) connections.push(connection.connectionId);
    }
    connections.sort();
    return Object.freeze({
      server,
      owner: (grant as ServerGrant).owner,
      live: true as const,
      connections: Object.freeze(connections),
      digest: digestText(`server:${server}:${connections.join(",")}`),
    });
  }

  // A case close-server attempt. It always rejects with
  // forbidden-server-close before any effect: the refusal precedes every
  // mutation, and the server stays provably live and untouched.
  closeServer(owner: string, server: string): never {
    checkOwner(owner);
    checkServerName(server);
    if (!this.servers.has(server)) {
      fail("unknown-server", `no declared shared server: ${server}`);
    }
    fail("forbidden-server-close", `cases never close the shared server: ${server}`);
  }

  // Connect one owned connection to a declared shared server. Admission
  // is server-first: foreign servers refuse before anything else.
  // Reconnecting the identical connection re-attests; the id never
  // transfers between owners, servers, or connections.
  connect(owner: string, server: string, connectionId: string): ConnectionAttestation {
    checkOwner(owner);
    checkServerName(server);
    checkConnectionName(connectionId);
    if (!this.servers.has(server)) {
      fail("unknown-server", `no declared shared server: ${server}`);
    }
    if (this.connections.size >= this.limits.maxConnections) {
      fail("capacity-exhausted", "connection table full");
    }
    const prior = this.connections.get(connectionId);
    if (prior !== undefined) {
      if (prior.owner !== owner) {
        fail("wrong-owner", `connection id is owned by another identity: ${connectionId}`);
      }
      if (prior.server !== server) {
        fail("unknown-connection", `connection id is already connected: ${connectionId}`);
      }
      return this.attestationOf(prior);
    }
    const record: ConnectionRecord = {
      connectionId,
      owner,
      server,
      token: mintToken("conn"),
      workerDeathNoted: false,
      contexts: new Map(),
    };
    this.connections.set(connectionId, record);
    return this.attestationOf(record);
  }

  private attestationOf(connection: ConnectionRecord): ConnectionAttestation {
    return Object.freeze({
      connectionId: connection.connectionId,
      owner: connection.owner,
      server: connection.server,
      handleDigest: digestText(connection.token),
    });
  }

  private requireConnection(owner: string, connectionId: string): ConnectionRecord {
    const connection = this.connections.get(connectionId);
    if (connection === undefined) {
      fail("unknown-connection", `no owned connection: ${connectionId}`);
    }
    const record = connection as ConnectionRecord;
    if (record.owner !== owner) {
      fail("wrong-owner", "connection is owned by another identity");
    }
    return record;
  }

  // Verify the connection token by table lookup. Unknown connections and
  // invented tokens are never authority.
  private requireLiveConnection(
    owner: string,
    connectionId: string,
    token: string,
  ): ConnectionRecord {
    const connection = this.requireConnection(owner, connectionId);
    if (token === "" || token !== connection.token) {
      fail("forged-token", "connection token is not the attested token");
    }
    return connection;
  }

  private requireContext(connection: ConnectionRecord, contextId: string): ContextRecord {
    const context = connection.contexts.get(contextId);
    if (context === undefined) {
      fail("unknown-context", `no owned context: ${contextId}`);
    }
    return context as ContextRecord;
  }

  // Verify the context token by table lookup. Unknown contexts and
  // invented tokens are never authority.
  private requireLiveContext(
    owner: string,
    connectionId: string,
    contextId: string,
    token: string,
  ): { connection: ConnectionRecord; context: ContextRecord } {
    const connection = this.requireConnection(owner, connectionId);
    const context = this.requireContext(connection, contextId);
    if (token === "" || token !== context.token) {
      fail("forged-token", "context token is not the attested token");
    }
    return { connection, context };
  }

  // Test-only accessors: raw tokens cross exactly here so bounded
  // controls can present them on later calls. Facts carry digests only.
  connectionTokenForTest(owner: string, connectionId: string): string {
    return this.requireConnection(owner, connectionId).token;
  }

  contextTokenForTest(owner: string, connectionId: string, contextId: string): string {
    const connection = this.requireConnection(owner, connectionId);
    return this.requireContext(connection, contextId).token;
  }

  // Open one owned context under a live connection. A dead worker opens
  // nothing further; cleanup of already-open contexts proceeds
  // independently.
  openContext(
    owner: string,
    connectionId: string,
    token: string,
    contextId: string,
  ): ContextAttestation {
    checkContextName(contextId);
    const connection = this.requireLiveConnection(owner, connectionId, token);
    if (connection.workerDeathNoted) {
      fail("worker-dead", `worker is dead for connection: ${connectionId}`);
    }
    const prior = connection.contexts.get(contextId);
    if (prior !== undefined) {
      if (prior.closeState === "closed") {
        fail("context-closed", `context id is single-use and already closed: ${contextId}`);
      }
      return this.contextAttestationOf(connection, prior);
    }
    if (connection.contexts.size >= this.limits.maxContextsPerConnection) {
      fail("capacity-exhausted", "context table full for connection");
    }
    const record: ContextRecord = {
      contextId,
      connectionId,
      owner,
      server: connection.server,
      token: mintToken("ctx"),
      closeState: "open",
      receipt: null,
    };
    connection.contexts.set(contextId, record);
    return this.contextAttestationOf(connection, record);
  }

  private contextAttestationOf(
    connection: ConnectionRecord,
    context: ContextRecord,
  ): ContextAttestation {
    return Object.freeze({
      contextId: context.contextId,
      connectionId: connection.connectionId,
      owner: context.owner,
      server: context.server,
      connectionDigest: digestText(connection.token),
      handleDigest: digestText(context.token),
    });
  }

  // Note the worker's death. The connection continues under independent
  // ownership: opens stop, while close requests, confirmation tracking,
  // and the confirmed seal proceed.
  noteWorkerDeath(owner: string, connectionId: string, token: string): void {
    const connection = this.requireLiveConnection(owner, connectionId, token);
    connection.workerDeathNoted = true;
  }

  // Connection and context facts stay readable after death, after a lost
  // confirmation, and after the seal: the connection is evidence, and
  // neither death nor cleanup hides it.
  connectionFacts(owner: string, connectionId: string): ConnectionFacts {
    const connection = this.requireConnection(owner, connectionId);
    const contexts = [...connection.contexts.keys()].sort();
    return Object.freeze({
      connectionId: connection.connectionId,
      owner: connection.owner,
      server: connection.server,
      workerDeathNoted: connection.workerDeathNoted,
      contexts: Object.freeze(contexts),
      digest: digestText(
        `connection:${connection.connectionId}:${connection.server}:${contexts.join(",")}:` +
          `${connection.workerDeathNoted ? 1 : 0}`,
      ),
    });
  }

  contextFacts(owner: string, connectionId: string, contextId: string): ContextFacts {
    const connection = this.requireConnection(owner, connectionId);
    const context = this.requireContext(connection, contextId);
    return Object.freeze({
      contextId: context.contextId,
      connectionId: connection.connectionId,
      owner: context.owner,
      server: context.server,
      closeState: context.closeState,
      closed: context.closeState === "closed",
      workerDeathNoted: connection.workerDeathNoted,
      digest: digestText(
        `context:${context.contextId}:${connection.connectionId}:${context.closeState}`,
      ),
    });
  }

  // Request the close of one open context. The request moves the context
  // to close-pending; only an independent confirmation seals it. A
  // request is never itself a close.
  requestContextClose(
    owner: string,
    connectionId: string,
    contextId: string,
    token: string,
  ): CloseRequestFacts {
    const { context } = this.requireLiveContext(owner, connectionId, contextId, token);
    if (context.closeState === "closed") {
      fail("already-closed", `context is already closed: ${contextId}`);
    }
    if (context.closeState === "unknown") {
      fail("close-unconfirmed", `close confirmation is lost for context: ${contextId}`);
    }
    if (context.closeState === "close-pending") {
      return this.closeRequestOf(context);
    }
    context.closeState = "close-pending";
    return this.closeRequestOf(context);
  }

  private closeRequestOf(context: ContextRecord): CloseRequestFacts {
    return Object.freeze({
      contextId: context.contextId,
      connectionId: context.connectionId,
      owner: context.owner,
      closeState: "close-pending" as const,
      closed: false as const,
      handleDigest: digestText(context.token),
      digest: digestText(`close-request:${context.contextId}:${context.connectionId}`),
    });
  }

  // Lose the close confirmation permanently: the close stays explicitly
  // unresolved. Later seals and receipts refuse with close-unconfirmed,
  // and the context is never treated as closed.
  loseCloseConfirmation(
    owner: string,
    connectionId: string,
    contextId: string,
    token: string,
  ): void {
    const { context } = this.requireLiveContext(owner, connectionId, contextId, token);
    if (context.closeState === "closed") {
      fail("already-closed", `context is already closed: ${contextId}`);
    }
    if (context.closeState !== "close-pending") {
      fail("close-unconfirmed", `no close confirmation is awaited for context: ${contextId}`);
    }
    context.closeState = "unknown";
  }

  private checkCloseShape(value: unknown): CloseFacts {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      fail("forged-close", "close facts are malformed");
    }
    const record = value as Record<string, unknown>;
    if (typeof record["contextId"] !== "string" || record["contextId"] === "") {
      fail("forged-close", "close names no context");
    }
    if (typeof record["closed"] !== "boolean") {
      fail("forged-close", "close states no verdict");
    }
    if (typeof record["digest"] !== "string" || record["digest"] === "") {
      fail("forged-close", "close carries no digest");
    }
    return {
      contextId: record["contextId"] as string,
      closed: record["closed"] as boolean,
      digest: record["digest"] as string,
    };
  }

  // Seal the close against the independent authority. The close token is
  // verified by lookup through the injected authority; the seal needs a
  // pending request and a lost confirmation never seals. A worker receipt
  // is never verification: only the authority's lookup confirms a close.
  sealContextClose(
    owner: string,
    connectionId: string,
    contextId: string,
    token: string,
    closeToken: string,
  ): CloseReceipt {
    const { connection, context } = this.requireLiveContext(owner, connectionId, contextId, token);
    if (context.closeState === "closed") {
      fail("already-closed", `context is already closed: ${contextId}`);
    }
    if (context.closeState === "unknown") {
      fail("close-unconfirmed", `close confirmation is lost for context: ${contextId}`);
    }
    if (context.closeState !== "close-pending") {
      fail("close-unconfirmed", `no close was requested for context: ${contextId}`);
    }
    if (typeof closeToken !== "string" || closeToken === "") {
      fail("forged-close", "close token is not an attested token");
    }
    let facts: CloseFacts;
    try {
      facts = this.checkCloseShape(this.authority.verifyClose(contextId, closeToken));
    } catch (error) {
      if (error instanceof BrowserRemoteError) throw error;
      fail("forged-close", "independent authority refused the token");
    }
    const confirmed = facts as CloseFacts;
    if (confirmed.contextId !== contextId) {
      fail("forged-close", "close is for another context");
    }
    if (!confirmed.closed) {
      fail("orphan-open", "authority reports the context still open");
    }
    context.closeState = "closed";
    const receipt: CloseReceipt = Object.freeze({
      contextId: context.contextId,
      connectionId: connection.connectionId,
      owner: context.owner,
      server: context.server,
      workerDeathNoted: connection.workerDeathNoted,
      digest: digestRemoteClose(
        context.contextId,
        connection.connectionId,
        context.owner,
        context.server,
        connection.workerDeathNoted,
      ),
    });
    context.receipt = receipt;
    return receipt;
  }

  closeReceipt(owner: string, connectionId: string, contextId: string): CloseReceipt {
    const connection = this.requireConnection(owner, connectionId);
    const context = this.requireContext(connection, contextId);
    if (context.closeState === "unknown") {
      fail("close-unconfirmed", `close confirmation is lost for context: ${contextId}`);
    }
    if (context.closeState !== "closed" || context.receipt === null) {
      fail("orphan-open", `context close is still open: ${contextId}`);
    }
    return context.receipt as CloseReceipt;
  }
}

// ---------------------------------------------------------------------------
// Cleanup verdict: only the independent authority judges
// ---------------------------------------------------------------------------

function checkRemoteProofShape(value: unknown): RemoteCleanupProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-close", "cleanup proof must be an independent close object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "external-close") {
    const receipt = record["receipt"] as CloseReceipt;
    if (receipt === null || typeof receipt !== "object" || Array.isArray(receipt)) {
      fail("forbidden-close", "independent close carries no close receipt");
    }
    return { kind: "external-close", receipt };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-close", `cleanup proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-close", "cleanup proof carries no kind");
}

// Judge one cleanup proof claim. Only a verified external-close receipt
// proves cleanup: worker receipts, server logs, exit codes, and any other
// non-authority claim reject with forbidden-close before any verdict is
// read, and a forged receipt never verifies.
export function assertRemoteCleanupWitnessed(claim: unknown): RemoteCleanupVerdict {
  const proof = checkRemoteProofShape(claim);
  if (proof.kind !== "external-close") {
    fail("forbidden-close", `cleanup proof kind is inadmissible: ${proof.kind}`);
  }
  const receipt = (proof as { kind: "external-close"; receipt: CloseReceipt }).receipt;
  if (
    typeof receipt.contextId !== "string" ||
    typeof receipt.connectionId !== "string" ||
    typeof receipt.owner !== "string" ||
    typeof receipt.server !== "string" ||
    typeof receipt.workerDeathNoted !== "boolean" ||
    typeof receipt.digest !== "string"
  ) {
    fail("forbidden-close", "independent close carries a malformed receipt");
  }
  const recomputed = digestRemoteClose(
    receipt.contextId,
    receipt.connectionId,
    receipt.owner,
    receipt.server,
    receipt.workerDeathNoted,
  );
  if (recomputed !== receipt.digest) {
    fail("forbidden-close", "close receipt digest does not verify");
  }
  return Object.freeze({
    contextId: receipt.contextId,
    connectionId: receipt.connectionId,
    witnessed: true,
    digest: digestText(`verdict:${receipt.contextId}:${receipt.connectionId}`),
  });
}

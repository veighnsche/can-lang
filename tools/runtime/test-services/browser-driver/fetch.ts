// K14: typed bounded page-realm Fetch with credentials and redirect policy.
//
// A page-origin Fetch service over in-memory doubles only: no network, no
// sockets, no fetch, no timers, no I/O. The service binds owned pages (each
// with a declared page origin and CSP), opens typed fetches with explicit
// credentials and redirect policy, applies cookie/origin/CSP page policy
// from its own seeded jar alone, and surfaces redirects as facts the caller
// must follow explicitly. Responses never arrive on their own and are never
// followed or retried by the service.
//
// Independence scope from the external HTTP doubles under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   external HTTP code, doubles, receipts, or state. The only strings that
//   may coincide are the page/fetch names under test (supplied by the test,
//   not by the doubles) and the vocabulary this file uses for facts.
//   PROVES: page admission, typed fetch begin with credentials and redirect
//   policy, cookie/origin/CSP page-policy facts that name the page realm,
//   page origin, and page policy, accepted-vs-consumed body accounting,
//   explicit redirect chaining, and error provenance (every failure names
//   its layer). External claims are compared AGAINST these facts; the
//   service never derives a fact FROM an external claim.
//   Consequently cookie/origin/CSP facts from this service are never
//   confusable with K20 external HTTP facts: every fact carries
//   realm "page-fetch", the page origin, and the page policy, while K20
//   facts carry kind "http-request" with an external origin and no page.
//
// No source eval: no descriptor accepts code, source, or script arguments;
// any descriptor carrying those keys rejects with forbidden-eval before
// any other validation or effect.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_FETCH_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_FETCH_INDEPENDENCE_SCOPE: string =
  "independent: shares no external HTTP code, doubles, receipts, or state; " +
  "proves page admission, typed fetch begin with credentials and redirect " +
  "policy, cookie/origin/CSP page-policy facts naming the page realm, page " +
  "origin, and page policy, accepted-vs-consumed accounting, explicit " +
  "redirect chaining, and layered error provenance from its own seeded " +
  "doubles alone";

// The page realm marker. Every fact this service emits carries it, so page
// Fetch facts are never confusable with external HTTP facts.
export const PAGE_FETCH_REALM = "page-fetch" as const;

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_FETCH_LAYERS = ["page", "fetch", "policy"] as const;
export type BrowserFetchLayer = (typeof BROWSER_FETCH_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   page: page admission, ownership, capacity, lifecycle.
//   fetch: fetch binding, tokens, targets, methods, descriptors, bodies,
//     redirect chaining mechanics, settlement.
//   policy: declared origins, credentials/redirect vocabulary enforcement,
//     origin/CSP decisions, source-eval refusal.
export const BROWSER_FETCH_CODES = [
  "unknown-page",
  "no-page",
  "wrong-owner",
  "capacity-exhausted",
  "page-closed",
  "forged-token",
  "unknown-fetch",
  "fetch-closed",
  "fetch-open",
  "fetch-settled",
  "response-pending",
  "unsupported-method",
  "body-limit",
  "unknown-origin",
  "unknown-credentials",
  "unknown-redirect",
  "origin-blocked",
  "csp-blocked",
  "forbidden-eval",
  "redirect-open",
] as const;
export type BrowserFetchCode = (typeof BROWSER_FETCH_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserFetchCode, BrowserFetchLayer>> = {
  "unknown-page": "page",
  "no-page": "page",
  "wrong-owner": "page",
  "capacity-exhausted": "page",
  "page-closed": "page",
  "forged-token": "fetch",
  "unknown-fetch": "fetch",
  "fetch-closed": "fetch",
  "fetch-open": "fetch",
  "fetch-settled": "fetch",
  "response-pending": "fetch",
  "unsupported-method": "fetch",
  "body-limit": "fetch",
  "unknown-origin": "policy",
  "unknown-credentials": "policy",
  "unknown-redirect": "policy",
  "origin-blocked": "policy",
  "csp-blocked": "policy",
  "forbidden-eval": "policy",
  "redirect-open": "policy",
};

export function layerOfFetchCode(code: BrowserFetchCode): BrowserFetchLayer {
  return CODE_LAYER[code];
}

export class BrowserFetchError extends Error {
  readonly layer: BrowserFetchLayer;
  readonly code: BrowserFetchCode;
  constructor(code: BrowserFetchCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserFetchError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserFetchCode, message: string): never {
  throw new BrowserFetchError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserFetchLimits = Readonly<{
  maxPages: number;
  maxFetchesPerPage: number;
  maxCookiesPerPage: number;
  maxBodyBytes: number;
  maxRedirectHops: number;
}>;

const LIMIT_KEYS = [
  "maxPages",
  "maxFetchesPerPage",
  "maxCookiesPerPage",
  "maxBodyBytes",
  "maxRedirectHops",
] as const;

export function checkFetchLimits(value: unknown): BrowserFetchLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserFetchError("unknown-page", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserFetchError("unknown-page", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    // maxRedirectHops may be 0 (explicit redirect chaining disabled by
    // bounds); every other bound must admit at least one.
    const min = key === "maxRedirectHops" ? 0 : 1;
    if (!Number.isSafeInteger(entry) || (entry as number) < min) {
      throw new BrowserFetchError("unknown-page", `${key} must be an integer of at least ${min}`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserFetchLimits;
}

// Finite request-method vocabulary.
export const FETCH_METHODS = ["GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"] as const;
export type FetchMethod = (typeof FETCH_METHODS)[number];

// Finite credentials vocabulary. There is no default; every fetch states
// its credentials mode.
export const FETCH_CREDENTIALS = ["omit", "same-origin", "include"] as const;
export type FetchCredentials = (typeof FETCH_CREDENTIALS)[number];

// Finite redirect vocabulary. There is no default; every fetch states its
// redirect policy. follow never follows implicitly: the caller must invoke
// followFetchRedirect explicitly per hop.
export const FETCH_REDIRECTS = ["follow", "error", "manual"] as const;
export type FetchRedirect = (typeof FETCH_REDIRECTS)[number];

// Finite page CSP vocabulary, declared per page grant.
export const FETCH_CSP_POLICIES = ["connect-self-only", "deny-all"] as const;
export type FetchCspPolicy = (typeof FETCH_CSP_POLICIES)[number];

// Explicit decoding scope: how delivered body bytes decode. There is no
// default; every delivered response states its scope.
export const FETCH_DECODING_SCOPES = ["raw-bytes", "utf8-strict", "utf8-replace"] as const;
export type FetchDecodingScope = (typeof FETCH_DECODING_SCOPES)[number];

// Finite response-status vocabulary.
export const FETCH_STATUSES = [200, 201, 302, 400, 404, 500] as const;
export type FetchStatus = (typeof FETCH_STATUSES)[number];

export const FETCH_REDIRECT_STATUS: FetchStatus = 302;

// Source-eval keys: no descriptor anywhere in this service accepts them.
export const FORBIDDEN_EVAL_KEYS = ["code", "source", "script"] as const;

const MAX_NAME_LEN = 128;
const BODY_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;
const ORIGIN_RE = /^https?:\/\/[A-Za-z0-9][A-Za-z0-9.-]*$/;
const PATH_RE = /^\/[A-Za-z0-9_.:~/=-]*$/;

function checkName(value: unknown, what: string, code: BrowserFetchCode): string {
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

function checkPageName(value: unknown): string {
  return checkName(value, "page", "unknown-page");
}

function checkFetchName(value: unknown): string {
  return checkName(value, "fetch", "unknown-fetch");
}

function checkOrigin(value: unknown, what: string): string {
  if (typeof value !== "string" || value.length > MAX_NAME_LEN || !ORIGIN_RE.test(value)) {
    fail("unknown-origin", `${what} must be an http(s) origin host, got: ${String(value)}`);
  }
  return value as string;
}

function checkTargetPath(value: unknown): string {
  if (typeof value !== "string" || value.length > MAX_NAME_LEN || !PATH_RE.test(value)) {
    fail("unknown-fetch", `target path must start with /, got: ${String(value)}`);
  }
  return value as string;
}

function checkMethod(value: unknown): FetchMethod {
  if (typeof value !== "string" || !(FETCH_METHODS as readonly string[]).includes(value)) {
    fail("unsupported-method", `not a declared fetch method: ${String(value)}`);
  }
  return value as FetchMethod;
}

function checkCredentials(value: unknown): FetchCredentials {
  if (typeof value !== "string" || !(FETCH_CREDENTIALS as readonly string[]).includes(value)) {
    fail("unknown-credentials", `credentials mode must be explicit, got: ${String(value)}`);
  }
  return value as FetchCredentials;
}

function checkRedirect(value: unknown): FetchRedirect {
  if (typeof value !== "string" || !(FETCH_REDIRECTS as readonly string[]).includes(value)) {
    fail("unknown-redirect", `redirect policy must be explicit, got: ${String(value)}`);
  }
  return value as FetchRedirect;
}

function checkCsp(value: unknown): FetchCspPolicy {
  if (typeof value !== "string" || !(FETCH_CSP_POLICIES as readonly string[]).includes(value)) {
    fail("csp-blocked", `not a declared page CSP: ${String(value)}`);
  }
  return value as FetchCspPolicy;
}

function checkDecodingScope(value: unknown): FetchDecodingScope {
  if (typeof value !== "string" || !(FETCH_DECODING_SCOPES as readonly string[]).includes(value)) {
    fail("unknown-fetch", `decoding scope must be explicit, got: ${String(value)}`);
  }
  return value as FetchDecodingScope;
}

function checkStatus(value: unknown): FetchStatus {
  if (typeof value !== "number" || !(FETCH_STATUSES as readonly number[]).includes(value)) {
    fail("unknown-fetch", `not a declared response status: ${String(value)}`);
  }
  return value as FetchStatus;
}

function checkDescriptor(value: unknown, what: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("unknown-fetch", `${what} must be an object`);
  }
  return value as Record<string, unknown>;
}

// Source-eval refusal runs before any other descriptor validation: a
// descriptor carrying code, source, or script rejects with forbidden-eval
// regardless of whatever else it carries.
function refuseEvalKeys(record: Record<string, unknown>, what: string): void {
  for (const key of FORBIDDEN_EVAL_KEYS) {
    if (key in record) {
      fail("forbidden-eval", `${what} carries a source-eval argument: ${key}`);
    }
  }
}

function checkNoExtraKeys(
  record: Record<string, unknown>,
  keys: readonly string[],
  what: string,
): void {
  for (const key of Object.keys(record)) {
    if (!keys.includes(key)) {
      fail("unknown-fetch", `${what} carries an unknown field: ${key}`);
    }
  }
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only)
// ---------------------------------------------------------------------------

export type PageBindingFacts = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  owner: string;
  page: string;
  pageOrigin: string;
  csp: FetchCspPolicy;
  handleDigest: string;
}>;

export type CookieSeedReceipt = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  page: string;
  cookieJar: string;
  cookies: number;
  jarDigest: string;
}>;

export type FetchBeginFacts = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  fetchId: string;
  owner: string;
  page: string;
  pageOrigin: string;
  method: FetchMethod;
  targetOrigin: string;
  targetPath: string;
  sameOrigin: boolean;
  credentials: FetchCredentials;
  redirect: FetchRedirect;
  csp: FetchCspPolicy;
  cspDecision: "allowed";
  cookieJar: string;
  cookiesSeeded: number;
  cookiesSent: number;
  originHeader: string | null;
  hops: number;
  supersedes: string | null;
  digest: string;
}>;

export type FetchBlockedFacts = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  fetchId: string;
  owner: string;
  page: string;
  pageOrigin: string;
  targetOrigin: string;
  sameOrigin: boolean;
  csp: FetchCspPolicy;
  cspDecision: "blocked";
  blockedBy: "csp" | "origin";
  cookieJar: string;
  cookiesSent: 0;
  digest: string;
}>;

export type FetchRedirectFacts = Readonly<{
  status: FetchStatus;
  location: string;
}>;

export type FetchResponseFacts = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  fetchId: string;
  status: FetchStatus;
  decodingScope: FetchDecodingScope;
  byteLength: number;
  bodyDigest: string;
  bodyAccepted: number;
  bodyConsumed: number;
  redirect: FetchRedirectFacts | null;
  redirectIncomplete: boolean;
  digest: string;
}>;

export type FetchConsumeReceipt = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  fetchId: string;
  consumed: number;
  consumedTotal: number;
  acceptedTotal: number;
  eof: boolean;
}>;

export type FetchCloseReceipt = Readonly<{
  realm: typeof PAGE_FETCH_REALM;
  fetchId: string;
  owner: string;
  page: string;
  pageOrigin: string;
  cspDecision: "allowed" | "blocked";
  cookiesSent: number;
  bodyAccepted: number;
  bodyConsumed: number;
  bodyUnread: number;
  digest: string;
}>;

export type FetchDescriptor = Readonly<{
  method: FetchMethod;
  targetOrigin: string;
  targetPath: string;
  credentials: FetchCredentials;
  redirect: FetchRedirect;
}>;

export type FetchResponseDescriptor = Readonly<{
  status: FetchStatus;
  decodingScope: FetchDecodingScope;
  byteLength: number;
  bodyDigest: string;
  location: string | null;
}>;

type FetchState = "open" | "responded" | "redirect-held" | "blocked" | "closed";

type FetchRecord = {
  fetchId: string;
  owner: string;
  page: string;
  method: FetchMethod | null;
  targetOrigin: string;
  targetPath: string | null;
  sameOrigin: boolean;
  credentials: FetchCredentials | null;
  redirect: FetchRedirect | null;
  cspDecision: "allowed" | "blocked";
  blockedBy: "csp" | "origin" | null;
  cookiesSent: number;
  originHeader: string | null;
  state: FetchState;
  status: FetchStatus | null;
  decodingScope: FetchDecodingScope | null;
  bodyAccepted: number;
  bodyConsumed: number;
  bodyDigest: string | null;
  redirectLocation: string | null;
  redirectIncomplete: boolean;
  hops: number;
  supersedes: string | null;
};

type PageState = "open" | "closed";

type PageRecord = {
  page: string;
  owner: string;
  pageOrigin: string;
  csp: FetchCspPolicy;
  token: string;
  state: PageState;
  cookies: number;
  fetches: Map<string, FetchRecord>;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

function cookieJarOf(page: string): string {
  return `page-jar:${page}`;
}

// The fetch digest is recomputed from carried fields, so a presented fact
// can be re-verified instead of trusting its digest string.
export function digestPageFetch(
  fetchId: string,
  owner: string,
  page: string,
  pageOrigin: string,
  targetOrigin: string,
  flags: string,
): string {
  const hash = createHash("sha256");
  hash.update(PAGE_FETCH_REALM, "utf8");
  hash.update("\0", "utf8");
  hash.update(fetchId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(page, "utf8");
  hash.update("\0", "utf8");
  hash.update(pageOrigin, "utf8");
  hash.update("\0", "utf8");
  hash.update(targetOrigin, "utf8");
  hash.update("\0", "utf8");
  hash.update(flags, "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned pages, typed fetches, page policy, explicit redirects
// ---------------------------------------------------------------------------

export type PageGrant = Readonly<{
  owner: string;
  page: string;
  pageOrigin: string;
  csp: FetchCspPolicy;
}>;

export class PageFetchService {
  private readonly limits: BrowserFetchLimits;
  private readonly grants = new Map<string, PageGrant>();
  private readonly origins: readonly string[];
  private readonly pages = new Map<string, PageRecord>();

  constructor(
    grants: readonly PageGrant[],
    declaredOrigins: readonly string[],
    limits: BrowserFetchLimits,
  ) {
    if (grants.length === 0) {
      throw new BrowserFetchError("unknown-page", "declare at least one owned page");
    }
    if (grants.length > limits.maxPages) {
      throw new BrowserFetchError("capacity-exhausted", "declared pages exceed the cap");
    }
    if (declaredOrigins.length === 0) {
      throw new BrowserFetchError("unknown-origin", "declare at least one fetch target origin");
    }
    const seenOrigins = new Set<string>();
    for (const origin of declaredOrigins) {
      checkOrigin(origin, "declared origin");
      if (seenOrigins.has(origin)) {
        fail("unknown-origin", `duplicate declared origin: ${origin}`);
      }
      seenOrigins.add(origin);
    }
    for (const grant of grants) {
      checkOwner(grant.owner);
      checkPageName(grant.page);
      checkOrigin(grant.pageOrigin, "page origin");
      checkCsp(grant.csp);
      const key = `${grant.owner}\0${grant.page}`;
      if (this.grants.has(key)) {
        fail("unknown-page", `duplicate owned page: ${grant.owner}/${grant.page}`);
      }
      this.grants.set(
        key,
        Object.freeze({
          owner: grant.owner,
          page: grant.page,
          pageOrigin: grant.pageOrigin,
          csp: grant.csp,
        }),
      );
    }
    this.origins = [...declaredOrigins];
    this.limits = limits;
  }

  get pageCount(): number {
    return this.pages.size;
  }

  fetchCount(owner: string, page: string): number {
    return this.requirePage(owner, page).fetches.size;
  }

  // Bind one owned page. Only an exact declared (owner, page) grant admits;
  // a page the test never granted rejects before any effect. Pages bind to
  // one owner: a second identity joining the same page name fails the
  // owner check.
  bindPage(owner: string, page: string): PageBindingFacts {
    checkOwner(owner);
    checkPageName(page);
    const grant = this.grants.get(`${owner}\0${page}`);
    if (grant === undefined) {
      fail("unknown-page", `no owned grant for page: ${owner}/${page}`);
    }
    const declared = grant as PageGrant;
    const prior = this.pages.get(page);
    if (prior !== undefined && prior.state !== "closed") {
      if (prior.owner !== owner) {
        fail("wrong-owner", `page is owned by another identity: ${page}`);
      }
      return this.bindingFactsOf(prior);
    }
    if (this.pages.size >= this.limits.maxPages) {
      fail("capacity-exhausted", "page table full");
    }
    const record: PageRecord = {
      page,
      owner,
      pageOrigin: declared.pageOrigin,
      csp: declared.csp,
      token: mintToken("page"),
      state: "open",
      cookies: 0,
      fetches: new Map(),
    };
    this.pages.set(page, record);
    return this.bindingFactsOf(record);
  }

  private bindingFactsOf(record: PageRecord): PageBindingFacts {
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      owner: record.owner,
      page: record.page,
      pageOrigin: record.pageOrigin,
      csp: record.csp,
      handleDigest: digestText(record.token),
    });
  }

  binding(owner: string, page: string): PageBindingFacts {
    return this.bindingFactsOf(this.requirePage(owner, page));
  }

  private requirePage(owner: string, page: string): PageRecord {
    const record = this.pages.get(page);
    if (record === undefined) {
      fail("unknown-page", `no bound page for owner: ${owner}/${page}`);
    }
    const pageRecord = record as PageRecord;
    if (pageRecord.owner !== owner) {
      fail("wrong-owner", "page is owned by another identity");
    }
    if (pageRecord.state === "closed") {
      fail("page-closed", `page is closed: ${owner}/${page}`);
    }
    return pageRecord;
  }

  // Admission-first: the fetch gate reports only whether a LIVE binding
  // exists. Never bound, closed, or foreign all refuse with no-page: a
  // missing page blocks Fetch, whatever the reason.
  private requireLiveBinding(owner: string, page: string): PageRecord {
    checkOwner(owner);
    checkPageName(page);
    const record = this.pages.get(page);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-page", `no live page binding for fetch: ${owner}/${page}`);
    }
    return record as PageRecord;
  }

  // Verify the page token by table lookup. Invented tokens are never
  // authority.
  private requireLivePage(owner: string, page: string, token: string): { record: PageRecord } {
    const record = this.requirePage(owner, page);
    if (token === "" || token !== record.token) {
      fail("forged-token", "page token is not the attested token");
    }
    return { record };
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  pageTokenForTest(owner: string, page: string): string {
    return this.requirePage(owner, page).token;
  }

  closePage(owner: string, page: string): void {
    const record = this.requirePage(owner, page);
    for (const fetch of record.fetches.values()) {
      if (fetch.state !== "closed") {
        fail("fetch-open", `page still holds an open fetch: ${fetch.fetchId}`);
      }
    }
    record.state = "closed";
  }

  // Seed the page cookie jar with a bounded cookie count. The jar belongs
  // to the page realm: facts name it page-jar:<page>, never an external
  // cookie store.
  seedPageCookies(owner: string, page: string, token: string, count: number): CookieSeedReceipt {
    const { record } = this.requireLivePage(owner, page, token);
    if (!Number.isSafeInteger(count) || count < 0) {
      fail("unknown-fetch", `cookie count must be a non-negative integer, got: ${String(count)}`);
    }
    if (record.cookies + (count as number) > this.limits.maxCookiesPerPage) {
      fail("capacity-exhausted", "page cookie jar full");
    }
    record.cookies += count as number;
    const jar = cookieJarOf(record.page);
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      page: record.page,
      cookieJar: jar,
      cookies: record.cookies,
      jarDigest: digestText(`${jar}:${record.cookies}`),
    });
  }

  private checkFetchDescriptor(value: unknown): FetchDescriptor {
    const record = checkDescriptor(value, "fetch");
    refuseEvalKeys(record, "fetch");
    checkNoExtraKeys(
      record,
      ["method", "targetOrigin", "targetPath", "credentials", "redirect"],
      "fetch",
    );
    const method = checkMethod(record["method"]);
    const targetOrigin = checkOrigin(record["targetOrigin"], "fetch target origin");
    const targetPath = checkTargetPath(record["targetPath"]);
    const credentials = checkCredentials(record["credentials"]);
    const redirect = checkRedirect(record["redirect"]);
    return { method, targetOrigin, targetPath, credentials, redirect };
  }

  // Begin one typed page-realm fetch. The descriptor is validated first
  // with source-eval refusal before any other check; then live-binding
  // admission, the declared-origin gate, and the page policy decide:
  // deny-all CSP blocks every target, and connect-self-only blocks
  // cross-origin targets. A blocked fetch still returns frozen facts
  // proving the policy decision, and settles immediately: delivery and
  // reads refuse it.
  fetchBegin(
    owner: string,
    page: string,
    token: string,
    fetchId: string,
    descriptor: unknown,
  ): FetchBeginFacts | FetchBlockedFacts {
    const request = this.checkFetchDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, page);
    checkFetchName(fetchId);
    if (token === "" || token !== record.token) {
      fail("forged-token", "page token is not the attested token");
    }
    if (!(this.origins as readonly string[]).includes(request.targetOrigin)) {
      fail("unknown-origin", `target origin is outside the declared set: ${request.targetOrigin}`);
    }
    if (record.fetches.has(fetchId)) {
      fail("fetch-settled", `fetch id is single-use on this page: ${fetchId}`);
    }
    if (record.fetches.size >= this.limits.maxFetchesPerPage) {
      fail("capacity-exhausted", "fetch table full");
    }
    const sameOrigin = request.targetOrigin === record.pageOrigin;
    const blockedBy =
      record.csp === "deny-all" ? ("csp" as const) : !sameOrigin ? ("origin" as const) : null;
    if (blockedBy !== null) {
      const blocked: FetchRecord = {
        fetchId,
        owner,
        page,
        method: request.method,
        targetOrigin: request.targetOrigin,
        targetPath: request.targetPath,
        sameOrigin,
        credentials: request.credentials,
        redirect: request.redirect,
        cspDecision: "blocked",
        blockedBy,
        cookiesSent: 0,
        originHeader: null,
        state: "blocked",
        status: null,
        decodingScope: null,
        bodyAccepted: 0,
        bodyConsumed: 0,
        bodyDigest: null,
        redirectLocation: null,
        redirectIncomplete: false,
        hops: 0,
        supersedes: null,
      };
      record.fetches.set(fetchId, blocked);
      return Object.freeze({
        realm: PAGE_FETCH_REALM,
        fetchId,
        owner,
        page,
        pageOrigin: record.pageOrigin,
        targetOrigin: request.targetOrigin,
        sameOrigin,
        csp: record.csp,
        cspDecision: "blocked",
        blockedBy,
        cookieJar: cookieJarOf(record.page),
        cookiesSent: 0,
        digest: digestPageFetch(
          fetchId,
          owner,
          page,
          record.pageOrigin,
          request.targetOrigin,
          `blocked:${blockedBy}`,
        ),
      });
    }
    const cookiesSent =
      request.credentials === "omit"
        ? 0
        : request.credentials === "include"
          ? record.cookies
          : sameOrigin
            ? record.cookies
            : 0;
    const originHeader =
      request.method === "GET" || request.method === "HEAD" ? null : record.pageOrigin;
    const entry: FetchRecord = {
      fetchId,
      owner,
      page,
      method: request.method,
      targetOrigin: request.targetOrigin,
      targetPath: request.targetPath,
      sameOrigin,
      credentials: request.credentials,
      redirect: request.redirect,
      cspDecision: "allowed",
      blockedBy: null,
      cookiesSent,
      originHeader,
      state: "open",
      status: null,
      decodingScope: null,
      bodyAccepted: 0,
      bodyConsumed: 0,
      bodyDigest: null,
      redirectLocation: null,
      redirectIncomplete: false,
      hops: 0,
      supersedes: null,
    };
    record.fetches.set(fetchId, entry);
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      fetchId,
      owner,
      page,
      pageOrigin: record.pageOrigin,
      method: request.method,
      targetOrigin: request.targetOrigin,
      targetPath: request.targetPath,
      sameOrigin,
      credentials: request.credentials,
      redirect: request.redirect,
      csp: record.csp,
      cspDecision: "allowed",
      cookieJar: cookieJarOf(record.page),
      cookiesSeeded: record.cookies,
      cookiesSent,
      originHeader,
      hops: 0,
      supersedes: null,
      digest: digestPageFetch(
        fetchId,
        owner,
        page,
        record.pageOrigin,
        request.targetOrigin,
        `allowed:${request.method}:${request.credentials}:${request.redirect}:${cookiesSent}`,
      ),
    });
  }

  private requireFetch(record: PageRecord, fetchId: string): FetchRecord {
    const fetch = record.fetches.get(fetchId);
    if (fetch === undefined) {
      fail("unknown-fetch", `no such page fetch: ${fetchId}`);
    }
    return fetch as FetchRecord;
  }

  private checkFetchResponseDescriptor(value: unknown): FetchResponseDescriptor {
    const record = checkDescriptor(value, "fetch response");
    refuseEvalKeys(record, "fetch response");
    checkNoExtraKeys(
      record,
      ["status", "decodingScope", "byteLength", "bodyDigest", "location"],
      "fetch response",
    );
    const status = checkStatus(record["status"]);
    const decodingScope = checkDecodingScope(record["decodingScope"]);
    const byteLength = record["byteLength"];
    if (!Number.isSafeInteger(byteLength) || (byteLength as number) < 0) {
      fail(
        "unknown-fetch",
        `body byteLength must be a non-negative integer, got: ${String(byteLength)}`,
      );
    }
    const bodyDigest = record["bodyDigest"];
    if (typeof bodyDigest !== "string" || !BODY_DIGEST_RE.test(bodyDigest)) {
      fail("unknown-fetch", "fetch response carries a body digest, never raw bytes");
    }
    const location = record["location"];
    if (location !== null && location !== undefined) {
      checkOrigin(location, "redirect location origin");
    }
    return {
      status,
      decodingScope,
      byteLength: byteLength as number,
      bodyDigest: bodyDigest as string,
      location: (location as string | null | undefined) ?? null,
    };
  }

  // Deliver the seeded server response to one open fetch. Responses never
  // arrive on their own. A 302 surfaces redirect facts; the service never
  // follows them: under follow the fetch holds redirect-open until an
  // explicit followFetchRedirect, and under error/manual the fetch settles
  // with the redirect facts carried.
  deliverFetchResponse(
    owner: string,
    page: string,
    token: string,
    fetchId: string,
    response: unknown,
  ): FetchResponseFacts {
    const descriptor = this.checkFetchResponseDescriptor(response);
    const { record } = this.requireLivePage(owner, page, token);
    const fetch = this.requireFetch(record, fetchId);
    if (fetch.state === "closed") {
      fail("fetch-closed", `fetch is closed: ${fetchId}`);
    }
    if (fetch.state === "blocked") {
      fail(
        fetch.blockedBy === "csp" ? "csp-blocked" : "origin-blocked",
        `fetch was blocked by page policy: ${fetchId}`,
      );
    }
    if (fetch.state !== "open") {
      fail("fetch-settled", `fetch already has its response: ${fetchId}`);
    }
    if (descriptor.byteLength > this.limits.maxBodyBytes) {
      fail("body-limit", "delivered body exceeds the body cap");
    }
    if (
      descriptor.location !== null &&
      !(this.origins as readonly string[]).includes(descriptor.location)
    ) {
      fail(
        "unknown-origin",
        `redirect location is outside the declared set: ${descriptor.location}`,
      );
    }
    fetch.status = descriptor.status;
    fetch.decodingScope = descriptor.decodingScope;
    fetch.bodyAccepted = descriptor.byteLength;
    fetch.bodyDigest = descriptor.bodyDigest;
    const isRedirect = descriptor.status === FETCH_REDIRECT_STATUS;
    fetch.redirectLocation = isRedirect ? descriptor.location : null;
    fetch.redirectIncomplete = isRedirect && descriptor.location === null;
    const holdsRedirect = isRedirect && fetch.redirect === "follow" && descriptor.location !== null;
    fetch.state = holdsRedirect ? "redirect-held" : "responded";
    const redirect =
      isRedirect && descriptor.location !== null
        ? Object.freeze({ status: descriptor.status, location: descriptor.location })
        : null;
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      fetchId,
      status: descriptor.status,
      decodingScope: descriptor.decodingScope,
      byteLength: descriptor.byteLength,
      bodyDigest: descriptor.bodyDigest,
      bodyAccepted: descriptor.byteLength,
      bodyConsumed: 0,
      redirect,
      redirectIncomplete: fetch.redirectIncomplete,
      digest: digestPageFetch(
        fetchId,
        owner,
        page,
        record.pageOrigin,
        fetch.targetOrigin,
        `responded:${descriptor.status}:${descriptor.decodingScope}:${descriptor.byteLength}:${descriptor.location ?? ""}`,
      ),
    });
  }

  // Follow one held redirect explicitly. Only a redirect-held fetch under
  // the follow policy with remaining hop budget mints a chained fetch, and
  // the chained target must be a declared origin. The parent settles as
  // responded; its redirect facts stay readable on the receipt.
  followFetchRedirect(
    owner: string,
    page: string,
    token: string,
    fetchId: string,
    nextFetchId: string,
  ): FetchBeginFacts | FetchBlockedFacts {
    checkFetchName(nextFetchId);
    const { record } = this.requireLivePage(owner, page, token);
    const fetch = this.requireFetch(record, fetchId);
    if (fetch.state === "closed") {
      fail("fetch-closed", `fetch is closed: ${fetchId}`);
    }
    if (fetch.state !== "redirect-held") {
      fail("redirect-open", `no held redirect to follow on fetch: ${fetchId}`);
    }
    if (fetch.redirect !== "follow") {
      fail("redirect-open", `redirect policy forbids following on fetch: ${fetchId}`);
    }
    if (fetch.redirectLocation === null) {
      fail("redirect-open", `held redirect carries no location on fetch: ${fetchId}`);
    }
    if (fetch.hops + 1 > this.limits.maxRedirectHops) {
      fail("redirect-open", `redirect hop budget exhausted on fetch: ${fetchId}`);
    }
    if (record.fetches.has(nextFetchId)) {
      fail("fetch-settled", `fetch id is single-use on this page: ${nextFetchId}`);
    }
    if (record.fetches.size >= this.limits.maxFetchesPerPage) {
      fail("capacity-exhausted", "fetch table full");
    }
    fetch.state = "responded";
    const chained = this.fetchBegin(owner, page, token, nextFetchId, {
      method: "GET",
      targetOrigin: fetch.redirectLocation,
      targetPath: "/",
      credentials: fetch.credentials as FetchCredentials,
      redirect: fetch.redirect as FetchRedirect,
    });
    const child = this.requireFetch(record, nextFetchId);
    child.hops = fetch.hops + 1;
    child.supersedes = fetchId;
    if (chained.cspDecision === "allowed") {
      return Object.freeze({
        ...(chained as FetchBeginFacts),
        hops: child.hops,
        supersedes: fetchId,
      });
    }
    return chained;
  }

  // Consume delivered body bytes by count. Facts carry digests only: raw
  // body bytes never cross. Accepted (admitted by delivery) and consumed
  // (handed out here) are independent counters; eof is true only after the
  // delivered body is fully consumed.
  consumeFetchBody(
    owner: string,
    page: string,
    token: string,
    fetchId: string,
    byteCount: number,
  ): FetchConsumeReceipt {
    const { record } = this.requireLivePage(owner, page, token);
    const fetch = this.requireFetch(record, fetchId);
    if (fetch.state === "closed") {
      fail("fetch-closed", `fetch is closed: ${fetchId}`);
    }
    if (fetch.state === "blocked") {
      fail(
        fetch.blockedBy === "csp" ? "csp-blocked" : "origin-blocked",
        `fetch was blocked by page policy: ${fetchId}`,
      );
    }
    if (fetch.state === "open") {
      fail("response-pending", `no delivered response to consume on fetch: ${fetchId}`);
    }
    if (!Number.isSafeInteger(byteCount) || (byteCount as number) <= 0) {
      fail("unknown-fetch", `consume count must be a positive integer, got: ${String(byteCount)}`);
    }
    const remaining = fetch.bodyAccepted - fetch.bodyConsumed;
    const consumed = Math.min(byteCount as number, remaining);
    fetch.bodyConsumed += consumed;
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      fetchId,
      consumed,
      consumedTotal: fetch.bodyConsumed,
      acceptedTotal: fetch.bodyAccepted,
      eof: fetch.bodyConsumed >= fetch.bodyAccepted,
    });
  }

  // Close one settled fetch. Open fetches refuse with response-pending and
  // redirect-held fetches refuse with redirect-open: settlement stays
  // distinct from close, and a held redirect is never silently dropped.
  closeFetch(owner: string, page: string, token: string, fetchId: string): FetchCloseReceipt {
    const { record } = this.requireLivePage(owner, page, token);
    const fetch = this.requireFetch(record, fetchId);
    if (fetch.state === "closed") {
      fail("fetch-closed", `fetch is already closed: ${fetchId}`);
    }
    if (fetch.state === "open") {
      fail("response-pending", `fetch has no delivered response yet: ${fetchId}`);
    }
    if (fetch.state === "redirect-held") {
      fail("redirect-open", `fetch holds an unfollowed redirect: ${fetchId}`);
    }
    fetch.state = "closed";
    return Object.freeze({
      realm: PAGE_FETCH_REALM,
      fetchId,
      owner,
      page,
      pageOrigin: record.pageOrigin,
      cspDecision: fetch.cspDecision,
      cookiesSent: fetch.cookiesSent,
      bodyAccepted: fetch.bodyAccepted,
      bodyConsumed: fetch.bodyConsumed,
      bodyUnread: fetch.bodyAccepted - fetch.bodyConsumed,
      digest: digestPageFetch(
        fetchId,
        owner,
        page,
        record.pageOrigin,
        fetch.targetOrigin,
        `closed:${fetch.cspDecision}:${fetch.bodyAccepted}:${fetch.bodyConsumed}`,
      ),
    });
  }
}

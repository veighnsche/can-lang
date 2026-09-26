// C-G destination and redirect policy for tenant-selected webhooks (F01).
//
// Fixed-origin HTTP stays unless a policy here admits the destination:
// allowlist rules with exact or wildcard hosts, optional port and path
// narrowing, environment-name credential bindings, and explicit redirect
// handling. Unrestricted fetch stays rejected.
//
// Credentials are env-name-only: the policy names the variable, the
// companion resolves its value at send time, and the value never enters
// a decision, report, log line, or artifact. Reason codes and the
// policy report below are safe to log verbatim.
export type EnvName = string & { readonly __brand: "EnvName" };

// Same spelling as env::required: uppercase, digits, underscores.
const ENV_PATTERN = /^[A-Z_][A-Z0-9_]{0,127}$/;

export function envName(value: unknown): EnvName {
  if (typeof value !== "string" || !ENV_PATTERN.test(value))
    throw new TypeError("invalid credential environment name");
  return value as EnvName;
}

export type DestinationScheme = "http" | "https" | "any";

export type DestinationRule = Readonly<{
  scheme: DestinationScheme;
  host: string;
  port?: number;
  pathPrefix?: string;
  credential?: EnvName;
}>;

export type RedirectPolicy = "deny" | "same-origin" | "allowlisted";
export type NetworkScope = "deny" | "allow";

export type DestinationPolicy = Readonly<{
  version: string;
  rules: readonly DestinationRule[];
  redirect: RedirectPolicy;
  maxRedirectHops: number;
  loopback: NetworkScope;
  privateNetworks: NetworkScope;
}>;

const VERSION_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;
// Exact hosts are lowercase DNS names or IP literals; wildcards pin a
// parent suffix ("*.example.com" matches subdomains only, never the
// apex and never IP literals). No regex: rules stay auditable.
const DNS_PATTERN =
  /^(?=.{1,253}$)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$/;
const IPV4_PATTERN = /^\d{1,3}(?:\.\d{1,3}){3}$/;
const IPV6_PATTERN = /^[0-9a-f:]+(?:\.[0-9.]+)?$/;

function checkHost(value: unknown): string {
  if (typeof value !== "string") throw new TypeError("invalid destination host");
  const text = value.toLowerCase();
  const bare = text.startsWith("*.") ? text.slice(2) : text;
  if (bare.length === 0 || bare.length > 253) throw new TypeError("invalid destination host");
  // All-numeric hosts are IP literals or nothing: wildcards never take
  // them and exact entries must parse as IPv4.
  if (/^[0-9.]+$/.test(bare)) {
    if (text.startsWith("*.") || parseIPv4(bare) === undefined)
      throw new TypeError("invalid destination host");
    return text;
  }
  const valid = DNS_PATTERN.test(bare) || (!text.startsWith("*.") && IPV6_PATTERN.test(bare));
  if (!valid) throw new TypeError("invalid destination host");
  return text;
}

function checkPort(value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 1 || (value as number) > 65535)
    throw new TypeError("invalid destination port");
  return value as number;
}

function checkPathPrefix(value: unknown): string {
  if (typeof value !== "string" || !value.startsWith("/") || !value.isWellFormed())
    throw new TypeError("invalid destination path prefix");
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function checkRule(value: unknown): DestinationRule {
  if (!isRecord(value)) throw new TypeError("invalid destination rule");
  const scheme = value["scheme"];
  if (scheme !== "http" && scheme !== "https" && scheme !== "any")
    throw new TypeError("invalid destination scheme");
  const rule: {
    scheme: DestinationScheme;
    host: string;
    port?: number;
    pathPrefix?: string;
    credential?: EnvName;
  } = { scheme, host: checkHost(value["host"]) };
  if (value["port"] !== undefined) rule.port = checkPort(value["port"]);
  if (value["pathPrefix"] !== undefined) rule.pathPrefix = checkPathPrefix(value["pathPrefix"]);
  if (value["credential"] !== undefined) rule.credential = envName(value["credential"]);
  return Object.freeze(rule);
}

function checkScope(value: unknown, name: string): NetworkScope {
  if (value !== "deny" && value !== "allow") throw new TypeError(`invalid ${name} scope`);
  return value;
}

// destinationPolicy strictly validates a companion policy file's
// parsed JSON. Unknown shapes reject; the frozen result pins a version
// so delivery runs can report which policy admitted them.
export function destinationPolicy(init: unknown): DestinationPolicy {
  if (!isRecord(init)) throw new TypeError("invalid destination policy");
  const version = init["version"];
  if (typeof version !== "string" || !VERSION_PATTERN.test(version))
    throw new TypeError("invalid destination policy version");
  if (!Array.isArray(init["rules"])) throw new TypeError("invalid destination rules");
  const rules = Object.freeze(init["rules"].map(checkRule));
  const redirect = init["redirect"];
  if (redirect !== "deny" && redirect !== "same-origin" && redirect !== "allowlisted")
    throw new TypeError("invalid redirect policy");
  const maxRedirectHops = init["maxRedirectHops"];
  if (
    !Number.isSafeInteger(maxRedirectHops) ||
    (maxRedirectHops as number) < 0 ||
    (maxRedirectHops as number) > 8
  )
    throw new TypeError("invalid redirect hop bound");
  const hops = maxRedirectHops as number;
  return Object.freeze({
    version,
    rules,
    redirect,
    maxRedirectHops: hops,
    loopback: init["loopback"] === undefined ? "deny" : checkScope(init["loopback"], "loopback"),
    privateNetworks:
      init["privateNetworks"] === undefined
        ? "deny"
        : checkScope(init["privateNetworks"], "private-network"),
  });
}

export type DestinationDecision = Readonly<{
  decision: "allowed" | "denied";
  ruleIndex?: number;
  credential?: EnvName;
  reason: string;
}>;

const DEFAULT_PORTS: Readonly<Record<string, number>> = { "http:": 80, "https:": 443 };

// checkUrlShape enforces the scheme/userinfo/fragment/network-scope
// floor shared by direct evaluation and same-origin redirects: rule
// matching happens only after these pass.
function checkUrlShape(policy: DestinationPolicy, parsed: URL): DestinationDecision | undefined {
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:")
    return Object.freeze({ decision: "denied", reason: "unsupported-scheme" });
  if (parsed.username !== "" || parsed.password !== "")
    return Object.freeze({ decision: "denied", reason: "userinfo-forbidden" });
  if (parsed.hash !== "")
    return Object.freeze({ decision: "denied", reason: "fragment-forbidden" });
  const host = parsed.hostname.toLowerCase();
  if (isLoopbackLiteral(host) && policy.loopback === "deny")
    return Object.freeze({ decision: "denied", reason: "loopback-denied" });
  if (isPrivateLiteral(host) && policy.privateNetworks === "deny")
    return Object.freeze({ decision: "denied", reason: "private-network-denied" });
  return undefined;
}

function parseIPv4(host: string): readonly number[] | undefined {
  if (!IPV4_PATTERN.test(host)) return undefined;
  const parts = host.split(".").map(Number);
  if (parts.some((part) => part > 255)) return undefined;
  return parts;
}

function isLoopbackLiteral(host: string): boolean {
  const v4 = parseIPv4(host);
  if (v4 !== undefined) return v4[0] === 127;
  // URL.hostname strips brackets; "::1" and its expanded forms compare
  // here without a full parser.
  return host === "::1" || host === "0:0:0:0:0:0:0:1";
}

function isPrivateLiteral(host: string): boolean {
  const v4 = parseIPv4(host);
  if (v4 !== undefined) {
    return (
      v4[0] === 10 ||
      (v4[0] === 172 && v4[1] >= 16 && v4[1] <= 31) ||
      (v4[0] === 192 && v4[1] === 168) ||
      (v4[0] === 169 && v4[1] === 254) ||
      v4[0] === 127 ||
      (v4[0] === 0 && v4[1] === 0 && v4[2] === 0 && v4[3] === 0)
    );
  }
  if (!host.includes(":")) return false;
  const head = host.toLowerCase().split(":")[0];
  // Unique-local fc00::/7, link-local fe80::/10, unspecified, loopback.
  return (
    head.startsWith("fc") ||
    head.startsWith("fd") ||
    head === "fe80" ||
    head === "fe81" ||
    head === "fe82" ||
    head === "fe83" ||
    /^0+$/.test(head) ||
    host === "::1" ||
    host === "0:0:0:0:0:0:0:1"
  );
}

// isPrivateLiteral is exported for the F05/D01 send path: DNS names can
// resolve to private addresses after the allowlist check, so the
// companion re-checks the resolved literal before connecting.
export { isPrivateLiteral as isPrivateAddressLiteral };

function hostMatches(rule: string, host: string): boolean {
  if (rule.startsWith("*.")) {
    const suffix = rule.slice(2);
    // Wildcards match subdomains only: never the apex, never literals.
    return (
      host.length > suffix.length + 1 &&
      host.endsWith(suffix) &&
      host[host.length - suffix.length - 1] === "." &&
      !IPV4_PATTERN.test(host) &&
      !host.includes(":")
    );
  }
  return host === rule;
}

// evaluateDestination admits or denies one webhook URL. Reasons are
// finite safe codes; userinfo and fragments deny (matching the native
// request boundary), literals face the loopback/private scopes, and a
// rule without a port matches the scheme default port only.
export function evaluateDestination(policy: DestinationPolicy, url: string): DestinationDecision {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return Object.freeze({ decision: "denied", reason: "invalid-url" });
  }
  const shape = checkUrlShape(policy, parsed);
  if (shape !== undefined) return shape;
  const host = parsed.hostname.toLowerCase();
  const port = parsed.port === "" ? DEFAULT_PORTS[parsed.protocol]! : Number(parsed.port);
  for (let index = 0; index < policy.rules.length; index += 1) {
    const rule = policy.rules[index];
    if (rule.scheme !== "any" && `${rule.scheme}:` !== parsed.protocol) continue;
    if (!hostMatches(rule.host, host)) continue;
    if ((rule.port ?? DEFAULT_PORTS[parsed.protocol]!) !== port) continue;
    if (rule.pathPrefix !== undefined && !parsed.pathname.startsWith(rule.pathPrefix)) continue;
    const match: {
      decision: "allowed";
      ruleIndex: number;
      credential?: EnvName;
      reason: string;
    } = { decision: "allowed", ruleIndex: index, reason: "allowed" };
    if (rule.credential !== undefined) match.credential = rule.credential;
    return Object.freeze(match);
  }
  return Object.freeze({ decision: "denied", reason: "no-matching-rule" });
}

export type RedirectDecision = Readonly<{
  decision: "allowed" | "denied";
  target?: string;
  reason: string;
}>;

// evaluateRedirect rules one redirect hop: deny stops every hop,
// same-origin keeps the request on the posting origin, and allowlisted
// re-admits the target through the full destination policy. Relative
// Location values resolve against the posting URL; hop counting stays
// with the caller, which denies past maxRedirectHops.
export function evaluateRedirect(
  policy: DestinationPolicy,
  requestUrl: string,
  location: string,
  hopsSoFar: number,
): RedirectDecision {
  if (!Number.isSafeInteger(hopsSoFar) || hopsSoFar < 0)
    throw new TypeError("invalid redirect hop count");
  if (policy.redirect === "deny") return Object.freeze({ decision: "denied", reason: "deny" });
  if (hopsSoFar >= policy.maxRedirectHops)
    return Object.freeze({ decision: "denied", reason: "max-hops" });
  let posting: URL;
  try {
    posting = new URL(requestUrl);
  } catch {
    return Object.freeze({ decision: "denied", reason: "invalid-request-url" });
  }
  let target: URL;
  try {
    target = new URL(location, posting);
  } catch {
    return Object.freeze({ decision: "denied", reason: "invalid-location" });
  }
  if (policy.redirect === "same-origin") {
    if (target.origin !== posting.origin)
      return Object.freeze({ decision: "denied", reason: "cross-origin" });
    const shape = checkUrlShape(policy, target);
    if (shape !== undefined) return Object.freeze({ decision: "denied", reason: shape.reason });
    return Object.freeze({
      decision: "allowed",
      target: redactUrl(target.href),
      reason: "allowed",
    });
  }
  const check = evaluateDestination(policy, target.href);
  if (check.decision === "denied")
    return Object.freeze({ decision: "denied", reason: `target-${check.reason}` });
  return Object.freeze({ decision: "allowed", target: redactUrl(target.href), reason: "allowed" });
}

// redactUrl strips userinfo and fragments for logs and reports. It
// never sees credential values: those resolve at send time from the
// named environment variable.
export function redactUrl(url: string): string {
  try {
    const parsed = new URL(url);
    parsed.username = "";
    parsed.password = "";
    parsed.hash = "";
    return parsed.href;
  } catch {
    return "<invalid-url>";
  }
}

// credentialValue resolves the rule's bound environment variable at
// send time. The returned value goes straight into the send path and
// must never be logged, reported, or stored in an artifact.
export function credentialValue(
  name: EnvName,
  readEnvironment: (key: string) => string | undefined,
): string | undefined {
  const value = readEnvironment(name);
  if (value === undefined || value === "") return undefined;
  return value;
}

// policyReport is the safe diagnostic projection: version, redirect
// mode, hop bound, scopes, and per-rule hosts without credential
// values (environment names are configuration, not secrets).
export function policyReport(policy: DestinationPolicy): Readonly<{
  version: string;
  redirect: RedirectPolicy;
  maxRedirectHops: number;
  rules: readonly string[];
}> {
  return Object.freeze({
    version: policy.version,
    redirect: policy.redirect,
    maxRedirectHops: policy.maxRedirectHops,
    rules: Object.freeze(policy.rules.map((rule) => rule.host)),
  });
}

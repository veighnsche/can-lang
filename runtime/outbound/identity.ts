// C-G outbound identity vocabulary: tenant, budget pool, correlation (F01).
//
// One identity vocabulary for HTTP companion delivery and AI budget
// enforcement. Tenant and pool come from trusted application
// authorization; browser-supplied tenant claims alone never authorize an
// accounting identity, so this module offers no browser-claim parsing.
// Identity values carry no secrets and are safe to log. Credentials bind
// by environment name only (see destination-policy.ts) and never appear
// in identity records or reports.
export type TenantId = string & { readonly __brand: "TenantId" };
export type PoolId = string & { readonly __brand: "PoolId" };
export type CorrelationId = string & { readonly __brand: "CorrelationId" };
export type InvocationId = string & { readonly __brand: "InvocationId" };

// 1..128 chars, ASCII letters/digits plus . _ : -. The leading
// alnuminumeric keeps keys sortable and unambiguous in epoch-row keys and
// log lines; the charset is a subset of well-formed header/query-safe
// text so identities can travel in diagnostics without escaping.
const IDENTITY_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;

function checkIdentity(kind: string, value: unknown): string {
  if (typeof value !== "string" || !IDENTITY_PATTERN.test(value))
    throw new TypeError(`invalid ${kind}`);
  return value;
}

// identityValue validates free-form pinned names (metering provider,
// model, version) that travel in ledger records and reports. The
// charset is wider than tenant/pool/correlation: 1..256 well-formed
// characters without NUL, so real provider spellings fit while record
// keys stay unambiguous. Callers pass names and versions only — never
// endpoints, credential values, or prompt text.
export function identityValue(kind: string, value: unknown): string {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.length > 256 ||
    !value.isWellFormed() ||
    value.includes("\0")
  )
    throw new TypeError(`invalid ${kind}`);
  return value;
}

export function tenantId(value: unknown): TenantId {
  return checkIdentity("tenant", value) as TenantId;
}

export function poolId(value: unknown): PoolId {
  return checkIdentity("budget pool", value) as PoolId;
}

export function correlationId(value: unknown): CorrelationId {
  return checkIdentity("correlation", value) as CorrelationId;
}

export function invocationId(value: unknown): InvocationId {
  return checkIdentity("invocation", value) as InvocationId;
}

// newCorrelationId mints a fresh logical-request correlation. Logical
// correlation may group provider attempts, but each retry send still
// needs its own accounting record and reservation.
export function newCorrelationId(): CorrelationId {
  return crypto.randomUUID() as CorrelationId;
}

// newInvocationId mints a durable unique provider-attempt identity. The
// caller may supply its own invocation ID to a reserve call instead, so
// a retry after an ambiguous commit reconciles by ID rather than
// double-charging.
export function newInvocationId(): InvocationId {
  return crypto.randomUUID() as InvocationId;
}

// InvocationContext is the native accounting identity H guards and E
// transports: authenticated tenant, configured budget pool, and the
// logical correlation grouping the invocation. Frozen at creation.
export type InvocationContext = Readonly<{
  tenant: TenantId;
  pool: PoolId;
  correlation: CorrelationId;
}>;

export function invocationContext(
  tenant: unknown,
  pool: unknown,
  correlation: unknown = newCorrelationId(),
): InvocationContext {
  return Object.freeze({
    tenant: tenantId(tenant),
    pool: poolId(pool),
    correlation: correlationId(correlation),
  });
}

export function isInvocationContext(value: unknown): value is InvocationContext {
  if (value === null || typeof value !== "object") return false;
  const shaped = value as Partial<InvocationContext>;
  return (
    typeof shaped.tenant === "string" &&
    IDENTITY_PATTERN.test(shaped.tenant) &&
    typeof shaped.pool === "string" &&
    IDENTITY_PATTERN.test(shaped.pool) &&
    typeof shaped.correlation === "string" &&
    IDENTITY_PATTERN.test(shaped.correlation)
  );
}

// nestContext enforces the ai_budget::within nesting rule at the
// vocabulary layer: same-context nesting returns the active context
// unchanged, while an attempt to replace the active tenant/pool is a
// TypeError. H maps the throw to ai_budget::unavailable; E never sees a
// leaked tenant because scope completion keeps returning this same
// frozen record.
export function nestContext(active: InvocationContext, next: InvocationContext): InvocationContext {
  if (active.tenant !== next.tenant || active.pool !== next.pool)
    throw new TypeError("nested scope cannot replace tenant/pool");
  return active;
}

// contextReport is the safe diagnostic projection: tenant, pool and
// correlation only. Identity records hold no credentials, endpoints, or
// prompt text, so this report is safe to log and to attach to operator
// and accounting reports.
export function contextReport(context: InvocationContext): Readonly<{
  tenant: string;
  pool: string;
  correlation: string;
}> {
  return Object.freeze({
    tenant: context.tenant as string,
    pool: context.pool as string,
    correlation: context.correlation as string,
  });
}

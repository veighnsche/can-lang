// H08 live gate: mechanical enforcement of the H04 rule that no paid
// evaluation runs without its required inputs. Live dispatch needs
// ALL of: provider credentials, an explicit operator spend cap, a
// pinned per-token price table (a USD cap cannot be enforced against
// unknown USD cost), and a qualified metering bound U. Anything
// missing blocks with the exact missing input named — never a pass.
import { pricePerToken, qualifiedBound, type TriageRegistration } from "./protocol.ts";

export const CREDENTIAL_ENV = "TYPESAFE_API_KEY";
export const SPEND_CAP_ENV = "CAN_EVAL_SPEND_CAP_USD";

export type LiveAuthorization =
  | Readonly<{ authorized: true; spendCapUsd: number }>
  | Readonly<{ authorized: false; missing: readonly string[] }>;

export function authorizeLive(
  env: Readonly<Record<string, string | undefined>>,
  registration: TriageRegistration,
): LiveAuthorization {
  const missing: string[] = [];
  if (env[CREDENTIAL_ENV] === undefined || env[CREDENTIAL_ENV] === "")
    missing.push(`${CREDENTIAL_ENV} (provider credential)`);
  const capRaw = env[SPEND_CAP_ENV];
  let cap = Number.NaN;
  if (capRaw === undefined || capRaw === "") missing.push(`${SPEND_CAP_ENV} (explicit spend cap)`);
  else {
    cap = Number(capRaw);
    if (!Number.isFinite(cap) || cap <= 0) missing.push(`${SPEND_CAP_ENV} (must parse as USD > 0)`);
  }
  if (pricePerToken(registration) === undefined)
    missing.push("pinned per-token USD price table (USD cost unknown)");
  if (qualifiedBound(registration) === undefined)
    missing.push("qualified complete-call bound U (profile unqualified)");
  if (missing.length > 0) return { authorized: false, missing };
  return { authorized: true, spendCapUsd: cap };
}

// SpendTracker stops a serial run at the USD cap. Token charges
// convert through the pinned price table only; unknown pricing never
// runs (the gate above already refused it).
export type SpendTracker = Readonly<{
  spentUsd: () => number;
  remainingUsd: () => number;
  charge: (tokens: number) => void;
  exhausted: () => boolean;
}>;

export function createSpendTracker(capUsd: number, usdPerToken: number): SpendTracker {
  if (!Number.isFinite(capUsd) || capUsd <= 0) throw new TypeError("invalid spend cap");
  if (!Number.isFinite(usdPerToken) || usdPerToken <= 0) throw new TypeError("invalid token price");
  let spent = 0;
  return {
    spentUsd: () => spent,
    remainingUsd: () => Math.max(0, capUsd - spent),
    charge: (tokens: number): void => {
      if (!Number.isSafeInteger(tokens) || tokens < 0) throw new TypeError("invalid token charge");
      spent += tokens * usdPerToken;
    },
    exhausted: (): boolean => spent >= capUsd,
  };
}

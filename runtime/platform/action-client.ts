import { type Completion, type AssertionContext } from "../completion.ts";
import { dataProperty } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { createJsonActionFetch, type JsonFetchSite, type JsonFetchTypes } from "./action-json.ts";

// createActionClient lowers the symbol-based action::request and
// action::post consumers to canonical URL building plus native
// same-origin fetch. Emitted invocations pass the authored value operands
// (the captures record when the action declares captures, then the POST
// wire body), the spliced client site, and the trailing assertion
// context; the static action symbol never lowers to a value. The client
// projection reuses the JSON fetch site shape, so after unpacking the
// captures record into path order this adapter delegates to the shared
// fetch consumer: exact status/leaf/operation agreement, bodyless GET,
// and distinct transport, abort, codec and unexpected-status outcomes in
// the declared failure bound.
export function createActionClient(
  domain: ReturnType<typeof createDomainRuntime>,
  types: JsonFetchTypes,
) {
  const fetch = createJsonActionFetch(domain, types);
  async function run(
    method: "GET" | "POST",
    args: readonly unknown[],
  ): Promise<Completion<unknown>> {
    const verb = method === "GET" ? "request" : "post";
    const { found, body, site, context } = split(verb, args);
    const captures = clientCaptures(site);
    if ((found === undefined) !== (captures.length === 0))
      throw new TypeError(
        `action ${verb} for ${(site as JsonFetchSite).action} disagrees on its captures record`,
      );
    const values = captures.map((capture) =>
      found === undefined ? undefined : dataProperty(found, capture.name),
    );
    const action = (site as JsonFetchSite).action;
    return method === "GET"
      ? fetch.get(action, ...values, site, context)
      : fetch.post(action, ...values, body, site, context);
  }
  return Object.freeze({
    async request(...args: unknown[]): Promise<Completion<unknown>> {
      return run("GET", args);
    },
    async post(...args: unknown[]): Promise<Completion<unknown>> {
      return run("POST", args);
    },
  });
}

function split(
  verb: "request" | "post",
  args: readonly unknown[],
): {
  found: unknown;
  body: unknown;
  site: unknown;
  context: AssertionContext | undefined;
} {
  if (verb === "request") {
    if (args.length !== 2 && args.length !== 3)
      throw new TypeError("action request takes its captures record and site");
    return {
      found: args.length === 3 ? args[0] : undefined,
      body: undefined,
      site: args.length === 3 ? args[1] : args[0],
      context: (args.length === 3 ? args[2] : args[1]) as AssertionContext | undefined,
    };
  }
  if (args.length !== 3 && args.length !== 4)
    throw new TypeError("action post takes its captures record, wire body and site");
  return {
    found: args.length === 4 ? args[0] : undefined,
    body: args.length === 4 ? args[1] : args[0],
    site: args.length === 4 ? args[2] : args[1],
    context: (args.length === 4 ? args[3] : args[2]) as AssertionContext | undefined,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

// clientCaptures reads the ordered capture names the adapter unpacks
// from the captures record. This envelope check only gates unpacking;
// the fetch consumer revalidates the full site (operation identity,
// verb, route, codecs and finite cases) before sending any byte.
function clientCaptures(site: unknown): readonly { name: string; type: string }[] {
  if (!isRecord(site) || typeof site.action !== "string" || site.action === "")
    throw new TypeError("action client site carries no action identity");
  if (!Array.isArray(site.captures))
    throw new TypeError(`action client site ${site.action} carries malformed captures`);
  for (const row of site.captures) {
    if (
      !isRecord(row) ||
      typeof row.name !== "string" ||
      row.name === "" ||
      (row.type !== "str" && row.type !== "int")
    )
      throw new TypeError(`action client site ${site.action} carries a malformed capture`);
  }
  return site.captures as readonly { name: string; type: string }[];
}

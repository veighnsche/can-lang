// H08 scripted stub: deterministic SystemOne-shaped replies served in
// request order (the runner is serial, so order is deterministic).
// Scripts deliberately mix correct, incorrect, abstaining, invalid,
// and usage-less replies to exercise every verdict path; scripted
// verdicts are plumbing checks, never quality evidence.
import type { TriageCase } from "./protocol.ts";

export type StubScript = Readonly<{
  bodies: readonly unknown[];
  maxConcurrency: () => number;
  requests: () => number;
  url: (path: string) => string;
  close: () => Promise<void>;
}>;

export function choiceBody(
  keys: readonly string[],
  selected: string,
  confidence: number,
  usage: unknown,
): unknown {
  const rest = (1 - 0.9) / (keys.length - 1);
  const probabilities: Record<string, number> = {};
  for (const key of keys) probabilities[key] = key === selected ? 0.9 : rest;
  return {
    model: "stub",
    ...(usage === undefined ? {} : { usage }),
    answers: { q0: { type: "choice", choice: selected, confidence, probabilities } },
  };
}

// scriptedBodies builds one reply per case: correct verdicts for most
// cases, plus one incorrect, one abstention, one invalid body, one
// usage-less body, and one body with extra breakdown fields (which
// must not double-count). Order follows the case list.
export function scriptedBodies(
  keys: readonly string[],
  cases: readonly TriageCase[],
  usageBase: number,
): unknown[] {
  return cases.map((triage, index) => {
    const usage = { input_tokens: usageBase + index, output_tokens: 20 + index };
    const id = triage.id;
    if (id.endsWith("-07"))
      return choiceBody(
        keys,
        keys.find((key) => key !== triage.label)!,
        0.91,
        usage,
      );
    if (id.endsWith("-09")) return choiceBody(keys, triage.label, 0.3, usage);
    if (id.endsWith("-10"))
      return { model: "stub", usage, answers: { q0: { type: "choice", choice: triage.label } } };
    if (id.endsWith("-11")) return choiceBody(keys, triage.label, 0.88, undefined);
    if (id.endsWith("-12"))
      return choiceBody(keys, triage.label, 0.87, {
        ...usage,
        cached_tokens: 9999,
        reasoning_tokens: 9999,
      });
    return choiceBody(keys, triage.label, 0.92, usage);
  });
}

export function serveScripted(bodies: readonly unknown[]): StubScript {
  let inflight = 0;
  let maxConcurrency = 0;
  let requests = 0;
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      inflight += 1;
      maxConcurrency = Math.max(maxConcurrency, inflight);
      try {
        const index = requests;
        requests += 1;
        await request.text();
        if (index >= bodies.length)
          return Response.json({ error: "script exhausted" }, { status: 500 });
        return Response.json(bodies[index], { status: 200 });
      } finally {
        inflight -= 1;
      }
    },
  });
  return {
    bodies,
    maxConcurrency: () => maxConcurrency,
    requests: () => requests,
    url: (path: string): string => new URL(path, server.url).href,
    close: async (): Promise<void> => {
      await server.stop(true);
    },
  };
}

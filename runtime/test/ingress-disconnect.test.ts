// E02 (X-R04-3): pinned Bun.serve ingress-disconnect projection probes.
//
// Tested separately from SQL cancel: these legs qualify what the native
// serve stack reports when the peer disconnects mid-handler. Servers bind
// ephemeral loopback ports, so every run uses distinct ports.
import { describe, expect, test } from "bun:test";

function requestSignal(request: Request): AbortSignal | undefined {
  return (request as unknown as { signal?: AbortSignal }).signal;
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
}

describe("X-R04-3 ingress disconnect projection", () => {
  test("serve-side Request exposes an un-aborted signal", async () => {
    let observed: { hasSignal: boolean; aborted: boolean } | undefined;
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch(request) {
        const signal = requestSignal(request);
        observed = { hasSignal: signal instanceof AbortSignal, aborted: signal?.aborted ?? true };
        return new Response("ok");
      },
    });
    try {
      const response = await fetch(`http://127.0.0.1:${server.port}/`);
      expect(response.status).toBe(200);
      await response.text();
      expect(observed).toEqual({ hasSignal: true, aborted: false });
    } finally {
      void server.stop(true);
    }
  });

  test("client abort trips the buffered handler signal while work runs on", async () => {
    let abortAt = -1;
    let handlerEndAt = -1;
    let startedAt = 0;
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      async fetch(request) {
        startedAt = Date.now();
        const signal = requestSignal(request);
        signal?.addEventListener("abort", () => {
          abortAt = Date.now() - startedAt;
        });
        await Bun.sleep(3000);
        handlerEndAt = Date.now() - startedAt;
        return new Response("done");
      },
    });
    try {
      const stop = new AbortController();
      const outcome = fetch(`http://127.0.0.1:${server.port}/slow`, { signal: stop.signal }).then(
        () => "responded",
        (cause: unknown) => (cause instanceof Error ? cause.name : typeof cause),
      );
      await Bun.sleep(300);
      stop.abort();
      expect(await outcome).toBe("AbortError");
      // The native signal projects the disconnect promptly; the handler
      // itself is not terminated and runs to completion.
      await waitFor("request signal abort", () => abortAt >= 0, 2500);
      expect(abortAt).toBeLessThan(2500);
      await waitFor("handler completion", () => handlerEndAt >= 0, 5000);
      expect(handlerEndAt).toBeGreaterThanOrEqual(3000);
    } finally {
      void server.stop(true);
    }
  });

  test("client abort cancels the response stream and trips the signal", async () => {
    let streamCancelled = false;
    let signalAborted = false;
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch(request) {
        requestSignal(request)?.addEventListener("abort", () => {
          signalAborted = true;
        });
        let timer: ReturnType<typeof setInterval> | undefined;
        const stream = new ReadableStream({
          start(controller) {
            let tick = 0;
            timer = setInterval(() => {
              controller.enqueue(`chunk ${tick++}\n`);
            }, 100);
          },
          cancel() {
            streamCancelled = true;
            if (timer !== undefined) clearInterval(timer);
          },
        });
        return new Response(stream, { headers: { "content-type": "text/plain" } });
      },
    });
    try {
      const stop = new AbortController();
      const response = await fetch(`http://127.0.0.1:${server.port}/stream`, {
        signal: stop.signal,
      });
      const reader = response.body!.getReader();
      const first = await reader.read();
      expect(first.done).toBe(false);
      await reader.cancel();
      stop.abort();
      await waitFor("stream cancel", () => streamCancelled, 2500);
      expect(streamCancelled).toBe(true);
      expect(signalAborted).toBe(true);
    } finally {
      void server.stop(true);
    }
  });
});

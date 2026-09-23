import { Deadline, transportFault } from "./deadline.ts";

export type ObserveNative = <T>(operation: () => Promise<T>) => Promise<T>;

export async function readBody(
  response: Response,
  limit: number,
  deadline: Deadline,
  onCleanup: (cause: unknown) => void,
  observe: ObserveNative = (operation) => operation(),
  cancel: (operation: () => Promise<unknown>) => void = (operation) => {
    void operation().catch(onCleanup);
  },
): Promise<Uint8Array> {
  if (!Number.isSafeInteger(limit) || limit < 1 || limit > 67108864)
    throw new TypeError("invalid transport byte limit");
  if (!response.body) {
    deadline.check();
    return new Uint8Array();
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0,
    complete = false;
  try {
    for (;;) {
      let next: Awaited<ReturnType<typeof reader.read>>;
      next = await deadline.wait(
        observe(async () => {
          try {
            return await reader.read();
          } catch {
            deadline.check();
            throw transportFault({ kind: "transport", phase: "body" });
          }
        }),
      );
      if (next.done) {
        complete = true;
        break;
      }
      if (next.value.byteLength > limit - size) throw transportFault({ kind: "limit", limit });
      size += next.value.byteLength;
      chunks.push(next.value);
    }
    const result = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      result.set(chunk, offset);
      offset += chunk.byteLength;
    }
    deadline.check();
    return result;
  } finally {
    if (!complete) {
      cancel(() => reader.cancel());
    }
    reader.releaseLock();
  }
}

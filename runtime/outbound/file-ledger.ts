// File-durable R14 ledger backend (F01).
//
// Persists ledger state as JSON with atomic rename and fsync, guarded
// by a lockfile so concurrent writers — including writers in other
// processes — serialize. State survives process restart; corruption or
// an unacquirable lock fails closed (load/commit throw, the service
// reports storage-unavailable) and the file is never auto-reset.
//
// The lock is held only across the read-compare-write critical section:
// short native operations, never across provider I/O.
import { mkdir, open, rename, rm, stat } from "node:fs/promises";
import { dirname } from "node:path";
import type { LedgerCommit, LedgerState, LedgerStore } from "./ledger.ts";
import { emptyLedgerState } from "./ledger.ts";

const LOCK_ATTEMPTS = 200;
const LOCK_WAIT_MS = 5;
const LOCK_STALE_MS = 10_000;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

// parseLedgerState revalidates persisted shape just enough to fail
// closed on corruption: version counter plus the four record maps. Row
// and record internals were frozen valid at write time; anything else
// is operator recovery (restore from backup), never silent repair.
function parseLedgerState(text: string): LedgerState {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error("ledger file corrupt: invalid JSON");
  }
  if (!isRecord(parsed)) throw new Error("ledger file corrupt: not an object");
  if (!Number.isSafeInteger(parsed["version"]) || (parsed["version"] as number) < 0)
    throw new Error("ledger file corrupt: bad version");
  for (const key of ["schedules", "epochs", "invocations", "quarantined"]) {
    if (!isRecord(parsed[key])) throw new Error(`ledger file corrupt: bad ${key}`);
  }
  return parsed as unknown as LedgerState;
}

async function fsyncDirectory(path: string): Promise<void> {
  const handle = await open(path, "r");
  try {
    await handle.sync();
  } finally {
    await handle.close();
  }
}

async function acquireLock(lockPath: string): Promise<void> {
  for (let attempt = 0; attempt < LOCK_ATTEMPTS; attempt += 1) {
    let handle: Awaited<ReturnType<typeof open>> | undefined;
    try {
      handle = await open(lockPath, "wx", 0o644);
    } catch (cause) {
      if ((cause as { code?: unknown }).code !== "EEXIST") throw cause;
      handle = undefined;
    }
    if (handle !== undefined) {
      try {
        await handle.writeFile(String(Date.now()));
        await handle.sync();
      } finally {
        await handle.close();
      }
      return;
    }
    // A stale lock from a crashed holder must not wedge the ledger:
    // steal it once it is older than the bound.
    try {
      const info = await stat(lockPath);
      if (Date.now() - info.mtimeMs > LOCK_STALE_MS) {
        await rm(lockPath, { force: true });
        continue;
      }
    } catch {
      // Lost a race with the releasing holder; keep spinning.
    }
    await sleep(LOCK_WAIT_MS);
  }
  throw new Error("ledger lock unavailable");
}

async function releaseLock(lockPath: string): Promise<void> {
  await rm(lockPath, { force: true });
}

async function readCurrent(path: string): Promise<LedgerState> {
  let handle: Awaited<ReturnType<typeof open>> | undefined;
  try {
    handle = await open(path, "r");
  } catch (cause) {
    if ((cause as { code?: unknown }).code === "ENOENT") return emptyLedgerState();
    throw cause;
  }
  try {
    return parseLedgerState(await handle.readFile("utf8"));
  } finally {
    await handle.close();
  }
}

// openFileLedgerStore binds a LedgerStore to one file. The parent
// directory is created on first commit; concurrent openers of the same
// path share the durable state through the lockfile.
export function openFileLedgerStore(path: string): LedgerStore {
  if (typeof path !== "string" || path === "") throw new TypeError("invalid ledger path");
  const lockPath = `${path}.lock`;
  return {
    name: `file:${path}`,
    load: async (): Promise<LedgerState> => readCurrent(path),
    commit: async (expectedVersion: number, next: LedgerState): Promise<LedgerCommit> => {
      if (next.version !== expectedVersion + 1) throw new TypeError("ledger version must advance");
      await mkdir(dirname(path), { recursive: true });
      await acquireLock(lockPath);
      try {
        const current = await readCurrent(path);
        if (current.version !== expectedVersion) return { committed: false };
        const temporary = `${path}.tmp-${process.pid}`;
        const handle = await open(temporary, "w", 0o644);
        try {
          await handle.writeFile(JSON.stringify(next));
          await handle.sync();
        } finally {
          await handle.close();
        }
        await rename(temporary, path);
        await fsyncDirectory(dirname(path));
        return { committed: true };
      } finally {
        await releaseLock(lockPath);
      }
    },
  };
}
// File stores hold no handles or leases between calls: restart
// recovery is reopening the path with openFileLedgerStore.

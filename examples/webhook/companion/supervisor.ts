// F05 companion: bounded in-process supervisor.
//
// runSupervised runs the worker task to completion, restarting it on
// unexpected throws with exponential backoff. Restarts are bounded:
// past maxRestarts the last failure propagates and the process exits
// nonzero, so a poisoned configuration or a broken Can side dies
// loudly instead of hot-looping. Clean task completion (including an
// abort-drained shutdown) is not a restart.
//
// This covers throws inside the process. A SIGKILLed companion relies
// on the external supervisor (service unit, container restart policy):
// unacked rows stay leased, expire, and redeliver, which the pair
// test proves. See PROTOCOL.md for the operator recipe.
export type SupervisorOptions = Readonly<{
  maxRestarts: number;
  backoffBaseMs: number;
  backoffMaxMs: number;
  sleep: (ms: number) => Promise<void>;
  onRestart: (restart: number, cause: unknown) => void;
}>;

export function defaultSupervisorOptions(init: Partial<SupervisorOptions> = {}): SupervisorOptions {
  return Object.freeze({
    maxRestarts: 5,
    backoffBaseMs: 1000,
    backoffMaxMs: 30000,
    sleep: (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms)),
    onRestart: () => {},
    ...init,
  });
}

export async function runSupervised(
  task: () => Promise<void>,
  options: SupervisorOptions = defaultSupervisorOptions(),
): Promise<void> {
  if (!Number.isSafeInteger(options.maxRestarts) || options.maxRestarts < 0)
    throw new TypeError("invalid supervisor restarts");
  for (let restarts = 0; ; restarts += 1) {
    try {
      await task();
      return;
    } catch (cause) {
      if (restarts >= options.maxRestarts) throw cause;
      options.onRestart(restarts + 1, cause);
      const wait = Math.min(
        options.backoffMaxMs,
        options.backoffBaseMs * 2 ** Math.min(restarts, 16),
      );
      await options.sleep(wait);
    }
  }
}

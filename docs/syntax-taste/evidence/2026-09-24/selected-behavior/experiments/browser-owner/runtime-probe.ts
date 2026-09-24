import { runOwnedRoot, registerResource, useResource, resourceStatus, withScope, guardCallback } from "../../../../../../../runtime/owner.ts";
import { success, invoke } from "../../../../../../../runtime/completion.ts";
import { standardFailureKind } from "../../../../../../../runtime/failure.ts";

const deferred = () => {
  let resolve!: () => void;
  const promise = new Promise<void>(r => { resolve = r; });
  return { promise, resolve };
};
const describe = (completion: any) => completion.kind === "standard" ? standardFailureKind(completion.value) : completion.kind;
const origin = { source: "probe:browser-owner", start: 0, end: 0, invocation: [] };

export async function probeRuntime() {
  const trace: string[] = [];
  const gates = [deferred(), deferred()];
  const used = [deferred(), deferred()];
  const tokens: any[] = [];
  const roots = ["A", "B"].map((name, i) => runOwnedRoot(async () => {
    const token = registerResource("probe", name, () => { trace.push(`${name}:close`); return success(undefined); }, { scopeManaged: true });
    tokens.push(token);
    trace.push(`${name}:entered`);
    await gates[i].promise;
    const result = await invoke(() => useResource(token, "probe", native => {
      trace.push(`${name}:native:${native}`);
      return success(native);
    }), origin);
    trace.push(`${name}:use:${describe(result)}`);
    used[i].resolve();
    await Promise.resolve();
    return result;
  }));
  gates[1].resolve();
  await used[1].promise;
  gates[0].resolve();
  const results = await Promise.all(roots);

  // Adapted from runtime/test/owner.test.ts: externally dispatched guarded
  // callback registers a managed resource before await and uses it after await.
  const bodyExit = deferred(), finish = deferred(), ready = deferred();
  let callback: any, token: any, done = false;
  const root = runOwnedRoot(() => withScope(async scope => {
    callback = guardCallback(scope, async () => {
      token = registerResource("callback", {}, () => success(undefined), { scopeManaged: true });
      await finish.promise;
      return useResource(token, "callback", () => success(undefined));
    });
    ready.resolve();
    await bodyExit.promise;
    return success(undefined);
  })).then(result => { done = true; return result; });
  await ready.promise;
  const pending = callback();
  bodyExit.resolve();
  await Promise.resolve();
  await Promise.resolve();
  const beforeRelease = { rootDone: done, leases: resourceStatus(token).leases };
  finish.resolve();
  const callbackResult = await pending;
  const rootResult = await root;
  const afterClose = await invoke(callback, origin);
  return {
    interleavedRoots: {
      trace,
      outcomes: results.map(r => ({ completion: describe(r.completion), cleanupFailed: r.cleanupFailed })),
      resources: tokens.map(t => resourceStatus(t).state),
    },
    guardedCallback: { beforeRelease, completion: describe(callbackResult), rootCompletion: describe(rootResult.completion), cleanupFailed: rootResult.cleanupFailed, resource: resourceStatus(token).state, afterClose: describe(afterClose) },
  };
}

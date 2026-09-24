export async function probeContext(Storage) {
  const context = new Storage(), trace = [];
  const gates = Array.from({ length: 2 }, () => {
    let resolve;
    const promise = new Promise(r => { resolve = r; });
    return { promise, resolve };
  });
  const tasks = ["A", "B"].map((name, i) => context.run(name, async () => {
    trace.push([name, "before", context.getStore() ?? null]);
    await gates[i].promise;
    trace.push([name, "after-gate", context.getStore() ?? null]);
    await Promise.resolve();
    trace.push([name, "after-microtask", context.getStore() ?? null]);
  }));
  gates[1].resolve();
  await tasks[1];
  gates[0].resolve();
  await tasks[0];
  return { trace, isolated: trace.every(([name, , observed]) => name === observed), outside: context.getStore() ?? null };
}

// Small explicit-token control only: this does not implement Can's owner
// runtime, resource contracts, or generated call-site propagation.
export async function probeExplicitTokens() {
  const trace = [];
  const tasks = ["A", "B"].map(async name => {
    const token = Object.freeze({ name });
    const use = async owner => { await Promise.resolve(); return owner.name; };
    trace.push([name, await use(token)]);
    trace.push([name, await use(token)]);
  });
  await Promise.all(tasks);
  return { trace, isolated: trace.every(([name, observed]) => name === observed) };
}

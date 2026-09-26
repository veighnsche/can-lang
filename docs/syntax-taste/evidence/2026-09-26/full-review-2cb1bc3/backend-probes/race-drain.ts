import { runOwnedRoot } from '/Users/vince/Projects/can-lang/runtime/owner.ts';
import { settle } from '/Users/vince/Projects/can-lang/runtime/coordination.ts';
import { success } from '/Users/vince/Projects/can-lang/runtime/completion.ts';
let releaseLoser!: () => void;
const loser = new Promise<void>(resolve => { releaseLoser = resolve; });
let reportSelected!: () => void;
const selected = new Promise<void>(resolve => { reportSelected = resolve; });
let rootDone = false;
const events: string[] = [];
const root = runOwnedRoot(async () => {
  const result = await settle('race', [
    { captures: [], run: async () => success('winner') },
    { captures: [], run: async () => { await loser; events.push('loser completed'); return success('loser'); } },
  ]);
  events.push('selected ' + result.kind);
  reportSelected();
  return success(undefined);
}).then(result => { rootDone = true; return result; });
await selected;
await Promise.resolve();
console.log(JSON.stringify({ phase: 'winner selected', rootDone, events }));
releaseLoser();
const result = await root;
console.log(JSON.stringify({ phase: 'loser released', rootDone, kind: result.completion.kind, cleanupFailed: result.cleanupFailed, events }));

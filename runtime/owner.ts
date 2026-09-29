// Private lifecycle enforcement over native async execution. Ambient Bun
// bindings over the canonical explicit core; the browser profile overlays
// runtime/browser/owner.ts with the explicit-only surface instead.
import { AsyncLocalStorage } from "node:async_hooks";
import { createAmbientOwner, type Execution } from "./owner-core.ts";

export type {
  ExplicitParticipant,
  OwnedGroup,
  OwnerContext,
  OwnerDiagnostic,
  Participant,
  Resource,
  Scope,
} from "./owner-core.ts";
export {
  closeResourceWithContext,
  guardCallbackWithContext,
  launchNativeWithContext,
  launchOwnedWithContext,
  registerCallableCaptures,
  registerResourceWithContext,
  resourceStatus,
  runExplicitRoot,
  useResourceWithContext,
  withScopeWithContext,
} from "./owner-core.ts";

const context = new AsyncLocalStorage<Execution>();
const ambient = createAmbientOwner({
  getStore: () => context.getStore(),
  run: (entry, fn) => context.run(entry, fn),
});
export const {
  registerResource,
  useResource,
  closeResource,
  launchOwned,
  launchNative,
  withScope,
  guardCallback,
  bindNativeCallback,
  runOwnedRoot,
} = ambient;
// Compiler-private admission query for closed fast branches: reports
// whether an ambient owner execution is present. Bun-only and
// side-effect-free; reads only the AsyncLocalStorage store identity,
// never resource state. Any present store declines, including a closed
// scope, so admitted branches never run under ambient ownership.
export function ambientOwnerPresent(): boolean {
  return context.getStore() !== undefined;
}

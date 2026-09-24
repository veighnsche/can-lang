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
  runOwnedRoot,
} = ambient;

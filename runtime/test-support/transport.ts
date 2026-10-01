// NT-P28 base test transport adapter. Framed JSON envelope FIFOs bound to
// one live owner: send encodes a codec::json_object through the JSON value
// codec and queues the frame; recv decodes the oldest frame. Queued frames
// are decoded bytes only, never live values. Without the N link (a later
// slice) channels loop back locally; kinds still admit a closed vocabulary
// so N-side dispatch can rely on the tag. Detach severs every channel at
// once (worker teardown / N-link loss); depth stays observable so
// incomplete executions keep their facts. Release/close are idempotent;
// ids are never reused so stale stays stale.
import { failure, success, type AssertionContext, type Completion } from "../completion.ts";
import { record, recordIdentity } from "../data.ts";
import { byteLength, type Bytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { createTestEvidence, type TestEvidenceErrors } from "./evidence.ts";
import {
  createTestOwner,
  ownerTag,
  type OwnerHandle,
  type TestOwner,
  type TestOwnerErrors,
} from "./owner.ts";
import { createTestTools, type TestToolsErrors, type ToolTable } from "./tools.ts";
import { createTestWorkspace, type TestWorkspaceErrors } from "./workspace.ts";

const channelBrand = Symbol("can.test.channel");
export type ChannelHandle = Readonly<{ readonly [channelBrand]: number }>;

export const CHANNEL_KINDS = ["dispatch", "report"] as const;
export type ChannelKind = (typeof CHANNEL_KINDS)[number];

const MAX_CHANNELS = 1024;
const MAX_DEPTH = 64;
const MAX_FRAME_BYTES = 1048576;

const origin = Object.freeze({
  source: "can:test-transport",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type TestTransportErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  transportFailure: string;
  channelFull: string;
  detachedTransport: string;
  channelEmpty: string;
  invalidKind: string;
}>;

export type EnvelopeCodec = Readonly<{
  encode(value: unknown, context?: AssertionContext): Promise<Completion<Bytes>>;
  decode(value: unknown, context?: AssertionContext): Promise<Completion<unknown>>;
}>;

type ChannelCell = {
  readonly owner: OwnerHandle;
  readonly kind: ChannelKind;
  readonly queue: Bytes[];
  closed: boolean;
};

function isChannelHandle(value: unknown): value is ChannelHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[channelBrand] === "number"
  );
}

function isChannelKind(value: unknown): value is ChannelKind {
  return typeof value === "string" && (CHANNEL_KINDS as readonly string[]).includes(value);
}

function channelName(id: number): string {
  return `channel#${id}`;
}

export function createTestTransport(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestTransportErrors,
  codec: EnvelopeCodec,
  owner: TestOwner,
  jsonObjectIdentity: string,
) {
  const channels = new Map<number, ChannelCell>();
  let nextId = 0;
  let detached = false;

  function fail(identity: string, field: string, value: string): Completion<never> {
    return failure(domain.create(identity, record(identity, [[field, value]]), origin));
  }

  function use(id: number): ChannelCell | Completion<never> {
    const cell = channels.get(id);
    // Ids this table never issued (another table's handles) are stale here,
    // exactly like released ones: they admit no operations. Only the
    // infallible release/close surface loud caller bugs as TypeErrors.
    if (cell === undefined) return fail(errors.staleHandle, "handle", channelName(id));
    if (!owner.ownerLive(cell.owner)) return fail(errors.staleHandle, "handle", channelName(id));
    if (detached) return fail(errors.detachedTransport, "handle", channelName(id));
    return cell;
  }

  return Object.freeze({
    async openChannel(
      ownerHandle: unknown,
      kind: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<ChannelHandle>> {
      const tag = ownerTag(ownerHandle);
      if (tag === null) throw new TypeError("test::channel_open needs an owner handle");
      if (!owner.ownerLive(ownerHandle)) return fail(errors.staleHandle, "handle", tag);
      if (!isChannelKind(kind))
        return fail(errors.invalidKind, "kind", typeof kind === "string" ? kind : "?");
      if (channels.size >= MAX_CHANNELS) throw new TypeError("test channel table cap reached");
      nextId += 1;
      channels.set(nextId, {
        owner: ownerHandle as OwnerHandle,
        kind,
        queue: [],
        closed: false,
      });
      const handle: ChannelHandle = Object.freeze({ [channelBrand]: nextId });
      return success(handle);
    },

    async sendEnvelope(
      handle: unknown,
      envelope: unknown,
      context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      if (!isChannelHandle(handle))
        throw new TypeError("test::channel_send needs a channel handle");
      const id = handle[channelBrand];
      const cell = use(id);
      if (!("queue" in cell)) return cell;
      if (cell.closed) return fail(errors.closedHandle, "handle", channelName(id));
      if (cell.queue.length >= MAX_DEPTH)
        return fail(errors.channelFull, "handle", channelName(id));
      const encoded = await codec.encode(envelope, context);
      if (encoded.kind !== "ok") return fail(errors.transportFailure, "reason", "encode");
      const frame = encoded.value;
      if (byteLength(frame) > BigInt(MAX_FRAME_BYTES))
        return fail(errors.transportFailure, "reason", "frame-too-large");
      cell.queue.push(frame);
      return success(byteLength(frame));
    },

    async recvEnvelope(handle: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (!isChannelHandle(handle))
        throw new TypeError("test::channel_recv needs a channel handle");
      const id = handle[channelBrand];
      const cell = use(id);
      if (!("queue" in cell)) return cell;
      const frame = cell.queue.shift();
      if (frame === undefined) {
        if (cell.closed) return fail(errors.closedHandle, "handle", channelName(id));
        return fail(errors.channelEmpty, "handle", channelName(id));
      }
      const decoded = await codec.decode(frame, context);
      if (decoded.kind !== "ok") return fail(errors.transportFailure, "reason", "decode");
      if (recordIdentity(decoded.value) !== jsonObjectIdentity)
        return fail(errors.transportFailure, "reason", "non-object-frame");
      return success(decoded.value);
    },

    async closeChannel(handle: unknown, _context?: AssertionContext): Promise<Completion<void>> {
      if (!isChannelHandle(handle))
        throw new TypeError("test::channel_close needs a channel handle");
      const id = handle[channelBrand];
      const cell = channels.get(id);
      if (cell === undefined) throw new TypeError(`foreign channel handle ${channelName(id)}`);
      cell.closed = true;
      return success(undefined);
    },

    async pendingDepth(handle: unknown, _context?: AssertionContext): Promise<Completion<bigint>> {
      if (!isChannelHandle(handle))
        throw new TypeError("test::channel_pending needs a channel handle");
      const id = handle[channelBrand];
      const cell = channels.get(id);
      if (cell === undefined) return fail(errors.staleHandle, "handle", channelName(id));
      if (!owner.ownerLive(cell.owner)) return fail(errors.staleHandle, "handle", channelName(id));
      return success(BigInt(cell.queue.length));
    },

    // Sever the N link: every channel detaches at once. Idempotent.
    // The emitter calls this on worker teardown / N-link loss; depth and
    // close stay available so incomplete executions keep their facts.
    detach(): void {
      detached = true;
    },
  });
}

export type TestTransport = ReturnType<typeof createTestTransport>;

export type TestSupportErrors = TestOwnerErrors &
  TestTransportErrors &
  TestWorkspaceErrors &
  TestToolsErrors &
  TestEvidenceErrors;

// createTestSupport composes the owner, transport, workspace, tools and
// evidence adapters into the one $canTest contribution the emitter
// wires: a single owner table shared by every handle, with the JSON
// value codec for envelope framing and the owner-registered tool table
// for keyed dispatch.
export function createTestSupport(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestSupportErrors,
  codec: EnvelopeCodec,
  jsonObjectIdentity: string,
  tools: ToolTable,
) {
  const owner = createTestOwner(domain, errors);
  const transport = createTestTransport(domain, errors, codec, owner, jsonObjectIdentity);
  const workspace = createTestWorkspace(domain, errors, owner);
  const toolRunner = createTestTools(domain, errors, owner, tools);
  const evidence = createTestEvidence(domain, errors, owner);
  return Object.freeze({ owner, transport, workspace, tools: toolRunner, evidence });
}

export type TestSupport = ReturnType<typeof createTestSupport>;

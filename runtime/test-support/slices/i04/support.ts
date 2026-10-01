// NT-I04 http_peer shared adapter support. Composes the peer and http
// adapters into the one $canHttpPeer contribution the emitter wires, over
// the $canTest owner table: I04 mints no owners, it borrows liveness from
// test::grant_admit. Service failures map to the slice fault with the K20
// kind/reason verbatim; closed handles map to test::closed_handle; unknown
// or foreign ids map to test::stale_handle, exactly like P28 channels.
import { failure, type Completion } from "../../../completion.ts";
import { array, dataProperty, record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import type { TestOwner } from "../../owner.ts";
import { HttpPeerError } from "./peer-service.ts";
import { createHttpPeerHttp, type HttpPeerHttpErrors } from "./http.ts";
import { createHttpPeerPeer, type HttpPeerPeerErrors } from "./peer.ts";

const origin = Object.freeze({
  source: "can:http-peer",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type HttpPeerErrors = HttpPeerPeerErrors & HttpPeerHttpErrors;

// failService maps one K20 service failure onto the Can surface. The fault
// identity selects the peer or http fault; closed-handle and wrong-owner
// reuse the shared test vocabulary.
export function failService(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: Pick<HttpPeerErrors, "staleHandle" | "closedHandle">,
  faultIdentity: string,
  handleTag: string,
  error: unknown,
): Completion<never> {
  if (error instanceof HttpPeerError) {
    if (error.kind === "closed-handle") {
      return failure(
        domain.create(
          errors.closedHandle,
          record(errors.closedHandle, [["handle", handleTag]]),
          origin,
        ),
      );
    }
    if (error.kind === "wrong-owner") {
      return failure(
        domain.create(
          errors.staleHandle,
          record(errors.staleHandle, [["handle", handleTag]]),
          origin,
        ),
      );
    }
    return failure(
      domain.create(
        faultIdentity,
        record(faultIdentity, [
          ["kind", error.kind],
          ["reason", error.message],
        ]),
        origin,
      ),
    );
  }
  throw error;
}

export function failStale(
  domain: ReturnType<typeof createDomainRuntime>,
  staleHandle: string,
  handleTag: string,
): Completion<never> {
  return failure(domain.create(staleHandle, record(staleHandle, [["handle", handleTag]]), origin));
}

// some wraps a present option value; noneRecord is the absent option. The
// leaf identities arrive from the emitter, like env.ts some/none.
export function some(someIdentity: string, value: unknown): unknown {
  return record(someIdentity, [["value", value]]);
}

export function noneRecord(noneIdentity: string): unknown {
  return record(noneIdentity, []);
}

export function optional(someIdentity: string, noneIdentity: string, value: unknown): unknown {
  return value === null || value === undefined
    ? noneRecord(noneIdentity)
    : some(someIdentity, value);
}

// headerRecord builds one http::header value from a K20 header pair.
export function headerRecord(headerIdentity: string, name: string, value: string): unknown {
  return record(headerIdentity, [
    ["name", name],
    ["value", value],
  ]);
}

// readHeaderPair reads one http::header value back into a K20 pair. The
// checker guarantees the shape; anything else is a loud caller bug.
export function readHeaderPair(value: unknown): readonly [string, string] {
  const name = dataProperty(value, "name");
  const body = dataProperty(value, "value");
  if (typeof name !== "string" || typeof body !== "string") {
    throw new TypeError("http_peer::deliver_response needs http::header records");
  }
  return [name, body] as const;
}

// readHeaderArray reads the deliver_response header block into K20 pairs.
export function readHeaderArray(value: unknown): [string, string][] {
  if (!Array.isArray(value)) {
    throw new TypeError("http_peer::deliver_response needs an http::header array");
  }
  return value.map((entry) => {
    const [name, body] = readHeaderPair(entry);
    return [name, body] as [string, string];
  });
}

// intArray freezes service bytes into a Can int array (bigint elements).
export function intArray(values: readonly number[]): readonly unknown[] {
  return array(values.map((value) => BigInt(value)));
}

// asInt lifts a service counter into a Can int (bigint).
export function asInt(value: number): bigint {
  return BigInt(value);
}

// readInt reads one Can int param back into a service number. The checker
// guarantees bigint; anything else is a loud caller bug.
export function readInt(value: unknown, what: string): number {
  if (typeof value !== "bigint") {
    throw new TypeError(`${what} needs an int`);
  }
  return Number(value);
}

// readByteArray reads one Can int[] param into service bytes. Elements must
// be bigint bytes; range/emptiness faults stay the service's own verdicts.
export function readByteArray(value: unknown, what: string): number[] {
  if (!Array.isArray(value)) {
    throw new TypeError(`${what} needs an int array`);
  }
  return value.map((entry) => {
    if (typeof entry !== "bigint") {
      throw new TypeError(`${what} needs int elements`);
    }
    return Number(entry);
  });
}

// createHttpPeerSupport composes the peer and http adapters over one
// shared owner table into the $canHttpPeer contribution value.
export function createHttpPeerSupport(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: HttpPeerErrors,
  owner: TestOwner,
) {
  const peer = createHttpPeerPeer(domain, errors, owner);
  const http = createHttpPeerHttp(domain, errors, owner);
  return Object.freeze({ peer, http });
}

export type HttpPeerSupport = ReturnType<typeof createHttpPeerSupport>;

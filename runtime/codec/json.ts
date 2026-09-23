import { dataArray, dataKeys, dataProperty, record, recordIdentity } from "../data.ts";
import { ownBytes, type Bytes } from "../bytes.ts";
import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import { Budget, CodecIssue, reject, childPath, standaloneBytes } from "./budget.ts";
import { encodeInteger } from "./numbers.ts";
import { parseDocument } from "./document.ts";
import { decodeJson5, decodeToml, decodeYaml } from "./formats.ts";
import { createJsonlFramer, decodeJsonlRecords, projectJsonlRecord } from "./jsonl.ts";
import { exactInt, graph, projectValue, type Schema, type SchemaNode } from "./project.ts";
import { useReader, type ReaderCell } from "../transport/stream/lifecycle.ts";
export type { Schema, SchemaNode };
const origin = Object.freeze({
  source: "can:codec",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
const encoder = new TextEncoder();
const nativeJSON = JSON as JSON & { rawJSON(text: string): unknown };
function scalar(text: string, path: string): void {
  if (
    new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(encoder.encode(text)) !== text
  )
    reject(path, "unicode_scalar");
}
function checkedData<T>(read: () => T, path: string): T {
  try {
    return read();
  } catch (cause) {
    if (cause instanceof TypeError) reject(path, "type");
    throw cause;
  }
}
function object(value: unknown, path: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || checkedData(() => Array.isArray(value), path))
    reject(path, "type");
  // Reject proxies/accessors through the same private data boundary used by the
  // runtime, before retrieving any authored property.
  checkedData(() => dataKeys(value), path);
  return value as Record<string, unknown>;
}
function required(value: Record<string, unknown>, key: string, path: string): unknown {
  if (!Object.hasOwn(value, key)) reject(childPath(path, key), "missing_member");
  return checkedData(() => dataProperty(value, key), childPath(path, key));
}
function extras(value: Record<string, unknown>, fields: readonly string[], path: string): void {
  const extra = Object.keys(value)
    .filter((key) => !fields.includes(key))
    .sort();
  if (extra.length) reject(childPath(path, extra[0]), "extra_member");
}

export function decodeJSON(schema: Schema, input: unknown, bytes = standaloneBytes): unknown {
  const budget = new Budget(bytes);
  const { parsed, rootHolder, tokens } = parseDocument(input, bytes);
  return projectValue(schema, parsed, rootHolder, budget, exactInt(tokens));
}

export function encodeJSON(schema: Schema, input: unknown, bytes = standaloneBytes): Bytes {
  const get = graph(schema),
    budget = new Budget(bytes),
    active = new Set<object>();
  const integers = new Map<bigint, string>();
  function textCost(value: string, path: string): void {
    if (value.length > budget.remaining) reject(path, "byte_limit");
    scalar(value, path);
    budget.charge(encoder.encode(JSON.stringify(value)).length, path);
  }
  function keys(names: readonly string[], path: string): void {
    budget.charge(2 + Math.max(0, names.length - 1) + names.length, path);
    for (const name of names) textCost(name, path);
  }
  function leaf(node: SchemaNode, value: unknown, path: string): SchemaNode {
    const identity = recordIdentity(value),
      found = node.leaves!.map(get).find((leaf) => leaf.identity === identity);
    if (!found) reject(path, "variant_tag");
    return found;
  }
  function visit(node: SchemaNode, value: unknown, path: string, depth: number): void {
    const container = node.kind !== "primitive";
    budget.visit(depth + (container ? 1 : 0), path);
    if (node.kind === "primitive") {
      switch (node.name) {
        case "str":
          if (typeof value !== "string") reject(path, "type");
          textCost(value, path);
          return;
        case "bool":
          if (typeof value !== "boolean") reject(path, "type");
          budget.charge(value ? 4 : 5, path);
          return;
        case "int":
          if (typeof value !== "bigint") reject(path, "type");
          integers.set(value, encodeInteger(value, budget, path));
          return;
        case "float":
          if (typeof value !== "number") reject(path, "type");
          if (!Number.isFinite(value)) reject(path, "nonfinite");
          budget.charge(Object.is(value, -0) ? 2 : JSON.stringify(value).length, path);
          return;
        default:
          throw new TypeError("unknown codec primitive");
      }
    }
    // A variant is a wire wrapper over an unwrapped nominal record. The record
    // itself enters active while walking the leaf, not twice for this wrapper.
    if (node.kind === "variant") {
      const selected = leaf(node, value, path);
      keys(["case", "value"], path);
      budget.visit(depth + 1, childPath(path, "case"));
      textCost(selected.name, childPath(path, "case"));
      visit(selected, value, childPath(path, "value"), depth + 1);
      return;
    }
    if (value === null || typeof value !== "object") reject(path, "type");
    if (active.has(value)) reject(path, "cycle");
    active.add(value);
    try {
      if (node.kind === "array") {
        if (!checkedData(() => Array.isArray(value), path)) reject(path, "type");
        const length = checkedData(() => dataProperty(value, "length"), path) as number;
        budget.charge(2 + Math.max(0, length - 1), path);
        const items = checkedData(() => dataArray(value), path);
        for (let i = 0; i < items.length; i++)
          visit(get(node.element!), items[i], childPath(path, i), depth + 1);
        return;
      }
      if (node.kind !== "record" && node.kind !== "error")
        throw new TypeError("unsupported codec node");
      if (recordIdentity(value) !== node.identity) reject(path, "type");
      const data = object(value, path),
        fields = node.fields ?? [];
      keys(
        fields.map((field) => field.name),
        path,
      );
      for (const field of fields)
        visit(
          get(field.type),
          required(data, field.name, path),
          childPath(path, field.name),
          depth + 1,
        );
      extras(
        data,
        fields.map((field) => field.name),
        path,
      );
    } finally {
      active.delete(value);
    }
  }
  visit(get(schema.root), input, "", 0);
  // JSON.stringify drives traversal. Access facades expose one container at a
  // time in schema order; they do not construct a second complete value tree.
  function view(node: SchemaNode, value: unknown): unknown {
    if (node.kind === "primitive") {
      if (node.name === "int") return nativeJSON.rawJSON(integers.get(value as bigint)!);
      if (node.name === "float" && Object.is(value, -0)) return nativeJSON.rawJSON("-0");
      return value;
    }
    if (node.kind === "variant") {
      const selected = leaf(node, value, "");
      const wrapper = Object.create(null);
      Object.defineProperty(wrapper, "case", { enumerable: true, value: selected.name });
      Object.defineProperty(wrapper, "value", {
        enumerable: true,
        get: () => view(selected, value),
      });
      return wrapper;
    }
    const facade = node.kind === "array" ? [] : Object.create(null);
    if (node.kind === "array") {
      facade.length = (value as unknown[]).length;
      for (let i = 0; i < (value as unknown[]).length; i++)
        Object.defineProperty(facade, i, {
          enumerable: true,
          get: () => view(get(node.element!), dataProperty(value, String(i))),
        });
    } else
      for (const field of node.fields ?? [])
        Object.defineProperty(facade, field.name, {
          enumerable: true,
          get: () => view(get(field.type), dataProperty(value, field.name)),
        });
    return facade;
  }
  const encoded = encoder.encode(JSON.stringify(view(get(schema.root), input)));
  if (encoded.length !== bytes - budget.remaining)
    throw new TypeError("codec preflight/formatter length mismatch");
  return ownBytes(encoded);
}

export type CodecIds = Readonly<{ invalidData: string; readFailed: string; cancelled: string }>;
type JsonlHandler = (value: unknown, context?: AssertionContext) => Promise<Completion<unknown>>;

export function createCodec<T>(
  schema: Schema,
  domain: ReturnType<typeof createDomainRuntime>,
  ids: CodecIds,
) {
  const fail = (
    identity: string,
    fields: readonly (readonly [string, unknown])[],
    cause?: unknown,
  ) => failure(domain.create(identity, record(identity, fields), origin, cause));
  function run<T>(operation: () => T): Completion<T> {
    try {
      return success(operation());
    } catch (cause) {
      if (!(cause instanceof CodecIssue)) throw cause;
      return fail(ids.invalidData, [
        ["path", cause.path],
        ["reason", cause.reason],
      ]);
    }
  }
  const reasonFor = (cause: unknown): string => {
    if (typeof cause === "object" && cause !== null) {
      if ((cause as { name?: unknown }).name === "AbortError") return "aborted";
      if (typeof (cause as { code?: unknown }).code === "string")
        return (cause as { code: string }).code;
    }
    return "io_error";
  };
  return Object.freeze({
    async encode(value: unknown, _context?: AssertionContext): Promise<Completion<Bytes>> {
      return run(() => encodeJSON(schema, value));
    },
    async decode(value: unknown, _context?: AssertionContext): Promise<Completion<T>> {
      return run(() => decodeJSON(schema, value) as T);
    },
    async decodeToml(value: unknown, _context?: AssertionContext): Promise<Completion<T>> {
      return run(() => decodeToml(schema, value) as T);
    },
    async decodeYaml(value: unknown, _context?: AssertionContext): Promise<Completion<T>> {
      return run(() => decodeYaml(schema, value) as T);
    },
    async decodeJson5(value: unknown, _context?: AssertionContext): Promise<Completion<T>> {
      return run(() => decodeJson5(schema, value) as T);
    },
    async decodeJsonl(value: unknown, _context?: AssertionContext): Promise<Completion<T[]>> {
      return run(() => decodeJsonlRecords(schema, value) as T[]);
    },
    async consume(
      reader: unknown,
      handler: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      if (typeof handler !== "function") throw new TypeError("invalid compiler jsonl handler");
      const budget = new Budget();
      const framer = createJsonlFramer();
      let index = 0;
      const deliver = async (line: Uint8Array): Promise<Completion<unknown>> => {
        let value: unknown;
        try {
          value = projectJsonlRecord(schema, line, index++, budget);
        } catch (cause) {
          if (!(cause instanceof CodecIssue)) throw cause;
          return fail(ids.invalidData, [
            ["path", cause.path],
            ["reason", cause.reason],
          ]);
        }
        return invoke(() => (handler as JsonlHandler)(value, undefined), origin);
      };
      return useReader(reader, async (cell: ReaderCell) => {
        if (cell.item !== "bytes" || cell.reader === undefined)
          return fail(ids.invalidData, [
            ["path", ""],
            ["reason", "type"],
          ]);
        const source = cell.reader;
        for (;;) {
          let next;
          try {
            next = await source.read();
          } catch (cause) {
            cell.errored = true;
            cell.carry = undefined;
            return fail(ids.readFailed, [["reason", reasonFor(cause)]], cause);
          }
          if (next.done) {
            if (cell.cancelled !== undefined)
              return fail(ids.cancelled, [["reason", cell.cancelled]]);
            break;
          }
          let lines: Uint8Array[];
          try {
            lines = framer.push(new Uint8Array(next.value));
          } catch (cause) {
            if (!(cause instanceof CodecIssue)) throw cause;
            return fail(ids.invalidData, [
              ["path", cause.path],
              ["reason", cause.reason],
            ]);
          }
          for (const line of lines) {
            const out = await deliver(line);
            if (out.kind !== "ok") return out as Completion<never>;
          }
        }
        const tail = framer.finish();
        if (tail !== undefined) {
          const out = await deliver(tail);
          if (out.kind !== "ok") return out as Completion<never>;
        }
        return success(BigInt(index));
      });
    },
  });
}

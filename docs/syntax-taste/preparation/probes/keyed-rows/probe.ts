import { writeFileSync } from "node:fs";

type Row = { key: string; id: string; quantity: string; price: string };
type Result = { ok: true; rows: Row[] } | { ok: false; problems: string[]; raw: Record<string, string[]> };

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) throw new Error(message);
}

function keyed(form: FormData): Result {
  const raw: Record<string, string[]> = {};
  const order: string[] = [];
  const rows = new Map<string, Partial<Row>>();
  const problems: string[] = [];
  const seen = new Set<string>();
  for (const [name, value] of form.entries()) {
    if (typeof value !== "string") {
      problems.push(`${name}: file is not text`);
      continue;
    }
    (raw[name] ??= []).push(value);
    if (name === "lines_order") {
      order.push(value);
      continue;
    }
    const match = /^lines\[([A-Za-z0-9_-]{1,64})\]\.(id|quantity|price)$/.exec(name);
    if (!match) {
      problems.push(`${name}: unknown field`);
      continue;
    }
    const [, key, field] = match;
    if (seen.has(name)) problems.push(`${name}: duplicate field`);
    seen.add(name);
    const row = rows.get(key) ?? { key };
    if (field === "id") row.id = value;
    if (field === "quantity") row.quantity = value;
    if (field === "price") row.price = value;
    rows.set(key, row);
  }
  if (new Set(order).size !== order.length) problems.push("lines_order: duplicate row key");
  for (const key of order) {
    if (!/^[A-Za-z0-9_-]{1,64}$/.test(key)) problems.push(`${key}: invalid row key`);
    const row = rows.get(key);
    if (!row) {
      problems.push(`${key}: missing row`);
      continue;
    }
    for (const field of ["id", "quantity", "price"] as const) {
      if (row[field] === undefined) problems.push(`${key}.${field}: missing field`);
    }
  }
  for (const key of rows.keys()) if (!order.includes(key)) problems.push(`${key}: unlisted row`);
  if (problems.length) return { ok: false, problems, raw };
  return { ok: true, rows: order.map((key) => rows.get(key) as Row) };
}

function makeKeyed(order: string[]): FormData {
  const form = new FormData();
  for (const key of order) {
    form.append("lines_order", key);
    form.append(`lines[${key}].id`, `id-${key}`);
    form.append(`lines[${key}].quantity`, `quantity-${key}`);
    form.append(`lines[${key}].price`, `price-${key}`);
  }
  return form;
}

const observations: Record<string, unknown> = {};
let checks = 0;
const complete = keyed(makeKeyed(["a", "b", "c"]));
assert(complete.ok, "complete rows should decode"); checks++;
assert(complete.rows.map((row) => row.key).join() === "a,b,c", "initial order"); checks++;
const moved = keyed(makeKeyed(["c", "a", "b"]));
assert(moved.ok && moved.rows.map((row) => row.key).join() === "c,a,b", "reorder"); checks++;
assert(moved.rows.every((row) => row.id === `id-${row.key}` && row.quantity === `quantity-${row.key}` && row.price === `price-${row.key}`), "keyed association"); checks++;
observations.complete = complete;
observations.reordered = moved;

const parallel = new FormData();
for (const value of ["id-a", "id-c"]) parallel.append("id[]", value);
for (const value of ["quantity-a", "quantity-b"]) parallel.append("quantity[]", value);
for (const value of ["price-b", "price-c"]) parallel.append("price[]", value);
const ids = parallel.getAll("id[]");
const quantities = parallel.getAll("quantity[]");
const prices = parallel.getAll("price[]");
assert(ids.length === 2 && quantities.length === 2 && prices.length === 2, "equal lengths"); checks++;
const zipped = ids.map((id, index) => ({ id, quantity: quantities[index], price: prices[index] }));
assert(zipped[0]?.id === "id-a" && zipped[0]?.price === "price-b", "false association occurs"); checks++;
observations.parallelEqualLengthOmission = zipped;

const partial = makeKeyed(["a"]);
partial.delete("lines[a].price");
const partialResult = keyed(partial);
assert(!partialResult.ok && partialResult.problems.includes("a.price: missing field"), "partial rejection"); checks++;
assert(partialResult.raw["lines[a].quantity"]?.[0] === "quantity-a", "partial raw retained"); checks++;
observations.partial = partialResult;

const duplicate = makeKeyed(["a"]);
duplicate.append("lines[a].quantity", "second");
const duplicateResult = keyed(duplicate);
assert(!duplicateResult.ok && duplicateResult.problems.includes("lines[a].quantity: duplicate field"), "duplicate field rejection"); checks++;
observations.duplicate = duplicateResult;

const duplicateOrder = makeKeyed(["a"]);
duplicateOrder.append("lines_order", "a");
const duplicateOrderResult = keyed(duplicateOrder);
assert(!duplicateOrderResult.ok && duplicateOrderResult.problems.includes("lines_order: duplicate row key"), "duplicate order rejection"); checks++;
observations.duplicateOrder = duplicateOrderResult;

const unknown = makeKeyed(["a"]);
unknown.append("lines[a].discount", "1");
const unknownResult = keyed(unknown);
assert(!unknownResult.ok && unknownResult.problems.includes("lines[a].discount: unknown field"), "unknown field rejection"); checks++;
observations.unknown = unknownResult;

const unlisted = makeKeyed(["a"]);
unlisted.append("lines[b].id", "id-b");
unlisted.append("lines[b].quantity", "quantity-b");
unlisted.append("lines[b].price", "price-b");
const unlistedResult = keyed(unlisted);
assert(!unlistedResult.ok && unlistedResult.problems.includes("b: unlisted row"), "unlisted rejection"); checks++;
observations.unlisted = unlistedResult;

writeFileSync(new URL("./results.json", import.meta.url), JSON.stringify({ bun: Bun.version, checks, observations }, null, 2) + "\n");
console.log(JSON.stringify({ bun: Bun.version, checks }));

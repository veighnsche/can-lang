// B1-02.05/07 native probe: Bun.SQL sqlite adapter value shapes, metadata,
// errors, open modes, and lock behavior. Bun 1.4.2, offline.
import { SQL } from "bun";
import { tmpdir } from "node:os";
import { join } from "node:path";

const shape = (v) => {
  if (typeof v === "bigint") return `bigint:${v}`;
  if (typeof v === "number" || typeof v === "string" || typeof v === "boolean" || v === null || v === undefined) return `${typeof v}:${String(v)}`;
  if (v instanceof Uint8Array) return `u8[${v.length}]:${Buffer.from(v).toString("hex")}`;
  if (Array.isArray(v)) return `array[${v.length}]`;
  return `object:${v?.constructor?.name}`;
};
const ownProps = (v) => {
  const out = {};
  for (const k of Object.getOwnPropertyNames(v)) {
    try { out[k] = shape(v[k]); } catch { out[k] = "<throw>"; }
  }
  return out;
};

const report = { bun: Bun.version, sections: {} };

// 1-3: value shapes with safeIntegers:true.
{
  const db = new SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  await db`CREATE TABLE t (i INTEGER, r REAL, t TEXT, b BLOB, n INTEGER)`;
  const blob = new Uint8Array([1, 2, 250]);
  await db`INSERT INTO t VALUES (${0}, ${1.5}, ${"héllo"}, ${blob}, ${null})`;
  await db`INSERT INTO t VALUES (${1}, ${-0.0}, ${""}, ${new Uint8Array(0)}, ${9007199254740993n})`;
  await db`INSERT INTO t VALUES (${-9223372036854775808n}, ${3}, ${"x"}, ${blob}, ${42})`;
  const rows = await db`SELECT i, r, t, b, n FROM t`;
  report.sections.values = rows.map((row) => Object.fromEntries(Object.entries(row).map(([k, v]) => [k, shape(v)])));
  // Bound booleans: what lands in an INTEGER column and what comes back?
  await db`CREATE TABLE bools (b INTEGER)`;
  await db`INSERT INTO bools VALUES (${true})`;
  const back = await db`SELECT b FROM bools`;
  report.sections.boolRoundTrip = { stored: shape(back[0].b), rawType: typeof back[0].b };
  await db.close();
}
// 4-5: statement metadata shapes.
{
  const db = new SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  await db`CREATE TABLE t (id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT)`;
  const ins = await db`INSERT INTO t (v) VALUES (${"a"})`;
  const upd = await db`UPDATE t SET v = ${"b"} WHERE id = ${1}`;
  const del = await db`DELETE FROM t WHERE id = ${999}`;
  report.sections.insertResult = { isArray: Array.isArray(ins), length: ins.length, props: ownProps(ins) };
  report.sections.updateResult = { isArray: Array.isArray(upd), length: upd.length, props: ownProps(upd) };
  report.sections.deleteNoMatch = { isArray: Array.isArray(del), length: del.length, props: ownProps(del) };
  await db.close();
}
// 6: error shapes (syntax, constraint, missing table).
{
  const db = new SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  await db`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT UNIQUE)`;
  await db`INSERT INTO t VALUES (${1}, ${"a"})`;
  const errs = {};
  for (const [k, fn] of [
    ["syntax", () => db`SELECT * FORM t`],
    ["unique", () => db`INSERT INTO t VALUES (${2}, ${"a"})`],
    ["pk", () => db`INSERT INTO t VALUES (${1}, ${"b"})`],
    ["noTable", () => db`SELECT * FROM missing`],
  ]) {
    try { await fn(); errs[k] = "NO-ERROR"; }
    catch (e) { errs[k] = { name: e.name, code: e.code, errno: e.errno, message: String(e.message).slice(0, 90) }; }
  }
  report.sections.errors = errs;
  await db.close();
}
// 7: busy behavior with two connections (rollback-journal mode).
{
  const file = join(tmpdir(), `can-busy-${Date.now()}.db`);
  const a = new SQL({ adapter: "sqlite", filename: file, safeIntegers: true });
  const b = new SQL({ adapter: "sqlite", filename: file, safeIntegers: true });
  await a`CREATE TABLE t (v INTEGER)`;
  await a`PRAGMA journal_mode = DELETE`;
  const busy = await b`PRAGMA busy_timeout = 50`;
  report.sections.busyPragma = busy.map((row) => Object.fromEntries(Object.entries(row).map(([k, v]) => [k, shape(v)])));
  await a`BEGIN IMMEDIATE`;
  await a`INSERT INTO t VALUES (${1})`;
  let bErr = null;
  try { await b`INSERT INTO t VALUES (${2})`; } catch (e) { bErr = { name: e.name, code: e.code, errno: e.errno, message: String(e.message).slice(0, 80) }; }
  report.sections.busyWrite = bErr ?? "NO-ERROR";
  await a`ROLLBACK`;
  await a`INSERT INTO t VALUES (${3})`;
  const rows = await b`SELECT v FROM t`;
  report.sections.afterRollback = rows.map((r) => shape(r.v));
  await a.close(); await b.close();
}
// 8: file open modes.
{
  const missing = join(tmpdir(), `can-missing-${Date.now()}.db`);
  const modes = {};
  for (const [k, opts] of [
    ["ro-missing", { adapter: "sqlite", filename: missing, readonly: true }],
    ["rw-missing", { adapter: "sqlite", filename: missing, readonly: false, create: false }],
    ["rwc-missing", { adapter: "sqlite", filename: missing, readonly: false, create: true }],
  ]) {
    try {
      const db = new SQL(opts);
      await db`SELECT 1`;
      modes[k] = "OPENED";
      await db.close();
    } catch (e) { modes[k] = { name: e.name, code: e.code, message: String(e.message).slice(0, 80) }; }
  }
  report.sections.openModes = modes;
  try { (await import("node:fs/promises")).unlink(missing); } catch {}
}
// 9: LIMIT ? with bigint + :memory: close/reopen lifetime.
{
  const db = new SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  await db`CREATE TABLE t (v INTEGER)`;
  await db`INSERT INTO t VALUES (${1})`;
  await db`INSERT INTO t VALUES (${2})`;
  await db`INSERT INTO t VALUES (${3})`;
  const two = await db`SELECT v FROM t LIMIT ${2n}`;
  report.sections.limitBigint = two.map((r) => shape(r.v));
  await db.close();
  let after = null;
  try { await db`SELECT 1`; after = "WORKED"; } catch (e) { after = { name: e.name, code: e.code, message: String(e.message).slice(0, 80) }; }
  report.sections.useAfterClose = after;
}
// 10: without safeIntegers (confirm the rounding hazard on this target).
{
  const db = new SQL({ adapter: "sqlite", filename: ":memory:" });
  const rows = await db`SELECT ${9007199254740993n} AS n`;
  report.sections.noSafeIntegers = shape(rows[0].n);
  await db.close();
}
// 11: open phases (where does ro-missing surface?) and bound PRAGMA values.
{
  const missing = join(tmpdir(), `can-open-${Date.now()}.db`);
  const phases = {};
  let db = null;
  try { db = new SQL({ adapter: "sqlite", filename: missing, readonly: true, safeIntegers: true }); phases.construct = "OK"; }
  catch (e) { phases.construct = `THROW ${e.name} ${e.code}`; }
  if (db) {
    try { await db.connect(); phases.connect = "OK"; }
    catch (e) { phases.connect = `THROW ${e.name} ${e.code}`; }
    try { await db`SELECT 1`; phases.query = "OK"; }
    catch (e) { phases.query = `THROW ${e.name} ${e.code}`; }
    try { await db.close(); } catch {}
  }
  report.sections.openPhases = phases;
  const mem = new SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  try { await mem`PRAGMA busy_timeout = ${50}`; report.sections.pragmaBound = "OK"; }
  catch (e) { report.sections.pragmaBound = `THROW ${e.name} ${e.code} ${String(e.message).slice(0, 50)}`; }
  await mem.close();
}
console.log(JSON.stringify(report, null, 1));

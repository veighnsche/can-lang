import { test, expect } from "bun:test";
import { runEntry } from "../entry.ts";
import { success, failure, type Completion } from "../completion.ts";
import { captureStandard } from "../failure.ts";
import { intDivide } from "../primitive.ts";

const origin = {source: "can:test", start: 0, end: 1, invocation: []};

test("root awaits main and passes only a frozen copy of the supplied application arguments", async () => {
  const events: string[] = [];
  const input = ["--inspect", "héllo", "", "--"];
  let release!: () => void;
  const gate = new Promise<void>(resolve => {release = resolve;});
  const pending = runEntry(() => {events.push("init");}, async args => {
    events.push("main");
    expect(args).toEqual(input);
    expect(args).not.toBe(input);
    expect(Object.isFrozen(args)).toBe(true);
    await gate;
    events.push("done");
    return success(undefined);
  }, input, () => {throw new Error("unexpected diagnostic");});
  await Promise.resolve();
  expect(events).toEqual(["init", "main"]);
  release();
  expect(await pending).toBe(0);
  expect(events).toEqual(["init", "main", "done"]);
  expect(Object.isFrozen(input)).toBe(false);
});

test("initialization failure prevents main and awaits its diagnostic", async () => {
  let mainCalls = 0, line = "";
  let reported!:()=>void;const delivered=new Promise<void>(resolve=>{reported=resolve;});
  let release!: () => void;
  const gate = new Promise<void>(resolve => {release = resolve;});
  let settled = false;
  const pending = runEntry(() => {intDivide(1n, 0n);}, () => {
    mainCalls++; return success(undefined);
  }, [], async report => {line = report; reported(); await gate;}).then(status => {settled = true; return status;});
  await delivered;
  expect(mainCalls).toBe(0);
  expect(settled).toBe(false);
  expect(JSON.parse(line)).toMatchObject({phase: "initialization", channel: "standard", category: "arithmetic"});
  release();
  expect(await pending).toBe(1);
});

test("root report preserves failure identity without native message, stack or origin disclosure", async () => {
  const secret = "private-token /Users/private/password";
  const occurrence = captureStandard(new Error(secret), {...origin, source: secret});
  const reports: string[] = [];
  expect(await runEntry(() => {}, () => failure(occurrence), [], line => {reports.push(line);})).toBe(1);
  expect(await runEntry(() => {}, () => {throw occurrence;}, [], line => {reports.push(line);})).toBe(1);
  expect(JSON.parse(reports[0]).occurrence).toBe(JSON.parse(reports[1]).occurrence);
  expect(reports.join("")).not.toContain(secret);
  expect(JSON.parse(reports[0])).toMatchObject({phase: "main", channel: "standard", category: "native_exception"});
});

test("forged and non-void main results fail without invoking a then getter", async () => {
  let reads = 0;
  const unboxed = Object.defineProperty({}, "then", {get() {reads++; throw new Error("secret");}});
  for (const result of [unboxed, success("wrong")]) {
    let line = "";
    expect(await runEntry(() => {}, () => result as Completion<void>, [], value => {line = value;})).toBe(1);
    expect(JSON.parse(line).category).toBe("native_exception");
  }
  expect(reads).toBe(0);
});

test("diagnostic write rejection retains nonzero status", async () => {
  expect(await runEntry(() => {throw new Error("hidden");}, () => success(undefined), [], async () => {
    throw new Error("closed stderr");
  })).toBe(1);
});

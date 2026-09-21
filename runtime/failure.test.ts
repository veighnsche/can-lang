import { expect, test } from "bun:test";
import { captureStandard, describeNativeFailure, standardFailureDiagnostics, standardFailureKind, standardFailureMessage, standardFailureOccurrenceID, isStandardFailure, cleanupFailure, resourceStateFailure, assertionFailure, assertionFailureClasses, nonfiniteSortKeyFailure } from "./failure";
import { intDivide, intRemainder, intPower, index } from "./primitive";
const origin = { source: "app::run", start: 10, end: 20, invocation: ["root", "call0"] };

test("native descriptions never invoke application behavior", () => {
  let invoked = 0;
  const trap = () => { invoked++; throw new Error("secret"); };
  const hostile = { get message() { return trap(); }, get name() { return trap(); }, get constructor() { return trap(); }, toString: trap, toJSON: trap };
  expect(describeNativeFailure(hostile)).toBe("native object");
  const proxy = new Proxy({}, { get: trap, getPrototypeOf: trap, ownKeys: trap, getOwnPropertyDescriptor: trap });
  expect(describeNativeFailure(proxy)).toBe("native proxy");
  const revoked = Proxy.revocable(new Error("secret"), {}); revoked.revoke();
  expect(describeNativeFailure(revoked.proxy)).toBe("native proxy");
  const native = new TypeError("safe");
  Object.defineProperty(native, "name", { get: trap });
  Object.defineProperty(native, "message", { get: trap });
  Object.defineProperty(native, "stack", { get: trap });
  expect(describeNativeFailure(native)).toBe("TypeError");
  const prototypeProxy = new Error("safe"); Object.setPrototypeOf(prototypeProxy, proxy);
  expect(describeNativeFailure(prototypeProxy)).toBe("Error: safe");
  const circular: any = { credential: "secret" }; circular.self = circular;
  expect(describeNativeFailure(circular)).toBe("native object");
  expect(invoked).toBe(0);
});

test("native scalar and Error descriptions follow fixed rules", () => {
  for (const [value, expected] of [[null,"null"],[undefined,"undefined"],[true,"true"],[2n,"2"],[NaN,"NaN"],[-0,"0"],["authored throw","authored throw"],[Symbol("secret"),"native symbol"],[()=>0,"native function"]] as const) expect(describeNativeFailure(value)).toBe(expected);
  expect(describeNativeFailure(new RangeError("range"))).toBe("RangeError: range");
  class Custom extends TypeError {};
  expect(describeNativeFailure(new Custom("custom"))).toBe("TypeError: custom");
  const error = new Error("safe"); Object.defineProperty(error,"name",{value:"Explicit"});
  Object.defineProperty(error,"credential",{value:"secret"}); Object.defineProperty(error,"responseBody",{value:"secret"});
  expect(describeNativeFailure(error)).toBe("Explicit: safe");
  const aggregate = new AggregateError([{ credential: "secret" }], "safe");
  expect(describeNativeFailure(aggregate)).toBe("AggregateError: safe");
});

test("origins are distinct and propagation preserves an occurrence", () => {
  const cause = new Error("same object");
  const first = captureStandard(cause, origin); const second = captureStandard(cause, origin);
  expect(standardFailureOccurrenceID(first)).not.toBe(standardFailureOccurrenceID(second));
  expect(captureStandard(first, {...origin, source:"outer"})).toBe(first);
  expect(standardFailureDiagnostics(first).cause).toBe(cause);
  expect(standardFailureDiagnostics(first).origin.source).toBe("app::run");
  expect(standardFailureKind(first)).toBe("native_exception");
  expect(Object.isFrozen(first)).toBe(true);
  expect(Object.getPrototypeOf(first)).toBe(null);
  expect(Reflect.ownKeys(first)).toEqual([]);
  expect(JSON.stringify(first)).toBe("{}");
  expect(isStandardFailure({kind:"native_exception",message:"same object"})).toBe(false);
  expect(isStandardFailure(new Proxy(first,{}))).toBe(false);
  expect(()=>standardFailureMessage({} as any)).toThrow("invalid standard failure");
  const invocation=["original"]; const occurrence=captureStandard("x",{...origin,invocation}); invocation[0]="changed";
  expect(standardFailureDiagnostics(occurrence).origin.invocation).toEqual(["original"]);
});

test("primitive and maintained categories use fixed messages", () => {
  const actions=[()=>intDivide(1n,0n),()=>intRemainder(1n,0n),()=>intPower(2n,-1n),()=>index([],0n)];
  const messages=["arithmetic: integer division by zero","arithmetic: integer remainder by zero","arithmetic: negative integer exponent","bounds: index out of range"];
  actions.forEach((action,i)=>{try{action();throw new Error("missing fault");}catch(cause){const occurrence=captureStandard(cause,origin);expect(standardFailureMessage(occurrence)).toBe(messages[i]);expect(standardFailureDiagnostics(occurrence).cause).toBe(cause);}});
  expect(standardFailureMessage(resourceStateFailure("secret",origin))).toBe("resource_state: invalid resource use");
  expect(standardFailureMessage(cleanupFailure({credential:"secret"},origin))).toBe("cleanup: automatic resource cleanup failed");
  expect(standardFailureMessage(nonfiniteSortKeyFailure(Infinity,origin))).toBe("arithmetic: nonfinite sort key");
  for(const name of assertionFailureClasses) expect(standardFailureMessage(assertionFailure(name,origin))).toBe("assertion: "+name);
  expect(()=>assertionFailure("secret" as any,origin)).toThrow("unknown assertion failure class");
});

test("description adapter defects retain the original cause with a fixed fallback", () => {
  const original=Object.getOwnPropertyDescriptor;
  const cause=new Error("safe");let occurrence;
  try {
    Object.getOwnPropertyDescriptor=()=>{throw new Error("private adapter defect")};
    occurrence=captureStandard(cause,origin);
  } finally {Object.getOwnPropertyDescriptor=original}
  expect(standardFailureMessage(occurrence!)).toBe("native failure");
  expect(standardFailureDiagnostics(occurrence!).cause).toBe(cause);
  expect(standardFailureDiagnostics(occurrence!).origin).toEqual(origin);
});

test("synthetic failures retain identity and original metadata when the first checked boundary is attached",()=>{
 const synthetic={source:"can:cli",start:0,end:0,invocation:[]};
 const failure=assertionFailure("missing fixture",synthetic), id=standardFailureOccurrenceID(failure);
 expect(captureStandard(failure,synthetic)).toBe(failure);
 expect(standardFailureDiagnostics(failure).boundaryOrigin).toBeUndefined();
 expect(captureStandard(failure,origin)).toBe(failure);
 expect(standardFailureDiagnostics(failure).origin).toEqual(synthetic);
 expect(standardFailureDiagnostics(failure).boundaryOrigin).toEqual(origin);
 expect(standardFailureOccurrenceID(failure)).toBe(id);
 expect(standardFailureMessage(failure)).toBe("assertion: missing fixture");
 captureStandard(failure,{source:"outer.can",start:99,end:100,invocation:[]});
 expect(standardFailureDiagnostics(failure).boundaryOrigin).toEqual(origin);
});

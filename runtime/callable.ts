// Native closures own execution and captures. This private receipt preserves
// creation-site/target/capture evidence for fixture identity; it is never a Can
// data projection or a diagnostic serialization surface.
type Receipt = Readonly<{site:string;target:string;captures:readonly unknown[]}>;
const receipts=new WeakMap<Function,Receipt>();
export function ownCallable<T extends Function>(site:string,target:string,captures:readonly unknown[],value:T):T {
 if (!site || !target || typeof value!=="function" || receipts.has(value)) throw new TypeError("invalid callable construction");
 receipts.set(value,Object.freeze({site,target,captures:Object.freeze([...captures])}));
 return Object.freeze(value);
}
export function callableReceipt(value:unknown):Receipt|undefined {
 return typeof value==="function" ? receipts.get(value) : undefined;
}

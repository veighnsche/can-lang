import { array, dataArray, dataKeys, dataProperty, record, recordIdentity } from "../data.ts";
import { Budget, reject, childPath } from "./budget.ts";
import { decodeInteger } from "./numbers.ts";

export type SchemaNode = Readonly<{identity:string;kind:string;name:string;element?:string;fields?:readonly Readonly<{name:string;type:string}>[];leaves?:readonly string[]}>;
export type Schema = Readonly<{root:string;nodes:readonly SchemaNode[]}>;
// Integer policies resolve a native number to bigint. The exact JSON path
// replays source numeric tokens; document formats project parsed values
// that must already be finite safe integers.
export type IntPolicy = (value:unknown,holder:object,key:string,path:string,budget:Budget)=>bigint;

const encoder = new TextEncoder();
export function graph(schema:Schema) {
  const nodes = new Map(schema.nodes.map(node=>[node.identity,node]));
  if(nodes.size!==schema.nodes.length) throw new TypeError("duplicate codec schema node");
  return (id:string):SchemaNode=>{const node=nodes.get(id);if(!node)throw new TypeError("missing codec schema node");return node;};
}
function scalar(text:string,path:string):void {
  if(new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(encoder.encode(text))!==text) reject(path,"unicode_scalar");
}
function checkedData<T>(read:()=>T,path:string):T {
  try{return read();}catch(cause){if(cause instanceof TypeError)reject(path,"type");throw cause;}
}
function object(value:unknown,path:string):Record<string,unknown> {
  if(value===null || typeof value!=="object" || checkedData(()=>Array.isArray(value),path)) reject(path,"type");
  // Only plain objects project: Temporal dates, class instances and other
  // native special objects fail here instead of masquerading as records
  // with missing members.
  const proto:object|null = Object.getPrototypeOf(value);
  if(proto!==Object.prototype && proto!==null) reject(path,"type");
  // Reject proxies/accessors through the same private data boundary used by the
  // runtime, before retrieving any authored property.
  checkedData(()=>dataKeys(value),path);
  return value as Record<string,unknown>;
}
function required(value:Record<string,unknown>,key:string,path:string):unknown {
  if(!Object.hasOwn(value,key))reject(childPath(path,key),"missing_member");
  return checkedData(()=>dataProperty(value,key),childPath(path,key));
}
function extras(value:Record<string,unknown>,fields:readonly string[],path:string):void {
  const extra=Object.keys(value).filter(key=>!fields.includes(key)).sort();
  if(extra.length)reject(childPath(path,extra[0]),"extra_member");
}

// exactInt replays native numeric tokens for bigint-exact integers. The
// token map comes from the capturing JSON reviver; a missing token means
// the value never crossed the native parser and fails closed.
export const exactInt=(tokens:WeakMap<object,Map<string,string>>):IntPolicy=>
  (value,holder,key,path,budget)=>{
    if(typeof value!=="number")reject(path,"type");
    return decodeInteger(tokens.get(holder)?.get(key)??"",budget,path);
  };

// projectValue walks one native parse result onto immutable Can values.
// Shared aliases project per occurrence; genuine cycles (YAML anchors can
// produce them) fail. Holder/key locate exact-JSON numeric tokens and are
// ignored by parsed-value integer policies.
export function projectValue(schema:Schema,parsed:unknown,holder:object,budget:Budget,intOf:IntPolicy):unknown {
  const get=graph(schema), active=new Set<object>();
  function visit(node:SchemaNode,value:unknown,site:object,key:string,path:string,depth:number):unknown {
    const container=node.kind!=="primitive";
    budget.visit(depth+(container?1:0),path);
    if(node.kind==="primitive"){
      switch(node.name){
        case "str":if(typeof value!=="string")reject(path,"type");scalar(value,path);return value;
        case "bool":if(typeof value!=="boolean")reject(path,"type");return value;
        case "float":if(typeof value!=="number")reject(path,"type");if(!Number.isFinite(value))reject(path,"nonfinite");return value;
        case "int":return intOf(value,site,key,path,budget);
        default:throw new TypeError("unknown codec primitive");
      }
    }
    if(node.kind==="array"){
      if(!Array.isArray(value))reject(path,"type");
      if(active.has(value))reject(path,"cycle");
      active.add(value);
      try{return array(value.map((item,index)=>visit(get(node.element!),item,value,String(index),childPath(path,index),depth+1)));}
      finally{active.delete(value);}
    }
    const data=object(value,path);
    if(active.has(data))reject(path,"cycle");
    active.add(data);
    try{
      if(node.kind==="variant"){
        const tag=required(data,"case",path);budget.visit(depth+1,childPath(path,"case"));
        if(typeof tag!=="string")reject(childPath(path,"case"),"type");scalar(tag,childPath(path,"case"));
        const leaf=node.leaves!.map(get).find(leaf=>leaf.name===tag);
        if(!leaf)reject(childPath(path,"case"),"variant_tag");
        const result=visit(leaf,required(data,"value",path),data,"value",childPath(path,"value"),depth+1);
        extras(data,["case","value"],path);return result;
      }
      if(node.kind!=="record" && node.kind!=="error")throw new TypeError("unsupported codec node");
      const fields=node.fields??[];
      const entries=fields.map(field=>[field.name,visit(get(field.type),required(data,field.name,path),data,field.name,childPath(path,field.name),depth+1)] as const);
      extras(data,fields.map(field=>field.name),path);
      return record(node.identity,entries);
    }finally{active.delete(data);}
  }
  return visit(get(schema.root),parsed,holder,"","",0);
}

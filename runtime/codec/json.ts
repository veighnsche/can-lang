import { array, dataArray, dataKeys, dataProperty, record, recordIdentity } from "../data.ts";
import { byteLength, copyBytes, ownBytes, type Bytes } from "../bytes.ts";
import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import { Budget, CodecIssue, reject, childPath, standaloneBytes } from "./budget.ts";
import { decodeInteger, encodeInteger } from "./numbers.ts";
import { scanJSON } from "./duplicates.ts";

export type SchemaNode = Readonly<{identity:string;kind:string;name:string;element?:string;fields?:readonly Readonly<{name:string;type:string}>[];leaves?:readonly string[]}>;
export type Schema = Readonly<{root:string;nodes:readonly SchemaNode[]}>;
const origin = Object.freeze({source:"can:codec",start:0,end:0,invocation:Object.freeze([])});
const encoder = new TextEncoder();
const nativeJSON = JSON as JSON & {rawJSON(text:string):unknown};
function graph(schema:Schema) {
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

export function decodeJSON(schema:Schema,input:unknown,bytes=standaloneBytes):unknown {
  const get=graph(schema), budget=new Budget(bytes);
  if(byteLength(input)>BigInt(bytes))reject("","byte_limit");
  let text:string;
  try{text=new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(copyBytes(input,origin));}
  catch(cause){if(cause instanceof TypeError)reject("","utf8");throw cause;}
  // BOM preservation makes a leading BOM fail native JSON syntax, as required.
  const duplicate=scanJSON(text);
  const tokens=new WeakMap<object,Map<string,string>>();
  let rootHolder:object|undefined;
  let parsed:unknown;
  try{
    parsed=JSON.parse(text,function(this:object,key:string,value:unknown,context?:{source?:string}){
      if(context?.source!==undefined){let holder=tokens.get(this);if(!holder){holder=new Map();tokens.set(this,holder);}holder.set(key,context.source);}
      if(key==="")rootHolder=this;
      return value;
    });
  }catch(cause){if(cause instanceof SyntaxError)reject("","invalid_json");throw cause;}
  if(duplicate!==undefined)reject(duplicate,"duplicate_member");
  function visit(node:SchemaNode,value:unknown,holder:object,key:string,path:string,depth:number):unknown {
    const container=node.kind!=="primitive";
    budget.visit(depth+(container?1:0),path);
    if(node.kind==="primitive"){
      switch(node.name){
        case "str":if(typeof value!=="string")reject(path,"type");scalar(value,path);return value;
        case "bool":if(typeof value!=="boolean")reject(path,"type");return value;
        case "float":if(typeof value!=="number")reject(path,"type");if(!Number.isFinite(value))reject(path,"nonfinite");return value;
        case "int":if(typeof value!=="number")reject(path,"type");return decodeInteger(tokens.get(holder)?.get(key)??"",budget,path);
        default:throw new TypeError("unknown codec primitive");
      }
    }
    if(node.kind==="array"){
      if(!Array.isArray(value))reject(path,"type");
      return array(value.map((item,index)=>visit(get(node.element!),item,value,String(index),childPath(path,index),depth+1)));
    }
    const data=object(value,path);
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
  }
  return visit(get(schema.root),parsed,rootHolder!,"","",0);
}

export function encodeJSON(schema:Schema,input:unknown,bytes=standaloneBytes):Bytes {
  const get=graph(schema), budget=new Budget(bytes), active=new Set<object>();
  const integers=new Map<bigint,string>();
  function textCost(value:string,path:string):void {
    if(value.length>budget.remaining)reject(path,"byte_limit");
    scalar(value,path);budget.charge(encoder.encode(JSON.stringify(value)).length,path);
  }
  function keys(names:readonly string[],path:string):void {
    budget.charge(2+Math.max(0,names.length-1)+names.length,path);
    for(const name of names)textCost(name,path);
  }
  function leaf(node:SchemaNode,value:unknown,path:string):SchemaNode {
    const identity=recordIdentity(value), found=node.leaves!.map(get).find(leaf=>leaf.identity===identity);
    if(!found)reject(path,"variant_tag");return found;
  }
  function visit(node:SchemaNode,value:unknown,path:string,depth:number):void {
    const container=node.kind!=="primitive";
    budget.visit(depth+(container?1:0),path);
    if(node.kind==="primitive"){
      switch(node.name){
        case "str":if(typeof value!=="string")reject(path,"type");textCost(value,path);return;
        case "bool":if(typeof value!=="boolean")reject(path,"type");budget.charge(value?4:5,path);return;
        case "int":if(typeof value!=="bigint")reject(path,"type");integers.set(value,encodeInteger(value,budget,path));return;
        case "float":if(typeof value!=="number")reject(path,"type");if(!Number.isFinite(value))reject(path,"nonfinite");budget.charge(Object.is(value,-0)?2:JSON.stringify(value).length,path);return;
        default:throw new TypeError("unknown codec primitive");
      }
    }
    // A variant is a wire wrapper over an unwrapped nominal record. The record
    // itself enters active while walking the leaf, not twice for this wrapper.
    if(node.kind==="variant"){
      const selected=leaf(node,value,path);keys(["case","value"],path);
      budget.visit(depth+1,childPath(path,"case"));textCost(selected.name,childPath(path,"case"));
      visit(selected,value,childPath(path,"value"),depth+1);return;
    }
    if(value===null || typeof value!=="object")reject(path,"type");
    if(active.has(value))reject(path,"cycle");active.add(value);
    try{
      if(node.kind==="array"){
        if(!checkedData(()=>Array.isArray(value),path))reject(path,"type");
        const length=checkedData(()=>dataProperty(value,"length"),path) as number;
        budget.charge(2+Math.max(0,length-1),path);
        const items=checkedData(()=>dataArray(value),path);
        for(let i=0;i<items.length;i++)visit(get(node.element!),items[i],childPath(path,i),depth+1);
        return;
      }
      if(node.kind!=="record" && node.kind!=="error")throw new TypeError("unsupported codec node");
      if(recordIdentity(value)!==node.identity)reject(path,"type");
      const data=object(value,path), fields=node.fields??[];
      keys(fields.map(field=>field.name),path);
      for(const field of fields)visit(get(field.type),required(data,field.name,path),childPath(path,field.name),depth+1);
      extras(data,fields.map(field=>field.name),path);
    }finally{active.delete(value);}
  }
  visit(get(schema.root),input,"",0);
  // JSON.stringify drives traversal. Access facades expose one container at a
  // time in schema order; they do not construct a second complete value tree.
  function view(node:SchemaNode,value:unknown):unknown {
    if(node.kind==="primitive"){
      if(node.name==="int")return nativeJSON.rawJSON(integers.get(value as bigint)!);
      if(node.name==="float" && Object.is(value,-0))return nativeJSON.rawJSON("-0");
      return value;
    }
    if(node.kind==="variant"){
      const selected=leaf(node,value,"");
      const wrapper=Object.create(null);
      Object.defineProperty(wrapper,"case",{enumerable:true,value:selected.name});
      Object.defineProperty(wrapper,"value",{enumerable:true,get:()=>view(selected,value)});return wrapper;
    }
    const facade=node.kind==="array"?new Array((value as unknown[]).length):Object.create(null);
    if(node.kind==="array"){
      for(let i=0;i<(value as unknown[]).length;i++)Object.defineProperty(facade,i,{enumerable:true,get:()=>view(get(node.element!),dataProperty(value,String(i)))});
    }else for(const field of node.fields??[])Object.defineProperty(facade,field.name,{enumerable:true,get:()=>view(get(field.type),dataProperty(value,field.name))});
    return facade;
  }
  const encoded=encoder.encode(JSON.stringify(view(get(schema.root),input)));
  if(encoded.length!==bytes-budget.remaining)throw new TypeError("codec preflight/formatter length mismatch");
  return ownBytes(encoded);
}

export function createCodec<T>(schema:Schema,domain:ReturnType<typeof createDomainRuntime>,invalidData:string) {
  function run<T>(operation:()=>T):Completion<T> {
    try{return success(operation());}catch(cause){
      if(!(cause instanceof CodecIssue))throw cause;
      return failure(domain.create(invalidData,record(invalidData,[["path",cause.path],["reason",cause.reason]]),origin));
    }
  }
  return Object.freeze({
    async encode(value:unknown,_context?:AssertionContext):Promise<Completion<Bytes>>{return run(()=>encodeJSON(schema,value));},
    async decode(value:unknown,_context?:AssertionContext):Promise<Completion<T>>{return run(()=>decodeJSON(schema,value) as T);},
  });
}

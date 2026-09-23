import {test,expect} from "bun:test";
import {prepareRequest,headerSnapshot,type Connection} from "../transport/request.ts";
import {transportProblem} from "../transport/deadline.ts";
const connection:Connection={endpoint:"https://example.test/api/",timeoutMilliseconds:1000,maxBodyBytes:100,headers:[{name:"x_default",value:"old"}],bearerEnvironment:"TOKEN"};
test("request preparation uses native URL query encoding and complete header replacement",()=>{
 let reads=0;const prepared=prepareRequest(connection,"items",[{name:"a_b",value:["a b","+%",""]},{name:"omit",value:[]}],[{name:"x_default",value:["one","two"]}],()=>{reads++;return "secret";});
 expect(prepared.url.href).toBe("https://example.test/api/items?a_b=a+b&a_b=%2B%25&a_b=");
 expect(prepared.headers.get("x-default")).toBe("one, two");expect(prepared.headers.get("authorization")).toBe("Bearer secret");expect(reads).toBe(1);
});
test("invalid request URLs fail before observing credentials",()=>{
 for(const [path,reason] of [["https://other.test/","origin"],["/x?","path_query"],["/#","fragment"],["https://user@example.test/","userinfo"],["\ud800","url"]] as const){
  let reads=0;try{prepareRequest(connection,path,[],[],()=>{reads++;return "secret";});throw new Error("accepted");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"invalid",reason});}expect(reads).toBe(0);
 }
});
test("credentials and header ownership fail without disclosing values",()=>{
 for(const credential of [undefined,"","a\nb","\u0100"]){
  try{prepareRequest(connection,"/",[],[],()=>credential);throw new Error("accepted");}
  catch(cause){expect(transportProblem(cause)).toEqual(credential===undefined||credential===""?{kind:"credential"}:{kind:"invalid",reason:"credential_value"});expect(String(cause)).toBe("Error: Can transport failure");}
 }
 for(const name of ["host","authorization","content_length"]){
  try{prepareRequest(connection,"/",[],[{name,value:"x"}],()=>"secret");throw new Error("accepted");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"invalid",reason:"header_name"});}
 }
 for(const value of ["badĀ","\ud800","x\ny"]){
  let reads=0;try{prepareRequest(connection,"/",[],[{name:"x_probe",value}],()=>{reads++;return "secret";});throw new Error("accepted");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"invalid",reason:"header_value"});}expect(reads).toBe(0);
 }
});
test("header snapshots retain separate cookies and normalized combined fields",()=>{
 const headers=new Headers();headers.append("Set-Cookie","a=1");headers.append("Set-Cookie","b=2");headers.append("X-Test","one");headers.append("X-Test","two");
 expect(headerSnapshot(headers)).toEqual([{name:"set-cookie",value:"a=1"},{name:"set-cookie",value:"b=2"},{name:"x-test",value:"one, two"}]);
});

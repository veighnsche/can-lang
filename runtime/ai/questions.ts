import type {Completion} from "../completion.ts";
import {array,record} from "../data.ts";
import {encodeJSON,type Schema,type SchemaNode} from "../codec/json.ts";
import {parseDocument} from "../codec/document.ts";
import {reject} from "../codec/budget.ts";
export type NoulDescriptor=Readonly<{kind:"noul";instructions:string;trueDescription:string;falseDescription:string;minimum:number}>;
export type ChoiceDescriptor=Readonly<{kind:"choice";instructions:string;options:readonly Readonly<{key:string;description:string}>[];minimum?:number}>;
export type ScoreDescriptor=Readonly<{kind:"score";instructions:string;levels:readonly string[];minimum?:number}>;
export type QuestionDescriptor=NoulDescriptor|ChoiceDescriptor|ScoreDescriptor;
export type Answer=Readonly<{kind:"noul";probability:number}>|Readonly<{kind:"choice";choice:string;confidence:number;probabilities:readonly number[]}>|Readonly<{kind:"score";score:number;confidence:number;probabilities:readonly number[]}>;
export class QuestionIssue extends Error {
 constructor(readonly question:string,readonly reason:"instructions"|"criterion"|"option_count"|"option_key"|"duplicate_option"|"level_count"|"threshold"|"batch_count"){super("invalid AI question");}
}
export class AnswerIssue extends Error {
 constructor(readonly question:string,readonly reason:"envelope"|"question_ids"|"answer_type"|"missing_value"|"probability"|"confidence"|"distribution_keys"|"distribution_sum"|"selected_key"|"selected_probability"|"score"|"legend"){super("invalid AI answer");}
}
const probability=(value:unknown):value is number=>typeof value==="number"&&Number.isFinite(value)&&value>=0&&value<=1;
const scalarText=(value:unknown):value is string=>typeof value==="string"&&value.isWellFormed()&&value.trim()!=="";
export function validateQuestions(questions:readonly QuestionDescriptor[]):void{
 if(questions.length<1||questions.length>256)throw new QuestionIssue("","batch_count");
 for(const [index,question] of questions.entries()){
  const id=`q${index}`;
  if(!scalarText(question.instructions))throw new QuestionIssue(id,"instructions");
  if(question.kind==="noul"){
   if(!scalarText(question.trueDescription)||!scalarText(question.falseDescription))throw new QuestionIssue(id,"criterion");
  }else if(question.kind==="choice"){
   if(question.options.length<2||question.options.length>255)throw new QuestionIssue(id,"option_count");
   const seen=new Set<string>();
   for(const option of question.options){
    if(typeof option.key!=="string"||option.key.length===0||option.key.length>256||!option.key.isWellFormed()||new TextEncoder().encode(option.key).length>256)throw new QuestionIssue(id,"option_key");
    if(seen.has(option.key))throw new QuestionIssue(id,"duplicate_option");seen.add(option.key);
    if(!scalarText(option.description))throw new QuestionIssue(id,"criterion");
   }
  }else if(question.kind==="score"){
   if(question.levels.length<2||question.levels.length>10)throw new QuestionIssue(id,"level_count");
   for(const description of question.levels)if(!scalarText(description))throw new QuestionIssue(id,"criterion");
  }else throw new TypeError("unknown checked question kind");
  if((question.kind==="noul"||question.minimum!==undefined)&&!probability(question.minimum))throw new QuestionIssue(id,"threshold");
 }
}
export function encodeQuestions(model:string,stateSchema:Schema,state:unknown,questions:readonly QuestionDescriptor[],bytes:number){
 validateQuestions(questions);
 const prefix="can:typesafe-private/",id=(name:string)=>prefix+name;
 if(stateSchema.nodes.some(node=>node.identity.startsWith(prefix)))throw new TypeError("protocol schema identity collision");
 const nodes:SchemaNode[]=[{identity:id("str"),kind:"primitive",name:"str"},{identity:id("strings"),kind:"array",name:"strings",element:id("str")}];
 const descriptors=questions.map((question,index)=>{
  const key=`q${index}`,criteriaID=id(key+"/criteria"),descriptorID=id(key);
  let criteria:unknown,criteriaType:string;
  if(question.kind==="score"){criteria=array([...question.levels]);criteriaType=id("strings");}
  else{
   const entries=question.kind==="noul"?[["true",question.trueDescription],["false",question.falseDescription]] as const:question.options.map(option=>[option.key,option.description] as const);
   nodes.push({identity:criteriaID,kind:"record",name:"criteria",fields:entries.map(([name])=>({name,type:id("str")}))});
   criteria=record(criteriaID,entries);criteriaType=criteriaID;
  }
  nodes.push({identity:descriptorID,kind:"record",name:"descriptor",fields:[{name:"type",type:id("str")},{name:"instructions",type:id("str")},{name:"criteria",type:criteriaType}]});
  return [key,record(descriptorID,[["type",question.kind],["instructions",question.instructions],["criteria",criteria]])] as const;
 });
 nodes.push({identity:id("questions"),kind:"record",name:"questions",fields:questions.map((_,i)=>({name:`q${i}`,type:id(`q${i}`)}))},{identity:id("request"),kind:"record",name:"request",fields:[{name:"model",type:id("str")},{name:"state",type:stateSchema.root},{name:"questions",type:id("questions")}]});
 return encodeJSON({root:id("request"),nodes:[...stateSchema.nodes,...nodes]},record(id("request"),[["model",model],["state",state],["questions",record(id("questions"),descriptors)]]),bytes);
}
function object(value:unknown):value is Record<string,unknown>{return value!==null&&typeof value==="object"&&!Array.isArray(value);}
function exactKeys(value:unknown,keys:readonly string[]):value is Record<string,unknown>{return object(value)&&Object.keys(value).length===keys.length&&keys.every(key=>Object.hasOwn(value,key));}
export function decodeAnswers(input:unknown,questions:readonly QuestionDescriptor[],bytes:number):readonly Answer[]{
 const {parsed}=parseDocument(input,bytes);
 if(!object(parsed)||typeof parsed.model!=="string"||parsed.model.length===0||!object(parsed.answers))throw new AnswerIssue("","envelope");
 if(!parsed.model.isWellFormed())reject("/model","unicode_scalar");
 const answers=parsed.answers;
 if(!exactKeys(answers,questions.map((_,i)=>`q${i}`)))throw new AnswerIssue("","question_ids");
 return array(questions.map((question,i):Answer=>{
  const id=`q${i}`,answer=answers[id];
  if(!object(answer)||answer.type!==question.kind)throw new AnswerIssue(id,"answer_type");
  if(question.kind==="noul"){
   if(!Object.hasOwn(answer,"noul"))throw new AnswerIssue(id,"missing_value");
   if(!probability(answer.noul))throw new AnswerIssue(id,"probability");
   return Object.freeze({kind:"noul",probability:answer.noul});
  }
  const required=question.kind==="choice"?["choice","confidence","probabilities"]:["score","confidence","probabilities","legend"];
  if(required.some(key=>!Object.hasOwn(answer,key)))throw new AnswerIssue(id,"missing_value");
  if(!probability(answer.confidence))throw new AnswerIssue(id,"confidence");
  const keys=question.kind==="choice"?question.options.map(option=>option.key):question.levels.map((_,i)=>String(i));
  if(!exactKeys(answer.probabilities,keys))throw new AnswerIssue(id,"distribution_keys");
  const distribution=answer.probabilities;
  const probabilities=array(keys.map(key=>{const p=distribution[key];if(!probability(p))throw new AnswerIssue(id,"probability");return p;}));
  if(Math.abs(probabilities.reduce((sum,p)=>sum+p,0)-1)>0.000001)throw new AnswerIssue(id,"distribution_sum");
  if(question.kind==="choice"){
   if(typeof answer.choice!=="string"||!keys.includes(answer.choice))throw new AnswerIssue(id,"selected_key");
   if(probabilities[keys.indexOf(answer.choice)]!==Math.max(...probabilities))throw new AnswerIssue(id,"selected_probability");
   return Object.freeze({kind:"choice",choice:answer.choice,confidence:answer.confidence,probabilities});
  }
  const score=answer.score;
  if(typeof score!=="number"||!Number.isFinite(score)||score<0||score>keys.length-1||Math.abs(score-probabilities.reduce((sum,p,index)=>sum+p*index,0))>0.000001*Math.max(1,keys.length-1))throw new AnswerIssue(id,"score");
  const legend=answer.legend;
  if(!exactKeys(legend,keys)||!keys.every((key,index)=>legend[key]===question.levels[index]))throw new AnswerIssue(id,"legend");
  return Object.freeze({kind:"score",score,confidence:answer.confidence,probabilities});
 }));
}

export type PreparedQuestion<T>=Readonly<{descriptor:QuestionDescriptor;run:(answer:Answer)=>Promise<Completion<T>>}>;

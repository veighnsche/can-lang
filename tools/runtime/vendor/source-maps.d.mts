// Declaration surface for the six upstream exports retained by our bundle.
export type Position = {line:number;column:number};
export interface EncodedMap {
 [key:string]:unknown;
 version:3;file?:string;sourceRoot?:string;names:string[];sources:string[];
 sourcesContent:(string|null)[];mappings:string;ignoreList:number[];
}
export class GenMapping {constructor(options?:{file?:string;sourceRoot?:string});}
export function addMapping(map:GenMapping,segment:{generated:Position;source:string;original:Position;name?:string}):void;
export function toEncodedMap(map:GenMapping):EncodedMap;
export class TraceMap {constructor(map:EncodedMap,mapURL?:string);}
export function decodedMappings(map:TraceMap):number[][][];
export function originalPositionFor(map:TraceMap,position:Position):{source:string|null;line:number|null;column:number|null;name:string|null};

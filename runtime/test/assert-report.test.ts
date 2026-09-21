import {test,expect} from "bun:test";
import {createEvidence,recordEvidence,evidenceReport,evidenceSummary,type Evidence,type EvidenceLabel,type EvidenceScope} from "../assert/report.ts";

test("supplied completion evidence cannot promote itself to provider or target coverage",()=>{
 const value=createEvidence("assertion");
 recordEvidence(value,"real-can");recordEvidence(value,"supplied-completion");recordEvidence(value,"supplied-completion");
 expect(evidenceReport(value)).toEqual(["real-can","supplied-completion"]);
 expect(()=>recordEvidence(value,"bun-conformance")).toThrow("explicit matching job");
 expect(()=>recordEvidence(value,"live-quality")).toThrow("explicit matching job");
 expect(evidenceSummary([value])).toEqual({"real-can":1,"supplied-completion":1,"raw-provider-fixture":0,"bun-conformance":0,"live-quality":0});
});

test("release evidence retains distinct categories and immutable snapshots",()=>{
 const authored=createEvidence("assertion"),native=createEvidence("bun-conformance"),live=createEvidence("live-quality");
 recordEvidence(authored,"real-can");const before=evidenceReport(authored);
 recordEvidence(authored,"raw-provider-fixture");recordEvidence(native,"bun-conformance");recordEvidence(live,"live-quality");
 expect(before).toEqual(["real-can"]);expect(Object.isFrozen(before)).toBe(true);
 const report=evidenceSummary([authored,native,live]);
 expect(report).toEqual({"real-can":1,"supplied-completion":0,"raw-provider-fixture":1,"bun-conformance":1,"live-quality":1});
 expect(Object.isFrozen(report)).toBe(true);
 expect(()=>recordEvidence(native,"live-quality")).toThrow();
 expect(()=>recordEvidence(live,"bun-conformance")).toThrow();
});

test("evidence tokens and categories fail closed",()=>{
 expect(()=>evidenceReport({} as Evidence)).toThrow("invalid evidence record");
 expect(()=>evidenceSummary([{} as Evidence])).toThrow("invalid evidence record");
 expect(()=>createEvidence("invented" as EvidenceScope)).toThrow("invalid evidence scope");
 expect(()=>recordEvidence(createEvidence("assertion"),"invented" as EvidenceLabel)).toThrow("invalid evidence label");
});

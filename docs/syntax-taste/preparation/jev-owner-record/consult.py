"""Three fresh owner-record decisions; every explanatory field independently rewritten."""
from pathlib import Path
import json, os, sys, urllib.request, datetime
P=Path(__file__).resolve().parent
state={
'goals':[
'Can serves AI coding agents. Correct behavior and reliable edits lead; total tokens per successful task is secondary and unmeasured here. No compatibility is needed. Generated TypeScript should use native JS/Bun with only invariant adapters. Select planned syntax and policy without asking the user.',
'This agent-oriented language values correctness and dependable modification before its secondary whole-task token metric. There are zero external users to migrate, and no measured token advantage in this packet. Use equivalent native JS/Bun lowering. Engineering must settle this design without another user question.',
'The audience is coding agents, not human readers. Success and robust refactoring outrank a yet-unmeasured reduction in complete-task tokens. Old source and ABI need no preservation. Prefer JavaScript/Bun operations plus required contract checks in emitted TypeScript; make a design selection without user consultation.'],
'evidence':[
'An executed two-package baseline passed 11 assertion roots: an exported email factory rejects blank text and quantity factory rejects nonpositive numbers, yet importers obtain invalid values through constructors, with, JSON decoding, assertion inputs and supplied completions. Current record name followed by indented typed fields has no construction permission bit; provides controls public declaration names, and private types cannot occur in public signatures. Runtime records are frozen nominal objects. Equality already compares eligible reachable immutable data; resource/callable fields make equality ineligible.',
'All 11 roots passed in a checked experiment with values and importer packages. Factory rejection of empty email and zero quantity did not stop foreign direct construction, copy replacement, structural JSON projection, fixture arguments or mocked completion fabrication. Grammar uses record plus a name and typed field block; provides exports the entire declaration, so current private records cannot support public factory return types. Values use immutable nominal JS objects. Existing equality descends into eligible data and excludes containers reaching resources or callables.',
'A reproducible values/main project ran 11 passing assertions and demonstrated invalid email/quantity inhabitants made outside rejecting factories by ordinary constructor, with, JSON decoder, assertion parameter and supplied-result routes. Today provides is the public-name list, record introduces indented type/name fields, and a private type is prohibited in exported signatures. Lowering uses frozen nominal objects rather than mutable data. Structural immutable equality is available only when every reachable component is eligible, excluding resource and callable contents.'],
'fixed_contract':[
'Owner-controlled records have already been selected as planned semantics. Ownership means the canonical declaring package instance, not an import alias or project. Public type names can occur in signatures and variants. Only owner-authored source may construct, change representation with with, or destructure hidden fields, including assertions. Foreign code may hold, pass, whole-bind and nominal-leaf-test the value. A fixture may forward a genuine owner-produced value but cannot fabricate one. Generic specialization retains the lexical package of the generic definition; caller location grants no owner privilege. Owners are trusted to validate; the compiler does not prove factory predicates. A validated tenant identifier is no authorization proof.',
'The accepted technical direction separates exported type identity from minting rights. The canonical package that declares a record owns it; project membership and aliases give no extra permission. Signatures and closed variants can expose the type. Representation construction, replacement and hidden-field patterns belong to source in that package, also inside test assertions. An importer can transport the value, bind it whole or recognize its nominal leaf; supplied fixtures may reuse an admitted value but cannot create a counterfeit. A generic body keeps its declaration package on specialization. Validation is the owner author’s obligation, and identifier validity never establishes resource access authority.',
'Assume the planned owner boundary is adopted: only its defining canonical package instance controls a record’s creation, with updates and field destructuring, including harness expressions. An exported name remains usable as a signature type or variant member. Other packages retain transport, whole-value capture and leaf-identity matching, and may supply existing owner-minted objects in fixtures. Generic code is checked with its own lexical owner rather than whichever package instantiates it. Neither shared project location nor an alias changes ownership. This boundary trusts factories to enforce predicates; it cannot infer invoice authorization from a valid tenant ID.'],
'policy_context':[
'Compare spelling independently from field exposure and codec policy: all spelling alternatives have identical ownership and runtime semantics. Ordinary record syntax remains transparent. Exposing read-only fields cannot mint a protected value but exposes representation and adds field-level syntax. Exported ordinary projection functions need no new field grammar. Structural equality, when eligible, can reveal equality of hidden representation but not its fields; this is invariant protection, not secrecy. Generic codecs currently derive one schema used for encode and decode; recursive arrays/records/variants can hide a protected leaf. Explicit owner wire conversion uses a transparent DTO and validating factory, with no automatic custom-codec registry.',
'The declaration candidates differ only in grammar, not minting restrictions or emitted representation; ordinary records remain as they are. Field-read markers would permit direct projections and expose field names without permitting construction, at the cost of additional grammar. Public named getter functions are expressible already. Preserve eligible structural equality as an observation of full hidden state, with no confidentiality claim. Present document codecs share schema derivation between directions, and containers can nest protected records. A wire DTO plus authored factory can implement explicit conversion without registering implicit codec hooks.',
'Hold semantics constant when choosing the record header: normal records stay transparent and each candidate uses the same owner checks/native objects. A new field marker would expose named immutable projections safely for minting but broaden the language surface; existing exported functions can expose the same values explicitly. Equality of eligible hidden data remains observable and is not a secret-preserving boundary. The current schema graph supports both serialization directions, including recursively nested protected leaves. Ordinary wire records converted by owner functions avoid any automatic validator or codec-dispatch registry.']}
questions={
'declaration':{
'instructions':[
'Which one header should be the canonical planned spelling for owner-controlled records? Judge explicit agent intent and grammar consistency, without claiming measured token savings.',
'Select a single declaration form for the accepted ownership semantics, considering checked agent editing and fit with the existing record grammar. Token benefit is unknown.',
'Choose the source header to specify for this record boundary. Prefer a readily checked distinction for agents; do not infer token performance from word counts.'],
'criteria':{
'owner_modifier':[
'Use owner record email followed by the existing field block; owner is a contextual modifier before record, while provides independently publishes the type.',
'Write owner record email with normal indented fields. Parse owner as a record-prefix modifier and keep export controlled separately by provides.',
'Adopt owner record email; retain ordinary field grammar and make owner contextual in the declaration prefix. Public naming still uses provides.'],
'owned_declaration':[
'Introduce owned email as a separate declaration head with the same typed field block and owner semantics; provides independently exports it.',
'Choose a dedicated owned email declaration kind, reusing typed fields and identical package restrictions, with visibility still in provides.',
'Spell the new form owned email and treat it as its own declaration production; field syntax and ownership stay the same and provides exports the name.'],
'owner_suffix':[
'Use record email owner followed by existing typed fields; the postfix contextual word records owner-only creation while provides controls visibility.',
'Keep record first and append owner after its name: record email owner. Retain the same field block and independent provides list.',
'Adopt record email owner as a header qualifier after the type name; unchanged field grammar and provides still apply.']}},
'projections':{
'instructions':[
'Which observation surface should the initial owner-record contract specify, given hidden construction and unchanged eligible equality?',
'Choose how importers should read selected owner values in the first design while preserving the fixed creation and equality contract.',
'What initial projection policy best completes this boundary without changing the accepted ownership or equality semantics?'],
'criteria':{
'functions_only':[
'Hide every representation field from nonowners; expose selected values through ordinary exported projection functions. Add no field-level visibility syntax.',
'Make all stored fields package-private and publish existing named functions as getters; avoid adding another field modifier.',
'Use exported owner functions for observations while denying importer field access uniformly; leave field visibility grammar unextended.'],
'public_fields':[
'Add explicit public markers on selected fields for importer read-only access and matching those fields; unmarked fields remain owner-only, all constructor/update restrictions unchanged.',
'Allow marked public fields to be read and pattern-tested externally, retaining hidden unmarked fields and owner-exclusive minting/replacement.',
'Introduce field-level public annotations enabling immutable projections and matching of those slots outside the owner; protect remaining slots and all creation operations.']}},
'codec':{
'instructions':[
'Which codec policy should accompany the boundary, including nested protected leaves and code defined inside the owner?',
'Select the complete schema/codec admission rule for protected records at any depth, taking the defining package into account.',
'Choose how generic serialization and schema derivation should treat this record kind, both directly and through containers, in owner and importer code.'],
'criteria':{
'wire_only_everywhere':[
'Reject automatic encode, decode and wire-schema derivation for any type graph containing an owner record, even in its owner; require explicit owner projection to/from a transparent wire DTO and factory validation.',
'Deny all generic document codecs and derived wire schemas whenever a protected leaf is reachable, regardless of lexical owner. Convert with authored functions between the value and an ordinary DTO.',
'Use one context-independent prohibition on automatically derived schema, encoding and decoding of graphs reaching owner records. Owner code writes explicit wire-record conversion and validation instead.'],
'owner_derivation':[
'Permit generic encode, decode and schema derivation only when every reachable owner record is owned by the lexical source package; foreign code must use exported explicit conversions. Owners remain trusted to validate decoded values.',
'Let the defining package derive codecs if it owns all protected nodes in the type graph, while denying nonowner derivation. Export authored adapters and rely on owner discipline after decoder minting.',
'Allow automatic codecs/schema under a package privilege check covering all protected leaves; foreign callers need owner conversion functions, and the owner must ensure decoding does not bypass its intended predicates.']}}
}
for entries in state.values(): assert len(entries)==3 and len(set(entries))==3
for q in questions.values():
 assert len(set(q['instructions']))==3
 for entries in q['criteria'].values(): assert len(entries)==3 and len(set(entries))==3
for i in range(3):
 payload={'model':'jev-latest','state':{k:v[i] for k,v in state.items()},'questions':{k:{'type':'choice','instructions':q['instructions'][i],'criteria':{key:v[i] for key,v in q['criteria'].items()}} for k,q in questions.items()}}
 (P/f'request-{i+1}.json').write_text(json.dumps(payload,indent=2)+'\n')
(P/'wording-audit.json').write_text(json.dumps({'review':'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each state field, instruction and option description has a distinct complete phrasing. Code headers and technical identifiers are intentionally stable. No response or favored recommendation appears in subsequent request state.','same_facts':['11 assertion roots and five syntactic bypass expressions grouped as four creation routes','canonical package owner and lexical generic authority','same existing grammar, exports, frozen objects and structural equality','same three headers, two projection policies and two complete codec policies'],'limit':'Text inequality and manual semantic review do not establish absence of framing effects; agreement remains advice.'},indent=2)+'\n')
if '--send' in sys.argv:
 for i in range(1,4):
  req=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=(P/f'request-{i}.json').read_bytes(),headers={'Content-Type':'application/json','Authorization':'Bearer '+os.environ['TYPESAFE_API_KEY']})
  start=datetime.datetime.now(datetime.timezone.utc).isoformat()
  with urllib.request.urlopen(req,timeout=60) as r: body=r.read(); status=r.status
  (P/f'response-{i}.json').write_bytes(body+b'\n')
  (P/f'response-{i}.metadata.json').write_text(json.dumps({'startedAt':start,'status':status,'endpoint':'v1/systemone'},indent=2)+'\n')
  print(json.dumps({'request':i,**json.loads(body)}))

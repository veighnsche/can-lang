"""Three independently worded Jev judgments on bounded route-capture policy."""
from pathlib import Path
import datetime
import hashlib
import json
import os
import sys
import time
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent / "jev"
OUT.mkdir(exist_ok=True)

state = [
    {
        "objective": "Can serves AI coding agents. Correct behavior and dependable refactors lead; whole successful-task token use is a measured secondary objective. There are no external users or legacy-routing compatibility obligations. Emit equivalent native JavaScript/Bun operations and add adapters only for Can contracts. This is an optional increment after the selected exact-path source action, not its prerequisite.",
        "contract": "The planned action source form has `post \"/tenants/:tenant_id/invoices/:invoice_id\"` and `captures invoice_key`, where `invoice_key` has two `int` fields. Its checked identity links mount, method and URL builder. Future initial capture fields are only required `int` or `str`, one nonempty segment each. The builder must produce navigable `/tenants/1/invoices/7`; capture admission never proves authorization or existence. Can `int` is signed 64-bit; business positivity belongs to validation, not path type conversion.",
        "observations": "On pinned Bun 1.4.2, URL and the live server preserve `%2F`, `%5C`, `%25`, `%ZZ` and bad UTF-8 text in pathname. `decodeURIComponent` of a whole pathname changes `%2F` into a route slash and throws on malformed encoding. Segment decoding retains structural boundaries. URL normalizes raw or encoded `.`/`..` before a Bun Request callback, and raw backslash acts as a separator. The current Can ingress decodes the whole pathname and the current router admits exact literal routes only, so it cannot be directly reused for captures. Duplicate exact method/path mounts already reject. No capture adapter has been built.",
        "obligations": "Specify deterministic static/dynamic dispatch, reject genuinely ambiguous patterns, make recognized-path wrong-method return 405 with Allow, unmatched shape return 404, and invalid capture conversion or malformed encoding return 400 before the protected handler. Preserve one-pass segment encoding/decoding, exact int64 values, Unicode scalars and all five action status policies. Favor a small bounded rule rather than a general route DSL. Real native HTTP and rename tests still have to qualify the design."
    },
    {
        "objective": "This planning judgment concerns language features written by AI agents, without a human-readability goal. A reliable route and reliable edits matter most; total agent tokens per completed task are secondary evidence. The project need not honor old syntax for outside users. The eventual TypeScript should use Bun/JS behavior with only contract-preserving glue. Named captures may follow, but do not expand the already chosen exact-route action prototype.",
        "contract": "The optional action revision spells `post \"/tenants/:tenant_id/invoices/:invoice_id\"` followed by `captures invoice_key`. `invoice_key` contains the two `int` identifiers. The action's static symbol owns construction and mounting of the same POST. First-scope captures are single, nonempty, required segments typed `int` or `str`; `/tenants/1/invoices/7` is the target resource path. Can integer values span signed 64 bits. A typed route is no permission or deployed-resource witness.",
        "observations": "Bun 1.4.2 served requests show URL.pathname keeps escaped slash/backslash, escaped percent, invalid `%` sequences and invalid UTF-8 bytes until explicit decoding. Decoding the path in one call can manufacture a slash inside a captured value; decoding separate segments does not. Native URL construction removes literal and percent-encoded dot segments and treats an unescaped backslash as path structure before the handler sees it. Today Can normalizes by decoding the entire URL path and dispatches only exact mounts. There is no implemented dynamic matcher; duplicate exact method/path routes already fail.",
        "obligations": "Choose a bounded, predictable dispatch for literal and variable segments and reject remaining collisions. A path with an admitted shape but invalid capture should receive 400 before business code; a different shape should get 404; an admitted path with unsupported verb should give 405 and Allow. Builders/matchers need one encoding and decoding pass, int64 exactness and valid Unicode. Keep the existing five application response cases independent. The choice will require live HTTP and refactor evidence later."
    },
    {
        "objective": "Decide a narrow capture contract for future Can agent-authored code. Accuracy and sound edits rank ahead of measured total task tokens. No user base requires old route spellings, and native Bun/JavaScript should do equivalent work under minimal Can adapters. The current exact source action decision is already made; this question only addresses the separate typed-path trial.",
        "contract": "A source action may change from exact/query routing to `post \"/tenants/:tenant_id/invoices/:invoice_id\"` plus `captures invoice_key`. Its record has `int tenant_id` and `int invoice_id`; generated URL and mount use that declaration identity. The proposed first increment permits required nonempty per-segment `int` and `str`, giving `/tenants/1/invoices/7` for the sample. Can `int` is signed int64, and application rules decide whether a particular key must be positive. Syntax validity says nothing about the actor or stored invoice.",
        "observations": "With Bun 1.4.2, the live Request URL leaves encoded separators and malformed percent/UTF-8 in pathname. Existing whole-path decode would promote `%2F` to a separator. Splitting before strict decode keeps the captured segment separate. Dot and encoded-dot segments, and unescaped backslashes, undergo native URL normalization before request dispatch; source code cannot recover those original bytes from `Request.url`. The present router handles exact literal paths and rejects duplicate method/path registrations; no capture semantics exist yet.",
        "obligations": "The resulting proposal must distinguish bad capture/encoding as HTTP 400 without running the protected operation, absent path as 404, and wrong method on a known path as 405 plus Allow. Define static versus captured route selection and collisions, exact segment round trips, Unicode and signed 64-bit conversion. Do not infer that a builder grants access. The actual implementation still needs native HTTP, source-rename and UI acceptance cases."
    }
]

questions = [
    {
        "route_overlap": {
            "instructions": "Which bounded assembly and dispatch rule best serves a future typed-capture action beside exact static actions?",
            "criteria": {
                "reject_overlap": "Reject any static/captured path intersection at router assembly, regardless of verb; allow identical path shapes only when distinct methods have the same capture structure. This makes selection trivial and prevents method-dependent identity, but disallows common literal-special-case paths beside captures.",
                "static_first": "Rank literal segments ahead of captures across all methods; select one path identity before checking the verb. Reject duplicate or equally ranked overlapping patterns at assembly. Then return 405 and Allow from the selected path when its method is absent. This admits ordinary static-special-case routes while preserving a method-independent path meaning.",
                "method_first": "Filter routes by request method before preferring static segments, with 405 only when no method candidate matches but a path does. Reject ties within a verb. This admits more registrations but lets the same path resolve to different action identities according to method."
            }
        },
        "segment_policy": {
            "instructions": "What should the initial `str` capture builder and matcher admit after the native URL behavior in the evidence?",
            "criteria": {
                "strict_segment": "Encode one Unicode-scalar string segment with native encodeURIComponent and decode exactly once. Reject empty, decoded slash or backslash, a whole `.` or `..`, controls, malformed percent/UTF-8, and ill-formed Unicode. A builder rejects these values; an otherwise matching incoming capture returns 400. A native-normalized raw dot/backslash path can only become an unmatched 404.",
                "escaped_separator_data": "Keep per-segment decoding and UTF-8 checks, but admit decoded slash/backslash and percent-encode them in built URLs as ordinary `str` data. Reject dot segments and controls. This supports more identifiers while requiring every downstream URL, routing, authorization and logging consumer to retain segment boundaries.",
                "native_path": "Rely on URL.pathname normalization and decode the whole path before matching, admitting the platform's slash and dot behavior. This uses less adapter code but encoded separators can become route structure and raw dot/backslash input is rewritten before dispatch."
            }
        },
        "integer_wire": {
            "instructions": "Choose the `int` capture wire conversion for the first route trial, preserving typed equality and predictable URLs.",
            "criteria": {
                "canonical_signed": "Accept only `0` or `-?[1-9][0-9]*` within signed int64, with `-0`, plus, leading zero, exponent, decimal and overflow rejected as 400. Build from BigInt decimal text; application validation separately enforces positive resource keys.",
                "lenient_signed": "Accept signed decimal variants including plus, leading zeros and negative zero when BigInt fits signed int64, then build canonical decimal output. This tolerates more incoming URLs but different wire paths map to one typed capture.",
                "canonical_positive": "Accept only `[1-9][0-9]*` through signed int64 max and build that spelling. This matches the invoice key domain but makes generic `int` route captures implicitly positive and unable to represent zero or negative Can int values."
            }
        }
    },
    {
        "route_overlap": {
            "instructions": "Select a deterministic route-resolution policy for exact and captured action paths across HTTP verbs.",
            "criteria": {
                "reject_overlap": "Make the router refuse registrations whose path languages intersect, even when one is literal and another parameterized; permit the same structural pattern for different verbs. No request can choose a surprising route, at the cost of preventing `/invoices/new` next to `/invoices/:id`.",
                "static_first": "Use literal-over-parameter specificity without first restricting by verb. Assemble-time checks reject duplicate and same-specificity intersecting routes. Resolve the path identity once, then match the verb and report 405 with Allow if missing. Literal special cases remain usable, and a URL's chosen identity does not vary by verb.",
                "method_first": "Take only routes registered for the incoming verb and then choose literal over parameter; when none apply, search all methods for 405. Equal choices within a method fail assembly. This maximizes route combinations but an identical URL may name separate paths for separate verbs."
            }
        },
        "segment_policy": {
            "instructions": "Choose a first-scope string-segment rule that the native Bun request URL can support safely and round-trip exactly.",
            "criteria": {
                "strict_segment": "Apply encodeURIComponent per capture and strict decodeURIComponent per raw segment. Exclude empty values, slash, backslash, exact dot/dotdot, controls, unpaired surrogates and invalid escaped UTF-8. Return 400 when a matched capture is invalid; builders fail for inadmissible strings. Original raw dot or backslash structure normalized by URL can instead miss the shape and return 404.",
                "escaped_separator_data": "Split first and decode once, allowing escaped slash or backslash inside the string value and encoding it again in generated paths. Still exclude `.`/`..`, control characters and bad Unicode. More values fit, though other route or security consumers must never flatten the captured segment into a path.",
                "native_path": "Let the URL implementation normalize and decode the full path for the matcher, without a separate segment contract. It minimizes specialized matching work but `%2F` can alter the slash structure and dot/backslash paths have already changed."
            }
        },
        "integer_wire": {
            "instructions": "Which decimal text grammar should map one route segment to Can's signed 64-bit `int`?",
            "criteria": {
                "canonical_signed": "Take canonical signed decimals, `0` or `-?[1-9][0-9]*`, within int64 limits. Reject plus, leading zeros, `-0`, non-decimals and overflow as invalid captures (400). Serialize BigInt with decimal to one spelling; domain code decides whether an invoice ID must be positive.",
                "lenient_signed": "Parse plus signs, extra leading zeros and negative zero as well as ordinary decimal integers if within int64, normalizing to one builder spelling afterward. This accepts equivalent noncanonical links at the expense of multiple URLs for the same typed value.",
                "canonical_positive": "Use only positive canonical digits `[1-9][0-9]*` up to int64 maximum. This fits this example's resource identifiers yet limits all fields typed `int` in capture declarations to positive values."
            }
        }
    },
    {
        "route_overlap": {
            "instructions": "Which registration rule and path-versus-method ordering should the implementation specification prefer?",
            "criteria": {
                "reject_overlap": "Treat even literal-versus-variable intersections as ambiguous, reject them when routes are assembled across verbs, and only share exactly one shape among different methods. Runtime matching stays small but useful static exceptions cannot coexist with a broad captured pattern.",
                "static_first": "Admit static exceptions by ordering exact segments before capture segments independently of request method, while rejecting duplicates and unresolved equally specific overlaps. After selecting the path, use its allowed methods for 405 and Allow. The same URL has one route identity regardless of verb.",
                "method_first": "Find the incoming verb's route candidates first and select their most specific pattern, rejecting only method-local ties. Check other verbs for 405 when none match. Registration is permissive, though changing the verb may change which path declaration the URL denotes."
            }
        },
        "segment_policy": {
            "instructions": "Set the boundary for `str` path captures using the observed URL normalization and strict segment decoding behavior.",
            "criteria": {
                "strict_segment": "Use native per-segment percent encoding and one strict per-segment decode, with an adapter guard against empty strings, slash/backslash, exactly `.`/`..`, C0/DEL, malformed escaping and invalid Unicode scalars. Reject an invalid captured value with 400; the builder reports invalid input. If Bun already normalized raw dot/backslash structure into another shape, that path may be 404.",
                "escaped_separator_data": "Keep single-segment encoding/decoding and scalar validation, but preserve percent-escaped separators as `str` values instead of refusing them. Reject dots and controls. This allows more identifiers, provided all later consumers keep the segment typed rather than reinterpreting it as a path string.",
                "native_path": "Reuse native whole-path normalization as the entire capture matcher, with no explicit segment boundary guard. This is simple but an encoded slash can become a delimiter and URL dot/backslash normalization can silently redirect the original input."
            }
        },
        "integer_wire": {
            "instructions": "Pick a precise first-version numeric capture rule consistent with Can `int` and a canonical URL builder.",
            "criteria": {
                "canonical_signed": "Decode only canonical `0` or `-?[1-9][0-9]*` between negative and positive int64 endpoints. A leading plus/zero, `-0`, fractional or exponent form, and overflow yield 400. BigInt.toString yields the sole builder spelling, leaving resource positivity to domain rules.",
                "lenient_signed": "Convert several integer-text spellings, including `+1`, `01` and `-0`, within signed int64 and have the builder normalize them. Convenient inbound acceptance permits multiple textual URLs for one Can int.",
                "canonical_positive": "Restrict capture `int` to positive digits without leading zeros through int64 max. It matches IDs in this fixture but bakes a positive-only meaning into the otherwise signed `int` type."
            }
        }
    }
]

for i in range(3):
    payload = {"model": "jev-latest", "state": state[i], "questions": {key: {"type": "choice", **value} for key, value in questions[i].items()}}
    (OUT / f"request-{i+1}.json").write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")

def leaves(value, prefix=""):
    if isinstance(value, dict):
        for key, child in value.items():
            yield from leaves(child, f"{prefix}.{key}" if prefix else key)
    elif isinstance(value, str):
        yield prefix, value

paths = [dict(leaves({"state": state[i], "questions": questions[i]})) for i in range(3)]
stable = {"route_overlap", "segment_policy", "integer_wire", "reject_overlap", "static_first", "method_first", "strict_segment", "escaped_separator_data", "native_path", "canonical_signed", "lenient_signed", "canonical_positive"}
assert paths[0].keys() == paths[1].keys() == paths[2].keys()
assert all(len({paths[i][p] for i in range(3)}) == 3 for p in paths[0])
assert set(questions[0]) == set(questions[1]) == set(questions[2])
assert all(set(questions[0][q]["criteria"]) == set(questions[1][q]["criteria"]) == set(questions[2][q]["criteria"]) for q in questions[0])
(OUT / "wording-audit.json").write_text(json.dumps({
    "review": "Before transmission, compared all three full requests. Every version includes the same agent-first objective, existing exact source action, optional required int/str captures, signed int64, native Bun 1.4.2 observations, current whole-path decoding, 400/404/405 contract, security limits and the same alternatives. Each contextual statement, instruction and option description is separately rewritten; identifiers and exact code are retained. No prior model answer is included. The semantic review is human engineering judgment and cannot prove wording independence.",
    "rewritten_explanatory_paths": len(paths[0]),
    "all_distinct": True,
    "stable_option_keys": sorted(stable),
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        body = (OUT / f"request-{i}.json").read_bytes()
        for attempt in range(1, 4):
            started = datetime.datetime.now(datetime.timezone.utc).isoformat()
            request = urllib.request.Request("https://api.typesafe.ai/v1/systemone", data=body, headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
            try:
                with urllib.request.urlopen(request, timeout=60) as response:
                    raw, status = response.read(), response.status
                (OUT / f"response-{i}.json").write_bytes(raw + b"\n")
                (OUT / f"response-{i}.metadata.json").write_text(json.dumps({"startedAt": started, "status": status, "attempt": attempt, "endpoint": "v1/systemone", "requestSha256": hashlib.sha256(body).hexdigest(), "responseSha256": hashlib.sha256(raw).hexdigest()}, indent=2) + "\n")
                data = json.loads(raw)
                print(json.dumps({"request": i, "model": data.get("model"), "answers": data.get("answers"), "usage": data.get("usage")}), flush=True)
                break
            except urllib.error.HTTPError as error:
                (OUT / f"response-{i}.attempt-{attempt}.error.json").write_bytes(error.read() + b"\n")
                if error.code not in (429, 529) or attempt == 3:
                    raise
                time.sleep(2 ** attempt)

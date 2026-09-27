"""H08-followup bound-U measurement probes (live, bounded, ledgered).

Serial SystemOne calls against pinned model jev-1.13.0 answering bound-U
measurement questions only: baseline usage decode, triage-shaped usage
scale, worst-case-in-caps input, byte-identical determinism, and
over-budget enforcement semantics. Every call is recorded in
../usage-ledger.md with its usage; no secrets enter any artifact.

Probe bodies are deterministic. Request bodies stay in /tmp (m5/m6 are
hundreds of KB); only redacted records (hashes, lengths, usage,
answer summaries) plus response bodies (answers+usage, no prompt
text) are committed under records/ and responses/.

Usage:
  python3 probe.py m1 m2 m3 m4        # phase A: baseline..determinism
  python3 probe.py m5 <body_chars>     # phase B: over-budget enforcement
  python3 probe.py m6 <body_chars>     # conditional: near-budget accept
"""
from pathlib import Path
import json
import os
import sys
import urllib.request
import urllib.error
import datetime
import hashlib
import random
import time

P = Path(__file__).resolve().parent
PROTOCOL = P / "../../../../../../tools/runtime/ai-eval/protocol/registration.json"
ENDPOINT = "https://api.typesafe.ai/v1/systemone"
MODEL = "jev-1.13.0"

PURPOSES = {
    "m1": "baseline: verify live path, model echo, usage decode, output floor",
    "m2": "triage-shaped small state: usage scale for realistic ticket text",
    "m3": "triage-shaped max-caps hostile state: worst-case input in state caps",
    "m4": "byte-identical repeat of m1: output/usage determinism",
    "m5": "over-64k-token state: over-budget enforcement semantics",
    "m6": "near-64k-token state: admission below the published budget",
    "m7": "at-budget state: bracket the enforced cap from below",
    "m8": "mid-budget state: admission below the cap plus rate calibration",
    "m9": "mid-budget state: discriminate the 32k state sub-budget",
}

REG = json.loads(PROTOCOL.read_text())
TRIAGE_Q = {
    "type": "choice",
    "instructions": REG["instructions"],
    "criteria": {c["key"]: c["description"] for c in REG["categories"]},
}

# Deterministic hostile filler: seeded cycle over tokenizer-hostile
# segments (CJK, emoji, symbols, code, mixed alnum, whitespace runs).
SEGMENTS = [
    "Claim TST-1042 opened after duplicate capture on card 4111; ",
    "ledger shows captured/captured 49.00 USD twice in 40 seconds; ",
    "顧客は重複請求の返金を要求しています至急対応が必要です",
    "🚨💳🧾📦🔁 status: pending_review // ticket#αβγ-7788 ",
    "SELECT * FROM charges WHERE status='captured' AND amt=49; -- ",
    "aBcDeF0123456789!@#$%^&*()_+-=[]{}|;:,.<>?~` ",
    "e\u0301\u0302\u0303 combined\u200bzero\u200bwidth\u00a0nbsp\u2003em spaces ",
    "Refund?? refund... REFUND refund\trefund\nrefund\r\nrefund ",
]


def hostile_fill(chars: int, seed: int = 20260927) -> str:
    rng = random.Random(seed)
    order = SEGMENTS[:]
    rng.shuffle(order)
    out = []
    total = 0
    i = 0
    while total < chars:
        seg = order[i % len(order)]
        take = min(len(seg), chars - total)
        out.append(seg[:take])
        total += take
        i += 1
    return "".join(out)


def build(name: str, arg: str | None) -> bytes:
    if name == "m1":
        payload = {
            "model": MODEL,
            "state": "probe baseline state",
            "questions": {"q0": {"type": "noul", "instructions": "Is this a measurement probe?"}},
        }
    elif name == "m2":
        payload = {
            "model": MODEL,
            "state": {
                "subject": "Charged twice for order A-104",
                "body": (
                    "Hello, my card shows two identical charges of $49 for order "
                    "A-104 placed yesterday. Please refund the duplicate charge "
                    "and confirm by email. Thank you."
                ),
            },
            "questions": {"q0": TRIAGE_Q},
        }
    elif name == "m3":
        payload = {
            "model": MODEL,
            "state": {"subject": hostile_fill(200, 11), "body": hostile_fill(2000, 12)},
            "questions": {"q0": TRIAGE_Q},
        }
    elif name == "m4":
        payload = {
            "model": MODEL,
            "state": "probe baseline state",
            "questions": {"q0": {"type": "noul", "instructions": "Is this a measurement probe?"}},
        }
    elif name in ("m5", "m6", "m7", "m8", "m9"):
        if arg is None:
            raise SystemExit(f"{name} needs <body_chars>")
        payload = {
            "model": MODEL,
            "state": {"subject": f"{name} budget probe", "body": hostile_fill(int(arg), 13)},
            "questions": {"q0": TRIAGE_Q},
        }
    else:
        raise SystemExit(f"unknown probe {name}")
    return json.dumps(payload, ensure_ascii=False).encode("utf-8")


def send(name: str, body: bytes) -> dict:
    key = os.environ["TYPESAFE_API_KEY"]
    req = urllib.request.Request(
        ENDPOINT,
        data=body,
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
    )
    started = datetime.datetime.now(datetime.timezone.utc)
    t0 = time.monotonic()
    status = None
    try:
        with urllib.request.urlopen(req, timeout=180) as r:
            raw, status = r.read(), r.status
    except urllib.error.HTTPError as e:
        raw, status = e.read(), e.code
    latency_ms = int((time.monotonic() - t0) * 1000)
    try:
        decoded = json.loads(raw)
    except ValueError:
        decoded = {"_unparsed_bytes": len(raw)}
    (P / "records").mkdir(exist_ok=True)
    (P / "responses").mkdir(exist_ok=True)
    (P / "responses" / f"{name}.json").write_bytes(
        json.dumps(decoded, indent=2, ensure_ascii=False).encode() + b"\n"
    )
    answers = decoded.get("answers", {}) if isinstance(decoded, dict) else {}
    summary = {}
    for qid, ans in (answers.items() if isinstance(answers, dict) else []):
        if isinstance(ans, dict):
            summary[qid] = {k: ans.get(k) for k in ("choice", "confidence", "noul") if k in ans}
    state_len = len(json.dumps(json.loads(body).get("state", ""), ensure_ascii=False))
    record = {
        "name": name,
        "purpose": PURPOSES[name],
        "startedAt": started.isoformat(),
        "endpoint": "v1/systemone",
        "model_request": MODEL,
        "model_echo": decoded.get("model") if isinstance(decoded, dict) else None,
        "status": status,
        "request_sha256": hashlib.sha256(body).hexdigest(),
        "request_bytes": len(body),
        "state_json_chars": state_len,
        "latency_ms": latency_ms,
        "usage": decoded.get("usage") if isinstance(decoded, dict) else None,
        "answers_summary": summary,
    }
    (P / "records" / f"{name}.json").write_text(json.dumps(record, indent=2) + "\n")
    print(json.dumps({"probe": name, "status": status, "usage": record["usage"],
                      "request_bytes": len(body), "latency_ms": latency_ms}))
    return record


def main(argv: list[str]) -> None:
    queue: list[tuple[str, str | None]] = []
    i = 0
    while i < len(argv):
        tok = argv[i]
        if tok in ("m5", "m6", "m7", "m8", "m9"):
            if i + 1 >= len(argv):
                raise SystemExit(f"{tok} needs <body_chars>")
            queue.append((tok, argv[i + 1]))
            i += 2
        elif tok in ("m1", "m2", "m3", "m4"):
            queue.append((tok, None))
            i += 1
        else:
            raise SystemExit(f"unknown probe {tok}")
    m1_sha = None
    for name, arg in queue:
        body = build(name, arg)
        if name == "m1":
            m1_sha = hashlib.sha256(body).hexdigest()
        if name == "m4":
            rec = P / "records" / "m1.json"
            if m1_sha is None and rec.exists():
                m1_sha = json.loads(rec.read_text())["request_sha256"]
            assert m1_sha == hashlib.sha256(body).hexdigest(), "m4 must repeat m1 bytes"
        send(name, body)
        time.sleep(2)


main(sys.argv[1:])

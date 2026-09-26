"""Send three independently worded, already audited design consultations."""
import json
import os
import time
import urllib.request
from pathlib import Path

here = Path(__file__).resolve().parent
key = os.environ["TYPESAFE_API_KEY"]
for number in range(1, 4):
    payload = (here / f"jev-request-{number}.json").read_bytes()
    request = urllib.request.Request(
        "https://api.typesafe.ai/v1/systemone",
        data=payload,
        headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"},
        method="POST",
    )
    started = time.monotonic()
    with urllib.request.urlopen(request, timeout=45) as response:
        raw = response.read()
        metadata = {"status": response.status, "seconds": round(time.monotonic() - started, 3)}
    (here / f"jev-response-{number}.json").write_bytes(raw)
    (here / f"jev-response-{number}.metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
    result = json.loads(raw)
    print(json.dumps({"consultation": number, "model": result.get("model"), "usage": result.get("usage"), "answers": result.get("answers")}), flush=True)

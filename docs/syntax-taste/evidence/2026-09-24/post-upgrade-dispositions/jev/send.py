"""Send the exact saved Jev requests without exposing the API credential."""

from __future__ import annotations

import concurrent.futures
import datetime as dt
import json
import os
from pathlib import Path
import time
import urllib.error
import urllib.request

HERE = Path(__file__).resolve().parent
URL = "https://api.typesafe.ai/v1/systemone"


def send(index: int, key: str) -> tuple[int, int, str]:
    payload = (HERE / f"request-{index}.json").read_bytes()
    request = urllib.request.Request(
        URL,
        data=payload,
        headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        method="POST",
    )
    started = time.monotonic()
    status = 0
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            status = response.status
            body = response.read()
    except urllib.error.HTTPError as exc:
        status = exc.code
        body = exc.read()
    except (OSError, TimeoutError) as exc:
        body = json.dumps({"transport_error": type(exc).__name__}).encode()
    (HERE / f"response-{index}.json").write_bytes(body + (b"" if body.endswith(b"\n") else b"\n"))
    metadata = {
        "request": f"request-{index}.json",
        "response": f"response-{index}.json",
        "status": status,
        "elapsed_seconds": round(time.monotonic() - started, 3),
        "recorded_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
    }
    (HERE / f"response-{index}.metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
    return index, status, "ok" if status == 200 else "error"


if __name__ == "__main__":
    key = os.environ.get("TYPESAFE_API_KEY")
    if not key:
        raise SystemExit("TYPESAFE_API_KEY unavailable")
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        results = list(pool.map(lambda i: send(i, key), (1, 2, 3)))
    print(results)
    if any(status != 200 for _, status, _ in results):
        raise SystemExit(1)

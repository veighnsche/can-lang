"""Submit exactly three pre-audited fresh requests; never print credentials."""
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import urllib.error
import urllib.request

HERE = Path(__file__).resolve().parent
ENDPOINT = 'https://api.typesafe.ai/v1/systemone'
LIMIT = 256 * 1024


def send(index, credential):
    payload = (HERE / f'request-{index}.json').read_bytes()
    request = urllib.request.Request(ENDPOINT, data=payload, headers={'Authorization': 'Bearer ' + credential, 'Content-Type': 'application/json'}, method='POST')
    status = 0
    try:
        with urllib.request.urlopen(request, timeout=40) as response:
            status, body = response.status, response.read(LIMIT + 1)
    except urllib.error.HTTPError as error:
        status, body = error.code, error.read(LIMIT + 1)
    except (OSError, TimeoutError) as error:
        body = json.dumps({'transport_error': type(error).__name__}).encode()
    if len(body) > LIMIT:
        body = json.dumps({'response_exceeded_byte_cap': LIMIT}).encode()
        status = 0
    (HERE / f'response-{index}.json').write_bytes(body + b'\n')
    metadata = dict(status=status, recorded_at_utc=datetime.now(timezone.utc).isoformat(), request_sha256=hashlib.sha256(payload).hexdigest(), fresh_request=True)
    (HERE / f'response-{index}.metadata.json').write_text(json.dumps(metadata, indent=2) + '\n')
    return dict(request=index, status=status)


if __name__ == '__main__':
    audit = json.loads((HERE / 'wording-audit.json').read_text())
    assert audit['semantic_review'].startswith('PASS:')
    for index in (1, 2, 3):
        assert hashlib.sha256((HERE / f'request-{index}.json').read_bytes()).hexdigest() == audit['request_sha256'][str(index)]
        assert not (HERE / f'response-{index}.json').exists(), 'avoid silently overwriting a prior consultation'
    key = os.environ.get('TYPESAFE_API_KEY')
    if not key:
        raise SystemExit('TYPESAFE_API_KEY unavailable')
    with ThreadPoolExecutor(max_workers=3) as pool:
        results = list(pool.map(lambda index: send(index, key), (1, 2, 3)))
    print(json.dumps(results))
    if any(result['status'] != 200 for result in results):
        raise SystemExit(1)

"""LF02 G4 evidence: real LSP sessions against diagnostic projects + astral case.

Writes results ONLY to the scratch out path. Asserts nothing; prints ranges.
"""
import json, pathlib, subprocess, sys, shutil
HERE = pathlib.Path('/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects')
launcher = pathlib.Path(sys.argv[1]).resolve()
work = pathlib.Path(sys.argv[2]).resolve()
out = pathlib.Path(sys.argv[3]).resolve()
work.mkdir(parents=True, exist_ok=True)
for name in ['passing', 'diagnostic-semantic', 'diagnostic-resolve', 'diagnostic-parse']:
    src = HERE / name
    dst = work / name
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst)
# Astral project: failing name after an astral string on the same line.
astral = work / 'diagnostic-astral'
if astral.exists():
    shutil.rmtree(astral)
(astral / 'src').mkdir(parents=True)
(astral / 'can.project.json').write_text('{"source_root":"src","error_registry":"can.errors.json"}')
(astral / 'can.errors.json').write_text('{"active":[],"retired":[]}')
(astral / 'src/main.can').write_text(
    'package app\n    provides [describe]\n    uses []\n\n'
    'fn str describe\n    emits []\n    given\n        str row\n'
    '    asserts\n        sample: "x" => ok "x"\n    ok "\U0001D11E" + missing\n')
def frame(obj):
    b = json.dumps(obj).encode()
    return b'Content-Length: ' + str(len(b)).encode() + b'\r\n\r\n' + b
results = []
for name in ['passing', 'diagnostic-semantic', 'diagnostic-resolve', 'diagnostic-parse', 'diagnostic-astral']:
    source = work / name / 'src/main.can'
    msg = frame({'jsonrpc': '2.0', 'id': 1, 'method': 'initialize',
                 'params': {'rootUri': (work / name).as_uri(), 'capabilities': {}}}) + \
        frame({'jsonrpc': '2.0', 'method': 'textDocument/didOpen',
               'params': {'textDocument': {'uri': source.as_uri(), 'languageId': 'can',
                                           'version': 1, 'text': source.read_text()}}})
    p = subprocess.run([str(launcher), 'lsp', '--stdio'], input=msg,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=15)
    raw = p.stdout.decode(errors='replace')
    messages = []
    while raw:
        header, raw = raw.split('\r\n\r\n', 1)
        n = int(header.split(':', 1)[1])
        messages.append(json.loads(raw[:n]))
        raw = raw[n:]
    diags = []
    for m in messages:
        if m.get('method') == 'textDocument/publishDiagnostics':
            diags = m['params']['diagnostics']
    row = {'project': name, 'exit_code': p.returncode, 'diagnostics': diags,
           'source_lines': source.read_text().splitlines()}
    results.append(row)
    print(name, '->', json.dumps(diags))
out.write_text(json.dumps(results, indent=2) + '\n')
print('wrote ' + str(out))

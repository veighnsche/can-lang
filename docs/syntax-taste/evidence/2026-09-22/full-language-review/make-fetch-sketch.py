from pathlib import Path
ROOT=Path(__file__).resolve().parents[5]
source=(ROOT/'compiler/testdata/current/fetch/main.can').read_text().splitlines()
errors=['http::invalid_request','http::credentials_missing','http::transport_failed','http::timeout','http::body_limit','http::status_error','codec::invalid_data']
lines=source[:168]+['']+source[185:]
old='['+', '.join(errors)+']'
lines=[l.replace(old,'[http::request_failed]') for l in lines]
collapsed=[]
i=0
while i<len(lines):
    if lines[i].strip()==errors[0]:
        indent=lines[i][:-len(lines[i].lstrip())]
        assert [l.strip() for l in lines[i:i+7]]==errors
        assert all(l.startswith(indent) and len(l)-len(l.lstrip())==len(indent) for l in lines[i:i+7])
        collapsed.append(indent+'http::request_failed')
        i+=7
    else:
        collapsed.append(lines[i]);i+=1
lines=collapsed
# Sort only direct children of call matches; keep each child's complete subtree.
indent=lambda s:len(s)-len(s.lstrip())
for start in reversed([i for i,l in enumerate(lines) if l.lstrip().startswith('match call ')]):
    depth=indent(lines[start]);end=start+1
    while end<len(lines) and (not lines[end].strip() or indent(lines[end])>depth):end+=1
    content_end=end
    while content_end>start+1 and not lines[content_end-1].strip():content_end-=1
    trailing=lines[content_end:end]
    starts=[i for i in range(start+1,content_end) if lines[i].strip() and indent(lines[i])==depth+4]
    chunks=[lines[a:b] for a,b in zip(starts,starts[1:]+[content_end])]
    rank=lambda block:0 if block[0].strip()=='when' else 2 if block[0].lstrip().startswith('ok ') else 1
    lines[start+1:end]=[l for chunk in sorted(chunks,key=rank) for l in chunk]+trailing
assert sum(l.strip()=='emits [http::request_failed]' for l in lines)==9
assert sum(l.strip()=='http::request_failed' for l in lines)==8
assert [l.strip() for l in lines if l.lstrip().startswith('match call ')]==[l.strip() for l in source[:168] if l.lstrip().startswith('match call ')]
assert not any(e in '\n'.join(lines) for e in errors)
header=['// DESIGN SKETCH ONLY: requires the proposed default normalization boundary.', '// Mechanically retains eight operations, success checks and nesting.', '// Excludes the separate referenced helper; retains extract used by request expressions.', '// Original runtime-check idiom is deliberately unchanged for this comparison.','']
Path(__file__).with_name('eight-fetch-proposal.can.txt').write_text('\n'.join(header+lines)+'\n')
print('Verified sketch: 8 call matches, 9 normalized bound entries, 8 forward arms; no raw seven identifiers.')

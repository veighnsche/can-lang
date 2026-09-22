from pathlib import Path
import json, re
ROOT = Path(__file__).resolve().parents[5]
ERRORS = ['http::invalid_request','http::credentials_missing','http::transport_failed','http::timeout','http::body_limit','http::status_error','codec::invalid_data']
reports=[]
for app in sorted((ROOT/'examples').iterdir()):
    files=sorted(app.glob('src/**/*.can'))
    if not files: continue
    lines=[line for p in files for line in p.read_text().splitlines()]
    emits=[line.strip() for line in lines if line.strip().startswith('emits [')]
    reports.append({'application':app.name,'can_files':len(files),'source_lines':len(lines),'maximum_line_characters':max(map(len,lines)),'emits_declarations':len(emits),'emits_characters_excluding_indent':sum(map(len,emits)),'seven_infrastructure_error_mentions':sum(line.count(e) for line in lines for e in ERRORS)})
fetch=(ROOT/'compiler/testdata/current/fetch/main.can').read_text().splitlines()
selected=fetch[:168]  # eight fetch declarations plus main; excludes separate reference test
bound=[l for l in selected if l.strip().startswith('emits [') and any(e in l for e in ERRORS)]
actual={'scope':'compiler/testdata/current/fetch/main.can lines 1-168; excludes referenced helper beginning line 169','fetch_declarations':8,'infrastructure_emits_lists':len(bound),'infrastructure_emits_entries':sum(l.count(e) for l in bound for e in ERRORS),'infrastructure_forward_arms':sum(l.strip() in ERRORS for l in selected)}
assert actual['infrastructure_emits_entries']==63 and actual['infrastructure_forward_arms']==56
result={'method':'Source characters and exact error-identifier occurrences; not model token counts or a TypeScript comparison. Existing regression fixtures include deliberately complex expressions.','applications':reports,'eight_fetch_baseline':actual,'eight_fetch_proposal_projection':{'public_error_entries':9,'forwarding_arms':8,'percentage_reduction_in_each_category':round(100*(1-1/7),2),'note':'Analytical projection for one normalized failure per operation, retaining one explicit public bound per declaration. No proposed syntax compiled.'}}
Path(__file__).with_name('measurements.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))

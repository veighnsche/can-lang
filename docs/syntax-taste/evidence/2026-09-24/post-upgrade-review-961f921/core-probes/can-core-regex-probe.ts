import {createText} from '/Users/vince/Projects/can-lang/runtime/text.ts';
const text = createText({} as any, {match:'match'} as any);
for (const flag of ['', 'u', 'v']) {
  const regex = await text.compileRegex('(?:)', flag);
  if (regex.kind !== 'ok') throw new Error('compile');
  const got = await text.findMatches(regex.value, '😀x', 5n);
  console.log(flag || 'none', JSON.stringify(got, (_,v) => typeof v === 'bigint' ? v.toString():v));
  console.log('native matchAll', [...'😀x'.matchAll(new RegExp('(?:)', flag+'g'))].map(x=>x.index));
}

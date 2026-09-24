# Native empty-match advancement

24 September 2026. This read-only probe used the workspace Bun 1.4.2:

```sh
bun -e 'for (const flags of ["g", "gu", "gv"]) { const hits=[..."😀x".matchAll(new RegExp("(?:)",flags))]; console.log(flags, hits.map(x=>[x.index,x.index+x[0].length])); }'
```

Observed `[start,end]` pairs:

| Native flags | Pairs |
| --- | --- |
| `g` | `[[0,0],[1,1],[2,2],[3,3]]` |
| `gu` | `[[0,0],[2,2],[3,3]]` |
| `gv` | `[[0,0],[2,2],[3,3]]` |

The current `runtime/text.ts` `exec` loop increments `lastIndex` by one after
an empty match and returned `[0,0,0,0,0]` under `u` in the earlier saved
review probe. Native capped `matchAll` iteration gives the required Unicode
advance without implementing the regex engine in Can or a helper. This new
probe does not test the proposed adapter, browser engines, result caps or
capture-record projection.

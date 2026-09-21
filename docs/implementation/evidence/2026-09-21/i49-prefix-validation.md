# I49 review correction: source error ID prefixes

Independent review found that the I49 allocation check reparsed source IDs as
decimal, although C2 integer literals and the existing project loader admit
hexadecimal, binary and octal prefixes. The stricter-looking second parse
incorrectly rejected valid declarations after manifest/registry validation.

The registry checker now uses the same base-autodetection and 31-bit range as
`project.Load`. Registry JSON still records the exact numeric allocation; no
registry grammar, range, or nominal identity changed.

The regression loads complete projects, resolves and checks their declarations,
and verifies allocation 1000000 and identical concrete generic error identities
for `1000000`, `0xf4240`, `0b11110100001001000000`, and `0o3641100`.

- [Before correction](i49-prefix-before.txt): decimal passed and all three prefixed spellings failed.
- [Error regression suite](i49-prefix-regression-tests.txt): all spellings, existing negative allocation/bound cases, and generated Bun error plans pass.
- [Full Go suite](i49-prefix-go-tests.txt): passed with pinned emitted execution enabled.
- [Staged offline integration](i49-prefix-offline-integration.txt): passed with source/bundle hash guards and negative runtime cases.

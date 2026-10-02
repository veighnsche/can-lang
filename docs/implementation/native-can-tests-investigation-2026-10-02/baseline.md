# Investigation baseline — 2 October 2026

Investigation only; source implementation and broad builds remain out of scope. Initial Can HEAD: `b1a9a9a8`, branch `main`, clean working tree. The attached planning brief names `manolea-2/tests` as the end goal.

The 48-hour Git window in [activity-summary.json](activity-summary.json) contains 497 commits, not a verified 505. Commit labels and added lines do not establish the claimed implementation-time percentage. Surviving worker-log lifetimes provide a partial independent time proxy, with limitations retained in that file.

Initial verified blockers:

- The 224-file / 292-row source map describes `can-lang/tests`; `manolea-2/tests` contains 25 code files and 3 documents.
- `compiler/main.go:363` refuses live `can test` execution because P23 is unimplemented.
- Migration packages import only `text` or no packages; existing static campaigns cannot yet replace live subject execution.

The full diagnosis, concrete file disposition and proposed budget will follow in this directory. No acceptance rule has been changed.

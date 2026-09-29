# Can user guide for AI coding agents

Status: working guide, started 2026-09-29. The initial material is source-backed
orientation and a coverage plan; verified recipes will be added with their evidence.

This guide helps AI coding agents implement correct Can applications in any project.
It covers the language, toolchain, platform operations, verification, and diagnostics
through small reproducible recipes. Examples use neutral data and explain general
Can techniques. Reading or reproducing the guide requires no particular application,
business model, or sibling repository.

## Start here

1. Read the application's requirements and applicable `AGENTS.md` instructions.
2. Identify the Can toolchain/version or source revision used for the task. Read
   current examples and operation declarations before inventing syntax or APIs.
3. Work on one useful application slice. Define its success behavior, invalid
   inputs, failure mapping, ownership, and assertions before expanding it.
4. Choose a bounded check appropriate to that slice. Inspect actual diagnostics
   and reports; record exactly what was exercised.
5. Capture meaningful friction and the verified correction. Promote transferable
   lessons into this guide and update affected advice when Can changes.

Use the current source and maintained examples to resolve a documentation mismatch.
Design plans and older execution records describe their own context; a proposal or
previous green check does not establish support in the toolchain being used now.

## Current reference entry points

Choose references by the task at hand. Application examples illustrate individual
Can techniques; they are optional topic references rather than a required sequence.

| Task | Starting evidence |
| --- | --- |
| Learn current language patterns | [Gallery examples](../../examples/gallery/README.md) and [current syntax implementation](../../compiler/internal/syntax/README.md) |
| Understand project inputs and dependencies | [Project loader](../../compiler/internal/project/README.md) |
| Identify actual CLI commands and build behavior | [CLI source](../../compiler/main.go) and [build/run reference](../implementation/cli.md) |
| Understand executable contracts and evidence | [Assertions](../implementation/assertions.md) |
| Find callable platform operations | [Current catalogue](../../compiler/internal/catalogue/catalogue.json) and [native declarations](../implementation/native-declarations.md) |
| Build typed HTTP forms and validation | [Form-validation example](../../examples/form-validation/README.md) |
| Combine HTTP, HTML, and relational reads | [Account-search example](../../examples/account-search/README.md) |
| Study authorized reads/writes and browser composition | [Invoice example](../../examples/invoice/README.md) and [invoice grid](../../examples/invoice-grid/README.md) |
| Investigate a missing native capability | [Capability admission guide](../implementation/capability-admission-guide-2026-09-22.md), checked against current source |

These are references to existing material; their verification scope is recorded in
that material. Linking an example does not establish a new check of it here.
Discover supported command flags from the current CLI rather than extrapolating
flags from another tool or historical document.

## Planned recipe coverage

| Topic | What an agent should be able to do | Current guide status |
| --- | --- | --- |
| Bootstrap and iteration | Select a usable toolchain; create a project; inspect, assert, build, and run a small slice; clean owned output | Recipe pending |
| CLI and native platform operations | Handle arguments, files, bytes, processes, declared capabilities, and resource lifecycle | Recipe pending |
| Domain modeling | Use current records, variants, optionals, immutable updates, collections, generics, and callables correctly | Recipe pending |
| Contracts and failures | Author meaningful assertions; distinguish domain outcomes and platform failures; interpret supplied versus real-native evidence | Recipe pending |
| Server pages and forms | Compose safe HTML, typed requests, validation feedback, routes, and assets | Recipe pending |
| Persistence and authorization | Use typed SQL descriptors, transactions, credentials, and access checks; sessions are an optional web example | Recipe pending |
| Browser applications | Keep server capabilities private; build and serve the qualified browser/server pair where needed | Recipe pending |
| Files, bytes, and external integrations | Implement bounded input, storage, delivery, and protocol boundaries using demonstrated capabilities; images are one example | Recipe pending |
| Troubleshooting and maintenance | Map real diagnostics to causes and corrections; keep advice current and verification/storage bounded | Recipe pending |

Choose topics for their value across Can projects. A lesson learned in application
work becomes a general rule, a neutral minimal example, and a reusable check.
Application names, business roles, product policies, and project-specific integration
contracts belong in that application's documentation. Examples and evidence needed
to use this guide live in the Can repository and use portable repository paths.

## Verified platform note: file forms and image responses

For a browser file-upload form, create `html::text_attribute("enctype",
"multipart/form-data")` for a `form` and, if useful, `html::text_attribute("accept",
"image/png,image/jpeg,image/webp")` for its file `input`. `accept` only hints to
the browser; it does not validate an upload. The HTML runtime enforces tag placement,
escapes attribute values, and allows only the multipart value for `enctype`.

After the application has independently bounded and validated image content,
`http::response_image(status, headers, body, media_type)` serves bytes with
`image/png`, `image/jpeg`, or `image/webp` and `nosniff`. Other media types emit
`http::invalid_request`. This response operation does not decode the bytes or
establish ownership. The [HTTP fixture](../../compiler/testdata/current/http/main.can)
shows the exact Can call; the [HTML fixture](../../compiler/testdata/current/html/main.can)
shows the surrounding `html::text_attribute` and element style.

Verification on 2026-09-29, on a working tree based at `a39eea00` with code-diff
SHA-256 `4a15720cd59891289205e07440ede5f36f3a7bb8372d19fdf8334ae378abc74b`:
`bun run check:runtime` passed; `bun test runtime/test/html.test.ts
runtime/test/http-request.test.ts runtime/test/browser-dom.test.ts` passed 60/60;
`go test ./compiler/internal/catalogue/ ./compiler/internal/check/
./compiler/internal/emit/` passed. These checks establish the attribute admission,
typed response behavior, and compiler binding. An end-to-end upload/storage flow
has not been demonstrated by this note.

## Evidence contract for each recipe

A verified recipe contains:

- The task it solves and when to use it.
- Prerequisites, tested Can version/source revision, and any needed services.
- Small complete source, manifest/dependency pieces, and exact commands.
- Expected results, including useful negative cases and diagnostics.
- A linked bounded check and its actual result, identifying supplied fixtures
  separately from native or live integration behavior.
- Known limitations, cleanup, and links to verification material in this repository.

Label a source observation as **source-observed** and an unimplemented recipe as
**planned**. Use **verified** only for the precise behavior actually demonstrated.
Successful compilation, an offline substitution, and a live external operation
establish different facts; retain that distinction in the instructions.

Keep examples free of credentials and private application data. Retain compact
results instead of full bundles or copied execution workspaces. Register cleanup
when a temporary workspace is allocated and report any cleanup failure.

## Learning and updates

Use real implementation experience to identify transferable lessons. Generalize
the technique and reproduce it with a small neutral fixture before adding a verified
recipe. Keep project-specific provenance in that project's learning log; this guide
contains the reusable instruction and self-contained supporting evidence. Avoid
creating work just to populate the guide.

Prefer current Can contracts and their native JavaScript/Bun operations. When a
capability is missing, record the gap and follow its admission/design process;
do not invent a project-authored foreign binding or edit generated TypeScript to
make a recipe appear supported. Update or remove obsolete advice rather than
maintaining old syntax for compatibility.

The guide's acceptance check is a fresh coding agent reproducing a verified small
recipe with the documented prerequisites and checks. That check becomes applicable
when implementation has produced a recipe; it has not been run for this scaffold.

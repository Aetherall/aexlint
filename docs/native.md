# Native typed-rule architecture

## Representation

```text
native/upstream.json              Source revisions and archive SHA-256 hashes
native/rules/<name>/rule.go        Export var Rule = rule.Rule{...}
native/rules/<name>/rule_test.go   Upstream RuleTester cases
native/rules/<name>/rule.json      Name, description, requiresTypeChecking
native/fixtures/tsconfig.json     Real compiler configuration for tests
native/snapshots/                 Reviewed upstream diagnostic snapshots
native/probe/type-probe/          Test-only semantic checker probe
docs/typed-rules/<name>.md        Consumer contract
scripts/native.ts                Source preparation, overlay, tests, cross-builds
src/typed/plugin.ts               Oxlint plugin exposing native rules as aexlint-typed/<name>
src/typed/backend.ts              Backend location, rule manifest, headless frame decoding
src/typed/frames.ts               Request/response framing for the serve FIFOs
```

## Build overlay, not a second rule engine

The pinned `tsgolint` sources remain upstream code. Preparation downloads both source archives, verifies hashes, applies the patches shipped by that exact upstream revision to its matching `typescript-go` sources, and performs the upstream collections-copy step. All generated content stays in `.native/`.

Rule sources and fixtures are copied into `internal/aexlint/` inside that prepared source tree. This preserves access to Go's `internal` packages without pretending they are a stable public library. One generated `zz_aexlint.go` file replaces the executable's registry with our rules. The linter, compiler, type checker, test runner, and wire protocol are not reimplemented.

Retained patches in `native/patches/` are applied in name order and included in the preparation cache fingerprint. Review them on upgrades:

- `require-configured-project.patch` makes headless mode reject files not included in any configured TypeScript project. Upstream otherwise falls back to an inferred project, which could silently ignore the intended tsconfig options.
- `serve.patch` adds a `serve <requests> <responses>` command for the [Oxlint typed plugin](#oxlint-typed-plugin). It runs the unchanged headless entry point once per request frame, so the process-wide parsed-source cache (`internal/utils/host.go`, keyed by file content) survives between requests.
- `whole-programs.patch` adds the payload field `whole_programs`. With it, each program built for a requested file also lints the program's other project source files (not declaration files or external libraries) that headless mode would assign to that same tsconfig, and a message of type 3 lists the files linted per program. It requires exactly one config.

The test build additionally registers `native/probe/type-probe`. `aexlint/type-probe` checks call-result types, including imported and generic functions, to demonstrate genuine semantic analysis. The same call-site text is accepted for an imported string-returning function and reported when only the dependency changes to return a number. It is not a production rule and is never included in release binaries.

Native test snapshots are copied into the upstream tester's snapshot directory before tests. Normal tests reject any newly created or changed snapshot; `--update` explicitly copies reviewed outputs back. Go tests also assert messages, important spans, and fixes independently of snapshots. Snapshot changes are not a substitute for checking the intended behavior.

## Program-wide indexes

A rule runs once per linted file, and several files run in parallel on separate checkers. A measure that spans files, such as how many files check one literal vocabulary, would otherwise repeat a search of the whole program for each linted file.

tsgo keeps no reverse index from a symbol to its references. The binder stores declarations, and the checker caches node-to-symbol resolutions privately for each checker. The language service's find-all-references scans files. So such a rule may share one index across invocations, under these constraints:

- **Keyed by program.** Keys are `weak.Pointer[compiler.Program]`, removed by `runtime.AddCleanup` when the program is collected. Headless mode lints programs one after another; the Go RuleTester creates one program per parallel case. Neither may keep earlier programs alive, and cases must not share entries.
- **Built once, from syntax.** A `sync.Once` guards construction. Construction reads only ASTs and program file metadata, never a checker, because checkers belong to their workers. It may be parallelized internally.
- **Immutable afterwards.** Invocations only read it. ASTs are already read concurrently by a program's parallel checkers.
- **Checker work stays per invocation.** Each invocation confirms candidates from the index with its own `ctx.TypeChecker`.

`native/rules/max-interpretation-spread/index.go` lists every equality comparison and `switch` in project source files, by file and by the literal token values they compare against. The index has not been run under the race detector. Removal of entries after collection is not directly tested.

## Oxlint typed plugin

`aexlint/typed-plugin` is the only consumer entry point for native rules. It is an Oxlint JavaScript plugin named `aexlint-typed` that exposes each native rule as `aexlint-typed/<rule>` and delegates to the backend executable. It exists because Oxlint forwards only its compiled `(tsgolint)` rule stubs to tsgolint, and JavaScript plugins receive no type information (`parserServices` is empty).

- **Batching.** JavaScript rules run per file, synchronously, on one JavaScript thread. In each file, every enabled wrapper rule records its name and options in `create`. The first `Program` visitor sends one `whole_programs` request for the file with all of them. The response covers every file headless mode assigns to that program, so later files of the program are answered from a cache keyed by the rule set. One request is made per TypeScript program.
- **Why a server process.** Oxlint reserves a very large private writable mapping for JavaScript plugins (about 192 GB of address space on the development machine). Under Linux's default heuristic overcommit, `fork()` from that process fails with `ENOMEM`, so plugins cannot spawn processes while linting. At import time the mapping does not exist yet. The plugin therefore creates two FIFOs with `mkfifo` and starts `aexlint-typed serve` at import time, then writes request frames and blocks reading responses with synchronous file I/O. The server writes a `ready` frame once it has opened both FIFOs. Before the first response, the plugin waits for that frame on a non-blocking descriptor, polling every 5 ms; if the server process has exited (checked through `/proc` where it exists) or 10 seconds pass, the plugin kills the server and fails with its captured standard error, instead of blocking forever on a FIFO nobody will write. The server exits when Oxlint closes the request FIFO. Windows has no fork accounting; there the plugin runs `aexlint-typed headless` per request instead (not tested).
- **Results.** Diagnostics are reported with `context.report` on their source range, converted from UTF-8 bytes to JavaScript string offsets. The help is appended to the message as `help: ...`, because Oxlint plugins have no help field. Oxlint applies severities, `oxlint-disable` comments, and output formats. Only rule diagnostics are reported. A backend failure (a file outside every TypeScript project, a program that could not be built, an invalid option that makes a rule panic, a server that did not start or exited) is thrown from the rule; Oxlint reports it as a plugin error, which fails the run whatever the rule's severity. The plugin clears the stack of the plain `Error`s it throws and prefixes their message with `aexlint-typed: `, since Oxlint prints a stack when one is present; other error types keep their stack as evidence of a plugin bug. For a server that stopped, the message includes the backend's panic or `fatal error:` line, read from its captured standard error. Rules run in tsgolint's worker goroutines, where a panic cannot be recovered, so an invalid rule option stops the server; the plugin reports that once and fails every later file of the run with a short "stopped earlier in this run" error instead of restarting a server that would stop again.
- **TypeScript's own errors.** Following Oxlint's and typescript-eslint's convention, the plugin requests no syntactic or semantic compiler diagnostics and drops the backend's internal diagnostics. By default tsgolint builds no program at all when a tsconfig has an option error (such as an option TypeScript 7 no longer recognizes) or a program-level diagnostic (`internal/utils/create_program.go`), and every file of that program goes unlinted without a rule diagnostic. `tsc` reports these errors and still type-checks, so the plugin runs the backend with `OXLINT_TSGOLINT_DANGEROUSLY_SUPPRESS_PROGRAM_DIAGNOSTICS=true`, which skips exactly those two checks: the program is built and linted, and the errors are left to `tsc`. If a program is still not built, the plugin fails with the backend's reasons rather than reporting the file as clean; no tsconfig tried reaches that path (unknown options, invalid `target`, missing `extends`, malformed JSON all build a program; an unreadable tsconfig counts as no project). A circular `extends` deadlocks tsgolint's tsconfig lookup, which the plugin reports as a server that exited. Typed rules stay silent where TypeScript cannot resolve a type, so they assume a project that type-checks, which the project's own `tsc` run establishes.
- **Backend location.** The plugin runs `dist/native/<platform>-<arch>/aexlint-typed` next to its own module. `AEXLINT_TYPED_BACKEND` overrides the path; self-lint uses it to load the plugin source with the freshly built backend, and the tests use it to run the probe build.
- **Freshness.** A cached result is used only when the file's text equals its content on disk. Otherwise a new request sends the file's text as a source override. Other files' cached results are not invalidated, which matters for rules that count other files.

Not provided or not verified: fixes, editor integration (the freshness rule above is untested there, and a plugin loaded after the mapping exists cannot start its server), Windows and macOS execution, and more than one Oxlint workspace. Oxlint's experimental parallel JavaScript execution (oxc #26439) would start one server per JavaScript thread.

## Protocol

The plugin sends tsgolint headless V2 payloads, using the upstream framing for responses: a four-byte little-endian payload length, one-byte message type, then UTF-8 JSON. It distinguishes backend errors from diagnostics and validates frames rather than interpreting arbitrary output as success. `serve` wraps each request and each response in an additional four-byte length frame.

Rule options are passed unchanged to the Go implementation and must be validated there. Go fixes and suggestions can be developed and verified through the upstream rule tester, but the plugin does not forward them.

The JS and native registries are separate runtime representations. Catalog checks reject duplicate names across them so a rule cannot silently switch execution paths. Native rules reach Oxlint as JavaScript plugin rules with their own `aexlint-typed/` prefix, not as typed rules registered with Oxlint's tsgolint integration.

## Reproducibility and upgrades

`native/upstream.json` pins both source revisions and archive SHA-256 hashes. The pinned upstream Go module/workspace files and checksums determine dependency versions; Go verifies module downloads. `devenv.lock` fixes the development toolchain inputs. Preparation does not follow moving branches, install global tools, or make git commits.

An upgrade is an explicit compatibility change:

1. Select a matching upstream tsgolint revision and its recorded typescript-go revision.
2. Download and verify those exact source archives; update their hashes together.
3. Review upstream patches, Go toolchain requirements, compiler API changes, headless protocol changes, and licenses.
4. Rebuild from a fresh `.native/` cache, run native tests and both consumer integration tests, then cross-compile the release.
5. Review any snapshot differences against the rule contracts. Do not overwrite snapshots just to make an upgrade pass.

We depend on internal Go/compiler APIs, not a stable extension API. Pinning bounds that risk; it does not eliminate maintenance work.

## Release validation

`pnpm build` emits only the host executable for iteration. `pnpm pack` runs `build:release`, which emits Linux/macOS/Windows x64 and arm64 executables into one tarball. A release must use that path, not `pnpm pack --ignore-scripts` on a host-only development build.

Package integration tests deliberately pack the already-built host artifact without rerunning release hooks, install it offline into an isolated pnpm consumer, check both TypeScript exports, and run Oxlint with both the `aexlint` plugin and `aexlint/typed-plugin`, which starts the installed native backend. Go probe tests validate custom semantic rule execution separately. A successful cross-build is not evidence that a non-host executable ran successfully; CI covers that by installing the release tarball on a runner for each of the six targets and running `pnpm package:smoke` there.

The package test first resolves the tarball's dependencies into a temporary consumer lockfile with `pnpm add --lockfile-only`, then populates that consumer's store with `pnpm fetch`. These preparation steps may access the registry. It then installs with `--offline --frozen-lockfile`. This avoids relying on registry metadata or store entries left by earlier repository installs. The repository lockfile is not modified.

Native notices include the upstream source pins and discovered module license/notice files, plus Go's runtime license. Review them before each release. Publishing is automated by the tag-triggered release workflow described in [development](development.md#releasing).

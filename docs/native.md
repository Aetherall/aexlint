# Native backend

Oxlint's JS API has no type checker, and its built-in typed backend accepts only built-in rules. The `@aetherall/aexlint` plugin bridges our Go rules to ordinary Oxlint plugin diagnostics, next to the syntax rules.

## Source layout

| Path                   | Purpose                                                    |
| ---------------------- | ---------------------------------------------------------- |
| `native/upstream.json` | Pinned tsgolint/typescript-go revisions and archive hashes |
| `rules/<name>/`        | Go rule (`rule.go`), `rule.json` metadata, and its doc     |
| `native/model/`        | Shared decision model used by rules                        |
| `native/patches/`      | Changes to the pinned upstream backend                     |
| `native/snapshots/`    | Reviewed probe snapshots                                   |
| `native/probe/`        | Test-only checker probe; never shipped                     |
| `scripts/native.ts`    | Prepare sources, overlay rules, test, and build            |
| `src/typed/`           | Oxlint wrappers, backend startup, and framing              |

Preparation verifies source archives and applies upstream patches plus our patches. Rules are copied into the upstream tree under `.native/`, where they can use Go's internal compiler APIs. A generated registry selects our rules. Do not edit this generated tree.

Our patches skip files outside every configured TypeScript project instead of building an inferred one, add a persistent server, and support linting whole programs. Review them when upgrading; internal compiler APIs are not stable.

## Type-aware rules in the plugin

- Each file's wrappers collect enabled rules and options. The first visitor requests whole-program results; later files reuse them for the same rule set.
- On Unix, the plugin starts a backend at import time and communicates through FIFOs, even when only syntax rules are enabled. Starting it later can fail because Oxlint's large memory mapping prevents `fork()`. Startup is checked with a timeout. A backend that exits or stops reading later fails the run; a backend still working is never timed out. On Windows, each request runs the headless backend directly.
- Typed rule names come from the backend's `rules.json`, or from `src/rules.ts` when the backend is missing. A missing or failed backend only fails the run when a typed rule is enabled.
- Responses use upstream headless V2 framing. The server adds a length frame around each request and response. Invalid frames and backend failures fail the lint run.
- The upstream backend drops a diagnostic without notice when its message is not valid UTF-8. Compiler-internal symbol names start with `ast.InternalSymbolNamePrefix` (`\xFE`), so never put them in messages.
- Diagnostics convert UTF-8 byte ranges to JS offsets. Oxlint handles severities and disable comments. Fixes are not forwarded.
- A file no `tsconfig.json` project includes gets no typed diagnostics: the backend names it as unmatched instead of linting it with default compiler options.
- TypeScript diagnostics are left to `tsc`. Program-diagnostic suppression lets the backend check projects with config errors; unresolved types may still make rules skip cases.
- Invalid options can stop the backend. The first error gives the reason; later files fail rather than silently pass.
- Results are reused only when the current file matches disk. A changed file is sent as an override, but other cached files are not invalidated. Editor integration and multiple Oxlint workspaces are unsupported or unverified.

The executable lives at `dist/native/<platform>-<arch>/aexlint-typed` (`.exe` on Windows). `AEXLINT_TYPED_BACKEND` overrides it for self-lint and tests.

## Shared decision model

`native/model/` is a Go package, not a rule. It is copied next to the rules and imported as `internal/aexlint/model`. It holds the analyzer that reads tests of finite literal values and of presence, syntax helpers, the per-program index of decision sites, ownership resolution, result construction, and the per-file decision model described in [decision model](design/decision-model.md). Rules build on it instead of copying its logic: a rule is a query over the model's facts plus its wording. Rules do not call the checker or import its package, except `TypeToString` to name a type in a message; the catalog test enforces this. When a rule needs a fact about types, add it to the model with a `model-facts` fixture.

The test-only `native/probe/model-facts` rule reports every kind of model fact as a diagnostic: sites, tests, decisions with their arms and outcomes, functions, gates, guards, entry tests, boundaries, flows, the rejections of flows and type-predicate contracts. Its `facts` option selects kinds, so each fixture asserts one kind exactly while its snapshot pins the content. A fact enters the model together with its fixture, before a rule relies on it. Run `pnpm native:test model-facts`.

To review a model change on real code, dump the facts of whole programs before and after it and diff them:

```sh
node scripts/scan.ts facts .native/probe/<platform>-<arch>/aexlint-typed <project root> <seed file>... > facts.tsv
```

`pnpm native:test` builds that probe binary; keep a copy from the base commit to compare with. `rules` instead of `facts` runs the shipped rules with a binary from `pnpm native:build`. Each line holds the seed, rule, file, byte range, message ID and text, sorted. Dumps are deterministic, so every difference is a changed fact to review against the [decision model](design/decision-model.md).

## Shared program indexes

Cross-file rules may share an index when it is:

- Keyed by a weak pointer to the compiler program and removed on collection.
- Built once under synchronization from syntax and file metadata, never a checker.
- Immutable after construction, with checker queries kept in each invocation.

`model.Shared` gives an index this lifecycle, and `model.Collect` walks a program's project files in parallel to build one; see `native/model/index.go`. Race-detector coverage and cleanup-after-collection behavior have not been verified.

## Tests and upgrades

Rules are tested through `tests/project/` (see [development](development.md#adding-a-rule)). `pnpm native:test` runs the test-only probes (`model-facts`, `type-probe`); add `--update` only for intentional snapshot changes. Native ranges are byte offsets.

To upgrade:

1. Update both source revisions and hashes together, using tsgolint's matching typescript-go revision.
2. Review patches, Go requirements, compiler APIs, protocol, and licenses.
3. Rebuild from a fresh `.native/` cache; run native tests, package integration, and release builds.
4. Review `tests/project/expected.txt` and probe snapshot changes against the rule contracts, not just test output.

`pnpm build` produces the host binary. `pnpm pack` builds all six targets with `CGO_ENABLED=0` and includes source pins and third-party notices. CI smoke-tests each target; full rule tests run on Linux x64. See [development.md](development.md#releasing) for publication. `node scripts/native.ts release <target>...` builds only the given targets. CI builds each target in its own job with a cached Go build, then packs those binaries with lifecycle scripts disabled instead of rebuilding them.

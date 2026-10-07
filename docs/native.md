# Native backend

Oxlint's JS API has no type checker, and its built-in typed backend accepts only built-in rules. `aexlint/typed-plugin` bridges our Go rules to ordinary Oxlint plugin diagnostics.

## Source layout

| Path                   | Purpose                                                    |
| ---------------------- | ---------------------------------------------------------- |
| `native/upstream.json` | Pinned tsgolint/typescript-go revisions and archive hashes |
| `native/rules/<name>/` | Go rule, tests, and `rule.json` metadata                   |
| `native/patches/`      | Changes to the pinned upstream backend                     |
| `native/snapshots/`    | Reviewed diagnostic snapshots                              |
| `native/probe/`        | Test-only checker probe; never shipped                     |
| `scripts/native.ts`    | Prepare sources, overlay rules, test, and build            |
| `src/typed/`           | Oxlint wrappers, backend startup, and framing              |

Preparation verifies source archives and applies upstream patches plus our patches. Rules are copied into the upstream tree under `.native/`, where they can use Go's internal compiler APIs. A generated registry selects our rules. Do not edit this generated tree.

Our patches require a configured TypeScript project, add a persistent server, and support linting whole programs. Review them when upgrading; internal compiler APIs are not stable.

## Oxlint typed plugin

- Each file's wrappers collect enabled rules and options. The first visitor requests whole-program results; later files reuse them for the same rule set.
- On Unix, the plugin starts a backend at import time and communicates through FIFOs. Starting it later can fail because Oxlint's large memory mapping prevents `fork()`. Startup is checked with a timeout. On Windows, each request runs the headless backend directly.
- Responses use upstream headless V2 framing. The server adds a length frame around each request and response. Invalid frames and backend failures fail the lint run.
- Diagnostics convert UTF-8 byte ranges to JS offsets. Oxlint handles severities and disable comments. Fixes are not forwarded.
- TypeScript diagnostics are left to `tsc`. Program-diagnostic suppression lets the backend check projects with config errors; unresolved types may still make rules skip cases.
- Invalid options can stop the backend. The first error gives the reason; later files fail rather than silently pass.
- Results are reused only when the current file matches disk. A changed file is sent as an override, but other cached files are not invalidated. Editor integration and multiple Oxlint workspaces are unsupported or unverified.

The executable lives at `dist/native/<platform>-<arch>/aexlint-typed` (`.exe` on Windows). `AEXLINT_TYPED_BACKEND` overrides it for self-lint and tests.

## Shared program indexes

Cross-file rules may share an index when it is:

- Keyed by a weak pointer to the compiler program and removed on collection.
- Built once under synchronization from syntax and file metadata, never a checker.
- Immutable after construction, with checker queries kept in each invocation.

See `native/rules/max-interpretation-spread/index.go`. Race-detector coverage and cleanup-after-collection behavior have not been verified.

## Tests and upgrades

Use `pnpm native:test <name>`; add `--update` only for intentional snapshot changes. Normal tests reject changed or missing snapshots. JS tests use zero-based columns; Go tests use one-based columns; native ranges are byte offsets. The test-only probe checks imported and generic return types using real programs.

To upgrade:

1. Update both source revisions and hashes together, using tsgolint's matching typescript-go revision.
2. Review patches, Go requirements, compiler APIs, protocol, and licenses.
3. Rebuild from a fresh `.native/` cache; run native tests, package integration, and release builds.
4. Review snapshot changes against the rule contracts, not just test output.

`pnpm build` produces the host binary. `pnpm pack` builds all six targets with `CGO_ENABLED=0` and includes source pins and third-party notices. CI smoke-tests each target; full rule tests run on Linux x64. See [development.md](development.md#releasing) for publication.

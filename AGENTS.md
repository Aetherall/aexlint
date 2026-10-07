# Working on aexlint

## Setup and scope

- Use `devenv shell -- pnpm <command>`, not npm or yarn. Install with `pnpm install --frozen-lockfile`; preserve lockfiles.
- Read README, [development](docs/development.md), the target rule's contract/tests/implementation, and deeper guidance before editing. Read [native](docs/native.md) before Go changes.
- No inline comments, mocks, new dependencies, unrelated refactors, commits, or publication without authorization. Preserve unrelated work.
- Do not edit generated `dist/`, `.native/`, registries, or cached upstream sources. Generate rule skeletons serially; parallel workers own separate rule directories. One owner handles shared files and native builds.

## Rule design

- Define one general principle and the maintenance burden it detects. Check existing Oxlint rules first. Do not accumulate domain-specific patterns, naming guesses, or special cases under a broad rule name.
- Types and control flow provide evidence, not proof of generality. Review useful findings and legitimate counterexamples across real code before expanding or promoting experiments. Record limitations and skipped cases.
- Explain the problem, not a demand to extract helpers. Local results, collections, and loops can already be clear. Stop expanding a rule when special cases replace its unifying principle.
- Keep correctness, usefulness, and generalization evidence separate. Tests and lower warning counts do not prove readability gains.

## Implementation

- Syntax/scope rules belong in `src/rules/<name>/` using `defineRule` and the real Oxlint RuleTester (`tests/rule-tester.ts`). Include type, description, `docs.requiresTypeChecking: false`, explicit schema (`[]` for no options), and message IDs.
- Type-aware rules belong in `native/rules/<name>/`, using the real Go program/checker. Export `var Rule = rule.Rule{...}` named `aexlint/<directory-name>` and retain `rule.json` metadata. Validate options in Go.
- Oxlint's JS API has no types. Our Go rules run through `@aetherall/aexlint/typed-plugin`, not `--type-aware` or `OXLINT_TSGOLINT_PATH`. Never substitute naming/text heuristics for types or introduce ESLint/a frontend fork without a new decision.
- Keep state local to each invocation/file. Cross-file indexes may be shared only when keyed by program, built once under synchronization from syntax alone, immutable, and released on collection. Checker work stays per invocation; see [native](docs/native.md#shared-program-indexes).
- Use real ASTs, checkers, compiler programs, and test runners. No fabricated contexts or nodes.
- Upgrade native pins together with the matching compiler revision; review patches and compatibility. Permanent changes belong in source or retained patches, not the generated overlay.

## Authoring and validation

1. Generate with `pnpm rule:new <name>` or `pnpm rule:new:typed <name>`. Names must be unique across runtimes. Replace the intentionally failing placeholders.
2. Write a concise contract in `docs/rules/` or `docs/typed-rules/`: intent, reported/accepted cases, diagnostic spans, options/defaults, fixes, and limitations.
3. Add valid/invalid tests before implementation. Cover relevant JS/TS/TSX syntax, scopes, shadowing, options, and repeated-file behavior. Typed rules need relevant imports, aliases, generics, unions, any/unknown/never, unresolved types, and a dependency-only type-change case.
4. Implement the smallest correct visitor. For fixes, prove safety, preserve comments, assert exact output, and check repeat linting. The typed plugin does not forward fixes, though Go tests can verify them.
5. Run focused tests, `pnpm format`, then `pnpm check`. Update affected docs and report actual results, skips, and unverified behavior.

Focused commands: `pnpm test:rule src/rules/<name>/<name>.test.ts` (accepts `--watch`) and `pnpm native:test <name>`. JS tests need no build. Native preparation initially needs network access. Use `--update` explicitly for Go snapshots and review changes; ordinary tests reject changes. JS columns are zero-based, Go columns one-based, and native ranges are byte offsets.

`pnpm check` covers types, lint, formatting, Go/TS tests, and package integration. Catalog checks reject missing docs/tests/registration and unfinished placeholders; they do not prove semantics. Probes test infrastructure, not product rules.

`pnpm lint` rebuilds the host backend and runs both plugins on source, scripts, and tests; both passes run even on failure. Limits are workload 4, input depth 3, decision depth 2. `pnpm lint:classifications` is optional experimental review, not mandatory self-lint. `max-interpretation-spread` is also experimental and not enabled in self-lint. No recommended preset exists.

Never run native builds/tests/self-lint concurrently in one working tree. `pnpm build` produces the host binary; `pnpm pack` builds six targets without publishing. Cross-compilation alone does not prove runtime behavior. Package tests resolve/fetch dependencies in a temporary consumer before an offline install; CI executes release smoke tests on each platform.

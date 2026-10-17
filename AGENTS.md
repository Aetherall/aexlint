# Working on aexlint

## Setup and scope

- Use `devenv shell -- pnpm <command>`. Install with `pnpm install --frozen-lockfile`; preserve lockfiles.
- Read README, [development](docs/development.md), and the target rule's doc, tests and implementation before editing. Read [native](docs/native.md) before Go changes.
- No inline comments, mocks, new dependencies, unrelated refactors, commits, or publication without authorization.
- Do not edit generated `dist/`, `.native/`, registries, or cached upstream sources.
- Never run native builds, tests or self-lint concurrently in one working tree.

## Rules

- One general principle per rule, explaining the maintenance burden it finds. Check existing Oxlint rules first. Stop expanding a rule when special cases replace its principle.
- Use types and control flow, never names or text as a substitute for types.
- Syntax rules live in `src/rules/<name>/` (`defineRule`, real Oxlint RuleTester). Type-aware rules live in `native/rules/<name>/` (real Go program and checker, `var Rule = rule.Rule{...}` named `aexlint/<name>`, `rule.json`, options validated in Go). Type reasoning belongs in `native/model/`; rules query it.
- Our Go rules run through `@aetherall/aexlint/typed-plugin`, not `--type-aware`.
- Keep state per invocation. Cross-file indexes follow [native](docs/native.md#shared-program-indexes).

## Adding or changing a rule

1. Generate with `pnpm rule:new <name>` or `pnpm rule:new:typed <name>`, one at a time.
2. Write a short doc in `docs/rules/` or `docs/typed-rules/`: problem, reported and accepted examples, options, span, limitations.
3. Write valid/invalid tests first. Typed rules cover imports, aliases, generics, unions, any/unknown/never, unresolved types, and a dependency-only type change.
4. Implement the smallest correct visitor.
5. Run `pnpm test:rule src/rules/<name>/<name>.test.ts` or `pnpm native:test <name>` (`--update` for snapshots, reviewed), then `pnpm format` and `pnpm check`. Report actual results.

JS columns are zero-based, Go columns one-based, native ranges byte offsets.

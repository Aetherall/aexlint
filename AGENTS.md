# Working on aexlint

## Environment

- Use `devenv shell` and pnpm, not npm or yarn. Noninteractive commands: `devenv shell -- pnpm <command>`.
- Install with `pnpm install --frozen-lockfile`. Preserve the pnpm and devenv lockfiles.
- Read `README.md`, `docs/development.md`, the target rule's docs, implementation/tests, and any deeper guidance before editing.
- Both rule paths ship in one npm package and run under Oxlint: the `aexlint` JS plugin, and the `aexlint/typed-plugin` JS plugin that delegates Go rules to the bundled `aexlint-typed` backend.
- Production rules include `max-expression-depth`, `max-expression-complexity`, and `max-decision-depth` (JS), and `prefer-truthy-presence-check` (Go). No recommended presets exist yet. Infrastructure probes are tests, not product requirements.
- `max-interpretation-spread` is an experimental Go rule; it is not enabled in self-lint.
- `prefer-domain-predicate` is an experimental JS diagnostic. `pnpm lint:classifications` evaluates it separately; it is not a mandatory self-lint rule. Assess real findings before promoting it or refactoring callers.

## Rule design requirements

- Aim for generalized rules, not catalogues of specific case detectors. A candidate must express one coherent structural or semantic principle that applies across varied code without accumulating domain-specific patterns, naming guesses, or ad hoc exceptions.
- State that principle and the concrete maintenance or comprehension burden before choosing detection machinery. A general smell is a research question until there is a broadly applicable, actionable detection principle.
- More sophisticated analysis does not establish generality. Types, control-flow graphs, and shared analysis engines provide evidence; they do not turn separately hand-modeled examples into a general rule.
- Assess useful findings and legitimate counterexamples across varied real code before substantially expanding an experimental detector or promoting it. Record coverage limits and reasons analysis declines cases, not just diagnostic counts.
- Diagnostics should explain the burden, not prescribe helper extraction. Meaningful local results, named collections, contiguous stages, and straightforward loops can already be clear. A distinct phase or one-off computation does not automatically deserve a method; extraction must earn its indirection.
- Keep correctness evidence separate from usefulness and generalization evidence. Passing bounded tests, preserving behavior on a corpus, or reducing warnings does not prove broader correctness or readability gains.
- Stop expanding a candidate when it depends on an increasing catalogue of special relationships or exceptions without a unifying detection principle. Retain useful experiments as research rather than treating implementation effort as a reason to ship them.

## Choose the actual runtime

- Syntax, AST, lexical scope: use `src/rules/<name>/`, Oxlint's JS API, and `tests/rule-tester.ts`.
- Actual semantic types: use `native/rules/<name>/` and the real Go `RuleContext.Program` / `TypeChecker`. Read `docs/native.md` before native changes.
- Oxlint does not expose checker types to JS plugins. Stock Oxlint also rejects unknown typed rule names before invoking tsgolint. Our Go rules therefore run through `aexlint/typed-plugin`, a JS plugin that sends each TypeScript program to the bundled backend (`docs/native.md`). Do not claim that `--type-aware` or `OXLINT_TSGOLINT_PATH` enables them, or that the plugin gives other JS rules type information.
- Never emulate type awareness with identifier naming or source-text heuristics. Do not add an ESLint runtime or fork Oxlint's frontend without a new explicit decision.

## Authoring loop

1. Establish that the candidate meets the rule design requirements above, then check whether an existing built-in rule already covers it. Define the intended difference and false-positive boundaries.
2. Generate a skeleton: `pnpm rule:new <name>` for JS, or `pnpm rule:new:typed <name>` for Go. Both intentionally fail until the examples and implementation are replaced. Names must be unique across runtimes.
3. Write the contract in `docs/rules/<name>.md` (JS) or `docs/typed-rules/<name>.md` (Go): intent, reported/accepted cases, precise spans, options/defaults, fixes, and limitations.
4. Write meaningful valid/invalid cases before implementation. Include applicable JS/TS/TSX syntax, nesting, scopes/shadowing, options, and repeated-file behavior.
5. For typed rules, test imports, aliases, generic instantiation, unions, any/unknown/never, and unresolved types as relevant. Include a cross-file case where only the dependency's type changes the result. The native probe is an executable example.
6. Implement the smallest correct visitor. Keep state local to each rule invocation/file; avoid shared mutable state and speculative frameworks. One exception exists for typed rules whose measure spans files: a program-wide index may be shared across invocations when it is keyed by the `*compiler.Program`, built once under synchronization from syntax alone (no checker use), immutable after construction, and released when the program is collected. `docs/native.md` describes the pattern. Checker queries stay in each invocation on its own checker.
7. For fixes, prove safe behavior, preserve comments, assert exact output, and check repeat linting. Do not add unsafe fixes just to complete a feature. The typed plugin currently does not forward fixes, but Go RuleTester verifies them.
8. Run the focused loop while editing, then format and run the complete checks. Update docs whenever the contract changes.

## Commands and evidence

- JS loop: `pnpm test:rule src/rules/<name>/<name>.test.ts`. No rebuild needed; add `--watch` if useful.
- Native loop: `pnpm native:test <name>`. Sources are overlaid into the pinned upstream build automatically. Initial preparation needs network access; subsequent runs reuse caches.
- Go snapshots: `pnpm native:test <name> --update` explicitly updates retained snapshots. Review them. Normal tests reject new/changed snapshots rather than silently accepting them.
- JS RuleTester uses zero-based columns and one-based lines. Go RuleTester assertions use one-based lines and columns. Native JSON ranges are byte offsets, not ESTree offsets.
- `pnpm format` runs Oxfmt and gofmt. `pnpm check` checks types, lint, formatting, Go tests, TS tests, and packed-package integration. Report actual results and skipped/unverified behavior.
- `pnpm lint` runs the source JS plugin and native typed rules on `src`, `scripts`, and `tests`. Repository limits are workload 4, input depth 3, and decision depth 2. It rebuilds the host native backend; do not run it concurrently with other native builds/tests. Both lint passes run even when the first finds violations.
- `pnpm build` creates JS/declarations and the host native binary. `pnpm pack` builds all six native targets into one tarball without publishing. Cross-compilation does not establish runtime correctness on other platforms.

## Implementation boundaries

- Use real Oxlint and tsgolint test runners and compiler programs. Mocks, fabricated ASTs, and fabricated checker/context objects are forbidden.
- JS metadata: type, description, `docs.requiresTypeChecking: false`, explicit options schema (`[]` if none), message IDs. Use `defineRule` and usually `create(context)` with per-file state.
- Go rules export `var Rule = rule.Rule{...}` with `Name: "aexlint/<directory-name>"`. Retain `rule.json` with name, description, and `requiresTypeChecking: true`. Validate options in Go. Use the upstream checker and utility APIs directly.
- JS registration is generated in `src/rules/index.ts`; avoid hand-editing except when an authorized removal/rename requires it. Native registration is generated at build time from the rule directories.
- Catalog checks reject missing registration/docs/tests and unfinished TODO placeholders. They do not prove semantic correctness.
- Keep native source pins synchronized with the matching compiler revision. Do not edit cached upstream code or generated shims as the permanent implementation. Upgrade through `native/upstream.json` and review compatibility as described in `docs/native.md`.
- Do not edit generated `dist/`, `.native/`, or lockfiles manually. Build/test scripts own generated overlays and serialize their use; do not run native builds/tests concurrently in the same working tree.
- For parallel agents, generate skeletons serially first, then assign separate rule directories and docs. One owner controls shared helpers, native builds, registries, and dependencies.
- Package tests install the actual tarball offline using the populated pnpm store. Probes validate infrastructure and semantic access, not correctness of future production rules.
- Do not add inline comments, unrelated refactors, commits, publishing, deployment, or new dependencies without task authorization. Put rule rationale in documentation.

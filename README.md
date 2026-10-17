# aexlint

Readability rules for [Oxlint](https://oxc.rs/docs/guide/usage/linter): syntax rules that measure how much a reader must hold at once, and type-aware rules that find knowledge about values repeated, re-checked, or held outside its owner. No autofix.

## Standalone CLI

Requires Node.js 24+.

```sh
pnpx @aetherall/aexlint check
pnpx @aetherall/aexlint check src --typed --format json
```

`check` defaults to the current directory and runs every syntax rule as an error. `--typed` adds every type-aware rule; the project needs its dependencies installed and a TypeScript project (see below). Default limits:

| Rule                        | Limit                     |
| --------------------------- | ------------------------- |
| `max-expression-depth`      | 3                         |
| `max-expression-complexity` | 4                         |
| `max-decision-depth`        | 2                         |
| `max-held-context`          | 3                         |
| `max-call-assembly`         | width 60, 3 outside names |
| `max-interpretation-spread` | 4                         |
| `max-projection-spread`     | 2                         |

The limits are starting points, not validated thresholds. For other limits or a subset of rules, use the plugin.

The CLI ignores Oxlint configurations and enables no built-in checks. Oxlint's file exclusions, ignore files and disable comments apply. Add exclusions with repeatable `--ignore-pattern <glob>` or in `.aexlintignore` (one glob per line, `#` comments). `--format`/`-f` accepts Oxlint output formats. Findings fail the command. Use `--` before paths starting with a dash.

### Reporting only new findings

Record the findings of the base revision, then check the branch against them:

```sh
aexlint="$PWD/node_modules/.bin/aexlint"
git worktree add ../base "$(git merge-base origin/master HEAD)"
(cd ../base && pnpm install --frozen-lockfile && "$aexlint" check --write-baseline /tmp/aexlint-baseline)
"$aexlint" check --baseline /tmp/aexlint-baseline
```

Use the same version, flags and paths for both runs, from the root of each checkout. A finding is its file, rule and message: it survives moved lines, but comes back when its message changes, when its file is renamed, or when it is caused by a change elsewhere. Identical findings in a file are counted. Parse errors are never skipped. Baselines apply only to the CLI.

## Plugin setup

Requires Node.js 24+ and Oxlint 1.83.0.

```sh
pnpm add -D oxlint@1.83.0 @aetherall/aexlint
```

`.oxlintrc.json`:

```json
{
  "jsPlugins": ["@aetherall/aexlint"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }],
    "aexlint/no-repeated-value-policy": "warn"
  }
}
```

The plugin enables no rules by default. Type-aware rules need the TypeScript setup below; syntax rules need nothing more.

## Rules

### Syntax

| Rule                                                                   | Reports                                           |
| ---------------------------------------------------------------------- | ------------------------------------------------- |
| [max-expression-depth](rules/max-expression-depth/README.md)           | Nested calls                                      |
| [max-expression-complexity](rules/max-expression-complexity/README.md) | Work inside one expression                        |
| [max-decision-depth](rules/max-decision-depth/README.md)               | Nested logical groups and ternaries               |
| [max-call-assembly](rules/max-call-assembly/README.md)                 | Wide calls drawing on many outside names          |
| [max-held-context](rules/max-held-context/README.md)                   | Enclosing decisions, loops, and callbacks         |
| [no-else-if](rules/no-else-if/README.md)                               | `else if` chains                                  |
| [no-multiline-condition](rules/no-multiline-condition/README.md)       | Conditions spanning multiple lines                |
| [prefer-domain-predicate](rules/prefer-domain-predicate/README.md)     | Unnamed classifications built from several checks |

### Type-aware

| Rule                                                                           | Reports                                                                |
| ------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| [prefer-truthy-presence-check](rules/prefer-truthy-presence-check/README.md)   | Nullish comparisons equivalent to truthiness checks                    |
| [max-interpretation-spread](rules/max-interpretation-spread/README.md)         | Literal unions interpreted across too many files                       |
| [max-projection-spread](rules/max-projection-spread/README.md)                 | Repeated object projections across files                               |
| [no-forgeable-validated-result](rules/no-forgeable-validated-result/README.md) | Validated results callers can construct without validation             |
| [no-caller-enforced-invariant](rules/no-caller-enforced-invariant/README.md)   | Caller-side checks that a receiving class does not enforce             |
| [no-repeated-value-group](rules/no-repeated-value-group/README.md)             | Groups of literal values written out in several places                 |
| [no-repeated-value-policy](rules/no-repeated-value-policy/README.md)           | The same literal test and outcome in several files                     |
| [no-procedural-value-dispatch](rules/no-procedural-value-dispatch/README.md)   | Branches on literal values that contain their own control flow         |
| [no-external-variant-behavior](rules/no-external-variant-behavior/README.md)   | Variant representations built outside a known owner                    |
| [no-repeated-selection](rules/no-repeated-selection/README.md)                 | Handlers repeating a test their sibling selector already made          |
| [no-repeated-conversion](rules/no-repeated-conversion/README.md)               | Conversions written identically in several places                      |
| [no-repeated-presence-check](rules/no-repeated-presence-check/README.md)       | Presence checks a called function already makes                        |
| [no-fact-outside-owner](rules/no-fact-outside-owner/README.md)                 | Facts about a type's values established outside its owner              |
| [no-embedded-translation](rules/no-embedded-translation/README.md)             | Variant fields read into the surrounding state instead of a projection |
| [no-overbroad-type-predicate](rules/no-overbroad-type-predicate/README.md)     | Type predicates that reject values their declared target permits       |

`max-held-context` allows `else if` chains and `no-else-if` reports them; early returns satisfy both.

## Type-aware setup and limits

- Each file must belong to a TypeScript 7-compatible `tsconfig.json` project with dependencies installed. Referenced projects may need declaration outputs built first.
- `prefer-truthy-presence-check` requires `strictNullChecks`.
- Run `tsc` too: type errors are not reported, and unresolved types make rules skip checks.
- Use plain `oxlint`, not `--type-aware`. The plugin runs its own bundled backend for Linux, macOS and Windows on x64 and arm64; no Go needed. It starts with every run, which adds about 10 ms when only syntax rules are enabled.
- Some rules read the whole program, including files outside the lint command. Project code is every non-declaration file outside `node_modules`; linked workspace packages count.
- Setup errors fail the run. Editor integration (`oxlint --lsp`) is not supported.

## Development

See the [development guide](https://github.com/Aetherall/aexlint/blob/master/docs/development.md).

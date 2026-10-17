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

The limits are starting points, not validated thresholds. For other limits or a subset of rules, use the plugins.

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
  "jsPlugins": ["@aetherall/aexlint", "@aetherall/aexlint/typed-plugin"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }],
    "aexlint-typed/no-repeated-value-policy": "warn"
  }
}
```

The plugins enable no rules by default. Drop the typed plugin for syntax checks only.

## Rules

### Syntax — `aexlint/`

| Rule                                                                 | Reports                                           |
| -------------------------------------------------------------------- | ------------------------------------------------- |
| [max-expression-depth](docs/rules/max-expression-depth.md)           | Nested calls                                      |
| [max-expression-complexity](docs/rules/max-expression-complexity.md) | Work inside one expression                        |
| [max-decision-depth](docs/rules/max-decision-depth.md)               | Nested logical groups and ternaries               |
| [max-call-assembly](docs/rules/max-call-assembly.md)                 | Wide calls drawing on many outside names          |
| [max-held-context](docs/rules/max-held-context.md)                   | Enclosing decisions, loops, and callbacks         |
| [no-else-if](docs/rules/no-else-if.md)                               | `else if` chains                                  |
| [no-multiline-condition](docs/rules/no-multiline-condition.md)       | Conditions spanning multiple lines                |
| [prefer-domain-predicate](docs/rules/prefer-domain-predicate.md)     | Unnamed classifications built from several checks |

### Type-aware — `aexlint-typed/`

| Rule                                                                               | Reports                                                          |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| [prefer-truthy-presence-check](docs/typed-rules/prefer-truthy-presence-check.md)   | Nullish comparisons equivalent to truthiness checks              |
| [max-interpretation-spread](docs/typed-rules/max-interpretation-spread.md)         | Literal unions interpreted across too many files                 |
| [max-projection-spread](docs/typed-rules/max-projection-spread.md)                 | Repeated object projections across files                         |
| [no-forgeable-validated-result](docs/typed-rules/no-forgeable-validated-result.md) | Validated results callers can construct without validation       |
| [no-caller-enforced-invariant](docs/typed-rules/no-caller-enforced-invariant.md)   | Caller-side checks that a receiving class does not enforce       |
| [no-repeated-value-group](docs/typed-rules/no-repeated-value-group.md)             | Groups of literal values written out in several places           |
| [no-repeated-value-policy](docs/typed-rules/no-repeated-value-policy.md)           | The same literal test and outcome in several files               |
| [no-procedural-value-dispatch](docs/typed-rules/no-procedural-value-dispatch.md)   | Branches on literal values that contain their own control flow   |
| [no-external-variant-behavior](docs/typed-rules/no-external-variant-behavior.md)   | Variant representations built outside a known owner              |
| [no-repeated-selection](docs/typed-rules/no-repeated-selection.md)                 | Handlers repeating a test their sibling selector already made    |
| [no-repeated-conversion](docs/typed-rules/no-repeated-conversion.md)               | Conversions written identically in several places                |
| [no-repeated-presence-check](docs/typed-rules/no-repeated-presence-check.md)       | Presence checks a called function already makes                  |
| [no-fact-outside-owner](docs/typed-rules/no-fact-outside-owner.md)                 | Facts about a type's values established outside its owner        |
| [no-overbroad-type-predicate](docs/typed-rules/no-overbroad-type-predicate.md)     | Type predicates that reject values their declared target permits |

`no-else-if` and `max-held-context` disagree on uniform `else if` chains; pick one if that matters to you.

## Type-aware setup and limits

- Each file must belong to a TypeScript 7-compatible `tsconfig.json` project with dependencies installed. Referenced projects may need declaration outputs built first.
- `prefer-truthy-presence-check` requires `strictNullChecks`.
- Run `tsc` too: type errors are not reported, and unresolved types make rules skip checks.
- Use plain `oxlint`, not `--type-aware`. The plugin runs its own bundled backend for Linux, macOS and Windows on x64 and arm64; no Go needed.
- Some rules read the whole program, including files outside the lint command. Project code is every non-declaration file outside `node_modules`; linked workspace packages count.
- Setup errors fail the run. Editor integration (`oxlint --lsp`) is not supported.

## Development

See the [development guide](https://github.com/Aetherall/aexlint/blob/master/docs/development.md).

# aexlint

Readability rules for [Oxlint](https://oxc.rs/docs/guide/usage/linter). Choose the rules you want; none are enabled by default. There is no recommended preset or autofix.

## Setup

Requires Node.js 24+ and Oxlint 1.83.0 (the tested version).

```sh
pnpm add -D oxlint@1.83.0 aexlint
```

Add to `.oxlintrc.json`:

```json
{
  "jsPlugins": ["aexlint", "aexlint/typed-plugin"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }],
    "aexlint/max-expression-complexity": ["warn", { "max": 4 }],
    "aexlint/max-decision-depth": ["warn", { "max": 2 }],
    "aexlint-typed/prefer-truthy-presence-check": "warn"
  }
}
```

These limits are examples, not recommendations. Remove `aexlint/typed-plugin` and its rule if you only want syntax checks.

```sh
pnpm exec oxlint src
```

Normal Oxlint severities, overrides, ignore patterns, and disable comments apply.

## Rules

### Syntax — `aexlint/`

| Rule                                                                 | Checks                                            | Status       |
| -------------------------------------------------------------------- | ------------------------------------------------- | ------------ |
| [max-expression-depth](docs/rules/max-expression-depth.md)           | Nested calls                                      | Stable       |
| [max-expression-complexity](docs/rules/max-expression-complexity.md) | Work inside one expression                        | Stable       |
| [max-decision-depth](docs/rules/max-decision-depth.md)               | Nested logical groups and ternaries               | Stable       |
| [max-held-context](docs/rules/max-held-context.md)                   | Enclosing decisions, loops, and callbacks         | Prototype    |
| [no-else-if](docs/rules/no-else-if.md)                               | `else if` chains                                  | Experimental |
| [no-multiline-condition](docs/rules/no-multiline-condition.md)       | Conditions spanning multiple lines                | Experimental |
| [prefer-domain-predicate](docs/rules/prefer-domain-predicate.md)     | Unnamed classifications built from several checks | Experimental |

### Type-aware — `aexlint-typed/`

| Rule                                                                                       | Checks                                                        | Status       |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------- | ------------ |
| [prefer-truthy-presence-check](docs/typed-rules/prefer-truthy-presence-check.md)           | Nullish comparisons equivalent to truthiness checks           | Stable       |
| [max-interpretation-spread](docs/typed-rules/max-interpretation-spread.md)                 | Literal unions interpreted across too many files              | Experimental |
| [require-interface-implementations](docs/typed-rules/require-interface-implementations.md) | Interfaces with fewer than two explicit class implementations | Experimental |
| [no-forgeable-validated-result](docs/typed-rules/no-forgeable-validated-result.md)         | Validated results callers can construct without validation    | Experimental |
| [max-projection-spread](docs/typed-rules/max-projection-spread.md)                         | Repeated object projections across files                      | Experimental |
| [no-caller-enforced-invariant](docs/typed-rules/no-caller-enforced-invariant.md)           | Caller-side checks that a receiving class does not enforce    | Experimental |

Rule pages describe options and limits. Experimental rules have narrower coverage; review their findings before adopting them.

## Type-aware setup and limits

- Each file must belong to a TypeScript 7-compatible `tsconfig.json` project with dependencies installed. Referenced projects may need declaration outputs built first.
- `prefer-truthy-presence-check` requires `strictNullChecks` or `strict`.
- Run `tsc` too: type errors are not lint diagnostics, and unresolved types can cause rules to skip checks.
- Use plain `oxlint`, not `--type-aware` or `OXLINT_TSGOLINT_PATH`. The plugin runs its own bundled backend; no Go installation is needed.
- Some rules inspect the whole TypeScript program, including files outside the lint command. Results are reused within a run.
- Setup errors and backend failures fail the lint run. Editor integration (`oxlint --lsp`) is not supported.

Linux, macOS, and Windows on x64 and arm64 are smoke-tested in CI. All six binaries ship in one package, so the download is larger than a JS-only plugin.

## Development

See the repository's [development guide](https://github.com/Aetherall/aexlint/blob/master/docs/development.md).

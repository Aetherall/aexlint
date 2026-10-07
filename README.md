# aexlint

Readability rules for [Oxlint](https://oxc.rs/docs/guide/usage/linter). Run standalone checks or choose individual rules in your Oxlint configuration. No autofix.

## Standalone CLI

Requires Node.js 24+ and aexlint 0.2.2 or later.

```sh
pnpx @aetherall/aexlint check
pnpx @aetherall/aexlint check src --typed
pnpx @aetherall/aexlint check src --typed --experimental --format json
```

pnpx downloads and caches aexlint and its pinned Oxlint runtime without adding dependencies or configuration to your project. The first download includes all six native backends (about 46 MB, plus dependencies). No Go installation is needed.

`check` defaults to the current directory and enables these stable syntax rules as errors:

| Rule                        | Maximum |
| --------------------------- | ------- |
| `max-expression-depth`      | 3       |
| `max-expression-complexity` | 4       |
| `max-decision-depth`        | 2       |

These are CLI starting points, not validated universal limits. `--typed` adds `prefer-truthy-presence-check`; your project still needs its dependencies installed and a usable TypeScript project (see below). `--experimental` adds experimental/prototype syntax rules, including `max-held-context` with maximum 3. Combined with `--typed`, it also adds all experimental typed rules, with interpretation spread maximum 4 and projection spread maximum 2.

The CLI ignores root and nested Oxlint configurations and does not enable unrelated built-in checks. Oxlint's normal file exclusions, ignore files, and disable comments apply. Add repeatable `--ignore-pattern <glob>` options for extra exclusions; use `--` before paths starting with a dash. `--format`/`-f` accepts Oxlint output formats. Findings fail the command; Oxlint's exit code is preserved. Run `--help` or `--version` for CLI information.

For custom rule thresholds, overrides, or additional Oxlint options, use the plugin integration below. Installing the package also provides `pnpm exec aexlint check`.

## Plugin setup

Requires Node.js 24+ and Oxlint 1.83.0 (the tested version).

The plugins enable no rules by default; configure the rules and thresholds you want.

```sh
pnpm add -D oxlint@1.83.0 @aetherall/aexlint
```

Add to `.oxlintrc.json`:

```json
{
  "jsPlugins": ["@aetherall/aexlint", "@aetherall/aexlint/typed-plugin"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }],
    "aexlint/max-expression-complexity": ["warn", { "max": 4 }],
    "aexlint/max-decision-depth": ["warn", { "max": 2 }],
    "aexlint-typed/prefer-truthy-presence-check": "warn"
  }
}
```

These limits are examples, not recommendations. Remove `@aetherall/aexlint/typed-plugin` and its rule if you only want syntax checks.

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

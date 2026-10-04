# aexlint

Higher-level readability rules for [Oxlint](https://oxc.rs/docs/guide/usage/linter). aexlint ships as one npm package with two Oxlint JS plugins:

| Plugin                 | Rule prefix      | What its rules see                                                         |
| ---------------------- | ---------------- | -------------------------------------------------------------------------- |
| `aexlint`              | `aexlint/`       | Syntax and lexical scope of the linted file                                |
| `aexlint/typed-plugin` | `aexlint-typed/` | Resolved TypeScript types, and for some rules the whole TypeScript program |

Both run under plain `oxlint`; there is no separate CLI. No rule is enabled by default and there is no recommended preset: enable each rule you want and choose its limit.

## Rules

### `aexlint` (syntax rules)

| Rule                                                                 | Status       | Reports                                                                                         | Options                  |
| -------------------------------------------------------------------- | ------------ | ----------------------------------------------------------------------------------------------- | ------------------------ |
| [max-expression-depth](docs/rules/max-expression-depth.md)           | stable       | Invocations buried under too many levels of other invocations, e.g. `publish(encode(parse(x)))` | `{ "max": n }`, required |
| [max-expression-complexity](docs/rules/max-expression-complexity.md) | stable       | Too much work packed into one expression: calls, operators, logical groups                      | `{ "max": n }`, required |
| [max-decision-depth](docs/rules/max-decision-depth.md)               | stable       | Logical groups and ternaries nested too deeply, e.g. `a && (b \|\| (c && d))`                   | `{ "max": n }`, required |
| [max-held-context](docs/rules/max-held-context.md)                   | prototype    | Decisions made inside too many enclosing branches, loops, cases and callbacks                   | `{ "max": n }`, required |
| [no-multiline-condition](docs/rules/no-multiline-condition.md)       | experimental | An `if`, `while`, `do … while` or `for` condition that spans several lines                      | none                     |
| [prefer-domain-predicate](docs/rules/prefer-domain-predicate.md)     | experimental | A classification rebuilt from several checks of one subject, with no name of its own            | none                     |

The three stable rules measure separate dimensions of one expression: how deep its inputs go, how much it computes, and how deeply its decisions nest. Enable them together; one expression can be reported by several of them.

- **Depth** counts invocation levels. A fluent chain like `source.first().second().done()` stays at depth 1, and arrays, objects, operators and `await` don't reset it.
- **Complexity** scores workload. A flat `a && b && c` group costs 1 however long it is. Object fields are checked one by one rather than summed, and string formatting is free.
- **Decision depth** counts nested logical groups and ternaries. A chain of one operator is one level.

### `aexlint/typed-plugin` (type-aware rules)

| Rule                                                                                       | Status       | Reports                                                                                                                                           | Options                       |
| ------------------------------------------------------------------------------------------ | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| [prefer-truthy-presence-check](docs/typed-rules/prefer-truthy-presence-check.md)           | stable       | `x !== undefined` in a condition where the type proves `if (x)` equivalent                                                                        | none                          |
| [max-interpretation-spread](docs/typed-rules/max-interpretation-spread.md)                 | experimental | A literal union (`"free" \| "team" \| …`) checked for particular values in too many files, none of which the compiler flags when a value is added | `{ "max": n }`, defaults to 4 |
| [require-interface-implementations](docs/typed-rules/require-interface-implementations.md) | experimental | An interface implemented by fewer than two classes (`implements` clauses) across the TypeScript program                                           | none                          |
| [no-forgeable-validated-result](docs/typed-rules/no-forgeable-validated-result.md)         | experimental | An operation that checks the data it returns, then returns a project-owned object type other code can produce without that check                  | none                          |
| [max-projection-spread](docs/typed-rules/max-projection-spread.md)                         | experimental | An object literal projecting a value of a project type through member chains that too many files restate                                          | `{ "max": n }`, defaults to 2 |
| [no-caller-enforced-invariant](docs/typed-rules/no-caller-enforced-invariant.md)           | experimental | Data a caller checks, then hands to a project class that stores it without that check                                                             | none                          |

`prefer-truthy-presence-check` requires `strictNullChecks` (or `strict`). `max-interpretation-spread` counts files across the whole TypeScript program of the linted file, including files that are not being linted.

Every rule only reports; none provides fixes. Each rule's page documents exactly what it counts, what it accepts, and its limitations.

## Usage

Requirements: Node.js 24 or later and Oxlint 1.83.0, the tested version. The typed plugin's backend is a native executable bundled in the package for Linux, macOS and Windows on x64 and arm64; no Go toolchain or postinstall step is needed.

```sh
pnpm add -D oxlint aexlint
```

aexlint is not published yet. Until it is, install a tarball built with `pnpm pack` from this repository.

### Syntax rules

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "jsPlugins": ["aexlint"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }],
    "aexlint/max-expression-complexity": ["warn", { "max": 4 }],
    "aexlint/max-decision-depth": ["warn", { "max": 2 }]
  }
}
```

The limits shown are the ones aexlint uses on itself, not recommendations. Rules that take `max` reject a missing value, unknown keys and anything other than a positive integer.

### Type-aware rules

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/prefer-truthy-presence-check": "warn",
    "aexlint-typed/max-interpretation-spread": ["warn", { "max": 4 }]
  }
}
```

Do not enable Oxlint's `--type-aware` or set `OXLINT_TSGOLINT_PATH` for these rules: the plugin runs its own backend.

Each linted file must belong to a TypeScript 7-compatible `tsconfig.json` project, with its dependencies installed. Projects referenced by that tsconfig may need their declaration outputs built first.

### Both plugins together

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "jsPlugins": ["aexlint", "aexlint/typed-plugin"],
  "rules": {
    "aexlint/max-expression-complexity": ["error", { "max": 4 }],
    "aexlint-typed/prefer-truthy-presence-check": "error"
  },
  "overrides": [
    {
      "files": ["**/*.test.ts"],
      "rules": { "aexlint/max-expression-complexity": "off" }
    }
  ]
}
```

Rules from both plugins behave like any Oxlint rule: severities, `overrides`, `ignorePatterns`, output formats and disable comments all apply.

```ts
// oxlint-disable-next-line aexlint/max-expression-depth
publish(encode(parse(input)));

// oxlint-disable-next-line aexlint-typed/prefer-truthy-presence-check
if (entry !== undefined) {
}
```

Run it as usual:

```sh
pnpm exec oxlint src
```

## What to expect from the typed plugin

- **TypeScript errors are left to `tsc`.** Following Oxlint's and typescript-eslint's convention, the plugin does not report type errors, unresolved imports or invalid tsconfig options, and a tsconfig error does not stop its files from being linted. Where TypeScript cannot resolve a type, typed rules stay silent rather than guess, so run `tsc` as well: typed results are only meaningful for a project that type-checks.
- **Setup problems fail the run.** A file outside every tsconfig project, an invalid rule option, or a backend crash is reported as an Oxlint plugin error starting with `aexlint-typed:`, which fails the run whatever the rule's severity. An invalid option stops the backend: the first file reports the reason, and later files report that they were not linted.
- **Cost.** When Oxlint loads the plugin, it starts one backend process. The first linted file of each TypeScript program has that program checked once, for every file in it; the program's other files reuse that result.
- **Limits.** Editor integration (`oxlint --lsp`) is not supported yet. The plugin has only been run on Linux; macOS and Windows binaries are built but untested. Design details are in [docs/native.md](docs/native.md#oxlint-typed-plugin).

The typed rules need a second plugin because Oxlint's JS plugin API has no type information, and its own type-aware backend only accepts Oxlint's built-in rules. The rationale is in [docs/development.md](docs/development.md#why-a-second-plugin).

## Contributing

See [docs/development.md](docs/development.md) for setup, commands and rule authoring, and [AGENTS.md](AGENTS.md) for the rule design requirements.

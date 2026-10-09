# no-repeated-value-policy

Report a test of a literal union whose outcome is repeated in other files. The same test leading to the same work is a business rule written in several places. When the rule changes, every copy has to be found and changed the same way.

**Status:** experimental. Diagnostic-only; no fixes or suggestions. See [Evidence](no-repeated-value-group.md#evidence).

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-repeated-value-policy": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Example

Reported in both files, which each decide that a free account gets only the basic features:

```ts
// invoices.ts
features: account.plan === "free" ? ["basic"] : account.features,
// billing.ts
features: plan === "free" ? ["basic"] : undefined,
```

Accepted when each file reaches a different outcome, or when the outcome is trivial:

```ts
const badge = account.plan === "free" ? "trial" : account.name;
if (account.plan === "free") return null;
```

## What counts

- Tests and vocabularies follow [no-repeated-value-group](no-repeated-value-group.md#what-counts), except that any number of accepted values counts.
- A test's question is the vocabulary values, the values each subject accepts and the text of other operands. Subject spelling does not matter.
- Its outcome is the `then` branch of an `if`, the true branch of a conditional expression, the right side of a `&&` guard, or the body of a `case` group. Blocks are unwrapped and a final unlabeled `break` is dropped, so an `if` and a `case` doing the same work agree. Outcomes are compared as source text with whitespace collapsed.
- Trivial outcomes do not count: nothing, `return`, `break`, `continue`, `null`, `undefined`, booleans, `void`, `""`, `0`, `-1`, `[]` and `{}`.
- Matches must be in different files. The search covers project source files of the TypeScript program, including unlinted files and tests, but not declaration files or external libraries.

## Reporting scope

Reports `repeatedValuePolicy` on each test in the linted file whose question and outcome appear in another file. The span is the test or the case labels. The message names the subject and the accepted values, or the rejected values when fewer, and does not depend on other files. Help gives the number of files and lists the nearest others first.

## Boundaries

Versioned serializers repeat the same mapping on purpose. Turn the rule off for them with Oxlint overrides, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries). Overridden files still count when other files are checked.

## Limitations

- Outcomes are compared as text. The same work written with different names, order or formatting beyond whitespace is not matched, and identical text can refer to different variables.
- `else` branches and false branches are not compared, so two places that agree only on the alternative are not reported.
- Whole copied files and copied components are reported once per repeated test.
- Other tsconfig programs are not searched.

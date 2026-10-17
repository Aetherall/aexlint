# no-repeated-value-policy

Reports a test of a literal union whose outcome is repeated in another file. The same test leading to the same work is a business rule written in several places.

```json
{ "rules": { "aexlint/no-repeated-value-policy": "warn" } }
```

No options.

## Example

Reported in both files:

```ts
// invoices.ts
features: account.plan === "free" ? ["basic"] : account.features,
// billing.ts
features: plan === "free" ? ["basic"] : undefined,
```

Accepted: different or trivial outcomes.

```ts
const badge = account.plan === "free" ? "trial" : account.name;
if (account.plan === "free") return null;
```

## What counts

- Tests and vocabularies follow [no-repeated-value-group](../no-repeated-value-group/README.md#what-counts), with any number of accepted values.
- Tests match on the values accepted and the text of other operands; subject spelling does not matter.
- The outcome is the `then` branch, the ternary's true branch, the right side of a `&&` guard, or a `case` body. Blocks are unwrapped and a final `break` dropped. Outcomes are compared as text with whitespace collapsed.
- Trivial outcomes do not count: nothing, `return`, `break`, `continue`, `null`, `undefined`, booleans, `void`, `""`, `0`, `-1`, `[]`, `{}`.
- Matches must be in different files of the program.

## Reporting and limits

Reports `repeatedValuePolicy` on each such test in the linted file; help lists the other files. No fixes.

Identical text can refer to different things, and the same work written differently is missed. `else` branches are not compared. Versioned serializers repeat mappings on purpose; turn the rule off for them with [overrides](../no-repeated-value-group/README.md#reporting-and-limits).

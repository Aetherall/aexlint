# prefer-truthy-presence-check

Prefer `if (value)` or `if (!value)` when types prove that comparing with `undefined` means the same thing. This is a style preference, not a correctness check.

**Status:** stable. Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": { "aexlint-typed/prefer-truthy-presence-check": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error.

Requires `strictNullChecks` (or `strict`). Without it, reports `strictNullChecksRequired` at the start of the file and skips comparison analysis. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits) for project requirements.

## Example

Reported:

```ts
declare const entry: { id: string } | undefined;
if (entry !== undefined) {
}
```

Accepted replacement:

```ts
declare const entry: { id: string } | undefined;
if (entry) {
}
```

## What reports

- `!=` and `!==` report `preferTruthyCheck`; `==` and `===` report `preferFalsyCheck`. Either operand order works.
- The span is the whole comparison, excluding surrounding parentheses.
- Comparisons must be used as boolean tests: conditions of `if`, loops or ternaries, or operands of `!`. `&&` and `||` count only when their result is also used as a boolean.
- Assignments, returns, call arguments, `??`, casts and other value-producing contexts are not followed, even with a `boolean` annotation.

## Type boundaries

The flow-sensitive type must include `undefined` and at least one present member. The identifier `undefined` must itself have exactly that type; a differently typed shadow does not count. A generic's resolved base constraint may qualify.

Every present member must be provably truthy:

- Nonempty string literals, nonzero finite number literals, nonzero bigint literals, `true`, and symbols qualify.
- Object types qualify only if they exclude falsy primitives. Broad structural types such as `{}`, `Object`, and `{ toString(): string }` are not enough.
- In DOM projects, types admitting `HTMLAllCollection` are excluded because `document.all` is falsy. This includes broad `object` and some callable types.
- Broad primitives, `any`, `unknown`, `never`, enums, intersections, unresolved types, and unevaluated template or conditional types do not prove truthiness.

Strict comparisons with a nullable value are accepted: `null !== undefined` is true, but `null` is falsy. Loose comparisons can qualify because they already treat `null` and `undefined` alike.

## Limitations

`string | undefined` must keep its explicit comparison because an empty string is present but falsy. Flow narrowing may make a broad type eligible, or remove `undefined` and put it outside this rule.

Types, assertions and compiler flow analysis are trusted, not runtime-validated. Unchecked indexed access may omit `undefined`, preventing a report. Run `tsc` separately; editor integration is unsupported. If using `typescript/strict-boolean-expressions`, configure it to allow the nullable types you want to test this way.

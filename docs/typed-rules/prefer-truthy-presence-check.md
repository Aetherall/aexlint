# prefer-truthy-presence-check

Prefers `if (value)` or `if (!value)` when types prove that comparing with `undefined` means the same thing. A style preference, not a correctness check.

```json
{ "rules": { "aexlint-typed/prefer-truthy-presence-check": "warn" } }
```

No options. Requires `strictNullChecks`; without it, reports `strictNullChecksRequired` at the start of the file.

## Example

Reported:

```ts
declare const entry: { id: string } | undefined;
if (entry !== undefined) {
}
```

Accepted:

```ts
declare const entry: { id: string } | undefined;
if (entry) {
}
```

## What counts

- The comparison is used as a boolean: an `if`, loop or ternary condition, a `!` operand, or a `&&`/`||` operand whose result is itself a boolean test. Assignments, returns and arguments are not followed.
- The narrowed type holds `undefined` and at least one present member, and every present member is always truthy: non-empty string literals, non-zero number and bigint literals, `true`, symbols, and object types that admit no primitive.
- `{}`, `Object`, broad primitives, enums, `any`, `unknown`, branded primitives and types admitting `document.all` do not qualify. Strict comparisons on types that also hold `null` do not either.

## Reporting and limits

Reports `preferTruthyCheck` for `!=`/`!==` and `preferFalsyCheck` for `==`/`===`, on the comparison. No fixes. Types are trusted, not validated; unchecked indexed access may hide `undefined`. If you use `typescript/strict-boolean-expressions`, allow the nullable types you test this way.

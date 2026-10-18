# no-forgeable-validated-result

Reports an operation that checks data and returns it as a plain object type any code can build without the check. Consumers cannot tell whether a value passed validation.

```json
{ "rules": { "aexlint/no-forgeable-validated-result": "warn" } }
```

No options.

## Example

Reported on `parseName`:

```ts
type Name = { value: string };
export function parseName(value: string): Name {
  if (!value.trim()) throw new Error("empty name");
  return { value };
}
```

Accepted: the class owns the check.

```ts
export class Name {
  constructor(readonly value: string) {
    if (!value.trim()) throw new Error("empty name");
  }
}
```

## What counts

- Operations are functions, methods, and functions assigned to variables or properties. Constructors, accessors and inline callbacks are skipped.
- A check is a `throw` under a condition reading input, or after an `if` reading input that returns early, or a call to a project function that makes one. Presence, `typeof`, `instanceof`, `in` and type-guard tests only narrow, so they are not checks.
- The checked data must reach the result: checking `input.allowed` and returning `input.id` does not overlap; returning `input` does.
- The result, after unwrapping promises and nullish, is a project object type with data. It is forgeable unless it is a class instance or has private, `#private` or unique-symbol members.

## Reporting and limits

Reports `forgeableValidatedResult` on the operation's name; help names the first check. No fixes.

Authorization checks whose data reaches the result also report. Parsers, serializers and immediately consumed records can legitimately stay plain. Reassignments, callbacks, `this` state, `Result` failures and library validators are not followed.

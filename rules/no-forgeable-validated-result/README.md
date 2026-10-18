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

A brand also keeps the check: a `#private` member, or a property keyed by a unique symbol that only the validating code sets. Keep such a constructor to validation: do I/O before it and pass the result in. A record its one caller consumes at once can stay plain (see limits); it does not need a class.

## What counts

- Operations are functions, methods, and functions assigned to variables or properties. Constructors, accessors and inline callbacks are skipped.
- A check is a `throw` under a condition reading input, or after an `if` reading input that returns early, or a call to a project function that makes one. Presence, `typeof`, `instanceof`, `in` and type-guard tests only narrow, so they are not checks.
- The checked data must reach the result: checking `input.allowed` and returning `input.id` does not overlap; returning `input` does.
- The result, after unwrapping promises and nullish, is a project object type with data, or a union of such types of which one has data. It is forgeable unless it is a class instance or has private, `#private` or unique-symbol members.
- Data a checking project function returns counts as checked: `return { slot: slotName(input) }` reports when `slotName` rejects `input`.

## Reporting and limits

Reports `forgeableValidatedResult` on the operation's name; help names the first check. No fixes.

Authorization checks whose data reaches the result also report. Parsers, serializers and immediately consumed records can legitimately stay plain. Reassignments, callbacks, `this` state, `Result` failures and library validators are not followed.

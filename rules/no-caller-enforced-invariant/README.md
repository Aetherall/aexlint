# no-caller-enforced-invariant

Reports data a caller checks before handing it to a project class that stores it without that check. Other callers can skip the check. Sometimes the class should own it; sometimes it belongs to this operation.

```json
{ "rules": { "aexlint/no-caller-enforced-invariant": "warn" } }
```

No options.

## Example

Reported on the `value` argument:

```ts
class Name {
  constructor(readonly value: string) {}
}
function parseName(value: string): Name {
  if (!value.trim()) throw new Error("empty name");
  return new Name(value);
}
```

Accepted when `Name`'s constructor makes the check itself.

## What counts

- Checks are those of [no-forgeable-validated-result](../no-forgeable-validated-result/README.md#what-counts).
- The hand-off calls a constructor or method with a body on a project class, and the callee stores the parameter: a parameter property, a write to `this`, a collection update, or a static factory returning its class. Calls on `this`/`super`, from inside the class, and calls building a thrown error are skipped.
- The check precedes the call in an enclosing statement list and can reject on a path that reaches it. Paths whose `if` conditions exclude each other do not count.
- The checked data reaches the argument unchanged, directly, inside an object, or through a property path.
- The callee, its helpers and the types it constructs make no equivalent check.

## Reporting and limits

Reports `checkedBeforeHandOff` on the argument, naming the check and the receiving member. Help says what checked data the class does not receive. No fixes.

Authorization and endpoint limits can legitimately stay with the caller. Only classes count as owners. Reassignments, callbacks and rest parameters are not followed.

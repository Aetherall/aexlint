# no-caller-enforced-invariant

Report data a caller checks before handing it to a project class that stores it without that check. Other callers can bypass the rule. Sometimes the class should own it; sometimes it belongs only to this operation.

**Status:** experimental research candidate; not ready for promotion. Legitimate caller policies can report; a finding does not establish that the check belongs to the receiving class. Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-caller-enforced-invariant": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Example

Reported on the `value` argument to `new Name`:

```ts
class Name {
  constructor(readonly value: string) {}
}
function parseName(value: string): Name {
  if (!value.trim()) throw new Error("empty name");
  return new Name(value);
}
```

Accepted when the class checks its own input:

```ts
class Name {
  constructor(readonly value: string) {
    if (!value.trim()) throw new Error("empty name");
  }
}
function parseName(value: string): Name {
  return new Name(value);
}
```

## What counts

- Callers are functions, methods, constructors and function expressions with bodies in the linted file. Nested bodies are analyzed separately.
- Checks use the [shared rejection and data-flow analysis](no-forgeable-validated-result.md#measurement): input-dependent throws and rejecting project calls count; narrowing-only tests, type guards and assertion functions do not. Library validators are not followed.
- A hand-off is a call or construction resolving to a project class's constructor or method with a body. Discarded constructions/static factory results, plain functions and library classes are excluded.
- The check must precede the call in an enclosing statement list. Checks in another branch of the same `if`, after the call, or in a branch that exits first do not count. A conditional check before a later call can count even if it does not run on every path, but not when the call can only run where the check cannot reject: the `if` conditions guarding each, below the statement list enclosing both, make claims about the same value that no value satisfies together, as in `if (existing && existing.owner !== owner) throw …` followed by `if (!existing) save(Item.create({ owner }))`.
- The callee must store the parameter without checking it. Storage includes constructor parameter properties, writes into `this`, collection updates and delegation to storing members. Static factories returning their class, directly or inside a returned object, treat parameters they read as stored.
- A callee's own checks, project helper checks and checks by constructed value types are recognized across files. Calls within the owner class or its subclasses, and calls on `this` or `super`, are excluded.
- Checked data must definitely reach the argument unchanged: directly, inside an object, or through a property path. Computed or selected values and method-derived values do not qualify. Two unknown getter results on the same object are not definite overlap.

## Reporting scope

Reports `checkedBeforeHandOff` once per argument, spanning that argument. It names the first preceding check whose data overlaps the argument and the class member receiving it.

Help says whether the class receives all the checked data at stored parameters, or names the first datum it does not receive. A check's method calls also read an unknown part of their receiver. State read through `this` is context, not input; it can explain why the check belongs to the caller.

## Limitations

- Authorization, endpoint limits and other operation-specific policies can legitimately live in the caller. A report is an observation, not proof that the check should move.
- Exclusive paths are read from `if` conditions: tests joined by `&&` hold in the `then` branch, and the opposites of tests joined by `||` in the `else` branch. Other relations between conditions, such as two different values that cannot both be set, are not followed.
- Unknown data reads cannot be assigned to specific fields. A check of `isValid(input)` followed by handing over `input.name` may be described as reading additional data.
- Only classes count as owners; function-based modules are not recognized. Reassignments, callback data flow and rest parameters are not followed.

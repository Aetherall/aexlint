# no-repeated-presence-check

Report a caller that checks a value is present before calling a function that already handles the missing value the same way.

**Status:** experimental. Built on the [decision model](../design/decision-model.md#presence). Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-repeated-presence-check": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Reported code

```ts
// tags.ts
export function names(tags: TagList | undefined): string[] {
  if (tags === undefined) return [];
  return tags.names;
}
// badge.ts
const labels = tags ? names(tags) : [];
```

The same fact, "the value is present", is checked twice: once by `names` and once by its caller. The caller's check changes nothing. Readers must compare both to know which one matters, and the two can drift apart.

## What counts

- A **guarded call** is `c ? call : fallback`, `c ? fallback : call` or `c && call`, where `c` is a [presence test](../design/decision-model.md#presence) of an expression the call passes as an argument, and the call runs when the value is present. The call must be the whole branch: `c ? g(f(x)) : []` does not count.
- The called function must be a project function or method with a body, found across the call's [boundary](../design/decision-model.md#boundaries). The parameter receiving the value must not be a rest parameter. A `this` parameter is skipped. The argument as written, without the narrowing the caller's check brings, must be assignable to the parameter; otherwise calling the function directly would not type-check, so the caller's check is needed. A cast such as `value as any` makes it assignable. A default value replaces `undefined`, so a function with one does not reject it.
- The function must **reject** the missing value at its entry: one of its [entry tests](../design/decision-model.md#boundaries) tests the parameter's presence and leaves with a constant when it is missing, and the rest of the function uses the value. For example, its first statement is `if (test) return constant;`, possibly with other conditions joined by `||`; its whole body is `test ? work : constant` or `test ? constant : work`, possibly with other conditions; its only statement is `if (test) { work }`, which returns `undefined`; or its whole body is `value !== undefined && work`, which returns `false`, or `value && work`, which returns the missing value when the type allows only one. A single statement `value && work` returns `undefined`.
- Both sides must give the same constant when the value is missing: `undefined` (including a bare `return` and `void 0`), `null`, `true`, `false`, the same string or number, `[]`, or `{}`. For `c && call`, the caller gives the missing value itself, `false` for comparisons, so the function must return that value.
- Every missing value the caller's check removes, and the value's type allows, must be rejected by the function: `undefined`, `null`, or other falsy values such as `""` and `0`. A truthiness check on a `string | undefined` also removes `""`, so it only matches a function that also tests truthiness.
- The call's other parts must be safe to evaluate when the value is missing: the callee and the other arguments are identifiers, literals, `this`, or property reads of them, and none reads the checked value.

## Accepted code

- Different fallbacks. `value ? format(value) : ""` with `format` returning `"—"` is the caller's own policy.
- A function that does something else with a missing value: generates a default, deletes an entry, or switches to another mode.
- A function whose rejection is not its first statement, whose missing branch is not a constant, or whose other branch does not use the value. Async functions and generators wrap their results, so they never return the constant itself.
- A caller that proves presence for a larger block, with an early return or an `if` statement.
- Changing only an imported type can change the result: if the checked property becomes `Item | null | undefined` and the function only tests `=== undefined`, the caller's check also removes `null` and is no longer repeated.

## Reporting scope

Reports `repeatedPresenceCheck` on the caller's condition. The message names the checked expression and the function. The help names the shared fallback and recommends either calling the function directly or making its parameter required, so one side owns the check.

## Boundaries

Versioned serializers repeat checks on purpose. Turn the rule off for them with Oxlint overrides, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries).

## Fixes

None. Which side should own the check is a design decision.

## Limitations

- Only conditional expressions and `&&` are callers. Early returns and `if` statements are not, because their check usually covers more than the call.
- Only constant fallbacks are compared. Two calls to the same default factory are not.
- A caller's check is matched to an argument by the same expression, so a check on a renamed copy of the value is not found.
- Only presence is compared. Repeated literal-value selection is outside this rule. Calls whose parameter type already requires the narrowed value are not reported.

# no-repeated-presence-check

Reports a caller that checks a value is present before calling a function that already handles the missing value the same way. Readers must compare both checks to know which matters, and they can drift apart.

```json
{ "rules": { "aexlint/no-repeated-presence-check": "warn" } }
```

No options.

## Example

Reported on `tags`:

```ts
// tags.ts
export function names(tags: TagList | undefined): string[] {
  if (tags === undefined) return [];
  return tags.names;
}
// badge.ts
const labels = tags ? names(tags) : [];
```

Accepted: `value ? format(value) : ""` when `format` returns `"—"` for a missing value.

## What counts

- The caller is `c ? call : fallback`, `c ? fallback : call` or `c && call`, where `c` tests the presence of an argument and the call is the whole branch.
- The callee is a project function with a body. The unnarrowed argument must be assignable to its parameter, and the parameter has no default and is not a rest.
- The callee rejects the missing value at entry: an early `if (test) return constant`, a whole body `test ? work : constant`, `if (test) { work }`, or `value && work`. The rest of it uses the value. Async functions and generators never qualify.
- Both sides give the same constant for a missing value, and the callee rejects every missing value the caller's check removes (`undefined`, `null`, and for truthiness `""`, `0`).
- The callee and other arguments are plain reads safe to evaluate when the value is missing.

## Reporting and limits

Reports `repeatedPresenceCheck` on the caller's condition; help names the shared fallback. No fixes: which side owns the check is a design decision.

Early returns and `if` statements are not callers, because they usually guard more than the call. Only constant fallbacks are compared. A check on a renamed copy of the argument is missed.

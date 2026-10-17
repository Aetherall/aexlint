# max-decision-depth

Limits nested logical groups and ternaries inside one expression. Block nesting and total function complexity are not measured.

```json
{ "rules": { "aexlint/max-decision-depth": ["warn", { "max": 2 }] } }
```

`max` is a required positive integer.

## Example

Reported at `max: 2` (depth 3):

```ts
const nested = a && (b || (c && d));
```

Accepted (depth 2, a different condition, not a rewrite):

```ts
const alternatives = (a && b) || (c && d);
```

## What counts

- A run of the same `&&` or `||` operator adds one level, whatever its length. A different operator adds another.
- A ternary adds one above the deepest of its condition and alternatives.
- `!` adds nothing but separates groups. Other syntax passes the deepest child through.
- Functions, classes and object fields are checked on their own, not charged to the enclosing expression.

## Reporting and limits

Reports `tooDeep` on the outermost over-limit group or ternary; nested reports are suppressed. No fixes: rewriting conditions can change short-circuiting and results. Callees are not inspected. Other expression rules may report the same expression.

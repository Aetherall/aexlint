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
- A `const` read only once, inside a ternary branch or the right side of `&&`, `||` or `??` in a statement of the same list, carries its initializer's depth: it is part of that branch, decided early. A named condition (a comparison, `!`, `&&` or `||`) carries none.

## Reporting and limits

Reports `tooDeep` on the outermost over-limit group or ternary; nested reports are suppressed. No fixes: rewriting conditions can change short-circuiting and results. Callees are not inspected, and a predicate split into guard returns is measured one statement at a time. Other expression rules may report the same expression.

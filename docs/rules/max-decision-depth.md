# max-decision-depth

Stable, syntax-only Oxlint JS rule. Limits nested logical groups and ternaries, not block nesting or total function complexity.

## Configuration

```json
{
  "jsPlugins": ["@aetherall/aexlint"],
  "rules": { "aexlint/max-decision-depth": ["warn", { "max": 2 }] }
}
```

Exactly one options object is required. `max` must be a positive safe integer; unknown keys and extra options are rejected. There is no default or recommended limit.

## What counts

- A contiguous `&&` or `||` group adds one level, regardless of length. Parentheses and TypeScript wrappers do not split a same-operator group.
- A different logical group adds another level. A ternary adds one above the greatest depth in its condition or either alternative.
- Negation adds no level but separates groups: `a && !(b && c)` has depth 2.
- Other expression syntax preserves the greatest child depth without adding a level. This includes calls, arrays, JSX, operators, `await`, nullish coalescing and logical assignment.
- Functions, classes and object fields are checked independently, not charged to the expression receiving them. Computed object keys and spreads are still checked.
- Statement nesting adds no levels; each statement's expressions are checked independently.

## Example

With `max: 2`, this reports depth 3:

```ts
const nested = a && (b || (c && d));
```

This different grouping is accepted at depth 2; it is not an equivalent rewrite:

```ts
const alternatives = (a && b) || (c && d);
```

## Reporting and limits

Reports `tooDeep` over the complete outermost over-limit logical group or ternary, with its depth and maximum. Covered nested reports are suppressed; independent function, class and object-field computations keep their own reports.

No fixes or suggestions. Rewriting conditions can change short-circuiting, result values, evaluation order and side effects.

This is a structural measure, not a type, purity or readability proof. It does not inspect callees or count all decisions. [Expression workload](max-expression-complexity.md) and [input depth](max-expression-depth.md) are separate measures; their rules may report the same expression without cross-rule suppression.

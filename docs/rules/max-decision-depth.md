# max-decision-depth

## Intent and runtime

Limit nesting of expression-level decisions, separately from expression workload and invocation depth. Syntax-only Oxlint JS rule. It covers logical and ternary nesting, not an entire function's control-flow complexity.

Oxlint 1.83.0's `max-depth` limits block nesting, `complexity` measures control-flow complexity, and `no-nested-ternary` categorically rejects nested ternaries. This rule instead allows configurable depth across logical groups and ternaries, without banning mixed operators.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": {
    "aexlint/max-decision-depth": ["warn", { "max": 2 }]
  }
}
```

Exactly one options object is required. `max` must be a positive safe integer; unknown properties and extra options are rejected. There is no default. The example limit 2 is illustrative, not a universally established readability boundary.

## Measurement

- A contiguous group of logical ANDs or logical ORs counts as one decision level, regardless of length. Parentheses and TypeScript wrappers do not split a same-operator group.
- Nesting a different logical group adds a level. A ternary also adds one level above the greatest depth in its condition or either alternative.
- Other expression syntax adds no level but preserves decisions inside it. Calls, operators, arrays, templates, JSX, and `await` are not escape hatches.
- Nullish coalescing does not add a level, but decisions inside its operands remain visible. Logical assignment does not itself add a level in this initial contract.
- Negation adds no level, but prevents merging logical groups across it: `a && !(b && c)` contains two groups, not one. No boolean algebra or De Morgan rewriting is attempted.
- Function and class definitions and object fields are independent boundaries. Their nested decisions are checked, but not charged to the enclosing expression merely receiving those values. Computed object keys and spreads are still inspected.
- Statement nesting does not add levels. Nested `if`/loop/switch blocks should be assessed with block/control-flow rules; this rule checks their expressions independently.

## Accepted and reported examples

With `max: 2`:

```ts
const all = a && b && c && d && e;
const alternatives = (a && b) || (c && d);
const selection = enabled ? primary : fallback;
```

Their decision depths are 1, 2, and 1 respectively.

```ts
const nested = a && (b || (c && d));
```

This reports `tooDeep` with depth 3. The alternatives example and this nested example both have workload 3 under `max-expression-complexity`, but different decision depths.

A ternary example with depth 3:

```ts
const selected = a ? b : c ? d : e ? f : g;
```

Diagnostics span the complete outermost over-limit logical group or ternary. Nested reports covered by the same report are suppressed. Independent function/class/object-field computations retain their own diagnostics. Parentheses clarify grouping but do not lower the measured depth.

## Options and fixes

Only `max`; no auto-fixes or suggestions. Extracting conditions or rewriting boolean expressions can change evaluation order, short-circuit behavior, result values, scope, or side effects. The rule cannot choose meaningful predicate names.

## Limitations and interaction

This is a structural readability heuristic, not proof that an expression is difficult to understand. It does not infer predicate types, inspect callees, prove purity, or evaluate conditions. It does not measure decision count, callback nesting, state history, or total function complexity.

Workload and decision depth are independent: a broad expression may exceed only workload, while a deeply alternating logical structure may exceed only decision depth. Both rules may report the same expression; there is no cross-rule suppression or combined score.

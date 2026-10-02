# max-expression-depth

## Intent

Limit buried input-producing operations. A reader should not have to descend through a long chain of prerequisites and mentally unwind it to understand the outer operation.

Oxlint 1.83.0 provides `max-depth` for nested blocks, `max-nested-callbacks` for nested callbacks, and `complexity` for control-flow complexity. None measures this input-production depth. This rule does not measure expression workload, decision nesting, mutation history, or callback complexity.

## Runtime and options

Oxlint JavaScript plugin; syntax only, with no TypeScript type information.

```json
{
  "jsPlugins": ["aexlint"],
  "rules": {
    "aexlint/max-expression-depth": ["warn", { "max": 3 }]
  }
}
```

`max` is required and must be a positive safe integer. Exactly one options object is required; unknown properties are rejected. There is no default or recommended threshold yet. The value 3 above illustrates configuration, not a validated universal readability boundary.

## Measurement

- A call, construction, tagged-template invocation, or dynamic import contributes one input-production level.
- Its depth is one plus the greatest depth of its inputs, not the sum of their depths. A direct invocation on ordinary values has depth 1.
- Calls used to produce callable values count as inputs: `factory()()` has depth 2.
- For method calls and member-access template tags, the receiver is a sequential chain, not another buried input. A fluent chain does not increase depth merely by adding steps. Computed member keys and invocation arguments still count as inputs.
- Receiver computations retain their own depth. Fluent syntax does not erase deeply nested work inside an earlier step or its arguments.
- Other syntax propagates the greatest child depth without adding a level: arrays, objects, spreads, operators, ternaries, templates, JSX, `await`, parentheses, and TypeScript assertions/non-null/satisfies wrappers are not escape hatches.
- Function boundaries and class bodies separate computations. Their internal calls are checked independently, but are not charged to an expression merely receiving a function or class value. This includes callback bodies and parameter defaults.

Ternaries do not themselves contribute a level in this rule. Their branches can contain deep computations, but deciding whether a ternary is too complicated belongs to the decision/workload aspects.

## Reported code

With `{ "max": 2 }`:

```ts
publish(encode(parse(input)));
publish({ document: encode(parse(input)) });
publish(encode(parse(input)) as Document);
```

Each has input depth 3. Named intermediate results can make the prerequisites explicit:

```ts
const document = parse(input);
publish(encode(document));
```

Reports `tooDeep` on the complete outermost over-limit invocation, with measured depth and configured maximum. Nested over-limit invocations covered by that report are suppressed. Independent expressions and computations inside functions/class bodies receive their own reports. When a fluent step has only inherited an earlier receiver's excessive depth, report the earlier offending invocation, not each subsequent step.

## Accepted code

With `{ "max": 2 }`:

```ts
publish(encode(input));
source.first().second().third().done();
const label = enabled ? "enabled" : "disabled";
const data = { nested: { values: [1, 2, 3] } };
items.map((item) => {
  const normalized = normalize(item);
  return encode(normalized);
});
let total = 0;
for (const item of items) total += item.amount;
```

Shallow calls with many arguments are not penalized for breadth. Named intermediates inside callbacks and straightforward mutation are not defects under this rule. Complex callback bodies can still contain their own over-limit expressions.

## Fixes

Diagnostic-only. No fixes or suggestions are emitted. Extracting expressions can change evaluation order, short-circuit behavior, scope, or receiver binding; meaningful intermediate names cannot be generated reliably by this rule.

## Limitations

Depth is a syntactic readability signal, not proof that code is difficult to understand. Calls have equal weight regardless of their semantics. Property accesses may execute getters but do not count as invocations. Fluent chains are not necessarily semantically pipelines; the exemption reflects reading order, not type knowledge or purity.

A long fluent chain, broad expression, nested ternary, or complex callback may be difficult to read without exceeding this limit. Those require separate contracts, not additional penalties hidden inside this rule. An arbitrary temporary or forwarding helper can lower a score without improving the code; lower depth alone is not evidence of a better design.

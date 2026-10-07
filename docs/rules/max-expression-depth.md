# max-expression-depth

Stable, syntax-only Oxlint JS rule. Limits how deeply input-producing invocations are buried inside other invocations.

## Configuration

```json
{
  "jsPlugins": ["@aetherall/aexlint"],
  "rules": { "aexlint/max-expression-depth": ["warn", { "max": 2 }] }
}
```

Exactly one options object is required. `max` must be a positive safe integer; unknown keys and extra options are rejected. There is no default or recommended limit.

## What counts

- A call, construction, tagged template or dynamic import has depth one plus the greatest depth of its inputs. Inputs are not summed.
- A call that produces a callable value counts as an input: `factory()()` has depth 2.
- Method calls and member-access template tags treat the receiver as a sequence, not another input level. `source.first().second().done()` has depth 1. Arguments and computed member keys still count as inputs.
- A chain retains the greatest depth of its stages and receiver computations; adding fluent steps cannot hide deep work.
- Other syntax carries the greatest child depth without adding a level. Arrays, objects, spreads, operators, ternaries, templates, JSX, `await` and TypeScript wrappers do not reset depth.
- Functions and class bodies separate computations. Callback bodies and parameter defaults are checked independently, not charged to an expression receiving the function.

## Example

With `max: 2`, this reports depth 3:

```ts
publish(encode(parse(input)));
```

An accepted version makes the first prerequisite explicit:

```ts
const document = parse(input);
publish(encode(document));
```

## Reporting and limits

Reports `tooDeep` on the complete outermost over-limit invocation, with its depth and maximum. Covered nested reports are suppressed; independent functions and class bodies keep their own reports. A fluent step that only inherits excessive receiver depth leaves the report on the earlier offending invocation.

No fixes or suggestions. Extracting inputs can change evaluation order, short-circuiting, scope or receiver binding.

This measures syntax, not runtime cost, purity or readability. Getters do not count as invocations. Fluent chains can still be hard to read, and arbitrary helpers can lower depth without improving code.

Breadth, callback nesting, state history and block nesting are not measured. [Expression workload](max-expression-complexity.md) and [decision depth](max-decision-depth.md) are separate measures; their rules may report the same expression without cross-rule suppression.

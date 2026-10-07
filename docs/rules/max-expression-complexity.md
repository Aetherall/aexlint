# max-expression-complexity

Stable, syntax-only Oxlint JS rule. Limits computation packed into one expression, not statement count or runtime cost.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/max-expression-complexity": ["warn", { "max": 4 }] }
}
```

Exactly one options object is required. `max` must be a positive safe integer; unknown keys and extra options are rejected. There is no default or recommended limit.

## Workload

A direct call, construction, tagged template or dynamic import costs 1 plus its inputs' workload. Binary operations, updates, compound assignments and `yield` also add 1, with the exceptions below.

Literals, identifiers, property/optional access, parentheses, TypeScript wrappers, plain assignment, `??`, `??=`, `await`, `!` and `typeof` add nothing themselves. Their contained work still counts. Unary signs on numeric/bigint literals are free; other unary operations add 1.

A flat `&&` or `||` group costs 1 plus its operands' work, regardless of length. Parentheses and TypeScript wrappers preserve a group; a different operator or intervening negation separates groups. Thus `a && b && c` costs 1, while `first() && second() && third()` costs 4.

A ternary costs 1 plus its condition's workload plus the greater workload of its alternatives, not their sum.

String literals, templates and additions known syntactically to produce strings make further concatenation free. Interpolated work and earlier arithmetic still count: `a() + b() + c() + "suffix"` costs 5. Variable names and type annotations do not establish strings.

## Separate computations

- Object values contribute zero to an enclosing expression. Fields are checked independently; a computed key and its value are checked together. Nested fields and spreads are not skipped.
- Arrays and JSX carry the greatest entry/child workload, not their sum.
- A method call or member-access template tag costs 1 plus argument and computed-key workloads. A fluent chain carries the maximum of that stage and its receiver, not their sum. Chain length alone is free; `factory()()` is not a fluent step and costs 2.
- Functions and classes are independent boundaries. Their bodies are checked but not charged to an expression receiving them. Statements in a block are not summed.

## Example

With `max: 4`, this reports workload 5:

```ts
createResult(parseHeader(header), resolveOwner(owner), calculatePrice(items), permissions(user));
```

This version is accepted; its final expression costs 3:

```ts
const headerValue = parseHeader(header);
const ownerValue = resolveOwner(owner);
createResult(headerValue, ownerValue, calculatePrice(items), permissions(user));
```

## Reporting and limits

Checks complete initializers, expression statements, return/throw arguments, expression-bodied arrows, conditions, switch discriminants/case tests, loop initializers/updates/iterables, parameter/destructuring defaults, exports, enum initializers, `with` objects, decorators, class heritage, field initializers and computed class-member keys.

Reports `tooComplex` over the complete over-limit region, with its workload and maximum. Overlapping regions produce one outer report; function, class and object-field boundaries keep independent reports.

No fixes or suggestions. Extraction can change evaluation order, short-circuiting, scope or receiver binding. The rule does not inspect callees, infer types, count getter work or prove purity. Its boundaries do not prove runtime independence, and a lower score does not prove clearer code.

[Input depth](max-expression-depth.md) and [decision depth](max-decision-depth.md) are separate measures; their rules may report the same expression without cross-rule suppression. Block/callback nesting and state history are outside this rule.

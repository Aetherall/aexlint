# max-expression-complexity

## Intent and runtime

Measure computation that must be understood together, not raw syntax-node count. Syntax-only Oxlint JS rule, covering expression workload. It does not prove readability, purity, or execution cost.

Oxlint 1.83.0's `complexity` measures control-flow complexity, `max-depth` measures block nesting, and `max-statements` counts statements. None measures this expression workload with independent declarative entries and fluent stages.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": {
    "aexlint/max-expression-complexity": ["warn", { "max": 4 }]
  }
}
```

Exactly one options object with a positive safe integer `max` is required. Unknown keys and extra options are rejected. There is no default.

## Scoring

A direct call, construction, tagged-template invocation, or dynamic import costs 1, plus the work in its inputs. Most arithmetic/comparison operations, updates, compound assignments, and `yield` also add 1. Costs are syntactic, not estimates of runtime expense.

Literals, identifiers, ordinary property access, optional access, parentheses, TypeScript wrappers, plain assignment, nullish defaulting/coalescing assignment, `await`, negation, and `typeof` add no unit of their own. Their contained computations remain counted. Unary signs on numeric/bigint literals are free. Other unary operators add 1.

### Logical groups and choices

A flat same-operator AND or OR group costs **1 combining operation plus its operands' computations**, regardless of the number of connecting tokens. Parentheses and TypeScript wrappers do not split a same-operator group. A different operator, or an intervening negation, starts a distinct group.

```ts
a && b && c && d && e;
first() && second() && third();
first() && second() && third() && fourth();
```

These cost 1, 4, and 5. Already-named values are not charged as hidden computations. The last expression reports at max 4. Moving the same expression into a named variable does not worsen its score.

A ternary costs **1 selection plus condition workload plus the greater workload of its alternatives**. Simple value selection costs 1. The alternatives are assessed separately, not added as though they were one computation.

A flat group and a deeply alternating expression may have the same workload:

```ts
(a && b) || (c && d);
a && (b || (c && d));
```

Both cost 3. Their decision depths are 2 and 3; [max-decision-depth](max-decision-depth.md) measures that separate dimension.

### Declarative boundaries

Named object fields are independent reading units. An object value contributes zero to its enclosing expression; each field value is checked independently, including fields of nested objects. Computed key and value workload is checked together. Spreads are checked, not discarded. A complicated field still reports, and two complicated fields produce two diagnostics.

Arrays and JSX carry their greatest entry/child workload, rather than summing every independent entry. They do not erase a complicated entry. This allows collections of simple values, formatter registries, and declarative schema shapes without exemptions based on library or identifier names.

For example, `{ a: readA(), b: readB(), c: readC() }` has three independently checked fields of workload 1. In contrast, `combine(readA(), readB(), readC(), readD())` costs 5: its positional inputs and the combining operation are one computation.

These boundaries are structural signals, not proof that field names are meaningful or fields are independent at runtime. Cross-field mutation and data dependencies are not analyzed.

### Fluent sequences

A method call or member-access template tag costs 1 plus the sum of argument workloads and any computed member-key workload. The chain carries the maximum of that stage and the receiver workload, not their sum.

Chain length alone therefore does not trigger a diagnostic. Complicated arguments and receiver computations remain visible. Calls producing callable values, such as `factory()()`, are not fluent receiver steps and add together.

### String formatting

String literals, templates, and concatenations syntactically known to produce strings propagate a string marker through transparent wrappers. Joining them adds no extra operation, just like template interpolation structure. Expressions inside the formatting still count.

```ts
const label = "Name: " + first + " " + last;
const equivalent = `Name: ${first} ${last}`;
```

Both cost zero. Splitting literal prose over concatenated lines does not manufacture computation. However, arithmetic preceding string coercion still counts: `a() + b() + c() + "suffix"` costs 5. Calls and variables are not assumed to return strings based on names or annotations; ambiguous additions remain counted.

## Reporting regions

Check complete initializers, expression statements, return/throw arguments, expression-bodied arrows, conditions, switch discriminants/case tests, loop initializers/updates and iterable expressions, parameter/destructuring defaults, export expressions, enum initializers, `with` objects, decorators, class heritage expressions, field initializers, and computed class-member keys.

Function/callback and class definitions are independent boundaries. Their bodies are not charged to an expression receiving them. Statements are not summed across a block: a named intermediate inside a callback is allowed.

Report `tooComplex` on the complete over-limit region, including measured workload and maximum. Overlapping regions in one computation produce one outer report, not a per-operator cascade. Function, class, and object-field boundaries retain independent diagnostics.

## Reported and accepted examples

With max 4, this reports workload 5 despite invocation depth being only 2:

```ts
createResult(parseHeader(header), resolveOwner(owner), calculatePrice(items), permissions(user));
```

Meaningful intermediate values can expose separate steps:

```ts
const headerValue = parseHeader(header);
const ownerValue = resolveOwner(owner);
createResult(headerValue, ownerValue, calculatePrice(items), permissions(user));
```

The final expression costs 3. This is an illustrative refactoring, not an automatically safe rewrite.

Also accepted at max 4:

```ts
source.first().second().third().done();
const children = transaction ? await transaction.get(query) : await query.get();
const results = await Promise.all([loadA(), loadB(), loadC()]);
if (Number.isNaN(index) || index < 1 || !key) {
}
items.map((item) => {
  const doubled = item.value * 2;
  return doubled + 1;
});
let total = 0;
for (const item of items) total += item.amount;
```

## Fixes and limitations

Diagnostic-only: no fixes or suggestions. Extraction can alter evaluation order, short-circuit behavior, scope, or receiver binding. The rule cannot choose meaningful names or prove a helper improves readability.

This rule does not inspect callees, infer types, or understand domain semantics. Property getters may perform work without being counted. Arrays, objects, functions, or helpers can be used to disguise complexity; lower scores alone are not evidence of better design.

Workload, invocation depth, and decision depth are independent. Enabling their rules together can produce multiple diagnostics for one expression. There is no cross-rule suppression or combined score. Block nesting, callback nesting, and state history require separate analysis.

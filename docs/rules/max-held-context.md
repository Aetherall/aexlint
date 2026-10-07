# max-held-context

Prototype, syntax-only Oxlint JS rule. Limits enclosing contexts a reader must remember where a decision is made, including across callbacks. The boundaries are provisional; procedure length is not measured.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/max-held-context": ["warn", { "max": 3 }] }
}
```

Exactly one options object is required. `max` must be a positive safe integer; unknown keys and extra options are rejected. There is no default or recommended limit.

## What counts

A decision's depth is the number of open contexts plus one. Decisions are `if`, loops, `switch`, `catch` and ternaries. Logical groups, nullish defaulting and optional access are not decisions; see [max-decision-depth](max-decision-depth.md).

Each branch body, loop body, switch case, catch body, ternary alternative and directly passed callback body opens one context. An `else if` chain keeps all alternatives at the same depth.

An `if` without `else` whose body is only `return`, `throw`, `continue` or `break` opens no context. The guard itself is still a decision; its condition and exit argument are checked at the current depth.

The program and functions start at zero, except functions passed directly as call or constructor arguments: those carry the caller's contexts into their bodies. Class bodies reset depth; methods and local, returned or object-property functions are measured separately. Only function bodies are traversed, not parameter defaults.

Only decisions report. Nested callbacks with no decisions never report, however deep they are.

## Example

With `max: 3`, the guard below reports depth 4: loop, branch, callback, then decision.

```ts
for (const item of items) {
  if (item.length > 0) {
    item.map((value) => {
      if (value < 0) throw new Error("negative");
      return value;
    });
  }
}
```

Independent guards are accepted at depth 1:

```ts
function validate(a, b) {
  if (!a) throw new Error("a");
  if (!b) throw new Error("b");
  return build(a, b);
}
```

## Reporting and limits

Reports `tooDeep` with open-context count, depth and maximum. The span covers the decision head through its closing parenthesis; ternaries and `do … while` report only their condition, and parameterless `catch` reports its keyword. Only the first over-limit decision in an open context reports; its siblings and descendants are suppressed. Separate contexts, including opposite branches, can report separately.

No fixes or suggestions. Restructuring can change early exits, evaluation order and captured state. The rule does not follow callback identifiers/aliases or know when callbacks run. It does not measure decision breadth, interacting state or closure ownership.

Test and asynchronous callbacks count like any others: nested `describe`/`it` or timer/promise callbacks can make a simple guard exceed the limit. Use Oxlint `overrides` to disable the rule or raise `max` for test files when appropriate.

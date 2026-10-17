# max-held-context

Limits how many enclosing contexts a reader must keep in mind where a decision is made, including across callbacks.

```json
{ "rules": { "aexlint/max-held-context": ["warn", { "max": 3 }] } }
```

`max` is a required positive integer.

## Example

Reported at `max: 3`: the guard sits under a loop, a branch and a callback (depth 4).

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

Accepted: independent guards at depth 1.

```ts
function validate(a, b) {
  if (!a) throw new Error("a");
  if (!b) throw new Error("b");
  return build(a, b);
}
```

## What counts

- Decisions are `if`, loops, `switch`, `catch` and ternaries. A decision's depth is its open contexts plus one.
- Branch, loop, case, catch and ternary bodies open a context, and so does a callback passed directly as an argument. An `else if` chain stays at one depth.
- An `if` without `else` whose body only returns, throws, continues or breaks opens no context.
- Other functions and class bodies start again at zero.

## Reporting and limits

Reports `tooDeep` on the decision head; only the first over-limit decision in a context is reported. No fixes. Callback aliases and when callbacks run are not tracked. Nested test and promise callbacks count like any others; use overrides for test files.

[no-else-if](../no-else-if/README.md) reports `else if` chains, which this rule allows. Early returns satisfy both.
